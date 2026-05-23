package kernel

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

func TestKernelResumeSessionSwitchesResidentSessionAndHistory(t *testing.T) {
	k, _ := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = k.Run(ctx) }()
	<-k.Render()

	history := []hermes.Message{
		{Role: "user", Content: "previous question"},
		{Role: "assistant", Content: "previous answer"},
	}
	if err := k.ResumeSession("sess-resume", history); err != nil {
		t.Fatalf("ResumeSession: %v", err)
	}
	history[0].Content = "mutated after resume"

	frame := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID == "sess-resume" && len(f.History) == 2
	}, time.Second)
	if frame.StatusText != "session resumed" {
		t.Fatalf("StatusText = %q, want session resumed", frame.StatusText)
	}
	if frame.History[0].Content != "previous question" || frame.History[1].Content != "previous answer" {
		t.Fatalf("History = %+v, want cloned resumed history", frame.History)
	}
}

func TestKernelResumeSessionRequiresSessionID(t *testing.T) {
	k, _ := fixture(t)
	if err := k.ResumeSession("  ", nil); !errors.Is(err, ErrResumeSessionIDRequired) {
		t.Fatalf("ResumeSession empty error = %v, want %v", err, ErrResumeSessionIDRequired)
	}
}
