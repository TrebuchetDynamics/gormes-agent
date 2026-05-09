package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	channelcaps "github.com/TrebuchetDynamics/gormes-agent/internal/channels"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func newChannelsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channels",
		Short: "Inspect channel capability metadata",
	}
	cmd.AddCommand(newChannelsCapabilitiesCommand())
	return cmd
}

func newChannelsCapabilitiesCommand() *cobra.Command {
	var channel string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:          "capabilities",
		Short:        "Show channel capabilities",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(nil)
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			reports, err := channelcaps.BuildCapabilityReports(channelcaps.CapabilityOptions{
				Configured: configuredChannelCapabilityDetails(cfg),
				Channel:    channel,
			})
			if err != nil {
				return newExitCodeError(1, err)
			}
			if jsonOutput {
				return renderChannelsCapabilitiesJSON(cmd, reports)
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), renderChannelsCapabilitiesText(reports))
			return err
		},
	}
	cmd.Flags().StringVar(&channel, "channel", "", "channel to inspect")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print capabilities as JSON")
	return cmd
}

func configuredChannelCapabilityDetails(cfg config.Config) map[string]string {
	details := map[string]string{}
	if cfg.Telegram.BotToken != "" {
		details["telegram"] = configuredTelegramGatewayStatusDetail(cfg.Telegram)
	}
	if cfg.Discord.Enabled() {
		detail := "first_run_discovery=" + strconv.FormatBool(cfg.Discord.FirstRunDiscovery)
		if cfg.Discord.AllowedChannelID != "" {
			detail = "allowed_channel_id=" + cfg.Discord.AllowedChannelID
		}
		details["discord"] = detail
	}
	if cfg.Slack.Enabled {
		details["slack"] = configuredSlackGatewayStatusDetail(cfg.Slack)
	}
	if cfg.Yuanbao.Enabled {
		details["yuanbao"] = cfg.Yuanbao.RedactedStatus()
	}
	return details
}

func renderChannelsCapabilitiesJSON(cmd *cobra.Command, reports []channelcaps.CapabilityReport) error {
	payload := struct {
		Build    buildProvenanceJSON            `json:"build"`
		Channels []channelcaps.CapabilityReport `json:"channels"`
	}{
		Build:    newBuildProvenance(),
		Channels: reports,
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func renderChannelsCapabilitiesText(reports []channelcaps.CapabilityReport) string {
	var b strings.Builder
	for i, report := range reports {
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "%s (%s)\n", report.DisplayName, report.Channel)
		status := "not configured"
		if report.Configured {
			status = "configured"
		}
		if report.ConfigDetail != "" {
			fmt.Fprintf(&b, "Status: %s (%s)\n", status, report.ConfigDetail)
		} else {
			fmt.Fprintf(&b, "Status: %s\n", status)
		}
		fmt.Fprintf(&b, "Support: %s\n", strings.Join(report.Features, " "))
		if len(report.Intents) > 0 {
			fmt.Fprintf(&b, "Intents: %s\n", strings.Join(report.Intents, ", "))
		}
		if len(report.Scopes) > 0 {
			fmt.Fprintf(&b, "Scopes: %s\n", strings.Join(report.Scopes, ", "))
		}
		if len(report.FormatLimitations) > 0 {
			fmt.Fprintf(&b, "Format limitations: %s\n", strings.Join(report.FormatLimitations, ", "))
		}
		if len(report.Degraded) > 0 {
			fmt.Fprintf(&b, "Degraded: %s\n", strings.Join(report.Degraded, ", "))
		}
	}
	if len(reports) > 0 {
		b.WriteString("\n")
	}
	return b.String()
}
