package main

import (
	"context"
	"io"
	"os"

	"github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/tuipick"
	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

type tuiPickChoice = tuipick.Choice

func runBubbleTeaPick(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []tuiPickChoice, defaultID string) (string, error) {
	return tuipick.RunPick(ctx, stdin, out, prompt, choices, defaultID)
}

func runBubbleTeaPickWithOptions(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []tuiPickChoice, defaultID string, extraOptions ...setupwizard.StepOption) (string, error) {
	return tuipick.RunPickWithOptions(ctx, stdin, out, prompt, choices, defaultID, extraOptions...)
}

func runBubbleTeaChecklist(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []tuiPickChoice, selectedIDs []string) ([]string, error) {
	return tuipick.RunChecklist(ctx, stdin, out, prompt, choices, selectedIDs)
}

func bubbleTeaPickShouldFallback(err error) bool {
	return tuipick.ShouldFallback(err)
}
