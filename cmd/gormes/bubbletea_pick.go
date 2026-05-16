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
	return runBubbleTeaPickWithOptions(ctx, stdin, out, prompt, choices, defaultID)
}

func runBubbleTeaPickWithOptions(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []tuiPickChoice, defaultID string, extraOptions ...setupwizard.StepOption) (string, error) {
	wizardChoices := make([]setupwizard.Choice, len(choices))
	for i, choice := range choices {
		wizardChoices[i] = setupwizard.Choice{ID: choice.ID, Label: choice.Label}
	}
	stepOptions := append([]setupwizard.StepOption(nil), extraOptions...)
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

func runBubbleTeaChecklist(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []tuiPickChoice, selectedIDs []string) ([]string, error) {
	wizardChoices := make([]setupwizard.Choice, len(choices))
	for i, choice := range choices {
		wizardChoices[i] = setupwizard.Choice{ID: choice.ID, Label: choice.Label}
	}
	result, err := setupwizard.New(
		setupwizard.WithInput(stdin),
		setupwizard.WithOutput(out),
	).Run(ctx, setupwizard.Checklist("selection", prompt, wizardChoices, setupwizard.WithDefaultChoices(selectedIDs...)))
	if errors.Is(err, setupwizard.ErrAbort) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return result.Choices("selection"), nil
}

func bubbleTeaPickShouldFallback(err error) bool {
	return errors.Is(err, setupwizard.ErrRequiresTTY)
}
