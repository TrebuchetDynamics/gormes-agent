package gateway

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPlatformReconnectDelay   = 30 * time.Second
	defaultChannelDisconnectTimeout = 5 * time.Second
	// defaultPlatformPauseAfterFailures mirrors Hermes
	// gateway/run.py _PAUSE_AFTER_FAILURES (PR #26600): pause a platform
	// after this many consecutive retryable failures.
	defaultPlatformPauseAfterFailures = 10
)

// RuntimeStatusWriterFunc adapts a function into RuntimeStatusWriter for
// lifecycle tests and small command seams.
type RuntimeStatusWriterFunc func(context.Context, RuntimeStatusUpdate) error

func (f RuntimeStatusWriterFunc) UpdateRuntimeStatus(ctx context.Context, update RuntimeStatusUpdate) error {
	if f == nil {
		return nil
	}
	return f(ctx, update)
}

// PlatformConnector starts or reconnects one platform without binding tests to
// live Telegram/Discord/Slack SDKs.
type PlatformConnector func(context.Context) (Channel, error)

// PlatformStartupPlan is a fakeable platform lifecycle unit.
type PlatformStartupPlan struct {
	Platform string
	Timeout  time.Duration
	Connect  PlatformConnector
}

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

// PlatformLifecycleOptions controls clock, retry, and status side effects.
type PlatformLifecycleOptions struct {
	Now        func() time.Time
	RetryDelay time.Duration
	StatusSink RuntimeStatusWriter
	// PauseAfterFailures is the per-platform circuit-breaker threshold of
	// consecutive retryable failures. <= 0 falls back to the env/default
	// (Hermes _PAUSE_AFTER_FAILURES = 10).
	PauseAfterFailures int
}

// PlatformLifecycleResult is the startup/reconnect read model.
type PlatformLifecycleResult struct {
	Connected map[string]Channel
	Failed    map[string]PlatformFailure
}

type platformConnectError struct {
	err       error
	retryable bool
}

func (e platformConnectError) Error() string {
	if e.err == nil {
		return "platform connect error"
	}
	return e.err.Error()
}

func (e platformConnectError) Unwrap() error { return e.err }

// NonRetryablePlatformError marks a platform startup failure as fatal until
// config changes, matching Hermes' non-retryable fatal-error queue removal.
func NonRetryablePlatformError(err error) error {
	return platformConnectError{err: err, retryable: false}
}

func StartPlatformLifecycle(ctx context.Context, plans []PlatformStartupPlan, opts PlatformLifecycleOptions) PlatformLifecycleResult {
	result := PlatformLifecycleResult{
		Connected: map[string]Channel{},
		Failed:    map[string]PlatformFailure{},
	}
	for _, plan := range plans {
		platform := normalizePlatformID(plan.Platform)
		if platform == "" {
			continue
		}
		writePlatformLifecycleStatus(ctx, opts.StatusSink, platform, PlatformStateStarting, "")
		ch, err := connectPlatformWithTimeout(ctx, plan)
		if err != nil {
			failure := newPlatformFailure(platform, err, 1, opts)
			if failure.Retryable {
				result.Failed[platform] = failure
			}
			writePlatformLifecycleStatus(ctx, opts.StatusSink, platform, PlatformStateFailed, failure.LastError)
			continue
		}
		if ch == nil {
			err := errors.New("platform connector returned nil channel")
			failure := newPlatformFailure(platform, err, 1, opts)
			result.Failed[platform] = failure
			writePlatformLifecycleStatus(ctx, opts.StatusSink, platform, PlatformStateFailed, failure.LastError)
			continue
		}
		result.Connected[platform] = ch
		writePlatformLifecycleStatus(ctx, opts.StatusSink, platform, PlatformStateRunning, "")
	}
	return result
}

func ReconnectFailedPlatforms(ctx context.Context, failures map[string]PlatformFailure, connected map[string]Channel, plans map[string]PlatformStartupPlan, opts PlatformLifecycleOptions) {
	if failures == nil {
		return
	}
	if connected == nil {
		connected = map[string]Channel{}
	}
	now := platformLifecycleNow(opts)
	threshold := resolvePlatformPauseThreshold(opts)
	for platform, failure := range failures {
		platform = normalizePlatformID(platform)
		if platform == "" || failure.Paused {
			// Circuit breaker open: keep the entry queued but do not
			// hammer it. Resume is the only way back, matching Hermes
			// _reconnect_watcher skipping paused platforms.
			continue
		}
		if !failure.NextRetry.IsZero() && now.Before(failure.NextRetry) {
			continue
		}
		plan, ok := plans[platform]
		if !ok {
			failure.Retryable = false
			failure.LastError = "platform reconnect plan missing"
			failures[platform] = failure
			continue
		}
		if plan.Platform == "" {
			plan.Platform = platform
		}
		writePlatformLifecycleStatus(ctx, opts.StatusSink, platform, PlatformStateStarting, "")
		ch, err := connectPlatformWithTimeout(ctx, plan)
		if err == nil && ch != nil {
			connected[platform] = ch
			delete(failures, platform)
			writePlatformLifecycleStatus(ctx, opts.StatusSink, platform, PlatformStateRunning, "")
			continue
		}
		next := newPlatformFailure(platform, firstNonNilError(err, errors.New("platform connector returned nil channel")), failure.Attempts+1, opts)
		if !next.Retryable {
			delete(failures, platform)
			writePlatformLifecycleStatus(ctx, opts.StatusSink, platform, PlatformStateFailed, next.LastError)
			continue
		}
		if next.Attempts >= threshold {
			next.Paused = true
			next.PauseReason = next.LastError
			if next.PauseReason == "" {
				next.PauseReason = "auto-paused after repeated failures"
			}
			next.NextRetry = platformPausedNextRetry(now)
			failures[platform] = next
			writePlatformLifecycleStatus(ctx, opts.StatusSink, platform, PlatformStatePaused, autoPauseGuidance(platform, next.Attempts, next.PauseReason))
			continue
		}
		failures[platform] = next
		writePlatformLifecycleStatus(ctx, opts.StatusSink, platform, PlatformStateFailed, next.LastError)
	}
}

func connectPlatformWithTimeout(ctx context.Context, plan PlatformStartupPlan) (Channel, error) {
	if plan.Connect == nil {
		return nil, NonRetryablePlatformError(errors.New("platform connector missing"))
	}
	timeout := plan.Timeout
	if timeout <= 0 {
		timeout = PlatformConnectTimeoutFromEnv(os.Getenv)
	}
	if timeout <= 0 {
		return plan.Connect(ctx)
	}
	connectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	type result struct {
		ch  Channel
		err error
	}
	done := make(chan result, 1)
	go func() {
		ch, err := plan.Connect(connectCtx)
		done <- result{ch: ch, err: err}
	}()
	select {
	case <-connectCtx.Done():
		if errors.Is(connectCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%s connect timed out after %s", normalizePlatformID(plan.Platform), timeout)
		}
		return nil, connectCtx.Err()
	case result := <-done:
		return result.ch, result.err
	}
}

func PlatformConnectTimeoutFromEnv(lookup func(string) string) time.Duration {
	if lookup == nil {
		lookup = func(string) string { return "" }
	}
	raw := strings.TrimSpace(lookup("HERMES_GATEWAY_PLATFORM_CONNECT_TIMEOUT"))
	if raw == "" {
		return 30 * time.Second
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 30 * time.Second
	}
	if value < 0 {
		value = 0
	}
	return time.Duration(value * float64(time.Second))
}

func DefaultChannelDisconnectTimeoutFromEnv() time.Duration {
	return ChannelDisconnectTimeoutFromEnv(os.Getenv)
}

func ChannelDisconnectTimeoutFromEnv(lookup func(string) string) time.Duration {
	if lookup == nil {
		lookup = func(string) string { return "" }
	}
	raw := strings.TrimSpace(lookup("HERMES_GATEWAY_ADAPTER_DISCONNECT_TIMEOUT"))
	if raw == "" {
		return defaultChannelDisconnectTimeout
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return defaultChannelDisconnectTimeout
	}
	if value < 0 {
		value = 0
	}
	return time.Duration(value * float64(time.Second))
}

func newPlatformFailure(platform string, err error, attempts int, opts PlatformLifecycleOptions) PlatformFailure {
	if attempts < 1 {
		attempts = 1
	}
	retryable := true
	var classified platformConnectError
	if errors.As(err, &classified) {
		retryable = classified.retryable
	}
	return PlatformFailure{
		Platform:  platform,
		Attempts:  attempts,
		NextRetry: platformLifecycleNow(opts).Add(platformLifecycleRetryDelay(opts)),
		LastError: sanitizePlatformLifecycleError(err),
		Retryable: retryable,
	}
}

func platformLifecycleNow(opts PlatformLifecycleOptions) time.Time {
	if opts.Now != nil {
		return opts.Now().UTC()
	}
	return time.Now().UTC()
}

func platformLifecycleRetryDelay(opts PlatformLifecycleOptions) time.Duration {
	if opts.RetryDelay > 0 {
		return opts.RetryDelay
	}
	return defaultPlatformReconnectDelay
}

func writePlatformLifecycleStatus(ctx context.Context, sink RuntimeStatusWriter, platform string, state PlatformState, message string) {
	if sink == nil {
		return
	}
	_ = sink.UpdateRuntimeStatus(ctx, RuntimeStatusUpdate{
		Platform:      platform,
		PlatformState: state,
		ErrorMessage:  message,
	})
}

func sanitizePlatformLifecycleError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	for _, marker := range []string{"token=", "api_key=", "password=", "secret="} {
		idx := strings.Index(strings.ToLower(msg), marker)
		if idx >= 0 {
			return strings.TrimSpace(msg[:idx+len(marker)]) + "[redacted]"
		}
	}
	return msg
}

func firstNonNilError(values ...error) error {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func normalizePlatformID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// PlatformPauseThresholdFromEnv resolves the per-platform circuit-breaker
// threshold. It honors HERMES_GATEWAY_PAUSE_AFTER_FAILURES first (upstream
// parity), then GORMES_GATEWAY_PAUSE_AFTER_FAILURES, and falls back to the
// Hermes default of 10 for empty, invalid, or non-positive values.
func PlatformPauseThresholdFromEnv(lookup func(string) string) int {
	if lookup == nil {
		lookup = func(string) string { return "" }
	}
	for _, key := range []string{"HERMES_GATEWAY_PAUSE_AFTER_FAILURES", "GORMES_GATEWAY_PAUSE_AFTER_FAILURES"} {
		raw := strings.TrimSpace(lookup(key))
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			return defaultPlatformPauseAfterFailures
		}
		return value
	}
	return defaultPlatformPauseAfterFailures
}

func resolvePlatformPauseThreshold(opts PlatformLifecycleOptions) int {
	if opts.PauseAfterFailures > 0 {
		return opts.PauseAfterFailures
	}
	return PlatformPauseThresholdFromEnv(os.Getenv)
}

// platformPausedNextRetry stands in for Hermes' float('inf') next_retry: a
// time far enough out that even a stale code path that misses the Paused flag
// will not fire the watcher on a paused platform.
func platformPausedNextRetry(now time.Time) time.Time {
	return now.Add(100 * 365 * 24 * time.Hour)
}

func autoPauseGuidance(platform string, attempts int, reason string) string {
	return fmt.Sprintf(
		"%s paused after %d consecutive failures (%s). Fix the underlying issue then run `/platform resume %s` to retry, or restart the gateway.",
		platform, attempts, reason, platform,
	)
}

// PauseFailedPlatform marks a queued platform paused (manual or breaker). It
// is idempotent: pausing an already-paused queued platform returns true.
// Pausing a platform that is not in the failed/retry set returns false.
func PauseFailedPlatform(failures map[string]PlatformFailure, name, reason string) bool {
	platform := normalizePlatformID(name)
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
	failure.NextRetry = platformPausedNextRetry(time.Now().UTC())
	failures[platform] = failure
	return true
}

// ResumePausedPlatform unpauses a platform, resets its attempt counter, and
// schedules an immediate retry. Returns false when the platform is not queued
// or was not paused.
func ResumePausedPlatform(failures map[string]PlatformFailure, name string, opts PlatformLifecycleOptions) bool {
	platform := normalizePlatformID(name)
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
	failure.NextRetry = platformLifecycleNow(opts)
	failures[platform] = failure
	return true
}

func knownGatewayPlatformID(name string) (string, bool) {
	want := normalizePlatformID(name)
	if want == "" {
		return "", false
	}
	for _, entry := range HermesGatewayPlatformManifest() {
		if normalizePlatformID(entry.ID) == want {
			return normalizePlatformID(entry.ID), true
		}
	}
	return "", false
}

// HandlePlatformCommand is the Go port of Hermes
// gateway/run.py:_handle_platform_command (PR #26600): the in-chat operator
// slash handler for `/platform <list|pause|resume> [name]`. It returns the
// operator-facing reply text and mutates the supplied failed-platform set for
// pause/resume, mirroring the upstream handler's effect on _failed_platforms.
func HandlePlatformCommand(text string, connected map[string]Channel, failures map[string]PlatformFailure) string {
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
		names := make([]string, 0, len(connected))
		for name := range connected {
			names = append(names, normalizePlatformID(name))
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
			failedNames = append(failedNames, normalizePlatformID(name))
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
		ResumePausedPlatform(failures, platform, PlatformLifecycleOptions{})
		return fmt.Sprintf("%s resumed; retrying on next watcher tick.", platform)

	default:
		return "Usage: /platform <list|pause|resume> [name]\n" +
			"  /platform list - show platform status\n" +
			"  /platform pause <name> - stop retrying a failing platform\n" +
			"  /platform resume <name> - re-queue a paused platform"
	}
}
