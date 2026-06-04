package gormescli

import (
	"context"
	"io"
	"os"

	apptuipick "github.com/TrebuchetDynamics/gormes-agent/internal/app/tuipick"
	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

type TUIPickChoice = apptuipick.Choice

func RunTUIPick(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []TUIPickChoice, defaultID string) (string, error) {
	return apptuipick.RunPick(ctx, stdin, out, prompt, choices, defaultID)
}

func RunTUIPickWithOptions(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []TUIPickChoice, defaultID string, extraOptions ...setupwizard.StepOption) (string, error) {
	return apptuipick.RunPickWithOptions(ctx, stdin, out, prompt, choices, defaultID, extraOptions...)
}

func RunTUIChecklist(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []TUIPickChoice, selectedIDs []string) ([]string, error) {
	return apptuipick.RunChecklist(ctx, stdin, out, prompt, choices, selectedIDs)
}

func TUIPickShouldFallback(err error) bool { return apptuipick.ShouldFallback(err) }
