package main

import (
	"context"
	"io"
	"os"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

type tuiPickChoice = gormescli.TUIPickChoice

func runBubbleTeaPick(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []tuiPickChoice, defaultID string) (string, error) {
	return gormescli.RunTUIPick(ctx, stdin, out, prompt, choices, defaultID)
}

func runBubbleTeaPickWithOptions(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []tuiPickChoice, defaultID string, extraOptions ...setupwizard.StepOption) (string, error) {
	return gormescli.RunTUIPickWithOptions(ctx, stdin, out, prompt, choices, defaultID, extraOptions...)
}

func runBubbleTeaChecklist(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []tuiPickChoice, selectedIDs []string) ([]string, error) {
	return gormescli.RunTUIChecklist(ctx, stdin, out, prompt, choices, selectedIDs)
}

func bubbleTeaPickShouldFallback(err error) bool {
	return gormescli.TUIPickShouldFallback(err)
}
