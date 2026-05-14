package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

type gatewayStatusRuntimeStore interface {
	ReadValidatedRuntimeStatusSnapshot(context.Context) (gateway.RuntimeStatusSnapshot, error)
}

var newGatewayStatusRuntimeStore = func(path string) gatewayStatusRuntimeStore {
	return gateway.NewRuntimeStatusStore(path)
}

func newGatewayStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Inspect configured gateway channels and persisted runtime state",
		RunE:  runGatewayStatus,
	}
	cmd.Flags().Bool("json", false, "print gateway status as JSON")
	return cmd
}

func runGatewayStatus(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	pairingStatus, err := gateway.NewXDGPairingStore().ReadPairingStatus(ctx)
	if err != nil {
		return fmt.Errorf("pairing status: %w", err)
	}

	runtimeSnapshot, err := newGatewayStatusRuntimeStore(config.GatewayRuntimeStatusPath()).ReadValidatedRuntimeStatusSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("runtime status: %w", err)
	}
	runtimeStatus := runtimeSnapshot.Status
	if runtimeSnapshot.Missing {
		runtimeStatus = gateway.RuntimeStatus{}
	}
	if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
		return renderGatewayStatusJSON(cmd, cfg, pairingStatus, runtimeStatus, runtimeSnapshot.Validation, runtimeSnapshot.Missing)
	}

	output := gateway.RenderStatusSummary(gateway.StatusSummary{
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
	_, err = fmt.Fprint(cmd.OutOrStdout(), output)
	return err
}

type gatewayStatusJSON struct {
	Build      buildProvenanceJSON              `json:"build"`
	Runtime    gateway.RuntimeStatus            `json:"runtime"`
	Channels   []gateway.StatusChannel          `json:"channels"`
	Pairing    gateway.PairingStatus            `json:"pairing"`
	Validation gateway.RuntimeProcessValidation `json:"validation"`
	Missing    bool                             `json:"missing"`
	Slack      string                           `json:"slack"`
}

func renderGatewayStatusJSON(cmd *cobra.Command, cfg config.Config, pairing gateway.PairingStatus, runtime gateway.RuntimeStatus, validation gateway.RuntimeProcessValidation, missing bool) error {
	// Normalize nil maps/slices on the empty/missing-runtime path so
	// `--json` consumers iterate over `[]` / `{}` instead of crashing
	// on `null`. Same convention as emitSessionListJSON /
	// gateway probe/discover.
	if runtime.Platforms == nil {
		runtime.Platforms = map[string]gateway.PlatformRuntimeStatus{}
	}
	if pairing.Platforms == nil {
		pairing.Platforms = []gateway.PairingPlatformStatus{}
	}
	if pairing.Pending == nil {
		pairing.Pending = []gateway.PairingPendingRecord{}
	}
	if pairing.Approved == nil {
		pairing.Approved = []gateway.PairingApprovedRecord{}
	}
	payload := gatewayStatusJSON{
		Build:      newBuildProvenance(),
		Runtime:    runtime,
		Channels:   configuredGatewayStatusChannels(cfg),
		Pairing:    pairing,
		Validation: validation,
		Missing:    missing,
		Slack:      doctorSlackGatewayConfig(cfg, runtime).Summary,
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func renderGatewaySlackDiagnosticLine(cfg config.Config, runtime gateway.RuntimeStatus) string {
	check := doctorSlackGatewayConfig(cfg, runtime)
	return fmt.Sprintf("gateway/slack: %s\n", check.Summary)
}

func renderGatewayStaleCodeLine(evidence *gateway.RuntimeStaleCodeEvidence) string {
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

func renderRuntimeValidationLine(validation gateway.RuntimeProcessValidation) string {
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

func configuredGatewayStatusChannels(cfg config.Config) []gateway.StatusChannel {
	channels := []gateway.StatusChannel{}
	if cfg.Telegram.BotToken != "" {
		channels = append(channels, gateway.StatusChannel{
			Name:   "telegram",
			Detail: configuredTelegramGatewayStatusDetail(cfg.Telegram),
		})
	}
	if cfg.Discord.Enabled() {
		detail := "first_run_discovery=" + strconv.FormatBool(cfg.Discord.FirstRunDiscovery)
		if cfg.Discord.AllowedChannelID != "" {
			detail = "allowed_channel_id=" + cfg.Discord.AllowedChannelID
		}
		channels = append(channels, gateway.StatusChannel{
			Name:   "discord",
			Detail: detail,
		})
	}
	if cfg.Slack.Enabled {
		channels = append(channels, gateway.StatusChannel{
			Name:   "slack",
			Detail: configuredSlackGatewayStatusDetail(cfg.Slack),
		})
	}
	if cfg.Teams.Enabled {
		channels = append(channels, gateway.StatusChannel{
			Name:   "teams",
			Detail: configuredTeamsGatewayStatusDetail(cfg.Teams),
		})
	}
	if cfg.Navibox.Enabled {
		channels = append(channels, gateway.StatusChannel{
			Name:   "navibox",
			Detail: configuredNaviboxGatewayStatusDetail(cfg.Navibox),
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

func configuredNaviboxGatewayStatusDetail(cfg config.NaviboxCfg) string {
	return fmt.Sprintf("bind=%s:%d exposure=%s auth=%s", cfg.BindHost, cfg.Port, cfg.ExposureMode, cfg.AuthMode)
}
