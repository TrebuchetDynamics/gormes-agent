package kernel

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestTitleFailureCallback_KernelAuxiliaryWarningFrame(t *testing.T) {
	t.Parallel()

	providerErr := errors.New("openrouter 402: credits exhausted")
	var warning RenderFrame
	foreground := RenderFrame{
		Phase: PhaseIdle,
		History: []llm.Message{
			{Role: "assistant", Content: "foreground answer survived"},
		},
	}

	result := llm.GenerateTitle(context.Background(), llm.TitleRequest{
		History: []llm.TitleMessage{
			{Role: "user", Content: "please name this turn"},
			{Role: "assistant", Content: "foreground answer survived"},
		},
		FailureCallback: func(ctx context.Context, evidence llm.TitleEvidence) error {
			warning = RenderFrame{
				Phase:      PhaseIdle,
				StatusText: string(evidence.Kind),
				LastError:  evidence.Message,
				History:    foreground.History,
			}
			return nil
		},
	}, func(ctx context.Context, req llm.TitleModelRequest) (string, error) {
		return "", providerErr
	})

	if result.Title != "" {
		t.Fatalf("Title = %q; want empty title", result.Title)
	}
	if result.Status != llm.TitleStatusProviderFailed {
		t.Fatalf("Status = %q; want %q", result.Status, llm.TitleStatusProviderFailed)
	}
	if warning.StatusText != string(llm.TitleStatusProviderFailed) {
		t.Fatalf("warning StatusText = %q; want %q", warning.StatusText, llm.TitleStatusProviderFailed)
	}
	if !strings.Contains(warning.LastError, "openrouter 402") {
		t.Fatalf("warning LastError = %q; want provider failure detail", warning.LastError)
	}
	if got := warning.History[len(warning.History)-1].Content; got != "foreground answer survived" {
		t.Fatalf("foreground answer = %q; want preserved answer", got)
	}
}
