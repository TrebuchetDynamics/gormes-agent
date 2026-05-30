package command

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms/identity"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms/inventory"
)

// PlatformFailure records retry queue state for one failed platform.
type PlatformFailure struct {
	Platform  string
	Attempts  int
	NextRetry time.Time
	LastError string
	Retryable bool
	// Paused is set by the per-platform circuit breaker after
	// PauseAfterFailures consecutive retryable failures, or by a manual
	// `/platform pause <name>`. A paused failure stays in the queue but the
	// reconnect watcher skips it until ResumePausedPlatform clears it.
	Paused bool
	// PauseReason is the operator-facing reason recorded when Paused is set.
	PauseReason string
}

// PauseFailedPlatform marks a queued platform paused (manual or breaker). It
// is idempotent: pausing an already-paused queued platform returns true.
// Pausing a platform that is not in the failed/retry set returns false.
func PauseFailedPlatform(failures map[string]PlatformFailure, name, reason string) bool {
	platform := identity.NormalizePlatformID(name)
	if failures == nil || platform == "" {
		return false
	}
	failure, ok := failures[platform]
	if !ok {
		return false
	}
	if failure.Paused {
		return true
	}
	if strings.TrimSpace(reason) == "" {
		reason = "paused via /platform pause"
	}
	failure.Paused = true
	failure.PauseReason = reason
	failure.NextRetry = PlatformPausedNextRetry(time.Now().UTC())
	failures[platform] = failure
	return true
}

// ResumePausedPlatform unpauses a platform, resets its attempt counter, and
// schedules an immediate retry. Returns false when the platform is not queued
// or was not paused.
func ResumePausedPlatform(failures map[string]PlatformFailure, name string, now func() time.Time) bool {
	platform := identity.NormalizePlatformID(name)
	if failures == nil || platform == "" {
		return false
	}
	failure, ok := failures[platform]
	if !ok || !failure.Paused {
		return false
	}
	failure.Paused = false
	failure.PauseReason = ""
	failure.Attempts = 0
	if now == nil {
		now = time.Now
	}
	failure.NextRetry = now().UTC()
	failures[platform] = failure
	return true
}

// PlatformPausedNextRetry stands in for Hermes' float('inf') next_retry: a
// time far enough out that even a stale code path that misses the Paused flag
// will not fire the watcher on a paused platform.
func PlatformPausedNextRetry(now time.Time) time.Time {
	return now.Add(100 * 365 * 24 * time.Hour)
}

func knownGatewayPlatformID(name string) (string, bool) {
	want := identity.NormalizePlatformID(name)
	if want == "" {
		return "", false
	}
	for _, entry := range inventory.HermesGatewayPlatformManifest() {
		if identity.NormalizePlatformID(entry.ID) == want {
			return identity.NormalizePlatformID(entry.ID), true
		}
	}
	return "", false
}

// HandlePlatformCommand is the Go port of Hermes
// gateway/run.py:_handle_platform_command (PR #26600): the in-chat operator
// slash handler for `/platform <list|pause|resume> [name]`. It returns the
// operator-facing reply text and mutates the supplied failed-platform set for
// pause/resume, mirroring the upstream handler's effect on _failed_platforms.
func HandlePlatformCommand(text string, connectedNames []string, failures map[string]PlatformFailure) string {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) > 0 && strings.HasPrefix(strings.ToLower(strings.TrimLeft(fields[0], "/")), "platform") {
		fields = fields[1:]
	}
	action := "list"
	if len(fields) > 0 {
		action = strings.ToLower(fields[0])
	}
	target := ""
	if len(fields) > 1 {
		target = strings.ToLower(fields[1])
	}

	switch action {
	case "list":
		var b strings.Builder
		b.WriteString("**Gateway platforms**\n")
		names := make([]string, 0, len(connectedNames))
		for _, name := range connectedNames {
			if normalized := identity.NormalizePlatformID(name); normalized != "" {
				names = append(names, normalized)
			}
		}
		sort.Strings(names)
		if len(names) > 0 {
			b.WriteString("Connected: " + strings.Join(names, ", ") + "\n")
		} else {
			b.WriteString("Connected: (none)\n")
		}
		if len(failures) == 0 {
			b.WriteString("Failed/paused: (none)")
			return b.String()
		}
		failedNames := make([]string, 0, len(failures))
		for name := range failures {
			failedNames = append(failedNames, identity.NormalizePlatformID(name))
		}
		sort.Strings(failedNames)
		for _, name := range failedNames {
			info := failures[name]
			if info.Paused {
				reason := info.PauseReason
				if reason == "" {
					reason = "paused"
				}
				b.WriteString(fmt.Sprintf("  - %s - PAUSED (%s). Resume with `/platform resume %s`.\n", name, reason, name))
				continue
			}
			b.WriteString(fmt.Sprintf("  - %s - retrying (attempt %d)\n", name, info.Attempts))
		}
		return strings.TrimRight(b.String(), "\n")

	case "pause", "resume":
		if target == "" {
			return fmt.Sprintf("Usage: /platform %s <name>", action)
		}
		platform, ok := knownGatewayPlatformID(target)
		if !ok {
			return fmt.Sprintf("Unknown platform: %s", target)
		}
		_, queued := failures[platform]
		if action == "pause" {
			if !queued {
				return fmt.Sprintf("%s is not in the retry queue (it's either connected or not enabled).", platform)
			}
			if failures[platform].Paused {
				return fmt.Sprintf("%s is already paused.", platform)
			}
			PauseFailedPlatform(failures, platform, "paused via /platform pause")
			return fmt.Sprintf("%s paused. Resume with `/platform resume %s` or restart the gateway to reset.", platform, platform)
		}
		if !queued {
			return fmt.Sprintf("%s is not in the retry queue; nothing to resume.", platform)
		}
		if !failures[platform].Paused {
			return fmt.Sprintf("%s is already retrying; no resume needed.", platform)
		}
		ResumePausedPlatform(failures, platform, nil)
		return fmt.Sprintf("%s resumed; retrying on next watcher tick.", platform)

	default:
		return "Usage: /platform <list|pause|resume> [name]\n" +
			"  /platform list - show platform status\n" +
			"  /platform pause <name> - stop retrying a failing platform\n" +
			"  /platform resume <name> - re-queue a paused platform"
	}
}
