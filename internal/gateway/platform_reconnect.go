package gateway

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPlatformReconnectDelay   = 30 * time.Second
	defaultChannelDisconnectTimeout = 5 * time.Second
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
}

// PlatformLifecycleOptions controls clock, retry, and status side effects.
type PlatformLifecycleOptions struct {
	Now        func() time.Time
	RetryDelay time.Duration
	StatusSink RuntimeStatusWriter
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
	for platform, failure := range failures {
		platform = normalizePlatformID(platform)
		if platform == "" || (!failure.NextRetry.IsZero() && now.Before(failure.NextRetry)) {
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
