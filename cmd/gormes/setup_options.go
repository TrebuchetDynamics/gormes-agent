package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
	"github.com/spf13/cobra"
)

type setupOptionChoice struct {
	ID      string
	Label   string
	Aliases []string
}

func promptSetupOptionChoice(cmd *cobra.Command, title, linePrompt, defaultID string, choices []setupOptionChoice) (string, error) {
	defaultID = normalizeSetupOptionChoice(defaultID, choices, defaultID)
	if defaultID == "" && len(choices) > 0 {
		defaultID = choices[0].ID
	}
	if stdin, ok := cmd.InOrStdin().(*os.File); ok && setupInputIsTerminal(stdin) {
		selected, err := runBubbleTeaPickWithOptions(
			cmd.Context(),
			stdin,
			cmd.OutOrStdout(),
			title,
			setupOptionPickerChoices(choices),
			defaultID,
			setupwizard.WithRadioChoices(),
		)
		if err == nil {
			if selected == "" {
				return "", newExitCodeError(130, fmt.Errorf("setup_option_cancelled"))
			}
			return selected, nil
		}
		if !bubbleTeaPickShouldFallback(err) {
			return "", err
		}
	}

	answer, err := promptString(cmd, linePrompt, defaultID)
	if err != nil {
		return "", err
	}
	return normalizeSetupOptionChoice(answer, choices, defaultID), nil
}

func promptSetupYesNoOption(cmd *cobra.Command, title, linePrompt string, defaultValue bool) (bool, bool, error) {
	defaultID := "no"
	if defaultValue {
		defaultID = "yes"
	}
	choice, err := promptSetupOptionChoice(cmd, title, linePrompt, defaultID, []setupOptionChoice{
		{ID: "yes", Label: "Yes", Aliases: []string{"y", "true", "1", "on"}},
		{ID: "no", Label: "No", Aliases: []string{"n", "false", "0", "off"}},
	})
	if err != nil {
		return false, false, err
	}
	value, ok := parseSetupYesNo(choice, defaultValue)
	return value, ok, nil
}

func promptSetupChoice(cmd *cobra.Command, title, linePrompt, defaultValue string, choices []setupChoice) (string, error) {
	return promptSetupOptionChoice(cmd, title, linePrompt, defaultValue, setupChoicesToOptions(choices))
}

func setupChoicesToOptions(choices []setupChoice) []setupOptionChoice {
	options := make([]setupOptionChoice, len(choices))
	for i, choice := range choices {
		options[i] = setupOptionChoice{ID: choice.value, Label: choice.label}
	}
	return options
}

func setupOptionPickerChoices(options []setupOptionChoice) []tuiPickChoice {
	choices := make([]tuiPickChoice, len(options))
	for i, option := range options {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			label = option.ID
		}
		choices[i] = tuiPickChoice{ID: option.ID, Label: label}
	}
	return choices
}

func normalizeSetupOptionChoice(answer string, options []setupOptionChoice, defaultID string) string {
	answer = strings.TrimSpace(stripSetupInputNoise(answer))
	if answer == "" {
		return strings.TrimSpace(defaultID)
	}
	if idx, err := strconv.Atoi(answer); err == nil && idx >= 1 && idx <= len(options) {
		return options[idx-1].ID
	}
	normalized := normalizeSetupChoice(answer)
	for _, option := range options {
		if normalized == normalizeSetupChoice(option.ID) || normalized == normalizeSetupChoice(option.Label) {
			return option.ID
		}
		for _, alias := range option.Aliases {
			if normalized == normalizeSetupChoice(alias) {
				return option.ID
			}
		}
	}
	return normalized
}
