package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

func firstRunSetupOptions(seams setupCommandSeams) []setupMenuOption {
	options := []setupMenuOption{
		{Action: setupActionQuick, Label: "Quick setup — provider, model & messaging (recommended)"},
		{Action: setupActionFull, Label: "Full setup — configure everything"},
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
	if stdin, ok := cmd.InOrStdin().(*os.File); ok {
		selected, err := runBubbleTeaPick(cmd.Context(), stdin, cmd.OutOrStdout(), "Where should quick setup take you first?", setupTargetPickerChoices(targets), string(targets[defaultOption].ID))
		if err == nil && selected != "" {
			return cli.SetupTargetID(selected), nil
		}
		if err != nil && !bubbleTeaPickShouldFallback(err) {
			return "", err
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

func setupTargetPickerChoices(targets []cli.SetupTargetOption) []tuiPickChoice {
	choices := make([]tuiPickChoice, len(targets))
	for i, target := range targets {
		choices[i] = tuiPickChoice{ID: string(target.ID), Label: target.Label}
	}
	return choices
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
		plan = buildFirstRunPlanFromConfig(cfg, target, !nonInteractive)
	}

	if setupQuickNavivoxChannelOnly(target, nonInteractive) {
		if err := runSetupQuickChannel(cmd, seams, target, nonInteractive); err != nil {
			return err
		}
		return runSetupQuickHandoff(cmd, seams, target, nonInteractive)
	}
	if setupQuickChannelBeforeMissingCore(target, plan, nonInteractive) {
		if err := runSetupQuickChannel(cmd, seams, target, nonInteractive); err != nil {
			return err
		}
		return runSetupQuickHandoff(cmd, seams, target, nonInteractive)
	}

	channelRanBeforeCore := false
	if setupQuickChannelBeforeCore(target, plan, nonInteractive) {
		if err := runSetupQuickChannel(cmd, seams, target, nonInteractive); err != nil {
			return err
		}
		channelRanBeforeCore = true
	}

	if err := runSetupQuickCore(cmd, seams, nonInteractive); err != nil {
		return err
	}
	plan, err = buildFirstRunPlanForSetupQuickTarget(target, !nonInteractive)
	if err != nil {
		return err
	}
	if _, missingChannel := plan.Step(cli.FirstRunStepChannel); missingChannel && isSetupQuickChannelTarget(target) && !channelRanBeforeCore {
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

func buildFirstRunPlanForSetupQuickTarget(target cli.SetupTargetID, interactive bool) (cli.FirstRunPlan, error) {
	cfg, err := config.Load(nil)
	if err != nil {
		return cli.FirstRunPlan{}, fmt.Errorf("quick setup: load config: %w", err)
	}
	return buildFirstRunPlanFromConfig(cfg, target, interactive), nil
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

// Navivox pairing is the selected destination, so always open the Navivox
// channel setup from this quick-target path instead of falling through to the
// provider/model picker or a generic handoff. Provider/model setup can be
// completed later from Navivox or the normal setup provider/model sections.
func setupQuickNavivoxChannelOnly(target cli.SetupTargetID, nonInteractive bool) bool {
	return !nonInteractive && normalizeSetupQuickTarget(target) == cli.SetupTargetNavivox
}

// Navivox pairing is channel-only above. Other interactive channel targets
// should still honor the first-run target picker: if core provider/model setup
// is missing, open the selected channel setup first and hand off to the next
// core setup step instead of surprising the operator with provider setup.
func setupQuickChannelBeforeMissingCore(target cli.SetupTargetID, plan cli.FirstRunPlan, nonInteractive bool) bool {
	if nonInteractive || normalizeSetupQuickTarget(target) == cli.SetupTargetNavivox || !isSetupQuickChannelTarget(target) {
		return false
	}
	return setupQuickMissingCore(plan)
}

func setupQuickChannelBeforeCore(target cli.SetupTargetID, plan cli.FirstRunPlan, nonInteractive bool) bool {
	if nonInteractive || !isSetupQuickChannelTarget(target) {
		return false
	}
	_, missingChannel := plan.Step(cli.FirstRunStepChannel)
	return missingChannel
}

func setupQuickMissingCore(plan cli.FirstRunPlan) bool {
	for _, id := range []cli.FirstRunStepID{cli.FirstRunStepProvider, cli.FirstRunStepAuth, cli.FirstRunStepModel} {
		if _, ok := plan.Step(id); ok {
			return true
		}
	}
	return false
}

func setupQuickNextCoreSetupCommand(plan cli.FirstRunPlan) string {
	if _, ok := plan.Step(cli.FirstRunStepProvider); ok {
		return "gormes setup provider"
	}
	if step, ok := plan.Step(cli.FirstRunStepAuth); ok && strings.TrimSpace(step.Command) != "" {
		return step.Command
	}
	if step, ok := plan.Step(cli.FirstRunStepModel); ok && strings.TrimSpace(step.Command) != "" {
		return step.Command
	}
	return "gormes setup provider"
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
	if target == cli.SetupTargetTelegram {
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
		return applySetupTelegramGatewayAnswers(cmd, cfg.Telegram, answers)
	}
	return seams.RunGatewayPlatform(cmd, string(target))
}

func runSetupQuickHandoff(cmd *cobra.Command, seams setupCommandSeams, target cli.SetupTargetID, nonInteractive bool) error {
	if normalizeSetupQuickTarget(target) == cli.SetupTargetNavivox {
		out := cmd.OutOrStdout()
		if cfg, err := config.Load(nil); err == nil {
			if command := navivoxProviderSetupCommand(cfg); command != "" {
				fmt.Fprintln(out, "Navivox channel setup checked.")
				fmt.Fprintln(out, "Provider/model setup is still required before `gormes gateway` can answer Navivox.")
				fmt.Fprintf(out, "Next setup command: %s\n", command)
				fmt.Fprintln(out, "After that, start gateway: gormes gateway")
				return nil
			}
		}
		fmt.Fprintln(out, "Navivox channel setup checked. Start gateway after scanning the QR: gormes gateway")
		return nil
	}
	if isSetupQuickChannelTarget(target) {
		out := cmd.OutOrStdout()
		if cfg, err := config.Load(nil); err == nil {
			plan := buildFirstRunPlanFromConfig(cfg, normalizeSetupQuickTarget(target), !nonInteractive)
			if !nonInteractive && setupQuickMissingCore(plan) {
				label := strings.TrimSpace(plan.TargetLabel)
				if label == "" {
					label = string(normalizeSetupQuickTarget(target))
				}
				fmt.Fprintf(out, "%s channel setup checked.\n", label)
				fmt.Fprintf(out, "Provider/model setup is still required before `gormes gateway` can answer %s.\n", label)
				fmt.Fprintf(out, "Next setup command: %s\n", setupQuickNextCoreSetupCommand(plan))
				fmt.Fprintln(out, "After that, start gateway: gormes gateway")
				return nil
			}
		}
		fmt.Fprintln(out, "Channel setup checked. Start messaging with: gormes gateway")
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
	case cli.SetupTargetNavivox:
		return cli.SetupTargetNavivox, true
	default:
		return "", false
	}
}

func isSetupQuickChannelTarget(target cli.SetupTargetID) bool {
	switch normalizeSetupQuickTarget(target) {
	case cli.SetupTargetTelegram, cli.SetupTargetWhatsApp, cli.SetupTargetDiscord, cli.SetupTargetSlack, cli.SetupTargetNavivox:
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
