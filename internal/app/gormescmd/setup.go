package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/profileapp"
	tuiapp "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/tuiapp"
	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var errSetupRequiresTTY = errors.New("setup_requires_tty")

var setupReadPassword = term.ReadPassword
var setupInputIsTerminal = stdinIsTerminal

var setupRegistry = gormescli.MustSetupRegistry(defaultSetupSections())
var setupSections = setupRegistry.Names()

// setupSectionLabels is the Gormes-owned section→label map for the boxed
// `│ Gormes Setup — <Label> │` per-section header (parity with hermes
// setup.py@55c9f3206:3199 `│ ⚕ Hermes Setup — {label} │`). Labels are
// Gormes-owned wording; some sections (provider/workspace/bindings) have no
// 1:1 Hermes equivalent — this map is the documented owned divergence.
var setupSectionLabels = setupRegistry.Labels()

func defaultSetupSections() []gormescli.SetupSection {
	sections := []gormescli.SetupSection{
		{Name: "agent", Label: "Agent Settings", Module: gormescli.SetupModuleGateway},
		{Name: "workspace", Label: "Workspace", Module: gormescli.SetupModuleGateway},
	}
	sections = append(providermodule.SetupSections(), sections...)
	sections = append(sections, profileapp.SetupSections()...)
	sections = append(sections,
		gormescli.SetupSection{Name: "bindings", Label: "Channel Bindings", Module: gormescli.SetupModuleGateway},
		gormescli.SetupSection{Name: "tts", Label: "Text-to-Speech", Module: gormescli.SetupModuleTTS},
		gormescli.SetupSection{Name: "terminal", Label: "Terminal Backend", Module: gormescli.SetupModuleTUI},
		gormescli.SetupSection{Name: "gateway", Label: "Messaging Gateway", Module: gormescli.SetupModuleGateway},
		gormescli.SetupSection{Name: "telegram", Label: "Telegram", Module: gormescli.SetupModuleGateway},
		gormescli.SetupSection{Name: "navivox", Label: "Navivox", Module: gormescli.SetupModuleNavivox},
		gormescli.SetupSection{Name: "tools", Label: "Tools", Module: gormescli.SetupModuleTools},
		gormescli.SetupSection{Name: "router", Label: "Router", Module: gormescli.SetupModuleProviders},
	)
	return sections
}

func setupCanonicalSection(section string) string {
	return gormescli.SetupCanonicalSection(section)
}

func setupKnownSection(section string) bool {
	return gormescli.SetupKnownSection(section)
}

func setupSectionLabel(section string) string {
	return gormescli.SetupSectionLabel(section)
}

const (
	providerOpenAI    = "openai"
	providerAnthropic = "anthropic"
	providerDeepSeek  = "deepseek"
	providerGroq      = "groq"
	providerOllama    = "ollama"
)

var knownProviderEndpoints = map[string]string{
	providerOpenAI:    "https://api.openai.com/v1",
	providerAnthropic: "https://api.anthropic.com/v1",
	providerDeepSeek:  "https://api.deepseek.com/v1",
	providerGroq:      "https://api.groq.com/openai/v1",
	providerOllama:    "http://localhost:11434/v1",
	"openai-codex":    "https://chatgpt.com/backend-api/codex",
	"openrouter":      providermodule.OpenRouterBaseURL,
	"opencode":        "https://opencode.ai/zen/v1",
	"opencode-go":     "https://opencode.ai/zen/go/v1",
}

var knownProviderModels = map[string]string{
	providerOpenAI:    "gpt-4o",
	providerAnthropic: "claude-sonnet-4-20250514",
	providerDeepSeek:  "deepseek-chat",
	providerGroq:      "llama-3.3-70b-versatile",
	providerOllama:    "llama3",
	"openai-codex":    "gpt-5.2",
	"openrouter":      "moonshotai/kimi-k2.6",
	"opencode":        "gpt-5.2",
	"opencode-go":     "gpt-5.2",
}

// Types aliased to gormescli — the canonical definitions now live in
// internal/platform/cli/gormescli/setup_command.go.
type setupCommandSeams = gormescli.SetupCommandSeams
type setupGatewayWizardResult = gormescli.SetupGatewayWizardResult
type setupTelegramGatewayAnswers = gormescli.SetupTelegramGatewayAnswers
type setupAction = gormescli.SetupAction
type setupMenuOption = gormescli.SetupMenuOption
type setupProviderCredentialAction = gormescli.SetupProviderCredentialAction
type setupProviderCredentialPrompt = gormescli.SetupProviderCredentialPrompt

const (
	setupActionQuick           = gormescli.SetupActionQuick
	setupActionFull            = gormescli.SetupActionFull
	setupActionModelProvider   = gormescli.SetupActionModelProvider
	setupActionFallback        = gormescli.SetupActionFallback
	setupActionTerminal        = gormescli.SetupActionTerminal
	setupActionGateway         = gormescli.SetupActionGateway
	setupActionTools           = gormescli.SetupActionTools
	setupActionAgent           = gormescli.SetupActionAgent
	setupActionMigrateHermes   = gormescli.SetupActionMigrateHermes
	setupActionMigrateOpenClaw = gormescli.SetupActionMigrateOpenClaw
	setupActionExit            = gormescli.SetupActionExit
)

const (
	setupProviderCredentialUseExisting    = gormescli.SetupProviderCredentialUseExisting
	setupProviderCredentialReauthenticate = gormescli.SetupProviderCredentialReauthenticate
	setupProviderCredentialCancel         = gormescli.SetupProviderCredentialCancel
)

func newSetupCommand() *cobra.Command {
	return newSetupCommandWithSeams(defaultSetupCommandSeams())
}

func newSetupCommandWithSeams(seams setupCommandSeams) *cobra.Command {
	gormescli.InitSetupRegistry(defaultSetupSections())
	seams = fillSetupCommandSeamDefaults(seams)
	return gormescli.NewSetupCommand(seams)
}

func fillSetupCommandSeamDefaults(seams setupCommandSeams) setupCommandSeams {
	defaults := defaultSetupCommandSeams()
	if seams.IsTTY == nil {
		seams.IsTTY = defaults.IsTTY
	}
	if seams.HasExistingInstall == nil {
		seams.HasExistingInstall = defaults.HasExistingInstall
	}
	if seams.ResetConfig == nil {
		seams.ResetConfig = defaults.ResetConfig
	}
	if seams.RunModelPicker == nil {
		seams.RunModelPicker = defaults.RunModelPicker
	}
	if seams.RunActiveProviderModelPicker == nil {
		seams.RunActiveProviderModelPicker = defaults.RunActiveProviderModelPicker
	}
	if seams.LoadCurrentModel == nil {
		seams.LoadCurrentModel = defaults.LoadCurrentModel
	}
	if seams.LoadProviderAuthStatus == nil {
		seams.LoadProviderAuthStatus = defaults.LoadProviderAuthStatus
	}
	if seams.ChooseSetupAction == nil {
		seams.ChooseSetupAction = defaults.ChooseSetupAction
	}
	if seams.ChooseSetupTarget == nil {
		seams.ChooseSetupTarget = defaults.ChooseSetupTarget
	}
	if seams.ChooseSetupProvider == nil {
		seams.ChooseSetupProvider = defaults.ChooseSetupProvider
	}
	if seams.ChooseProviderCredentialAction == nil {
		seams.ChooseProviderCredentialAction = defaults.ChooseProviderCredentialAction
	}
	if seams.RunSetupProvider == nil {
		seams.RunSetupProvider = func(cmd *cobra.Command, nonInteractive bool) error {
			return runSetupProviderSection(cmd, seams, nonInteractive)
		}
	}
	if seams.RunProviderLiveTest == nil {
		seams.RunProviderLiveTest = defaults.RunProviderLiveTest
	}
	if seams.RunProviderAuth == nil {
		seams.RunProviderAuth = defaults.RunProviderAuth
	}
	if seams.DetectHermesMigrationSource == nil {
		seams.DetectHermesMigrationSource = defaults.DetectHermesMigrationSource
	}
	if seams.DetectOpenClawMigrationSource == nil {
		seams.DetectOpenClawMigrationSource = defaults.DetectOpenClawMigrationSource
	}
	if seams.RunFullWizard == nil {
		seams.RunFullWizard = func(cmd *cobra.Command, nonInteractive bool) error {
			return runSetupFullWizard(cmd, seams, nonInteractive)
		}
	}
	if seams.RunSetupGateway == nil {
		seams.RunSetupGateway = func(cmd *cobra.Command, nonInteractive bool) error {
			return runSetupGatewaySection(cmd, seams, nonInteractive)
		}
	}
	if seams.RunSetupTools == nil {
		seams.RunSetupTools = func(cmd *cobra.Command, nonInteractive bool) error {
			return gormescli.RunSetupToolsSection(cmd, nonInteractive, gormescli.SetupToolsOptions{})
		}
	}
	if seams.RunGatewaySetupWizard == nil {
		seams.RunGatewaySetupWizard = gormescli.RunSetupGatewayBubbleTeaWizard
	}
	if seams.RunTelegramGatewayWizard == nil {
		seams.RunTelegramGatewayWizard = gormescli.RunSetupTelegramBubbleTeaWizard
	}
	if seams.RunGatewayPlatform == nil {
		seams.RunGatewayPlatform = func(cmd *cobra.Command, platform string) error {
			return gormescli.RunSetupGatewayPlatform(cmd, platform, setupGatewayRuntime(cmd, seams))
		}
	}
	if seams.RunWhatsAppSetup == nil {
		seams.RunWhatsAppSetup = runSetupWhatsAppCommand
	}
	if seams.LaunchChat == nil {
		seams.LaunchChat = launchSetupChat
	}
	seams.RunSetupSection = defaults.RunSetupSection
	seams.RunSetupQuick = defaults.RunSetupQuick
	seams.RunSetupFirstTimeChoice = defaults.RunSetupFirstTimeChoice
	seams.BuildProvenance = defaults.BuildProvenance
	seams.NewExitCodeError = defaults.NewExitCodeError
	return seams
}

func defaultSetupCommandSeams() setupCommandSeams {
	return setupCommandSeams{
		IsTTY:              isStdinTTY,
		HasExistingInstall: defaultSetupHasExistingInstall,
		ResetConfig:        gormescli.ResetSetupDefaultConfig,
		RunModelPicker: func(cmd *cobra.Command) error {
			pickerCmd := gormescli.NewModelCommand()
			pickerCmd.SetOut(cmd.OutOrStdout())
			pickerCmd.SetErr(cmd.ErrOrStderr())
			pickerCmd.SetIn(cmd.InOrStdin())
			pickerCmd.SetArgs([]string{})
			pickerCmd.SilenceUsage = true
			pickerCmd.SilenceErrors = true
			return pickerCmd.ExecuteContext(cmd.Context())
		},
		LoadCurrentModel:             defaultSetupLoadCurrentModel,
		RunActiveProviderModelPicker: gormescli.RunSetupActiveProviderModelPicker,
		LoadProviderAuthStatus: func(provider string) (cli.ProviderAuthStatus, error) {
			return cli.ResolveAuthStatus(context.Background(), provider, cli.AuthStatusOptions{})
		},
		ChooseSetupAction:              promptSetupAction,
		ChooseSetupTarget:              promptSetupTarget,
		ChooseSetupProvider:            promptSetupProviderChoice,
		ChooseProviderCredentialAction: promptSetupProviderCredentialAction,
		RunProviderLiveTest:            runSetupProviderLiveTest,
		RunProviderAuth:                runSetupProviderAuth,
		DetectHermesMigrationSource:    tuiapp.DetectHermesMigrationSource,
		DetectOpenClawMigrationSource:  tuiapp.DetectOpenClawMigrationSource,
		RunSetupSection:                runSetupSection,
		RunSetupQuick:                  runSetupQuick,
		RunSetupFirstTimeChoice:        runSetupFirstTimeChoice,
		BuildProvenance: func() gormescli.BuildProvenance {
			b := newBuildProvenance()
			return gormescli.BuildProvenance{Version: b.Version, GitCommit: b.GitCommit}
		},
		NewExitCodeError: newExitCodeError,
	}
}

func defaultSetupLoadCurrentModel() (cli.ProviderModel, error) {
	cfg, err := config.Load(nil)
	if err != nil {
		return cli.ProviderModel{}, err
	}
	return cli.ProviderModel{Provider: cfg.Hermes.Provider, Model: cfg.Hermes.Model}, nil
}

func defaultSetupHasExistingInstall() (bool, error) {
	cfg, err := config.Load(nil)
	if err != nil {
		return false, fmt.Errorf("setup: load config: %w", err)
	}
	return strings.TrimSpace(cfg.Hermes.Provider) != "" ||
		strings.TrimSpace(cfg.Hermes.Endpoint) != "" ||
		strings.TrimSpace(os.Getenv("GORMES_API_KEY")) != "" ||
		setupProviderAutoDetectEnvProvider() != "", nil
}

func printSetupSections(cmd *cobra.Command) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Available setup sections:")
	for _, section := range setupRegistry.Sections() {
		fmt.Fprintf(out, "  - %-10s %s\n", section.Name, section.Label)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Quick starts:")
	fmt.Fprintln(out, "  Interactive menu: gormes setup")
	fmt.Fprintln(out, "  Terminal/TUI quick setup: gormes setup --quick --target tui")
	fmt.Fprintln(out, "  Provider setup: gormes setup provider")
	fmt.Fprintln(out, "  Telegram setup: gormes setup telegram")
	fmt.Fprintln(out, "  Router setup:   gormes setup router")
}

func firstRunSetupOptions(seams setupCommandSeams) []setupMenuOption {
	options := gormescli.FirstRunSetupOptions(gormescli.SetupFirstRunSourceSeams{
		DetectHermesMigrationSource:   seams.DetectHermesMigrationSource,
		DetectOpenClawMigrationSource: seams.DetectOpenClawMigrationSource,
	})
	out := make([]setupMenuOption, len(options))
	for i, option := range options {
		out[i] = setupMenuOption{Action: setupAction(option.Action), Label: option.Label}
	}
	return out
}

func promptSetupTarget(cmd *cobra.Command, targets []cli.SetupTargetOption, defaultOption int) (cli.SetupTargetID, error) {
	return gormescli.PromptSetupTarget(cmd, targets, defaultOption, gormescli.SetupTargetPromptOptions{
		PromptString:       promptString,
		NewExitCodeError:   newExitCodeError,
		PickShouldFallback: bubbleTeaPickShouldFallback,
	})
}

func runSetupQuick(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool, requestedTarget cli.SetupTargetID) error {
	return gormescli.RunSetupQuick(cmd, setupQuickSeams(seams), nonInteractive, requestedTarget)
}

func setupQuickSeams(seams setupCommandSeams) gormescli.SetupQuickSeams {
	return gormescli.SetupQuickSeams{
		ChooseSetupTarget:   seams.ChooseSetupTarget,
		RunSetupProvider:    seams.RunSetupProvider,
		RunProviderLiveTest: seams.RunProviderLiveTest,
		LoadCurrentModel:    seams.LoadCurrentModel,
		RunSetupModelSection: func(cmd *cobra.Command, nonInteractive bool) error {
			return runSetupModelSection(cmd, seams, nonInteractive)
		},
		RunWhatsAppSetup:              seams.RunWhatsAppSetup,
		RunTelegramSetup:              func(cmd *cobra.Command) error { return runSetupQuickTelegram(cmd, seams) },
		RunGatewayPlatform:            seams.RunGatewayPlatform,
		LaunchChat:                    seams.LaunchChat,
		DetectHermesMigrationSource:   seams.DetectHermesMigrationSource,
		DetectOpenClawMigrationSource: seams.DetectOpenClawMigrationSource,
		NewExitCodeError:              newExitCodeError,
	}
}

func runSetupQuickTelegram(cmd *cobra.Command, seams setupCommandSeams) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("setup telegram: load config: %w", err)
	}
	answers, err := seams.RunTelegramGatewayWizard(cmd, cfg.Telegram)
	if err != nil {
		if errors.Is(err, setupwizard.ErrRequiresTTY) {
			return newExitCodeError(2, fmt.Errorf("setup_telegram_requires_tty: run `gormes setup gateway --plan` for offline guidance, or run `gormes setup --quick --target telegram` in a terminal"))
		}
		return err
	}
	return gormescli.ApplySetupTelegramGatewayAnswers(cmd, cfg.Telegram, answers, setupGatewayRuntime(cmd, seams))
}

func runSetupProviderLiveTest(cmd *cobra.Command) error {
	return gormescli.RunSetupProviderLiveTest(cmd.Context())
}

func runSetupRoot(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	if nonInteractive || !seams.IsTTY() {
		printSetupSections(cmd)
		return nil
	}

	options := setupTopLevelOptions()
	defaultOption := 0
	action, err := seams.ChooseSetupAction(cmd, options, defaultOption)
	if err != nil {
		return err
	}

	switch action {
	case setupActionQuick:
		return runSetupQuick(cmd, seams, nonInteractive, "")
	case setupActionFull:
		return seams.RunFullWizard(cmd, nonInteractive)
	case setupActionModelProvider:
		return runSetupModelSection(cmd, seams, nonInteractive)
	case setupActionFallback:
		return runSetupFallbackSection(cmd, seams, nonInteractive)
	case setupActionTerminal:
		return gormescli.RunSetupTerminalSection(cmd, nonInteractive, gormescli.SetupTerminalOptions{})
	case setupActionGateway:
		return seams.RunSetupGateway(cmd, nonInteractive || !seams.IsTTY())
	case setupActionTools:
		return seams.RunSetupTools(cmd, nonInteractive)
	case setupActionAgent:
		return runSetupAgentSettingsSection(cmd, nonInteractive)
	case setupActionMigrateHermes:
		return runSetupMigrate(cmd, "hermes")
	case setupActionMigrateOpenClaw:
		return runSetupMigrate(cmd, "openclaw")
	case setupActionExit:
		return nil
	default:
		return setupSectionUnsupported(cmd, string(action))
	}
}

func runSetupMigrate(cmd *cobra.Command, kind string) error {
	out := cmd.OutOrStdout()
	cli.PrintHeader(out, fmt.Sprintf("Migrate from %s", kind))
	fmt.Fprintln(out)
	source := ""
	switch kind {
	case "hermes":
		if home, err := os.UserHomeDir(); err == nil {
			candidate := filepath.Join(home, ".hermes")
			if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
				source = candidate
			}
		}
		if source == "" {
			source = os.Getenv("HERMES_HOME")
		}
	case "openclaw":
		if home, err := os.UserHomeDir(); err == nil {
			for _, dir := range []string{".openclaw", ".clawdbot", ".moltbot"} {
				candidate := filepath.Join(home, dir)
				if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
					source = candidate
					break
				}
			}
		}
	}
	if source == "" {
		fmt.Fprintf(out, "No %s source found. Pass --source or set %s_HOME.\n\n", kind, strings.ToUpper(kind))
		fmt.Fprintln(out, "Run this outside the setup menu:")
		fmt.Fprintf(out, "  gormes migrate %s --dry-run          # preview what would be migrated\n", kind)
		fmt.Fprintf(out, "  gormes migrate %s --yes              # apply migration\n", kind)
		fmt.Fprintf(out, "  gormes migrate %s --yes --source PATH  # specify source directory\n\n", kind)
		return nil
	}
	fmt.Fprintf(out, "Found %s at: %s\n", kind, source)
	fmt.Fprintln(out, "Migration is handled via the dedicated command:")
	fmt.Fprintf(out, "  gormes migrate %s --dry-run          # preview what would be migrated\n", kind)
	fmt.Fprintf(out, "  gormes migrate %s --yes --source %s  # apply migration\n\n", kind, source)
	fmt.Fprintln(out, "The migration command does not overwrite files without --overwrite.")
	fmt.Fprintln(out, "It reports conflicts and skipped secrets so you can review before applying.")
	return nil
}

func runSetupFirstTimeChoice(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	if nonInteractive || !seams.IsTTY() {
		printSetupSections(cmd)
		return nil
	}
	options := firstRunSetupOptions(seams)
	out := cmd.OutOrStdout()
	cli.ClearScreen(out)
	printSetupWizardHeader(cmd)
	fmt.Fprintln(out, cli.Dim(out, "  No existing Gormes configuration was found."))
	fmt.Fprintln(out)
	action, err := seams.ChooseSetupAction(cmd, options, 0)
	if err != nil {
		return err
	}
	switch action {
	case setupActionQuick:
		return runSetupQuick(cmd, seams, false, "")
	case setupActionFull:
		return seams.RunFullWizard(cmd, false)
	case setupActionMigrateHermes:
		return runSetupMigrate(cmd, "hermes")
	case setupActionMigrateOpenClaw:
		return runSetupMigrate(cmd, "openclaw")
	case setupActionExit:
		return nil
	default:
		return setupSectionUnsupported(cmd, string(action))
	}
}

// runSetupSection wraps `gormes setup <section>` with the same boxed chrome
// the full wizard and upstream `hermes setup <section>` use: a
// `│ Gormes Setup — <Label> │` header (the shared 59-wide box, reused from
// internal/doctor.RenderDoctorHeader — not a third chrome variant) before
// the section runs, and a uniform `<Label> configuration complete!` footer
// only on success for sections that do not render their own success receipt.
// An unknown section keeps the existing unsupported behavior with no box.
// Section prompts/logic are unchanged.
func runSetupSection(cmd *cobra.Command, seams setupCommandSeams, section string, nonInteractive bool) error {
	section = setupCanonicalSection(section)
	if !setupKnownSection(section) {
		return setupSectionUnsupported(cmd, section)
	}
	label := setupSectionLabel(section)
	out := cmd.OutOrStdout()
	cli.ClearScreen(out)
	fmt.Fprint(out, gormescli.RenderSetupSectionHeader(label))

	// Tee section output so the success-only footer can be suppressed when a
	// section cleanly cancels (returns nil but prints its own cancel line),
	// without changing any section's prompts or logic.
	var captured bytes.Buffer
	cmd.SetOut(io.MultiWriter(out, &captured))
	err := dispatchSetupSection(cmd, seams, section, nonInteractive)
	cmd.SetOut(out)

	if err == nil && !gormescli.SetupSectionSuppressSuccessFooter(section, captured.String()) {
		fmt.Fprintf(out, "\n%s configuration complete!\n", label)
	}
	return err
}

func dispatchSetupSection(cmd *cobra.Command, seams setupCommandSeams, section string, nonInteractive bool) error {
	switch section {
	case "provider":
		return runSetupProviderSection(cmd, seams, nonInteractive)
	case "model":
		return runSetupModelSection(cmd, seams, nonInteractive)
	case "fallback":
		return runSetupFallbackSection(cmd, seams, nonInteractive)
	case "router":
		return gormescli.RunSetupRouterSection(cmd)
	case "agent":
		if !nonInteractive && !seams.IsTTY() {
			return errSetupRequiresTTY
		}
		return runSetupAgentSettingsSection(cmd, nonInteractive)
	case "workspace":
		return runSetupAgentSection(cmd, section, seams, nonInteractive)
	case "profiles":
		return gormescli.RunSetupProfilesSection(cmd, gormescli.SetupProfilesOptions{NonInteractive: nonInteractive, IsTTY: seams.IsTTY, RequiresTTYError: errSetupRequiresTTY})
	case "bindings":
		return runSetupBindingsSection(cmd, seams, nonInteractive)
	case "tts":
		return gormescli.RunSetupTTSSection(cmd, nonInteractive, gormescli.SetupTTSOptions{})
	case "terminal":
		return gormescli.RunSetupTerminalSection(cmd, nonInteractive, gormescli.SetupTerminalOptions{})
	case "gateway":
		return seams.RunSetupGateway(cmd, nonInteractive || !seams.IsTTY())
	case "telegram":
		return runSetupTelegramSection(cmd, seams, nonInteractive)
	case "navivox":
		if nonInteractive || !seams.IsTTY() {
			return errSetupRequiresTTY
		}
		cfg, err := config.Load(nil)
		if err != nil {
			return err
		}
		return gormescli.RunSetupNavivoxGateway(cmd, cfg, setupNavivoxGatewayOptions(cmd))
	case "tools":
		return seams.RunSetupTools(cmd, nonInteractive || !seams.IsTTY())
	default:
		return setupSectionUnsupported(cmd, section)
	}
}

func setupTopLevelOptions() []setupMenuOption {
	return []setupMenuOption{
		{Action: setupActionQuick, Label: "Quick Setup - configure missing items only"},
		{Action: setupActionFull, Label: "Full Setup - reconfigure everything"},
		{Action: setupActionModelProvider, Label: "Model & Provider"},
		{Action: setupActionFallback, Label: "Fallback Providers"},
		{Action: setupActionTerminal, Label: "Terminal Backend"},
		{Action: setupActionGateway, Label: "Messaging Platforms (Gateway)"},
		{Action: setupActionTools, Label: "Tools"},
		{Action: setupActionAgent, Label: "Agent Settings"},
		{Action: setupActionMigrateHermes, Label: "Migrate from Hermes"},
		{Action: setupActionMigrateOpenClaw, Label: "Migrate from OpenClaw"},
		{Action: setupActionExit, Label: "Exit"},
	}
}

func printSetupTopLevelMenu(cmd *cobra.Command, options []setupMenuOption, defaultOption int) {
	printSetupActionMenu(cmd, setupActionPromptTitle(options), options, defaultOption)
}

func setupActionPromptTitle(options []setupMenuOption) string {
	if len(options) >= 2 && options[0].Action == setupActionQuick && options[1].Action == setupActionFull {
		onlyFirstRunActions := true
		for _, option := range options[2:] {
			switch option.Action {
			case setupActionMigrateHermes, setupActionMigrateOpenClaw:
			default:
				onlyFirstRunActions = false
			}
		}
		if onlyFirstRunActions {
			return "How would you like to set up Gormes?"
		}
	}
	return "What would you like to do?"
}

func printSetupActionMenu(cmd *cobra.Command, title string, options []setupMenuOption, defaultOption int) {
	out := cmd.OutOrStdout()
	cli.ClearScreen(out)
	cli.PrintHeader(out, title)
	fmt.Fprintln(out)
	for i, option := range options {
		var prefix, label string
		if i == defaultOption {
			prefix = " " + cli.BrightCyan(out, "→") + " " + cli.BrightCyan(out, "(●)")
			label = cli.Bold(out, option.Label)
		} else {
			prefix = "   " + cli.Dim(out, "(○)")
			label = option.Label
		}
		fmt.Fprintf(out, "%s %s\n", prefix, label)
	}
	fmt.Fprintln(out)
}

func promptSetupAction(cmd *cobra.Command, options []setupMenuOption, defaultOption int) (setupAction, error) {
	if stdin, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(stdin.Fd())) {
		stepOptions := []setupwizard.StepOption{}
		if setupActionPromptTitle(options) == "How would you like to set up Gormes?" {
			stepOptions = append(stepOptions, setupwizard.WithRadioChoices())
		}
		selected, err := runBubbleTeaPickWithOptions(cmd.Context(), stdin, cmd.OutOrStdout(), setupActionPromptTitle(options), setupActionPickerChoices(options), string(options[defaultOption].Action), stepOptions...)
		if err == nil {
			if selected == "" {
				return setupActionExit, nil
			}
			for _, o := range options {
				if string(o.Action) == selected {
					return o.Action, nil
				}
			}
		} else if !bubbleTeaPickShouldFallback(err) {
			return "", err
		}
	}
	// Fallback to line-buffered input (for tests, CI, piped stdin).
	return promptSetupActionText(cmd, options, defaultOption)
}

func setupActionPickerChoices(options []setupMenuOption) []tuiPickChoice {
	choices := make([]tuiPickChoice, len(options))
	for i, option := range options {
		choices[i] = tuiPickChoice{ID: string(option.Action), Label: option.Label}
	}
	return choices
}

// promptSetupActionText is the fallback line-input version used for piped
// stdin, CI, and other non-interactive command runners.
func promptSetupActionText(cmd *cobra.Command, options []setupMenuOption, defaultOption int) (setupAction, error) {
	printSetupTopLevelMenu(cmd, options, defaultOption)
	defaultText := strconv.Itoa(defaultOption + 1)
	answer, err := promptString(cmd, fmt.Sprintf("Select option [%s]: ", defaultText), defaultText)
	if err != nil {
		return "", err
	}
	answer = strings.ToLower(strings.TrimSpace(cli.StripANSI(answer)))
	if answer == "" {
		return options[defaultOption].Action, nil
	}
	if answer == "q" || answer == "quit" || answer == "exit" {
		return setupActionExit, nil
	}
	if n, err := strconv.Atoi(answer); err == nil && n >= 1 && n <= len(options) {
		return options[n-1].Action, nil
	}
	for _, option := range options {
		if answer == string(option.Action) || strings.Contains(normalizeSetupChoice(option.Label), answer) {
			return option.Action, nil
		}
	}
	return "", newExitCodeError(2, fmt.Errorf("setup_menu_invalid_selection: %s", answer))
}

func stripSetupInputNoise(answer string) string {
	return gormescli.StripSetupInputNoise(answer)
}

func runSetupFullWizard(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	printSetupWizardHeader(cmd)
	if nonInteractive {
		if err := runSetupModelSection(cmd, seams, true); err != nil {
			return err
		}
		if err := gormescli.RunSetupTTSSection(cmd, true, gormescli.SetupTTSOptions{}); err != nil {
			return err
		}
		if err := gormescli.RunSetupTerminalSection(cmd, true, gormescli.SetupTerminalOptions{}); err != nil {
			return err
		}
		if err := runSetupAgentSettingsSection(cmd, true); err != nil {
			return err
		}
		if err := seams.RunSetupGateway(cmd, true); err != nil {
			return err
		}
		if err := seams.RunSetupTools(cmd, true); err != nil {
			return err
		}
		printSetupSummary(cmd)
		return nil
	}

	continued, err := runSetupInferenceProviderSection(cmd, seams)
	if err != nil {
		return err
	}
	if !continued {
		return nil
	}
	if err := gormescli.RunSetupTTSSection(cmd, false, gormescli.SetupTTSOptions{}); err != nil {
		return err
	}
	if err := gormescli.RunSetupTerminalSection(cmd, false, gormescli.SetupTerminalOptions{}); err != nil {
		return err
	}
	if err := runSetupAgentSettingsSection(cmd, false); err != nil {
		return err
	}
	if err := seams.RunSetupGateway(cmd, false); err != nil {
		return err
	}
	if err := seams.RunSetupTools(cmd, false); err != nil {
		return err
	}
	printSetupSummary(cmd)
	return offerSetupLaunchChat(cmd, seams)
}

func printSetupWizardHeader(cmd *cobra.Command) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "┌─────────────────────────────────────────────────────────┐")
	fmt.Fprintln(out, "│              Gormes Agent Setup Wizard                  │")
	fmt.Fprintln(out, "├─────────────────────────────────────────────────────────┤")
	fmt.Fprintln(out, "│  Configure your Gormes Agent installation.              │")
	fmt.Fprintln(out, "│  Press Ctrl+C at any time to exit.                      │")
	fmt.Fprintln(out, "└─────────────────────────────────────────────────────────┘")
	fmt.Fprintln(out)
}

func runSetupInferenceProviderSection(cmd *cobra.Command, seams setupCommandSeams) (bool, error) {
	if !seams.IsTTY() {
		fmt.Fprintln(cmd.ErrOrStderr(), "setup_requires_tty: run `gormes setup model --non-interactive` to use defaults without prompts")
		return false, errSetupRequiresTTY
	}

	current, err := seams.LoadCurrentModel()
	if err != nil {
		return false, fmt.Errorf("setup model: load current model: %w", err)
	}
	provider := strings.TrimSpace(current.Provider)
	label := setupProviderDisplayLabel(provider)

	printSetupReconfigureBlock(cmd)
	printSetupConfigurationLocation(cmd)
	printSetupInferenceProviderBlock(cmd, current, label)

	entries, defaultIndex := cli.HermesProviderCatalogMenu(provider)
	idx, err := seams.ChooseSetupProvider(cmd, entries, defaultIndex)
	if err != nil {
		if errors.Is(err, cli.ErrModelPickerCancelled) {
			fmt.Fprintln(cmd.OutOrStdout(), "No change.")
			return true, nil
		}
		return false, err
	}
	if idx < 0 || idx >= len(entries) || entries[idx].ID == cli.ProviderCatalogLeaveUnchanged {
		fmt.Fprintln(cmd.OutOrStdout(), "No change.")
		return true, nil
	}
	selectedProvider := strings.TrimSpace(entries[idx].ID)
	if selectedProvider == cli.ProviderCatalogAuxConfig {
		fmt.Fprintln(cmd.OutOrStdout(), "Auxiliary model setup is not available in the Go wizard yet.")
		return true, nil
	}
	if selectedProvider == "custom-endpoint" {
		selectedProvider = "custom"
	}
	if err := runSetupSelectedProviderFlow(cmd, seams, current, selectedProvider); err != nil {
		if errors.Is(err, cli.ErrModelPickerCancelled) {
			fmt.Fprintln(cmd.OutOrStdout(), "No change.")
			return true, nil
		}
		return false, err
	}
	return true, nil
}

func printSetupReconfigureBlock(cmd *cobra.Command) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "◆ Reconfigure")
	fmt.Fprintln(out, "✓ You already have Gormes configured.")
	fmt.Fprintln(out, "  Running the full wizard - each prompt shows your current value.")
	fmt.Fprintln(out, "  Press Enter to keep it, or type a new value to change it.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  Tip: jump straight to any focused setup section:")
	fmt.Fprintf(out, "       gormes setup %s\n", setupSectionPipeList(0, 6))
	fmt.Fprintf(out, "                    %s\n", setupSectionPipeList(6, len(setupSections)))
	fmt.Fprintln(out, "       Or fill only missing items with: gormes setup --quick")
	fmt.Fprintln(out)
}

func setupSectionPipeList(start, end int) string {
	return gormescli.SetupSectionPipeList(start, end)
}

func printSetupConfigurationLocation(cmd *cobra.Command) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "◆ Configuration Location")
	fmt.Fprintf(out, "  Config file:  %s\n", config.ConfigPath())
	fmt.Fprintf(out, "  Secrets file: %s\n", config.EnvPath())
	fmt.Fprintf(out, "  Data folder:  %s\n", config.GormesHome())
	fmt.Fprintf(out, "  Install dir:  %s\n", setupManagedInstallDir())
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  You can edit these files directly or use 'gormes config edit'")
	fmt.Fprintln(out)
}

func printSetupInferenceProviderBlock(cmd *cobra.Command, current cli.ProviderModel, providerLabel string) {
	out := cmd.OutOrStdout()
	model := strings.TrimSpace(current.Model)
	if model == "" {
		model = "(not configured)"
	}
	if providerLabel == "" {
		providerLabel = "(not configured)"
	}
	fmt.Fprintln(out, "◆ Inference Provider")
	fmt.Fprintln(out, "  Choose how to connect to your main chat model.")
	fmt.Fprintln(out, "     Guide: https://hermes-agent.nousresearch.com/docs/integrations/providers")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  Current model:    %s\n", model)
	fmt.Fprintf(out, "  Active provider:  %s\n", providerLabel)
	fmt.Fprintln(out)
}

func printSetupProviderCredentialChoices(cmd *cobra.Command) {
	gormescli.SetupProviderCredentialChoices(cmd)
}

func promptSetupProviderChoice(cmd *cobra.Command, entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
	if len(entries) == 0 {
		return -1, cli.ErrModelPickerNoProviders
	}
	if defaultIndex < 0 || defaultIndex >= len(entries) {
		defaultIndex = len(entries) - 1
	}

	if stdin, ok := cmd.InOrStdin().(*os.File); ok && setupInputIsTerminal(stdin) {
		selected, err := runBubbleTeaPickWithOptions(
			cmd.Context(),
			stdin,
			cmd.OutOrStdout(),
			"Select provider",
			gormescli.ProviderPickerChoices(entries),
			strconv.Itoa(defaultIndex),
			setupwizard.WithSearchChoices(),
		)
		if err == nil {
			if selected == "" {
				return -1, cli.ErrModelPickerCancelled
			}
			index, parseErr := strconv.Atoi(selected)
			if parseErr != nil || index < 0 || index >= len(entries) {
				return -1, newExitCodeError(2, fmt.Errorf("setup_provider_invalid_selection: %s", selected))
			}
			return index, nil
		}
		if !bubbleTeaPickShouldFallback(err) {
			return -1, err
		}
	}

	return promptSetupProviderChoiceText(cmd, entries, defaultIndex)
}

// promptSetupProviderChoiceText is the fallback line-input version used for
// piped stdin, CI, and command runners that do not expose a terminal.
func promptSetupProviderChoiceText(cmd *cobra.Command, entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
	out := cmd.OutOrStdout()
	if len(entries) == 0 {
		return -1, cli.ErrModelPickerNoProviders
	}
	if defaultIndex < 0 || defaultIndex >= len(entries) {
		defaultIndex = len(entries) - 1
	}

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)

	fmt.Fprintln(out, "Select provider:")
	for i, entry := range entries {
		marker := " "
		if i == defaultIndex {
			marker = "→"
		}
		fmt.Fprintf(out, "  %s %d. %s\n", marker, i+1, entry.Label)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Choice [1-%d] (%d): ", len(entries), defaultIndex+1)

	answer, err := scanPromptString(cmd, strconv.Itoa(defaultIndex+1))
	select {
	case <-interrupts:
		fmt.Fprintln(out)
		return -1, cli.ErrModelPickerCancelled
	default:
	}
	if err != nil {
		fmt.Fprintln(out)
		return -1, cli.ErrModelPickerCancelled
	}
	answer = strings.TrimSpace(stripSetupInputNoise(cli.StripANSI(answer)))
	if answer == "" {
		return defaultIndex, nil
	}
	if strings.EqualFold(answer, "q") || strings.EqualFold(answer, "cancel") {
		return -1, cli.ErrModelPickerCancelled
	}
	idx, err := strconv.Atoi(answer)
	if err != nil || idx < 1 || idx > len(entries) {
		return -1, newExitCodeError(2, fmt.Errorf("setup_provider_invalid_selection: %s", answer))
	}
	return idx - 1, nil
}

func runSetupSelectedProviderFlow(cmd *cobra.Command, seams setupCommandSeams, current cli.ProviderModel, provider string) error {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return cli.ErrSelectorNoMatch
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Selected provider: %s\n", setupProviderDisplayLabel(provider))
	if providermodule.AuthProviderDefaultsToOAuth(provider) {
		return runSetupOAuthProviderFlow(cmd, seams, current, provider)
	}
	if err := seams.RunActiveProviderModelPicker(cmd, cli.ProviderModel{Provider: provider, Model: setupModelSeedForProvider(current, provider)}); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Provider auth was not changed. If credentials are missing, run: gormes auth add %s\n", provider)
	return nil
}

func setupModelSeedForProvider(current cli.ProviderModel, provider string) string {
	provider = strings.TrimSpace(provider)
	if strings.EqualFold(strings.TrimSpace(current.Provider), provider) && strings.TrimSpace(current.Model) != "" {
		return strings.TrimSpace(current.Model)
	}
	if resolved := llm.ResolveProviderDefaultModel(provider, llm.ProviderDefaultModelOptions{}); strings.TrimSpace(resolved.Model) != "" {
		return strings.TrimSpace(resolved.Model)
	}
	return strings.TrimSpace(current.Model)
}

func runSetupOpenAICodexProviderFlow(cmd *cobra.Command, seams setupCommandSeams, current cli.ProviderModel) error {
	return runSetupOAuthProviderFlow(cmd, seams, current, config.CodexOAuthProvider)
}

func runSetupOAuthProviderFlow(cmd *cobra.Command, seams setupCommandSeams, current cli.ProviderModel, provider string) error {
	status, statusErr := seams.LoadProviderAuthStatus(provider)
	label := setupProviderDisplayLabel(provider)
	fmt.Fprintf(cmd.OutOrStdout(), "  %s credentials: %s\n", label, setupCredentialStatusMark(status, statusErr))
	if status.Authenticated || status.Status == cli.AuthStatusLoggedIn {
		prompt := setupProviderCredentialPrompt{Provider: provider, ProviderLabel: label, Status: status}
		printSetupProviderCredentialChoices(cmd)
		action, err := seams.ChooseProviderCredentialAction(cmd, prompt)
		if err != nil {
			return err
		}
		switch action {
		case setupProviderCredentialUseExisting:
		case setupProviderCredentialReauthenticate:
			fmt.Fprintf(cmd.OutOrStdout(), "Starting a fresh %s login...\n\n", label)
			if err := seams.RunProviderAuth(cmd, provider); err != nil {
				return err
			}
		case setupProviderCredentialCancel:
			return cli.ErrModelPickerCancelled
		default:
			return newExitCodeError(2, fmt.Errorf("setup_provider_credentials_invalid_selection: %s", action))
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Not logged into %s. Starting login...\n", label)
		fmt.Fprintln(cmd.OutOrStdout())
		if err := seams.RunProviderAuth(cmd, provider); err != nil {
			return err
		}
	}
	return seams.RunActiveProviderModelPicker(cmd, cli.ProviderModel{Provider: provider, Model: setupModelSeedForProvider(current, provider)})
}

func promptSetupProviderCredentialAction(cmd *cobra.Command, _ setupProviderCredentialPrompt) (setupProviderCredentialAction, error) {
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)

	answer, err := promptSetupOptionChoice(cmd, "Provider credentials", "  Choice [1/2/3]: ", string(setupProviderCredentialUseExisting), []setupOptionChoice{
		{ID: string(setupProviderCredentialUseExisting), Label: "Use existing credentials", Aliases: []string{"1", "use", "use-existing", "existing"}},
		{ID: string(setupProviderCredentialReauthenticate), Label: "Reauthenticate (new OAuth login)", Aliases: []string{"2", "reauth", "reauthenticate", "login"}},
		{ID: string(setupProviderCredentialCancel), Label: "Cancel", Aliases: []string{"3", "cancel", "q", "quit", "exit"}},
	})
	select {
	case <-interrupts:
		fmt.Fprintln(cmd.OutOrStdout())
		return setupProviderCredentialUseExisting, nil
	default:
	}
	if err != nil {
		return setupProviderCredentialUseExisting, nil
	}
	switch setupProviderCredentialAction(answer) {
	case setupProviderCredentialUseExisting:
		return setupProviderCredentialUseExisting, nil
	case setupProviderCredentialReauthenticate:
		return setupProviderCredentialReauthenticate, nil
	case setupProviderCredentialCancel:
		return setupProviderCredentialCancel, nil
	default:
		return "", newExitCodeError(2, fmt.Errorf("setup_provider_credentials_invalid_selection: %s", answer))
	}
}

func runSetupProviderAuth(cmd *cobra.Command, provider string) error {
	authCmd := providermodule.NewAuthCommand(providerCommandOptions())
	authCmd.SetOut(cmd.OutOrStdout())
	authCmd.SetErr(cmd.ErrOrStderr())
	authCmd.SetIn(cmd.InOrStdin())
	authCmd.SetArgs([]string{"add", provider, "--type", "oauth"})
	authCmd.SilenceUsage = true
	authCmd.SilenceErrors = true
	return authCmd.ExecuteContext(cmd.Context())
}

func setupManagedInstallDir() string {
	if dir, err := gormescli.ResolveManagedCheckoutDir(); err == nil && strings.TrimSpace(dir) != "" {
		return dir
	}
	return filepath.Join(config.GormesHome(), "gormes-agent")
}

func setupCredentialStatusMark(status cli.ProviderAuthStatus, err error) string {
	if err != nil || status.Status == cli.AuthStatusError {
		return "!"
	}
	if status.Authenticated || status.Status == cli.AuthStatusLoggedIn {
		return "✓"
	}
	return "missing"
}

func setupProviderDisplayLabel(provider string) string {
	return gormescli.SetupProviderDisplayLabel(provider)
}

func setupTitleProviderWord(word string) string {
	return gormescli.SetupTitleProviderWord(word)
}

func printSetupSummary(cmd *cobra.Command) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "┌─────────────────────────────────────────────────────────┐")
	fmt.Fprintln(out, "│              ✓ Setup Complete!                          │")
	fmt.Fprintln(out, "└─────────────────────────────────────────────────────────┘")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "📁 All your files are in %s/:\n", config.GormesHome())
	fmt.Fprintln(out)
	fmt.Fprintf(out, "   Settings:  %s\n", config.ConfigPath())
	fmt.Fprintf(out, "   API Keys:  %s\n", config.EnvPath())
	fmt.Fprintf(out, "   Data:      %s/cron/, sessions/, logs/\n", config.GormesHome())
	fmt.Fprintln(out)
	fmt.Fprintln(out, "────────────────────────────────────────────────────────────")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "📝 To edit your configuration:")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "   gormes setup          Re-run the full wizard")
	fmt.Fprintln(out, "   gormes setup model    Change model/provider")
	fmt.Fprintln(out, "   gormes setup fallback Add fallback providers")
	fmt.Fprintln(out, "   gormes setup terminal Change terminal backend")
	fmt.Fprintln(out, "   gormes setup gateway  Configure messaging")
	fmt.Fprintln(out, "   gormes setup telegram Configure Telegram")
	fmt.Fprintln(out, "   gormes setup tools    Configure tool providers")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "   gormes config         View current settings")
	fmt.Fprintln(out, "   gormes config edit    Open config in your editor")
	fmt.Fprintln(out, "   gormes config set <key> <value>")
	fmt.Fprintln(out, "                          Set a specific value")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "   Or edit the files directly:")
	fmt.Fprintf(out, "   nano %s\n", config.ConfigPath())
	fmt.Fprintf(out, "   nano %s\n", config.EnvPath())
	fmt.Fprintln(out)
	fmt.Fprintln(out, "────────────────────────────────────────────────────────────")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "🚀 Ready to go!")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "   gormes              Start chatting")
	fmt.Fprintln(out, "   gormes gateway      Start messaging gateway")
	fmt.Fprintln(out, "   gormes doctor       Check for issues")
	fmt.Fprintln(out)
}

func offerSetupLaunchChat(cmd *cobra.Command, seams setupCommandSeams) error {
	launch, ok, err := promptSetupYesNoOption(cmd, "Launch gormes chat now?", "Launch gormes chat now? [Y/n]: ", true)
	if err != nil {
		return err
	}
	if !ok {
		return newExitCodeError(2, fmt.Errorf("setup_launch_invalid_selection"))
	}
	if launch {
		return seams.LaunchChat(cmd)
	}
	return nil
}

func launchSetupChat(cmd *cobra.Command) error {
	root := newRootCommand()
	root.SetArgs([]string{})
	root.SetIn(cmd.InOrStdin())
	root.SetOut(cmd.OutOrStdout())
	root.SetErr(cmd.ErrOrStderr())
	root.SilenceUsage = true
	root.SilenceErrors = true
	return root.ExecuteContext(cmd.Context())
}

func runSetupProviderSection(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	if nonInteractive {
		return setupProviderNonInteractive(cmd)
	}
	if !seams.IsTTY() {
		fmt.Fprintln(cmd.ErrOrStderr(), setupProviderRequiresTTYGuidance(setupProviderRequestedOrAutoDetectedID()))
		return errSetupRequiresTTY
	}
	return setupProviderInteractive(cmd, seams)
}

func setupProviderRequiresTTYGuidance(provider string) string {
	provider = setupCanonicalProviderID(provider)
	apiKey, apiKeyEnvNames := setupProviderAPIKeyDefault(provider)
	if setupProviderShouldUseOAuth(provider, apiKey) {
		label := setupProviderReceiptLabel(provider, provider)
		return fmt.Sprintf("setup_requires_tty: %s uses OAuth; run `gormes setup provider --non-interactive` to write provider defaults, then `gormes auth add %s --type oauth` to sign in", label, provider)
	}
	endpointEnvNames := setupProviderEndpointEnvNames(provider)
	return fmt.Sprintf("setup_requires_tty: run `gormes setup provider --non-interactive` to use %s plus %s env vars", setupEnvOptions(endpointEnvNames, "or"), setupEnvOptions(apiKeyEnvNames, "or"))
}

func setupProviderNonInteractive(cmd *cobra.Command) error {
	provider := setupProviderRequestedOrAutoDetectedID()
	model := firstNonEmptySetup(os.Getenv("GORMES_MODEL"), os.Getenv("GORMES_INFERENCE_MODEL"))
	endpoint := cleanSetupProviderEndpoint(os.Getenv("GORMES_ENDPOINT"))
	if endpoint == "" && provider != "" {
		endpoint = setupProviderEndpointDefault(provider)
	}
	if model == "" && provider != "" {
		model = setupProviderModelDefault(cli.ProviderModel{}, provider)
	}

	apiKey, apiKeyEnvNames := setupProviderAPIKeyDefault(provider)
	if setupProviderShouldUseOAuth(provider, apiKey) {
		if endpoint == "" {
			return fmt.Errorf("setup provider --non-interactive: GORMES_ENDPOINT must be set for %s", provider)
		}
		return writeOAuthProviderConfig(cmd, provider, endpoint, model)
	}

	endpointEnvNames := setupProviderEndpointEnvNames(provider)
	if endpoint == "" && apiKey == "" {
		return fmt.Errorf("setup provider --non-interactive: endpoint must be set via %s; API key must be set via %s", setupEnvOptions(endpointEnvNames, "or"), setupEnvOptions(apiKeyEnvNames, "or"))
	}
	if endpoint == "" {
		return fmt.Errorf("setup provider --non-interactive: endpoint must be set via %s", setupEnvOptions(endpointEnvNames, "or"))
	}
	if apiKey == "" {
		return fmt.Errorf("setup provider --non-interactive: API key must be set via %s", setupEnvOptions(apiKeyEnvNames, "or"))
	}
	return writeProviderConfig(cmd, provider, endpoint, apiKey, model)
}

func setupProviderRequestedOrAutoDetectedID() string {
	provider := setupCanonicalProviderID(os.Getenv("GORMES_INFERENCE_PROVIDER"))
	if provider != "" {
		return provider
	}
	if strings.TrimSpace(os.Getenv("GORMES_API_KEY")) != "" {
		return ""
	}
	return setupProviderAutoDetectEnvProvider()
}

func setupProviderAutoDetectEnvProvider() string {
	// Match Hermes' no-provider fallback first: generic OpenRouter-compatible
	// OPENROUTER_API_KEY / OPENAI_API_KEY values route through OpenRouter.
	if strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")) != "" || strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) != "" {
		return "openrouter"
	}
	for _, entry := range llm.HermesProviderRegistryManifest() {
		if entry.ID == "github-copilot" {
			// GH_TOKEN/GITHUB_TOKEN are often present for git tooling and Hermes
			// intentionally avoids treating them as provider setup by default.
			continue
		}
		for _, envName := range entry.EnvVars {
			if setupProviderImplicitAPIKeyEnv(envName) {
				// Set by tools such as Claude Code themselves; Hermes does not treat
				// them as an operator provider setup signal.
				continue
			}
			if strings.TrimSpace(os.Getenv(envName)) != "" {
				return setupCanonicalProviderID(entry.ID)
			}
		}
	}
	return ""
}

func setupProviderShouldUseOAuth(provider, apiKey string) bool {
	provider = setupCanonicalProviderID(provider)
	if !providermodule.AuthProviderDefaultsToOAuth(provider) {
		return false
	}
	if strings.TrimSpace(apiKey) != "" && setupProviderSupportsAPIKey(provider) {
		return false
	}
	return true
}

func setupProviderSupportsAPIKey(provider string) bool {
	provider = setupCanonicalProviderID(provider)
	if entry, ok := llm.ResolveProviderManifestEntry(provider); ok {
		return strings.EqualFold(strings.TrimSpace(entry.AuthType), "api_key")
	}
	return !providermodule.AuthProviderDefaultsToOAuth(provider)
}

func setupProviderImplicitAPIKeyEnv(envName string) bool {
	return strings.EqualFold(strings.TrimSpace(envName), "CLAUDE_CODE_OAUTH_TOKEN")
}

func setupProviderAPIKeyDefault(provider string) (string, []string) {
	envNames := setupProviderAPIKeyEnvNames(provider)
	for _, envName := range envNames {
		if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
			return value, envNames
		}
	}
	return "", envNames
}

func setupProviderAPIKeyEnvNames(provider string) []string {
	envNames := make([]string, 0, 4)
	addSetupEnvName(&envNames, "GORMES_API_KEY")
	if entry, ok := llm.ResolveProviderManifestEntry(provider); ok {
		for _, envName := range entry.EnvVars {
			if setupProviderImplicitAPIKeyEnv(envName) {
				continue
			}
			addSetupEnvName(&envNames, envName)
		}
	}
	return envNames
}

func setupProviderEndpointEnvNames(provider string) []string {
	envNames := make([]string, 0, 2)
	addSetupEnvName(&envNames, "GORMES_ENDPOINT")
	if entry, ok := llm.ResolveProviderManifestEntry(provider); ok {
		addSetupEnvName(&envNames, entry.BaseURLEnvVar)
	}
	return envNames
}

func addSetupEnvName(envNames *[]string, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	for _, existing := range *envNames {
		if existing == name {
			return
		}
	}
	*envNames = append(*envNames, name)
}

func setupEnvOptions(names []string, conjunction string) string {
	cleaned := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			cleaned = append(cleaned, name)
		}
	}
	if len(cleaned) == 0 {
		return "the required environment variable"
	}
	if len(cleaned) == 1 {
		return cleaned[0]
	}
	conjunction = strings.TrimSpace(conjunction)
	if conjunction == "" {
		conjunction = "or"
	}
	if len(cleaned) == 2 {
		return cleaned[0] + " " + conjunction + " " + cleaned[1]
	}
	return strings.Join(cleaned[:len(cleaned)-1], ", ") + ", " + conjunction + " " + cleaned[len(cleaned)-1]
}

func setupProviderInteractive(cmd *cobra.Command, seams setupCommandSeams) error {
	seams = setupProviderInteractiveSeams(seams)
	out := cmd.OutOrStdout()
	current, _ := loadSetupProviderCurrent()
	entries, defaultIndex := cli.HermesProviderCatalogMenu(current.Provider)
	idx, err := seams.ChooseSetupProvider(cmd, entries, defaultIndex)
	if err != nil {
		if errors.Is(err, cli.ErrModelPickerCancelled) {
			fmt.Fprintln(out, "No change.")
			return nil
		}
		return err
	}
	if idx < 0 || idx >= len(entries) || entries[idx].ID == cli.ProviderCatalogLeaveUnchanged {
		fmt.Fprintln(out, "No change.")
		return nil
	}
	if entries[idx].ID == cli.ProviderCatalogAuxConfig {
		fmt.Fprintln(out, "Auxiliary model setup is not available in the Go wizard yet.")
		return nil
	}

	provider := setupCanonicalProviderID(entries[idx].ID)
	if providermodule.AuthProviderDefaultsToOAuth(provider) {
		return runSetupSelectedProviderFlow(cmd, seams, current, provider)
	}
	var endpoint string
	if provider == "custom" {
		endpoint, err = promptString(cmd, "Endpoint URL: ", "")
		if err != nil {
			return err
		}
	} else {
		endpoint = setupProviderEndpointDefault(provider)
	}
	if endpoint == "" {
		endpoint, err = promptString(cmd, "Endpoint URL: ", "")
		if err != nil {
			return err
		}
		if endpoint == "" {
			return fmt.Errorf("endpoint URL is required for custom provider")
		}
	}

	apiKey, err := promptSecret(cmd, "API key: ")
	if err != nil {
		return err
	}
	if apiKey == "" {
		return fmt.Errorf("API key is required; get one from your provider's dashboard")
	}

	defaultModel := setupProviderModelDefault(current, provider)
	model, err := gormescli.PromptModelChoiceWithOptions(cmd.InOrStdin(), out, provider, defaultModel, gormescli.DefaultModelPickerSuggestionSet(provider).Models, gormescli.ModelChoicePromptOptions{
		Context:         cmd.Context(),
		SuggestionLimit: gormescli.ModelChoiceSuggestionLimitUnlimited,
	})
	if err != nil {
		if errors.Is(err, cli.ErrModelPickerCancelled) {
			fmt.Fprintln(out, "No change.")
			return nil
		}
		return err
	}
	model = llm.NormalizeProviderModelID(provider, model)

	return writeProviderConfig(cmd, provider, endpoint, apiKey, model)
}

func setupProviderInteractiveSeams(seams setupCommandSeams) setupCommandSeams {
	if seams.ChooseSetupProvider == nil {
		seams.ChooseSetupProvider = promptSetupProviderChoice
	}
	if seams.LoadProviderAuthStatus == nil {
		seams.LoadProviderAuthStatus = func(provider string) (cli.ProviderAuthStatus, error) {
			return cli.ResolveAuthStatus(context.Background(), provider, cli.AuthStatusOptions{})
		}
	}
	if seams.RunProviderAuth == nil {
		seams.RunProviderAuth = runSetupProviderAuth
	}
	if seams.RunActiveProviderModelPicker == nil {
		seams.RunActiveProviderModelPicker = gormescli.RunSetupActiveProviderModelPicker
	}
	return seams
}

func loadSetupProviderCurrent() (cli.ProviderModel, error) {
	cfg, err := config.Load(nil)
	if err != nil {
		return cli.ProviderModel{}, err
	}
	return cli.ProviderModel{Provider: cfg.Hermes.Provider, Model: cfg.Hermes.Model}, nil
}

func setupCanonicalProviderID(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "custom-endpoint" {
		return "custom"
	}
	if entry, ok := llm.ResolveProviderManifestEntry(provider); ok {
		return strings.TrimSpace(entry.ID)
	}
	return provider
}

func setupProviderEndpointDefault(provider string) string {
	provider = setupCanonicalProviderID(provider)
	if endpoint := setupProviderEndpointEnvDefault(provider); endpoint != "" {
		return endpoint
	}
	if endpoint := providermodule.ProviderBaseURL(provider, ""); strings.TrimSpace(endpoint) != "" {
		return cleanSetupProviderEndpoint(endpoint)
	}
	if endpoint := knownProviderEndpoints[provider]; strings.TrimSpace(endpoint) != "" {
		return cleanSetupProviderEndpoint(endpoint)
	}
	if entry, ok := llm.ResolveProviderManifestEntry(provider); ok {
		if endpoint := setupProviderEndpointEnvDefault(entry.ID); endpoint != "" {
			return endpoint
		}
		if endpoint := providermodule.ProviderBaseURL(entry.ID, ""); strings.TrimSpace(endpoint) != "" {
			return cleanSetupProviderEndpoint(endpoint)
		}
		if endpoint := knownProviderEndpoints[entry.ID]; strings.TrimSpace(endpoint) != "" {
			return cleanSetupProviderEndpoint(endpoint)
		}
		if endpoint := strings.TrimSpace(entry.BaseURLOverride); endpoint != "" {
			return cleanSetupProviderEndpoint(endpoint)
		}
	}
	return ""
}

func setupProviderEndpointEnvDefault(provider string) string {
	if entry, ok := llm.ResolveProviderManifestEntry(provider); ok {
		if endpoint := cleanSetupProviderEndpoint(os.Getenv(entry.BaseURLEnvVar)); endpoint != "" {
			return endpoint
		}
	}
	return ""
}

func cleanSetupProviderEndpoint(endpoint string) string {
	return strings.TrimRight(strings.TrimSpace(endpoint), "/")
}

func setupProviderModelDefault(current cli.ProviderModel, provider string) string {
	provider = setupCanonicalProviderID(provider)
	if strings.EqualFold(setupCanonicalProviderID(current.Provider), provider) && strings.TrimSpace(current.Model) != "" {
		return strings.TrimSpace(current.Model)
	}
	if resolved := llm.ResolveProviderDefaultModel(provider, llm.ProviderDefaultModelOptions{}); strings.TrimSpace(resolved.Model) != "" {
		return strings.TrimSpace(resolved.Model)
	}
	if model := knownProviderModels[provider]; strings.TrimSpace(model) != "" {
		return strings.TrimSpace(model)
	}
	return ""
}

type setupProviderReceipt struct {
	Title         string
	ProviderLabel string
	Endpoint      string
	Model         string
	ConfigPath    string
	SecretsPath   string
	AuthMethod    string
	AuthEvidence  string
	NextSteps     []string
}

func writeOAuthProviderConfig(cmd *cobra.Command, provider, endpoint, model string) error {
	configPath := config.ConfigPath()

	if provider != "" {
		if err := config.WriteTOMLValue(configPath, "hermes.provider", provider); err != nil {
			return fmt.Errorf("write provider: %w", err)
		}
	}
	if err := config.WriteTOMLValue(configPath, "hermes.endpoint", endpoint); err != nil {
		return fmt.Errorf("write endpoint: %w", err)
	}
	if model != "" {
		if err := config.WriteTOMLValue(configPath, "hermes.model", model); err != nil {
			return fmt.Errorf("write model: %w", err)
		}
	}

	label := setupProviderReceiptLabel(provider, provider)
	writeSetupProviderReceipt(cmd, setupProviderReceipt{
		Title:         fmt.Sprintf("%s configured.", label),
		ProviderLabel: label,
		Endpoint:      endpoint,
		Model:         model,
		ConfigPath:    configPath,
		AuthMethod:    "OAuth credential pool",
		AuthEvidence:  "API key: not required or stored",
		NextSteps: []string{
			fmt.Sprintf("Sign in: gormes auth add %s --type oauth", provider),
			fmt.Sprintf("Verify:  gormes auth status %s", provider),
			"Check:   gormes doctor --offline",
		},
	})
	return nil
}

func writeProviderConfig(cmd *cobra.Command, provider, endpoint, apiKey, model string) error {
	configPath := config.ConfigPath()

	if provider != "" {
		if err := config.WriteTOMLValue(configPath, "hermes.provider", provider); err != nil {
			return fmt.Errorf("write provider: %w", err)
		}
	}

	if err := config.WriteTOMLValue(configPath, "hermes.endpoint", endpoint); err != nil {
		return fmt.Errorf("write endpoint: %w", err)
	}

	envPath := config.EnvPath()
	if err := config.WriteEnvValue(envPath, "GORMES_API_KEY", apiKey); err != nil {
		return fmt.Errorf("write API key: %w", err)
	}

	if model != "" {
		if err := config.WriteTOMLValue(configPath, "hermes.model", model); err != nil {
			return fmt.Errorf("write model: %w", err)
		}
	}

	writeSetupProviderReceipt(cmd, setupProviderReceipt{
		Title:         "Provider configured.",
		ProviderLabel: setupProviderReceiptLabel(provider, "custom endpoint"),
		Endpoint:      endpoint,
		Model:         model,
		ConfigPath:    configPath,
		SecretsPath:   envPath,
		AuthMethod:    "API key",
		AuthEvidence:  "API key:  stored (redacted)",
		NextSteps: []string{
			"Verify: gormes config check",
			"Check:  gormes doctor --offline",
			"Test it:  gormes chat",
		},
	})
	return nil
}

func setupProviderReceiptLabel(provider, fallback string) string {
	if label := setupProviderDisplayLabel(provider); label != "" {
		return label
	}
	if fallback = strings.TrimSpace(fallback); fallback != "" {
		return fallback
	}
	return "provider"
}

func writeSetupProviderReceipt(cmd *cobra.Command, receipt setupProviderReceipt) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	if strings.TrimSpace(receipt.Title) != "" {
		fmt.Fprintln(out, receipt.Title)
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out, "Connection")
	if strings.TrimSpace(receipt.ProviderLabel) != "" {
		fmt.Fprintf(out, "  Provider: %s\n", receipt.ProviderLabel)
	}
	fmt.Fprintf(out, "  Endpoint: %s\n", receipt.Endpoint)
	if strings.TrimSpace(receipt.Model) != "" {
		fmt.Fprintf(out, "  Model:    %s\n", receipt.Model)
	}
	if strings.TrimSpace(receipt.ConfigPath) != "" {
		fmt.Fprintln(out, "  Config:")
		fmt.Fprintf(out, "  %s\n", receipt.ConfigPath)
	}
	if strings.TrimSpace(receipt.SecretsPath) != "" {
		fmt.Fprintln(out, "  Secrets:")
		fmt.Fprintf(out, "  %s\n", receipt.SecretsPath)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Authentication")
	if strings.TrimSpace(receipt.AuthMethod) != "" {
		fmt.Fprintf(out, "  Method: %s\n", receipt.AuthMethod)
	}
	if strings.TrimSpace(receipt.AuthEvidence) != "" {
		fmt.Fprintf(out, "  %s\n", receipt.AuthEvidence)
	}
	if len(receipt.NextSteps) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Next steps")
		for i, step := range receipt.NextSteps {
			fmt.Fprintf(out, "  %d. %s\n", i+1, step)
		}
	}
}

func promptString(cmd *cobra.Command, prompt, defaultVal string) (string, error) {
	fmt.Fprint(cmd.OutOrStdout(), prompt)
	return scanPromptString(cmd, defaultVal)
}

func promptSecret(cmd *cobra.Command, prompt string) (string, error) {
	fmt.Fprint(cmd.OutOrStdout(), prompt)
	if file, ok := cmd.InOrStdin().(*os.File); ok && setupInputIsTerminal(file) {
		input, err := setupReadPassword(int(file.Fd()))
		fmt.Fprintln(cmd.OutOrStdout())
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(input)), nil
	}
	return scanPromptString(cmd, "")
}

func scanPromptString(cmd *cobra.Command, defaultVal string) (string, error) {
	var input string
	_, err := fmt.Fscanln(cmd.InOrStdin(), &input)
	if err != nil {
		if err.Error() == "unexpected newline" || strings.Contains(err.Error(), "expected") {
			return defaultVal, nil
		}
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func runSetupAgentSection(cmd *cobra.Command, section string, seams setupCommandSeams, nonInteractive bool) error {
	out := cmd.OutOrStdout()

	if section == "workspace" {
		if nonInteractive {
			fmt.Fprintln(out, "Workspace setup in non-interactive mode uses defaults.")
			fmt.Fprintf(out, "Default workspace: %s/workspace\n", config.GormesHome())
			fmt.Fprintln(out, "Override in config.toml: [agents.defaults] workspace = \"/path/to/workspace\"")
			return nil
		}
		if !seams.IsTTY() {
			return errSetupRequiresTTY
		}

		fmt.Fprintln(out, "\nMulti-workspace setup")
		fmt.Fprintln(out, "Each agent can have its own workspace directory for file access.")
		workspace, err := promptString(cmd, fmt.Sprintf("Default workspace path [%s/workspace]: ", config.GormesHome()), "")
		if err != nil {
			return err
		}
		if workspace == "" {
			workspace = config.GormesHome() + "/workspace"
		}
		configPath := config.ConfigPath()
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Add this to your config.toml:")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "  [agents.defaults]")
		fmt.Fprintf(out, "  workspace = %q\n", workspace)
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Or open your editor: gormes config edit\n")
		fmt.Fprintf(out, "Config path: %s\n", configPath)
		fmt.Fprintln(out, "Per-agent workspaces go under [[agents.list]] entries.")
		return nil
	}

	// section == "agent"
	if nonInteractive {
		fmt.Fprintln(out, "Agent setup in non-interactive mode creates default agent template.")
		fmt.Fprintln(out, "Run: gormes agent reset")
		return nil
	}
	if !seams.IsTTY() {
		return errSetupRequiresTTY
	}

	fmt.Fprintln(out, "\nMulti-agent setup")
	fmt.Fprintln(out, "Agents are independent personalities with their own workspaces and skills.")
	fmt.Fprintln(out, "The default 'main' agent is created automatically.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "To create additional agents:")
	fmt.Fprintln(out, "  gormes agent reset                      # seed agent templates")
	fmt.Fprintln(out, "  gormes config edit                      # add to [[agents.list]]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Example config.toml addition:")
	fmt.Fprintln(out, "  [[agents.list]]")
	fmt.Fprintln(out, "  id = \"coder\"")
	fmt.Fprintln(out, "  name = \"Coder\"")
	fmt.Fprintln(out, "  workspace = \"/home/xel/projects\"")
	fmt.Fprintln(out, "  model = \"claude-sonnet-4-20250514\"")
	fmt.Fprintln(out)
	return nil
}

func runSetupBindingsSection(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	out := cmd.OutOrStdout()

	if nonInteractive {
		fmt.Fprintln(out, "Bindings setup in non-interactive mode:")
		fmt.Fprintln(out, "Edit config.toml and add [[bindings]] sections:")
		fmt.Fprintln(out, "  [[bindings]]")
		fmt.Fprintln(out, "  agent_id = \"alerts\"")
		fmt.Fprintln(out, "  [bindings.match]")
		fmt.Fprintln(out, "  channel = \"telegram\"")
		fmt.Fprintln(out, "  account_id = \"my-bot\"")
		return nil
	}
	if !seams.IsTTY() {
		return errSetupRequiresTTY
	}

	fmt.Fprintln(out, "\nChannel → Agent Binding Setup")
	fmt.Fprintln(out, "Route messages from specific channels to specific agents.")
	fmt.Fprintln(out)

	channel, err := promptSetupOptionChoice(cmd, "Channel", "Channel (telegram/discord/slack): ", "telegram", []setupOptionChoice{
		{ID: "telegram", Label: "Telegram"},
		{ID: "discord", Label: "Discord"},
		{ID: "slack", Label: "Slack"},
	})
	if err != nil {
		return err
	}
	agentID, err := promptString(cmd, "Agent ID to route to: ", "main")
	if err != nil {
		return err
	}
	accountID, err := promptString(cmd, "Account/bot ID (optional): ", "")
	if err != nil {
		return err
	}

	configPath := config.ConfigPath()

	// Write the binding. Since TOML tables-of-tables need append semantics,
	// we guide the user to the config file and print what to add.
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Add this to your config.toml under [agents]:")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  [[bindings]]\n")
	fmt.Fprintf(out, "  agent_id = %q\n", agentID)
	fmt.Fprintf(out, "  [bindings.match]\n")
	fmt.Fprintf(out, "  channel = %q\n", channel)
	if accountID != "" {
		fmt.Fprintf(out, "  account_id = %q\n", accountID)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Or open your editor: gormes config edit\n")
	fmt.Fprintf(out, "Config path: %s\n", configPath)

	return nil
}

func runSetupModelSection(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
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
	if err := seams.RunModelPicker(cmd); err != nil {
		if errors.Is(err, cli.ErrModelPickerCancelled) {
			fmt.Fprintln(cmd.OutOrStdout(), "Setup cancelled.")
			return nil
		}
		return err
	}
	return nil
}

func runSetupFallbackSection(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	cfg, err := providermodule.LoadFallbackConfig(config.ConfigPath())
	if err != nil {
		return fmt.Errorf("setup fallback: load config: %w", err)
	}

	fmt.Fprintln(out, "Fallback Providers")
	fmt.Fprintln(out, "Fallback providers are tried in order when the primary provider fails.")
	fmt.Fprintln(out)

	if len(cfg.Chain) == 0 {
		fmt.Fprintln(out, "No fallback providers configured.")
	} else {
		fmt.Fprintf(out, "Current fallback chain (%d entries):\n", len(cfg.Chain))
		for i, entry := range cfg.Chain {
			fmt.Fprintf(out, "  %d. %s (via %s)\n", i+1, entry.Model, entry.Provider)
		}
	}
	fmt.Fprintln(out)

	if nonInteractive {
		fmt.Fprintln(out, "Skipped (use `gormes fallback add` to configure)")
		return nil
	}
	if !seams.IsTTY() {
		fmt.Fprintln(cmd.ErrOrStderr(), "setup_requires_tty: run `gormes setup fallback --non-interactive` to skip")
		return errSetupRequiresTTY
	}

	fmt.Fprintln(out, "Actions:")
	fmt.Fprintln(out, "  1. Add a fallback provider")
	fmt.Fprintln(out, "  2. Keep current configuration")
	fmt.Fprintln(out, "  3. Clear fallback chain")
	fmt.Fprintln(out)

	choice, err := promptSetupOptionChoice(cmd, "Fallback action", "Select action [2]: ", "keep", []setupOptionChoice{
		{ID: "add", Label: "Add a fallback provider", Aliases: []string{"1"}},
		{ID: "keep", Label: "Keep current configuration", Aliases: []string{"2"}},
		{ID: "clear", Label: "Clear fallback chain", Aliases: []string{"3"}},
	})
	if err != nil {
		return err
	}
	choice = strings.TrimSpace(strings.ToLower(choice))

	switch choice {
	case "", "keep":
		fmt.Fprintln(out, "Keeping current fallback configuration.")
		return nil
	case "clear":
		if err := providermodule.WriteFallbackChain(config.ConfigPath(), nil); err != nil {
			return err
		}
		fmt.Fprintln(out, "Fallback chain cleared.")
		return nil
	case "add":
		return runSetupFallbackAdd(cmd, seams)
	default:
		fmt.Fprintf(out, "Unknown action: %s — keeping current configuration.\n", choice)
		return nil
	}
}

func runSetupFallbackAdd(cmd *cobra.Command, seams setupCommandSeams) error {
	out := cmd.OutOrStdout()
	entries, defaultIndex := cli.HermesProviderCatalogMenu("")
	idx, err := seams.ChooseSetupProvider(cmd, entries, defaultIndex)
	if err != nil {
		if errors.Is(err, cli.ErrModelPickerCancelled) {
			fmt.Fprintln(out, "No change.")
			return nil
		}
		return err
	}
	if idx < 0 || idx >= len(entries) || entries[idx].ID == cli.ProviderCatalogLeaveUnchanged {
		fmt.Fprintln(out, "No change.")
		return nil
	}

	provider := entries[idx].ID
	if provider == cli.ProviderCatalogAuxConfig {
		fmt.Fprintln(out, "Auxiliary model setup is not available for fallback providers.")
		return nil
	}
	provider = setupCanonicalProviderID(provider)

	suggestions := gormescli.DefaultModelPickerSuggestionSet(provider)
	model, err := gormescli.PromptModelChoiceWithOptions(cmd.InOrStdin(), out, provider, "", suggestions.Models, gormescli.ModelChoicePromptOptions{
		Context:         cmd.Context(),
		SuggestionLimit: gormescli.ModelChoiceSuggestionLimitUnlimited,
	})
	if err != nil {
		if errors.Is(err, cli.ErrModelPickerCancelled) {
			fmt.Fprintln(out, "No change.")
			return nil
		}
		return err
	}
	model = llm.NormalizeProviderModelID(provider, model)

	wrote, err := providermodule.AppendFallbackSelection(config.ConfigPath(), cli.Selection{
		Provider: provider,
		Model:    model,
	})
	if err != nil {
		return err
	}
	if !wrote {
		fmt.Fprintf(out, "  %s (%s) is already in the fallback chain — skipped.\n", model, provider)
		return nil
	}
	fmt.Fprintf(out, "  Added fallback: %s (via %s)\n", model, provider)
	return nil
}

func setupGatewaySeams(seams setupCommandSeams) gormescli.SetupGatewaySeams {
	return gormescli.SetupGatewaySeams{
		RunGatewaySetupWizard:    seams.RunGatewaySetupWizard,
		RunTelegramGatewayWizard: seams.RunTelegramGatewayWizard,
		RunGatewayPlatform:       seams.RunGatewayPlatform,
	}
}

func setupGatewayRuntime(cmd *cobra.Command, seams setupCommandSeams) gormescli.SetupGatewayRuntime {
	return gormescli.SetupGatewayRuntime{
		NewExitCodeError: newExitCodeError,
		PromptString:     promptString,
		PromptSecret:     promptSecret,
		RunWhatsAppSetup: seams.RunWhatsAppSetup,
		RunNavivoxGateway: func(cmd *cobra.Command, cfg config.Config) error {
			return gormescli.RunSetupNavivoxGateway(cmd, cfg, setupNavivoxGatewayOptions(cmd))
		},
	}
}

func runSetupTelegramSection(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	return gormescli.RunSetupTelegramSection(cmd, nonInteractive || !seams.IsTTY(), setupGatewaySeams(seams), setupGatewayRuntime(cmd, seams))
}

func runSetupGatewaySection(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	return gormescli.RunSetupGatewaySection(cmd, nonInteractive, setupGatewaySeams(seams), setupGatewayRuntime(cmd, seams))
}

func runSetupGatewayPlatform(cmd *cobra.Command, platform string, runWhatsAppSetup func(*cobra.Command) error) error {
	return gormescli.RunSetupGatewayPlatform(cmd, platform, gormescli.SetupGatewayRuntime{
		NewExitCodeError: newExitCodeError,
		PromptString:     promptString,
		PromptSecret:     promptSecret,
		RunWhatsAppSetup: runWhatsAppSetup,
		RunNavivoxGateway: func(cmd *cobra.Command, cfg config.Config) error {
			return gormescli.RunSetupNavivoxGateway(cmd, cfg, setupNavivoxGatewayOptions(cmd))
		},
	})
}

func runSetupWhatsAppCommand(cmd *cobra.Command) error {
	whatsAppCmd := newWhatsAppCommand()
	whatsAppCmd.SetOut(cmd.OutOrStdout())
	whatsAppCmd.SetErr(cmd.ErrOrStderr())
	whatsAppCmd.SetIn(cmd.InOrStdin())
	whatsAppCmd.SetArgs([]string{"--plan"})
	whatsAppCmd.SilenceUsage = true
	whatsAppCmd.SilenceErrors = true
	return whatsAppCmd.ExecuteContext(cmd.Context())
}

var setupToolProgressModes = []string{"off", "new", "all", "verbose"}

func promptSetupToolProgressMode(cmd *cobra.Command, current string) (mode string, selected bool, invalid string, err error) {
	current = normalizeSetupChoice(current)
	if !isKnownToolProgressMode(current) {
		current = "all"
	}
	if stdin, ok := cmd.InOrStdin().(*os.File); ok && setupInputIsTerminal(stdin) {
		selectedMode, pickErr := runBubbleTeaPick(
			cmd.Context(),
			stdin,
			cmd.OutOrStdout(),
			"Tool progress mode",
			setupToolProgressPickerChoices(),
			current,
		)
		if pickErr == nil {
			if selectedMode == "" {
				return current, false, "", nil
			}
			return selectedMode, true, "", nil
		}
		if !bubbleTeaPickShouldFallback(pickErr) {
			return "", false, "", pickErr
		}
	}
	return promptSetupToolProgressModeText(cmd, current)
}

func promptSetupToolProgressModeText(cmd *cobra.Command, current string) (mode string, selected bool, invalid string, err error) {
	out := cmd.OutOrStdout()
	defaultIndex := indexSetupToolProgressMode(current)
	if defaultIndex < 0 {
		defaultIndex = indexSetupToolProgressMode("all")
	}
	fmt.Fprintln(out, "Select tool progress mode:")
	for i, mode := range setupToolProgressModes {
		marker := " "
		if i == defaultIndex {
			marker = "→"
		}
		fmt.Fprintf(out, "  %s %d. %s\n", marker, i+1, mode)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Choice [1-%d] (%d), or q to cancel: ", len(setupToolProgressModes), defaultIndex+1)
	answer, err := scanPromptString(cmd, strconv.Itoa(defaultIndex+1))
	if err != nil {
		return "", false, "", err
	}
	rawAnswer := strings.TrimSpace(answer)
	normalized := normalizeSetupChoice(rawAnswer)
	if normalized == "q" || normalized == "cancel" || normalized == "keep" {
		return current, false, "", nil
	}
	if idx, parseErr := strconv.Atoi(rawAnswer); parseErr == nil {
		if idx < 1 || idx > len(setupToolProgressModes) {
			return current, false, rawAnswer, nil
		}
		return setupToolProgressModes[idx-1], true, "", nil
	}
	if isKnownToolProgressMode(normalized) {
		return normalized, true, "", nil
	}
	return current, false, rawAnswer, nil
}

func setupToolProgressPickerChoices() []tuiPickChoice {
	choices := make([]tuiPickChoice, len(setupToolProgressModes))
	for i, mode := range setupToolProgressModes {
		choices[i] = tuiPickChoice{ID: mode, Label: mode}
	}
	return choices
}

func indexSetupToolProgressMode(current string) int {
	return gormescli.SetupToolProgressModeIndex(current)
}

func runSetupAgentSettingsSection(cmd *cobra.Command, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	cfg, _ := config.Load(nil)
	maxIterations := cfg.Runtime.MaxToolIterations
	if maxIterations <= 0 {
		maxIterations = 90
	}
	toolProgress := firstNonEmptySetup(cfg.Display.ToolProgress, "all")
	compressionThreshold := cfg.Runtime.CompressionThreshold
	if compressionThreshold <= 0 {
		compressionThreshold = 0.5
	}
	sessionPolicy := firstNonEmptySetup(cfg.Runtime.SessionResetPolicy, "inactivity")

	fmt.Fprintln(out, "Agent Settings")
	fmt.Fprintf(out, "Max iterations [%d]\n", maxIterations)
	fmt.Fprintf(out, "Tool progress mode [%s]\n", toolProgress)
	fmt.Fprintf(out, "Compression threshold [%.2g]\n", compressionThreshold)
	fmt.Fprintf(out, "Session reset policy [%s]\n", sessionPolicy)
	if nonInteractive {
		fmt.Fprintln(out, "\nSkipped (keeping current)")
		return nil
	}

	maxText, err := promptString(cmd, fmt.Sprintf("Max iterations [%d]: ", maxIterations), strconv.Itoa(maxIterations))
	if err != nil {
		return err
	}
	if parsed, ok := parsePositiveInt(maxText); ok {
		if err := config.WriteTOMLValue(config.ConfigPath(), "runtime.max_tool_iterations", strconv.Itoa(parsed)); err != nil {
			return err
		}
		fmt.Fprintf(out, "Max iterations set to %d\n", parsed)
	} else {
		fmt.Fprintf(out, "setup_agent_value_ignored: max_iterations=%q\n", maxText)
	}

	progress, selectedProgress, invalidProgress, err := promptSetupToolProgressMode(cmd, toolProgress)
	if err != nil {
		return err
	}
	if invalidProgress != "" {
		fmt.Fprintf(out, "Unknown tool progress mode %q; keeping %s\n", invalidProgress, toolProgress)
	} else if selectedProgress {
		if err := config.WriteTOMLValue(config.ConfigPath(), "display.tool_progress", progress); err != nil {
			return err
		}
		fmt.Fprintf(out, "Tool progress set to: %s\n", progress)
	} else {
		fmt.Fprintf(out, "Tool progress unchanged: %s\n", progress)
	}

	thresholdText, err := promptString(cmd, fmt.Sprintf("Compression threshold [%.2g]: ", compressionThreshold), strconv.FormatFloat(compressionThreshold, 'f', -1, 64))
	if err != nil {
		return err
	}
	if parsed, ok := parseThreshold(thresholdText); ok {
		if err := config.WriteTOMLValue(config.ConfigPath(), "runtime.compression_threshold", strconv.FormatFloat(parsed, 'f', -1, 64)); err != nil {
			return err
		}
		fmt.Fprintf(out, "Compression threshold set to %.2g\n", parsed)
	} else {
		fmt.Fprintf(out, "setup_agent_value_ignored: compression_threshold=%q\n", thresholdText)
	}

	policy, err := promptSetupOptionChoice(cmd, "Session reset policy", fmt.Sprintf("Session reset policy [%s]: ", sessionPolicy), sessionPolicy, []setupOptionChoice{
		{ID: "inactivity", Label: "Inactivity"},
		{ID: "daily", Label: "Daily"},
		{ID: "manual", Label: "Manual"},
		{ID: "off", Label: "Off", Aliases: []string{"none"}},
	})
	if err != nil {
		return err
	}
	policy = normalizeSetupChoice(policy)
	if policy == "" || policy == "keep" {
		policy = sessionPolicy
	}
	if isKnownSessionResetPolicy(policy) {
		if err := config.WriteTOMLValue(config.ConfigPath(), "runtime.session_reset_policy", policy); err != nil {
			return err
		}
		fmt.Fprintf(out, "Session reset policy set to: %s\n", policy)
	} else {
		fmt.Fprintf(out, "setup_agent_value_ignored: session_reset_policy=%q\n", policy)
	}
	return nil
}

type setupChoice = gormescli.SetupChoice
type setupOptionChoice = gormescli.SetupOptionChoice

func setupOptionPromptRuntime() gormescli.SetupOptionPromptRuntime {
	return gormescli.SetupOptionPromptRuntime{
		IsTerminal:         setupInputIsTerminal,
		RunPick:            runBubbleTeaPickWithOptions,
		PickShouldFallback: bubbleTeaPickShouldFallback,
		PromptString:       promptString,
		ExitCodeError:      newExitCodeError,
	}
}

func promptSetupOptionChoice(cmd *cobra.Command, title, linePrompt, defaultID string, choices []setupOptionChoice) (string, error) {
	return gormescli.PromptSetupOptionChoice(cmd, title, linePrompt, defaultID, choices, setupOptionPromptRuntime())
}

func setupNavivoxGatewayOptions(cmd *cobra.Command) gormescli.SetupNavivoxOptions {
	return gormescli.SetupNavivoxOptions{
		AskYesNo: func(title, linePrompt string, defaultValue bool) (bool, bool, error) {
			return promptSetupYesNoOption(cmd, title, linePrompt, defaultValue)
		},
		PromptChoice: func(title, linePrompt, defaultID string, choices []setupOptionChoice) (string, error) {
			return promptSetupOptionChoice(cmd, title, linePrompt, defaultID, choices)
		},
		PromptString: func(prompt, defaultValue string) (string, error) {
			return promptString(cmd, prompt, defaultValue)
		},
		ParsePositiveInt: parsePositiveInt,
		WriteProfileChannelBinding: func(opts gormescli.SetupNavivoxProfileChannelOptions) (gormescli.SetupNavivoxProfileChannelBinding, error) {
			binding, err := gormescli.WriteSetupGatewayProfileChannelBinding(gormescli.SetupGatewayProfileChannelOptions{
				ChannelID:      opts.ChannelID,
				AllowedChats:   opts.AllowedChats,
				AllowedUsers:   opts.AllowedUsers,
				RequireMention: opts.RequireMention,
				ToolProgress:   opts.ToolProgress,
			})
			return gormescli.SetupNavivoxProfileChannelBinding{
				ProfileID:     binding.ProfileID,
				ChannelID:     binding.ChannelID,
				CredentialID:  binding.CredentialID,
				SecretEnvName: binding.SecretEnvName,
				RegistryPath:  binding.RegistryPath,
			}, err
		},
		WriteGatewayTokenEnv: func(binding gormescli.SetupNavivoxProfileChannelBinding, legacyEnvName, token string) error {
			return gormescli.WriteSetupGatewayTokenEnv(gormescli.SetupGatewayProfileChannelBinding{
				ProfileID:     binding.ProfileID,
				ChannelID:     binding.ChannelID,
				CredentialID:  binding.CredentialID,
				SecretEnvName: binding.SecretEnvName,
				RegistryPath:  binding.RegistryPath,
			}, legacyEnvName, token)
		},
	}
}

func promptSetupYesNoOption(cmd *cobra.Command, title, linePrompt string, defaultValue bool) (bool, bool, error) {
	return gormescli.PromptSetupYesNoOption(cmd, title, linePrompt, defaultValue, setupOptionPromptRuntime())
}

func promptSetupChoice(cmd *cobra.Command, title, linePrompt, defaultValue string, choices []setupChoice) (string, error) {
	return gormescli.PromptSetupChoice(cmd, title, linePrompt, defaultValue, choices, setupOptionPromptRuntime())
}

func setupChoicesToOptions(choices []setupChoice) []setupOptionChoice {
	return gormescli.SetupChoicesToOptions(choices)
}

func setupOptionPickerChoices(options []setupOptionChoice) []tuiPickChoice {
	return gormescli.SetupOptionPickerChoices(options)
}

func normalizeSetupOptionChoice(answer string, options []setupOptionChoice, defaultID string) string {
	return gormescli.NormalizeSetupAnswer(answer, options, defaultID)
}

func setupSelectionSeparator(r rune) bool {
	return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == ';'
}

func normalizeSetupChoice(value string) string {
	return gormescli.NormalizeSetupValue(value)
}

func parsePositiveInt(value string) (int, bool) {
	return gormescli.SetupParsePositiveInt(value)
}

func parseThreshold(value string) (float64, bool) {
	return gormescli.SetupParseCompressionThreshold(value)
}

func isKnownToolProgressMode(value string) bool {
	return gormescli.SetupIsKnownToolProgressMode(value)
}

func isKnownSessionResetPolicy(value string) bool {
	return gormescli.SetupIsKnownSessionResetPolicy(value)
}

func firstNonEmptySetup(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func setupSectionUnsupported(cmd *cobra.Command, section string) error {
	gormescli.WriteSetupSectionUnsupported(cmd.ErrOrStderr(), section, setupSectionList())
	return newExitCodeError(2, gormescli.SetupSectionUnsupportedError(section))
}

func setupSectionList() string {
	return setupRegistry.SectionList()
}

func setupSectionOwnership(section string) string {
	return gormescli.SetupSectionOwnership(section)
}
