package gormescli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

// SetupWizardHeader prints the wizard header box.
func SetupWizardHeader(cmd *cobra.Command) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "┌─────────────────────────────────────────────────────────┐")
	fmt.Fprintln(out, "│              Gormes Agent Setup Wizard                  │")
	fmt.Fprintln(out, "├─────────────────────────────────────────────────────────┤")
	fmt.Fprintln(out, "│  Configure your Gormes Agent installation.              │")
	fmt.Fprintln(out, "│  Press Ctrl+C at any time to exit.                      │")
	fmt.Fprintln(out, "└─────────────────────────────────────────────────────────┘")
	fmt.Fprintln(out)
}

// SetupSummary prints the post-setup summary.
func SetupSummary(cmd *cobra.Command) {
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

// SetupInferenceProviderBlock prints the inference provider info block.
func SetupInferenceProviderBlock(cmd *cobra.Command, current cli.ProviderModel, providerLabel string) {
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

// SetupProviderCredentialChoices prints the provider credential choices.
func SetupProviderCredentialChoices(cmd *cobra.Command) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "    1. Use existing credentials")
	fmt.Fprintln(out, "    2. Reauthenticate (new OAuth login)")
	fmt.Fprintln(out, "    3. Cancel")
	fmt.Fprintln(out)
}

// SetupProviderDisplayLabel returns a human-readable label for a provider ID.
func SetupProviderDisplayLabel(provider string) string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
		return ""
	}
	switch provider {
	case config.CodexOAuthProvider:
		return "OpenAI Codex"
	case config.AnthropicProvider:
		return "Anthropic"
	case config.NousOAuthProvider:
		return "Nous"
	case "openrouter":
		return "OpenRouter"
	case "google-gemini-cli":
		return "Google Gemini CLI"
	case "qwen-oauth":
		return "Qwen OAuth"
	}
	if entry, ok := llm.ResolveProviderManifestEntry(provider); ok {
		provider = entry.ID
	}
	parts := strings.Fields(strings.ReplaceAll(provider, "-", " "))
	for i, part := range parts {
		parts[i] = SetupTitleProviderWord(part)
	}
	return strings.Join(parts, " ")
}

// SetupTitleProviderWord title-cases a single word for provider display.
func SetupTitleProviderWord(word string) string {
	switch strings.ToLower(word) {
	case "ai":
		return "AI"
	case "api":
		return "API"
	case "cli":
		return "CLI"
	case "oauth":
		return "OAuth"
	case "":
		return ""
	default:
		return strings.ToUpper(word[:1]) + word[1:]
	}
}

// SetupProviderCredentialStatus returns a status marker for provider credentials.
func SetupProviderCredentialStatus(status cli.ProviderAuthStatus, err error) string {
	if err != nil || status.Status == cli.AuthStatusError {
		return "!"
	}
	if status.Authenticated || status.Status == cli.AuthStatusLoggedIn {
		return "✓"
	}
	return "missing"
}