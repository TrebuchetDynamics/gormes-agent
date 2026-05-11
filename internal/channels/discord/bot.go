package discord

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const (
	ackEmoji        = "👀"
	successEmoji    = "✅"
	failureEmoji    = "❌"
	placeholderText = "⏳"
)

type Config struct {
	AllowedChannelID       string
	AllowedChannelIDs      []string
	IgnoredChannelIDs      []string
	FreeResponseChannelIDs []string
	NoThreadChannelIDs     []string
	ChannelSkillBindings   any
	ChannelPrompts         any
	RequireMention         bool
	RequireMentionSet      bool
	AutoThread             bool
	AutoThreadSet          bool
	AllowBots              string
	SelfUserID             string
	ReplyToMode            string
	FirstRunDiscovery      bool
	// AttachmentCacheDir stores Discord attachment downloads before the gateway
	// sees local attachment descriptors. Empty uses the user cache dir.
	AttachmentCacheDir string
	// ThreadStatePath stores the bounded set of participated Discord threads.
	// Empty uses GormesHome()/discord_threads.json.
	ThreadStatePath string
	// ThreadParticipationLimit caps persisted participated threads. Empty uses
	// Hermes' 500-thread default.
	ThreadParticipationLimit int
	// AttachmentHTTPClient fetches SSRF-gated URL fallbacks in tests and
	// production. Empty uses a bounded default client.
	AttachmentHTTPClient *http.Client
	AccountID             string
	// SkillCollector returns the current gateway-visible skill command set for
	// Discord autocomplete refreshes. Nil means this adapter has no cached skill
	// group to refresh.
	SkillCollector func(context.Context) ([]gateway.PlatformCommand, error)
	// PluginCommands are plugin-provided slash commands that Discord can expose
	// as top-level native commands when they do not shadow built-ins.
	PluginCommands []gateway.PlatformCommand
}

type Bot struct {
	cfg     Config
	session discordSession
	log     *slog.Logger

	reactionsMu sync.Mutex
	reactions   map[string]bool

	threadsMu sync.RWMutex
	threads   map[string]discordThread

	participatedThreads *ThreadParticipationTracker

	replyMu        sync.Mutex
	replyFirstSent map[string]bool

	skillMu       sync.RWMutex
	skillCommands []gateway.PlatformCommand
	skillHidden   int
}

type discordThread struct {
	id       string
	parentID string
	name     string
	guildID  string
}

var (
	_ gateway.Channel            = (*Bot)(nil)
	_ gateway.DisconnectCapable  = (*Bot)(nil)
	_ gateway.MessageEditor      = (*Bot)(nil)
	_ gateway.MediaSender        = (*Bot)(nil)
	_ gateway.PlaceholderCapable = (*Bot)(nil)
	_ gateway.ReactionCapable    = (*Bot)(nil)
)

func New(cfg Config, session discordSession, log *slog.Logger) *Bot {
	if log == nil {
		log = slog.Default()
	}
	b := &Bot{
		cfg:            cfg,
		session:        session,
		log:            log,
		reactions:      map[string]bool{},
		threads:        map[string]discordThread{},
		replyFirstSent: map[string]bool{},
		participatedThreads: NewThreadParticipationTracker(ThreadParticipationOptions{
			Path:       cfg.ThreadStatePath,
			MaxTracked: cfg.ThreadParticipationLimit,
		}),
	}
	if ev := b.participatedThreads.LoadEvidence(); ev.Code != "" {
		b.log.Warn("discord thread participation tracker reset", "evidence", ev.Code)
	}
	return b
}

func (b *Bot) Name() string {
	if b.cfg.AccountID != "" {
		return "discord:" + b.cfg.AccountID
	}
	return "discord"
}

func isForumChannel(ch *discordgo.Channel) bool {
	return ch != nil && ch.Type == discordgo.ChannelTypeGuildForum
}

func (b *Bot) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	b.session.AddHandler(func(_ *discordgo.Session, t *discordgo.ThreadCreate) {
		if t == nil {
			return
		}
		ev, ok := b.toThreadLifecycleEvent(t.Channel)
		if !ok {
			return
		}
		select {
		case inbox <- ev:
		case <-ctx.Done():
		}
	})
	b.session.AddHandler(func(_ *discordgo.Session, t *discordgo.ThreadUpdate) {
		if t == nil {
			return
		}
		ev, ok := b.toThreadLifecycleEvent(t.Channel)
		if !ok {
			return
		}
		select {
		case inbox <- ev:
		case <-ctx.Done():
		}
	})
	b.session.AddHandler(func(_ *discordgo.Session, t *discordgo.ThreadDelete) {
		if t == nil {
			return
		}
		ev, ok := b.toThreadLifecycleEvent(t.Channel)
		if !ok {
			return
		}
		ev.ThreadLifecycle.State = gateway.ThreadLifecycleClosed
		select {
		case inbox <- ev:
		case <-ctx.Done():
		}
	})
	b.session.AddHandler(func(_ *discordgo.Session, i *discordgo.InteractionCreate) {
		b.handleInteraction(ctx, inbox, b.session, i)
	})
	b.session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m == nil || m.Message == nil {
			return
		}
		if m.Author == nil {
			return
		}
		if !b.acceptMessage(s, m.Message) {
			return
		}
		ev, ok := b.toInboundEventWithContext(ctx, m.Message)
		if !ok {
			return
		}
		select {
		case inbox <- ev:
		case <-ctx.Done():
		}
	})
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("discord: open session: %w", err)
	}
	if err := b.registerSlashCommands(ctx); err != nil {
		b.log.Warn("discord slash command registration unavailable", "err", err)
	}
	<-ctx.Done()
	_ = b.Disconnect(ctx)
	return nil
}

func (b *Bot) Disconnect(context.Context) error {
	return b.session.Close()
}

func (b *Bot) toInboundEvent(m *discordgo.Message) (gateway.InboundEvent, bool) {
	return b.toInboundEventWithContext(context.Background(), m)
}

func (b *Bot) toInboundEventWithContext(ctx context.Context, m *discordgo.Message) (gateway.InboundEvent, bool) {
	text, attachments := b.discordInboundTextAndAttachments(ctx, m)
	if strings.TrimSpace(text) == "" && len(attachments) == 0 {
		return gateway.InboundEvent{}, false
	}
	kind, body := gateway.ParseInboundText(text)

	userID := ""
	if m.Author != nil {
		userID = m.Author.ID
	}
	chatID := m.ChannelID
	scopeChannelID := strings.TrimSpace(m.ChannelID)
	threadID := ""
	chatName := ""
	parentChatID := ""
	guildID := strings.TrimSpace(m.GuildID)
	if thread, ok := b.threadForMessageChannel(m.ChannelID); ok {
		chatID = thread.parentID
		threadID = thread.id
		chatName = thread.name
		parentChatID = thread.parentID
		if guildID == "" {
			guildID = thread.guildID
		}
	}
	if threadID != "" {
		if ev, err := b.participatedThreads.Mark(threadID); err != nil {
			b.log.Warn("discord thread participation tracker write failed", "evidence", ev.Code)
		}
	}
	messageID := strings.TrimSpace(m.ID)
	return gateway.InboundEvent{
		Platform:     "discord",
		ChatID:       chatID,
		ChatName:     chatName,
		UserID:       userID,
		ThreadID:     threadID,
		MsgID:        messageID,
		GuildID:      guildID,
		ParentChatID: parentChatID,
		MessageID:    messageID,
		Kind:         kind,
		Text:         body,
		AutoSkills:   gateway.ResolveChannelSkills(b.cfg.ChannelSkillBindings, scopeChannelID, parentChatID),
		ChannelPrompt: gateway.ResolveChannelPrompt(
			b.cfg.ChannelPrompts,
			scopeChannelID,
			parentChatID,
		),
		Attachments: attachments,
	}, true
}

// RefreshSkillGroup refreshes the adapter-owned skill command cache used by
// Discord autocomplete. It mutates only in-memory command state; no Discord API
// sync is required.
func (b *Bot) RefreshSkillGroup(ctx context.Context) (gateway.SkillGroupRefreshResult, error) {
	b.skillMu.RLock()
	previousCount := len(b.skillCommands)
	previousHidden := b.skillHidden
	b.skillMu.RUnlock()
	if b.cfg.SkillCollector == nil {
		return gateway.SkillGroupRefreshResult{Channel: b.Name(), Count: previousCount, Hidden: previousHidden}, nil
	}
	commands, err := b.cfg.SkillCollector(ctx)
	if err != nil {
		return gateway.SkillGroupRefreshResult{Channel: b.Name(), Count: previousCount, Hidden: previousHidden}, err
	}
	commands, hidden := normalizeSkillGroupCommands(commands)
	b.skillMu.Lock()
	b.skillCommands = commands
	b.skillHidden = hidden
	b.skillMu.Unlock()
	return gateway.SkillGroupRefreshResult{Channel: b.Name(), Count: len(commands), Hidden: hidden}, nil
}

// SkillGroupCommands returns a defensive copy of the cached Discord skill
// autocomplete commands.
func (b *Bot) SkillGroupCommands() []gateway.PlatformCommand {
	b.skillMu.RLock()
	defer b.skillMu.RUnlock()
	return append([]gateway.PlatformCommand(nil), b.skillCommands...)
}

func normalizeSkillGroupCommands(commands []gateway.PlatformCommand) ([]gateway.PlatformCommand, int) {
	out := make([]gateway.PlatformCommand, 0, len(commands))
	seen := map[string]struct{}{}
	hidden := 0
	for _, cmd := range commands {
		name := strings.TrimSpace(cmd.Name)
		if name == "" {
			hidden++
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			hidden++
			continue
		}
		seen[key] = struct{}{}
		out = append(out, gateway.PlatformCommand{
			Name:        name,
			Description: strings.TrimSpace(cmd.Description),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, hidden
}

func (b *Bot) acceptMessage(s *discordgo.Session, m *discordgo.Message) bool {
	if m.Type != discordgo.MessageTypeDefault && m.Type != discordgo.MessageTypeReply {
		return false
	}
	ctx := b.admissionContext(s, m)
	result := EvaluateAdmission(b.admissionPolicy(), ctx)
	return result.Allowed
}

func (b *Bot) admissionPolicy() AdmissionPolicy {
	requireMention := true
	if b.cfg.RequireMentionSet {
		requireMention = b.cfg.RequireMention
	}
	autoThread := true
	if b.cfg.AutoThreadSet {
		autoThread = b.cfg.AutoThread
	}
	allowed := append([]string{}, b.cfg.AllowedChannelIDs...)
	if b.cfg.AllowedChannelID != "" {
		allowed = append(allowed, b.cfg.AllowedChannelID)
	}
	return AdmissionPolicy{
		AllowedChannelIDs:     allowed,
		IgnoredChannelIDs:     b.cfg.IgnoredChannelIDs,
		FreeResponseChannels:  b.cfg.FreeResponseChannelIDs,
		NoThreadChannelIDs:    b.cfg.NoThreadChannelIDs,
		RequireMention:        requireMention,
		AutoThread:            autoThread,
		AllowBots:             b.cfg.AllowBots,
		KnownThreadBypass:     true,
		ParticipatedThreadIDs: b.participatedThreads.Snapshot(),
	}
}

func (b *Bot) admissionContext(s *discordgo.Session, m *discordgo.Message) AdmissionContext {
	channelID := strings.TrimSpace(m.ChannelID)
	parentID := ""
	isThread := false
	if thread, ok := b.threadForMessageChannel(channelID); ok {
		parentID = thread.parentID
		isThread = true
	}
	selfUserID := strings.TrimSpace(b.cfg.SelfUserID)
	if selfUserID == "" && s != nil && s.State != nil && s.State.User != nil {
		selfUserID = strings.TrimSpace(s.State.User.ID)
	}
	mentioned := false
	if selfUserID != "" {
		for _, user := range m.Mentions {
			if user != nil && strings.TrimSpace(user.ID) == selfUserID {
				mentioned = true
				break
			}
		}
	}
	authorID := ""
	authorBot := false
	if m.Author != nil {
		authorID = strings.TrimSpace(m.Author.ID)
		authorBot = m.Author.Bot
	}
	return AdmissionContext{
		ChannelID:          channelID,
		ParentChannelID:    parentID,
		GuildID:            strings.TrimSpace(m.GuildID),
		AuthorID:           authorID,
		AuthorBot:          authorBot,
		SelfUserID:         selfUserID,
		Mentioned:          mentioned,
		ParticipatedThread: b.hasParticipatedThread(channelID),
		IsDM:               strings.TrimSpace(m.GuildID) == "",
		IsThread:           isThread,
		IsReply:            m.Type == discordgo.MessageTypeReply,
	}
}

func (b *Bot) rememberThread(ch *discordgo.Channel) {
	if ch == nil || !ch.IsThread() || strings.TrimSpace(ch.ID) == "" {
		return
	}
	parentID := strings.TrimSpace(ch.ParentID)
	if parentID == "" {
		return
	}
	b.threadsMu.Lock()
	defer b.threadsMu.Unlock()
	b.threads[ch.ID] = discordThread{
		id:       strings.TrimSpace(ch.ID),
		parentID: parentID,
		name:     strings.TrimSpace(ch.Name),
		guildID:  strings.TrimSpace(ch.GuildID),
	}
}

func (b *Bot) threadForMessageChannel(channelID string) (discordThread, bool) {
	b.threadsMu.RLock()
	defer b.threadsMu.RUnlock()
	thread, ok := b.threads[strings.TrimSpace(channelID)]
	return thread, ok
}

func (b *Bot) hasParticipatedThread(threadID string) bool {
	return b.participatedThreads.Contains(threadID)
}

func (b *Bot) toThreadLifecycleEvent(ch *discordgo.Channel) (gateway.InboundEvent, bool) {
	if ch == nil || !ch.IsThread() || strings.TrimSpace(ch.ID) == "" || strings.TrimSpace(ch.ParentID) == "" {
		return gateway.InboundEvent{}, false
	}
	b.rememberThread(ch)

	archived := false
	locked := false
	if ch.ThreadMetadata != nil {
		archived = ch.ThreadMetadata.Archived
		locked = ch.ThreadMetadata.Locked
	}

	state := gateway.ThreadLifecycleOpen
	switch {
	case locked:
		state = gateway.ThreadLifecycleClosed
	case archived:
		state = gateway.ThreadLifecycleArchived
	}

	threadID := strings.TrimSpace(ch.ID)
	parentID := strings.TrimSpace(ch.ParentID)
	name := strings.TrimSpace(ch.Name)
	return gateway.InboundEvent{
		Platform:     "discord",
		ChatID:       parentID,
		ChatName:     name,
		ThreadID:     threadID,
		GuildID:      strings.TrimSpace(ch.GuildID),
		ParentChatID: parentID,
		Kind:         gateway.EventThreadLifecycle,
		ThreadLifecycle: &gateway.ThreadLifecycleEvent{
			ID:       threadID,
			ParentID: parentID,
			Name:     name,
			State:    state,
			Archived: archived,
			Locked:   locked,
		},
	}, true
}

func (b *Bot) Send(_ context.Context, chatID, text string) (string, error) {
	msg, err := b.session.ChannelMessageSendComplex(chatID, &discordgo.MessageSend{
		Content:         text,
		AllowedMentions: BuildAllowedMentionsFromEnv(),
	})
	if err != nil {
		return "", fmt.Errorf("discord: send: %w", err)
	}
	return msg.ID, nil
}

func (b *Bot) SendReply(_ context.Context, chatID, replyToMsgID, text string) (string, error) {
	replyToMsgID = strings.TrimSpace(replyToMsgID)
	data := &discordgo.MessageSend{
		Content:         text,
		AllowedMentions: BuildAllowedMentionsFromEnv(),
	}
	mode := normalizeReplyToMode(b.cfg.ReplyToMode)
	markFirst := false
	if replyToMsgID != "" && mode != "off" && b.shouldAttachReplyReference(chatID, replyToMsgID, mode) {
		failIfMissing := false
		data.Reference = &discordgo.MessageReference{
			MessageID:       replyToMsgID,
			ChannelID:       strings.TrimSpace(chatID),
			FailIfNotExists: &failIfMissing,
		}
		markFirst = mode == "first"
	}
	msg, err := b.session.ChannelMessageSendComplex(chatID, data)
	if err != nil && data.Reference != nil && isMissingDiscordReplyReference(err) {
		data.Reference = nil
		msg, err = b.session.ChannelMessageSendComplex(chatID, data)
	}
	if err != nil {
		return "", fmt.Errorf("discord: send reply: %w", err)
	}
	if markFirst {
		b.markReplyReferenceSent(chatID, replyToMsgID)
	}
	return msg.ID, nil
}

func (b *Bot) shouldAttachReplyReference(chatID, replyToMsgID, mode string) bool {
	if mode == "all" {
		return true
	}
	key := replyReferenceKey(chatID, replyToMsgID)
	b.replyMu.Lock()
	defer b.replyMu.Unlock()
	return !b.replyFirstSent[key]
}

func (b *Bot) markReplyReferenceSent(chatID, replyToMsgID string) {
	key := replyReferenceKey(chatID, replyToMsgID)
	b.replyMu.Lock()
	b.replyFirstSent[key] = true
	b.replyMu.Unlock()
}

func replyReferenceKey(chatID, replyToMsgID string) string {
	return strings.TrimSpace(chatID) + ":" + strings.TrimSpace(replyToMsgID)
}

func (b *Bot) SendMedia(_ context.Context, chatID, replyToMsgID string, media gateway.OutboundMedia) (string, error) {
	mediaPath := strings.TrimSpace(media.Path)
	if mediaPath == "" {
		return "", fmt.Errorf("discord: media path is required")
	}
	file, err := os.Open(mediaPath)
	if err != nil {
		return "", fmt.Errorf("discord: media unavailable")
	}
	defer file.Close()

	targetChannelID := strings.TrimSpace(media.ThreadID)
	if targetChannelID == "" {
		targetChannelID = strings.TrimSpace(chatID)
	}
	data := &discordgo.MessageSend{
		AllowedMentions: BuildAllowedMentionsFromEnv(),
		Files: []*discordgo.File{{
			Name:   filepath.Base(mediaPath),
			Reader: file,
		}},
	}
	if replyToMsgID = strings.TrimSpace(replyToMsgID); replyToMsgID != "" {
		failIfMissing := false
		data.Reference = &discordgo.MessageReference{
			MessageID:       replyToMsgID,
			ChannelID:       targetChannelID,
			FailIfNotExists: &failIfMissing,
		}
	}
	msg, err := b.session.ChannelMessageSendComplex(targetChannelID, data)
	if err != nil {
		return "", fmt.Errorf("discord: send media: %w", err)
	}
	return msg.ID, nil
}

func (b *Bot) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	return b.Send(ctx, chatID, placeholderText)
}

func (b *Bot) EditMessage(_ context.Context, chatID, msgID, text string) error {
	if _, err := b.session.ChannelMessageEdit(chatID, msgID, text); err != nil {
		return fmt.Errorf("discord: edit: %w", err)
	}
	return nil
}

func (b *Bot) ReactToMessage(_ context.Context, chatID, msgID string) (func(), error) {
	if err := b.session.MessageReactionAdd(chatID, msgID, ackEmoji); err != nil {
		return nil, fmt.Errorf("discord: reaction add: %w", err)
	}

	key := chatID + ":" + msgID
	return func() {
		b.reactionsMu.Lock()
		if b.reactions[key] {
			b.reactionsMu.Unlock()
			return
		}
		b.reactions[key] = true
		b.reactionsMu.Unlock()
		_ = b.session.MessageReactionRemoveMe(chatID, msgID, ackEmoji)
	}, nil
}

func discordReactionsEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("DISCORD_REACTIONS"))) {
	case "false", "0", "no":
		return false
	default:
		return true
	}
}

func (b *Bot) OnProcessingStart(_ context.Context, chatID, msgID string) error {
	if !discordReactionsEnabled() || strings.TrimSpace(chatID) == "" || strings.TrimSpace(msgID) == "" {
		return nil
	}
	if err := b.session.MessageReactionAdd(chatID, msgID, ackEmoji); err != nil {
		b.log.Debug("discord reaction add failed", "emoji", ackEmoji, "err", err)
	}
	return nil
}

func (b *Bot) OnProcessingComplete(_ context.Context, chatID, msgID string, outcome gateway.ProcessingOutcome) error {
	if !discordReactionsEnabled() || strings.TrimSpace(chatID) == "" || strings.TrimSpace(msgID) == "" {
		return nil
	}
	if err := b.session.MessageReactionRemoveMe(chatID, msgID, ackEmoji); err != nil {
		b.log.Debug("discord reaction remove failed", "emoji", ackEmoji, "err", err)
	}
	switch outcome {
	case gateway.ProcessingOutcomeSuccess:
		if err := b.session.MessageReactionAdd(chatID, msgID, successEmoji); err != nil {
			b.log.Debug("discord reaction add failed", "emoji", successEmoji, "err", err)
		}
	case gateway.ProcessingOutcomeFailure:
		if err := b.session.MessageReactionAdd(chatID, msgID, failureEmoji); err != nil {
			b.log.Debug("discord reaction add failed", "emoji", failureEmoji, "err", err)
		}
	}
	return nil
}
