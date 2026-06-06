package setupfirst

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

type Action string

const (
	ActionQuick           Action = "quick"
	ActionFull            Action = "full"
	ActionMigrateHermes   Action = "migrate_hermes"
	ActionMigrateOpenClaw Action = "migrate_openclaw"
)

type Option struct {
	Action Action
	Label  string
}

type SourceSeams struct {
	DetectHermesMigrationSource   func() string
	DetectOpenClawMigrationSource func() string
}

type QuickRuntime struct {
	Out             io.Writer
	NonInteractive  bool
	RequestedTarget cli.SetupTargetID

	ChooseSetupTarget    func([]cli.SetupTargetOption, int) (cli.SetupTargetID, error)
	RunSetupProvider     func(bool) error
	RunProviderLiveTest  func() error
	LoadCurrentModel     func() (cli.ProviderModel, error)
	RunSetupModelSection func(bool) error
	RunWhatsAppSetup     func() error
	RunTelegramSetup     func() error
	RunGatewayPlatform   func(string) error
	LaunchChat           func() error

	BuildFirstRunPlan           func(config.Config, cli.SetupTargetID, bool) cli.FirstRunPlan
	SetupNavivoxProviderCommand func(config.Config, bool) string
	NewExitCodeError            func(int, error) error
}

func FirstRunSetupOptions(seams SourceSeams) []Option {
	options := []Option{
		{Action: ActionQuick, Label: "Quick setup — provider, model & messaging (recommended)"},
		{Action: ActionFull, Label: "Full setup — configure everything"},
	}
	if seams.DetectHermesMigrationSource != nil && strings.TrimSpace(seams.DetectHermesMigrationSource()) != "" {
		options = append(options, Option{Action: ActionMigrateHermes, Label: "Migrate Hermes"})
	}
	if seams.DetectOpenClawMigrationSource != nil && strings.TrimSpace(seams.DetectOpenClawMigrationSource()) != "" {
		options = append(options, Option{Action: ActionMigrateOpenClaw, Label: "Migrate OpenClaw"})
	}
	return options
}

func PrintQuickSetupTargets(out io.Writer, targets []cli.SetupTargetOption) {
	fmt.Fprintln(out, "Quick setup targets:")
	for _, target := range targets {
		command := strings.TrimSpace(target.SetupCommand)
		if command == "" {
			continue
		}
		fmt.Fprintf(out, "  - %s: %s\n", target.Label, command)
	}
}

func RunQuick(runtime QuickRuntime) error {
	out := runtime.Out
	if out == nil {
		out = io.Discard
	}
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("quick setup: load config: %w", err)
	}
	rawTarget := strings.TrimSpace(string(runtime.RequestedTarget))
	normalizedTarget, ok := ParseQuickTarget(rawTarget)
	if !ok {
		return exitCodeError(runtime, 2, fmt.Errorf("setup_target_invalid_selection: %s", rawTarget))
	}
	plan := buildPlan(runtime, cfg, normalizedTarget, !runtime.NonInteractive)
	target := normalizedTarget

	if rawTarget == "" {
		target = plan.DefaultTarget
		if runtime.NonInteractive {
			PrintQuickSetupTargets(out, plan.Targets)
			fmt.Fprintln(out)
			PrintFirstRunGuidance(out, plan)
			return nil
		}
		if runtime.ChooseSetupTarget == nil {
			return fmt.Errorf("setup target chooser unavailable")
		}
		selected, err := runtime.ChooseSetupTarget(plan.Targets, TargetDefaultIndex(plan.Targets, plan.DefaultTarget))
		if err != nil {
			return err
		}
		var ok bool
		target, ok = ParseQuickTarget(string(selected))
		if !ok {
			return exitCodeError(runtime, 2, fmt.Errorf("setup_target_invalid_selection: %s", selected))
		}
		plan = buildPlan(runtime, cfg, target, !runtime.NonInteractive)
	}

	if QuickNavivoxChannelOnly(target, runtime.NonInteractive) {
		if err := runQuickChannel(runtime, target); err != nil {
			return err
		}
		return runQuickHandoff(runtime, target)
	}
	if QuickChannelBeforeMissingCore(target, plan, runtime.NonInteractive) {
		if err := runQuickChannel(runtime, target); err != nil {
			return err
		}
		return runQuickHandoff(runtime, target)
	}

	channelRanBeforeCore := false
	if QuickChannelBeforeCore(target, plan, runtime.NonInteractive) {
		if err := runQuickChannel(runtime, target); err != nil {
			return err
		}
		channelRanBeforeCore = true
	}

	if err := runQuickCore(runtime); err != nil {
		return err
	}
	plan, err = BuildPlanForQuickTarget(runtime, target, !runtime.NonInteractive)
	if err != nil {
		return err
	}
	if _, missingChannel := plan.Step(cli.FirstRunStepChannel); missingChannel && IsQuickChannelTarget(target) && !channelRanBeforeCore {
		if err := runQuickChannel(runtime, target); err != nil {
			return err
		}
	}
	if runtime.RunProviderLiveTest == nil {
		return fmt.Errorf("provider live test seam unavailable")
	}
	if err := runtime.RunProviderLiveTest(); err != nil {
		fmt.Fprintln(out, "Provider live test failed. Chat was not opened.")
		fmt.Fprintf(out, "Repair: %s\n", QuickRepairCommand(runtime, target))
		return exitCodeError(runtime, 1, RedactedLiveTestError(err))
	}
	return runQuickHandoff(runtime, target)
}

func BuildPlanForQuickTarget(runtime QuickRuntime, target cli.SetupTargetID, interactive bool) (cli.FirstRunPlan, error) {
	cfg, err := config.Load(nil)
	if err != nil {
		return cli.FirstRunPlan{}, fmt.Errorf("quick setup: load config: %w", err)
	}
	return buildPlan(runtime, cfg, target, interactive), nil
}

func buildPlan(runtime QuickRuntime, cfg config.Config, target cli.SetupTargetID, interactive bool) cli.FirstRunPlan {
	if runtime.BuildFirstRunPlan != nil {
		return runtime.BuildFirstRunPlan(cfg, target, interactive)
	}
	return BuildFirstRunPlanFromConfig(cfg, target, interactive, SourceSeams{})
}

func RunQuickCore(runtime QuickRuntime) error {
	return runQuickCore(runtime)
}

func runQuickCore(runtime QuickRuntime) error {
	out := runtime.Out
	if out == nil {
		out = io.Discard
	}
	cli.ClearScreen(out)
	cli.PrintHeader(out, "Quick Setup - configure missing items only")
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("quick setup: load config: %w", err)
	}
	if strings.TrimSpace(cfg.Hermes.Endpoint) == "" || !config.ConfiguredProviderAuthPresent(cfg) {
		fmt.Fprintln(out, "Provider endpoint or auth is missing.")
		if runtime.RunSetupProvider == nil {
			return fmt.Errorf("setup provider seam unavailable")
		}
		if err := runtime.RunSetupProvider(runtime.NonInteractive); err != nil {
			return err
		}
	}
	if runtime.LoadCurrentModel == nil {
		return fmt.Errorf("load current model seam unavailable")
	}
	current, err := runtime.LoadCurrentModel()
	if err != nil {
		return fmt.Errorf("quick setup: load current model: %w", err)
	}
	if strings.TrimSpace(current.Provider) == "" || strings.TrimSpace(current.Model) == "" {
		fmt.Fprintln(out, "Model/provider defaults are missing.")
		if runtime.RunSetupModelSection == nil {
			return fmt.Errorf("setup model seam unavailable")
		}
		return runtime.RunSetupModelSection(runtime.NonInteractive)
	}
	fmt.Fprintf(out, "Current model/provider: %s via %s\n", current.Model, current.Provider)
	fmt.Fprintln(out, "No missing core setup items detected.")
	return nil
}

func QuickNavivoxChannelOnly(target cli.SetupTargetID, nonInteractive bool) bool {
	return !nonInteractive && NormalizeQuickTarget(target) == cli.SetupTargetNavivox
}

func QuickChannelBeforeMissingCore(target cli.SetupTargetID, plan cli.FirstRunPlan, nonInteractive bool) bool {
	if nonInteractive || NormalizeQuickTarget(target) == cli.SetupTargetNavivox || !IsQuickChannelTarget(target) {
		return false
	}
	return QuickMissingCore(plan)
}

func QuickChannelBeforeCore(target cli.SetupTargetID, plan cli.FirstRunPlan, nonInteractive bool) bool {
	if nonInteractive || !IsQuickChannelTarget(target) {
		return false
	}
	_, missingChannel := plan.Step(cli.FirstRunStepChannel)
	return missingChannel
}

func QuickMissingCore(plan cli.FirstRunPlan) bool {
	for _, id := range []cli.FirstRunStepID{cli.FirstRunStepProvider, cli.FirstRunStepAuth, cli.FirstRunStepModel} {
		if _, ok := plan.Step(id); ok {
			return true
		}
	}
	return false
}

func NextCoreSetupCommand(plan cli.FirstRunPlan) string {
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

func runQuickChannel(runtime QuickRuntime, target cli.SetupTargetID) error {
	out := runtime.Out
	if out == nil {
		out = io.Discard
	}
	if runtime.NonInteractive {
		if target == cli.SetupTargetWhatsApp {
			fmt.Fprintln(out, "WhatsApp setup command: gormes whatsapp --plan")
			return nil
		}
		fmt.Fprintln(out, "Channel setup command: gormes setup gateway")
		return nil
	}
	if target == cli.SetupTargetWhatsApp {
		if runtime.RunWhatsAppSetup == nil {
			return fmt.Errorf("whatsapp setup seam unavailable")
		}
		return runtime.RunWhatsAppSetup()
	}
	if target == cli.SetupTargetTelegram {
		if runtime.RunTelegramSetup == nil {
			return fmt.Errorf("telegram setup seam unavailable")
		}
		return runtime.RunTelegramSetup()
	}
	if runtime.RunGatewayPlatform == nil {
		return fmt.Errorf("gateway platform setup seam unavailable")
	}
	return runtime.RunGatewayPlatform(string(target))
}

func runQuickHandoff(runtime QuickRuntime, target cli.SetupTargetID) error {
	out := runtime.Out
	if out == nil {
		out = io.Discard
	}
	if NormalizeQuickTarget(target) == cli.SetupTargetNavivox {
		if cfg, err := config.Load(nil); err == nil && runtime.SetupNavivoxProviderCommand != nil {
			if command := runtime.SetupNavivoxProviderCommand(cfg, config.ConfiguredProviderAuthPresent(cfg)); command != "" {
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
	if IsQuickChannelTarget(target) {
		if cfg, err := config.Load(nil); err == nil {
			plan := buildPlan(runtime, cfg, NormalizeQuickTarget(target), !runtime.NonInteractive)
			if !runtime.NonInteractive && QuickMissingCore(plan) {
				label := strings.TrimSpace(plan.TargetLabel)
				if label == "" {
					label = string(NormalizeQuickTarget(target))
				}
				fmt.Fprintf(out, "%s channel setup checked.\n", label)
				fmt.Fprintf(out, "Provider/model setup is still required before `gormes gateway` can answer %s.\n", label)
				fmt.Fprintf(out, "Next setup command: %s\n", NextCoreSetupCommand(plan))
				fmt.Fprintln(out, "After that, start gateway: gormes gateway")
				return nil
			}
		}
		fmt.Fprintln(out, "Channel setup checked. Start messaging with: gormes gateway")
		return nil
	}
	if runtime.NonInteractive {
		fmt.Fprintln(out, "Terminal chat ready. Start chatting with: gormes")
		return nil
	}
	if runtime.LaunchChat == nil {
		return fmt.Errorf("launch chat seam unavailable")
	}
	return runtime.LaunchChat()
}

func NormalizeQuickTarget(target cli.SetupTargetID) cli.SetupTargetID {
	normalized, ok := ParseQuickTarget(string(target))
	if !ok {
		return cli.SetupTargetTerminal
	}
	return normalized
}

func ParseQuickTarget(target string) (cli.SetupTargetID, bool) {
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

func IsQuickChannelTarget(target cli.SetupTargetID) bool {
	switch NormalizeQuickTarget(target) {
	case cli.SetupTargetTelegram, cli.SetupTargetWhatsApp, cli.SetupTargetDiscord, cli.SetupTargetSlack, cli.SetupTargetNavivox:
		return true
	default:
		return false
	}
}

func TargetDefaultIndex(targets []cli.SetupTargetOption, target cli.SetupTargetID) int {
	target = NormalizeQuickTarget(target)
	for i, option := range targets {
		if option.ID == target {
			return i
		}
	}
	return 0
}

func QuickRepairCommand(runtime QuickRuntime, target cli.SetupTargetID) string {
	cfg, err := config.Load(nil)
	if err != nil {
		return "gormes doctor"
	}
	plan := buildPlan(runtime, cfg, NormalizeQuickTarget(target), false)
	if len(plan.MissingSteps) > 0 && strings.TrimSpace(plan.MissingSteps[0].Command) != "" {
		return plan.MissingSteps[0].Command
	}
	if strings.TrimSpace(plan.NextCommand) != "" && plan.NextCommand != "gormes" && plan.NextCommand != "gormes gateway" {
		return plan.NextCommand
	}
	return "gormes doctor"
}

func RedactedLiveTestError(err error) error {
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
	if dotenv := cli.ReadDotenvValues(config.EnvPath()); dotenv != nil {
		secrets = append(secrets, dotenv["GORMES_API_KEY"])
	}
	return fmt.Errorf("%s", RedactSecretText(message, secrets...))
}

func RedactSecretText(text string, secrets ...string) string {
	redacted := text
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, secret, "[REDACTED]")
	}
	return redacted
}

func BuildFirstRunPlanFromConfig(cfg config.Config, target cli.SetupTargetID, interactive bool, seams SourceSeams) cli.FirstRunPlan {
	hermesSource := ""
	if seams.DetectHermesMigrationSource != nil {
		hermesSource = seams.DetectHermesMigrationSource()
	}
	openClawSource := ""
	if seams.DetectOpenClawMigrationSource != nil {
		openClawSource = seams.DetectOpenClawMigrationSource()
	}
	return cli.BuildFirstRunPlan(cli.FirstRunPlanInput{
		Interactive:        interactive,
		Provider:           cfg.Hermes.Provider,
		Endpoint:           cfg.Hermes.Endpoint,
		Model:              cfg.Hermes.Model,
		APIKeyPresent:      config.ConfiguredProviderAuthPresent(cfg),
		Target:             target,
		Channels:           ChannelStates(cfg),
		HermesSourcePath:   hermesSource,
		OpenClawSourcePath: openClawSource,
	})
}

func ChannelStates(cfg config.Config) []cli.ChannelState {
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
			Target:         cli.SetupTargetNavivox,
			Label:          "Navivox",
			Configured:     cfg.Navivox.Enabled,
			Detail:         configuredNavivoxGatewayStatusDetail(cfg.Navivox),
			SetupCommand:   "gormes setup navivox",
			HandoffCommand: "gormes gateway",
		},
	}
}

func PrintFirstRunGuidance(out io.Writer, plan cli.FirstRunPlan) {
	fmt.Fprintln(out, "Gormes setup needed")
	if strings.TrimSpace(plan.NextCommand) != "" {
		fmt.Fprintf(out, "Next: %s\n", plan.NextCommand)
	}
	fmt.Fprintln(out, "Non-interactive mode will not prompt.")
	if len(plan.MissingSteps) == 0 {
		return
	}
	fmt.Fprintln(out, "Missing setup:")
	for _, step := range plan.MissingSteps {
		fmt.Fprintf(out, "  - %s: %s\n", step.Label, step.Command)
	}
}

func configuredTelegramGatewayStatusDetail(cfg config.TelegramCfg) string {
	detail := "first_run_discovery=" + strconv.FormatBool(cfg.FirstRunDiscovery)
	if cfg.AllowedChatID != 0 {
		detail = "allowed_chat_id=" + strconv.FormatInt(cfg.AllowedChatID, 10)
	}
	if len(cfg.AllowedUserIDs) > 0 {
		userDetail := "allowed_users=" + strconv.Itoa(len(cfg.AllowedUserIDs))
		if detail == "" {
			return userDetail
		}
		return detail + " " + userDetail
	}
	return detail
}

func configuredSlackGatewayStatusDetail(cfg config.SlackCfg) string {
	if missing := missingSlackCredentials(cfg); len(missing) > 0 {
		return "missing_tokens=" + strings.Join(missing, ",")
	}
	detail := "first_run_discovery=" + strconv.FormatBool(cfg.FirstRunDiscovery)
	if cfg.AllowedChannelID != "" {
		detail = "allowed_channel_id=" + cfg.AllowedChannelID
	}
	return detail
}

func configuredNavivoxGatewayStatusDetail(cfg config.NavivoxCfg) string {
	return fmt.Sprintf("bind=%s:%d exposure=%s auth=%s", cfg.BindHost, cfg.Port, cfg.ExposureMode, cfg.AuthMode)
}

func missingSlackCredentials(cfg config.SlackCfg) []string {
	missing := []string{}
	if strings.TrimSpace(cfg.BotToken) == "" {
		missing = append(missing, "bot_token")
	}
	if strings.TrimSpace(cfg.AppToken) == "" {
		missing = append(missing, "app_token")
	}
	return missing
}

func exitCodeError(runtime QuickRuntime, code int, err error) error {
	if runtime.NewExitCodeError != nil {
		return runtime.NewExitCodeError(code, err)
	}
	return err
}
