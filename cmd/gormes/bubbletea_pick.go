package main

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

type tuiPickChoice struct {
	ID    string
	Label string
}

func runBubbleTeaPick(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []tuiPickChoice, defaultID string) (string, error) {
	wizardChoices := make([]setupwizard.Choice, len(choices))
	for i, choice := range choices {
		wizardChoices[i] = setupwizard.Choice{ID: choice.ID, Label: choice.Label}
	}
	stepOptions := []setupwizard.StepOption{}
	if strings.TrimSpace(defaultID) != "" {
		stepOptions = append(stepOptions, setupwizard.WithDefaultChoice(defaultID))
	}
	result, err := setupwizard.New(
		setupwizard.WithInput(stdin),
		setupwizard.WithOutput(out),
	).Run(ctx, setupwizard.Pick("selection", prompt, wizardChoices, stepOptions...))
	if errors.Is(err, setupwizard.ErrAbort) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return result.Choice("selection"), nil
}

func bubbleTeaPickShouldFallback(err error) bool {
	return errors.Is(err, setupwizard.ErrRequiresTTY)
}
