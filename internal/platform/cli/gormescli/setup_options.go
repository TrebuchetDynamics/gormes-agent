package gormescli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

type SetupOptionPromptRuntime struct {
	IsTerminal         func(*os.File) bool
	RunPick            func(context.Context, *os.File, io.Writer, string, []TUIPickChoice, string, ...setupwizard.StepOption) (string, error)
	PickShouldFallback func(error) bool
	PromptString       func(*cobra.Command, string, string) (string, error)
	ExitCodeError      func(int, error) error
}

func PromptSetupOptionChoice(cmd *cobra.Command, title, linePrompt, defaultID string, choices []SetupOptionChoice, runtime SetupOptionPromptRuntime) (string, error) {
	defaultID = NormalizeSetupAnswer(defaultID, choices, defaultID)
	if defaultID == "" && len(choices) > 0 {
		defaultID = choices[0].ID
	}
	if stdin, ok := cmd.InOrStdin().(*os.File); ok && setupPromptInputIsTerminal(runtime, stdin) {
		selected, err := setupPromptRunPick(runtime, cmd.Context(), stdin, cmd.OutOrStdout(), title, SetupOptionPickerChoices(choices), defaultID, setupwizard.WithRadioChoices())
		if err == nil {
			if selected == "" {
				return "", setupPromptExitCodeError(runtime, 130, fmt.Errorf("setup_option_cancelled"))
			}
			return selected, nil
		}
		if !setupPromptPickShouldFallback(runtime, err) {
			return "", err
		}
	}

	answer, err := setupPromptString(runtime, cmd, linePrompt, defaultID)
	if err != nil {
		return "", err
	}
	return NormalizeSetupAnswer(answer, choices, defaultID), nil
}

func PromptSetupYesNoOption(cmd *cobra.Command, title, linePrompt string, defaultValue bool, runtime SetupOptionPromptRuntime) (bool, bool, error) {
	defaultID := "no"
	if defaultValue {
		defaultID = "yes"
	}
	choice, err := PromptSetupOptionChoice(cmd, title, linePrompt, defaultID, []SetupOptionChoice{
		{ID: "yes", Label: "Yes", Aliases: []string{"y", "true", "1", "on"}},
		{ID: "no", Label: "No", Aliases: []string{"n", "false", "0", "off"}},
	}, runtime)
	if err != nil {
		return false, false, err
	}
	value, ok := ParseSetupYesNo(choice, defaultValue)
	return value, ok, nil
}

func PromptSetupChoice(cmd *cobra.Command, title, linePrompt, defaultValue string, choices []SetupChoice, runtime SetupOptionPromptRuntime) (string, error) {
	return PromptSetupOptionChoice(cmd, title, linePrompt, defaultValue, SetupChoicesToOptions(choices), runtime)
}

func SetupChoicesToOptions(choices []SetupChoice) []SetupOptionChoice {
	options := make([]SetupOptionChoice, len(choices))
	for i, choice := range choices {
		options[i] = SetupOptionChoice{ID: choice.Value, Label: choice.Label}
	}
	return options
}

func SetupOptionPickerChoices(options []SetupOptionChoice) []TUIPickChoice {
	choices := make([]TUIPickChoice, len(options))
	for i, option := range options {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			label = option.ID
		}
		choices[i] = TUIPickChoice{ID: option.ID, Label: label}
	}
	return choices
}

func setupPromptInputIsTerminal(runtime SetupOptionPromptRuntime, stdin *os.File) bool {
	if runtime.IsTerminal == nil {
		return false
	}
	return runtime.IsTerminal(stdin)
}

func setupPromptRunPick(runtime SetupOptionPromptRuntime, ctx context.Context, stdin *os.File, out io.Writer, title string, choices []TUIPickChoice, defaultID string, options ...setupwizard.StepOption) (string, error) {
	if runtime.RunPick == nil {
		return "", fmt.Errorf("setup option picker is not configured")
	}
	return runtime.RunPick(ctx, stdin, out, title, choices, defaultID, options...)
}

func setupPromptPickShouldFallback(runtime SetupOptionPromptRuntime, err error) bool {
	if runtime.PickShouldFallback == nil {
		return false
	}
	return runtime.PickShouldFallback(err)
}

func setupPromptString(runtime SetupOptionPromptRuntime, cmd *cobra.Command, prompt, defaultValue string) (string, error) {
	if runtime.PromptString == nil {
		return "", fmt.Errorf("setup option text prompt is not configured")
	}
	return runtime.PromptString(cmd, prompt, defaultValue)
}

func setupPromptExitCodeError(runtime SetupOptionPromptRuntime, code int, err error) error {
	if runtime.ExitCodeError != nil {
		return runtime.ExitCodeError(code, err)
	}
	return NewExitCodeError(code, err)
}
