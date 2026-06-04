package tuipick

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

type Choice struct {
	ID    string
	Label string
}

func RunPick(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []Choice, defaultID string) (string, error) {
	return RunPickWithOptions(ctx, stdin, out, prompt, choices, defaultID)
}

func RunPickWithOptions(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []Choice, defaultID string, extraOptions ...setupwizard.StepOption) (string, error) {
	wizardChoices := wizardChoices(choices)
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

func RunChecklist(ctx context.Context, stdin *os.File, out io.Writer, prompt string, choices []Choice, selectedIDs []string) ([]string, error) {
	result, err := setupwizard.New(
		setupwizard.WithInput(stdin),
		setupwizard.WithOutput(out),
	).Run(ctx, setupwizard.Checklist("selection", prompt, wizardChoices(choices), setupwizard.WithDefaultChoices(selectedIDs...)))
	if errors.Is(err, setupwizard.ErrAbort) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return result.Choices("selection"), nil
}

func ShouldFallback(err error) bool {
	return errors.Is(err, setupwizard.ErrRequiresTTY)
}

func wizardChoices(choices []Choice) []setupwizard.Choice {
	wizardChoices := make([]setupwizard.Choice, len(choices))
	for i, choice := range choices {
		wizardChoices[i] = setupwizard.Choice{ID: choice.ID, Label: choice.Label}
	}
	return wizardChoices
}
