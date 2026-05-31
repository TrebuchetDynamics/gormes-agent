package limits

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

const (
	LimitMinimum = 1
	LimitMaximum = 100

	SearchMembersDefaultLimit = 20
	FetchMessagesDefaultLimit = 50
)

// Evidence is the stable evidence label returned with normalized Discord
// limit values.
type Evidence string

const (
	EvidenceProvided  Evidence = "discord_limit_provided"
	EvidenceClamped   Evidence = "discord_limit_clamped"
	EvidenceDefaulted Evidence = "discord_limit_defaulted"
)

// Normalization is the bounded value future Discord REST handlers should use,
// plus evidence for operator-visible degraded input reporting.
type Normalization struct {
	Limit    int
	Evidence Evidence
}

// NormalizeDiscordLimit coerces the model-provided limit argument for Discord
// actions that expose bounded result limits.
func Normalize(action string, arguments map[string]any) Normalization {
	defaultLimit := discordDefaultLimit(action)
	raw, ok := arguments["limit"]
	if !ok || raw == nil {
		return Normalization{Limit: defaultLimit, Evidence: EvidenceDefaulted}
	}

	limit, ok := coerceDiscordLimit(raw)
	if !ok {
		return Normalization{Limit: defaultLimit, Evidence: EvidenceDefaulted}
	}
	if limit < LimitMinimum {
		return Normalization{Limit: LimitMinimum, Evidence: EvidenceClamped}
	}
	if limit > LimitMaximum {
		return Normalization{Limit: LimitMaximum, Evidence: EvidenceClamped}
	}
	return Normalization{Limit: limit, Evidence: EvidenceProvided}
}

func discordDefaultLimit(action string) int {
	switch action {
	case "search_members":
		return SearchMembersDefaultLimit
	case "fetch_messages":
		return FetchMessagesDefaultLimit
	default:
		return FetchMessagesDefaultLimit
	}
}

func coerceDiscordLimit(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int8:
		return int(value), true
	case int16:
		return int(value), true
	case int32:
		return int(value), true
	case int64:
		return clampInt64ToInt(value), true
	case uint:
		return clampUint64ToInt(uint64(value)), true
	case uint8:
		return int(value), true
	case uint16:
		return int(value), true
	case uint32:
		return clampUint64ToInt(uint64(value)), true
	case uint64:
		return clampUint64ToInt(value), true
	case float32:
		return coerceDiscordFloat(float64(value))
	case float64:
		return coerceDiscordFloat(value)
	case string:
		return coerceDiscordString(value)
	case json.Number:
		return coerceDiscordJSONNumber(value)
	default:
		return 0, false
	}
}

func coerceDiscordString(value string) (int, bool) {
	limit, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	return limit, true
}

func coerceDiscordJSONNumber(value json.Number) (int, bool) {
	if limit, err := value.Int64(); err == nil {
		return clampInt64ToInt(limit), true
	}
	limit, err := value.Float64()
	if err != nil {
		return 0, false
	}
	return coerceDiscordFloat(limit)
}

func coerceDiscordFloat(value float64) (int, bool) {
	if math.IsNaN(value) {
		return 0, false
	}
	if math.IsInf(value, 1) {
		return LimitMaximum + 1, true
	}
	if math.IsInf(value, -1) {
		return LimitMinimum - 1, true
	}
	if value < LimitMinimum {
		return LimitMinimum - 1, true
	}
	if value > LimitMaximum {
		return LimitMaximum + 1, true
	}
	if value != math.Trunc(value) {
		return 0, false
	}
	if value > float64(math.MaxInt) {
		return math.MaxInt, true
	}
	if value < float64(math.MinInt) {
		return math.MinInt, true
	}
	return int(value), true
}

func clampInt64ToInt(value int64) int {
	if value > int64(math.MaxInt) {
		return math.MaxInt
	}
	if value < int64(math.MinInt) {
		return math.MinInt
	}
	return int(value)
}

func clampUint64ToInt(value uint64) int {
	if value > uint64(math.MaxInt) {
		return math.MaxInt
	}
	return int(value)
}
