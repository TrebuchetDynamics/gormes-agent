package gormescli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	appwhatsapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/whatsapp"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

var errSetupRequiresTTY = errors.New("setup_requires_tty")

type setupCommandSeams struct {
	IsTTY                    func() bool
	RunSetupGateway          func(*cobra.Command, bool) error
	RunGatewaySetupWizard    func(*cobra.Command, config.Config) (setupGatewayWizardResult, error)
	RunTelegramGatewayWizard func(*cobra.Command, config.TelegramCfg) (setupTelegramGatewayAnswers, error)
	RunGatewayPlatform       func(*cobra.Command, string) error
	RunWhatsAppSetup         func(*cobra.Command) error
}

type setupCommandFakeSeams struct {
	isTTY                    bool
	runSetupGateway          func(*cobra.Command, bool) error
	runGatewaySetupWizard    func(*cobra.Command, config.Config) (setupGatewayWizardResult, error)
	runTelegramGatewayWizard func(*cobra.Command, config.TelegramCfg) (setupTelegramGatewayAnswers, error)
	runGatewayPlatform       func(*cobra.Command, string) error
	runWhatsAppSetup         func(*cobra.Command) error
}

func (f *setupCommandFakeSeams) seams() setupCommandSeams {
	seams := setupCommandSeams{
		IsTTY:                    func() bool { return f.isTTY },
		RunSetupGateway:          f.runSetupGateway,
		RunGatewaySetupWizard:    firstSetupGatewayWizardSeam(f.runGatewaySetupWizard),
		RunTelegramGatewayWizard: f.runTelegramGatewayWizard,
		RunGatewayPlatform:       f.runGatewayPlatform,
		RunWhatsAppSetup:         f.runWhatsAppSetup,
	}
	if seams.RunWhatsAppSetup == nil {
		seams.RunWhatsAppSetup = runSetupWhatsAppPlanForTest
	}
	return seams
}

func newSetupCommandWithSeams(seams setupCommandSeams) *cobra.Command {
	if seams.IsTTY == nil {
		seams.IsTTY = func() bool { return false }
	}
	if seams.RunWhatsAppSetup == nil {
		seams.RunWhatsAppSetup = runSetupWhatsAppPlanForTest
	}
	if seams.RunGatewaySetupWizard == nil {
		seams.RunGatewaySetupWizard = firstSetupGatewayWizardSeam(nil)
	}
	if seams.RunGatewayPlatform == nil {
		seams.RunGatewayPlatform = func(cmd *cobra.Command, platform string) error {
			return RunSetupGatewayPlatform(cmd, platform, setupGatewayTestRuntime(seams))
		}
	}
	if seams.RunSetupGateway == nil {
		seams.RunSetupGateway = func(cmd *cobra.Command, nonInteractive bool) error {
			return RunSetupGatewaySection(cmd, nonInteractive, setupGatewayTestSeams(seams), setupGatewayTestRuntime(seams))
		}
	}

	var nonInteractive bool
	cmd := &cobra.Command{
		Use:          "setup [section]",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			section := ""
			if len(args) > 0 {
				section = strings.TrimSpace(args[0])
			}
			switch section {
			case "profiles":
				return runSetupProfilesTestSection(cmd, seams, nonInteractive)
			case "gateway":
				return runSetupGatewayTestSection(cmd, seams, nonInteractive)
			case "telegram":
				return runSetupTelegramTestSection(cmd, seams, nonInteractive)
			case "navivox":
				return runSetupNavivoxTestSection(cmd, seams, nonInteractive)
			case "tts":
				return runSetupTTSTestSection(cmd, nonInteractive)
			case "terminal":
				return runSetupTerminalTestSection(cmd, nonInteractive)
			default:
				WriteSetupSectionUnsupported(cmd.ErrOrStderr(), section, "profiles,gateway,telegram,navivox,tts,terminal")
				return SetupSectionUnsupportedError(section)
			}
		},
	}
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "run without prompts")
	cmd.Flags().Bool("plan", false, "show messaging-platform setup plan without writing files or calling live APIs")
	return cmd
}

func runSetupProfilesTestSection(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprint(out, RenderSetupSectionHeader("Profiles"))
	var captured bytes.Buffer
	cmd.SetOut(io.MultiWriter(out, &captured))
	err := RunSetupProfilesSection(cmd, SetupProfilesOptions{NonInteractive: nonInteractive, IsTTY: seams.IsTTY, RequiresTTYError: errSetupRequiresTTY})
	cmd.SetOut(out)
	if err == nil && !SetupSectionSuppressSuccessFooter("profiles", captured.String()) {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Profiles configuration complete!")
	}
	return err
}

func runSetupGatewayTestSection(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprint(out, RenderSetupSectionHeader("Messaging Gateway"))
	var captured bytes.Buffer
	cmd.SetOut(io.MultiWriter(out, &captured))
	err := seams.RunSetupGateway(cmd, nonInteractive || !seams.IsTTY())
	cmd.SetOut(out)
	if err == nil && !SetupSectionSuppressSuccessFooter("gateway", captured.String()) {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Messaging Gateway configuration complete!")
	}
	return err
}

func runSetupTelegramTestSection(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprint(out, RenderSetupSectionHeader("Telegram"))
	var captured bytes.Buffer
	cmd.SetOut(io.MultiWriter(out, &captured))
	err := RunSetupTelegramSection(cmd, nonInteractive || !seams.IsTTY(), setupGatewayTestSeams(seams), setupGatewayTestRuntime(seams))
	cmd.SetOut(out)
	if err == nil && !SetupSectionSuppressSuccessFooter("telegram", captured.String()) {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Telegram configuration complete!")
	}
	return err
}

func runSetupNavivoxTestSection(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprint(out, RenderSetupSectionHeader("Navivox"))
	if nonInteractive || !seams.IsTTY() {
		return errSetupRequiresTTY
	}
	cfg, err := config.Load(nil)
	if err != nil {
		return err
	}
	var captured bytes.Buffer
	cmd.SetOut(io.MultiWriter(out, &captured))
	err = RunSetupNavivoxGateway(cmd, cfg, setupGatewayNavivoxOptions(cmd, setupGatewayRuntimeDefaults(setupGatewayTestRuntime(seams))))
	cmd.SetOut(out)
	if err == nil && !SetupSectionSuppressSuccessFooter("navivox", captured.String()) {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Navivox configuration complete!")
	}
	return err
}

func runSetupTTSTestSection(cmd *cobra.Command, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprint(out, RenderSetupSectionHeader("Text-to-Speech"))
	var captured bytes.Buffer
	cmd.SetOut(io.MultiWriter(out, &captured))
	err := RunSetupTTSSection(cmd, nonInteractive, SetupTTSOptions{})
	cmd.SetOut(out)
	if err == nil && !SetupSectionSuppressSuccessFooter("tts", captured.String()) {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Text-to-Speech configuration complete!")
	}
	return err
}

func runSetupTerminalTestSection(cmd *cobra.Command, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprint(out, RenderSetupSectionHeader("Terminal Backend"))
	var captured bytes.Buffer
	cmd.SetOut(io.MultiWriter(out, &captured))
	err := RunSetupTerminalSection(cmd, nonInteractive, SetupTerminalOptions{})
	cmd.SetOut(out)
	if err == nil && !SetupSectionSuppressSuccessFooter("terminal", captured.String()) {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Terminal Backend configuration complete!")
	}
	return err
}

func setupGatewayTestSeams(seams setupCommandSeams) SetupGatewaySeams {
	return SetupGatewaySeams{
		RunGatewaySetupWizard:    seams.RunGatewaySetupWizard,
		RunTelegramGatewayWizard: seams.RunTelegramGatewayWizard,
		RunGatewayPlatform:       seams.RunGatewayPlatform,
	}
}

func setupGatewayTestRuntime(seams setupCommandSeams) SetupGatewayRuntime {
	return SetupGatewayRuntime{
		NewExitCodeError: NewExitCodeError,
		RunWhatsAppSetup: seams.RunWhatsAppSetup,
	}
}

func firstSetupGatewayWizardSeam(fn func(*cobra.Command, config.Config) (setupGatewayWizardResult, error)) func(*cobra.Command, config.Config) (setupGatewayWizardResult, error) {
	if fn != nil {
		return fn
	}
	return func(cmd *cobra.Command, cfg config.Config) (setupGatewayWizardResult, error) {
		options := setupGatewayPlatformOptions(cfg)
		selection, err := setupGatewayPromptString(cmd, "Messaging platforms (comma-separated numbers or ids, blank to keep current): ", "")
		if err != nil {
			return setupGatewayWizardResult{}, err
		}
		selected, keepCurrent, err := parseSetupGatewaySelection(selection, options)
		if err != nil {
			return setupGatewayWizardResult{}, err
		}
		if keepCurrent {
			return setupGatewayWizardResult{}, nil
		}
		return setupGatewayWizardResult{SelectedPlatforms: selected}, nil
	}
}

func runSetupWhatsAppPlanForTest(cmd *cobra.Command) error {
	whatsAppCmd := appwhatsapp.NewCommand(appwhatsapp.Options{BuildProvenance: func() appwhatsapp.BuildProvenance {
		return appwhatsapp.BuildProvenance{Version: "test-version", GitCommit: "test-git"}
	}})
	whatsAppCmd.SetOut(cmd.OutOrStdout())
	whatsAppCmd.SetErr(cmd.ErrOrStderr())
	whatsAppCmd.SetIn(cmd.InOrStdin())
	whatsAppCmd.SetArgs([]string{"--plan"})
	whatsAppCmd.SilenceUsage = true
	whatsAppCmd.SilenceErrors = true
	return whatsAppCmd.ExecuteContext(cmd.Context())
}

func runSetupTestCommand(t *testing.T, seams setupCommandSeams, args ...string) (string, string, error) {
	t.Helper()
	return runSetupTestCommandWithInput(t, seams, "", args...)
}

func runSetupTestCommandWithInput(t *testing.T, seams setupCommandSeams, input string, args ...string) (string, string, error) {
	t.Helper()
	return executeCobraCommandForTest(newSetupCommandWithSeams(seams), cobraCommandExecutionOptions{Input: strings.NewReader(input), SilenceUsage: true, SilenceErrors: true}, args...)
}
