package llm

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/timehelpers"
)

// GetTimezone resolves the user's configured timezone, mirroring
// hermes-agent/hermes_time.py get_timezone.
func GetTimezone() *time.Location {
	return timehelpers.GetTimezone()
}

// Now returns the current wall clock in the configured timezone, or local
// time when no timezone is configured. Mirrors hermes-agent/hermes_time.py now().
func Now() time.Time {
	return timehelpers.Now()
}
