package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func init() {
	gatewayCmd.AddCommand(gatewayStatusCmd)
}

var gatewayStatusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Print configured channels and runtime state without starting transports",
	Long:         "Read-only summary of gateway configuration sourced from config.toml plus any runtime lifecycle state. Does not open Telegram/Discord sessions, the session map, or the memory store.",
	SilenceUsage: true,
	RunE:         runGatewayStatus,
}

func runGatewayStatus(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	view := gateway.ConfiguredChannels{}
	if cfg.Telegram.BotToken != "" {
		view.Telegram = &gateway.ConfiguredChannel{
			Platform:          "telegram",
			AllowedChatID:     telegramAllowedChatIDString(cfg.Telegram.AllowedChatID),
			FirstRunDiscovery: cfg.Telegram.FirstRunDiscovery,
		}
	}
	if cfg.Discord.Enabled() {
		view.Discord = &gateway.ConfiguredChannel{
			Platform:          "discord",
			AllowedChatID:     cfg.Discord.AllowedChannelID,
			FirstRunDiscovery: cfg.Discord.FirstRunDiscovery,
		}
	}

	fmt.Fprint(cmd.OutOrStdout(), gateway.RenderStatusSummary(view))
	return nil
}

func telegramAllowedChatIDString(id int64) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatInt(id, 10)
}
