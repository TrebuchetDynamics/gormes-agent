package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/spf13/cobra"
)

var errSetupRequiresTTY = errors.New("setup_requires_tty")

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
	IsTTY             func() bool
	RunModelPicker    func(*cobra.Command) error
	LoadCurrentModel  func() (cli.ProviderModel, error)
	ChooseSetupAction func(*cobra.Command, []setupMenuOption, int) (setupAction, error)
	RunFullWizard     func(*cobra.Command, bool) error
}

type setupAction string

const (
	setupActionQuick         setupAction = "quick"
	setupActionFull          setupAction = "full"
	setupActionModelProvider setupAction = "model_provider"
	setupActionTerminal      setupAction = "terminal"
	setupActionGateway       setupAction = "gateway"
	setupActionTools         setupAction = "tools"
	setupActionAgent         setupAction = "agent"
	setupActionExit          setupAction = "exit"
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
	if seams.ChooseSetupAction == nil {
		seams.ChooseSetupAction = promptSetupAction
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
	cmd := &cobra.Command{
		Use:          "setup [section]",
		Short:        "Guided interactive setup — provider, model, and more",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if quick {
				return runSetupQuick(cmd, seams, nonInteractive || !seams.IsTTY())
			}
			if reset || reconfigure {
				return seams.RunFullWizard(cmd, true)
			}
			if len(args) == 0 {
				return runSetupRoot(cmd, seams, nonInteractive)
			}
			section := strings.ToLower(strings.TrimSpace(args[0]))
			switch section {
			case "provider":
				return runSetupProviderSection(cmd, seams, nonInteractive)
			case "model":
				return runSetupModelSection(cmd, seams, nonInteractive)
			case "agent":
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
				return runSetupGatewaySection(cmd, nonInteractive)
			case "tools":
				return runSetupToolsSection(cmd, nonInteractive)
			default:
				return setupSectionUnsupported(cmd, section)
			}
		},
	}
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "use defaults/env and never prompt")
	cmd.Flags().BoolVar(&reset, "reset", false, "run the full setup wizard in compatibility mode")
	cmd.Flags().BoolVar(&reconfigure, "reconfigure", false, "re-run the full setup wizard in compatibility mode")
	cmd.Flags().BoolVar(&quick, "quick", false, "configure missing setup items only")
	return cmd
}

func defaultSetupCommandSeams() setupCommandSeams {
	return setupCommandSeams{
		IsTTY:             isStdinTTY,
		LoadCurrentModel:  defaultSetupLoadCurrentModel,
		ChooseSetupAction: promptSetupAction,
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
		return runSetupQuick(cmd, seams, nonInteractive)
	case setupActionFull:
		return seams.RunFullWizard(cmd, nonInteractive)
	case setupActionModelProvider:
		return runSetupModelSection(cmd, seams, nonInteractive)
	case setupActionTerminal:
		return runSetupTerminalSection(cmd, nonInteractive)
	case setupActionGateway:
		return runSetupGatewaySection(cmd, nonInteractive)
	case setupActionTools:
		return runSetupToolsSection(cmd, nonInteractive)
	case setupActionAgent:
		return runSetupAgentSettingsSection(cmd, nonInteractive)
	case setupActionExit:
		return nil
	default:
		return setupSectionUnsupported(cmd, string(action))
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
		{Action: setupActionExit, Label: "Exit"},
	}
}

func printSetupTopLevelMenu(cmd *cobra.Command, options []setupMenuOption, defaultOption int) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "What would you like to do?")
	fmt.Fprintln(out, "  ↑↓ navigate  ENTER/SPACE select  ESC cancel")
	fmt.Fprintln(out)
	for i, option := range options {
		prefix := "   (○)"
		if i == defaultOption {
			prefix = " → (●)"
		}
		fmt.Fprintf(out, "%s %s\n", prefix, option.Label)
	}
	fmt.Fprintln(out)
}

func promptSetupAction(cmd *cobra.Command, options []setupMenuOption, defaultOption int) (setupAction, error) {
	defaultText := strconv.Itoa(defaultOption + 1)
	answer, err := promptString(cmd, fmt.Sprintf("Select option [%s]: ", defaultText), defaultText)
	if err != nil {
		return "", err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
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

func runSetupQuick(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Quick Setup - configure missing items only")
	current, err := seams.LoadCurrentModel()
	if err != nil {
		return fmt.Errorf("quick setup: load current model: %w", err)
	}
	if strings.TrimSpace(current.Provider) == "" || strings.TrimSpace(current.Model) == "" {
		fmt.Fprintln(out, "Model/provider defaults are missing.")
		return runSetupModelSection(cmd, seams, nonInteractive)
	}
	fmt.Fprintf(out, "Current model/provider: %s via %s\n", current.Model, current.Provider)
	fmt.Fprintln(out, "No missing core setup items detected.")
	printSetupSummary(cmd)
	return nil
}

func runSetupFullWizard(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	printSetupWizardHeader(cmd)
	if nonInteractive {
		if err := runSetupModelSection(cmd, seams, true); err != nil {
			return err
		}
		for _, run := range []func(*cobra.Command, bool) error{
			runSetupTTSSection,
			runSetupTerminalSection,
			runSetupAgentSettingsSection,
			runSetupGatewaySection,
			runSetupToolsSection,
		} {
			if err := run(cmd, true); err != nil {
				return err
			}
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
	if err := runSetupGatewaySection(cmd, false); err != nil {
		return err
	}
	if err := runSetupToolsSection(cmd, false); err != nil {
		return err
	}
	printSetupSummary(cmd)
	return nil
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
	fmt.Fprintf(out, "All your files are in %s/:\n", config.GormesHome())
	fmt.Fprintln(out)
	fmt.Fprintf(out, "   Settings:  %s\n", config.ConfigPath())
	fmt.Fprintf(out, "   API Keys:  %s\n", config.EnvPath())
	fmt.Fprintf(out, "   Data:      %s/cron/, sessions/, logs/\n", config.GormesHome())
	fmt.Fprintln(out)
	fmt.Fprintln(out, "To edit your configuration:")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "   gormes setup          Re-run the setup wizard")
	fmt.Fprintln(out, "   gormes setup model    Change model/provider")
	fmt.Fprintln(out, "   gormes setup terminal Change terminal backend")
	fmt.Fprintln(out, "   gormes setup gateway  Configure messaging")
	fmt.Fprintln(out, "   gormes setup tools    Configure tool providers")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "   gormes config         View current settings")
	fmt.Fprintln(out, "   gormes config edit    Open config in your editor")
	fmt.Fprintln(out, "   gormes config set <key> <value>")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Ready to go:")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "   gormes              Start chatting")
	fmt.Fprintln(out, "   gormes gateway      Start messaging gateway")
	fmt.Fprintln(out, "   gormes doctor       Check for issues")
	fmt.Fprintln(out)
}

func runSetupProviderSection(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	fmt.Fprintln(cmd.OutOrStdout(), "Setup section: provider")
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

	apiKey, err := promptString(cmd, "API key: ", "")
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
	fmt.Fprintf(out, "API key:  %s***%s\n", apiKey[:min(4, len(apiKey))], apiKey[max(len(apiKey)-4, 0):])
	if model != "" {
		fmt.Fprintf(out, "Model:    %s\n", model)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Test it:  gormes --oneshot \"hello\"")
	return nil
}

func promptString(cmd *cobra.Command, prompt, defaultVal string) (string, error) {
	fmt.Fprint(cmd.OutOrStdout(), prompt)
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

func runSetupGatewaySection(cmd *cobra.Command, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Messaging Platforms")
	fmt.Fprintln(out, "Which platforms would you like to set up?")
	fmt.Fprintln(out)
	for _, option := range []struct {
		value string
		label string
	}{
		{"telegram", "Telegram"},
		{"discord", "Discord"},
		{"slack", "Slack"},
	} {
		fmt.Fprintf(out, "  [ ] %s\n", option.label)
		_ = option.value
	}
	if nonInteractive {
		fmt.Fprintln(out, "\nSkipped (no messaging platforms selected).")
		fmt.Fprintln(out, "Run `gormes setup gateway` interactively or configure credentials with `gormes config edit`.")
		fmt.Fprintln(out, "Start messaging with: gormes gateway")
		return nil
	}

	selection, err := promptString(cmd, "Platforms (comma-separated, blank to skip): ", "")
	if err != nil {
		return err
	}
	selected := splitSetupSelection(selection)
	if len(selected) == 0 {
		fmt.Fprintln(out, "No messaging platforms selected.")
		return nil
	}
	for _, platform := range selected {
		switch platform {
		case "telegram":
			fmt.Fprintln(out, "Telegram selected. Set GORMES_TELEGRAM_BOT_TOKEN and telegram.allowed_chat_id.")
		case "discord":
			fmt.Fprintln(out, "Discord selected. Set GORMES_DISCORD_TOKEN and discord.allowed_channel_id.")
		case "slack":
			if err := config.WriteTOMLValue(config.ConfigPath(), "slack.enabled", "true"); err != nil {
				return err
			}
			fmt.Fprintln(out, "Slack selected. Set GORMES_SLACK_BOT_TOKEN and GORMES_SLACK_APP_TOKEN.")
		default:
			fmt.Fprintf(cmd.ErrOrStderr(), "setup_gateway_platform_row_backed: platform=%s\n", platform)
			return newExitCodeError(2, fmt.Errorf("setup_gateway_platform_row_backed: %s", platform))
		}
	}
	fmt.Fprintln(out, "Start messaging with: gormes gateway")
	return nil
}

func runSetupToolsSection(cmd *cobra.Command, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Tools for CLI")
	fmt.Fprintln(out)
	for _, option := range setupToolOptions() {
		marker := "[✓]"
		if option.defaultOff {
			marker = "[ ]"
		}
		suffix := ""
		if option.noAPIKey {
			suffix = "  [no API key]"
		}
		fmt.Fprintf(out, "  %s %s  (%s)%s\n", marker, option.label, option.tools, suffix)
	}
	if nonInteractive {
		fmt.Fprintln(out, "\nSkipped (keeping current tool selection).")
		return nil
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Tool selection is currently runtime-manifest backed. Use `gormes config edit` for platform-specific toolsets.")
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
	label      string
	tools      string
	defaultOff bool
	noAPIKey   bool
}

func setupToolOptions() []setupToolOption {
	return []setupToolOption{
		{label: "Web Search & Scraping", tools: "web_search, web_extract", noAPIKey: true},
		{label: "Browser Automation", tools: "navigate, click, type, scroll"},
		{label: "Terminal & Processes", tools: "terminal, process"},
		{label: "File Operations", tools: "read, write, patch, search"},
		{label: "Code Execution", tools: "execute_code"},
		{label: "Vision / Image Analysis", tools: "vision_analyze"},
		{label: "Image Generation", tools: "image_generate", noAPIKey: true},
		{label: "Mixture of Agents", tools: "mixture_of_agents", defaultOff: true, noAPIKey: true},
		{label: "Text-to-Speech", tools: "text_to_speech"},
		{label: "Skills", tools: "list, view, manage"},
		{label: "Task Planning", tools: "todo"},
		{label: "Memory", tools: "persistent memory across sessions"},
		{label: "Session Search", tools: "search past conversations"},
		{label: "Clarifying Questions", tools: "clarify"},
		{label: "Task Delegation", tools: "delegate_task"},
		{label: "Cron Jobs", tools: "create/list/update/pause/resume/run"},
		{label: "Cross-Platform Messaging", tools: "send_message"},
		{label: "RL Training", tools: "Tinker-Atropos training tools", defaultOff: true, noAPIKey: true},
		{label: "Home Assistant", tools: "smart home device control", noAPIKey: true},
	}
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

func splitSetupSelection(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	})
	var out []string
	for _, field := range fields {
		normalized := normalizeSetupChoice(field)
		if normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
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
	return newExitCodeError(2, fmt.Errorf("setup_section_unsupported: %s", section))
}

func setupSectionList() string {
	return strings.Join(setupSections, "|")
}
