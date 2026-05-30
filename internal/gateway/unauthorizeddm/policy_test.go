package unauthorizeddm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/pairing"
)

type sentReply struct {
	chatID string
	text   string
}

func TestHandle_DenyModeSendsDeterministicDenialAndRecordsEvidence(t *testing.T) {
	store := newTestStore(t)
	var sent []sentReply
	ev := Event{
		Platform:      "telegram",
		ChatID:        "unauthorized-dm",
		UserID:        "stranger",
		UserName:      "Mallory",
		DirectMessage: true,
		PairingUserID: "stranger",
	}

	decision, err := Handle(context.Background(), ev, Policy{
		Behavior:            BehaviorDeny,
		GeneratePairingCode: store.GeneratePairingCode,
		Send:                captureSend(&sent),
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !decision.Handled || decision.StartAgent || !decision.ReplySent {
		t.Fatalf("decision = %#v, want handled denial reply without agent start", decision)
	}
	if len(sent) != 1 || sent[0].chatID != "unauthorized-dm" || sent[0].text != DenialText {
		t.Fatalf("sent = %#v, want one deterministic denial to original DM", sent)
	}
	assertNoAuthorizedSessionLeak(t, sent[0].text)
	assertDegradedEvidence(t, store, pairing.PairingDegradedAllowlistDenied, "telegram", "stranger")
}

func TestHandle_PairModeSendsOneBoundedPromptAndRecordsPending(t *testing.T) {
	store := newTestStore(t)
	var sent []sentReply
	ev := Event{
		Platform:      "telegram",
		ChatID:        "424242",
		ChatName:      "Private Chat",
		DirectMessage: true,
		PairingUserID: "424242",
	}

	decision, err := Handle(context.Background(), ev, Policy{
		Behavior:            BehaviorPair,
		GeneratePairingCode: store.GeneratePairingCode,
		Send:                captureSend(&sent),
	})
	if err != nil {
		t.Fatalf("Handle(first): %v", err)
	}
	if !decision.Handled || decision.StartAgent || !decision.ReplySent || decision.PairingStatus != pairing.PairingCodeIssued {
		t.Fatalf("decision = %#v, want issued pairing prompt without agent start", decision)
	}
	if len(sent) != 1 {
		t.Fatalf("sent = %#v, want one pairing prompt", sent)
	}
	if len(sent[0].text) > 240 {
		t.Fatalf("pairing prompt length = %d, want bounded <= 240: %q", len(sent[0].text), sent[0].text)
	}
	assertNoAuthorizedSessionLeak(t, sent[0].text)

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
	assertPairingCode(t, pending.Code)
	if !strings.Contains(sent[0].text, pending.Code) || !strings.Contains(sent[0].text, "gormes pairing approve telegram "+pending.Code) {
		t.Fatalf("pairing prompt = %q, want code and operator approval command", sent[0].text)
	}

	second, err := Handle(context.Background(), ev, Policy{
		Behavior:            BehaviorPair,
		GeneratePairingCode: store.GeneratePairingCode,
		Send:                captureSend(&sent),
	})
	if err != nil {
		t.Fatalf("Handle(second): %v", err)
	}
	if !second.Handled || second.StartAgent || second.ReplySent || second.PairingStatus != pairing.PairingCodeRateLimited {
		t.Fatalf("second decision = %#v, want silent rate-limited handling without agent start", second)
	}
	if got := len(sent); got != 1 {
		t.Fatalf("send count after rate-limited repeat = %d, want still one prompt", got)
	}
}

func TestHandle_IgnoreModeStaysSilentAndDoesNotStartAgent(t *testing.T) {
	store := newTestStore(t)
	var sent []sentReply

	decision, err := Handle(context.Background(), Event{
		Platform:      "telegram",
		ChatID:        "unauthorized-dm",
		DirectMessage: true,
		PairingUserID: "stranger",
	}, Policy{
		Behavior:            BehaviorIgnore,
		GeneratePairingCode: store.GeneratePairingCode,
		Send:                captureSend(&sent),
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !decision.Handled || decision.StartAgent || decision.ReplySent {
		t.Fatalf("decision = %#v, want silent handled drop without agent start", decision)
	}
	if len(sent) != 0 {
		t.Fatalf("sent = %#v, want no platform reply", sent)
	}
	assertPairingFileNotCreated(t, store)
}

func TestHandle_GroupOrChannelMessagesStaySilent(t *testing.T) {
	store := newTestStore(t)
	var sent []sentReply

	for _, name := range []string{"group", "channel", "forum"} {
		t.Run(name, func(t *testing.T) {
			decision, err := Handle(context.Background(), Event{
				Platform:      "telegram",
				ChatID:        "-100",
				DirectMessage: false,
				PairingUserID: "stranger",
			}, Policy{
				Behavior:            BehaviorPair,
				GeneratePairingCode: store.GeneratePairingCode,
				Send:                captureSend(&sent),
			})
			if err != nil {
				t.Fatalf("Handle: %v", err)
			}
			if !decision.Handled || decision.StartAgent || decision.ReplySent {
				t.Fatalf("decision = %#v, want silent unauthorized shared-chat drop", decision)
			}
		})
	}
	if len(sent) != 0 {
		t.Fatalf("sent = %#v, want no group/channel replies", sent)
	}
	assertPairingFileNotCreated(t, store)
}

func captureSend(sent *[]sentReply) SendFunc {
	return func(_ context.Context, chatID, text string) error {
		*sent = append(*sent, sentReply{chatID: chatID, text: text})
		return nil
	}
}

var testStorePaths = map[*pairing.PairingStore]string{}

func newTestStore(t *testing.T) *pairing.PairingStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pairing.json")
	store := pairing.NewPairingStore(path)
	testStorePaths[store] = path
	return store
}

func assertPairingCode(t *testing.T, code string) {
	t.Helper()
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	if len(code) != 8 {
		t.Fatalf("len(%q) = %d, want %d", code, len(code), 8)
	}
	for _, c := range code {
		if !strings.ContainsRune(alphabet, c) {
			t.Fatalf("code %q contains %q outside Hermes alphabet %q", code, c, alphabet)
		}
	}
}

func assertNoAuthorizedSessionLeak(t *testing.T, text string) {
	t.Helper()
	for _, leak := range []string{"allowed", "authorized", "session", "allowed-chat-42"} {
		if strings.Contains(strings.ToLower(text), leak) {
			t.Fatalf("response %q leaks authorized-session state marker %q", text, leak)
		}
	}
}

func assertDegradedEvidence(t *testing.T, store *pairing.PairingStore, reason pairing.PairingDegradedReason, platform, userID string) {
	t.Helper()
	status, err := store.ReadPairingStatus(context.Background())
	if err != nil {
		t.Fatalf("ReadPairingStatus: %v", err)
	}
	if len(status.Pending) != 0 || len(status.Approved) != 0 {
		t.Fatalf("status = %+v, want denied user evidence without pending or approved records", status)
	}
	for _, evidence := range status.Degraded {
		if evidence.Reason == reason && evidence.Platform == platform && evidence.UserID == userID {
			return
		}
	}
	t.Fatalf("degraded evidence = %+v, want %s for %s/%s", status.Degraded, reason, platform, userID)
}

func assertPairingFileNotCreated(t *testing.T, store *pairing.PairingStore) {
	t.Helper()
	path := testStorePaths[store]
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("pairing store file err = %v, want not created", err)
	}
}
