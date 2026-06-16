package startup

import (
	"errors"
	"net"
	"regexp"
	"strings"
)

type Code string

const (
	CodeBotTokenLock         Code = "telegram_bot_token_lock"
	CodePollingConflict      Code = "telegram_polling_conflict"
	CodeConnectError         Code = "telegram_connect_error"
	CodeWebhookSecretMissing Code = "telegram_webhook_secret_missing"
)

// Error is the typed operator evidence returned when Telegram startup must
// fail before message ingress begins.
type Error struct {
	Code      Code
	Message   string
	Retryable bool
	Err       error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) == "" {
		return string(e.Code)
	}
	return string(e.Code) + ": " + e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewError(code Code, message string, retryable bool, err error) *Error {
	return &Error{
		Code:      code,
		Message:   SanitizeText(message),
		Retryable: retryable,
		Err:       err,
	}
}

func LooksLikePollingConflict(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "terminated by other getupdates request") ||
		strings.Contains(text, "another bot instance is running") ||
		strings.Contains(text, "conflict")
}

func LooksLikeNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	text := strings.ToLower(err.Error())
	for _, needle := range []string{
		"temporary failure in name resolution",
		"no such host",
		"connection refused",
		"connection reset",
		"network is unreachable",
		"i/o timeout",
		"timeout",
		"too many requests",
		"retry after",
	} {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func SanitizeError(err error) string {
	if err == nil {
		return ""
	}
	return SanitizeText(err.Error())
}

var (
	botTokenPattern = regexp.MustCompile(`\bbot\d{5,}:[A-Za-z0-9_-]{6,}\b`)
	tokenPattern    = regexp.MustCompile(`\b\d{5,}:[A-Za-z0-9_-]{6,}\b`)
	bearerPattern   = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]+`)
)

func SanitizeText(text string) string {
	text = strings.TrimSpace(text)
	text = botTokenPattern.ReplaceAllString(text, "bot<redacted-telegram-token>")
	text = tokenPattern.ReplaceAllString(text, "<redacted-telegram-token>")
	text = bearerPattern.ReplaceAllString(text, "Bearer <redacted>")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 500 {
		return text[:500]
	}
	return text
}
