package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/spf13/cobra"
)

var errSetupRequiresTTY = errors.New("setup_requires_tty")

var setupSections = []string{"model", "tts", "terminal", "gateway", "tools", "agent"}

type setupCommandSeams struct {
	IsTTY            func() bool
	RunModelPicker   func(*cobra.Command) error
	LoadCurrentModel func() (cli.ProviderModel, error)
}

func newSetupCommand() *cobra.Command {
	return newSetupCommandWithSeams(defaultSetupCommandSeams())
}

func newSetupCommandWithSeams(seams setupCommandSeams) *cobra.Command {
	if seams.IsTTY == nil {
		seams.IsTTY = isStdinTTY
	}
	if seams.RunModelPicker == nil {
		seams.RunModelPicker = func(cmd *cobra.Command) error {
			pickerCmd := newModelCommand()
			pickerCmd.SetOut(cmd.OutOrStdout())
			pickerCmd.SetErr(cmd.ErrOrStderr())
			pickerCmd.SetIn(cmd.InOrStdin())
			pickerCmd.SilenceUsage = true
			pickerCmd.SilenceErrors = true
			return pickerCmd.ExecuteContext(cmd.Context())
		}
	}
	if seams.LoadCurrentModel == nil {
		seams.LoadCurrentModel = defaultSetupLoadCurrentModel
	}

	var nonInteractive bool
	var reset bool
	var reconfigure bool
	cmd := &cobra.Command{
		Use:          "setup [section]",
		Aliases:      []string{"onboard"},
		Short:        "Configure Gormes runtime sections",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if reset || reconfigure {
				return setupFullWizardUnsupported(cmd)
			}
			if len(args) == 0 {
				printSetupSections(cmd)
				return nil
			}
			section := strings.ToLower(strings.TrimSpace(args[0]))
			switch section {
			case "model":
				return runSetupModelSection(cmd, seams, nonInteractive)
			case "tts", "terminal", "gateway", "tools", "agent":
				return setupSectionUnsupported(cmd, section)
			default:
				return setupSectionUnsupported(cmd, section)
			}
		},
	}
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "use defaults/env and never prompt")
	cmd.Flags().BoolVar(&reset, "reset", false, "reset setup state (full wizard unsupported in this slice)")
	cmd.Flags().BoolVar(&reconfigure, "reconfigure", false, "re-run the full setup wizard (unsupported in this slice)")
	cmd.Flags().Bool("quick", false, "reserved for Hermes setup compatibility; no effect in this minimal slice")
	return cmd
}

func defaultSetupCommandSeams() setupCommandSeams {
	return setupCommandSeams{
		IsTTY:            isStdinTTY,
		LoadCurrentModel: defaultSetupLoadCurrentModel,
	}
}

func defaultSetupLoadCurrentModel() (cli.ProviderModel, error) {
	cfg, err := config.Load(nil)
	if err != nil {
		return cli.ProviderModel{}, err
	}
	return cli.ProviderModel{Provider: cfg.Hermes.Provider, Model: cfg.Hermes.Model}, nil
}

func printSetupSections(cmd *cobra.Command) {
	fmt.Fprintln(cmd.OutOrStdout(), "Available setup sections:")
	for _, section := range setupSections {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", section)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Run `gormes setup model` to configure the model/provider picker.")
}

func runSetupModelSection(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "Setup section: model")
	if nonInteractive {
		current, err := seams.LoadCurrentModel()
		if err != nil {
			return fmt.Errorf("setup model: load defaults: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "setup_model_defaults: provider=%s model=%s\n", current.Provider, current.Model)
		fmt.Fprintln(cmd.OutOrStdout(), "Provider auth was not changed. If credentials are missing, run: gormes auth add <provider>")
		return nil
	}
	if !seams.IsTTY() {
		fmt.Fprintln(cmd.ErrOrStderr(), "setup_requires_tty: run `gormes setup model --non-interactive` to use defaults without prompts")
		return errSetupRequiresTTY
	}
	return seams.RunModelPicker(cmd)
}

func setupSectionUnsupported(cmd *cobra.Command, section string) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "setup_section_unsupported: section=%s available=%s\n", section, setupSectionList())
	fmt.Fprintln(cmd.ErrOrStderr(), "Only `gormes setup model` is implemented in this slice; use `gormes config edit` for other setup surfaces.")
	return newExitCodeError(2, fmt.Errorf("setup_section_unsupported: %s", section))
}

func setupFullWizardUnsupported(cmd *cobra.Command) error {
	fmt.Fprintln(cmd.ErrOrStderr(), "setup_full_wizard_unsupported: --reset and --reconfigure require the full setup wizard row")
	fmt.Fprintln(cmd.ErrOrStderr(), "Use `gormes config edit` for configuration edits and `gormes auth add <provider>` for provider credentials.")
	return newExitCodeError(2, errors.New("setup_full_wizard_unsupported"))
}

func setupSectionList() string {
	return strings.Join(setupSections, "|")
}
