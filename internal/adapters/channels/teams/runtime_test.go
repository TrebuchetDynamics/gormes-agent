package teams

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

type fakeTeamsClient struct {
	connectErr error

	inboundMu sync.Mutex
	inbound   []Activity

	sentText     []string
	sentTyping   []string
	sentApproval []ApprovalCard
}

func (c *fakeTeamsClient) queue(activity Activity) {
	c.inboundMu.Lock()
	defer c.inboundMu.Unlock()
	c.inbound = append(c.inbound, activity)
}

func (c *fakeTeamsClient) Connect(context.Context) error {
	return c.connectErr
}

func (c *fakeTeamsClient) Run(ctx context.Context, deliver func(context.Context, Activity)) error {
	for {
		c.inboundMu.Lock()
		if len(c.inbound) == 0 {
			c.inboundMu.Unlock()
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Millisecond):
				continue
			}
		}
		next := c.inbound[0]
		c.inbound = c.inbound[1:]
		c.inboundMu.Unlock()
		deliver(ctx, next)
	}
}

func (c *fakeTeamsClient) SendText(_ context.Context, conversationID, text string) (string, error) {
	c.sentText = append(c.sentText, conversationID+"="+text)
	return "teams-msg-1", nil
}

func (c *fakeTeamsClient) SendTyping(_ context.Context, conversationID string) error {
	c.sentTyping = append(c.sentTyping, conversationID)
	return nil
}

func (c *fakeTeamsClient) SendApprovalCard(_ context.Context, conversationID string, card ApprovalCard) (string, error) {
	card.ConversationID = conversationID
	c.sentApproval = append(c.sentApproval, card)
	return "teams-card-1", nil
}

func TestTeamsPluginMetadataMatchesHermesPlugin(t *testing.T) {
	meta := PluginMetadata()
	if meta.Name != PlatformName || meta.Label != "Microsoft Teams" {
		t.Fatalf("metadata identity = %+v", meta)
	}
	if !reflect.DeepEqual(meta.RequiredEnv, []string{"TEAMS_CLIENT_ID", "TEAMS_CLIENT_SECRET", "TEAMS_TENANT_ID"}) {
		t.Fatalf("RequiredEnv = %v", meta.RequiredEnv)
	}
	if meta.AllowedUsersEnv != "TEAMS_ALLOWED_USERS" || meta.AllowAllEnv != "TEAMS_ALLOW_ALL_USERS" {
		t.Fatalf("auth env metadata = %+v", meta)
	}
	if meta.MaxMessageLength != MaxMessageLength {
		t.Fatalf("MaxMessageLength = %d, want %d", meta.MaxMessageLength, MaxMessageLength)
	}
	if !meta.SetupAvailable || !strings.Contains(meta.PlatformHint, "Microsoft Teams") {
		t.Fatalf("metadata missing setup/platform hint: %+v", meta)
	}
}

func TestTeamsInboundActivityNormalization(t *testing.T) {
	client := &fakeTeamsClient{}
	client.queue(Activity{
		ID:               "activity-1",
		Text:             "<at>Gormes</at> /status",
		FromID:           "teams-user-id",
		FromAADID:        "aad-object-id",
		FromName:         "Ada",
		ConversationID:   "conv-dm",
		ConversationName: "Ada chat",
		ConversationType: "personal",
		TenantID:         "tenant-1",
	})
	client.queue(Activity{
		ID:               "activity-2",
		Text:             "hello group",
		FromID:           "group-user",
		FromName:         "Grace",
		ConversationID:   "conv-group",
		ConversationName: "Group Chat",
		ConversationType: "groupChat",
		TenantID:         "tenant-1",
	})
	client.queue(Activity{
		ID:               "activity-3",
		Text:             "hello channel",
		FromID:           "channel-user",
		FromName:         "Linus",
		ConversationID:   "conv-channel",
		ConversationName: "Engineering",
		ConversationType: "channel",
		TenantID:         "tenant-2",
	})
	client.queue(Activity{
		ID:             "activity-self",
		Text:           "self echo",
		FromID:         "bot-id",
		ConversationID: "conv-dm",
	})
	client.queue(Activity{
		ID:             "activity-3",
		Text:           "duplicate",
		FromID:         "channel-user",
		ConversationID: "conv-channel",
	})

	ch := NewChannel(Config{ClientID: "bot-id", TenantID: "tenant-fallback"}, client, slog.Default())
	inbox := make(chan gateway.InboundEvent, 4)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() { _ = ch.Run(ctx, inbox) }()

	got := collectTeamsEvents(t, inbox, 3)
	if got[0].Platform != PlatformName || got[0].ChatType != "dm" || got[0].Kind != gateway.EventStatus {
		t.Fatalf("dm event = %+v", got[0])
	}
	if got[0].UserID != "aad-object-id" || got[0].UserName != "Ada" || got[0].Text != "" || got[0].GuildID != "tenant-1" {
		t.Fatalf("dm provenance/text = %+v", got[0])
	}
	if got[1].ChatType != "group" || got[1].UserID != "group-user" || got[1].Text != "hello group" {
		t.Fatalf("group event = %+v", got[1])
	}
	if got[2].ChatType != "channel" || got[2].MessageID != "activity-3" || got[2].MsgID != "activity-3" {
		t.Fatalf("channel event = %+v", got[2])
	}
}

func TestTeamsImageAttachmentNormalization(t *testing.T) {
	cacheCalls := 0
	ch := NewChannel(Config{
		ClientID: "bot-id",
		CacheImage: func(_ context.Context, url, contentType string) (string, error) {
			cacheCalls++
			if strings.Contains(url, "broken") {
				return "", errors.New("cache denied")
			}
			return "/tmp/gormes-cache/team-image.png", nil
		},
	}, &fakeTeamsClient{}, slog.Default())

	ev, ok := ch.NormalizeActivity(context.Background(), Activity{
		ID:             "activity-image",
		Text:           "see image",
		FromID:         "user-1",
		ConversationID: "conv-image",
		Attachments: []Attachment{
			{ContentURL: "https://cdn.example/image.png", ContentType: "image/png"},
			{ContentURL: "https://cdn.example/broken.png", ContentType: "image/jpeg"},
			{ContentURL: "https://cdn.example/doc.pdf", ContentType: "application/pdf"},
		},
	})
	if !ok {
		t.Fatal("NormalizeActivity ok = false, want true")
	}
	if cacheCalls != 2 {
		t.Fatalf("cacheCalls = %d, want image attachments only", cacheCalls)
	}
	if len(ev.Attachments) != 2 {
		t.Fatalf("attachments = %+v, want cached image plus degraded image", ev.Attachments)
	}
	if ev.Attachments[0].Kind != "photo" || ev.Attachments[0].URL != "/tmp/gormes-cache/team-image.png" {
		t.Fatalf("cached attachment = %+v", ev.Attachments[0])
	}
	if ev.Attachments[1].Error != "teams_attachment_cache_failed" || ev.Attachments[1].URL != "" {
		t.Fatalf("degraded attachment = %+v", ev.Attachments[1])
	}
}

func TestTeamsApprovalCardSanitizesPromptBreakingMarkup(t *testing.T) {
	client := &fakeTeamsClient{}
	ch := NewChannel(Config{ClientID: "bot-id"}, client, slog.Default())

	_, err := ch.SendExecApproval(context.Background(), "conv-1", ApprovalRequest{
		Command:     "rm -rf\n**approve everything** `token`",
		SessionKey:  "sess-1",
		Description: "dangerous\n[click me](https://evil.example) `secret`",
	})
	if err != nil {
		t.Fatalf("SendExecApproval: %v", err)
	}
	if len(client.sentApproval) != 1 {
		t.Fatalf("sent approval cards = %+v, want one", client.sentApproval)
	}
	card := client.sentApproval[0]
	for _, text := range []string{card.CommandPreview, card.Description, card.Actions[0].Data["cmd"], card.Actions[0].Data["desc"]} {
		for _, forbidden := range []string{"\n", "**approve", "`token`", "[click me](https://evil.example)", "`secret`"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("approval card leaked Teams markup %q in %q", forbidden, text)
			}
		}
	}
	if !strings.Contains(card.CommandPreview, "rm -rf ''approve everything'' 'token'") {
		t.Fatalf("CommandPreview = %q, want sanitized command", card.CommandPreview)
	}
	if !strings.Contains(card.Description, "dangerous (click me)(https://evil.example) 'secret'") {
		t.Fatalf("Description = %q, want sanitized description", card.Description)
	}
}

func TestTeamsApprovalActionCarriesTicketID(t *testing.T) {
	store := newFakeApprovalStore()
	store.blocking["sess-1"] = true
	ch := NewChannel(Config{
		ClientID:      "bot-id",
		AllowAllUsers: true,
		ApprovalStore: store,
	}, &fakeTeamsClient{}, slog.Default())

	_, err := ch.SendExecApproval(context.Background(), "conv-1", ApprovalRequest{
		Command:     "rm -rf /tmp/project",
		SessionKey:  "sess-1",
		TicketID:    99,
		Description: "dangerous command",
	})
	if err != nil {
		t.Fatalf("SendExecApproval: %v", err)
	}
	card := ch.client.(*fakeTeamsClient).sentApproval[0]
	if got := card.Actions[0].Data["ticket_id"]; got != "99" {
		t.Fatalf("approval action ticket_id = %q, want 99", got)
	}

	got := ch.HandleApprovalAction(ApprovalAction{
		ClickerAADID: "aad-allowed",
		SessionKey:   "sess-1",
		TicketID:     99,
		Action:       "approve_once",
	})
	if got.Status != "resolved" || store.resolved != "sess-1=99=once" {
		t.Fatalf("ticketed approval action = %+v store=%q, want ticketed resolve", got, store.resolved)
	}
}

func TestTeamsApprovalActionCanonicalizesClickerID(t *testing.T) {
	store := newFakeApprovalStore()
	store.blocking["sess-1"] = true
	ch := NewChannel(Config{
		ClientID:      "bot-id",
		AllowedUsers:  []string{" AAD-ALLOWED "},
		ApprovalStore: store,
	}, &fakeTeamsClient{}, slog.Default())

	got := ch.HandleApprovalAction(ApprovalAction{
		ClickerAADID: " aad-allowed ",
		SessionKey:   "sess-1",
		Action:       "approve_once",
	})
	if got.Status != "resolved" || store.resolved != "sess-1=once" {
		t.Fatalf("canonicalized approval action = %+v store=%q, want resolved", got, store.resolved)
	}
}

func TestTeamsSendTextTypingAndApprovalCards(t *testing.T) {
	client := &fakeTeamsClient{}
	store := newFakeApprovalStore()
	store.blocking["sess-1"] = true
	ch := NewChannel(Config{
		ClientID:        "bot-id",
		AllowedUsers:    []string{"aad-allowed"},
		ApprovalStore:   store,
		MaxMessageBytes: 12,
	}, client, slog.Default())

	msgID, err := ch.Send(context.Background(), "conv-1", "hello Teams, split me")
	if err != nil || msgID != "teams-msg-1" {
		t.Fatalf("Send = %q/%v", msgID, err)
	}
	if len(client.sentText) < 2 {
		t.Fatalf("sent text chunks = %v, want split by MaxMessageBytes", client.sentText)
	}
	if err := ch.SendTyping(context.Background(), "conv-1"); err != nil {
		t.Fatalf("SendTyping: %v", err)
	}
	if !reflect.DeepEqual(client.sentTyping, []string{"conv-1"}) {
		t.Fatalf("sent typing = %v", client.sentTyping)
	}

	cardID, err := ch.SendExecApproval(context.Background(), "conv-1", ApprovalRequest{
		Command:     strings.Repeat("x", 260),
		SessionKey:  "sess-1",
		Description: "dangerous command",
	})
	if err != nil || cardID != "teams-card-1" {
		t.Fatalf("SendExecApproval = %q/%v", cardID, err)
	}
	if len(client.sentApproval) != 1 || len(client.sentApproval[0].Actions) != 4 {
		t.Fatalf("sent approval cards = %+v", client.sentApproval)
	}
	if got := client.sentApproval[0].Actions[0].Data["cmd"]; len(got) > 203 {
		t.Fatalf("button cmd preview length = %d, want bounded", len(got))
	}

	unauth := ch.HandleApprovalAction(ApprovalAction{
		ClickerAADID: "aad-other",
		SessionKey:   "sess-1",
		Action:       "approve_once",
	})
	if unauth.Status != "denied" || !strings.Contains(unauth.Message, "Not authorized") {
		t.Fatalf("unauthorized action result = %+v", unauth)
	}
	if store.resolved != "" {
		t.Fatalf("approval store resolved after unauthorized click: %q", store.resolved)
	}

	allowed := ch.HandleApprovalAction(ApprovalAction{
		ClickerAADID: "aad-allowed",
		SessionKey:   "sess-1",
		Action:       "approve_session",
	})
	if allowed.Status != "resolved" || store.resolved != "sess-1=session" {
		t.Fatalf("allowed action = %+v store=%q", allowed, store.resolved)
	}
	again := ch.HandleApprovalAction(ApprovalAction{
		ClickerAADID: "aad-allowed",
		SessionKey:   "sess-1",
		Action:       "deny",
	})
	if again.Status != "expired" {
		t.Fatalf("already-resolved action = %+v", again)
	}
}

func TestTeamsChannel_RunSurfacesConnectFailure(t *testing.T) {
	connectErr := errors.New("teams connect denied")
	ch := NewChannel(Config{ClientID: "bot-id"}, &fakeTeamsClient{connectErr: connectErr}, slog.Default())
	err := ch.Run(context.Background(), make(chan gateway.InboundEvent, 1))
	if !errors.Is(err, connectErr) {
		t.Fatalf("Run = %v, want %v", err, connectErr)
	}
}

func collectTeamsEvents(t *testing.T, inbox <-chan gateway.InboundEvent, count int) []gateway.InboundEvent {
	t.Helper()
	got := make([]gateway.InboundEvent, 0, count)
	deadline := time.After(time.Second)
	for len(got) < count {
		select {
		case ev := <-inbox:
			got = append(got, ev)
		case <-deadline:
			t.Fatalf("collected %d events, want %d", len(got), count)
		}
	}
	return got
}

type fakeApprovalStore struct {
	blocking map[string]bool
	resolved string
}

func newFakeApprovalStore() *fakeApprovalStore {
	return &fakeApprovalStore{blocking: map[string]bool{}}
}

func (s *fakeApprovalStore) HasBlockingApproval(sessionKey string) bool {
	return s.blocking[sessionKey]
}

func (s *fakeApprovalStore) ResolveApproval(sessionKey, choice string) error {
	if !s.blocking[sessionKey] {
		return errors.New("approval not pending")
	}
	s.blocking[sessionKey] = false
	s.resolved = sessionKey + "=" + choice
	return nil
}

func (s *fakeApprovalStore) ResolveApprovalWithTicket(sessionKey string, ticketID uint64, choice string) error {
	if !s.blocking[sessionKey] {
		return errors.New("approval not pending")
	}
	s.blocking[sessionKey] = false
	s.resolved = sessionKey + "=" + strconv.FormatUint(ticketID, 10) + "=" + choice
	return nil
}
