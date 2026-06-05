package config

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/channels"

type TelegramAccountCfg = channels.TelegramAccountCfg
type TelegramHomeChannelCfg = channels.TelegramHomeChannelCfg
type TelegramCfg = channels.TelegramCfg
type DiscordAccountCfg = channels.DiscordAccountCfg
type DiscordCfg = channels.DiscordCfg
type SlackAccountCfg = channels.SlackAccountCfg
type SlackCfg = channels.SlackCfg

func applyTelegramHomeChannel(cfg *Config, value string) {
	channels.ApplyTelegramHomeChannel(&cfg.Telegram, value)
}

func normalizeTelegramConfig(cfg *TelegramCfg) {
	channels.NormalizeTelegramConfig(cfg)
}
