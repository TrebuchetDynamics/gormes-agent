package gormescli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setup"
	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setupterminal"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type SetupTerminalOptions struct {
	ConfigPath       string
	PromptString     func(prompt, defaultValue string) (string, error)
	NewExitCodeError func(int, error) error
}

func RunSetupTerminalSection(cmd *cobra.Command, nonInteractive bool, opts SetupTerminalOptions) error {
	if opts.ConfigPath == "" {
		opts.ConfigPath = config.ConfigPath()
	}
	if opts.PromptString == nil {
		opts.PromptString = func(prompt, defaultValue string) (string, error) {
			return setupTerminalPromptString(cmd, prompt, defaultValue)
		}
	}
	if opts.NewExitCodeError == nil {
		opts.NewExitCodeError = NewExitCodeError
	}
	return setupterminal.Run(cmd.OutOrStdout(), cmd.ErrOrStderr(), nonInteractive, setupterminal.Runtime{
		ConfigPath:                  opts.ConfigPath,
		PromptChoice:                setupTerminalPromptChoice(cmd, opts),
		ShouldPrintStaticChoiceMenu: setupTerminalShouldPrintStaticChoiceMenu(cmd, nonInteractive),
		NewExitCodeError:            opts.NewExitCodeError,
	})
}

func setupTerminalPromptChoice(cmd *cobra.Command, opts SetupTerminalOptions) func(string, string, string, []setup.Choice) (string, error) {
	return func(title, linePrompt, defaultValue string, choices []setup.Choice) (string, error) {
		return PromptSetupChoice(cmd, title, linePrompt, defaultValue, choices, SetupOptionPromptRuntime{
			IsTerminal:         StdinIsTerminal,
			RunPick:            RunTUIPickWithOptions,
			PickShouldFallback: TUIPickShouldFallback,
			PromptString: func(cmd *cobra.Command, prompt, defaultValue string) (string, error) {
				return opts.PromptString(prompt, defaultValue)
			},
			ExitCodeError: opts.NewExitCodeError,
		})
	}
}

func setupTerminalShouldPrintStaticChoiceMenu(cmd *cobra.Command, nonInteractive bool) bool {
	if nonInteractive {
		return true
	}
	stdin, ok := cmd.InOrStdin().(*os.File)
	return !ok || !StdinIsTerminal(stdin)
}

func setupTerminalPromptString(cmd *cobra.Command, prompt, defaultValue string) (string, error) {
	fmt.Fprint(cmd.OutOrStdout(), prompt)
	var input string
	_, err := fmt.Fscanln(cmd.InOrStdin(), &input)
	if err != nil {
		if err.Error() == "unexpected newline" || strings.Contains(err.Error(), "expected") {
			return defaultValue, nil
		}
		return "", err
	}
	return strings.TrimSpace(input), nil
}
