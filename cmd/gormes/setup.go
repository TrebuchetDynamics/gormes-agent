package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/plugins"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var errSetupRequiresTTY = errors.New("setup_requires_tty")

var setupReadPassword = term.ReadPassword
var setupInputIsTerminal = stdinIsTerminal

var setupSections = []string{"provider", "model", "agent", "workspace", "bindings", "tts", "terminal", "gateway", "tools"}

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
	"opencode":        "gpt-5.2",
	"opencode-go":     "gpt-5.2",
}

type setupCommandSeams struct {
	IsTTY                         func() bool
	HasExistingInstall            func() (bool, error)
	ResetConfig                   func() (string, error)
	RunModelPicker                func(*cobra.Command) error
	LoadCurrentModel              func() (cli.ProviderModel, error)
	ChooseSetupAction             func(*cobra.Command, []setupMenuOption, int) (setupAction, error)
	ChooseSetupTarget             func(*cobra.Command, []cli.SetupTargetOption, int) (cli.SetupTargetID, error)
	RunSetupProvider              func(*cobra.Command, bool) error
	RunProviderLiveTest           func(*cobra.Command) error
	DetectHermesMigrationSource   func() string
	DetectOpenClawMigrationSource func() string
	RunFullWizard                 func(*cobra.Command, bool) error
	RunSetupGateway               func(*cobra.Command, bool) error
	RunSetupTools                 func(*cobra.Command, bool) error
	RunGatewaySetupWizard         func(*cobra.Command, config.Config) (setupGatewayWizardResult, error)
	RunGatewayPlatform            func(*cobra.Command, string) error
	RunWhatsAppSetup              func(*cobra.Command) error
	LaunchChat                    func(*cobra.Command) error
}

type setupGatewayWizardResult struct {
	SelectedPlatforms []string
	Telegram          *setupTelegramGatewayAnswers
	BubbleTea         bool
}

type setupTelegramGatewayAnswers struct {
	Token        string
	AccessPolicy string
	AllowedUsers string
	HomeChatID   string
	HomeThreadID string
	Apply        bool
}

type setupAction string

const (
	setupActionQuick           setupAction = "quick"
	setupActionFull            setupAction = "full"
	setupActionModelProvider   setupAction = "model_provider"
	setupActionTerminal        setupAction = "terminal"
	setupActionGateway         setupAction = "gateway"
	setupActionTools           setupAction = "tools"
	setupActionAgent           setupAction = "agent"
	setupActionMigrateHermes   setupAction = "migrate_hermes"
	setupActionMigrateOpenClaw setupAction = "migrate_openclaw"
	setupActionExit            setupAction = "exit"
)

type setupMenuOption struct {
	Action setupAction
	Label  string
}

func newSetupCommand() *cobra.Command {
	return newSetupCommandWithSeams(defaultSetupCommandSeams())
}

func newSetupCommandWithSeams(seams setupCommandSeams) *cobra.Command {
	if seams.IsTTY == nil {
		seams.IsTTY = isStdinTTY
	}
	if seams.HasExistingInstall == nil {
		seams.HasExistingInstall = defaultSetupHasExistingInstall
	}
	if seams.ResetConfig == nil {
		seams.ResetConfig = resetSetupDefaultConfig
	}
	if seams.RunModelPicker == nil {
		seams.RunModelPicker = func(cmd *cobra.Command) error {
			pickerCmd := newModelCommand()
			pickerCmd.SetOut(cmd.OutOrStdout())
			pickerCmd.SetErr(cmd.ErrOrStderr())
			pickerCmd.SetIn(cmd.InOrStdin())
			pickerCmd.SetArgs([]string{})
			pickerCmd.SilenceUsage = true
			pickerCmd.SilenceErrors = true
			return pickerCmd.ExecuteContext(cmd.Context())
		}
	}
	if seams.LoadCurrentModel == nil {
		seams.LoadCurrentModel = defaultSetupLoadCurrentModel
	}
	if seams.ChooseSetupAction == nil {
		seams.ChooseSetupAction = promptSetupAction
	}
	if seams.ChooseSetupTarget == nil {
		seams.ChooseSetupTarget = promptSetupTarget
	}
	if seams.RunSetupProvider == nil {
		seams.RunSetupProvider = func(cmd *cobra.Command, nonInteractive bool) error {
			return runSetupProviderSection(cmd, seams, nonInteractive)
		}
	}
	if seams.RunProviderLiveTest == nil {
		seams.RunProviderLiveTest = runSetupProviderLiveTest
	}
	if seams.DetectHermesMigrationSource == nil {
		seams.DetectHermesMigrationSource = detectHermesMigrationSource
	}
	if seams.DetectOpenClawMigrationSource == nil {
		seams.DetectOpenClawMigrationSource = detectOpenClawMigrationSource
	}
	if seams.RunSetupTools == nil {
		seams.RunSetupTools = runSetupToolsSection
	}
	if seams.RunWhatsAppSetup == nil {
		seams.RunWhatsAppSetup = runSetupWhatsAppCommand
	}
	if seams.RunGatewaySetupWizard == nil {
		seams.RunGatewaySetupWizard = runSetupGatewayBubbleTeaWizard
	}
	if seams.RunGatewayPlatform == nil {
		seams.RunGatewayPlatform = func(cmd *cobra.Command, platform string) error {
			return runSetupGatewayPlatform(cmd, platform, seams.RunWhatsAppSetup)
		}
	}
	if seams.LaunchChat == nil {
		seams.LaunchChat = launchSetupChat
	}
	if seams.RunSetupGateway == nil {
		seams.RunSetupGateway = func(cmd *cobra.Command, nonInteractive bool) error {
			return runSetupGatewaySection(cmd, seams, nonInteractive)
		}
	}
	if seams.RunFullWizard == nil {
		seams.RunFullWizard = func(cmd *cobra.Command, nonInteractive bool) error {
			return runSetupFullWizard(cmd, seams, nonInteractive)
		}
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
				if asJSON {
					body, marshalErr := json.MarshalIndent(setupResetReportJSON{
						Build:          newBuildProvenance(),
						Action:         "reset",
						ConfigPath:     config.ConfigPath(),
						BreadcrumbPath: breadcrumb,
					}, "", "  ")
					if marshalErr != nil {
						return marshalErr
					}
					fmt.Fprintln(cmd.OutOrStdout(), string(body))
					return nil
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Configuration reset to defaults.")
				if breadcrumb != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Prior config preserved at %s — restore with `cp %s %s`\n", breadcrumb, breadcrumb, config.ConfigPath())
				}
			}
			if len(args) > 0 {
				section := strings.ToLower(strings.TrimSpace(args[0]))
				return runSetupSection(cmd, seams, section, nonInteractive)
			}
			if quick {
				return runSetupQuick(cmd, seams, headless, cli.SetupTargetID(targetFlag))
			}
			if headless {
				printSetupSections(cmd)
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
				return runSetupFirstTimeChoice(cmd, seams, false)
			}
			if existing {
				return seams.RunFullWizard(cmd, false)
			}
			return runSetupFirstTimeChoice(cmd, seams, false)
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

// setupResetReportJSON is the wire shape for `setup --reset --json`.
// Fleet automation running reset across machines parses this to record
// where each operator's prior config was preserved (`breadcrumb_path`)
// — the recovery handle scripts capture for rollback.
type setupResetReportJSON struct {
	Build          buildProvenanceJSON `json:"build"`
	Action         string              `json:"action"`
	ConfigPath     string              `json:"config_path"`
	BreadcrumbPath string              `json:"breadcrumb_path"`
}

func defaultSetupCommandSeams() setupCommandSeams {
	return setupCommandSeams{
		IsTTY:                         isStdinTTY,
		HasExistingInstall:            defaultSetupHasExistingInstall,
		ResetConfig:                   resetSetupDefaultConfig,
		LoadCurrentModel:              defaultSetupLoadCurrentModel,
		ChooseSetupAction:             promptSetupAction,
		ChooseSetupTarget:             promptSetupTarget,
		RunProviderLiveTest:           runSetupProviderLiveTest,
		DetectHermesMigrationSource:   detectHermesMigrationSource,
		DetectOpenClawMigrationSource: detectOpenClawMigrationSource,
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
		strings.TrimSpace(os.Getenv("GORMES_API_KEY")) != "", nil
}

func resetSetupDefaultConfig() (string, error) {
	path := config.ConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("setup reset: mkdir %s: %w", dir, err)
	}
	var breadcrumb string
	if prior, readErr := os.ReadFile(path); readErr == nil {
		breadcrumb = path + ".before-reset." + time.Now().UTC().Format("20060102T150405Z")
		if err := os.WriteFile(breadcrumb, prior, 0o600); err != nil {
			return "", fmt.Errorf("setup reset: write breadcrumb %s: %w", breadcrumb, err)
		}
	} else if !os.IsNotExist(readErr) {
		return "", fmt.Errorf("setup reset: read prior %s: %w", path, readErr)
	}
	body, err := toml.Marshal(map[string]any{"_config_version": int64(config.CurrentConfigVersion)})
	if err != nil {
		return "", fmt.Errorf("setup reset: marshal defaults: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".config.toml.reset-*")
	if err != nil {
		return "", fmt.Errorf("setup reset: tempfile: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", fmt.Errorf("setup reset: write temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return "", fmt.Errorf("setup reset: chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("setup reset: close temp: %w", err)
	}
	if _, err := toolspkg.AtomicReplace(tmpName, path, toolspkg.AtomicReplaceOptions{FirstWriteMode: 0o600}); err != nil {
		os.Remove(tmpName)
		return "", fmt.Errorf("setup reset: replace %s: %w", path, err)
	}
	return breadcrumb, nil
}

func printSetupSections(cmd *cobra.Command) {
	fmt.Fprintln(cmd.OutOrStdout(), "Available setup sections:")
	for _, section := range setupSections {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", section)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "\nQuick start: run `gormes setup provider` to configure your LLM provider.")
}

func runSetupRoot(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	if nonInteractive || !seams.IsTTY() {
		printSetupSections(cmd)
		return nil
	}

	options := setupTopLevelOptions()
	defaultOption := 0
	printSetupTopLevelMenu(cmd, options, defaultOption)
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
	case setupActionTerminal:
		return runSetupTerminalSection(cmd, nonInteractive)
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
	cli.PrintHeader(out, "How would you like to set up Gormes?")
	fmt.Fprintln(out, cli.Dim(out, "  No existing Gormes configuration was found."))
	fmt.Fprintln(out)
	for i, option := range options {
		var prefix, label string
		if i == 0 {
			prefix = " " + cli.BrightCyan(out, "→") + " " + cli.BrightCyan(out, "(●)")
			label = cli.Bold(out, option.Label)
		} else {
			prefix = "   " + cli.Dim(out, "(○)")
			label = option.Label
		}
		fmt.Fprintf(out, "%s %s\n", prefix, label)
	}
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

func runSetupSection(cmd *cobra.Command, seams setupCommandSeams, section string, nonInteractive bool) error {
	switch section {
	case "provider":
		return runSetupProviderSection(cmd, seams, nonInteractive)
	case "model":
		return runSetupModelSection(cmd, seams, nonInteractive)
	case "agent":
		if !nonInteractive && !seams.IsTTY() {
			return errSetupRequiresTTY
		}
		return runSetupAgentSettingsSection(cmd, nonInteractive)
	case "workspace":
		return runSetupAgentSection(cmd, section, seams, nonInteractive)
	case "bindings":
		return runSetupBindingsSection(cmd, seams, nonInteractive)
	case "tts":
		return runSetupTTSSection(cmd, nonInteractive)
	case "terminal":
		return runSetupTerminalSection(cmd, nonInteractive)
	case "gateway":
		return seams.RunSetupGateway(cmd, nonInteractive || !seams.IsTTY())
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
	out := cmd.OutOrStdout()
	cli.ClearScreen(out)
	cli.PrintHeader(out, "What would you like to do?")
	fmt.Fprintln(out, cli.Dim(out, "  Use ↑/↓ arrows to navigate, Enter to select, or type a number."))
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
		selected, err := runBubbleTeaPick(cmd.Context(), stdin, cmd.OutOrStdout(), "What would you like to do?", setupActionPickerChoices(options), string(options[defaultOption].Action))
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
	var b strings.Builder
	for i := 0; i < len(answer); {
		ch := answer[i]
		if ch == 0x1b {
			i++
			if i < len(answer) && answer[i] == '[' {
				i++
				for i < len(answer) {
					final := answer[i]
					i++
					if final >= 0x40 && final <= 0x7e {
						break
					}
				}
				continue
			}
			if i < len(answer) {
				i++
			}
			continue
		}
		if ch < 0x20 || ch == 0x7f {
			i++
			continue
		}
		b.WriteByte(ch)
		i++
	}
	return b.String()
}

func runSetupFullWizard(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	printSetupWizardHeader(cmd)
	if nonInteractive {
		if err := runSetupModelSection(cmd, seams, true); err != nil {
			return err
		}
		if err := runSetupTTSSection(cmd, true); err != nil {
			return err
		}
		if err := runSetupTerminalSection(cmd, true); err != nil {
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

	if err := runSetupModelSection(cmd, seams, false); err != nil {
		return err
	}
	if err := runSetupTTSSection(cmd, false); err != nil {
		return err
	}
	if err := runSetupTerminalSection(cmd, false); err != nil {
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
	fmt.Fprintln(out, "   gormes setup terminal Change terminal backend")
	fmt.Fprintln(out, "   gormes setup gateway  Configure messaging")
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
	answer, err := promptString(cmd, "Launch gormes chat now? [Y/n]: ", "y")
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "", "y", "yes":
		return seams.LaunchChat(cmd)
	case "n", "no":
		return nil
	default:
		return newExitCodeError(2, fmt.Errorf("setup_launch_invalid_selection: %s", answer))
	}
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
	out := cmd.OutOrStdout()
	cli.ClearScreen(out)
	cli.PrintHeader(out, "Setup section: provider")
	if nonInteractive {
		return setupProviderNonInteractive(cmd)
	}
	if !seams.IsTTY() {
		fmt.Fprintln(cmd.ErrOrStderr(), "setup_requires_tty: run `gormes setup provider --non-interactive` to use GORMES_ENDPOINT + GORMES_API_KEY env vars")
		return errSetupRequiresTTY
	}
	return setupProviderInteractive(cmd)
}

func setupProviderNonInteractive(cmd *cobra.Command) error {
	endpoint := strings.TrimSpace(os.Getenv("GORMES_ENDPOINT"))
	apiKey := strings.TrimSpace(os.Getenv("GORMES_API_KEY"))
	if endpoint == "" || apiKey == "" {
		return fmt.Errorf("setup provider --non-interactive: GORMES_ENDPOINT and GORMES_API_KEY must be set")
	}
	model := strings.TrimSpace(os.Getenv("GORMES_MODEL"))
	return writeProviderConfig(cmd, "", endpoint, apiKey, model)
}

func setupProviderInteractive(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "\nSelect a provider or enter a custom endpoint:")
	fmt.Fprintln(out)
	for _, p := range []string{providerOpenAI, providerAnthropic, providerDeepSeek, "openai-codex", "opencode", providerGroq, providerOllama} {
		fmt.Fprintf(out, "  %-12s %s\n", p, knownProviderEndpoints[p])
	}
	fmt.Fprintln(out, "  custom       enter your own endpoint URL")
	fmt.Fprintln(out)

	provider, err := promptString(cmd, "Provider [openai]: ", "openai")
	if err != nil {
		return err
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = providerOpenAI
	}

	var endpoint string
	if ep, ok := knownProviderEndpoints[provider]; ok {
		endpoint = ep
	} else {
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

	defaultModel := knownProviderModels[provider]
	if defaultModel == "" {
		defaultModel = ""
	}
	model, err := promptString(cmd, fmt.Sprintf("Model [%s]: ", defaultModel), defaultModel)
	if err != nil {
		return err
	}

	return writeProviderConfig(cmd, provider, endpoint, apiKey, model)
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

	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Provider configured.\n\n")
	fmt.Fprintf(out, "Config:  %s\n", configPath)
	fmt.Fprintf(out, "Secrets: %s\n", envPath)
	if provider != "" {
		fmt.Fprintf(out, "Provider: %s\n", provider)
	}
	fmt.Fprintf(out, "Endpoint: %s\n", endpoint)
	fmt.Fprintln(out, "API key:  stored (redacted)")
	if model != "" {
		fmt.Fprintf(out, "Model:    %s\n", model)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Test it:  gormes --oneshot \"hello\"")
	return nil
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

	channel, err := promptString(cmd, "Channel (telegram/discord/slack): ", "telegram")
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
	out := cmd.OutOrStdout()
	cli.ClearScreen(out)
	cli.PrintHeader(out, "Setup section: model")
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

func runSetupTTSSection(cmd *cobra.Command, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	cfg, _ := config.Load(nil)
	current := firstNonEmptySetup(cfg.Runtime.TTSProvider, "edge")

	fmt.Fprintln(out, "Text-to-Speech Provider")
	fmt.Fprintf(out, "Current: %s\n", ttsProviderLabel(current))
	fmt.Fprintln(out)
	for _, option := range ttsProviderOptions() {
		selected := "○"
		if option.value == "keep" {
			selected = "●"
		}
		fmt.Fprintf(out, "  (%s) %s\n", selected, option.label)
	}
	if nonInteractive {
		fmt.Fprintln(out, "\nSkipped (keeping current)")
		return nil
	}

	choice, err := promptString(cmd, "Select TTS provider [keep]: ", "keep")
	if err != nil {
		return err
	}
	choice = normalizeSetupChoice(choice)
	if choice == "" || choice == "keep" {
		fmt.Fprintln(out, "Keeping current TTS provider.")
		return nil
	}
	switch choice {
	case "edge", "openai":
		if err := config.WriteTOMLValue(config.ConfigPath(), "runtime.tts_provider", choice); err != nil {
			return err
		}
		fmt.Fprintf(out, "TTS provider set to: %s\n", ttsProviderLabel(choice))
		return nil
	default:
		fmt.Fprintf(cmd.ErrOrStderr(), "setup_tts_provider_row_backed: provider=%s\n", choice)
		return newExitCodeError(2, fmt.Errorf("setup_tts_provider_row_backed: %s", choice))
	}
}

func runSetupTerminalSection(cmd *cobra.Command, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	cfg, _ := config.Load(nil)
	current := firstNonEmptySetup(cfg.Runtime.TerminalBackend, "local")

	fmt.Fprintln(out, "Terminal Backend")
	fmt.Fprintf(out, "Current: %s\n", terminalBackendLabel(current))
	fmt.Fprintln(out)
	for _, option := range terminalBackendOptions() {
		selected := "○"
		if option.value == "keep" {
			selected = "●"
		}
		fmt.Fprintf(out, "  (%s) %s\n", selected, option.label)
	}
	if nonInteractive {
		fmt.Fprintf(out, "\nKeeping current backend: %s\n", current)
		return nil
	}

	choice, err := promptString(cmd, "Select terminal backend [keep]: ", "keep")
	if err != nil {
		return err
	}
	choice = normalizeSetupChoice(choice)
	if choice == "" || choice == "keep" {
		fmt.Fprintf(out, "Keeping current backend: %s\n", current)
		return nil
	}
	switch choice {
	case "local":
		if err := config.WriteTOMLValue(config.ConfigPath(), "runtime.terminal_backend", choice); err != nil {
			return err
		}
		fmt.Fprintln(out, "Terminal backend set to: local")
		return nil
	default:
		fmt.Fprintf(cmd.ErrOrStderr(), "setup_terminal_backend_row_backed: backend=%s\n", choice)
		return newExitCodeError(2, fmt.Errorf("setup_terminal_backend_row_backed: %s", choice))
	}
}

func runSetupGatewaySection(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("setup gateway: load config: %w", err)
	}

	if setupGatewayPlanFlag(cmd) {
		renderSetupGatewayPlan(out, cfg, true)
		return nil
	}
	if nonInteractive {
		renderSetupGatewayPlan(out, cfg, true)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Skipped (keeping current gateway platform configuration).")
		fmt.Fprintln(out, "Run `gormes setup gateway` from a TTY or configure credentials with `gormes config edit`.")
		fmt.Fprintln(out, "Start messaging with: gormes gateway")
		return nil
	}

	result, err := seams.RunGatewaySetupWizard(cmd, cfg)
	if err != nil {
		if errors.Is(err, setupwizard.ErrRequiresTTY) {
			return newExitCodeError(2, fmt.Errorf("setup_gateway_requires_tty: run `gormes setup gateway --plan` for offline guidance, or run `gormes setup gateway` in a terminal"))
		}
		return err
	}
	selected := compactStringsSetup(result.SelectedPlatforms)
	if len(selected) == 0 {
		fmt.Fprintln(out, "No platform setup changes selected.")
		fmt.Fprintln(out, "Keeping current gateway platform configuration.")
		return nil
	}
	for _, platform := range selected {
		if platform == "telegram" && result.Telegram != nil {
			if err := applySetupTelegramGatewayAnswers(cmd, cfg.Telegram, *result.Telegram); err != nil {
				return err
			}
			continue
		}
		if result.BubbleTea && platform != "whatsapp" {
			fmt.Fprintf(out, "%s Bubble Tea setup is not shipped in this slice; use `gormes setup gateway --plan` and `gormes config edit` for now.\n", setupGatewayPlatformFallbackLabel(platform))
			continue
		}
		if platform == "navivox" {
			if err := runSetupNavivoxGateway(cmd, cfg); err != nil {
				return err
			}
			continue
		}
		if err := seams.RunGatewayPlatform(cmd, platform); err != nil {
			return err
		}
	}
	fmt.Fprintln(out, "Start messaging with: gormes gateway")
	return nil
}

func setupGatewayPlanFlag(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	value, err := cmd.Flags().GetBool("plan")
	return err == nil && value
}

func renderSetupGatewayPlan(out io.Writer, cfg config.Config, planOnly bool) {
	plan := gateway.BuildChannelSetupPlan(cfg)
	fmt.Fprintln(out, "Messaging Platforms")
	if planOnly {
		fmt.Fprintln(out, "Plan only: no files will be written and no live APIs will be called.")
	}
	for _, entry := range plan.Channels {
		fmt.Fprintf(out, "\n%s (%s): %s\n", entry.DisplayName, entry.ID, entry.Status)
		if len(entry.RequiredFields) > 0 {
			fmt.Fprintf(out, "  Required: %s\n", strings.Join(entry.RequiredFields, ", "))
		}
		if len(entry.CurrentValues) > 0 {
			fmt.Fprintf(out, "  Current: %s\n", strings.Join(entry.CurrentValues, ", "))
		}
		if len(entry.PlannedWrites) > 0 {
			fmt.Fprintf(out, "  Planned writes: %s\n", strings.Join(entry.PlannedWrites, ", "))
		}
		if len(entry.Warnings) > 0 {
			fmt.Fprintf(out, "  Warnings: %s\n", strings.Join(entry.Warnings, " "))
		}
		if entry.NextCommand != "" {
			fmt.Fprintf(out, "  Next: %s\n", entry.NextCommand)
		}
	}
	if plan.GatewayAction != "" {
		fmt.Fprintf(out, "\nGateway action: %s\n", plan.GatewayAction)
	}
}

type setupGatewayPlatformOption struct {
	key        string
	label      string
	configured bool
	detail     string
}

func setupGatewayPlatformOptions(cfg config.Config) []setupGatewayPlatformOption {
	configured := map[string]string{}
	for _, channel := range configuredGatewayStatusChannels(cfg) {
		configured[channel.Name] = channel.Detail
	}
	manifestByID := map[string]gateway.PlatformManifestEntry{}
	for _, entry := range gateway.HermesGatewayPlatformManifest() {
		manifestByID[entry.ID] = entry
	}

	out := make([]setupGatewayPlatformOption, 0, 5)
	for _, key := range []string{"telegram", "discord", "slack", "whatsapp", "navivox"} {
		label := setupGatewayPlatformFallbackLabel(key)
		if entry, ok := manifestByID[key]; ok && strings.TrimSpace(entry.DisplayName) != "" {
			label = entry.DisplayName
		}
		detail, ok := configured[key]
		out = append(out, setupGatewayPlatformOption{
			key:        key,
			label:      label,
			configured: ok,
			detail:     detail,
		})
	}
	return out
}

func setupGatewayPlatformFallbackLabel(key string) string {
	switch key {
	case "telegram":
		return "Telegram"
	case "discord":
		return "Discord"
	case "slack":
		return "Slack"
	case "whatsapp":
		return "WhatsApp"
	case "navivox":
		return "Navivox"
	default:
		return key
	}
}

func parseSetupGatewaySelection(input string, options []setupGatewayPlatformOption) ([]string, bool, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, true, nil
	}
	byKey := make(map[string]setupGatewayPlatformOption, len(options))
	byLabel := make(map[string]setupGatewayPlatformOption, len(options))
	for _, option := range options {
		byKey[option.key] = option
		byLabel[normalizeSetupChoice(option.label)] = option
	}

	var selected []string
	seen := map[string]bool{}
	for _, token := range strings.FieldsFunc(input, setupSelectionSeparator) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		var key string
		if index, err := strconv.Atoi(token); err == nil {
			if index < 1 || index > len(options) {
				return nil, false, newExitCodeError(2, fmt.Errorf("setup_gateway_invalid_selection: %s", token))
			}
			key = options[index-1].key
		} else {
			normalized := normalizeSetupChoice(token)
			if option, ok := byKey[normalized]; ok {
				key = option.key
			} else if option, ok := byLabel[normalized]; ok {
				key = option.key
			} else {
				return nil, false, newExitCodeError(2, fmt.Errorf("setup_gateway_invalid_selection: %s", token))
			}
		}
		if !seen[key] {
			selected = append(selected, key)
			seen[key] = true
		}
	}
	return selected, false, nil
}

var setupTelegramBotTokenPattern = regexp.MustCompile(`^\d+:[A-Za-z0-9_-]{30,}$`)

func runSetupGatewayBubbleTeaWizard(cmd *cobra.Command, cfg config.Config) (setupGatewayWizardResult, error) {
	choices := []setupwizard.Choice{
		{ID: "telegram", Label: "Telegram"},
		{ID: "whatsapp", Label: "WhatsApp"},
		{ID: "discord", Label: "Discord"},
		{ID: "slack", Label: "Slack"},
		{ID: "navivox", Label: "Navivox"},
		{ID: "keep", Label: "Keep current configuration"},
	}
	result, err := setupwizard.New(
		setupwizard.WithInput(os.Stdin),
		setupwizard.WithOutput(cmd.OutOrStdout()),
	).Run(cmd.Context(), setupwizard.Pick("platform", "Messaging Platforms (Gateway)", choices, setupwizard.WithDefaultChoice("telegram")))
	if errors.Is(err, setupwizard.ErrAbort) {
		return setupGatewayWizardResult{}, nil
	}
	if err != nil {
		return setupGatewayWizardResult{}, err
	}
	platform := normalizeSetupChoice(result.Choice("platform"))
	if platform == "" || platform == "keep" {
		return setupGatewayWizardResult{}, nil
	}
	out := setupGatewayWizardResult{SelectedPlatforms: []string{platform}, BubbleTea: true}
	if platform == "telegram" {
		answers, err := runSetupTelegramBubbleTeaWizard(cmd, cfg.Telegram)
		if err != nil {
			return setupGatewayWizardResult{}, err
		}
		out.Telegram = &answers
	}
	return out, nil
}

func runSetupTelegramBubbleTeaWizard(cmd *cobra.Command, cfg config.TelegramCfg) (setupTelegramGatewayAnswers, error) {
	defaultPolicy := "allowlist"
	if cfg.GuestMode {
		defaultPolicy = "open"
	} else if len(cfg.AllowedUserIDs) == 0 && cfg.FirstRunDiscovery {
		defaultPolicy = "pairing"
	}
	result, err := setupwizard.New(
		setupwizard.WithInput(os.Stdin),
		setupwizard.WithOutput(cmd.OutOrStdout()),
	).Run(cmd.Context(),
		setupwizard.Password("token", "Telegram bot token from BotFather (blank keeps current)", setupwizard.WithPlaceholder("123456:...")),
		setupwizard.Pick("access_policy", "Telegram access policy", []setupwizard.Choice{
			{ID: "allowlist", Label: "Allowlisted Telegram user IDs"},
			{ID: "pairing", Label: "Pairing/first-run discovery"},
			{ID: "open", Label: "Open access (risky)"},
		}, setupwizard.WithDefaultChoice(defaultPolicy)),
		setupwizard.Text("allowed_users", "Allowed Telegram user IDs (comma-separated; used for allowlist)", setupwizard.WithPlaceholder("6586915095,12345")),
		setupwizard.Text("home_chat_id", "Home channel chat ID (blank to set later with /set-home)", setupwizard.WithPlaceholder("-1001234567890")),
		setupwizard.Text("home_thread_id", "Home channel thread ID (optional)", setupwizard.WithPlaceholder("42")),
		setupwizard.Confirm("apply", "Write these Telegram settings now?"),
	)
	if errors.Is(err, setupwizard.ErrAbort) {
		return setupTelegramGatewayAnswers{Apply: false}, nil
	}
	if err != nil {
		return setupTelegramGatewayAnswers{}, err
	}
	return setupTelegramGatewayAnswers{
		Token:        strings.TrimSpace(result.String("token")),
		AccessPolicy: normalizeSetupChoice(result.Choice("access_policy")),
		AllowedUsers: strings.TrimSpace(result.String("allowed_users")),
		HomeChatID:   strings.TrimSpace(result.String("home_chat_id")),
		HomeThreadID: strings.TrimSpace(result.String("home_thread_id")),
		Apply:        result.Bool("apply"),
	}, nil
}

func applySetupTelegramGatewayAnswers(cmd *cobra.Command, cfg config.TelegramCfg, answers setupTelegramGatewayAnswers) error {
	out := cmd.OutOrStdout()
	token := strings.TrimSpace(answers.Token)
	if token != "" && !setupTelegramBotTokenPattern.MatchString(token) {
		return newExitCodeError(2, fmt.Errorf("setup telegram: invalid bot token format; expected BotFather token like 123456:ABC..."))
	}
	effectiveToken := token
	if effectiveToken == "" {
		effectiveToken = strings.TrimSpace(cfg.BotToken)
	}
	if effectiveToken == "" {
		return newExitCodeError(2, fmt.Errorf("setup telegram: missing bot token; enter a Telegram bot token or configure one before enabling Telegram"))
	}

	accessPolicy := normalizeSetupChoice(answers.AccessPolicy)
	if accessPolicy == "" {
		accessPolicy = "allowlist"
	}
	allowedUsers, err := parseSetupTelegramAllowedUsers(answers.AllowedUsers)
	if err != nil {
		return err
	}

	fmt.Fprintln(out, "Review Telegram gateway changes")
	if token != "" {
		fmt.Fprintln(out, "  .env GORMES_TELEGRAM_BOT_TOKEN=[REDACTED]")
	}
	switch accessPolicy {
	case "allowlist":
		fmt.Fprintf(out, "  config.toml telegram.allowed_user_ids=%d\n", len(allowedUsers))
	case "pairing":
		fmt.Fprintln(out, "  config.toml telegram.first_run_discovery=true")
	case "open":
		fmt.Fprintln(out, "  config.toml telegram.guest_mode=true")
	default:
		return newExitCodeError(2, fmt.Errorf("setup telegram: unsupported access policy %q", answers.AccessPolicy))
	}
	if strings.TrimSpace(answers.HomeChatID) != "" {
		fmt.Fprintf(out, "  config.toml telegram.home_channel.chat_id=%s\n", strings.TrimSpace(answers.HomeChatID))
	}
	if strings.TrimSpace(answers.HomeThreadID) != "" {
		fmt.Fprintf(out, "  config.toml telegram.home_channel.thread_id=%s\n", strings.TrimSpace(answers.HomeThreadID))
	}
	fmt.Fprintln(out, "  group guidance: in BotFather, disable privacy when needed; add the bot as admin; after permission changes, remove and re-add the bot to the group.")
	if !answers.Apply {
		fmt.Fprintln(out, "Telegram setup canceled; no files were written.")
		return nil
	}

	if token != "" {
		envName := config.SecretEnvName("telegram.bot_token")
		if err := config.WriteEnvValue(config.EnvPath(), envName, token); err != nil {
			return fmt.Errorf("setup telegram: write token: %w", err)
		}
		if err := os.Setenv(envName, token); err != nil {
			return fmt.Errorf("setup telegram: activate token: %w", err)
		}
	}
	switch accessPolicy {
	case "allowlist":
		if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.allowed_user_ids", formatSetupInt64CSV(allowedUsers)); err != nil {
			return fmt.Errorf("setup telegram: write allowed users: %w", err)
		}
		if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.guest_mode", "false"); err != nil {
			return fmt.Errorf("setup telegram: write guest mode: %w", err)
		}
	case "pairing":
		if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.first_run_discovery", "true"); err != nil {
			return fmt.Errorf("setup telegram: write discovery config: %w", err)
		}
	case "open":
		if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.guest_mode", "true"); err != nil {
			return fmt.Errorf("setup telegram: write guest mode: %w", err)
		}
	}
	if chatID := strings.TrimSpace(answers.HomeChatID); chatID != "" {
		if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.home_channel.chat_id", chatID); err != nil {
			return fmt.Errorf("setup telegram: write home channel: %w", err)
		}
	}
	if threadID := strings.TrimSpace(answers.HomeThreadID); threadID != "" {
		if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.home_channel.thread_id", threadID); err != nil {
			return fmt.Errorf("setup telegram: write home channel thread: %w", err)
		}
	}
	fmt.Fprintln(out, "Telegram gateway channel configured.")
	if strings.TrimSpace(answers.HomeChatID) == "" {
		fmt.Fprintln(out, "Telegram home channel can be set later with /set-home after the bot is in the target chat.")
	}
	return nil
}

func parseSetupTelegramAllowedUsers(value string) ([]int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	out := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, newExitCodeError(2, fmt.Errorf("setup telegram: invalid allowed user ID %q", part))
		}
		out = append(out, id)
	}
	return out, nil
}

func formatSetupInt64CSV(values []int64) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.FormatInt(value, 10))
	}
	return strings.Join(parts, ",")
}

func compactStringsSetup(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = normalizeSetupChoice(value)
		if value == "" || seen[value] {
			continue
		}
		out = append(out, value)
		seen[value] = true
	}
	return out
}

func runSetupGatewayPlatformRowBacked(cmd *cobra.Command, platform string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "setup_gateway_platform_row_backed: platform=%s recommended_command=\"gormes setup gateway\"\n", platform)
	return nil
}

func runSetupGatewayPlatform(cmd *cobra.Command, platform string, runWhatsAppSetup func(*cobra.Command) error) error {
	switch normalizeSetupChoice(platform) {
	case "telegram":
		return runSetupTelegramGatewayPlatform(cmd)
	case "discord":
		return runSetupDiscordGatewayPlatform(cmd)
	case "slack":
		return runSetupSlackGatewayPlatform(cmd)
	case "whatsapp":
		return runWhatsAppSetup(cmd)
	default:
		return runSetupGatewayPlatformRowBacked(cmd, platform)
	}
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

func runSetupTelegramGatewayPlatform(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("setup telegram: load config: %w", err)
	}
	token, err := promptSecret(cmd, "Telegram bot token (stored in .env, blank to keep current): ")
	if err != nil {
		return err
	}
	if token != "" {
		envName := config.SecretEnvName("telegram.bot_token")
		if err := config.WriteEnvValue(config.EnvPath(), envName, token); err != nil {
			return fmt.Errorf("setup telegram: write token: %w", err)
		}
		if err := os.Setenv(envName, token); err != nil {
			return fmt.Errorf("setup telegram: activate token: %w", err)
		}
	}
	if strings.TrimSpace(token) == "" && strings.TrimSpace(cfg.Telegram.BotToken) == "" {
		return newExitCodeError(2, fmt.Errorf("setup telegram: missing bot token; enter a Telegram bot token or configure one before enabling Telegram"))
	}

	chatID, err := promptString(cmd, "Allowed chat ID (blank for first-run discovery): ", "")
	if err != nil {
		return err
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.first_run_discovery", "true"); err != nil {
			return fmt.Errorf("setup telegram: write discovery config: %w", err)
		}
		fmt.Fprintln(out, "Telegram gateway channel configured for first-run discovery.")
		return nil
	}
	if _, err := strconv.ParseInt(chatID, 10, 64); err != nil {
		return newExitCodeError(2, fmt.Errorf("setup telegram: invalid allowed chat ID"))
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.allowed_chat_id", chatID); err != nil {
		return fmt.Errorf("setup telegram: write allowed chat ID: %w", err)
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.first_run_discovery", "false"); err != nil {
		return fmt.Errorf("setup telegram: write discovery config: %w", err)
	}
	fmt.Fprintln(out, "Telegram gateway channel configured.")
	return nil
}

func runSetupDiscordGatewayPlatform(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("setup discord: load config: %w", err)
	}
	token, err := promptSecret(cmd, "Discord bot token (stored in .env, blank to keep current): ")
	if err != nil {
		return err
	}
	if token != "" {
		envName := config.SecretEnvName("discord.token")
		if err := config.WriteEnvValue(config.EnvPath(), envName, token); err != nil {
			return fmt.Errorf("setup discord: write token: %w", err)
		}
		if err := os.Setenv(envName, token); err != nil {
			return fmt.Errorf("setup discord: activate token: %w", err)
		}
	}
	if strings.TrimSpace(token) == "" && strings.TrimSpace(cfg.Discord.Token) == "" {
		return newExitCodeError(2, fmt.Errorf("setup discord: missing bot token; enter a Discord bot token or configure one before enabling Discord"))
	}

	channelID, err := promptString(cmd, "Allowed channel ID (blank for first-run discovery): ", "")
	if err != nil {
		return err
	}
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		if err := config.WriteTOMLValue(config.ConfigPath(), "discord.first_run_discovery", "true"); err != nil {
			return fmt.Errorf("setup discord: write discovery config: %w", err)
		}
		fmt.Fprintln(out, "Discord gateway channel configured for first-run discovery.")
		return nil
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "discord.allowed_channel_id", channelID); err != nil {
		return fmt.Errorf("setup discord: write allowed channel ID: %w", err)
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "discord.first_run_discovery", "false"); err != nil {
		return fmt.Errorf("setup discord: write discovery config: %w", err)
	}
	fmt.Fprintln(out, "Discord gateway channel configured.")
	return nil
}

func runSetupSlackGatewayPlatform(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("setup slack: load config: %w", err)
	}
	botToken, err := promptSecret(cmd, "Slack bot token (xoxb, stored in .env, blank to keep current): ")
	if err != nil {
		return err
	}
	if botToken != "" {
		envName := config.SecretEnvName("slack.bot_token")
		if err := config.WriteEnvValue(config.EnvPath(), envName, botToken); err != nil {
			return fmt.Errorf("setup slack: write bot token: %w", err)
		}
		if err := os.Setenv(envName, botToken); err != nil {
			return fmt.Errorf("setup slack: activate bot token: %w", err)
		}
	}
	appToken, err := promptSecret(cmd, "Slack app token (xapp, stored in .env, blank to keep current): ")
	if err != nil {
		return err
	}
	if appToken != "" {
		envName := config.SecretEnvName("slack.app_token")
		if err := config.WriteEnvValue(config.EnvPath(), envName, appToken); err != nil {
			return fmt.Errorf("setup slack: write app token: %w", err)
		}
		if err := os.Setenv(envName, appToken); err != nil {
			return fmt.Errorf("setup slack: activate app token: %w", err)
		}
	}

	effectiveBotToken := strings.TrimSpace(botToken)
	if effectiveBotToken == "" {
		effectiveBotToken = strings.TrimSpace(cfg.Slack.BotToken)
	}
	effectiveAppToken := strings.TrimSpace(appToken)
	if effectiveAppToken == "" {
		effectiveAppToken = strings.TrimSpace(cfg.Slack.AppToken)
	}
	if effectiveBotToken == "" || effectiveAppToken == "" {
		return newExitCodeError(2, fmt.Errorf("setup slack: missing Slack tokens; enter both bot and app tokens, or configure both before enabling Slack"))
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "slack.enabled", "true"); err != nil {
		return fmt.Errorf("setup slack: write enabled config: %w", err)
	}
	channelID, err := promptString(cmd, "Allowed channel ID (blank for first-run discovery): ", "")
	if err != nil {
		return err
	}
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		if err := config.WriteTOMLValue(config.ConfigPath(), "slack.first_run_discovery", "true"); err != nil {
			return fmt.Errorf("setup slack: write discovery config: %w", err)
		}
		fmt.Fprintln(out, "Slack gateway channel configured for first-run discovery.")
		return nil
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "slack.allowed_channel_id", channelID); err != nil {
		return fmt.Errorf("setup slack: write allowed channel ID: %w", err)
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "slack.first_run_discovery", "false"); err != nil {
		return fmt.Errorf("setup slack: write discovery config: %w", err)
	}
	fmt.Fprintln(out, "Slack gateway channel configured.")
	return nil
}

func runSetupToolsSection(cmd *cobra.Command, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	doc, toolCfg, err := loadSetupToolsConfig(config.ConfigPath())
	if err != nil {
		return err
	}
	status, err := toolCfg.PlatformStatus("cli")
	if err != nil {
		return err
	}
	options, err := setupToolOptions()
	if err != nil {
		return err
	}
	selected := stringSet(status.RuntimeToolsets)

	fmt.Fprintln(out, "Tools for CLI")
	fmt.Fprintln(out)
	for i, option := range options {
		marker := "[ ]"
		if selected[option.key] {
			marker = "[x]"
		}
		fmt.Fprintf(out, "  %2d. %s %-28s %-16s %s\n", i+1, marker, option.label, option.key, option.description)
	}
	if nonInteractive {
		fmt.Fprintln(out, "\nSkipped (keeping current tool selection).")
		return nil
	}
	fmt.Fprintln(out)
	selection, err := promptString(cmd, "Toolsets (comma-separated numbers or keys, blank to keep current): ", "")
	if err != nil {
		return err
	}
	chosen, err := parseSetupToolSelection(selection, options, status.RuntimeToolsets)
	if err != nil {
		return err
	}
	report, err := toolCfg.SavePlatformSelection("cli", chosen)
	if err != nil {
		return err
	}
	doc["platform_toolsets"] = toolCfg.PlatformToolsets
	if err := writeSetupToolsConfig(config.ConfigPath(), doc); err != nil {
		return err
	}
	fmt.Fprintf(out, "Saved CLI tool configuration: %s\n", strings.Join(report.PersistedToolsets, ", "))
	for _, issue := range report.Issues {
		if issue.Platform == "cli" || issue.Platform == "" {
			fmt.Fprintf(out, "setup_tools_issue: kind=%s toolset=%s detail=%s\n", issue.Kind, issue.Toolset, issue.Detail)
		}
	}
	renderSetupToolsProviderRows(out, chosen)
	return nil
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

	progress, err := promptString(cmd, fmt.Sprintf("Tool progress mode [%s]: ", toolProgress), toolProgress)
	if err != nil {
		return err
	}
	progress = normalizeSetupChoice(progress)
	if isKnownToolProgressMode(progress) {
		if err := config.WriteTOMLValue(config.ConfigPath(), "display.tool_progress", progress); err != nil {
			return err
		}
		fmt.Fprintf(out, "Tool progress set to: %s\n", progress)
	} else {
		fmt.Fprintf(out, "setup_agent_value_ignored: tool_progress=%q\n", progress)
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

	policy, err := promptString(cmd, fmt.Sprintf("Session reset policy [%s]: ", sessionPolicy), sessionPolicy)
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

type setupChoice struct {
	value string
	label string
}

func ttsProviderOptions() []setupChoice {
	return []setupChoice{
		{"edge", "Edge TTS (free, cloud-based, no setup needed)"},
		{"elevenlabs", "ElevenLabs (premium quality, needs API key)"},
		{"openai", "OpenAI TTS (good quality, needs API key)"},
		{"xai", "xAI TTS (Grok voices, needs API key)"},
		{"minimax", "MiniMax TTS (high quality with voice cloning, needs API key)"},
		{"mistral", "Mistral Voxtral TTS (multilingual, native Opus, needs API key)"},
		{"gemini", "Google Gemini TTS (30 prebuilt voices, prompt-controllable, needs API key)"},
		{"neutts", "NeuTTS (local on-device, free, model download)"},
		{"keep", "Keep current"},
	}
}

func terminalBackendOptions() []setupChoice {
	return []setupChoice{
		{"local", "Local - run directly on this machine (default)"},
		{"docker", "Docker - isolated container with configurable resources"},
		{"modal", "Modal - serverless cloud sandbox"},
		{"ssh", "SSH - run on a remote machine"},
		{"daytona", "Daytona - persistent cloud development environment"},
		{"singularity", "Singularity/Apptainer - HPC-friendly container"},
		{"keep", "Keep current"},
	}
}

type setupToolOption struct {
	key         string
	label       string
	description string
}

func setupToolOptions() ([]setupToolOption, error) {
	report, err := cli.EffectiveToolsetPickerOptions(plugins.Inventory{})
	if err != nil {
		return nil, err
	}
	out := make([]setupToolOption, 0, len(report.Options))
	for _, option := range report.Options {
		out = append(out, setupToolOption{
			key:         option.Key,
			label:       option.Label,
			description: option.Description,
		})
	}
	return out, nil
}

func loadSetupToolsConfig(path string) (map[string]any, cli.PlatformToolsetConfig, error) {
	doc := map[string]any{}
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg, _ := cli.ParsePlatformToolsetConfig(doc)
			return doc, cfg, nil
		}
		return nil, cli.PlatformToolsetConfig{}, fmt.Errorf("setup tools: read %s: %w", path, err)
	}
	if err := toml.Unmarshal(body, &doc); err != nil {
		return nil, cli.PlatformToolsetConfig{}, fmt.Errorf("setup tools: parse %s: %w", path, err)
	}
	cfg, _ := cli.ParsePlatformToolsetConfig(doc)
	return doc, cfg, nil
}

func writeSetupToolsConfig(path string, doc map[string]any) error {
	body, err := toml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("setup tools: marshal config: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("setup tools: mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".config.toml.*")
	if err != nil {
		return fmt.Errorf("setup tools: tempfile: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return fmt.Errorf("setup tools: write temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("setup tools: chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("setup tools: close temp: %w", err)
	}
	if _, err := toolspkg.AtomicReplace(tmpName, path, toolspkg.AtomicReplaceOptions{FirstWriteMode: 0o600}); err != nil {
		return fmt.Errorf("setup tools: rename config: %w", err)
	}
	return nil
}

func parseSetupToolSelection(input string, options []setupToolOption, current []string) ([]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return append([]string(nil), current...), nil
	}
	byKey := make(map[string]setupToolOption, len(options))
	for _, option := range options {
		byKey[option.key] = option
	}
	var selected []string
	for _, token := range strings.FieldsFunc(input, setupSelectionSeparator) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if index, err := strconv.Atoi(token); err == nil {
			if index < 1 || index > len(options) {
				return nil, newExitCodeError(2, fmt.Errorf("setup_tools_invalid_selection: %s", token))
			}
			selected = append(selected, options[index-1].key)
			continue
		}
		key := normalizeSetupChoice(token)
		key = strings.ReplaceAll(key, "-", "_")
		if option, ok := byKey[key]; ok {
			selected = append(selected, option.key)
			continue
		}
		selected = append(selected, token)
	}
	return selected, nil
}

func setupSelectionSeparator(r rune) bool {
	return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == ';'
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

type setupToolsProviderRow struct {
	Toolset string
	Kind    string
	Label   string
}

var setupToolsProviderRows = map[string][]setupToolsProviderRow{
	"web": {
		{Toolset: "web", Kind: "web", Label: "Web search and extraction"},
	},
	"browser": {
		{Toolset: "browser", Kind: "browser", Label: "Browser backend"},
	},
	"image_gen": {
		{Toolset: "image_gen", Kind: "image_gen", Label: "Image generation provider"},
	},
	"rl": {
		{Toolset: "rl", Kind: "rl", Label: "RL training provider"},
	},
	"tts": {
		{Toolset: "tts", Kind: "tts", Label: "Voice/TTS provider"},
	},
	"skills": {
		{Toolset: "skills", Kind: "github_skills_hub", Label: "GitHub Skills Hub"},
	},
	"memory": {
		{Toolset: "memory", Kind: "honcho", Label: "Honcho/Goncho memory provider"},
	},
	"homeassistant": {
		{Toolset: "homeassistant", Kind: "homeassistant", Label: "Home Assistant credentials"},
	},
}

func renderSetupToolsProviderRows(out io.Writer, selected []string) {
	selectedSet := stringSet(selected)
	printedHeader := false
	for _, option := range providerRowToolsetOrder() {
		if !selectedSet[option] {
			continue
		}
		for _, row := range setupToolsProviderRows[option] {
			if !printedHeader {
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Provider/API key setup")
				printedHeader = true
			}
			fmt.Fprintf(out, "  setup_tools_provider_row_backed: toolset=%s provider=%s label=%s\n", row.Toolset, row.Kind, row.Label)
		}
	}
}

func providerRowToolsetOrder() []string {
	return []string{"web", "browser", "image_gen", "rl", "tts", "skills", "memory", "homeassistant"}
}

func terminalBackendLabel(value string) string {
	switch normalizeSetupChoice(value) {
	case "local":
		return "Local"
	case "docker":
		return "Docker"
	case "modal":
		return "Modal"
	case "ssh":
		return "SSH"
	case "daytona":
		return "Daytona"
	case "singularity", "apptainer":
		return "Singularity/Apptainer"
	default:
		return value
	}
}

func ttsProviderLabel(value string) string {
	switch normalizeSetupChoice(value) {
	case "edge":
		return "Edge TTS"
	case "openai":
		return "OpenAI TTS"
	default:
		for _, option := range ttsProviderOptions() {
			if option.value == normalizeSetupChoice(value) {
				return strings.Split(option.label, " (")[0]
			}
		}
		return value
	}
}

func normalizeSetupChoice(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "-", "_")
	if value == "apptainer" {
		return "singularity"
	}
	return value
}

func parsePositiveInt(value string) (int, bool) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func parseThreshold(value string) (float64, bool) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || parsed < 0.5 || parsed > 0.95 {
		return 0, false
	}
	return parsed, true
}

func isKnownToolProgressMode(value string) bool {
	switch value {
	case "off", "new", "all", "verbose":
		return true
	default:
		return false
	}
}

func isKnownSessionResetPolicy(value string) bool {
	switch value {
	case "inactivity", "daily", "manual", "off", "none":
		return true
	default:
		return false
	}
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
	fmt.Fprintf(cmd.ErrOrStderr(), "setup_section_unsupported: section=%s available=%s\n", section, setupSectionList())
	fmt.Fprintln(cmd.ErrOrStderr(), "Implemented sections: provider, model, agent, workspace, bindings, tts, terminal, gateway, and tools.")
	fmt.Fprintln(cmd.ErrOrStderr(), "setup_section_row_backed: recommended_command=\"gormes setup\"")
	return newExitCodeError(2, fmt.Errorf("setup_section_unsupported: %s", section))
}

func setupSectionList() string {
	return strings.Join(setupSections, "|")
}

func setupSectionOwnership(section string) string {
	switch normalizeSetupChoice(section) {
	case "model", "tts", "terminal", "gateway", "tools", "agent":
		return "hermes_owned"
	case "provider", "workspace", "bindings", "onboard":
		return "gormes_owned_extension"
	default:
		return "unknown"
	}
}
