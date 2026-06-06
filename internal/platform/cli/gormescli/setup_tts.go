package gormescli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setup"
	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setuptts"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type SetupTTSOptions struct {
	ConfigPath       string
	PromptString     func(prompt, defaultValue string) (string, error)
	NewExitCodeError func(int, error) error
}

func RunSetupTTSSection(cmd *cobra.Command, nonInteractive bool, opts SetupTTSOptions) error {
	if opts.ConfigPath == "" {
		opts.ConfigPath = config.ConfigPath()
	}
	if opts.PromptString == nil {
		opts.PromptString = func(prompt, defaultValue string) (string, error) {
			return setupTTSPromptString(cmd, prompt, defaultValue)
		}
	}
	if opts.NewExitCodeError == nil {
		opts.NewExitCodeError = NewExitCodeError
	}
	return setuptts.Run(cmd.OutOrStdout(), cmd.ErrOrStderr(), nonInteractive, setuptts.Runtime{
		ConfigPath:                  opts.ConfigPath,
		PromptChoice:                setupTTSPromptChoice(cmd, opts),
		PromptString:                opts.PromptString,
		ShouldPrintStaticChoiceMenu: setupTTSShouldPrintStaticChoiceMenu(cmd, nonInteractive),
		NewExitCodeError:            opts.NewExitCodeError,
	})
}

func setupTTSPromptChoice(cmd *cobra.Command, opts SetupTTSOptions) func(string, string, string, []setup.Choice) (string, error) {
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

func setupTTSShouldPrintStaticChoiceMenu(cmd *cobra.Command, nonInteractive bool) bool {
	if nonInteractive {
		return true
	}
	stdin, ok := cmd.InOrStdin().(*os.File)
	return !ok || !StdinIsTerminal(stdin)
}

func setupTTSPromptString(cmd *cobra.Command, prompt, defaultValue string) (string, error) {
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

func SetupTTSVoiceModel(ttsConfig map[string]any, provider string) string {
	return setuptts.VoiceModel(ttsConfig, provider)
}

func SetupIsTTSProviderChoice(value string) bool {
	return setuptts.IsProviderChoice(value)
}
