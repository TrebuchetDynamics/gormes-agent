package tuipick

import (
	"context"
	"io"
	"os"

	apptuipick "github.com/TrebuchetDynamics/gormes-agent/internal/app/tuipick"
	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

type Choice = apptuipick.Choice

func RunPick(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []Choice, defaultID string) (string, error) {
	return apptuipick.RunPick(ctx, stdin, out, prompt, choices, defaultID)
}

func RunPickWithOptions(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []Choice, defaultID string, extraOptions ...setupwizard.StepOption) (string, error) {
	return apptuipick.RunPickWithOptions(ctx, stdin, out, prompt, choices, defaultID, extraOptions...)
}

func RunChecklist(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []Choice, selectedIDs []string) ([]string, error) {
	return apptuipick.RunChecklist(ctx, stdin, out, prompt, choices, selectedIDs)
}

func ShouldFallback(err error) bool { return apptuipick.ShouldFallback(err) }
