package simplex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestSimpleXConfigAndStatus(t *testing.T) {
	missing := ConfigFromEnv(mapLookup(nil))
	if missing.Enabled() {
		t.Fatalf("missing env Enabled() = true, want false")
	}
	if got := missing.MissingConfig(); len(got) != 1 || got[0] != "ws_url" {
		t.Fatalf("MissingConfig = %v, want [ws_url]", got)
	}

	cfg := ConfigFromEnv(mapLookup(map[string]string{
		"SIMPLEX_WS_URL":            "ws://user:secret@127.0.0.1:5225/simplex?token=topsecret",
		"SIMPLEX_ALLOWED_USERS":     " contact-42, member-7 ",
		"SIMPLEX_ALLOW_ALL_USERS":   "true",
		"SIMPLEX_HOME_CHANNEL":      "group:ops",
		"SIMPLEX_HOME_CHANNEL_NAME": "Ops Room",
	}))
	if !cfg.Enabled() || cfg.WSURL == "" {
		t.Fatalf("ConfigFromEnv enabled cfg = %+v", cfg)
	}
	if cfg.HomeDeliveryTarget().ChatID != "group:ops" || cfg.HomeDeliveryTarget().Platform != PlatformName {
		t.Fatalf("HomeDeliveryTarget = %+v", cfg.HomeDeliveryTarget())
	}
	if got := cfg.AllowedUserSet(); !got["contact-42"] || !got["member-7"] || len(got) != 2 {
		t.Fatalf("AllowedUserSet = %v", got)
	}
	if !cfg.AllowAllUsers {
		t.Fatalf("AllowAllUsers = false, want true")
	}

	ch := NewChannel(cfg, nil, nil)
	status := ch.Status(context.Background())
	if status.State != StatusDegraded || status.Code != StatusCodeDaemonUnavailable {
		t.Fatalf("nil-transport Status = %+v, want daemon-unavailable degraded", status)
	}
	if strings.Contains(status.Evidence, "secret") || strings.Contains(status.Evidence, "topsecret") {
		t.Fatalf("Status evidence leaked secret-bearing URL: %q", status.Evidence)
	}

	transport := newFakeTransport()
	transport.healthErr = errors.New("dial tcp ws://user:secret@127.0.0.1:5225: connection refused")
	status = NewChannel(cfg, transport, nil).Status(context.Background())
	if status.State != StatusDegraded || status.Code != StatusCodeDaemonUnavailable {
		t.Fatalf("health-error Status = %+v, want daemon-unavailable degraded", status)
	}
	if strings.Contains(status.Evidence, "secret") {
		t.Fatalf("Status evidence leaked transport error: %q", status.Evidence)
	}

	transport.healthErr = nil
	status = NewChannel(cfg, transport, nil).Status(context.Background())
	if status.State != StatusConnected || status.Code != StatusCodeConnected {
		t.Fatalf("healthy fake Status = %+v, want connected", status)
	}
}

func TestSimpleXInboundNormalization(t *testing.T) {
	transport := newFakeTransport()
	transport.enqueue(simplexDMEvent("msg-1", "contact-42", "Ada", "/start"))
	transport.enqueue(simplexGroupEvent("msg-2", "grp-99", "Ops", "member-7", "Lin", "hello group"))
	transport.enqueue([]byte(`{"corrId":"hermes-echo-1","type":"newChatItem","chatInfo":{"contact":{"contactId":"contact-echo"}},"chatItem":{"content":{"msgContent":{"text":"echo"}}}}`))
	transport.closeEvents()

	ch := NewChannel(Config{WSURL: "ws://127.0.0.1:5225"}, transport, nil)
	inbox := make(chan gateway.InboundEvent, 4)
	if err := ch.Run(context.Background(), inbox); err != nil {
		t.Fatalf("Run: %v", err)
	}
	close(inbox)

	var events []gateway.InboundEvent
	for ev := range inbox {
		events = append(events, ev)
	}
	if len(events) != 2 {
		t.Fatalf("events = %+v, want two normalized events and one echo ignored", events)
	}
	dm := events[0]
	if dm.Platform != PlatformName || dm.ChatID != "contact-42" || dm.ChatType != "dm" || dm.UserID != "contact-42" || dm.Kind != gateway.EventStart || dm.Text != "" || dm.MsgID != "msg-1" {
		t.Fatalf("dm event = %+v", dm)
	}
	group := events[1]
	if group.Platform != PlatformName || group.ChatID != "group:grp-99" || group.ChatType != "group" || group.UserID != "member-7" || group.UserName != "Lin" || group.Text != "hello group" || group.Kind != gateway.EventSubmit {
		t.Fatalf("group event = %+v", group)
	}
}

func TestSimpleXAllowlistAndPairing(t *testing.T) {
	cfg := ConfigFromEnv(mapLookup(map[string]string{
		"SIMPLEX_WS_URL":        "ws://127.0.0.1:5225",
		"SIMPLEX_ALLOWED_USERS": "contact-42,member-7",
	}))
	if cfg.AllowAllUsers {
		t.Fatal("AllowAllUsers = true, want false")
	}
	if allowed := cfg.AllowedUserSet(); !allowed["contact-42"] || allowed["mallory"] {
		t.Fatalf("AllowedUserSet = %+v", allowed)
	}

	transport := newFakeTransport()
	ch := NewChannel(cfg, transport, nil)
	dmEvents, ok := ch.NormalizeEvent(simplexDMEvent("msg-3", "mallory", "Mallory", "hello"))
	if !ok || len(dmEvents) != 1 {
		t.Fatalf("NormalizeEvent dm = %+v ok=%t", dmEvents, ok)
	}
	store := gateway.NewPairingStore(t.TempDir() + "/pairing.json")
	decision, err := gateway.HandleUnauthorizedDM(context.Background(), ch, dmEvents[0], gateway.UnauthorizedDMPolicy{
		Behavior:     gateway.UnauthorizedDMPair,
		PairingStore: store,
	})
	if err != nil {
		t.Fatalf("HandleUnauthorizedDM: %v", err)
	}
	if !decision.Handled || decision.StartAgent || !decision.ReplySent || decision.PairingStatus != gateway.PairingCodeIssued {
		t.Fatalf("decision = %+v, want one pairing prompt without agent start", decision)
	}
	sent := transport.sentCommands()
	if len(sent) != 1 || !strings.HasPrefix(sent[0], "@[mallory] ") || !strings.Contains(sent[0], "gormes pairing approve simplex ") {
		t.Fatalf("sent pairing command = %v", sent)
	}

	groupEvents, ok := ch.NormalizeEvent(simplexGroupEvent("msg-4", "grp-99", "Ops", "intruder", "Eve", "hello"))
	if !ok || len(groupEvents) != 1 {
		t.Fatalf("NormalizeEvent group = %+v ok=%t", groupEvents, ok)
	}
	before := len(transport.sentCommands())
	decision, err = gateway.HandleUnauthorizedDM(context.Background(), ch, groupEvents[0], gateway.UnauthorizedDMPolicy{
		Behavior:     gateway.UnauthorizedDMPair,
		PairingStore: store,
	})
	if err != nil {
		t.Fatalf("HandleUnauthorizedDM(group): %v", err)
	}
	if !decision.Handled || decision.StartAgent || decision.ReplySent {
		t.Fatalf("group decision = %+v, want silent group unauthorized handling", decision)
	}
	if got := len(transport.sentCommands()); got != before {
		t.Fatalf("sent count after group = %d, want %d", got, before)
	}
}

func TestSimpleXOutboundDelivery(t *testing.T) {
	transport := newFakeTransport()
	ch := NewChannel(Config{WSURL: "ws://127.0.0.1:5225", HomeChannel: "contact-home"}, transport, nil)

	msgID, err := ch.Send(context.Background(), "contact-42", "Hello, SimpleX!")
	if err != nil {
		t.Fatalf("Send DM: %v", err)
	}
	if !strings.HasPrefix(msgID, "hermes-") {
		t.Fatalf("Send msgID = %q, want corr id", msgID)
	}
	if _, err := ch.Send(context.Background(), "group:grp-99", "Hello, group!"); err != nil {
		t.Fatalf("Send group: %v", err)
	}
	if _, err := ch.SendImage(context.Background(), "group:grp-99", "https://example.test/image.png", "caption"); err != nil {
		t.Fatalf("SendImage fallback: %v", err)
	}
	stop, err := ch.StartTyping(context.Background(), "contact-42")
	if err != nil {
		t.Fatalf("StartTyping: %v", err)
	}
	stop()
	stop()

	got := transport.sentCommands()
	want := []string{
		"@[contact-42] Hello, SimpleX!",
		"#[grp-99] Hello, group!",
		"#[grp-99] caption\nhttps://example.test/image.png",
	}
	if len(got) != len(want) {
		t.Fatalf("sent commands = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sent[%d] = %q, want %q (all=%v)", i, got[i], want[i], got)
		}
	}
}

type fakeTransport struct {
	mu        sync.Mutex
	events    chan []byte
	sent      [][]byte
	healthErr error
	closed    bool
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{events: make(chan []byte, 16)}
}

func (t *fakeTransport) Connect(context.Context) error { return nil }

func (t *fakeTransport) Health(context.Context) error { return t.healthErr }

func (t *fakeTransport) Receive(ctx context.Context) ([]byte, error) {
	select {
	case payload, ok := <-t.events:
		if !ok {
			return nil, io.EOF
		}
		return payload, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Second):
		return nil, errors.New("fake simplex receive timeout")
	}
}

func (t *fakeTransport) Send(_ context.Context, payload []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	copyPayload := append([]byte(nil), payload...)
	t.sent = append(t.sent, copyPayload)
	return nil
}

func (t *fakeTransport) Close(context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	return nil
}

func (t *fakeTransport) enqueue(payload []byte) { t.events <- payload }
func (t *fakeTransport) closeEvents()           { close(t.events) }

func (t *fakeTransport) sentCommands() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, 0, len(t.sent))
	for _, payload := range t.sent {
		var msg struct {
			Cmd string `json:"cmd"`
		}
		_ = json.Unmarshal(payload, &msg)
		out = append(out, msg.Cmd)
	}
	return out
}

func simplexDMEvent(msgID, contactID, displayName, text string) []byte {
	payload := map[string]any{
		"type": "newChatItem",
		"chatInfo": map[string]any{
			"type": "direct",
			"contact": map[string]any{
				"contactId":   contactID,
				"displayName": displayName,
			},
		},
		"chatItem": map[string]any{
			"meta": map[string]any{
				"itemId":     msgID,
				"itemStatus": map[string]any{"type": "rcvNew"},
				"itemTs":     "2026-05-23T08:00:00Z",
			},
			"content": map[string]any{"msgContent": map[string]any{"text": text}},
		},
	}
	return mustJSON(payload)
}

func simplexGroupEvent(msgID, groupID, groupName, memberID, memberName, text string) []byte {
	payload := map[string]any{
		"type": "newChatItems",
		"chatItems": []any{map[string]any{
			"chatInfo": map[string]any{
				"type": "group",
				"groupInfo": map[string]any{
					"groupId":     groupID,
					"displayName": groupName,
				},
			},
			"chatItem": map[string]any{
				"meta": map[string]any{
					"itemId":     msgID,
					"itemStatus": map[string]any{"type": "rcvNew"},
				},
				"chatItemMember": map[string]any{
					"memberId":    memberID,
					"displayName": memberName,
				},
				"content": map[string]any{"msgContent": map[string]any{"text": text}},
			},
		}},
	}
	return mustJSON(payload)
}

func mustJSON(value any) []byte {
	payload, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return payload
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
