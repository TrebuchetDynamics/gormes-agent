package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	runtimegateway "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type gatewayStatusRuntimeStore interface {
	ReadValidatedRuntimeStatusSnapshot(context.Context) (runtimegateway.RuntimeStatusSnapshot, error)
}

var newGatewayStatusRuntimeStore = func(path string) gatewayStatusRuntimeStore {
	return runtimegateway.NewRuntimeStatusStore(path)
}

func NewStatusCommand(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Inspect configured gateway channels and persisted runtime state",
		RunE:  func(cmd *cobra.Command, args []string) error { return runGatewayStatus(cmd, args, opts) },
	}
	cmd.Flags().Bool("json", false, "print gateway status as JSON")
	return cmd
}

func runGatewayStatus(cmd *cobra.Command, _ []string, opts Options) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	pairingStatus, err := runtimegateway.NewXDGPairingStore().ReadPairingStatus(ctx)
	if err != nil {
		return fmt.Errorf("pairing status: %w", err)
	}

	runtimeSnapshot, err := newGatewayStatusRuntimeStore(config.GatewayRuntimeStatusPath()).ReadValidatedRuntimeStatusSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("runtime status: %w", err)
	}
	runtimeStatus := runtimeSnapshot.Status
	if runtimeSnapshot.Missing {
		runtimeStatus = runtimegateway.RuntimeStatus{}
	}
	if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
		return renderGatewayStatusJSON(cmd, opts, cfg, pairingStatus, runtimeStatus, runtimeSnapshot.Validation, runtimeSnapshot.Missing)
	}

	output := runtimegateway.RenderStatusSummary(runtimegateway.StatusSummary{
		Channels: configuredGatewayStatusChannels(cfg),
		Pairing:  pairingStatus,
		Runtime:  runtimeStatus,
	})
	if staleCodeLine := renderGatewayStaleCodeLine(runtimeStatus.StaleCode); staleCodeLine != "" {
		output += staleCodeLine + "\n"
	}
	output += renderGatewaySlackDiagnosticLine(cfg, runtimeStatus)
	if validationLine := renderRuntimeValidationLine(runtimeSnapshot.Validation); validationLine != "" {
		output += validationLine + "\n"
	}
	if gatewayTermuxDetected(opts) {
		output += gatewayTermuxLifecycleGuidanceLine(opts) + "\n"
		output += gatewayTermuxNotificationStatusLine(opts)
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), output)
	return err
}

type gatewayStatusJSON struct {
	Build      gormescli.BuildProvenance               `json:"build"`
	Runtime    runtimegateway.RuntimeStatus            `json:"runtime"`
	Channels   []runtimegateway.StatusChannel          `json:"channels"`
	Pairing    runtimegateway.PairingStatus            `json:"pairing"`
	Validation runtimegateway.RuntimeProcessValidation `json:"validation"`
	Missing    bool                                    `json:"missing"`
	Slack      string                                  `json:"slack"`
}

func renderGatewayStatusJSON(cmd *cobra.Command, opts Options, cfg config.Config, pairing runtimegateway.PairingStatus, runtime runtimegateway.RuntimeStatus, validation runtimegateway.RuntimeProcessValidation, missing bool) error {
	// Normalize nil maps/slices on the empty/missing-runtime path so
	// `--json` consumers iterate over `[]` / `{}` instead of crashing
	// on `null`. Same convention as emitSessionListJSON /
	// gateway probe/discover.
	if runtime.Platforms == nil {
		runtime.Platforms = map[string]runtimegateway.PlatformRuntimeStatus{}
	}
	if pairing.Platforms == nil {
		pairing.Platforms = []runtimegateway.PairingPlatformStatus{}
	}
	if pairing.Pending == nil {
		pairing.Pending = []runtimegateway.PairingPendingRecord{}
	}
	if pairing.Approved == nil {
		pairing.Approved = []runtimegateway.PairingApprovedRecord{}
	}
	payload := gatewayStatusJSON{
		Build:      gatewayBuildProvenance(opts),
		Runtime:    runtime,
		Channels:   configuredGatewayStatusChannels(cfg),
		Pairing:    pairing,
		Validation: validation,
		Missing:    missing,
		Slack:      gatewaySlackDiagnosticSummary(cfg, runtime),
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func renderGatewaySlackDiagnosticLine(cfg config.Config, runtime runtimegateway.RuntimeStatus) string {
	return fmt.Sprintf("gateway/slack: %s\n", gatewaySlackDiagnosticSummary(cfg, runtime))
}

func renderGatewayStaleCodeLine(evidence *runtimegateway.RuntimeStaleCodeEvidence) string {
	if evidence == nil || evidence.Status == "" {
		return ""
	}
	parts := []string{fmt.Sprintf("stale_code: %s", evidence.Status)}
	if evidence.BootGitSHA != "" {
		parts = append(parts, "boot="+shortGatewayStatusSHA(evidence.BootGitSHA))
	}
	if evidence.CurrentGitSHA != "" {
		parts = append(parts, "current="+shortGatewayStatusSHA(evidence.CurrentGitSHA))
	}
	if evidence.RestartSuggested {
		parts = append(parts, "restart_suggested=true")
	}
	if len(evidence.Evidence) > 0 {
		parts = append(parts, "evidence="+strings.Join(evidence.Evidence, ","))
	}
	if evidence.Message != "" {
		parts = append(parts, "message="+strconv.Quote(evidence.Message))
	}
	return strings.Join(parts, " ")
}

func shortGatewayStatusSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) <= 12 {
		return sha
	}
	return sha[:12]
}

func renderRuntimeValidationLine(validation runtimegateway.RuntimeProcessValidation) string {
	if validation.Status == "" {
		return ""
	}
	line := fmt.Sprintf("runtime_validation: %s live=%t", validation.Status, validation.Live)
	if validation.PID > 0 {
		line += fmt.Sprintf(" pid=%d", validation.PID)
	}
	if validation.ExpectedStartTime > 0 {
		line += fmt.Sprintf(" expected_start_time=%d", validation.ExpectedStartTime)
	}
	if validation.ActualStartTime > 0 {
		line += fmt.Sprintf(" actual_start_time=%d", validation.ActualStartTime)
	}
	if validation.Command != "" {
		line += fmt.Sprintf(" command=%q", validation.Command)
	}
	if validation.Message != "" {
		line += fmt.Sprintf(" message=%q", validation.Message)
	}
	return line
}

func ConfiguredStatusChannels(cfg config.Config) []runtimegateway.StatusChannel {
	return configuredGatewayStatusChannels(cfg)
}

func ConfiguredTelegramStatusDetail(cfg config.TelegramCfg) string {
	return configuredTelegramGatewayStatusDetail(cfg)
}

func ConfiguredSlackStatusDetail(cfg config.SlackCfg) string {
	return configuredSlackGatewayStatusDetail(cfg)
}

func ConfiguredTeamsStatusDetail(cfg config.TeamsCfg) string {
	return configuredTeamsGatewayStatusDetail(cfg)
}

func ConfiguredNavivoxStatusDetail(cfg config.NavivoxCfg) string {
	return configuredNavivoxGatewayStatusDetail(cfg)
}

func configuredGatewayStatusChannels(cfg config.Config) []runtimegateway.StatusChannel {
	channels := []runtimegateway.StatusChannel{}
	if cfg.Telegram.BotToken != "" {
		channels = append(channels, runtimegateway.StatusChannel{
			Name:   "telegram",
			Detail: configuredTelegramGatewayStatusDetail(cfg.Telegram),
		})
	}
	if cfg.Discord.Enabled() {
		detail := "first_run_discovery=" + strconv.FormatBool(cfg.Discord.FirstRunDiscovery)
		if cfg.Discord.AllowedChannelID != "" {
			detail = "allowed_channel_id=" + cfg.Discord.AllowedChannelID
		}
		channels = append(channels, runtimegateway.StatusChannel{
			Name:   "discord",
			Detail: detail,
		})
	}
	if cfg.Slack.Enabled {
		channels = append(channels, runtimegateway.StatusChannel{
			Name:   "slack",
			Detail: configuredSlackGatewayStatusDetail(cfg.Slack),
		})
	}
	if cfg.Teams.Enabled {
		channels = append(channels, runtimegateway.StatusChannel{
			Name:   "teams",
			Detail: configuredTeamsGatewayStatusDetail(cfg.Teams),
		})
	}
	if cfg.Navivox.Enabled {
		channels = append(channels, runtimegateway.StatusChannel{
			Name:   "navivox",
			Detail: configuredNavivoxGatewayStatusDetail(cfg.Navivox),
		})
	}
	return channels
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

func configuredTeamsGatewayStatusDetail(cfg config.TeamsCfg) string {
	return cfg.RedactedStatus()
}

func configuredNavivoxGatewayStatusDetail(cfg config.NavivoxCfg) string {
	return fmt.Sprintf("bind=%s:%d exposure=%s auth=%s", cfg.BindHost, cfg.Port, cfg.ExposureMode, cfg.AuthMode)
}

func gatewayBuildProvenance(opts Options) gormescli.BuildProvenance {
	if opts.BuildProvenance == nil {
		return gormescli.BuildProvenance{}
	}
	return opts.BuildProvenance()
}

func gatewayTermuxDetected(opts Options) bool {
	return opts.TermuxDetected != nil && opts.TermuxDetected()
}

func gatewayTermuxLifecycleGuidanceLine(opts Options) string {
	if opts.TermuxLifecycleGuidanceLine == "" {
		return "Termux gateway: foreground/tmux lifecycle; run `gormes gateway` inside tmux; termux-wake-lock and Android battery settings are best-effort only, and Android may still stop background processes."
	}
	return opts.TermuxLifecycleGuidanceLine
}

func gatewayTermuxNotificationStatusLine(opts Options) string {
	if opts.TermuxNotificationStatus == nil {
		return ""
	}
	return opts.TermuxNotificationStatus()
}

func gatewaySlackDiagnosticSummary(cfg config.Config, runtime runtimegateway.RuntimeStatus) string {
	slackCfg := cfg.Slack
	if !slackCfg.Enabled {
		return "disabled"
	}
	if missing := missingSlackCredentials(slackCfg); len(missing) > 0 {
		return "missing_tokens=" + strings.Join(missing, ",")
	}
	platform, ok := runtime.Platforms["slack"]
	switch {
	case ok && platform.State == runtimegateway.PlatformStateRunning:
		return "running"
	case ok && platform.State == runtimegateway.PlatformStateFailed:
		return "startup_failed"
	default:
		return "configured_not_running"
	}
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
