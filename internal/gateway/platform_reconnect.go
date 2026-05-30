package gateway

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	gatewayplatforms "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms"
)

const (
	defaultPlatformReconnectDelay   = 30 * time.Second
	defaultChannelDisconnectTimeout = gatewayplatforms.DefaultChannelDisconnectTimeout
	// defaultPlatformPauseAfterFailures mirrors Hermes
	// gateway/run.py _PAUSE_AFTER_FAILURES (PR #26600): pause a platform
	// after this many consecutive retryable failures.
	defaultPlatformPauseAfterFailures = gatewayplatforms.DefaultPlatformPauseAfterFailures
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

type PlatformFailure = gatewayplatforms.PlatformFailure

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
	return gatewayplatforms.PlatformConnectTimeoutFromEnv(lookup)
}

func DefaultChannelDisconnectTimeoutFromEnv() time.Duration {
	return ChannelDisconnectTimeoutFromEnv(os.Getenv)
}

func ChannelDisconnectTimeoutFromEnv(lookup func(string) string) time.Duration {
	return gatewayplatforms.ChannelDisconnectTimeoutFromEnv(lookup)
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
	return gatewayplatforms.NormalizePlatformID(value)
}

// PlatformPauseThresholdFromEnv resolves the per-platform circuit-breaker
// threshold. It honors HERMES_GATEWAY_PAUSE_AFTER_FAILURES first (upstream
// parity), then GORMES_GATEWAY_PAUSE_AFTER_FAILURES, and falls back to the
// Hermes default of 10 for empty, invalid, or non-positive values.
func PlatformPauseThresholdFromEnv(lookup func(string) string) int {
	return gatewayplatforms.PlatformPauseThresholdFromEnv(lookup)
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
	return gatewayplatforms.PlatformPausedNextRetry(now)
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
	return gatewayplatforms.PauseFailedPlatform(failures, name, reason)
}

// ResumePausedPlatform unpauses a platform, resets its attempt counter, and
// schedules an immediate retry. Returns false when the platform is not queued
// or was not paused.
func ResumePausedPlatform(failures map[string]PlatformFailure, name string, opts PlatformLifecycleOptions) bool {
	return gatewayplatforms.ResumePausedPlatform(failures, name, func() time.Time { return platformLifecycleNow(opts) })
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
	connectedNames := make([]string, 0, len(connected))
	for name := range connected {
		connectedNames = append(connectedNames, name)
	}
	return gatewayplatforms.HandlePlatformCommand(text, connectedNames, failures)
}
