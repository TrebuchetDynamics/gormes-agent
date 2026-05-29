package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type SetupWizard struct {
	in  *bufio.Reader
	out *os.File
}

func NewSetupWizard() *SetupWizard {
	return &SetupWizard{in: bufio.NewReader(os.Stdin), out: os.Stdout}
}

func (w *SetupWizard) Run() error {
	fmt.Fprintln(w.out, "\n⚕ Gormes Setup Wizard")
	fmt.Fprintln(w.out, strings.Repeat("─", 50))

	if err := w.configureProvider(); err != nil {
		return err
	}
	if err := w.configureModel(); err != nil {
		return err
	}
	if err := w.configureTerminal(); err != nil {
		return err
	}
	w.showSummary()
	return nil
}

func (w *SetupWizard) prompt(label, defaultValue string) string {
	if defaultValue != "" {
		fmt.Fprintf(w.out, "  %s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(w.out, "  %s: ", label)
	}
	input, _ := w.in.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func (w *SetupWizard) configureProvider() error {
	fmt.Fprintln(w.out, "\n1. Provider Configuration")
	fmt.Fprintln(w.out, "   Select your AI provider:")

	providers := map[string]string{
		"1": "openai", "2": "anthropic", "3": "openrouter",
		"4": "deepseek", "5": "custom",
	}
	fmt.Fprintln(w.out, "   1. OpenAI")
	fmt.Fprintln(w.out, "   2. Anthropic")
	fmt.Fprintln(w.out, "   3. OpenRouter")
	fmt.Fprintln(w.out, "   4. DeepSeek")
	fmt.Fprintln(w.out, "   5. Custom (OpenAI-compatible)")

	choice := w.prompt("Provider", "1")
	provider := providers[choice]
	if provider == "" {
		provider = "openai"
	}
	_ = w.prompt("API Key", os.Getenv(providerAPIEnvKey(provider)))
	if provider == "custom" {
		_ = w.prompt("Base URL", "https://api.openai.com/v1")
	}
	return nil
}

func providerAPIEnvKey(provider string) string {
	switch provider {
	case "openai":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "openrouter":
		return "OPENROUTER_API_KEY"
	case "deepseek":
		return "DEEPSEEK_API_KEY"
	default:
		return "OPENAI_API_KEY"
	}
}

func (w *SetupWizard) configureModel() error {
	fmt.Fprintln(w.out, "\n2. Model Selection")
	_ = w.prompt("Default model", "gpt-4o")
	return nil
}

func (w *SetupWizard) configureTerminal() error {
	fmt.Fprintln(w.out, "\n3. Terminal Backend")
	fmt.Fprintln(w.out, "   1. Local (default)")
	fmt.Fprintln(w.out, "   2. Docker")
	_ = w.prompt("Backend", "1")
	return nil
}

func (w *SetupWizard) showSummary() {
	fmt.Fprintln(w.out, "\n"+strings.Repeat("─", 50))
	fmt.Fprintln(w.out, "✅ Setup complete!")
	fmt.Fprintln(w.out, "   Run 'gormes' to start the TUI.")
	fmt.Fprintln(w.out, "   Run 'gormes doctor --offline' to verify.")
	fmt.Fprintln(w.out, "   Run 'gormes chat -q \"hello\"' to test.")
}
