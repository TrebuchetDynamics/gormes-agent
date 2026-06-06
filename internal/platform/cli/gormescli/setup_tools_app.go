package gormescli

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	appsetup "github.com/TrebuchetDynamics/gormes-agent/internal/app/setup"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/toolsets"
)

type SetupToolOption = appsetup.ToolOption
type SetupInvalidToolSelectionError = appsetup.InvalidToolSelectionError
type SetupToolsOptions = appsetup.ToolsOptions
type SetupToolsChecklistChoice = appsetup.ToolsChecklistChoice

func RunSetupToolsSection(cmd *cobra.Command, nonInteractive bool, opts SetupToolsOptions) error {
	opts.Out = cmd.OutOrStdout()
	opts.NonInteractive = nonInteractive
	if !nonInteractive && opts.RunChecklist == nil {
		if stdin, ok := cmd.InOrStdin().(*os.File); ok && StdinIsTerminal(stdin) {
			opts.RunChecklist = func(title string, choices []SetupToolsChecklistChoice, selected []string) ([]string, error) {
				return RunTUIChecklist(cmd.Context(), stdin, cmd.OutOrStdout(), title, setupToolsChecklistChoices(choices), selected)
			}
			opts.PickShouldFallback = TUIPickShouldFallback
		}
	}
	err := appsetup.RunTools(opts)
	var invalid SetupInvalidToolSelectionError
	if errors.As(err, &invalid) {
		return NewExitCodeError(2, err)
	}
	return err
}

func setupToolsChecklistChoices(options []SetupToolsChecklistChoice) []TUIPickChoice {
	choices := make([]TUIPickChoice, len(options))
	for i, option := range options {
		choices[i] = TUIPickChoice{ID: option.ID, Label: option.Label}
	}
	return choices
}

func SetupToolOptions() ([]SetupToolOption, error) {
	return appsetup.ToolOptions()
}

func LoadSetupToolsConfig(path string) (map[string]any, toolsets.PlatformToolsetConfig, error) {
	return appsetup.LoadToolsConfig(path)
}

func WriteSetupToolsConfig(path string, doc map[string]any) error {
	return appsetup.WriteToolsConfig(path, doc)
}

func ParseSetupToolSelection(input string, options []SetupToolOption, current []string) ([]string, error) {
	return appsetup.ParseToolSelection(input, options, current)
}

func SetupToolsProviderRows(selected []string) []appsetup.ToolsProviderRow {
	return appsetup.ToolsProviderRows(selected)
}

func RenderSetupToolsProviderRows(out interface{ Write([]byte) (int, error) }, selected []string) {
	appsetup.RenderToolsProviderRows(out, selected)
}
