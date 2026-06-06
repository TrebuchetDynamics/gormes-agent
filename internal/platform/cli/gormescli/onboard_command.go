package gormescli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

// OnboardCommandSeams carries prompt and action seams for the internal
// first-run onboarding command.
type OnboardCommandSeams struct {
	IsTTY        func() bool
	PromptAction func(*cobra.Command, cli.OnboardStep, string) (string, error)
	RunAction    func(*cobra.Command, cli.OnboardStep) error
}

// OnboardCommandOptions carries binary-owned values into the importable
// onboarding command without depending on cmd/gormes.
type OnboardCommandOptions struct {
	BuildProvenance func() OnboardBuildProvenance
	Home            func() string
	ConfigPath      func() string
	ExitCodeError   func(code int, err error) error

	BuildFirstRunPlan func(config.Config, cli.SetupTargetID, bool) cli.FirstRunPlan
	NormalizeChoice   func(string) string
	FirstRunCommand   func(string) string

	Seams OnboardCommandSeams
}

func (o OnboardCommandOptions) buildProvenance() OnboardBuildProvenance {
	if o.BuildProvenance == nil {
		return OnboardBuildProvenance{}
	}
	return o.BuildProvenance()
}

func (o OnboardCommandOptions) home() string {
	if o.Home == nil {
		return config.GormesHome()
	}
	return o.Home()
}

func (o OnboardCommandOptions) configPath() string {
	if o.ConfigPath == nil {
		return config.ConfigPath()
	}
	return o.ConfigPath()
}

func (o OnboardCommandOptions) normalizeChoice() func(string) string {
	if o.NormalizeChoice == nil {
		return NormalizeSetupValue
	}
	return o.NormalizeChoice
}

func (o OnboardCommandOptions) firstRunCommand() func(string) string {
	if o.FirstRunCommand == nil {
		return strings.TrimSpace
	}
	return o.FirstRunCommand
}

func (o OnboardCommandOptions) exitCodeError(code int, err error) error {
	if o.ExitCodeError == nil {
		return NewExitCodeError(code, err)
	}
	return o.ExitCodeError(code, err)
}

// NewOnboardCommand returns the hidden first-run onboarding command. The public
// replacement for operators remains `gormes setup`; this constructor preserves
// the internal status and wizard seams used by setup integrations and tests.
func NewOnboardCommand(opts OnboardCommandOptions) *cobra.Command {
	seams := normalizeOnboardCommandSeams(opts.Seams)

	var wizard bool
	var nonInteractive bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "onboard",
		Short:        "Use gormes setup",
		Hidden:       true,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(nil)
			if err != nil {
				return err
			}
			runtime := onboardCommandRuntime(cmd, opts, seams)
			if asJSON {
				if wizard {
					return WriteOnboardWizardPlanJSON(cmd.OutOrStdout(), cfg, nonInteractive || !seams.IsTTY(), runtime)
				}
				return WriteOnboardStatusJSON(cmd.OutOrStdout(), cfg, runtime)
			}
			if wizard {
				if nonInteractive || !seams.IsTTY() {
					PrintOnboardWizardPlan(cmd.OutOrStdout(), cfg, true)
					return nil
				}
				return runOnboardWizardWithRuntime(cmd, cfg, runtime, opts)
			}
			PrintOnboardStatus(cmd.OutOrStdout(), cfg, runtime)
			return nil
		},
	}
	cmd.Flags().BoolVar(&wizard, "wizard", false, "show the first-run wizard plan")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "render the wizard without prompts or external launches")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, home, config_path, provider, auth_configured, agents, bindings, ...}")
	return cmd
}

func normalizeOnboardCommandSeams(seams OnboardCommandSeams) OnboardCommandSeams {
	if seams.IsTTY == nil {
		seams.IsTTY = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
	}
	if seams.PromptAction == nil {
		seams.PromptAction = func(cmd *cobra.Command, step cli.OnboardStep, defaultAction string) (string, error) {
			return PromptOnboardAction(cmd.InOrStdin(), step, defaultAction)
		}
	}
	if seams.RunAction == nil {
		seams.RunAction = func(cmd *cobra.Command, step cli.OnboardStep) error {
			return RunOnboardActionRowBacked(cmd.OutOrStdout(), step)
		}
	}
	return seams
}

func onboardCommandRuntime(cmd *cobra.Command, opts OnboardCommandOptions, seams OnboardCommandSeams) OnboardRuntime {
	runtime := OnboardRuntime{
		Build:             opts.buildProvenance(),
		Home:              opts.home(),
		ConfigPath:        opts.configPath(),
		BuildFirstRunPlan: opts.BuildFirstRunPlan,
		NormalizeChoice:   opts.normalizeChoice(),
		FirstRunCommand:   opts.firstRunCommand(),
	}
	if seams.PromptAction != nil {
		runtime.PromptAction = func(_ io.Reader, step cli.OnboardStep, defaultAction string) (string, error) {
			return seams.PromptAction(cmd, step, defaultAction)
		}
	}
	if seams.RunAction != nil {
		runtime.RunAction = func(_ io.Writer, step cli.OnboardStep) error {
			return seams.RunAction(cmd, step)
		}
	}
	return runtime
}

func runOnboardWizardWithRuntime(cmd *cobra.Command, cfg config.Config, runtime OnboardRuntime, opts OnboardCommandOptions) error {
	err := RunOnboardWizard(cmd.InOrStdin(), cmd.OutOrStdout(), cfg, runtime)
	if err != nil && strings.HasPrefix(err.Error(), "onboard_action_invalid: ") {
		return opts.exitCodeError(2, err)
	}
	return err
}

func OnboardInvalidActionError(action string) error {
	return NewExitCodeError(2, fmt.Errorf("onboard_action_invalid: %s", action))
}
