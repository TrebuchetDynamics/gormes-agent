package gormescli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

type SetupGatewaySeams struct {
	RunGatewaySetupWizard    func(*cobra.Command, config.Config) (SetupGatewayWizardResult, error)
	RunTelegramGatewayWizard func(*cobra.Command, config.TelegramCfg) (SetupTelegramGatewayAnswers, error)
	RunGatewayPlatform       func(*cobra.Command, string) error
}

type SetupGatewayRuntime struct {
	NewExitCodeError  func(int, error) error
	PromptString      func(*cobra.Command, string, string) (string, error)
	PromptSecret      func(*cobra.Command, string) (string, error)
	RunWhatsAppSetup  func(*cobra.Command) error
	RunNavivoxGateway func(*cobra.Command, config.Config) error
}

type SetupGatewayWizardResult struct {
	SelectedPlatforms []string
	Telegram          *SetupTelegramGatewayAnswers
	BubbleTea         bool
}

type SetupTelegramGatewayAnswers struct {
	Token        string
	AccessPolicy string
	AllowedUsers string
	HomeChatID   string
	HomeThreadID string
	Apply        bool
}

type SetupGatewayPlatformOption struct {
	key        string
	label      string
	configured bool
	detail     string
}

// Backwards-compatible package-local names keep the moved setup gateway tests
// close to their original cmd/gormes spelling while the exported API remains
// available to the root CLI shim.
type setupGatewayWizardResult = SetupGatewayWizardResult
type setupTelegramGatewayAnswers = SetupTelegramGatewayAnswers
type setupGatewayPlatformOption = SetupGatewayPlatformOption

func RunSetupTelegramSection(cmd *cobra.Command, nonInteractive bool, seams SetupGatewaySeams, runtime SetupGatewayRuntime) error {
	runtime = setupGatewayRuntimeDefaults(runtime)
	if nonInteractive {
		return runtime.exitCodeError(2, fmt.Errorf("setup_telegram_requires_tty: run `gormes setup gateway --plan` for offline guidance, or run `gormes setup telegram` in a terminal"))
	}
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("setup telegram: load config: %w", err)
	}
	wizard := seams.RunTelegramGatewayWizard
	if wizard == nil {
		wizard = RunSetupTelegramBubbleTeaWizard
	}
	answers, err := wizard(cmd, cfg.Telegram)
	if err != nil {
		if errors.Is(err, setupwizard.ErrRequiresTTY) {
			return runtime.exitCodeError(2, fmt.Errorf("setup_telegram_requires_tty: run `gormes setup gateway --plan` for offline guidance, or run `gormes setup telegram` in a terminal"))
		}
		return err
	}
	return ApplySetupTelegramGatewayAnswers(cmd, cfg.Telegram, answers, runtime)
}

func RunSetupGatewaySection(cmd *cobra.Command, nonInteractive bool, seams SetupGatewaySeams, runtime SetupGatewayRuntime) error {
	runtime = setupGatewayRuntimeDefaults(runtime)
	out := cmd.OutOrStdout()
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("setup gateway: load config: %w", err)
	}

	if SetupGatewayPlanFlag(cmd) {
		RenderSetupGatewayPlan(out, cfg, true)
		return nil
	}
	if nonInteractive {
		RenderSetupGatewayPlan(out, cfg, true)
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Skipped (keeping current gateway platform configuration).")
		fmt.Fprintln(out, "Run `gormes setup gateway` from a TTY or configure credentials with `gormes config edit`.")
		fmt.Fprintln(out, "Start messaging with: gormes gateway")
		return nil
	}

	wizard := seams.RunGatewaySetupWizard
	if wizard == nil {
		wizard = RunSetupGatewayBubbleTeaWizard
	}
	result, err := wizard(cmd, cfg)
	if err != nil {
		if errors.Is(err, setupwizard.ErrRequiresTTY) {
			return runtime.exitCodeError(2, fmt.Errorf("setup_gateway_requires_tty: run `gormes setup gateway --plan` for offline guidance, or run `gormes setup gateway` in a terminal"))
		}
		return err
	}
	selected := compactSetupGatewayStrings(result.SelectedPlatforms)
	if len(selected) == 0 {
		fmt.Fprintln(out, "No platform setup changes selected.")
		fmt.Fprintln(out, "Keeping current gateway platform configuration.")
		return nil
	}
	for _, platform := range selected {
		if platform == "telegram" && result.Telegram != nil {
			if err := ApplySetupTelegramGatewayAnswers(cmd, cfg.Telegram, *result.Telegram, runtime); err != nil {
				return err
			}
			continue
		}
		if platform == "navivox" {
			if err := runtime.RunNavivoxGateway(cmd, cfg); err != nil {
				return err
			}
			continue
		}
		if result.BubbleTea && platform != "whatsapp" {
			fmt.Fprintf(out, "%s Bubble Tea setup is not shipped in this slice; use `gormes setup gateway --plan` and `gormes config edit` for now.\n", SetupGatewayPlatformFallbackLabel(platform))
			continue
		}
		runPlatform := seams.RunGatewayPlatform
		if runPlatform == nil {
			runPlatform = func(cmd *cobra.Command, platform string) error {
				return RunSetupGatewayPlatform(cmd, platform, runtime)
			}
		}
		if err := runPlatform(cmd, platform); err != nil {
			return err
		}
	}
	fmt.Fprintln(out, "Start messaging with: gormes gateway")
	return nil
}

func SetupGatewayPlanFlag(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	value, err := cmd.Flags().GetBool("plan")
	return err == nil && value
}

func RenderSetupGatewayPlan(out io.Writer, cfg config.Config, planOnly bool) {
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

func SetupGatewayPlatformOptions(cfg config.Config) []SetupGatewayPlatformOption {
	configured := setupGatewayConfiguredDetails(cfg)
	manifestByID := map[string]gateway.PlatformManifestEntry{}
	for _, entry := range gateway.HermesGatewayPlatformManifest() {
		manifestByID[entry.ID] = entry
	}

	out := make([]SetupGatewayPlatformOption, 0, 5)
	for _, key := range []string{"telegram", "discord", "slack", "whatsapp", "navivox"} {
		label := SetupGatewayPlatformFallbackLabel(key)
		if entry, ok := manifestByID[key]; ok && strings.TrimSpace(entry.DisplayName) != "" {
			label = entry.DisplayName
		}
		detail, ok := configured[key]
		out = append(out, SetupGatewayPlatformOption{
			key:        key,
			label:      label,
			configured: ok,
			detail:     detail,
		})
	}
	return out
}

func SetupGatewayPlatformFallbackLabel(key string) string {
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

func ParseSetupGatewaySelection(input string, options []SetupGatewayPlatformOption) ([]string, bool, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, true, nil
	}
	byKey := make(map[string]SetupGatewayPlatformOption, len(options))
	byLabel := make(map[string]SetupGatewayPlatformOption, len(options))
	for _, option := range options {
		byKey[option.key] = option
		byLabel[NormalizeSetupValue(option.label)] = option
	}

	var selected []string
	seen := map[string]bool{}
	for _, token := range strings.FieldsFunc(input, setupGatewaySelectionSeparator) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		var key string
		if index, err := strconv.Atoi(token); err == nil {
			if index < 1 || index > len(options) {
				return nil, false, NewExitCodeError(2, fmt.Errorf("setup_gateway_invalid_selection: %s", token))
			}
			key = options[index-1].key
		} else {
			normalized := NormalizeSetupValue(token)
			if option, ok := byKey[normalized]; ok {
				key = option.key
			} else if option, ok := byLabel[normalized]; ok {
				key = option.key
			} else {
				return nil, false, NewExitCodeError(2, fmt.Errorf("setup_gateway_invalid_selection: %s", token))
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

func RunSetupGatewayBubbleTeaWizard(cmd *cobra.Command, cfg config.Config) (SetupGatewayWizardResult, error) {
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
		return SetupGatewayWizardResult{}, nil
	}
	if err != nil {
		return SetupGatewayWizardResult{}, err
	}
	platform := NormalizeSetupValue(result.Choice("platform"))
	if platform == "" || platform == "keep" {
		return SetupGatewayWizardResult{}, nil
	}
	out := SetupGatewayWizardResult{SelectedPlatforms: []string{platform}, BubbleTea: true}
	if platform == "telegram" {
		answers, err := RunSetupTelegramBubbleTeaWizard(cmd, cfg.Telegram)
		if err != nil {
			return SetupGatewayWizardResult{}, err
		}
		out.Telegram = &answers
	}
	return out, nil
}

func RunSetupTelegramBubbleTeaWizard(cmd *cobra.Command, cfg config.TelegramCfg) (SetupTelegramGatewayAnswers, error) {
	runner := setupwizard.New(
		setupwizard.WithInput(os.Stdin),
		setupwizard.WithOutput(cmd.OutOrStdout()),
	)
	result, err := runner.Run(cmd.Context(), setupTelegramGatewayWizardSteps(cfg)...)
	if errors.Is(err, setupwizard.ErrAbort) {
		return SetupTelegramGatewayAnswers{Apply: false}, nil
	}
	if err != nil {
		return SetupTelegramGatewayAnswers{}, err
	}
	answers := SetupTelegramGatewayAnswers{
		Token:        strings.TrimSpace(result.String("token")),
		AccessPolicy: NormalizeSetupValue(result.Choice("access_policy")),
		Apply:        true,
	}
	if answers.AccessPolicy == "" {
		answers.AccessPolicy = setupTelegramDefaultAccessPolicy(cfg)
	}
	if answers.AccessPolicy == "allowlist" {
		allowedResult, err := runner.Run(cmd.Context(), setupTelegramGatewayAllowedUsersSteps()...)
		if errors.Is(err, setupwizard.ErrAbort) {
			return SetupTelegramGatewayAnswers{Apply: false}, nil
		}
		if err != nil {
			return SetupTelegramGatewayAnswers{}, err
		}
		answers.AllowedUsers = strings.TrimSpace(allowedResult.String("allowed_users"))
	}
	return answers, nil
}

func setupTelegramGatewayWizardSteps(cfg config.TelegramCfg) []setupwizard.Step {
	return []setupwizard.Step{
		setupwizard.Password("token", "Telegram bot token from BotFather (blank keeps current)", setupwizard.WithPlaceholder("123456:...")),
		setupwizard.Pick("access_policy", "Telegram access policy", []setupwizard.Choice{
			{ID: "allowlist", Label: "Allowlisted Telegram user IDs"},
			{ID: "pairing", Label: "Pairing/first-run discovery"},
			{ID: "open", Label: "Open access (risky)"},
		}, setupwizard.WithDefaultChoice(setupTelegramDefaultAccessPolicy(cfg))),
	}
}

func setupTelegramGatewayAllowedUsersSteps() []setupwizard.Step {
	return []setupwizard.Step{
		setupwizard.Text("allowed_users", "Allowed Telegram user IDs (comma-separated)", setupwizard.WithPlaceholder("6586915095,12345")),
	}
}

func setupTelegramDefaultAccessPolicy(cfg config.TelegramCfg) string {
	if cfg.GuestMode {
		return "open"
	}
	if len(cfg.AllowedUserIDs) == 0 && cfg.FirstRunDiscovery {
		return "pairing"
	}
	return "allowlist"
}

func ApplySetupTelegramGatewayAnswers(cmd *cobra.Command, cfg config.TelegramCfg, answers SetupTelegramGatewayAnswers, runtime SetupGatewayRuntime) error {
	runtime = setupGatewayRuntimeDefaults(runtime)
	out := cmd.OutOrStdout()
	token := strings.TrimSpace(answers.Token)
	if token != "" && !setupTelegramBotTokenPattern.MatchString(token) {
		return runtime.exitCodeError(2, fmt.Errorf("setup telegram: invalid bot token format; expected BotFather token like 123456:ABC..."))
	}
	effectiveToken := token
	if effectiveToken == "" {
		effectiveToken = strings.TrimSpace(cfg.BotToken)
	}
	if effectiveToken == "" {
		return runtime.exitCodeError(2, fmt.Errorf("setup telegram: missing bot token; enter a Telegram bot token or configure one before enabling Telegram"))
	}

	accessPolicy := NormalizeSetupValue(answers.AccessPolicy)
	if accessPolicy == "" {
		accessPolicy = "allowlist"
	}
	allowedUsers, err := ParseSetupTelegramAllowedUsers(answers.AllowedUsers)
	if err != nil {
		return err
	}
	profileBinding := SetupGatewayProfileChannelPreview("telegram")

	fmt.Fprintln(out, "Review Telegram gateway changes")
	fmt.Fprintf(out, "  profiles.%s.channels.telegram.credential=%s\n", profileBinding.ProfileID, profileBinding.CredentialID)
	if token != "" {
		fmt.Fprintf(out, "  .env %s=[REDACTED]\n", profileBinding.SecretEnvName)
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
		return runtime.exitCodeError(2, fmt.Errorf("setup telegram: unsupported access policy %q", answers.AccessPolicy))
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

	allowedProfileChats := []string(nil)
	if chatID := strings.TrimSpace(answers.HomeChatID); chatID != "" {
		allowedProfileChats = []string{chatID}
	}
	profileBinding, err = WriteSetupGatewayProfileChannelBinding(SetupGatewayProfileChannelOptions{
		ChannelID:    "telegram",
		AllowedChats: allowedProfileChats,
		AllowedUsers: SetupInt64Strings(allowedUsers),
	})
	if err != nil {
		return fmt.Errorf("setup telegram: write profile channel binding: %w", err)
	}
	if token != "" {
		if err := WriteSetupGatewayTokenEnv(profileBinding, config.SecretEnvName("telegram.bot_token"), token); err != nil {
			return fmt.Errorf("setup telegram: write token: %w", err)
		}
		if err := WriteSetupGatewayRuntimeSecretRef("telegram.bot_token_ref", profileBinding.SecretEnvName); err != nil {
			return fmt.Errorf("setup telegram: write token ref: %w", err)
		}
	}
	switch accessPolicy {
	case "allowlist":
		if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.allowed_user_ids", FormatSetupInt64CSV(allowedUsers)); err != nil {
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

func ParseSetupTelegramAllowedUsers(value string) ([]int64, error) {
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
			return nil, NewExitCodeError(2, fmt.Errorf("setup telegram: invalid allowed user ID %q", part))
		}
		out = append(out, id)
	}
	return out, nil
}

func FormatSetupInt64CSV(values []int64) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.FormatInt(value, 10))
	}
	return strings.Join(parts, ",")
}

func RunSetupGatewayPlatformRowBacked(cmd *cobra.Command, platform string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "setup_gateway_platform_row_backed: platform=%s recommended_command=\"gormes setup gateway\"\n", platform)
	return nil
}

func RunSetupGatewayPlatform(cmd *cobra.Command, platform string, runtime SetupGatewayRuntime) error {
	runtime = setupGatewayRuntimeDefaults(runtime)
	switch NormalizeSetupValue(platform) {
	case "telegram":
		return runSetupTelegramGatewayPlatform(cmd, runtime)
	case "discord":
		return runSetupDiscordGatewayPlatform(cmd, runtime)
	case "slack":
		return runSetupSlackGatewayPlatform(cmd, runtime)
	case "whatsapp":
		return runtime.RunWhatsAppSetup(cmd)
	case "navivox":
		cfg, err := config.Load(nil)
		if err != nil {
			return fmt.Errorf("setup navivox: load config: %w", err)
		}
		return runtime.RunNavivoxGateway(cmd, cfg)
	default:
		return RunSetupGatewayPlatformRowBacked(cmd, platform)
	}
}

func runSetupTelegramGatewayPlatform(cmd *cobra.Command, runtime SetupGatewayRuntime) error {
	out := cmd.OutOrStdout()
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("setup telegram: load config: %w", err)
	}
	token, err := runtime.PromptSecret(cmd, "Telegram bot token (stored in .env, blank to keep current): ")
	if err != nil {
		return err
	}
	profileBinding := SetupGatewayProfileChannelPreview("telegram")
	if strings.TrimSpace(token) == "" && strings.TrimSpace(cfg.Telegram.BotToken) == "" {
		return runtime.exitCodeError(2, fmt.Errorf("setup telegram: missing bot token; enter a Telegram bot token or configure one before enabling Telegram"))
	}

	chatID, err := runtime.PromptString(cmd, "Allowed chat ID (blank for first-run discovery): ", "")
	if err != nil {
		return err
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		profileBinding, err = WriteSetupGatewayProfileChannelBinding(SetupGatewayProfileChannelOptions{ChannelID: "telegram"})
		if err != nil {
			return fmt.Errorf("setup telegram: write profile channel binding: %w", err)
		}
		if token != "" {
			if err := WriteSetupGatewayTokenEnv(profileBinding, config.SecretEnvName("telegram.bot_token"), token); err != nil {
				return fmt.Errorf("setup telegram: write token: %w", err)
			}
			if err := WriteSetupGatewayRuntimeSecretRef("telegram.bot_token_ref", profileBinding.SecretEnvName); err != nil {
				return fmt.Errorf("setup telegram: write token ref: %w", err)
			}
		}
		if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.first_run_discovery", "true"); err != nil {
			return fmt.Errorf("setup telegram: write discovery config: %w", err)
		}
		fmt.Fprintf(out, "Telegram profile channel configured: profiles.%s.channels.telegram\n", profileBinding.ProfileID)
		fmt.Fprintln(out, "Telegram gateway channel configured for first-run discovery.")
		return nil
	}
	if _, err := strconv.ParseInt(chatID, 10, 64); err != nil {
		return runtime.exitCodeError(2, fmt.Errorf("setup telegram: invalid allowed chat ID"))
	}
	profileBinding, err = WriteSetupGatewayProfileChannelBinding(SetupGatewayProfileChannelOptions{ChannelID: "telegram", AllowedChats: []string{chatID}})
	if err != nil {
		return fmt.Errorf("setup telegram: write profile channel binding: %w", err)
	}
	if token != "" {
		if err := WriteSetupGatewayTokenEnv(profileBinding, config.SecretEnvName("telegram.bot_token"), token); err != nil {
			return fmt.Errorf("setup telegram: write token: %w", err)
		}
		if err := WriteSetupGatewayRuntimeSecretRef("telegram.bot_token_ref", profileBinding.SecretEnvName); err != nil {
			return fmt.Errorf("setup telegram: write token ref: %w", err)
		}
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.allowed_chat_id", chatID); err != nil {
		return fmt.Errorf("setup telegram: write allowed chat ID: %w", err)
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.first_run_discovery", "false"); err != nil {
		return fmt.Errorf("setup telegram: write discovery config: %w", err)
	}
	fmt.Fprintf(out, "Telegram profile channel configured: profiles.%s.channels.telegram\n", profileBinding.ProfileID)
	fmt.Fprintln(out, "Telegram gateway channel configured.")
	return nil
}

func runSetupDiscordGatewayPlatform(cmd *cobra.Command, runtime SetupGatewayRuntime) error {
	out := cmd.OutOrStdout()
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("setup discord: load config: %w", err)
	}
	token, err := runtime.PromptSecret(cmd, "Discord bot token (stored in .env, blank to keep current): ")
	if err != nil {
		return err
	}
	profileBinding := SetupGatewayProfileChannelPreview("discord")
	if strings.TrimSpace(token) == "" && strings.TrimSpace(cfg.Discord.Token) == "" {
		return runtime.exitCodeError(2, fmt.Errorf("setup discord: missing bot token; enter a Discord bot token or configure one before enabling Discord"))
	}

	channelID, err := runtime.PromptString(cmd, "Allowed channel ID (blank for first-run discovery): ", "")
	if err != nil {
		return err
	}
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		profileBinding, err = WriteSetupGatewayProfileChannelBinding(SetupGatewayProfileChannelOptions{ChannelID: "discord"})
		if err != nil {
			return fmt.Errorf("setup discord: write profile channel binding: %w", err)
		}
		if token != "" {
			if err := WriteSetupGatewayTokenEnv(profileBinding, config.SecretEnvName("discord.token"), token); err != nil {
				return fmt.Errorf("setup discord: write token: %w", err)
			}
			if err := WriteSetupGatewayRuntimeSecretRef("discord.token_ref", profileBinding.SecretEnvName); err != nil {
				return fmt.Errorf("setup discord: write token ref: %w", err)
			}
		}
		if err := config.WriteTOMLValue(config.ConfigPath(), "discord.first_run_discovery", "true"); err != nil {
			return fmt.Errorf("setup discord: write discovery config: %w", err)
		}
		fmt.Fprintf(out, "Discord profile channel configured: profiles.%s.channels.discord\n", profileBinding.ProfileID)
		fmt.Fprintln(out, "Discord gateway channel configured for first-run discovery.")
		return nil
	}
	profileBinding, err = WriteSetupGatewayProfileChannelBinding(SetupGatewayProfileChannelOptions{ChannelID: "discord", AllowedChats: []string{channelID}})
	if err != nil {
		return fmt.Errorf("setup discord: write profile channel binding: %w", err)
	}
	if token != "" {
		if err := WriteSetupGatewayTokenEnv(profileBinding, config.SecretEnvName("discord.token"), token); err != nil {
			return fmt.Errorf("setup discord: write token: %w", err)
		}
		if err := WriteSetupGatewayRuntimeSecretRef("discord.token_ref", profileBinding.SecretEnvName); err != nil {
			return fmt.Errorf("setup discord: write token ref: %w", err)
		}
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "discord.allowed_channel_id", channelID); err != nil {
		return fmt.Errorf("setup discord: write allowed channel ID: %w", err)
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "discord.first_run_discovery", "false"); err != nil {
		return fmt.Errorf("setup discord: write discovery config: %w", err)
	}
	fmt.Fprintf(out, "Discord profile channel configured: profiles.%s.channels.discord\n", profileBinding.ProfileID)
	fmt.Fprintln(out, "Discord gateway channel configured.")
	return nil
}

func runSetupSlackGatewayPlatform(cmd *cobra.Command, runtime SetupGatewayRuntime) error {
	out := cmd.OutOrStdout()
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("setup slack: load config: %w", err)
	}
	botToken, err := runtime.PromptSecret(cmd, "Slack bot token (xoxb, stored in .env, blank to keep current): ")
	if err != nil {
		return err
	}
	profileBinding := SetupGatewayProfileChannelPreview("slack")
	appToken, err := runtime.PromptSecret(cmd, "Slack app token (xapp, stored in .env, blank to keep current): ")
	if err != nil {
		return err
	}
	appBinding := SetupGatewayProfileChannelPreview("slack_app")

	effectiveBotToken := strings.TrimSpace(botToken)
	if effectiveBotToken == "" {
		effectiveBotToken = strings.TrimSpace(cfg.Slack.BotToken)
	}
	effectiveAppToken := strings.TrimSpace(appToken)
	if effectiveAppToken == "" {
		effectiveAppToken = strings.TrimSpace(cfg.Slack.AppToken)
	}
	if effectiveBotToken == "" || effectiveAppToken == "" {
		return runtime.exitCodeError(2, fmt.Errorf("setup slack: missing Slack tokens; enter both bot and app tokens, or configure both before enabling Slack"))
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "slack.enabled", "true"); err != nil {
		return fmt.Errorf("setup slack: write enabled config: %w", err)
	}
	channelID, err := runtime.PromptString(cmd, "Allowed channel ID (blank for first-run discovery): ", "")
	if err != nil {
		return err
	}
	channelID = strings.TrimSpace(channelID)
	allowedSlackChats := []string(nil)
	if channelID != "" {
		allowedSlackChats = []string{channelID}
	}
	profileBinding, err = WriteSetupGatewayProfileChannelBinding(SetupGatewayProfileChannelOptions{ChannelID: "slack", AllowedChats: allowedSlackChats})
	if err != nil {
		return fmt.Errorf("setup slack: write profile channel binding: %w", err)
	}
	appBinding, err = WriteSetupGatewayProfileChannelCredential("slack_app")
	if err != nil {
		return fmt.Errorf("setup slack: write profile app credential: %w", err)
	}
	if botToken != "" {
		if err := WriteSetupGatewayTokenEnv(profileBinding, config.SecretEnvName("slack.bot_token"), botToken); err != nil {
			return fmt.Errorf("setup slack: write bot token: %w", err)
		}
		if err := WriteSetupGatewayRuntimeSecretRef("slack.bot_token_ref", profileBinding.SecretEnvName); err != nil {
			return fmt.Errorf("setup slack: write bot token ref: %w", err)
		}
	}
	if appToken != "" {
		if err := WriteSetupGatewayTokenEnv(appBinding, config.SecretEnvName("slack.app_token"), appToken); err != nil {
			return fmt.Errorf("setup slack: write app token: %w", err)
		}
		if err := WriteSetupGatewayRuntimeSecretRef("slack.app_token_ref", appBinding.SecretEnvName); err != nil {
			return fmt.Errorf("setup slack: write app token ref: %w", err)
		}
	}
	if channelID == "" {
		if err := config.WriteTOMLValue(config.ConfigPath(), "slack.first_run_discovery", "true"); err != nil {
			return fmt.Errorf("setup slack: write discovery config: %w", err)
		}
		fmt.Fprintf(out, "Slack profile channel configured: profiles.%s.channels.slack\n", profileBinding.ProfileID)
		fmt.Fprintln(out, "Slack gateway channel configured for first-run discovery.")
		return nil
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "slack.allowed_channel_id", channelID); err != nil {
		return fmt.Errorf("setup slack: write allowed channel ID: %w", err)
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "slack.first_run_discovery", "false"); err != nil {
		return fmt.Errorf("setup slack: write discovery config: %w", err)
	}
	fmt.Fprintf(out, "Slack profile channel configured: profiles.%s.channels.slack\n", profileBinding.ProfileID)
	fmt.Fprintln(out, "Slack gateway channel configured.")
	return nil
}

func setupGatewayRuntimeDefaults(runtime SetupGatewayRuntime) SetupGatewayRuntime {
	if runtime.NewExitCodeError == nil {
		runtime.NewExitCodeError = NewExitCodeError
	}
	if runtime.PromptString == nil {
		runtime.PromptString = setupGatewayPromptString
	}
	if runtime.PromptSecret == nil {
		runtime.PromptSecret = setupGatewayPromptSecret
	}
	if runtime.RunWhatsAppSetup == nil {
		runtime.RunWhatsAppSetup = func(cmd *cobra.Command) error {
			return RunSetupGatewayPlatformRowBacked(cmd, "whatsapp")
		}
	}
	if runtime.RunNavivoxGateway == nil {
		runtime.RunNavivoxGateway = func(cmd *cobra.Command, cfg config.Config) error {
			return RunSetupNavivoxGateway(cmd, cfg, setupGatewayNavivoxOptions(cmd, runtime))
		}
	}
	return runtime
}

func (runtime SetupGatewayRuntime) exitCodeError(code int, err error) error {
	if runtime.NewExitCodeError != nil {
		return runtime.NewExitCodeError(code, err)
	}
	return NewExitCodeError(code, err)
}

func setupGatewayNavivoxOptions(cmd *cobra.Command, runtime SetupGatewayRuntime) SetupNavivoxOptions {
	promptRuntime := SetupOptionPromptRuntime{
		PromptString:  runtime.PromptString,
		ExitCodeError: runtime.NewExitCodeError,
	}
	return SetupNavivoxOptions{
		AskYesNo: func(title, linePrompt string, defaultValue bool) (bool, bool, error) {
			return PromptSetupYesNoOption(cmd, title, linePrompt, defaultValue, promptRuntime)
		},
		PromptChoice: func(title, linePrompt, defaultID string, choices []SetupOptionChoice) (string, error) {
			return PromptSetupOptionChoice(cmd, title, linePrompt, defaultID, choices, promptRuntime)
		},
		PromptString: func(prompt, defaultValue string) (string, error) {
			return runtime.PromptString(cmd, prompt, defaultValue)
		},
		ParsePositiveInt: SetupParsePositiveInt,
		WriteProfileChannelBinding: func(opts SetupNavivoxProfileChannelOptions) (SetupNavivoxProfileChannelBinding, error) {
			binding, err := WriteSetupGatewayProfileChannelBinding(SetupGatewayProfileChannelOptions{
				ChannelID:      opts.ChannelID,
				AllowedChats:   opts.AllowedChats,
				AllowedUsers:   opts.AllowedUsers,
				RequireMention: opts.RequireMention,
				ToolProgress:   opts.ToolProgress,
			})
			return SetupNavivoxProfileChannelBinding{
				ProfileID:     binding.ProfileID,
				ChannelID:     binding.ChannelID,
				CredentialID:  binding.CredentialID,
				SecretEnvName: binding.SecretEnvName,
				RegistryPath:  binding.RegistryPath,
			}, err
		},
		WriteGatewayTokenEnv: func(binding SetupNavivoxProfileChannelBinding, legacyEnvName, token string) error {
			return WriteSetupGatewayTokenEnv(SetupGatewayProfileChannelBinding{
				ProfileID:     binding.ProfileID,
				ChannelID:     binding.ChannelID,
				CredentialID:  binding.CredentialID,
				SecretEnvName: binding.SecretEnvName,
				RegistryPath:  binding.RegistryPath,
			}, legacyEnvName, token)
		},
	}
}

func setupGatewayPromptString(cmd *cobra.Command, prompt, defaultVal string) (string, error) {
	fmt.Fprint(cmd.OutOrStdout(), prompt)
	return setupGatewayScanPromptString(cmd, defaultVal)
}

func setupGatewayPromptSecret(cmd *cobra.Command, prompt string) (string, error) {
	fmt.Fprint(cmd.OutOrStdout(), prompt)
	if file, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		input, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(cmd.OutOrStdout())
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(input)), nil
	}
	return setupGatewayScanPromptString(cmd, "")
}

func setupGatewayScanPromptString(cmd *cobra.Command, defaultVal string) (string, error) {
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

func setupGatewayConfiguredDetails(cfg config.Config) map[string]string {
	configured := map[string]string{}
	if cfg.Telegram.BotToken != "" {
		configured["telegram"] = configuredSetupTelegramGatewayStatusDetail(cfg.Telegram)
	}
	if cfg.Discord.Enabled() {
		detail := "first_run_discovery=" + strconv.FormatBool(cfg.Discord.FirstRunDiscovery)
		if cfg.Discord.AllowedChannelID != "" {
			detail = "allowed_channel_id=" + cfg.Discord.AllowedChannelID
		}
		configured["discord"] = detail
	}
	if cfg.Slack.Enabled {
		configured["slack"] = configuredSetupSlackGatewayStatusDetail(cfg.Slack)
	}
	if cfg.Navivox.Enabled {
		configured["navivox"] = fmt.Sprintf("bind=%s:%d exposure=%s auth=%s", cfg.Navivox.BindHost, cfg.Navivox.Port, cfg.Navivox.ExposureMode, cfg.Navivox.AuthMode)
	}
	return configured
}

func configuredSetupTelegramGatewayStatusDetail(cfg config.TelegramCfg) string {
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

func configuredSetupSlackGatewayStatusDetail(cfg config.SlackCfg) string {
	if missing := missingSetupSlackCredentials(cfg); len(missing) > 0 {
		return "missing_tokens=" + strings.Join(missing, ",")
	}
	detail := "first_run_discovery=" + strconv.FormatBool(cfg.FirstRunDiscovery)
	if cfg.AllowedChannelID != "" {
		detail = "allowed_channel_id=" + cfg.AllowedChannelID
	}
	return detail
}

func missingSetupSlackCredentials(cfg config.SlackCfg) []string {
	missing := []string{}
	if strings.TrimSpace(cfg.BotToken) == "" {
		missing = append(missing, "bot_token")
	}
	if strings.TrimSpace(cfg.AppToken) == "" {
		missing = append(missing, "app_token")
	}
	return missing
}

func setupGatewaySelectionSeparator(r rune) bool {
	switch r {
	case ',', ' ', '\t', '\n', ';':
		return true
	default:
		return false
	}
}

func compactSetupGatewayStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = NormalizeSetupValue(value)
		if value == "" || seen[value] {
			continue
		}
		out = append(out, value)
		seen[value] = true
	}
	return out
}

func setupGatewayPlatformOptions(cfg config.Config) []setupGatewayPlatformOption {
	return SetupGatewayPlatformOptions(cfg)
}

func setupGatewayPlatformFallbackLabel(key string) string {
	return SetupGatewayPlatformFallbackLabel(key)
}

func parseSetupGatewaySelection(input string, options []setupGatewayPlatformOption) ([]string, bool, error) {
	return ParseSetupGatewaySelection(input, options)
}

func runSetupGatewayBubbleTeaWizard(cmd *cobra.Command, cfg config.Config) (setupGatewayWizardResult, error) {
	return RunSetupGatewayBubbleTeaWizard(cmd, cfg)
}

func runSetupTelegramBubbleTeaWizard(cmd *cobra.Command, cfg config.TelegramCfg) (setupTelegramGatewayAnswers, error) {
	return RunSetupTelegramBubbleTeaWizard(cmd, cfg)
}

func applySetupTelegramGatewayAnswers(cmd *cobra.Command, cfg config.TelegramCfg, answers setupTelegramGatewayAnswers) error {
	return ApplySetupTelegramGatewayAnswers(cmd, cfg, answers, SetupGatewayRuntime{})
}

func parseSetupTelegramAllowedUsers(value string) ([]int64, error) {
	return ParseSetupTelegramAllowedUsers(value)
}

func formatSetupInt64CSV(values []int64) string {
	return FormatSetupInt64CSV(values)
}

func compactStringsSetup(values []string) []string {
	return compactSetupGatewayStrings(values)
}

func runSetupGatewayPlatformRowBacked(cmd *cobra.Command, platform string) error {
	return RunSetupGatewayPlatformRowBacked(cmd, platform)
}

func runSetupGatewayPlatform(cmd *cobra.Command, platform string, runWhatsAppSetup func(*cobra.Command) error) error {
	return RunSetupGatewayPlatform(cmd, platform, SetupGatewayRuntime{RunWhatsAppSetup: runWhatsAppSetup})
}
