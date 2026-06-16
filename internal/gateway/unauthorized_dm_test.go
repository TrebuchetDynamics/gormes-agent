package gateway

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/unauthorizeddmtest"
)

func TestUnauthorizedDM_DenyModeSendsDeterministicDenialAndRecordsEvidence(t *testing.T) {
	store := newUnauthorizedDMTestStore(t)
	ch := newFakeChannel("telegram")
	ev := InboundEvent{
		Platform: "telegram",
		ChatID:   "unauthorized-dm",
		ChatType: "private",
		UserID:   "stranger",
		UserName: "Mallory",
		Kind:     EventSubmit,
		Text:     "hello",
	}

	decision, err := HandleUnauthorizedDM(context.Background(), ch, ev, UnauthorizedDMPolicy{
		Behavior:     UnauthorizedDMDeny,
		PairingStore: store,
	})
	if err != nil {
		t.Fatalf("HandleUnauthorizedDM: %v", err)
	}
	if !decision.Handled || decision.StartAgent || !decision.ReplySent {
		t.Fatalf("decision = %#v, want handled denial reply without agent start", decision)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent = %#v, want one deterministic denial", sent)
	}
	if sent[0].ChatID != "unauthorized-dm" {
		t.Fatalf("denial chat = %q, want original DM", sent[0].ChatID)
	}
	if sent[0].Text != UnauthorizedDMDenialText {
		t.Fatalf("denial text = %q, want %q", sent[0].Text, UnauthorizedDMDenialText)
	}
	unauthorizeddmtest.AssertNoAuthorizedSessionLeak(t, sent[0].Text)
	unauthorizeddmtest.AssertDegradedEvidence(t, store, PairingDegradedAllowlistDenied, "telegram", "stranger")
}

func TestUnauthorizedDM_PairModeSendsOneBoundedPromptAndRecordsPending(t *testing.T) {
	store := newUnauthorizedDMTestStore(t)
	ch := newFakeChannel("telegram")
	ev := InboundEvent{
		Platform: "telegram",
		ChatID:   "424242",
		ChatName: "Private Chat",
		ChatType: "private",
		Kind:     EventSubmit,
		Text:     "hello",
	}

	decision, err := HandleUnauthorizedDM(context.Background(), ch, ev, UnauthorizedDMPolicy{
		Behavior:     UnauthorizedDMPair,
		PairingStore: store,
	})
	if err != nil {
		t.Fatalf("HandleUnauthorizedDM(first): %v", err)
	}
	if !decision.Handled || decision.StartAgent || !decision.ReplySent || decision.PairingStatus != PairingCodeIssued {
		t.Fatalf("decision = %#v, want issued pairing prompt without agent start", decision)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent = %#v, want one pairing prompt", sent)
	}
	if len(sent[0].Text) > 240 {
		t.Fatalf("pairing prompt length = %d, want bounded <= 240: %q", len(sent[0].Text), sent[0].Text)
	}
	unauthorizeddmtest.AssertNoAuthorizedSessionLeak(t, sent[0].Text)

	status, err := store.ReadPairingStatus(context.Background())
	if err != nil {
		t.Fatalf("ReadPairingStatus: %v", err)
	}
	if len(status.Pending) != 1 {
		t.Fatalf("pending = %+v, want one pending pairing", status.Pending)
	}
	pending := status.Pending[0]
	if pending.Platform != "telegram" || pending.UserID != "424242" || pending.UserName != "Private Chat" {
		t.Fatalf("pending = %+v, want telegram private-chat fallback identity", pending)
	}
	unauthorizeddmtest.AssertPairingCode(t, pending.Code)
	if !strings.Contains(sent[0].Text, pending.Code) {
		t.Fatalf("pairing prompt = %q, want code %q", sent[0].Text, pending.Code)
	}
	if !strings.Contains(sent[0].Text, "gormes pairing approve telegram "+pending.Code) {
		t.Fatalf("pairing prompt = %q, want operator approval command", sent[0].Text)
	}

	second, err := HandleUnauthorizedDM(context.Background(), ch, ev, UnauthorizedDMPolicy{
		Behavior:     UnauthorizedDMPair,
		PairingStore: store,
	})
	if err != nil {
		t.Fatalf("HandleUnauthorizedDM(second): %v", err)
	}
	if !second.Handled || second.StartAgent || second.ReplySent || second.PairingStatus != PairingCodeRateLimited {
		t.Fatalf("second decision = %#v, want silent rate-limited handling without agent start", second)
	}
	if got := len(ch.sentSnapshot()); got != 1 {
		t.Fatalf("send count after rate-limited repeat = %d, want still one prompt", got)
	}
}

func TestUnauthorizedDM_IgnoreModeStaysSilentAndDoesNotStartAgent(t *testing.T) {
	store := newUnauthorizedDMTestStore(t)
	ch := newFakeChannel("telegram")

	decision, err := HandleUnauthorizedDM(context.Background(), ch, InboundEvent{
		Platform: "telegram",
		ChatID:   "unauthorized-dm",
		ChatType: "private",
		UserID:   "stranger",
		Kind:     EventSubmit,
		Text:     "hello",
	}, UnauthorizedDMPolicy{
		Behavior:     UnauthorizedDMIgnore,
		PairingStore: store,
	})
	if err != nil {
		t.Fatalf("HandleUnauthorizedDM: %v", err)
	}
	if !decision.Handled || decision.StartAgent || decision.ReplySent {
		t.Fatalf("decision = %#v, want silent handled drop without agent start", decision)
	}
	if sent := ch.sentSnapshot(); len(sent) != 0 {
		t.Fatalf("sent = %#v, want no platform reply", sent)
	}
	unauthorizeddmtest.AssertPairingFileNotCreated(t, store)
}

func TestUnauthorizedDM_GroupOrChannelMessagesStaySilent(t *testing.T) {
	store := newUnauthorizedDMTestStore(t)
	ch := newFakeChannel("telegram")

	for _, chatType := range []string{"group", "channel", "forum"} {
		t.Run(chatType, func(t *testing.T) {
			decision, err := HandleUnauthorizedDM(context.Background(), ch, InboundEvent{
				Platform: "telegram",
				ChatID:   "-100",
				ChatType: chatType,
				UserID:   "stranger",
				Kind:     EventSubmit,
				Text:     "hello",
			}, UnauthorizedDMPolicy{
				Behavior:     UnauthorizedDMPair,
				PairingStore: store,
			})
			if err != nil {
				t.Fatalf("HandleUnauthorizedDM: %v", err)
			}
			if !decision.Handled || decision.StartAgent || decision.ReplySent {
				t.Fatalf("decision = %#v, want silent unauthorized shared-chat drop", decision)
			}
		})
	}
	if sent := ch.sentSnapshot(); len(sent) != 0 {
		t.Fatalf("sent = %#v, want no group/channel replies", sent)
	}
	unauthorizeddmtest.AssertPairingFileNotCreated(t, store)
}

func newUnauthorizedDMTestStore(t *testing.T) *PairingStore {
	t.Helper()
	return unauthorizeddmtest.NewStore(t)
}
