package gateway

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// recordingSubmitter records every kernel submit so the test driver can prove
// both fixtures dispatched through the same native runtime path.
type recordingSubmitter struct {
	mu      sync.Mutex
	submits []kernel.PlatformEvent
	err     error
}

func (r *recordingSubmitter) Submit(ev kernel.PlatformEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.submits = append(r.submits, ev)
	return nil
}

func (r *recordingSubmitter) ResetSession() error               { return nil }
func (r *recordingSubmitter) Render() <-chan kernel.RenderFrame { return nil }

func (r *recordingSubmitter) snapshot() []kernel.PlatformEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneSlice(r.submits)
}

// turnAdapterFixtureChannel is a minimal Channel used by both Telegram and
// non-Telegram fixtures. The shared turn adapter must treat both identically;
// only Name() differs.
type turnAdapterFixtureChannel struct {
	name string

	mu        sync.Mutex
	sent      []fakeSent
	nextMsgID int
}

func newTurnAdapterFixtureChannel(name string) *turnAdapterFixtureChannel {
	return &turnAdapterFixtureChannel{name: name, nextMsgID: 9000}
}

func (c *turnAdapterFixtureChannel) Name() string { return c.name }

func (c *turnAdapterFixtureChannel) Run(ctx context.Context, _ chan<- InboundEvent) error {
	<-ctx.Done()
	return nil
}

func (c *turnAdapterFixtureChannel) Send(_ context.Context, chatID, text string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := strconv.Itoa(c.nextMsgID)
	c.nextMsgID++
	c.sent = append(c.sent, fakeSent{ChatID: chatID, Text: text, MsgID: id})
	return id, nil
}

func (c *turnAdapterFixtureChannel) sentSnapshot() []fakeSent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return cloneSlice(c.sent)
}

// channelNeutralCapture pins the channel-neutral fields the adapter exposes to
// hooks/state callbacks so the driver can prove both fixtures produced the
// same shape.
type channelNeutralCapture struct {
	channelName       string
	source            SessionSource
	sessionKey        string
	resolvedSessionID string
	submitText        string
	commandKind       EventKind
	replyChatID       string
	replyMsgID        string
	attachments       []Attachment
	turnCleared       bool
}

// runChannelNeutralFixture is the single test driver. It accepts a fixture
// description, builds a TurnRequest from a channel-neutral InboundEvent, and
// dispatches it through the shared TurnAdapter. It returns the captured
// fields so the test can compare both fixtures byte-for-byte.
func runChannelNeutralFixture(t *testing.T, channel Channel, ev InboundEvent, runtimeErr error) (channelNeutralCapture, []fakeSent, []kernel.PlatformEvent) {
	t.Helper()

	sub := &recordingSubmitter{err: runtimeErr}

	var (
		captureMu sync.Mutex
		capture   channelNeutralCapture
	)

	adapter := &TurnAdapter{
		Submitter: sub,
		OnTurnStart: func(req TurnRequest) {
			captureMu.Lock()
			defer captureMu.Unlock()
			capture.channelName = req.Channel.Name()
			capture.source = req.Source
			capture.sessionKey = req.SessionKey
			capture.resolvedSessionID = req.ResolvedSessionID
			capture.submitText = req.SubmitText
			capture.commandKind = req.CommandKind
			capture.replyChatID = req.ReplyChatID
			capture.replyMsgID = req.ReplyMsgID
			capture.attachments = append([]Attachment(nil), req.Attachments...)
		},
		OnTurnFailure: func(_ TurnRequest, _ error) {
			captureMu.Lock()
			defer captureMu.Unlock()
			capture.turnCleared = true
		},
	}

	req := TurnRequest{
		Channel:           channel,
		Source:            sessionSourceFromInbound(ev),
		SessionKey:        ev.ChatKey(),
		ResolvedSessionID: ev.ChatKey(),
		SubmitText:        ev.SubmitText(),
		Attachments:       ev.Attachments,
		CommandKind:       ev.Kind,
		ReplyChatID:       ev.ChatID,
		ReplyMsgID:        ev.MsgID,
	}

	_ = adapter.Dispatch(context.Background(), req)

	captureMu.Lock()
	out := capture
	captureMu.Unlock()

	var sent []fakeSent
	if rec, ok := channel.(interface{ sentSnapshot() []fakeSent }); ok {
		sent = rec.sentSnapshot()
	}

	return out, sent, sub.snapshot()
}

func TestChannelNeutralTurnAdapter_DispatchesBothFixturesThroughSameRuntime(t *testing.T) {
	telegramChannel := newTurnAdapterFixtureChannel("telegram")
	otherChannel := newTurnAdapterFixtureChannel("fakechannel")

	telegramEvent := InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		ChatType: "private",
		UserID:   "tg-user-1",
		MsgID:    "tg-msg-1",
		Kind:     EventSubmit,
		Text:     "hello from telegram",
		Attachments: []Attachment{{
			Kind:      "image",
			URL:       "https://example.invalid/tg.png",
			MediaType: "image",
			SourceID:  "tg-img",
		}},
	}
	otherEvent := InboundEvent{
		Platform: "fakechannel",
		ChatID:   "C-abc",
		ChatType: "dm",
		UserID:   "fc-user-1",
		MsgID:    "fc-msg-1",
		Kind:     EventSubmit,
		Text:     "hello from fakechannel",
		Attachments: []Attachment{{
			Kind:      "image",
			URL:       "https://example.invalid/fc.png",
			MediaType: "image",
			SourceID:  "fc-img",
		}},
	}

	tgCap, _, tgSubmits := runChannelNeutralFixture(t, telegramChannel, telegramEvent, nil)
	fcCap, _, fcSubmits := runChannelNeutralFixture(t, otherChannel, otherEvent, nil)

	// Both fixtures must take the same native runtime path: each produced one
	// kernel.PlatformEventSubmit through the shared adapter.
	if len(tgSubmits) != 1 || tgSubmits[0].Kind != kernel.PlatformEventSubmit {
		t.Fatalf("telegram fixture: want one PlatformEventSubmit, got %+v", tgSubmits)
	}
	if len(fcSubmits) != 1 || fcSubmits[0].Kind != kernel.PlatformEventSubmit {
		t.Fatalf("non-telegram fixture: want one PlatformEventSubmit, got %+v", fcSubmits)
	}

	// Channel-neutral request shape must carry the listed fields verbatim from
	// each channel; the channel name must differ but every other dispatch
	// concept is derived from channel-neutral fields.
	if tgCap.channelName != "telegram" || fcCap.channelName != "fakechannel" {
		t.Fatalf("channel names = (%q, %q), want (telegram, fakechannel)",
			tgCap.channelName, fcCap.channelName)
	}
	for _, c := range []channelNeutralCapture{tgCap, fcCap} {
		if c.commandKind != EventSubmit {
			t.Fatalf("command kind = %v, want %v", c.commandKind, EventSubmit)
		}
		if c.sessionKey == "" || c.resolvedSessionID == "" {
			t.Fatalf("session key/id missing in capture: %+v", c)
		}
		if c.replyChatID == "" {
			t.Fatalf("reply chat id missing in capture: %+v", c)
		}
		if c.replyMsgID == "" {
			t.Fatalf("reply msg id missing in capture: %+v", c)
		}
		if c.source.Platform == "" || c.source.UserID == "" {
			t.Fatalf("session source incomplete: %+v", c.source)
		}
		if len(c.attachments) != 1 {
			t.Fatalf("media references missing: %+v", c.attachments)
		}
		if !strings.Contains(c.submitText, "hello from") {
			t.Fatalf("submit text missing: %q", c.submitText)
		}
	}
}

func TestChannelNeutralTurnAdapter_RuntimeFailureRendersSafeErrorAndClearsTurn(t *testing.T) {
	rawErr := errors.New("internal provider error: API key sk-deadbeef invalid")

	for _, fixture := range []struct {
		name    string
		channel Channel
		event   InboundEvent
	}{
		{
			name:    "telegram",
			channel: newTurnAdapterFixtureChannel("telegram"),
			event: InboundEvent{
				Platform: "telegram", ChatID: "42", UserID: "tg-u", MsgID: "tg-m",
				Kind: EventSubmit, Text: "hello",
			},
		},
		{
			name:    "fakechannel",
			channel: newTurnAdapterFixtureChannel("fakechannel"),
			event: InboundEvent{
				Platform: "fakechannel", ChatID: "C-abc", UserID: "fc-u", MsgID: "fc-m",
				Kind: EventSubmit, Text: "hello",
			},
		},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			cap, sent, submits := runChannelNeutralFixture(t, fixture.channel, fixture.event, rawErr)
			if len(submits) != 0 {
				t.Fatalf("runtime failure must not record a submit, got %+v", submits)
			}
			if !cap.turnCleared {
				t.Fatalf("runtime failure must clear active turn state")
			}
			if len(sent) != 1 {
				t.Fatalf("runtime failure must render exactly one external reply, got %d", len(sent))
			}
			got := sent[0].Text
			// The shared safe-error helper must NOT leak the raw provider error.
			if strings.Contains(got, "sk-deadbeef") || strings.Contains(got, "API key") {
				t.Fatalf("safe-error helper leaked raw provider error: %q", got)
			}
			if strings.Contains(got, "internal provider error") {
				t.Fatalf("safe-error helper leaked raw provider error: %q", got)
			}
			if strings.TrimSpace(got) == "" {
				t.Fatalf("safe-error helper produced empty external reply")
			}
		})
	}
}
