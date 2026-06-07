package channels

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/channels/chat"

type TelegramAccountCfg = chat.TelegramAccountCfg
type TelegramHomeChannelCfg = chat.TelegramHomeChannelCfg
type TelegramCfg = chat.TelegramCfg

func ApplyTelegramHomeChannel(cfg *TelegramCfg, value string) {
	chat.ApplyTelegramHomeChannel(cfg, value)
}

func NormalizeTelegramConfig(cfg *TelegramCfg) {
	chat.NormalizeTelegramConfig(cfg)
}

type DiscordAccountCfg = chat.DiscordAccountCfg
type DiscordCfg = chat.DiscordCfg

type SlackAccountCfg = chat.SlackAccountCfg
type SlackCfg = chat.SlackCfg
