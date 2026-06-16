package discord

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/discord/interactions"
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const discordCommandPayloadSoftLimit = 24000

func (b *Bot) registerSlashCommands(ctx context.Context) error {
	registrar, ok := b.session.(discordCommandRegistrar)
	if !ok {
		return fmt.Errorf("discord_slash_registration_unavailable")
	}
	appID := strings.TrimSpace(registrar.CurrentUserID())
	if appID == "" {
		return fmt.Errorf("discord_slash_registration_unavailable: application id missing")
	}
	_, err := registrar.ApplicationCommandBulkOverwrite(appID, "", b.discordSlashCommands(ctx))
	if err != nil {
		return fmt.Errorf("discord_slash_registration_unavailable: %w", err)
	}
	return nil
}

func (b *Bot) discordSlashCommands(ctx context.Context) []*discordgo.ApplicationCommand {
	b.refreshSkillCommandsBestEffort(ctx)
	commands := make([]*discordgo.ApplicationCommand, 0, len(gateway.CommandRegistry)+2)
	seen := map[string]bool{}

	add := func(cmd *discordgo.ApplicationCommand) {
		if cmd == nil {
			return
		}
		name := normalizeDiscordCommandName(cmd.Name)
		if name == "" || seen[name] {
			return
		}
		cmd.Name = name
		cmd.Description = boundedDiscordDescription(cmd.Description, "Run /"+name)
		seen[name] = true
		commands = append(commands, cmd)
	}

	add(discordThreadCommand())
	add(discordSkillCommand())
	for _, cmd := range gateway.CommandRegistry {
		name := normalizeDiscordCommandName(cmd.Name)
		if name == "" || seen[name] {
			continue
		}
		add(discordGatewayCommand(cmd))
	}
	for _, cmd := range sortedDiscordPlatformCommands(b.cfg.PluginCommands) {
		name := normalizeDiscordCommandName(cmd.Name)
		if name == "" || seen[name] {
			continue
		}
		add(discordPluginCommand(cmd))
	}

	sort.SliceStable(commands, func(i, j int) bool {
		return commands[i].Name < commands[j].Name
	})
	return commands
}

func discordGatewayCommand(cmd gateway.CommandDef) *discordgo.ApplicationCommand {
	out := &discordgo.ApplicationCommand{
		Name:        normalizeDiscordCommandName(cmd.Name),
		Description: boundedDiscordDescription(cmd.Description, "Run /"+cmd.Name),
	}
	if discordCommandCarriesArgs(cmd) {
		out.Options = []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "args",
			Description: "Optional command arguments",
			Required:    false,
		}}
	}
	return out
}

func discordPluginCommand(cmd gateway.PlatformCommand) *discordgo.ApplicationCommand {
	name := normalizeDiscordCommandName(cmd.Name)
	return &discordgo.ApplicationCommand{
		Name:        name,
		Description: boundedDiscordDescription(cmd.Description, "Run /"+name),
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "args",
			Description: "Optional command arguments",
			Required:    false,
		}},
	}
}

func discordThreadCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "thread",
		Description: "Create a new thread and start a Gormes session in it",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Thread name",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "message",
				Description: "Optional first message to send to Gormes in the thread",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "auto_archive_duration",
				Description: "Auto-archive in minutes",
				Required:    false,
			},
		},
	}
}

func discordSkillCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "skill",
		Description: "Run an enabled skill command",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "name",
				Description:  "Skill command name",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "args",
				Description: "Optional skill instruction",
				Required:    false,
			},
		},
	}
}

func discordCommandCarriesArgs(cmd gateway.CommandDef) bool {
	switch cmd.Kind {
	case gateway.EventSteer, gateway.EventTitle, gateway.EventSessions, gateway.EventProfile,
		gateway.EventSkills, gateway.EventReasoning, gateway.EventBusy, gateway.EventTTS,
		gateway.EventGoal, gateway.EventTopic, gateway.EventModel, gateway.EventSpawn:
		return true
	default:
		return cmd.ActiveTurnPolicy == gateway.CommandActiveTurnPolicyUnavailable
	}
}

func (b *Bot) refreshSkillCommandsBestEffort(ctx context.Context) {
	if b.cfg.SkillCollector == nil {
		return
	}
	_, _ = b.RefreshSkillGroup(ctx)
}

func (b *Bot) setSkillCommandsForTest(commands []gateway.PlatformCommand) {
	commands, hidden := normalizeSkillGroupCommands(commands)
	b.skillMu.Lock()
	b.skillCommands = commands
	b.skillHidden = hidden
	b.skillMu.Unlock()
}

func (b *Bot) handleInteraction(ctx context.Context, inbox chan<- gateway.InboundEvent, session discordSession, i *discordgo.InteractionCreate) {
	if i == nil || i.Interaction == nil {
		return
	}
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		b.handleAutocomplete(session, i)
		return
	}
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	data, ok := i.Data.(discordgo.ApplicationCommandInteractionData)
	if !ok {
		return
	}
	name := normalizeDiscordCommandName(data.Name)
	switch name {
	case "thread":
		b.handleThreadCreateSlash(ctx, inbox, session, i, data)
	case "skill":
		b.handleSkillSlash(ctx, inbox, session, i, data)
	default:
		if !b.authorizeInteractionOrRespond(session, i, name) {
			return
		}
		if b.pluginCommandExists(name) {
			text := "/" + name
			if args := strings.TrimSpace(discordOptionString(data, "args")); args != "" {
				text += " " + args
			}
			ev := b.inboundEventFromInteraction(i, text)
			ev.Kind = gateway.EventSubmit
			ev.Text = text
			b.enqueueInteraction(ctx, inbox, ev)
			_ = b.respondEphemeral(session, i.Interaction, "Command queued.")
			return
		}
		ev := b.inboundEventFromInteraction(i, b.discordSlashText(data))
		b.enqueueInteraction(ctx, inbox, ev)
		_ = b.respondEphemeral(session, i.Interaction, "Command queued.")
	}
}

func (b *Bot) handleAutocomplete(session discordSession, i *discordgo.InteractionCreate) {
	data, ok := i.Data.(discordgo.ApplicationCommandInteractionData)
	if !ok || normalizeDiscordCommandName(data.Name) != "skill" {
		return
	}
	current := ""
	if opt := data.GetOption("name"); opt != nil && opt.Type == discordgo.ApplicationCommandOptionString {
		if value, ok := opt.Value.(string); ok {
			current = value
		}
	}
	names := make([]string, 0, len(b.SkillGroupCommands()))
	for _, cmd := range b.SkillGroupCommands() {
		names = append(names, cmd.Name)
	}
	choices := AuthorizedSkillAutocomplete(b.interactionContext(i), b.interactionPolicy(), names, current)
	out := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(choices))
	for _, choice := range choices {
		out = append(out, &discordgo.ApplicationCommandOptionChoice{Name: choice, Value: choice})
		if len(out) == 25 {
			break
		}
	}
	if responder, ok := session.(discordInteractionResponder); ok {
		_ = responder.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{Choices: out},
		})
	}
}

func (b *Bot) handleSkillSlash(ctx context.Context, inbox chan<- gateway.InboundEvent, session discordSession, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	if !b.authorizeInteractionOrRespond(session, i, "skill") {
		return
	}
	name := normalizeDiscordCommandName(discordOptionString(data, "name"))
	if name == "" || !b.skillCommandExists(name) {
		_ = b.respondEphemeral(session, i.Interaction, "unknown skill `"+name+"`. Use `/skill` autocomplete to choose an enabled skill.")
		return
	}
	text := "/" + name
	if args := strings.TrimSpace(discordOptionString(data, "args")); args != "" {
		text += " " + args
	}
	ev := b.inboundEventFromInteraction(i, text)
	ev.Kind = gateway.EventSubmit
	ev.Text = text
	b.enqueueInteraction(ctx, inbox, ev)
	_ = b.respondEphemeral(session, i.Interaction, "Skill queued.")
}

func (b *Bot) skillCommandExists(name string) bool {
	name = normalizeDiscordCommandName(name)
	for _, cmd := range b.SkillGroupCommands() {
		if normalizeDiscordCommandName(cmd.Name) == name {
			return true
		}
	}
	return false
}

func (b *Bot) pluginCommandExists(name string) bool {
	name = normalizeDiscordCommandName(name)
	for _, cmd := range b.cfg.PluginCommands {
		if normalizeDiscordCommandName(cmd.Name) == name {
			return true
		}
	}
	return false
}

func (b *Bot) discordSlashText(data discordgo.ApplicationCommandInteractionData) string {
	name := normalizeDiscordCommandName(data.Name)
	args := strings.TrimSpace(discordOptionString(data, "args"))
	if args == "" {
		for _, opt := range data.Options {
			if opt == nil || opt.Name == "args" {
				continue
			}
			if value := discordOptionValueString(opt); value != "" {
				args = value
				break
			}
		}
	}
	if args == "" {
		return "/" + name
	}
	return "/" + name + " " + args
}

func (b *Bot) inboundEventFromInteraction(i *discordgo.InteractionCreate, text string) gateway.InboundEvent {
	kind, body := gateway.ParseInboundTextPreserveUnknown(text)
	chatID := strings.TrimSpace(i.ChannelID)
	threadID := ""
	parentChatID := ""
	chatName := ""
	guildID := strings.TrimSpace(i.GuildID)
	if thread, ok := b.threadForMessageChannel(chatID); ok {
		chatID = thread.parentID
		threadID = thread.id
		parentChatID = thread.parentID
		chatName = thread.name
		if guildID == "" {
			guildID = thread.guildID
		}
	}
	return gateway.InboundEvent{
		Platform:     b.Name(),
		AccountID:    strings.TrimSpace(b.cfg.AccountID),
		ChatID:       chatID,
		ChatName:     chatName,
		UserID:       interactionUserID(i),
		UserName:     interactionUserName(i),
		ThreadID:     threadID,
		GuildID:      guildID,
		ParentChatID: parentChatID,
		Kind:         kind,
		Text:         body,
	}
}

func (b *Bot) enqueueInteraction(ctx context.Context, inbox chan<- gateway.InboundEvent, ev gateway.InboundEvent) {
	select {
	case inbox <- ev:
	case <-ctx.Done():
	}
}

func (b *Bot) authorizeInteractionOrRespond(session discordSession, i *discordgo.InteractionCreate, command string) bool {
	result := EvaluateInteractionAuthorization(b.interactionContext(i), b.interactionPolicy())
	if result.Allowed {
		return true
	}
	_ = b.respondEphemeral(session, i.Interaction, "not authorized to run /"+command+": "+result.Reason)
	return false
}

func (b *Bot) interactionContext(i *discordgo.InteractionCreate) DiscordInteractionContext {
	channelID := strings.TrimSpace(i.ChannelID)
	parentID := ""
	if thread, ok := b.threadForMessageChannel(channelID); ok {
		parentID = thread.parentID
	}
	return DiscordInteractionContext{
		UserID:          interactionUserID(i),
		RoleIDs:         interactionRoleIDs(i),
		ChannelID:       channelID,
		ParentChannelID: parentID,
		IsDM:            strings.TrimSpace(i.GuildID) == "",
	}
}

func (b *Bot) interactionPolicy() DiscordInteractionPolicy {
	allowed := append([]string{}, b.cfg.AllowedChannelIDs...)
	if b.cfg.AllowedChannelID != "" {
		allowed = append([]string{b.cfg.AllowedChannelID}, allowed...)
	}
	return DiscordInteractionPolicy{
		AllowedChannelIDs: compactStrings(allowed),
		IgnoredChannelIDs: b.cfg.IgnoredChannelIDs,
	}
}

func (b *Bot) respondEphemeral(session discordSession, interaction *discordgo.Interaction, content string) error {
	responder, ok := session.(discordInteractionResponder)
	if !ok || interaction == nil {
		return nil
	}
	return responder.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) deferEphemeral(session discordSession, interaction *discordgo.Interaction) error {
	responder, ok := session.(discordInteractionResponder)
	if !ok || interaction == nil {
		return nil
	}
	return responder.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (b *Bot) followupEphemeral(session discordSession, interaction *discordgo.Interaction, content string) error {
	responder, ok := session.(discordInteractionResponder)
	if !ok || interaction == nil {
		return nil
	}
	_, err := responder.FollowupMessageCreate(interaction, false, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	return err
}

func discordOptionString(data discordgo.ApplicationCommandInteractionData, name string) string {
	return interactions.OptionString(data, name)
}

func discordOptionValueString(opt *discordgo.ApplicationCommandInteractionDataOption) string {
	return interactions.OptionValueString(opt)
}

func discordOptionInt(data discordgo.ApplicationCommandInteractionData, name string, fallback int) int {
	return interactions.OptionInt(data, name, fallback)
}

func interactionUserID(i *discordgo.InteractionCreate) string {
	user := interactionUser(i)
	if user == nil {
		return ""
	}
	return strings.TrimSpace(user.ID)
}

func interactionUserName(i *discordgo.InteractionCreate) string {
	user := interactionUser(i)
	if user == nil {
		return ""
	}
	if name := strings.TrimSpace(user.GlobalName); name != "" {
		return name
	}
	return strings.TrimSpace(user.Username)
}

func interactionUser(i *discordgo.InteractionCreate) *discordgo.User {
	if i == nil || i.Interaction == nil {
		return nil
	}
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User
	}
	return i.User
}

func interactionRoleIDs(i *discordgo.InteractionCreate) []string {
	if i == nil || i.Member == nil {
		return nil
	}
	return append([]string(nil), i.Member.Roles...)
}

func normalizeDiscordCommandName(name string) string {
	return interactions.NormalizeCommandName(name)
}

func boundedDiscordDescription(desc, fallback string) string {
	return interactions.BoundedDescription(desc, fallback)
}

func discordCommandPayloadBytes(commands []*discordgo.ApplicationCommand) []byte {
	return interactions.CommandPayloadBytes(commands)
}

func compactStrings(values []string) []string { return channelutil.UniqueStrings(values) }

func sortedDiscordPlatformCommands(commands []gateway.PlatformCommand) []gateway.PlatformCommand {
	out := append([]gateway.PlatformCommand(nil), commands...)
	sort.SliceStable(out, func(i, j int) bool {
		left := normalizeDiscordCommandName(out[i].Name)
		right := normalizeDiscordCommandName(out[j].Name)
		if left != right {
			return left < right
		}
		return out[i].Description < out[j].Description
	})
	return out
}
