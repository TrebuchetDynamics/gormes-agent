package firstrun

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	gatewaymodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway"
)

// BuildPlanFromConfig builds first-run readiness from a loaded config.
func BuildPlanFromConfig(cfg config.Config, target cli.SetupTargetID, interactive bool) cli.FirstRunPlan {
	return cli.BuildFirstRunPlan(cli.FirstRunPlanInput{
		Interactive:        interactive,
		Provider:           cfg.Hermes.Provider,
		Endpoint:           cfg.Hermes.Endpoint,
		Model:              cfg.Hermes.Model,
		APIKeyPresent:      gormescli.ConfiguredProviderAuthPresent(cfg),
		Target:             target,
		Channels:           ChannelStates(cfg),
		HermesSourcePath:   DetectHermesMigrationSource(),
		OpenClawSourcePath: DetectOpenClawMigrationSource(),
	})
}

// ChannelStates returns channel-specific setup readiness for first-run guidance.
func ChannelStates(cfg config.Config) []cli.ChannelState {
	return []cli.ChannelState{
		{
			Target:         cli.SetupTargetTelegram,
			Label:          "Telegram",
			Configured:     strings.TrimSpace(cfg.Telegram.BotToken) != "",
			Detail:         gatewaymodule.ConfiguredTelegramStatusDetail(cfg.Telegram),
			SetupCommand:   "gormes setup --quick --target telegram",
			HandoffCommand: "gormes gateway",
		},
		{
			Target:         cli.SetupTargetWhatsApp,
			Label:          "WhatsApp",
			Configured:     strings.EqualFold(strings.TrimSpace(os.Getenv("WHATSAPP_ENABLED")), "true"),
			Detail:         "WhatsApp channel",
			SetupCommand:   "gormes whatsapp --plan",
			HandoffCommand: "gormes gateway",
		},
		{
			Target:         cli.SetupTargetDiscord,
			Label:          "Discord",
			Configured:     cfg.Discord.Enabled(),
			Detail:         "Discord channel",
			SetupCommand:   "gormes setup --quick --target discord",
			HandoffCommand: "gormes gateway",
		},
		{
			Target:         cli.SetupTargetSlack,
			Label:          "Slack",
			Configured:     cfg.Slack.Enabled,
			Detail:         gatewaymodule.ConfiguredSlackStatusDetail(cfg.Slack),
			SetupCommand:   "gormes setup --quick --target slack",
			HandoffCommand: "gormes gateway",
		},
		{
			Target:         cli.SetupTargetNavivox,
			Label:          "Navivox",
			Configured:     cfg.Navivox.Enabled,
			Detail:         gatewaymodule.ConfiguredNavivoxStatusDetail(cfg.Navivox),
			SetupCommand:   "gormes setup --quick --target navivox",
			HandoffCommand: "gormes gateway",
		},
	}
}

// PrintGuidance writes the root-command first-run guidance text.
func PrintGuidance(out io.Writer, plan cli.FirstRunPlan) {
	fmt.Fprintln(out, "Gormes setup needed")
	if plan.Summary != "" {
		fmt.Fprintf(out, "%s\n", plan.Summary)
	}
	for _, step := range plan.MissingSteps {
		if step.Detail == "" {
			continue
		}
		if command := GuidanceCommand(step.Command); command != "" {
			fmt.Fprintf(out, "- %s: %s (run: %s)\n", step.Label, step.Detail, command)
		} else {
			fmt.Fprintf(out, "- %s: %s\n", step.Label, step.Detail)
		}
	}
	if command := GuidanceCommand(plan.NextCommand); command != "" {
		fmt.Fprintf(out, "Next: %s\n", command)
	}
	fmt.Fprintln(out, "Non-interactive mode will not prompt.")
}

// GuidanceCommand normalizes a setup command for guidance output.
func GuidanceCommand(command string) string { return strings.TrimSpace(command) }

// DetectHermesMigrationSource returns a local Hermes source path when present.
func DetectHermesMigrationSource() string {
	if path := ExistingDir(strings.TrimSpace(os.Getenv("HERMES_HOME"))); path != "" {
		return path
	}
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home != "" {
		if path := ExistingDir(filepath.Join(home, ".hermes")); path != "" {
			return path
		}
	}
	return ""
}

// DetectOpenClawMigrationSource returns a local OpenClaw source path when present.
func DetectOpenClawMigrationSource() string {
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		return ""
	}
	for _, name := range []string{".openclaw", ".clawdbot", ".moltbot"} {
		if path := ExistingDir(filepath.Join(home, name)); path != "" {
			return path
		}
	}
	return ""
}

// ExistingDir returns path when it exists and is a directory.
func ExistingDir(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return ""
	}
	return path
}
