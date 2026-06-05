package telegram

import (
	"net/http"

	telegramnetwork "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/network"
)

// TelegramFallbackTransport retries Telegram Bot API requests through configured
// public fallback IPs when the primary api.telegram.org connection fails.
type TelegramFallbackTransport = telegramnetwork.TelegramFallbackTransport

// ParseTelegramFallbackIPEnv validates TELEGRAM_FALLBACK_IPS-style CSV values.
// It mirrors Hermes' Bot API fallback contract: public IPv4 addresses only.
func ParseTelegramFallbackIPEnv(value string) []string {
	return telegramnetwork.ParseTelegramFallbackIPEnv(value)
}

func NewTelegramFallbackTransport(fallbackIPs []string, primary http.RoundTripper) *TelegramFallbackTransport {
	return telegramnetwork.NewTelegramFallbackTransport(fallbackIPs, primary)
}

func telegramHTTPClientFromEnv() *http.Client {
	return telegramnetwork.HTTPClientFromEnv()
}
