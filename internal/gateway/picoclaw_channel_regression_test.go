package gateway

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestPicoClawChannelRegression_SenderIdentityAndAllowlist(t *testing.T) {
	matrix := newFakeChannel("matrix")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"matrix": "!ops:example.org"},
		AllowedUsers: map[string]map[string]bool{
			"matrix": {"@alice:example.org": true},
		},
	}, fk, slog.Default())
	if err := m.Register(matrix); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	matrix.pushInbound(InboundEvent{
		Platform: "matrix",
		ChatID:   "!ops:example.org",
		ChatName: "Ops",
		UserID:   "@alice:example.org",
		UserName: "Alice",
		ThreadID: "$mx-thread-root",
		MsgID:    "$mx-message-1",
		Kind:     EventSubmit,
		Text:     "inspect channel state",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})
	got := fk.submitsSnapshot()[0]
	for _, want := range []string{
		"**Source:** matrix chat `!ops:example.org`",
		"**User ID:** `@alice:example.org`",
		"**Thread ID:** `$mx-thread-root`",
		"**Message ID:** `$mx-message-1`",
	} {
		if !strings.Contains(got.SessionContext, want) {
			t.Fatalf("SessionContext missing %q in:\n%s", want, got.SessionContext)
		}
	}

	matrix.pushInbound(InboundEvent{
		Platform: "matrix",
		ChatID:   "!ops:example.org",
		UserID:   "@mallory:example.org",
		MsgID:    "$mx-message-2",
		Kind:     EventSubmit,
		Text:     "exfiltrate logs",
	})
	time.Sleep(50 * time.Millisecond)
	if submits := fk.submitsSnapshot(); len(submits) != 1 {
		t.Fatalf("disallowed Matrix sender submitted to provider; submits=%#v", submits)
	}
}

func TestPicoClawChannelRegression_RichMediaEnvelopePersists(t *testing.T) {
	ch := newFakeChannel("dingtalk")
	frames := make(chan kernel.RenderFrame, 2)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"dingtalk": "dm-media"},
	}, fk, slog.Default())
	m.setRenderChan(frames)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{
		Platform:  "dingtalk",
		ChatID:    "dm-media",
		UserID:    "staff-1",
		MsgID:     "media-msg-1",
		MessageID: "media-msg-1",
		Kind:      EventSubmit,
		Text:      "summarize the attached channel evidence",
		Attachments: []Attachment{
			{
				Kind:      "image",
				URL:       "https://media.example.test/channel/screenshot.png",
				MediaType: "image/png",
				FileName:  "screenshot.png",
				SourceID:  "img-123",
				SizeBytes: 12345,
			},
			{
				Kind:      "document",
				URL:       "https://media.example.test/channel/report.pdf",
				MediaType: "application/pdf",
				FileName:  "report.pdf",
				SourceID:  "doc-456",
				SizeBytes: 67890,
			},
			{
				Kind:      "voice_transcript",
				URL:       "transcript:Please review the PDF before answering.",
				MediaType: "text/plain",
				FileName:  "voice-note.txt",
				SourceID:  "voice-789",
			},
		},
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})
	got := fk.submitsSnapshot()[0].Text
	for _, want := range []string{
		"Attachments:",
		"- image screenshot.png: https://media.example.test/channel/screenshot.png (mediaType=image/png, sourceId=img-123, sizeBytes=12345)",
		"- document report.pdf: https://media.example.test/channel/report.pdf (mediaType=application/pdf, sourceId=doc-456, sizeBytes=67890)",
		"- voice_transcript voice-note.txt: transcript:Please review the PDF before answering. (mediaType=text/plain, sourceId=voice-789)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("submitted media envelope missing %q in:\n%s", want, got)
		}
	}

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []hermes.Message{
			{Role: "user", Content: got},
			{Role: "assistant", Content: "The image, PDF, and voice transcript metadata were preserved."},
		},
	}
	waitFor(t, 500*time.Millisecond, func() bool {
		sent := ch.sentSnapshot()
		return len(sent) >= 1 && strings.Contains(sent[len(sent)-1].Text, "voice transcript metadata")
	})
}

func TestPicoClawChannelRegression_FinalDeliveryDoesNotEditToolPlaceholder(t *testing.T) {
	ch := &freshFinalFakeChannel{fakeChannel: newFakeChannel("telegram")}
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}
	var nowMu sync.Mutex
	now := time.Date(2026, 5, 15, 20, 30, 0, 0, time.UTC)

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:    map[string]string{"telegram": "42"},
		CoalesceMs:      10,
		FreshFinalAfter: time.Minute,
		Now: func() time.Time {
			nowMu.Lock()
			defer nowMu.Unlock()
			return now
		},
	}, fk, slog.Default())
	m.setRenderChan(frames)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "operator-1",
		MsgID:    "origin-msg",
		Kind:     EventSubmit,
		Text:     "apply the queued steer and finish cleanly",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		DraftText: "Using the operator steering before the final answer.",
	}
	waitFor(t, 500*time.Millisecond, func() bool {
		sent := ch.sentSnapshot()
		return len(sent) >= 1 && strings.Contains(sent[0].Text, "operator steering")
	})
	placeholderID := ch.sentSnapshot()[0].MsgID

	nowMu.Lock()
	now = now.Add(time.Minute)
	nowMu.Unlock()
	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []hermes.Message{
			{Role: "user", Content: "apply the queued steer and finish cleanly"},
			{Role: "assistant", Content: "Final answer after steering."},
		},
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		sent := ch.sentSnapshot()
		return len(sent) >= 2 && strings.Contains(sent[len(sent)-1].Text, "Final answer after steering")
	})
	if edits := ch.editsSnapshot(); len(edits) != 0 {
		t.Fatalf("final delivery edited an existing tool/stream placeholder; edits=%#v", edits)
	}
	if deletes := ch.deletesSnapshot(); len(deletes) != 1 || deletes[0].MsgID != placeholderID {
		t.Fatalf("fresh final did not delete the old placeholder; deletes=%#v placeholder=%q", deletes, placeholderID)
	}
}

func TestPicoClawChannelRegression_ToolProgressNotificationsAreComplete(t *testing.T) {
	ch := newFakeChannel("feishu")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"feishu": "oc_media"},
		CoalesceMs:   10,
	}, fk, slog.Default())
	m.setRenderChan(frames)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{
		Platform:  "feishu",
		ChatID:    "oc_media",
		UserID:    "ou_sender",
		MsgID:     "feishu-msg-1",
		MessageID: "feishu-msg-1",
		Kind:      EventSubmit,
		Text:      "run the multi-tool inspection",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseStreaming,
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: browser_navigate: https://example.test/status"},
			{At: time.Now(), Text: "tool: terminal: curl -fsS https://example.test/status.json"},
		},
	}
	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []hermes.Message{
			{Role: "user", Content: "run the multi-tool inspection"},
			{Role: "assistant", Content: "Inspection complete."},
		},
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		sent := ch.sentSnapshot()
		return len(sent) >= 2 &&
			strings.Contains(sent[0].Text, "browser") &&
			strings.Contains(sent[0].Text, "ACTION [network] Fetching remote content") &&
			sent[len(sent)-1].Text == "Inspection complete."
	})
	sent := ch.sentSnapshot()
	if strings.Contains(sent[len(sent)-1].Text, "browser_navigate") || strings.Contains(sent[len(sent)-1].Text, "terminal") ||
		strings.Contains(sent[len(sent)-1].Text, "Fetching remote content") {
		t.Fatalf("final answer included notification-center tool evidence; sent=%#v", sent)
	}
}
