// Package teams provides a fakeable Microsoft Teams gateway seam. The live Bot
// Framework HTTP binding is intentionally deferred; this package owns the
// channel-neutral runtime contract and Hermes-compatible metadata.
package teams

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const (
	PlatformName     = "teams"
	DefaultPort      = 3978
	MaxMessageLength = 28000
	cacheFailedCode  = "teams_attachment_cache_failed"
)

var mentionTagPattern = regexp.MustCompile(`(?i)<at>[^<]*</at>\s*`)

type PluginInfo struct {
	Name             string
	Label            string
	RequiredEnv      []string
	AllowedUsersEnv  string
	AllowAllEnv      string
	MaxMessageLength int
	SetupAvailable   bool
	PlatformHint     string
}

func PluginMetadata() PluginInfo {
	return PluginInfo{
		Name:             PlatformName,
		Label:            "Microsoft Teams",
		RequiredEnv:      []string{"TEAMS_CLIENT_ID", "TEAMS_CLIENT_SECRET", "TEAMS_TENANT_ID"},
		AllowedUsersEnv:  "TEAMS_ALLOWED_USERS",
		AllowAllEnv:      "TEAMS_ALLOW_ALL_USERS",
		MaxMessageLength: MaxMessageLength,
		SetupAvailable:   true,
		PlatformHint:     "You are chatting via Microsoft Teams. Keep responses clear and professional.",
	}
}

type Activity struct {
	ID               string
	Text             string
	FromID           string
	FromAADID        string
	FromName         string
	ConversationID   string
	ConversationName string
	ConversationType string
	TenantID         string
	Attachments      []Attachment
}

type Attachment struct {
	ContentURL  string
	ContentType string
	Name        string
}

type Client interface {
	Connect(ctx context.Context) error
	Run(ctx context.Context, deliver func(context.Context, Activity)) error
	SendText(ctx context.Context, conversationID, text string) (string, error)
	SendTyping(ctx context.Context, conversationID string) error
	SendApprovalCard(ctx context.Context, conversationID string, card ApprovalCard) (string, error)
}

type ImageCacheFunc func(ctx context.Context, url, contentType string) (string, error)

type ApprovalStore interface {
	HasBlockingApproval(sessionKey string) bool
	ResolveApproval(sessionKey, choice string) error
}

type Config struct {
	ClientID        string
	ClientSecret    string
	TenantID        string
	Port            int
	AllowedUsers    []string
	AllowAllUsers   bool
	CacheImage      ImageCacheFunc
	ApprovalStore   ApprovalStore
	MaxMessageBytes int
}

type Channel struct {
	cfg       Config
	client    Client
	log       *slog.Logger
	delivered map[string]bool
}

var _ gateway.Channel = (*Channel)(nil)

func NewChannel(cfg Config, client Client, log *slog.Logger) *Channel {
	if log == nil {
		log = slog.Default()
	}
	return &Channel{
		cfg:       cfg,
		client:    client,
		log:       log,
		delivered: make(map[string]bool),
	}
}

func (c *Channel) Name() string { return PlatformName }

func (c *Channel) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	if c.client == nil {
		return errors.New("teams_client_unavailable")
	}
	if err := c.client.Connect(ctx); err != nil {
		return err
	}
	return c.client.Run(ctx, func(ctx context.Context, activity Activity) {
		ev, ok := c.NormalizeActivity(ctx, activity)
		if !ok {
			return
		}
		select {
		case inbox <- ev:
		case <-ctx.Done():
		}
	})
}

func (c *Channel) Send(ctx context.Context, chatID, text string) (string, error) {
	if c.client == nil {
		return "", errors.New("teams_client_unavailable")
	}
	chunks := chunkText(text, c.maxMessageBytes())
	var last string
	for _, chunk := range chunks {
		id, err := c.client.SendText(ctx, chatID, chunk)
		if err != nil {
			return "", err
		}
		last = id
	}
	return last, nil
}

func (c *Channel) SendTyping(ctx context.Context, chatID string) error {
	if c.client == nil {
		return errors.New("teams_client_unavailable")
	}
	return c.client.SendTyping(ctx, chatID)
}

func (c *Channel) NormalizeActivity(ctx context.Context, activity Activity) (gateway.InboundEvent, bool) {
	id := strings.TrimSpace(activity.ID)
	if id != "" {
		if c.delivered[id] {
			return gateway.InboundEvent{}, false
		}
	}
	botID := strings.TrimSpace(c.cfg.ClientID)
	if botID != "" && strings.TrimSpace(activity.FromID) == botID {
		return gateway.InboundEvent{}, false
	}

	chatID := strings.TrimSpace(activity.ConversationID)
	userID := firstNonEmpty(activity.FromAADID, activity.FromID)
	text := strings.TrimSpace(stripTeamsMentions(activity.Text))
	attachments := c.normalizeAttachments(ctx, activity.Attachments)
	if chatID == "" || strings.TrimSpace(userID) == "" || (text == "" && len(attachments) == 0) {
		return gateway.InboundEvent{}, false
	}

	if id != "" {
		c.delivered[id] = true
	}
	kind, body := gateway.ParseInboundText(text)
	return gateway.InboundEvent{
		Platform:    PlatformName,
		ChatID:      chatID,
		ChatName:    strings.TrimSpace(activity.ConversationName),
		ChatType:    teamsChatType(activity.ConversationType),
		UserID:      strings.TrimSpace(userID),
		UserName:    strings.TrimSpace(activity.FromName),
		GuildID:     firstNonEmpty(activity.TenantID, c.cfg.TenantID),
		MsgID:       id,
		MessageID:   id,
		Kind:        kind,
		Text:        body,
		Attachments: attachments,
	}, true
}

func (c *Channel) normalizeAttachments(ctx context.Context, attachments []Attachment) []gateway.Attachment {
	out := []gateway.Attachment{}
	for _, attachment := range attachments {
		contentURL := strings.TrimSpace(attachment.ContentURL)
		contentType := strings.TrimSpace(attachment.ContentType)
		if contentURL == "" || !strings.HasPrefix(strings.ToLower(contentType), "image/") {
			continue
		}
		item := gateway.Attachment{
			Kind:      "photo",
			MediaType: contentType,
			FileName:  strings.TrimSpace(attachment.Name),
			SourceID:  contentURL,
		}
		if c.cfg.CacheImage == nil {
			item.URL = contentURL
			out = append(out, item)
			continue
		}
		cached, err := c.cfg.CacheImage(ctx, contentURL, contentType)
		if err != nil || strings.TrimSpace(cached) == "" {
			item.Error = cacheFailedCode
			out = append(out, item)
			continue
		}
		item.URL = strings.TrimSpace(cached)
		out = append(out, item)
	}
	return out
}

type ApprovalRequest struct {
	Command     string
	SessionKey  string
	Description string
}

type ApprovalCard struct {
	ConversationID string
	Title          string
	CommandPreview string
	Description    string
	Actions        []ApprovalCardAction
}

type ApprovalCardAction struct {
	Title string
	Data  map[string]string
	Style string
}

func (c *Channel) SendExecApproval(ctx context.Context, chatID string, req ApprovalRequest) (string, error) {
	if c.client == nil {
		return "", errors.New("teams_client_unavailable")
	}
	req.Command = sanitizeTeamsApprovalText(req.Command)
	req.Description = sanitizeTeamsApprovalText(req.Description)
	card := ApprovalCard{
		Title:          "Command Approval Required",
		CommandPreview: bounded(req.Command, 2000),
		Description:    req.Description,
		Actions: []ApprovalCardAction{
			{Title: "Allow Once", Style: "positive", Data: approvalData(req, "approve_once")},
			{Title: "Allow Session", Data: approvalData(req, "approve_session")},
			{Title: "Always Allow", Data: approvalData(req, "approve_always")},
			{Title: "Deny", Style: "destructive", Data: approvalData(req, "deny")},
		},
	}
	return c.client.SendApprovalCard(ctx, chatID, card)
}

type ApprovalAction struct {
	ClickerAADID string
	ClickerID    string
	SessionKey   string
	Action       string
}

type ApprovalActionResult struct {
	Status  string
	Message string
	Choice  string
}

func (c *Channel) HandleApprovalAction(action ApprovalAction) ApprovalActionResult {
	sessionKey := strings.TrimSpace(action.SessionKey)
	if sessionKey == "" {
		return ApprovalActionResult{Status: "unknown", Message: "Unknown action."}
	}
	if !c.approvalClickAllowed(action) {
		return ApprovalActionResult{Status: "denied", Message: "Not authorized."}
	}
	choice, ok := mapApprovalChoice(action.Action)
	if !ok {
		return ApprovalActionResult{Status: "unknown", Message: "Unknown action."}
	}
	if c.cfg.ApprovalStore == nil || !c.cfg.ApprovalStore.HasBlockingApproval(sessionKey) {
		return ApprovalActionResult{Status: "expired", Message: "Approval already resolved or expired."}
	}
	if err := c.cfg.ApprovalStore.ResolveApproval(sessionKey, choice); err != nil {
		return ApprovalActionResult{Status: "expired", Message: "Approval already resolved or expired."}
	}
	return ApprovalActionResult{Status: "resolved", Choice: choice, Message: approvalChoiceLabel(choice)}
}

func (c *Channel) approvalClickAllowed(action ApprovalAction) bool {
	if c.cfg.AllowAllUsers || len(c.cfg.AllowedUsers) == 0 {
		return true
	}
	clicker := strings.TrimSpace(firstNonEmpty(action.ClickerAADID, action.ClickerID))
	for _, allowed := range c.cfg.AllowedUsers {
		allowed = strings.TrimSpace(allowed)
		if allowed == "*" || strings.EqualFold(allowed, clicker) {
			return true
		}
	}
	return false
}

func approvalData(req ApprovalRequest, action string) map[string]string {
	return map[string]string{
		"session_key":   strings.TrimSpace(req.SessionKey),
		"cmd":           bounded(sanitizeTeamsApprovalText(req.Command), 200),
		"desc":          sanitizeTeamsApprovalText(req.Description),
		"hermes_action": action,
	}
}

func sanitizeTeamsApprovalText(value string) string {
	replacer := strings.NewReplacer(
		"`", "'",
		"*", "'",
		"[", "(",
		"]", ")",
	)
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func mapApprovalChoice(action string) (string, bool) {
	switch strings.TrimSpace(action) {
	case "approve_once":
		return "once", true
	case "approve_session":
		return "session", true
	case "approve_always":
		return "always", true
	case "deny":
		return "deny", true
	default:
		return "", false
	}
}

func approvalChoiceLabel(choice string) string {
	switch choice {
	case "once":
		return "Allowed (once)"
	case "session":
		return "Allowed (session)"
	case "always":
		return "Always allowed"
	case "deny":
		return "Denied"
	default:
		return "Resolved"
	}
}

func stripTeamsMentions(text string) string {
	return strings.TrimSpace(mentionTagPattern.ReplaceAllString(text, ""))
}

func teamsChatType(value string) string {
	switch strings.TrimSpace(value) {
	case "groupChat":
		return "group"
	case "channel":
		return "channel"
	default:
		return "dm"
	}
}

func (c *Channel) maxMessageBytes() int {
	if c.cfg.MaxMessageBytes > 0 && c.cfg.MaxMessageBytes < MaxMessageLength {
		return c.cfg.MaxMessageBytes
	}
	return MaxMessageLength
}

func chunkText(text string, limit int) []string {
	if limit <= 0 {
		limit = MaxMessageLength
	}
	runes := []rune(text)
	if len(runes) == 0 {
		return []string{""}
	}
	out := make([]string, 0, (len(runes)/limit)+1)
	for len(runes) > limit {
		out = append(out, string(runes[:limit]))
		runes = runes[limit:]
	}
	out = append(out, string(runes))
	return out
}

func bounded(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit]) + "..."
}

func firstNonEmpty(values ...string) string { return channelutil.FirstNonEmpty(values...) }
