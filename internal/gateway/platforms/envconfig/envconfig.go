package envconfig

import (
	"math"
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
	value, ok := parseFloatSecondsCandidate(raw)
	if !ok || value <= 0 {
		return fallback
	}
	return durationFromSeconds(value)
}

// NonNegativeFloatSeconds parses seconds, clamping negative values to zero.
func NonNegativeFloatSeconds(raw string, fallback time.Duration) time.Duration {
	value, ok := parseFloatSecondsCandidate(raw)
	if !ok {
		return fallback
	}
	if value < 0 {
		value = 0
	}
	return durationFromSeconds(value)
}

func parseFloatSecondsCandidate(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value > maxDurationSeconds() {
		return 0, false
	}
	return value, true
}

func maxDurationSeconds() float64 {
	return float64(time.Duration(1<<63-1)) / float64(time.Second)
}

func durationFromSeconds(value float64) time.Duration {
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
