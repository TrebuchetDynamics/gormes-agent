package timehelpers

import (
	"os"
	"strings"
	"time"
)

// GetTimezone resolves the user's configured timezone, mirroring
// hermes-agent/hermes_time.py get_timezone.
//
// Priority:
//  1. GORMES_TIMEZONE env var (highest)
//  2. HERMES_TIMEZONE env var
//
// Returns nil when no timezone is configured (callers should use time.Local).
func GetTimezone() *time.Location {
	name := resolveTimezoneName()
	if name == "" {
		return nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil
	}
	return loc
}

// Now returns the current wall clock in the configured timezone, or local
// time when no timezone is configured. Mirrors hermes-agent/hermes_time.py now().
func Now() time.Time {
	loc := GetTimezone()
	if loc == nil {
		return time.Now()
	}
	return time.Now().In(loc)
}

// resolveTimezoneName reads the configured IANA timezone string.
// Mirrors hermes-agent/hermes_time.py _resolve_timezone_name.
func resolveTimezoneName() string {
	// 1. Gormes env var (highest priority)
	if tz := strings.TrimSpace(os.Getenv("GORMES_TIMEZONE")); tz != "" {
		return tz
	}
	// 2. Hermes compat env var
	if tz := strings.TrimSpace(os.Getenv("HERMES_TIMEZONE")); tz != "" {
		return tz
	}
	return ""
}
