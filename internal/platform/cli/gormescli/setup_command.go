// Package gormescli provides the Cobra command wiring for the gormes agent.
// This file contains the setup command types and factory, moved from
// cmd/gormes/setup.go.
package gormescli

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

// SetupCommandSeams is the dependency injection struct for the setup command.
// Every field defaults to the production implementation when nil.
type SetupCommandSeams struct {
	IsTTY                          func() bool
	HasExistingInstall             func() (bool, error)
	ResetConfig                    func() (string, error)
	RunModelPicker                 func(*cobra.Command) error
	RunActiveProviderModelPicker   func(*cobra.Command, cli.ProviderModel) error
	LoadCurrentModel               func() (cli.ProviderModel, error)
	LoadProviderAuthStatus         func(string) (cli.ProviderAuthStatus, error)
	ChooseSetupAction              func(*cobra.Command, []SetupMenuOption, int) (SetupAction, error)
	ChooseSetupTarget              func(*cobra.Command, []cli.SetupTargetOption, int) (cli.SetupTargetID, error)
	ChooseSetupProvider            func(*cobra.Command, []cli.ProviderMenuEntry, int) (int, error)
	ChooseProviderCredentialAction func(*cobra.Command, SetupProviderCredentialPrompt) (SetupProviderCredentialAction, error)
	RunSetupProvider               func(*cobra.Command, bool) error
	RunProviderLiveTest            func(*cobra.Command) error
	RunProviderAuth                func(*cobra.Command, string) error
	RunProviderReauthenticate      func(*cobra.Command, string) error
	DetectHermesMigrationSource    func() string
	DetectOpenClawMigrationSource  func() string
	RunFullWizard                  func(*cobra.Command, bool) error
	RunSetupGateway                func(*cobra.Command, bool) error
	RunSetupTools                  func(*cobra.Command, bool) error
	RunGatewaySetupWizard          func(*cobra.Command, config.Config) (SetupGatewayWizardResult, error)
	RunTelegramGatewayWizard       func(*cobra.Command, config.TelegramCfg) (SetupTelegramGatewayAnswers, error)
	RunGatewayPlatform             func(*cobra.Command, string) error
	RunWhatsAppSetup               func(*cobra.Command) error
	LaunchChat                     func(*cobra.Command) error
	// Injected from root for RunE dispatch
	RunSetupSection         func(*cobra.Command, SetupCommandSeams, string, bool) error
	RunSetupQuick           func(*cobra.Command, SetupCommandSeams, bool, cli.SetupTargetID) error
	RunSetupFirstTimeChoice func(*cobra.Command, SetupCommandSeams, bool) error
	BuildProvenance         func() BuildProvenance
	NewExitCodeError        func(int, error) error
}

// SetupGatewayWizardResult is the result from the gateway bubble tea wizard.
// type alias, defined in setup_gateway.go as:

// SetupAction is a menu action identifier for the setup wizard.
type SetupAction string

const (
	SetupActionQuick           SetupAction = "quick"
	SetupActionFull            SetupAction = "full"
	SetupActionModelProvider   SetupAction = "model_provider"
	SetupActionFallback        SetupAction = "fallback"
	SetupActionTerminal        SetupAction = "terminal"
	SetupActionGateway         SetupAction = "gateway"
	SetupActionTools           SetupAction = "tools"
	SetupActionAgent           SetupAction = "agent"
	SetupActionMigrateHermes   SetupAction = "migrate_hermes"
	SetupActionMigrateOpenClaw SetupAction = "migrate_openclaw"
	SetupActionExit            SetupAction = "exit"
)

// SetupMenuOption is a setup wizard menu entry.
type SetupMenuOption struct {
	Action SetupAction
	Label  string
}

// SetupProviderCredentialAction describes a credential management choice.
type SetupProviderCredentialAction string

const (
	SetupProviderCredentialUseExisting    SetupProviderCredentialAction = "use_existing"
	SetupProviderCredentialReauthenticate SetupProviderCredentialAction = "reauthenticate"
	SetupProviderCredentialCancel         SetupProviderCredentialAction = "cancel"
)

// SetupProviderCredentialPrompt is passed to credential action prompts.
type SetupProviderCredentialPrompt struct {
	Provider      string
	ProviderLabel string
	Status        cli.ProviderAuthStatus
}

// NewSetupCommand builds the setup cobra command with flags and RunE dispatch.
// The seams struct provides all injected dependencies; nil fields get default
// implementations.
func NewSetupCommand(seams SetupCommandSeams) *cobra.Command {
	if seams.IsTTY == nil {
		seams.IsTTY = func() bool { return false }
	}
	if seams.HasExistingInstall == nil {
		seams.HasExistingInstall = func() (bool, error) { return false, nil }
	}
	if seams.ResetConfig == nil {
		seams.ResetConfig = ResetSetupDefaultConfig
	}
	if seams.RunModelPicker == nil {
		seams.RunModelPicker = func(cmd *cobra.Command) error {
			pickerCmd := NewModelCommand()
			pickerCmd.SetOut(cmd.OutOrStdout())
			pickerCmd.SetErr(cmd.ErrOrStderr())
			pickerCmd.SetIn(cmd.InOrStdin())
			pickerCmd.SetArgs([]string{})
			pickerCmd.SilenceUsage = true
			pickerCmd.SilenceErrors = true
			return pickerCmd.ExecuteContext(cmd.Context())
		}
	}
	if seams.RunActiveProviderModelPicker == nil {
		seams.RunActiveProviderModelPicker = RunSetupActiveProviderModelPicker
	}
	if seams.LoadCurrentModel == nil {
		seams.LoadCurrentModel = func() (cli.ProviderModel, error) {
			return cli.ProviderModel{}, nil
		}
	}
	if seams.LoadProviderAuthStatus == nil {
		seams.LoadProviderAuthStatus = func(provider string) (cli.ProviderAuthStatus, error) {
			return cli.ResolveAuthStatus(context.Background(), provider, cli.AuthStatusOptions{})
		}
	}
	if seams.ChooseSetupAction == nil {
		seams.ChooseSetupAction = func(*cobra.Command, []SetupMenuOption, int) (SetupAction, error) {
			return SetupActionExit, nil
		}
	}
	if seams.ChooseSetupTarget == nil {
		seams.ChooseSetupTarget = func(cmd *cobra.Command, targets []cli.SetupTargetOption, defaultOption int) (cli.SetupTargetID, error) {
			return "", nil
		}
	}
	if seams.ChooseSetupProvider == nil {
		seams.ChooseSetupProvider = func(*cobra.Command, []cli.ProviderMenuEntry, int) (int, error) {
			return -1, nil
		}
	}
	if seams.ChooseProviderCredentialAction == nil {
		seams.ChooseProviderCredentialAction = func(*cobra.Command, SetupProviderCredentialPrompt) (SetupProviderCredentialAction, error) {
			return SetupProviderCredentialUseExisting, nil
		}
	}
	if seams.RunSetupProvider == nil {
		seams.RunSetupProvider = func(*cobra.Command, bool) error { return nil }
	}
	if seams.RunProviderLiveTest == nil {
		seams.RunProviderLiveTest = func(*cobra.Command) error { return nil }
	}
	if seams.RunProviderAuth == nil {
		seams.RunProviderAuth = func(*cobra.Command, string) error { return nil }
	}
	if seams.RunProviderReauthenticate == nil {
		seams.RunProviderReauthenticate = seams.RunProviderAuth
	}
	if seams.DetectHermesMigrationSource == nil {
		seams.DetectHermesMigrationSource = func() string { return "" }
	}
	if seams.DetectOpenClawMigrationSource == nil {
		seams.DetectOpenClawMigrationSource = func() string { return "" }
	}
	if seams.RunSetupTools == nil {
		seams.RunSetupTools = func(cmd *cobra.Command, nonInteractive bool) error {
			return RunSetupToolsSection(cmd, nonInteractive, SetupToolsOptions{})
		}
	}
	if seams.RunWhatsAppSetup == nil {
		seams.RunWhatsAppSetup = func(*cobra.Command) error { return nil }
	}
	if seams.RunGatewaySetupWizard == nil {
		seams.RunGatewaySetupWizard = RunSetupGatewayBubbleTeaWizard
	}
	if seams.RunTelegramGatewayWizard == nil {
		seams.RunTelegramGatewayWizard = RunSetupTelegramBubbleTeaWizard
	}
	if seams.RunGatewayPlatform == nil {
		seams.RunGatewayPlatform = func(*cobra.Command, string) error { return nil }
	}
	if seams.LaunchChat == nil {
		seams.LaunchChat = func(*cobra.Command) error { return nil }
	}
	if seams.RunSetupGateway == nil {
		seams.RunSetupGateway = func(*cobra.Command, bool) error { return nil }
	}
	if seams.RunFullWizard == nil {
		seams.RunFullWizard = func(*cobra.Command, bool) error { return nil }
	}
	// RunE dispatch functions must be injected from root
	if seams.RunSetupSection == nil {
		seams.RunSetupSection = func(*cobra.Command, SetupCommandSeams, string, bool) error { return nil }
	}
	if seams.RunSetupQuick == nil {
		seams.RunSetupQuick = func(*cobra.Command, SetupCommandSeams, bool, cli.SetupTargetID) error { return nil }
	}
	if seams.RunSetupFirstTimeChoice == nil {
		seams.RunSetupFirstTimeChoice = func(*cobra.Command, SetupCommandSeams, bool) error { return nil }
	}
	if seams.BuildProvenance == nil {
		seams.BuildProvenance = func() BuildProvenance { return BuildProvenance{} }
	}
	if seams.NewExitCodeError == nil {
		seams.NewExitCodeError = func(code int, err error) error { return err }
	}

	var nonInteractive bool
	var reset bool
	var reconfigure bool
	var quick bool
	var targetFlag string
	var asJSON bool
	var plan bool
	cmd := &cobra.Command{
		Use:          "setup [section]",
		Short:        "Guided interactive setup — provider, model, and more",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			headless := nonInteractive || !seams.IsTTY()
			if reset {
				breadcrumb, err := seams.ResetConfig()
				if err != nil {
					return err
				}
				done, err := EmitSetupResetResult(cmd.OutOrStdout(), VersionBuildProvenance{Version: seams.BuildProvenance().Version, GitCommit: seams.BuildProvenance().GitCommit}, config.ConfigPath(), breadcrumb, asJSON)
				if err != nil || done {
					return err
				}
			}
			if len(args) > 0 {
				section := strings.ToLower(strings.TrimSpace(args[0]))
				return seams.RunSetupSection(cmd, seams, section, nonInteractive)
			}
			if quick {
				return seams.RunSetupQuick(cmd, seams, headless, cli.SetupTargetID(targetFlag))
			}
			if headless {
				PrintSetupSections(cmd)
				return nil
			}
			existing, err := seams.HasExistingInstall()
			if err != nil {
				return err
			}
			if reconfigure {
				if existing {
					return seams.RunFullWizard(cmd, false)
				}
				return seams.RunSetupFirstTimeChoice(cmd, seams, false)
			}
			if existing {
				return seams.RunFullWizard(cmd, false)
			}
			return seams.RunSetupFirstTimeChoice(cmd, seams, false)
		},
	}
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "use defaults/env and never prompt")
	cmd.Flags().BoolVar(&reset, "reset", false, "DESTRUCTIVE: overwrite config.toml back to defaults, then re-run the setup wizard")
	cmd.Flags().BoolVar(&reconfigure, "reconfigure", false, "re-run the setup wizard against the current config (non-destructive; existing values are kept where the operator skips a step)")
	cmd.Flags().BoolVar(&quick, "quick", false, "configure missing setup items only")
	cmd.Flags().StringVar(&targetFlag, "target", "", "setup target for --quick: terminal, telegram, whatsapp, discord, slack, or navivox")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON for `--reset`: `{build, action: 'reset', config_path, breadcrumb_path}`")
	cmd.Flags().BoolVar(&plan, "plan", false, "show messaging-platform setup plan without writing files or calling live APIs")
	return cmd
}

// DefaultSetupLoadCurrentModel loads the current provider/model from config.
func DefaultSetupLoadCurrentModel() (cli.ProviderModel, error) {
	cfg, err := config.Load(nil)
	if err != nil {
		return cli.ProviderModel{}, err
	}
	return cli.ProviderModel{Provider: cfg.Hermes.Provider, Model: cfg.Hermes.Model}, nil
}

// DefaultSetupHasExistingInstall reports whether a prior install is detected.
func DefaultSetupHasExistingInstall(cfg config.Config) bool {
	apiKey := strings.TrimSpace(cfg.Hermes.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("GORMES_API_KEY"))
	}
	return strings.TrimSpace(cfg.Hermes.Provider) != "" ||
		strings.TrimSpace(cfg.Hermes.Endpoint) != "" ||
		apiKey != ""
}

// SetupTopLevelOptions returns the main setup wizard menu options.
func SetupTopLevelOptions() []SetupMenuOption {
	return []SetupMenuOption{
		{Action: SetupActionQuick, Label: "Quick Setup - configure missing items only"},
		{Action: SetupActionFull, Label: "Full Setup - reconfigure everything"},
		{Action: SetupActionModelProvider, Label: "Model & Provider"},
		{Action: SetupActionFallback, Label: "Fallback Providers"},
		{Action: SetupActionTerminal, Label: "Terminal Backend"},
		{Action: SetupActionGateway, Label: "Messaging Platforms (Gateway)"},
		{Action: SetupActionTools, Label: "Tools"},
		{Action: SetupActionAgent, Label: "Agent Settings"},
		{Action: SetupActionMigrateHermes, Label: "Migrate from Hermes"},
		{Action: SetupActionMigrateOpenClaw, Label: "Migrate from OpenClaw"},
		{Action: SetupActionExit, Label: "Exit"},
	}
}
