package simplex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const corrPrefix = "hermes-"

type StatusState string

type StatusCode string

const (
	StatusDegraded  StatusState = "degraded"
	StatusConnected StatusState = "connected"

	StatusCodeMissingWSURL      StatusCode = "simplex_missing_ws_url"
	StatusCodeDaemonUnavailable StatusCode = "simplex_daemon_unavailable"
	StatusCodeConnected         StatusCode = "simplex_connected"
)

type StatusReport struct {
	State    StatusState
	Code     StatusCode
	Evidence string
}

// Transport is the seam between the SimpleX channel module and the local
// simplex-chat WebSocket daemon. Tests use an in-process fake transport; live
// gateway runs use WebSocketTransport.
type Transport interface {
	Connect(context.Context) error
	Receive(context.Context) ([]byte, error)
	Send(context.Context, []byte) error
	Health(context.Context) error
	Close(context.Context) error
}

type Channel struct {
	cfg       Config
	transport Transport
	log       *slog.Logger

	mu      sync.Mutex
	counter int64
	pending map[string]struct{}
}

var _ gateway.Channel = (*Channel)(nil)
var _ gateway.TypingCapable = (*Channel)(nil)
var _ gateway.MediaSender = (*Channel)(nil)

func NewChannel(cfg Config, transport Transport, log *slog.Logger) *Channel {
	if log == nil {
		log = slog.Default()
	}
	return &Channel{cfg: cfg, transport: transport, log: log, pending: map[string]struct{}{}}
}

func (c *Channel) Name() string { return PlatformName }

func (c *Channel) Status(ctx context.Context) StatusReport {
	if !c.cfg.Enabled() {
		return StatusReport{State: StatusDegraded, Code: StatusCodeMissingWSURL, Evidence: "simplex: missing ws_url"}
	}
	if c.transport == nil {
		return StatusReport{State: StatusDegraded, Code: StatusCodeDaemonUnavailable, Evidence: "simplex: daemon unavailable ws_url=" + redactEvidence(c.cfg.WSURL)}
	}
	if err := c.transport.Health(ctx); err != nil {
		return StatusReport{State: StatusDegraded, Code: StatusCodeDaemonUnavailable, Evidence: "simplex: daemon unavailable: " + redactEvidence(err.Error())}
	}
	return StatusReport{State: StatusConnected, Code: StatusCodeConnected, Evidence: "simplex: connected ws_url=" + redactEvidence(c.cfg.WSURL)}
}

func (c *Channel) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	if !c.cfg.Enabled() {
		return errors.New(string(StatusCodeMissingWSURL))
	}
	if c.transport == nil {
		return errors.New("simplex_transport_not_configured")
	}
	if err := c.transport.Connect(ctx); err != nil {
		return err
	}
	defer c.transport.Close(context.Background())
	closeOnce := sync.Once{}
	go func() {
		<-ctx.Done()
		closeOnce.Do(func() { _ = c.transport.Close(context.Background()) })
	}()
	for {
		payload, err := c.transport.Receive(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				return nil
			}
			return err
		}
		events, ok := c.NormalizeEvent(payload)
		if !ok {
			continue
		}
		for _, ev := range events {
			select {
			case inbox <- ev:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func (c *Channel) Send(ctx context.Context, chatID, text string) (string, error) {
	if c.transport == nil {
		return "", errors.New("simplex_transport_not_configured")
	}
	corrID := c.nextCorrID()
	payload := map[string]string{
		"corrId": corrID,
		"cmd":    formatSimpleXCommand(chatID, text),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	if err := c.transport.Send(ctx, raw); err != nil {
		return "", err
	}
	return corrID, nil
}

func (c *Channel) SendImage(ctx context.Context, chatID, imageURL, caption string) (string, error) {
	text := strings.TrimSpace(imageURL)
	if strings.TrimSpace(caption) != "" {
		text = strings.TrimSpace(caption) + "\n" + text
	}
	return c.Send(ctx, chatID, text)
}

func (c *Channel) SendMedia(ctx context.Context, chatID, _ string, media gateway.OutboundMedia) (string, error) {
	text := strings.TrimSpace(media.Path)
	if text == "" {
		return "", errors.New("simplex_media_path_missing")
	}
	return c.Send(ctx, chatID, text)
}

func (c *Channel) StartTyping(context.Context, string) (func(), error) {
	return func() {}, nil
}

func (c *Channel) nextCorrID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counter++
	corrID := fmt.Sprintf("%s%d-%d", corrPrefix, time.Now().UnixMilli(), c.counter)
	c.pending[corrID] = struct{}{}
	if len(c.pending) > 200 {
		for key := range c.pending {
			delete(c.pending, key)
			if len(c.pending) <= 100 {
				break
			}
		}
	}
	return corrID
}

func (c *Channel) discardCorrID(corrID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pending, corrID)
}

func formatSimpleXCommand(chatID, text string) string {
	chatID = strings.TrimSpace(chatID)
	text = strings.TrimSpace(text)
	if strings.HasPrefix(chatID, "group:") {
		return "#\u005b" + strings.TrimPrefix(chatID, "group:") + "\u005d " + text
	}
	return "@\u005b" + chatID + "\u005d " + text
}

func (c *Channel) NormalizeEvent(payload []byte) ([]gateway.InboundEvent, bool) {
	var event simplexEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, false
	}
	return c.normalizeEvent(event)
}

func (c *Channel) normalizeEvent(event simplexEvent) ([]gateway.InboundEvent, bool) {
	if corrID := strings.TrimSpace(event.CorrID); strings.HasPrefix(corrID, corrPrefix) {
		c.discardCorrID(corrID)
		return nil, false
	}
	eventType := firstNonEmpty(event.Type, event.Resp.Type)
	switch eventType {
	case "newChatItem":
		ev, ok := normalizeChatItem(event)
		if !ok {
			return nil, false
		}
		return []gateway.InboundEvent{ev}, true
	case "newChatItems":
		out := make([]gateway.InboundEvent, 0, len(event.ChatItems))
		for _, item := range event.ChatItems {
			ev, ok := normalizeChatItem(item)
			if ok {
				out = append(out, ev)
			}
		}
		return out, len(out) > 0
	default:
		return nil, false
	}
}

func normalizeChatItem(wrapper simplexEvent) (gateway.InboundEvent, bool) {
	chatItem := wrapper.ChatItem
	msgContent := chatItem.Content.MsgContent
	text := strings.TrimSpace(firstNonEmpty(msgContent.Text, chatItem.Content.Text))
	if text == "" {
		return gateway.InboundEvent{}, false
	}
	direction := strings.TrimSpace(chatItem.Meta.ItemStatus.Type)
	if strings.HasPrefix(direction, "snd") {
		return gateway.InboundEvent{}, false
	}

	chatInfo := wrapper.ChatInfo
	chatTypeRaw := strings.TrimSpace(chatInfo.Type)
	isGroup := chatTypeRaw == "group" || chatTypeRaw == "groupInfo"
	var chatID, chatName, userID, userName, chatType string
	if isGroup {
		groupID := anyString(firstNonEmpty(chatInfo.GroupInfo.GroupID, chatInfo.GroupInfo.ID, chatInfo.Group.GroupID, chatInfo.Group.ID))
		if groupID == "" {
			return gateway.InboundEvent{}, false
		}
		chatID = "group:" + groupID
		chatName = firstNonEmpty(chatInfo.GroupInfo.DisplayName, chatInfo.GroupInfo.GroupProfile.DisplayName, chatInfo.Group.DisplayName, chatInfo.Group.GroupProfile.DisplayName)
		member := chatItem.ChatItemMember
		userID = anyString(firstNonEmpty(member.MemberID, member.ID, chatID))
		userName = firstNonEmpty(member.DisplayName, member.LocalDisplayName, userID)
		chatType = "group"
	} else {
		contactID := anyString(firstNonEmpty(chatInfo.Contact.ContactID, chatInfo.Contact.ID))
		if contactID == "" {
			return gateway.InboundEvent{}, false
		}
		chatID = contactID
		chatName = firstNonEmpty(chatInfo.Contact.DisplayName, chatInfo.Contact.LocalDisplayName, contactID)
		userID = contactID
		userName = chatName
		chatType = "dm"
	}
	kind, body := gateway.ParseInboundText(text)
	msgID := anyString(firstNonEmpty(chatItem.Meta.ItemID, chatItem.Meta.ID))
	return gateway.InboundEvent{
		Platform:  PlatformName,
		ChatID:    chatID,
		ChatName:  chatName,
		ChatType:  chatType,
		UserID:    userID,
		UserName:  userName,
		MsgID:     msgID,
		MessageID: msgID,
		Kind:      kind,
		Text:      body,
	}, true
}

type simplexEvent struct {
	Type      string          `json:"type"`
	Resp      simplexResp     `json:"resp"`
	CorrID    string          `json:"corrId"`
	ChatInfo  simplexChat     `json:"chatInfo"`
	ChatItem  simplexChatItem `json:"chatItem"`
	ChatItems []simplexEvent  `json:"chatItems"`
}

type simplexResp struct {
	Type string `json:"type"`
}

type simplexChat struct {
	Type      string         `json:"type"`
	Contact   simplexContact `json:"contact"`
	GroupInfo simplexGroup   `json:"groupInfo"`
	Group     simplexGroup   `json:"group"`
}

type simplexContact struct {
	ContactID        string `json:"contactId"`
	ID               string `json:"id"`
	DisplayName      string `json:"displayName"`
	LocalDisplayName string `json:"localDisplayName"`
}

type simplexGroup struct {
	GroupID      string                `json:"groupId"`
	ID           string                `json:"id"`
	DisplayName  string                `json:"displayName"`
	GroupProfile simplexDisplayProfile `json:"groupProfile"`
}

type simplexDisplayProfile struct {
	DisplayName string `json:"displayName"`
}

type simplexChatItem struct {
	Meta           simplexMeta     `json:"meta"`
	Content        simplexContent  `json:"content"`
	ChatItemMember simplexMember   `json:"chatItemMember"`
	File           map[string]any  `json:"file"`
	Raw            json.RawMessage `json:"-"`
}

type simplexMeta struct {
	ItemID     string            `json:"itemId"`
	ID         string            `json:"id"`
	ItemStatus simplexItemStatus `json:"itemStatus"`
}

type simplexItemStatus struct {
	Type string `json:"type"`
}

type simplexContent struct {
	Text       string            `json:"text"`
	MsgContent simplexMsgContent `json:"msgContent"`
}

type simplexMsgContent struct {
	Text string `json:"text"`
}

type simplexMember struct {
	MemberID         string `json:"memberId"`
	ID               string `json:"id"`
	DisplayName      string `json:"displayName"`
	LocalDisplayName string `json:"localDisplayName"`
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func anyString(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return value
}

func stringifyAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
