package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func maybeHandleRootFirstRun(cmd *cobra.Command, invocation tuiInvocation, runtime rootRuntime) (bool, error) {
	if rootFirstRunBypass(cmd, invocation) {
		return false, nil
	}
	interactive := runtime.isTTY != nil && runtime.isTTY()
	plan := buildFirstRunPlanFromConfig(invocation.Config, cli.SetupTargetTerminal, interactive)
	if plan.Ready {
		return false, nil
	}
	if interactive {
		return true, runtime.runFirstRunSetup(cmd)
	}
	printFirstRunGuidance(cmd, plan)
	return true, nil
}

func rootFirstRunBypass(cmd *cobra.Command, invocation tuiInvocation) bool {
	if offline, _ := cmd.Flags().GetBool("offline"); offline {
		return true
	}
	return strings.TrimSpace(invocation.RemoteURL) != ""
}

func runFirstRunSetupCommand(cmd *cobra.Command) error {
	setup := newSetupCommand()
	setup.SetOut(cmd.OutOrStdout())
	setup.SetErr(cmd.ErrOrStderr())
	setup.SetIn(cmd.InOrStdin())
	setup.SetArgs([]string{})
	return setup.ExecuteContext(cmd.Context())
}

func buildFirstRunPlanFromConfig(cfg config.Config, target cli.SetupTargetID, interactive bool) cli.FirstRunPlan {
	return cli.BuildFirstRunPlan(cli.FirstRunPlanInput{
		Interactive:        interactive,
		Provider:           cfg.Hermes.Provider,
		Endpoint:           cfg.Hermes.Endpoint,
		Model:              cfg.Hermes.Model,
		APIKeyPresent:      configuredProviderAuthPresent(cfg),
		Target:             target,
		Channels:           firstRunChannelStates(cfg),
		HermesSourcePath:   detectHermesMigrationSource(),
		OpenClawSourcePath: detectOpenClawMigrationSource(),
	})
}

func firstRunChannelStates(cfg config.Config) []cli.ChannelState {
	return []cli.ChannelState{
		{
			Target:         cli.SetupTargetTelegram,
			Label:          "Telegram",
			Configured:     strings.TrimSpace(cfg.Telegram.BotToken) != "",
			Detail:         configuredTelegramGatewayStatusDetail(cfg.Telegram),
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
			Detail:         configuredSlackGatewayStatusDetail(cfg.Slack),
			SetupCommand:   "gormes setup --quick --target slack",
			HandoffCommand: "gormes gateway",
		},
		{
			Target:         cli.SetupTargetNavibox,
			Label:          "Navibox",
			Configured:     cfg.Navibox.Enabled,
			Detail:         configuredNaviboxGatewayStatusDetail(cfg.Navibox),
			SetupCommand:   "gormes setup --quick --target navibox",
			HandoffCommand: "gormes gateway",
		},
	}
}

func printFirstRunGuidance(cmd *cobra.Command, plan cli.FirstRunPlan) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Gormes setup needed")
	if plan.Summary != "" {
		fmt.Fprintf(out, "%s\n", plan.Summary)
	}
	for _, step := range plan.MissingSteps {
		if step.Detail == "" {
			continue
		}
		if command := firstRunGuidanceCommand(step.Command); command != "" {
			fmt.Fprintf(out, "- %s: %s (run: %s)\n", step.Label, step.Detail, command)
		} else {
			fmt.Fprintf(out, "- %s: %s\n", step.Label, step.Detail)
		}
	}
	if command := firstRunGuidanceCommand(plan.NextCommand); command != "" {
		fmt.Fprintf(out, "Next: %s\n", command)
	}
	fmt.Fprintln(out, "Non-interactive mode will not prompt.")
}

func firstRunGuidanceCommand(command string) string {
	return strings.TrimSpace(command)
}

func detectHermesMigrationSource() string {
	if path := existingDir(strings.TrimSpace(os.Getenv("HERMES_HOME"))); path != "" {
		return path
	}
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home != "" {
		if path := existingDir(filepath.Join(home, ".hermes")); path != "" {
			return path
		}
	}
	return ""
}

func detectOpenClawMigrationSource() string {
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		return ""
	}
	for _, name := range []string{".openclaw", ".clawdbot", ".moltbot"} {
		if path := existingDir(filepath.Join(home, name)); path != "" {
			return path
		}
	}
	return ""
}

func existingDir(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return ""
	}
	return path
}
