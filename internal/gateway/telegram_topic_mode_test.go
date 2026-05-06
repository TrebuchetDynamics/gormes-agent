package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestTelegramTopicHelpDoesNotTouchStore(t *testing.T) {
	store := newFakeTelegramTopicStore()
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:       map[string]string{"telegram": "42"},
		TelegramTopicStore: store,
	}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	if err := m.handleInbound(context.Background(), telegramTopicEvent("/topic help", "7")); err != nil {
		t.Fatal(err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1: %#v", len(sent), sent)
	}
	for _, want := range []string{"/topic help", "/topic off", "/topic <id>"} {
		if !strings.Contains(sent[0].Text, want) {
			t.Fatalf("help text missing %q:\n%s", want, sent[0].Text)
		}
	}
	if store.calls != 0 {
		t.Fatalf("/topic help touched store %d time(s), want 0", store.calls)
	}
}

func TestTelegramTopicOffDisablesAndClearsBindingsIdempotently(t *testing.T) {
	store := newFakeTelegramTopicStore()
	store.enabled["42\x007"] = true
	store.bindings["42\x0017585"] = "topic-sess"
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:       map[string]string{"telegram": "42"},
		TelegramTopicStore: store,
	}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	if err := m.handleInbound(context.Background(), telegramTopicEvent("/topic off", "7")); err != nil {
		t.Fatal(err)
	}
	if store.enabled["42\x007"] {
		t.Fatal("topic mode still enabled after /topic off")
	}
	if _, ok := store.bindings["42\x0017585"]; ok {
		t.Fatal("topic binding survived /topic off")
	}
	sent := ch.sentSnapshot()
	if len(sent) != 1 || !strings.Contains(strings.ToLower(sent[0].Text), "off") {
		t.Fatalf("first /topic off reply = %#v, want visible off confirmation", sent)
	}

	if err := m.handleInbound(context.Background(), telegramTopicEvent("/topic off", "7")); err != nil {
		t.Fatal(err)
	}
	sent = ch.sentSnapshot()
	if len(sent) != 2 || !strings.Contains(sent[1].Text, "not currently enabled") {
		t.Fatalf("second /topic off reply = %#v, want idempotent no-op guidance", sent)
	}
}

func TestTelegramTopicMutationRefusesUnauthorizedUser(t *testing.T) {
	store := newFakeTelegramTopicStore()
	store.enabled["42\x007"] = true
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:       map[string]string{"telegram": "42"},
		AllowedUsers:       map[string]map[string]bool{"telegram": {"operator": true}},
		TelegramTopicStore: store,
	}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	if err := m.handleInbound(context.Background(), telegramTopicEvent("/topic off", "7")); err != nil {
		t.Fatal(err)
	}

	if !store.enabled["42\x007"] {
		t.Fatal("unauthorized /topic mutation changed topic mode")
	}
	if store.disableCalls != 0 || store.enableCalls != 0 {
		t.Fatalf("unauthorized /topic mutation touched store: enable=%d disable=%d", store.enableCalls, store.disableCalls)
	}
	sent := ch.sentSnapshot()
	if len(sent) != 1 || !strings.Contains(strings.ToLower(sent[0].Text), "not authorized") {
		t.Fatalf("unauthorized reply = %#v, want auth-denied guidance", sent)
	}
}

func TestTelegramTopicCapabilityGuidanceDebouncedAndResetByOff(t *testing.T) {
	store := newFakeTelegramTopicStore()
	store.enabled["42\x007"] = true
	now := time.Unix(1000, 0)
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:       map[string]string{"telegram": "42"},
		TelegramTopicStore: store,
		Now: func() time.Time {
			return now
		},
		TelegramTopicCapabilities: func(context.Context, InboundEvent) (TelegramTopicCapabilities, error) {
			return TelegramTopicCapabilities{
				Checked:                   true,
				HasTopicsEnabled:          false,
				AllowsUsersToCreateTopics: true,
			}, nil
		},
	}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := m.handleInbound(context.Background(), telegramTopicEvent("/topic", "7")); err != nil {
			t.Fatal(err)
		}
	}
	sent := ch.sentSnapshot()
	if len(sent) != 2 {
		t.Fatalf("sent count = %d, want 2: %#v", len(sent), sent)
	}
	if !strings.Contains(sent[0].Text, "Open @BotFather") {
		t.Fatalf("first capability reply missing BotFather guidance:\n%s", sent[0].Text)
	}
	if strings.Contains(sent[1].Text, "Open @BotFather") {
		t.Fatalf("second capability reply repeated BotFather guidance inside debounce window:\n%s", sent[1].Text)
	}
	if !strings.Contains(sent[1].Text, "telegram_topic_capability_hint_debounced") {
		t.Fatalf("second capability reply missing debounce evidence:\n%s", sent[1].Text)
	}

	if err := m.handleInbound(context.Background(), telegramTopicEvent("/topic off", "7")); err != nil {
		t.Fatal(err)
	}
	if err := m.handleInbound(context.Background(), telegramTopicEvent("/topic", "7")); err != nil {
		t.Fatal(err)
	}
	sent = ch.sentSnapshot()
	if len(sent) != 4 || !strings.Contains(sent[3].Text, "Open @BotFather") {
		t.Fatalf("/topic off did not reset capability debounce; sent=%#v", sent)
	}

	now = now.Add(5*time.Minute + time.Second)
	if err := m.handleInbound(context.Background(), telegramTopicEvent("/topic", "7")); err != nil {
		t.Fatal(err)
	}
	sent = ch.sentSnapshot()
	if len(sent) != 5 || !strings.Contains(sent[4].Text, "Open @BotFather") {
		t.Fatalf("capability hint did not reopen after cooldown; sent=%#v", sent)
	}
}

func telegramTopicEvent(text, userID string) InboundEvent {
	return InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		ChatType: "private",
		UserID:   userID,
		MsgID:    "msg-1",
		Kind:     EventTopic,
		Text:     text,
	}
}

type fakeTelegramTopicStore struct {
	enabled      map[string]bool
	bindings     map[string]string
	calls        int
	enableCalls  int
	disableCalls int
}

func newFakeTelegramTopicStore() *fakeTelegramTopicStore {
	return &fakeTelegramTopicStore{
		enabled:  map[string]bool{},
		bindings: map[string]string{},
	}
}

func (s *fakeTelegramTopicStore) IsTelegramTopicModeEnabled(_ context.Context, chatID, userID string) (bool, error) {
	s.calls++
	return s.enabled[chatID+"\x00"+userID], nil
}

func (s *fakeTelegramTopicStore) EnableTelegramTopicMode(_ context.Context, record TelegramTopicModeRecord) error {
	s.calls++
	s.enableCalls++
	s.enabled[record.ChatID+"\x00"+record.UserID] = true
	return nil
}

func (s *fakeTelegramTopicStore) DisableTelegramTopicMode(_ context.Context, chatID string) error {
	s.calls++
	s.disableCalls++
	for key := range s.enabled {
		if strings.HasPrefix(key, chatID+"\x00") {
			s.enabled[key] = false
		}
	}
	for key := range s.bindings {
		if strings.HasPrefix(key, chatID+"\x00") {
			delete(s.bindings, key)
		}
	}
	return nil
}
