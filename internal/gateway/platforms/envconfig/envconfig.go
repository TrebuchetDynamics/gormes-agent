package envconfig

import (
	"strconv"
	"strings"
	"time"
)

// Lookup returns a non-nil environment lookup function.
func Lookup(lookup func(string) string) func(string) string {
	if lookup != nil {
		return lookup
	}
	return func(string) string { return "" }
}

// PositiveFloatSeconds parses a positive duration expressed as seconds.
func PositiveFloatSeconds(raw string, fallback time.Duration) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return time.Duration(value * float64(time.Second))
}

// NonNegativeFloatSeconds parses seconds, clamping negative values to zero.
func NonNegativeFloatSeconds(raw string, fallback time.Duration) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	if value < 0 {
		value = 0
	}
	return time.Duration(value * float64(time.Second))
}

// PositiveInt parses a strictly positive integer.
func PositiveInt(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
