package unauthorizeddm

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/pairing"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/unauthorizeddmtest"
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
	unauthorizeddmtest.AssertNoAuthorizedSessionLeak(t, sent[0].text)
	unauthorizeddmtest.AssertDegradedEvidence(t, store, pairing.PairingDegradedAllowlistDenied, "telegram", "stranger")
}

func TestHandle_PairModeFallsBackToEventUserIDWhenPairingUserIDMissing(t *testing.T) {
	store := newTestStore(t)
	var sent []sentReply

	decision, err := Handle(context.Background(), Event{
		Platform:      "telegram",
		ChatID:        "424242",
		ChatName:      "Private Chat",
		UserID:        "telegram-user-42",
		UserName:      "Mallory",
		DirectMessage: true,
	}, Policy{
		Behavior:            BehaviorPair,
		GeneratePairingCode: store.GeneratePairingCode,
		Send:                captureSend(&sent),
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !decision.ReplySent || decision.PairingStatus != pairing.PairingCodeIssued {
		t.Fatalf("decision = %#v, want pairing code issued using event UserID fallback", decision)
	}

	status, err := store.ReadPairingStatus(context.Background())
	if err != nil {
		t.Fatalf("ReadPairingStatus: %v", err)
	}
	if len(status.Pending) != 1 {
		t.Fatalf("pending = %+v, want one pending pairing", status.Pending)
	}
	if status.Pending[0].UserID != "telegram-user-42" || status.Pending[0].UserName != "Mallory" {
		t.Fatalf("pending identity = %+v, want fallback event user identity", status.Pending[0])
	}
	if len(sent) != 1 || !strings.Contains(sent[0].text, status.Pending[0].Code) {
		t.Fatalf("sent = %+v, want pairing prompt with issued code", sent)
	}
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
	unauthorizeddmtest.AssertNoAuthorizedSessionLeak(t, sent[0].text)

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

func TestFormatPairingPromptSanitizesOperatorCommandFields(t *testing.T) {
	got := FormatPairingPrompt(" telegram\nrm -rf /` ", " ABCD1234\nBAD ")
	for _, forbidden := range []string{"\nrm -rf", "`telegram", "ABCD1234\nBAD"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatPairingPrompt leaked unsafe field %q in:\n%s", forbidden, got)
		}
	}
	if !strings.Contains(got, "Pairing code: `ABCD1234 BAD`") {
		t.Fatalf("prompt missing sanitized code in:\n%s", got)
	}
	if !strings.Contains(got, "`gormes pairing approve telegram rm -rf /' ABCD1234 BAD`") {
		t.Fatalf("prompt missing sanitized operator command in:\n%s", got)
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
	unauthorizeddmtest.AssertPairingFileNotCreated(t, store)
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
	unauthorizeddmtest.AssertPairingFileNotCreated(t, store)
}

func captureSend(sent *[]sentReply) SendFunc {
	return func(_ context.Context, chatID, text string) error {
		*sent = append(*sent, sentReply{chatID: chatID, text: text})
		return nil
	}
}

func newTestStore(t *testing.T) *pairing.PairingStore {
	t.Helper()
	return unauthorizeddmtest.NewStore(t)
}
