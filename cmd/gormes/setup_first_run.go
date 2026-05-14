package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func firstRunSetupOptions(seams setupCommandSeams) []setupMenuOption {
	options := []setupMenuOption{
		{Action: setupActionQuick, Label: "Quick setup - provider, model, and messaging"},
		{Action: setupActionFull, Label: "Full setup - configure everything"},
	}
	if strings.TrimSpace(seams.DetectHermesMigrationSource()) != "" {
		options = append(options, setupMenuOption{Action: setupActionMigrateHermes, Label: "Migrate Hermes"})
	}
	if strings.TrimSpace(seams.DetectOpenClawMigrationSource()) != "" {
		options = append(options, setupMenuOption{Action: setupActionMigrateOpenClaw, Label: "Migrate OpenClaw"})
	}
	return options
}

func printQuickSetupTargets(cmd *cobra.Command, targets []cli.SetupTargetOption) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Quick setup targets:")
	for _, target := range targets {
		command := strings.TrimSpace(target.SetupCommand)
		if command == "" {
			continue
		}
		fmt.Fprintf(out, "  - %s: %s\n", target.Label, command)
	}
}

func promptSetupTarget(cmd *cobra.Command, targets []cli.SetupTargetOption, defaultOption int) (cli.SetupTargetID, error) {
	if len(targets) == 0 {
		return cli.SetupTargetTerminal, nil
	}
	if defaultOption < 0 || defaultOption >= len(targets) {
		defaultOption = 0
	}
	if stdin, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(stdin.Fd())) {
		menu := cli.NewInteractiveMenu(cmd.OutOrStdout(), stdin, "Select target:")
		menu.WithHeader("Where should quick setup take you first?")
		cliOpts := make([]cli.MenuOption, len(targets))
		for i, target := range targets {
			cliOpts[i] = cli.MenuOption{ID: string(target.ID), Label: target.Label, Enabled: true}
		}
		menu.WithOptions(cliOpts).WithDefaultIndex(defaultOption)
		selected, err := menu.Run()
		if err == nil && selected != "" {
			return cli.SetupTargetID(selected), nil
		}
	}

	out := cmd.OutOrStdout()
	cli.ClearScreen(out)
	cli.PrintHeader(out, "Where should quick setup take you first?")
	fmt.Fprintln(out)
	for i, target := range targets {
		marker := "( )"
		label := target.Label
		if i == defaultOption {
			marker = "(*)"
			label = cli.Bold(out, label)
		}
		fmt.Fprintf(out, "  %d. %s %s\n", i+1, marker, label)
	}
	fmt.Fprintln(out)

	defaultText := strconv.Itoa(defaultOption + 1)
	answer, err := promptString(cmd, fmt.Sprintf("Select target [%s]: ", defaultText), defaultText)
	if err != nil {
		return "", err
	}
	answer = strings.ToLower(strings.TrimSpace(cli.StripANSI(answer)))
	if answer == "" {
		return targets[defaultOption].ID, nil
	}
	if n, err := strconv.Atoi(answer); err == nil && n >= 1 && n <= len(targets) {
		return targets[n-1].ID, nil
	}
	for _, target := range targets {
		if answer == string(target.ID) || strings.Contains(strings.ToLower(target.Label), answer) {
			return target.ID, nil
		}
	}
	return "", newExitCodeError(2, fmt.Errorf("setup_target_invalid_selection: %s", answer))
}

func runSetupQuick(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool, requestedTarget cli.SetupTargetID) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("quick setup: load config: %w", err)
	}
	rawTarget := strings.TrimSpace(string(requestedTarget))
	normalizedTarget, ok := parseSetupQuickTarget(rawTarget)
	if !ok {
		return newExitCodeError(2, fmt.Errorf("setup_target_invalid_selection: %s", rawTarget))
	}
	plan := buildFirstRunPlanFromConfig(cfg, normalizedTarget, !nonInteractive)
	target := normalizedTarget

	if rawTarget == "" {
		target = plan.DefaultTarget
		if nonInteractive {
			printQuickSetupTargets(cmd, plan.Targets)
			fmt.Fprintln(cmd.OutOrStdout())
			printFirstRunGuidance(cmd, plan)
			return nil
		}
		selected, err := seams.ChooseSetupTarget(cmd, plan.Targets, setupTargetDefaultIndex(plan.Targets, plan.DefaultTarget))
		if err != nil {
			return err
		}
		var ok bool
		target, ok = parseSetupQuickTarget(string(selected))
		if !ok {
			return newExitCodeError(2, fmt.Errorf("setup_target_invalid_selection: %s", selected))
		}
	}

	if err := runSetupQuickCore(cmd, seams, nonInteractive); err != nil {
		return err
	}
	if isSetupQuickChannelTarget(target) {
		if err := runSetupQuickChannel(cmd, seams, target, nonInteractive); err != nil {
			return err
		}
	}
	if err := seams.RunProviderLiveTest(cmd); err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "Provider live test failed. Chat was not opened.")
		fmt.Fprintf(cmd.OutOrStdout(), "Repair: %s\n", setupQuickRepairCommand(cmd, target))
		return newExitCodeError(1, redactedSetupQuickLiveTestError(err))
	}
	return runSetupQuickHandoff(cmd, seams, target, nonInteractive)
}

func runSetupQuickCore(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	out := cmd.OutOrStdout()
	cli.ClearScreen(out)
	cli.PrintHeader(out, "Quick Setup - configure missing items only")
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("quick setup: load config: %w", err)
	}
	if strings.TrimSpace(cfg.Hermes.Endpoint) == "" || !configuredProviderAuthPresent(cfg) {
		fmt.Fprintln(out, "Provider endpoint or auth is missing.")
		if err := seams.RunSetupProvider(cmd, nonInteractive); err != nil {
			return err
		}
	}
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
	return nil
}

func runSetupQuickChannel(cmd *cobra.Command, seams setupCommandSeams, target cli.SetupTargetID, nonInteractive bool) error {
	if nonInteractive {
		if target == cli.SetupTargetWhatsApp {
			fmt.Fprintln(cmd.OutOrStdout(), "WhatsApp setup command: gormes whatsapp --plan")
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Channel setup command: gormes setup gateway")
		return nil
	}
	if target == cli.SetupTargetWhatsApp {
		return seams.RunWhatsAppSetup(cmd)
	}
	return seams.RunGatewayPlatform(cmd, string(target))
}

func runSetupQuickHandoff(cmd *cobra.Command, seams setupCommandSeams, target cli.SetupTargetID, nonInteractive bool) error {
	if isSetupQuickChannelTarget(target) {
		fmt.Fprintln(cmd.OutOrStdout(), "Channel setup checked. Start messaging with: gormes gateway")
		return nil
	}
	if nonInteractive {
		fmt.Fprintln(cmd.OutOrStdout(), "Terminal chat ready. Start chatting with: gormes")
		return nil
	}
	return seams.LaunchChat(cmd)
}

func runSetupProviderLiveTest(cmd *cobra.Command) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return err
	}
	client, err := newProviderHTTPClient(cfg, cfg.Hermes.Provider)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()
	return client.Health(ctx)
}

func normalizeSetupQuickTarget(target cli.SetupTargetID) cli.SetupTargetID {
	normalized, ok := parseSetupQuickTarget(string(target))
	if !ok {
		return cli.SetupTargetTerminal
	}
	return normalized
}

func parseSetupQuickTarget(target string) (cli.SetupTargetID, bool) {
	switch cli.SetupTargetID(strings.ToLower(strings.TrimSpace(target))) {
	case "", cli.SetupTargetTerminal, "chat", "tui":
		return cli.SetupTargetTerminal, true
	case cli.SetupTargetTelegram:
		return cli.SetupTargetTelegram, true
	case cli.SetupTargetWhatsApp, "wa":
		return cli.SetupTargetWhatsApp, true
	case cli.SetupTargetDiscord:
		return cli.SetupTargetDiscord, true
	case cli.SetupTargetSlack:
		return cli.SetupTargetSlack, true
	case cli.SetupTargetNavibox:
		return cli.SetupTargetNavibox, true
	default:
		return "", false
	}
}

func isSetupQuickChannelTarget(target cli.SetupTargetID) bool {
	switch normalizeSetupQuickTarget(target) {
	case cli.SetupTargetTelegram, cli.SetupTargetWhatsApp, cli.SetupTargetDiscord, cli.SetupTargetSlack, cli.SetupTargetNavibox:
		return true
	default:
		return false
	}
}

func setupTargetDefaultIndex(targets []cli.SetupTargetOption, target cli.SetupTargetID) int {
	target = normalizeSetupQuickTarget(target)
	for i, option := range targets {
		if option.ID == target {
			return i
		}
	}
	return 0
}

func setupQuickRepairCommand(cmd *cobra.Command, target cli.SetupTargetID) string {
	cfg, err := config.Load(nil)
	if err != nil {
		return "gormes doctor"
	}
	plan := buildFirstRunPlanFromConfig(cfg, normalizeSetupQuickTarget(target), false)
	if len(plan.MissingSteps) > 0 && strings.TrimSpace(plan.MissingSteps[0].Command) != "" {
		return plan.MissingSteps[0].Command
	}
	if strings.TrimSpace(plan.NextCommand) != "" && plan.NextCommand != "gormes" && plan.NextCommand != "gormes gateway" {
		return plan.NextCommand
	}
	return "gormes doctor"
}

func redactedSetupQuickLiveTestError(err error) error {
	message := "quick setup: provider live test failed"
	if err != nil {
		message += ": " + err.Error()
	}
	cfg, loadErr := config.Load(nil)
	var secrets []string
	if loadErr == nil {
		secrets = append(secrets, cfg.Hermes.APIKey)
	}
	secrets = append(secrets, os.Getenv("GORMES_API_KEY"))
	if dotenv := readDotenvValues(config.EnvPath()); dotenv != nil {
		secrets = append(secrets, dotenv["GORMES_API_KEY"])
	}
	return fmt.Errorf("%s", redactRuntimeSecretText(message, secrets...))
}
