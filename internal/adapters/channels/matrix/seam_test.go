package matrix

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/threadtext"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestMatrixSeamNormalizeInboundThreadRoot(t *testing.T) {
	s := NewSeam(threadtext.ReplyModeThread, MentionAndFreeRoomInputs{Kind: RoomKindFree}, nil)
	msg := threadtext.InboundMessage{
		ChatID:       "!room:matrix.org",
		UserID:       "@alice:matrix.org",
		UserName:     "Alice",
		MessageID:    "$event-42",
		Text:         "hello world",
		ThreadID:     "thread-reply",
		ThreadRootID: "$thread-root",
	}
	ev, ok := s.NormalizeInbound(msg)
	if !ok {
		t.Fatal("NormalizeInbound() ok = false, want true")
	}
	if ev.Platform != "matrix" {
		t.Fatalf("Platform = %q, want %q", ev.Platform, "matrix")
	}
	if ev.ChatID != "!room:matrix.org" {
		t.Fatalf("ChatID = %q, want %q", ev.ChatID, "!room:matrix.org")
	}
	if ev.UserID != "@alice:matrix.org" {
		t.Fatalf("UserID = %q, want %q", ev.UserID, "@alice:matrix.org")
	}
	if ev.UserName != "Alice" {
		t.Fatalf("UserName = %q, want %q", ev.UserName, "Alice")
	}
	if ev.MsgID != "$event-42" {
		t.Fatalf("MsgID = %q, want %q", ev.MsgID, "$event-42")
	}
	if ev.ThreadID != "$thread-root" {
		t.Fatalf("ThreadID = %q, want %q (canonical thread root)", ev.ThreadID, "$thread-root")
	}
	if ev.Text != "hello world" {
		t.Fatalf("Text = %q, want %q", ev.Text, "hello world")
	}
	if ev.Kind != gateway.EventSubmit {
		t.Fatalf("Kind = %v, want %v", ev.Kind, gateway.EventSubmit)
	}
}

func TestMatrixSeamResolveReplyTargetModes(t *testing.T) {
	msg := threadtext.InboundMessage{
		ChatID:    "!room:matrix.org",
		MessageID: "$event-42",
	}
	t.Run("flat mode omits thread metadata", func(t *testing.T) {
		s := NewSeam(threadtext.ReplyModeFlat, MentionAndFreeRoomInputs{Kind: RoomKindFree}, nil)
		got, ok := s.ResolveReplyTarget(msg)
		if !ok {
			t.Fatal("ResolveReplyTarget() ok = false, want true")
		}
		if got.ThreadID != "" {
			t.Fatalf("ThreadID = %q, want empty (flat mode omits thread)", got.ThreadID)
		}
	})
	t.Run("thread mode starts from root messages", func(t *testing.T) {
		s := NewSeam(threadtext.ReplyModeThread, MentionAndFreeRoomInputs{Kind: RoomKindFree}, nil)
		got, ok := s.ResolveReplyTarget(msg)
		if !ok {
			t.Fatal("ResolveReplyTarget() ok = false, want true")
		}
		if got.ThreadID != "$event-42" {
			t.Fatalf("ThreadID = %q, want %q (root message starts thread)", got.ThreadID, "$event-42")
		}
		if got.ReplyToMessageID != "$event-42" {
			t.Fatalf("ReplyToMessageID = %q, want %q", got.ReplyToMessageID, "$event-42")
		}
	})
	t.Run("existing thread preserved regardless of mode", func(t *testing.T) {
		s := NewSeam(threadtext.ReplyModeFlat, MentionAndFreeRoomInputs{Kind: RoomKindFree}, nil)
		msgWithThread := threadtext.InboundMessage{
			ChatID:       "!room:matrix.org",
			MessageID:    "$event-43",
			ThreadID:     "reply",
			ThreadRootID: "$thread-root",
		}
		got, ok := s.ResolveReplyTarget(msgWithThread)
		if !ok {
			t.Fatal("ResolveReplyTarget() ok = false, want true")
		}
		if got.ThreadID != "$thread-root" {
			t.Fatalf("ThreadID = %q, want %q", got.ThreadID, "$thread-root")
		}
	})
}

func TestMatrixSeamMentionAndFreeRoomInputs(t *testing.T) {
	t.Run("preserves DM room kind", func(t *testing.T) {
		inputs := MentionAndFreeRoomInputs{Kind: RoomKindDM, MentionedInMsg: true}
		s := NewSeam(threadtext.ReplyModeFlat, inputs, nil)
		got := s.RoomInputs()
		if got.Kind != RoomKindDM {
			t.Fatalf("Kind = %q, want %q", got.Kind, RoomKindDM)
		}
		if !got.MentionedInMsg {
			t.Fatal("MentionedInMsg = false, want true")
		}
	})
	t.Run("preserves free-room kind", func(t *testing.T) {
		inputs := MentionAndFreeRoomInputs{Kind: RoomKindFree, MentionedInMsg: false}
		s := NewSeam(threadtext.ReplyModeThread, inputs, nil)
		got := s.RoomInputs()
		if got.Kind != RoomKindFree {
			t.Fatalf("Kind = %q, want %q", got.Kind, RoomKindFree)
		}
		if got.MentionedInMsg {
			t.Fatal("MentionedInMsg = true, want false")
		}
	})
	t.Run("preserves require-mention kind", func(t *testing.T) {
		inputs := MentionAndFreeRoomInputs{Kind: RoomKindRequireMention, MentionedInMsg: false}
		s := NewSeam(threadtext.ReplyModeFlat, inputs, nil)
		got := s.RoomInputs()
		if got.Kind != RoomKindRequireMention {
			t.Fatalf("Kind = %q, want %q", got.Kind, RoomKindRequireMention)
		}
	})
	t.Run("seam does not inspect env at send time", func(t *testing.T) {
		inputs := MentionAndFreeRoomInputs{Kind: RoomKindDM}
		s := NewSeam(threadtext.ReplyModeFlat, inputs, nil)
		msg := threadtext.InboundMessage{
			ChatID:    "!room:matrix.org",
			UserID:    "@bot:matrix.org",
			Text:      "send",
			MessageID: "$event-1",
		}
		_, ok := s.NormalizeInbound(msg)
		if !ok {
			t.Fatal("NormalizeInbound() ok = false, want true — room kind is pure input")
		}
		got := s.RoomInputs()
		if got.Kind != RoomKindDM {
			t.Fatalf("RoomInputs().Kind changed from %q to %q after NormalizeInbound", RoomKindDM, got.Kind)
		}
	})
}

func TestMatrixSeamProcessingHooks(t *testing.T) {
	t.Run("hooks ordered: start then complete", func(t *testing.T) {
		var order []string
		hooks := &ProcessingHooks{
			OnStart: func(_ threadtext.InboundMessage) {
				order = append(order, "start")
			},
			OnComplete: func(_ threadtext.InboundMessage) {
				order = append(order, "complete")
			},
		}
		s := NewSeam(threadtext.ReplyModeFlat, MentionAndFreeRoomInputs{Kind: RoomKindFree}, hooks)
		msg := threadtext.InboundMessage{
			ChatID:    "!room:matrix.org",
			UserID:    "@alice:matrix.org",
			Text:      "ok",
			MessageID: "$event-1",
		}
		_, ok := s.NormalizeInbound(msg)
		if !ok {
			t.Fatal("NormalizeInbound() ok = false, want true")
		}
		if len(order) != 2 {
			t.Fatalf("got %d hook calls, want 2", len(order))
		}
		if order[0] != "start" || order[1] != "complete" {
			t.Fatalf("hook order = %v, want [start complete]", order)
		}
	})
	t.Run("start then failure when normalize fails", func(t *testing.T) {
		var order []string
		hooks := &ProcessingHooks{
			OnStart: func(_ threadtext.InboundMessage) {
				order = append(order, "start")
			},
			OnFailure: func(_ threadtext.InboundMessage) {
				order = append(order, "failure")
			},
		}
		s := NewSeam(threadtext.ReplyModeFlat, MentionAndFreeRoomInputs{Kind: RoomKindFree}, hooks)
		msg := threadtext.InboundMessage{
			ChatID: "!room:matrix.org",
			// Missing required UserID, Text — should trigger failure
			MessageID: "$event-1",
		}
		_, ok := s.NormalizeInbound(msg)
		if ok {
			t.Fatal("NormalizeInbound() ok = true, want false (missing required fields)")
		}
		if len(order) != 2 {
			t.Fatalf("got %d hook calls, want 2", len(order))
		}
		if order[0] != "start" || order[1] != "failure" {
			t.Fatalf("hook order = %v, want [start failure]", order)
		}
	})
	t.Run("cancel suppresses terminal hooks and fires OnCancel", func(t *testing.T) {
		var order []string
		hooks := &ProcessingHooks{
			OnStart: func(_ threadtext.InboundMessage) {
				order = append(order, "start")
			},
			OnComplete: func(_ threadtext.InboundMessage) {
				order = append(order, "complete")
			},
			OnCancel: func(_ threadtext.InboundMessage) {
				order = append(order, "cancel")
			},
		}
		s := NewSeam(threadtext.ReplyModeFlat, MentionAndFreeRoomInputs{Kind: RoomKindFree}, hooks)
		msg := threadtext.InboundMessage{
			ChatID:    "!room:matrix.org",
			UserID:    "@alice:matrix.org",
			Text:      "ok",
			MessageID: "$event-1",
		}
		s.Cancel(msg)
		_, ok := s.NormalizeInbound(msg)
		if !ok {
			t.Fatal("NormalizeInbound() ok = false, want true (cancel does not block normalization)")
		}
		if len(order) != 2 {
			t.Fatalf("got %d hook calls, want 2", len(order))
		}
		if order[0] != "cancel" || order[1] != "start" {
			t.Fatalf("hook order = %v, want [cancel start] — complete suppressed after cancel", order)
		}
	})
	t.Run("nil hooks are valid no-ops", func(t *testing.T) {
		s := NewSeam(threadtext.ReplyModeFlat, MentionAndFreeRoomInputs{Kind: RoomKindFree}, nil)
		msg := threadtext.InboundMessage{
			ChatID:    "!room:matrix.org",
			UserID:    "@alice:matrix.org",
			Text:      "ok",
			MessageID: "$event-1",
		}
		_, ok := s.NormalizeInbound(msg)
		if !ok {
			t.Fatal("NormalizeInbound() ok = false, want true (nil hooks are no-ops)")
		}
		s.Cancel(msg)
		if !s.Cancelled() {
			t.Fatal("Cancelled() = false, want true")
		}
	})
}

func TestMatrixSeam_ReplyToField(t *testing.T) {
	// The InboundMessage struct has no separate ReplyTo field in the
	// threadtext contract — reply-to data is conveyed through ThreadID
	// and ThreadRootID. This test proves the seam does not invent a
	// ReplyTo field that the threadtext contract does not support.
	msg := threadtext.InboundMessage{
		ChatID:       "!room:matrix.org",
		UserID:       "@alice:matrix.org",
		Text:         "reply",
		MessageID:    "$event-42",
		ThreadID:     "reply-to-41",
		ThreadRootID: "$thread-root",
	}
	s := NewSeam(threadtext.ReplyModeThread, MentionAndFreeRoomInputs{Kind: RoomKindFree}, nil)
	target, ok := s.ResolveReplyTarget(msg)
	if !ok {
		t.Fatal("ResolveReplyTarget() ok = false, want true")
	}
	if target.ThreadID != "$thread-root" {
		t.Fatalf("ThreadID = %q, want %q", target.ThreadID, "$thread-root")
	}
	if target.ReplyToMessageID != "$event-42" {
		t.Fatalf("ReplyToMessageID = %q, want %q", target.ReplyToMessageID, "$event-42")
	}
}
