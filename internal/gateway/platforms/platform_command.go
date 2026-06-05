package platforms

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms/command"
)

// PlatformFailure records retry queue state for one failed platform.
type PlatformFailure = command.PlatformFailure

// PauseFailedPlatform marks a queued platform paused (manual or breaker). It
// is idempotent: pausing an already-paused queued platform returns true.
// Pausing a platform that is not in the failed/retry set returns false.
func PauseFailedPlatform(failures map[string]PlatformFailure, name, reason string) bool {
	return command.PauseFailedPlatform(failures, name, reason)
}

// ResumePausedPlatform unpauses a platform, resets its attempt counter, and
// schedules an immediate retry. Returns false when the platform is not queued
// or was not paused.
func ResumePausedPlatform(failures map[string]PlatformFailure, name string, now func() time.Time) bool {
	return command.ResumePausedPlatform(failures, name, now)
}

// PlatformPausedNextRetry stands in for Hermes' float('inf') next_retry: a
// time far enough out that even a stale code path that misses the Paused flag
// will not fire the watcher on a paused platform.
func PlatformPausedNextRetry(now time.Time) time.Time {
	return command.PlatformPausedNextRetry(now)
}

// HandlePlatformCommand is the Go port of Hermes
// gateway/run.py:_handle_platform_command (PR #26600): the in-chat operator
// slash handler for `/platform <list|pause|resume> [name]`. It returns the
// operator-facing reply text and mutates the supplied failed-platform set for
// pause/resume, mirroring the upstream handler's effect on _failed_platforms.
func HandlePlatformCommand(text string, connectedNames []string, failures map[string]PlatformFailure) string {
	return command.HandlePlatformCommand(text, connectedNames, failures)
}
