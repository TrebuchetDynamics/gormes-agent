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

func init() {
	gatewayCmd.AddCommand(gatewayStatusCmd)
}

type gatewayStatusRuntimeStore interface {
	ReadValidatedRuntimeStatusSnapshot(context.Context) (gateway.RuntimeStatusSnapshot, error)
}

var newGatewayStatusRuntimeStore = func(path string) gatewayStatusRuntimeStore {
	return gateway.NewRuntimeStatusStore(path)
}

var gatewayStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Inspect configured gateway channels and persisted runtime state",
	RunE:  runGatewayStatus,
}

func init() {
	gatewayStatusCmd.Flags().Bool("json", false, "print gateway status as JSON")
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
	output += renderGatewaySlackDiagnosticLine(cfg, runtimeStatus)
	if validationLine := renderRuntimeValidationLine(runtimeSnapshot.Validation); validationLine != "" {
		output += validationLine + "\n"
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), output)
	return err
}

type gatewayStatusJSON struct {
	Runtime    gateway.RuntimeStatus            `json:"runtime"`
	Channels   []gateway.StatusChannel          `json:"channels"`
	Pairing    gateway.PairingStatus            `json:"pairing"`
	Validation gateway.RuntimeProcessValidation `json:"validation"`
	Missing    bool                             `json:"missing"`
	Slack      string                           `json:"slack"`
}

func renderGatewayStatusJSON(cmd *cobra.Command, cfg config.Config, pairing gateway.PairingStatus, runtime gateway.RuntimeStatus, validation gateway.RuntimeProcessValidation, missing bool) error {
	payload := gatewayStatusJSON{
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
		detail := "first_run_discovery=" + strconv.FormatBool(cfg.Telegram.FirstRunDiscovery)
		if cfg.Telegram.AllowedChatID != 0 {
			detail = "allowed_chat_id=" + strconv.FormatInt(cfg.Telegram.AllowedChatID, 10)
		}
		channels = append(channels, gateway.StatusChannel{
			Name:   "telegram",
			Detail: detail,
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
	return channels
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
