package main

import (
	"errors"
	"fmt"
	"os"
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
		Short:        "Guided interactive setup — provider, model, and more",
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
			case "provider":
				return runSetupProviderSection(cmd, seams, nonInteractive)
			case "model":
				return runSetupModelSection(cmd, seams, nonInteractive)
			case "agent", "workspace":
				return runSetupAgentSection(cmd, section, seams, nonInteractive)
			case "bindings":
				return runSetupBindingsSection(cmd, seams, nonInteractive)
			case "tts", "terminal", "gateway", "tools":
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
	fmt.Fprintln(cmd.OutOrStdout(), "\nQuick start: run `gormes setup provider` to configure your LLM provider.")
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

func setupSectionUnsupported(cmd *cobra.Command, section string) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "setup_section_unsupported: section=%s available=%s\n", section, setupSectionList())
	fmt.Fprintln(cmd.ErrOrStderr(), "Implemented sections: provider, model, agent, workspace, and bindings. Use `gormes config edit` for other setup surfaces.")
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
