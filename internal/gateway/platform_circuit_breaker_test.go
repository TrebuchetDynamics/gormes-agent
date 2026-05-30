package gateway

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

var errMockPlatformDown = errors.New("platform connection refused")

func TestPlatformPauseThresholdFromEnv(t *testing.T) {
	if got := PlatformPauseThresholdFromEnv(func(string) string { return "" }); got != 10 {
		t.Fatalf("default threshold = %d, want 10 (Hermes _PAUSE_AFTER_FAILURES)", got)
	}
	if got := PlatformPauseThresholdFromEnv(func(k string) string {
		if k == "HERMES_GATEWAY_PAUSE_AFTER_FAILURES" {
			return "3"
		}
		return ""
	}); got != 3 {
		t.Fatalf("HERMES env override = %d, want 3", got)
	}
	if got := PlatformPauseThresholdFromEnv(func(k string) string {
		if k == "GORMES_GATEWAY_PAUSE_AFTER_FAILURES" {
			return "5"
		}
		return ""
	}); got != 5 {
		t.Fatalf("GORMES env override = %d, want 5", got)
	}
	if got := PlatformPauseThresholdFromEnv(func(string) string { return "not-a-number" }); got != 10 {
		t.Fatalf("invalid env fallback = %d, want 10", got)
	}
}

func TestPlatformCircuitBreakerPausesAfterThreshold(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	var pausedStatus []RuntimeStatusUpdate
	opts := PlatformLifecycleOptions{
		Now:                func() time.Time { return now },
		RetryDelay:         time.Minute,
		PauseAfterFailures: 3,
		StatusSink: RuntimeStatusWriterFunc(func(_ context.Context, u RuntimeStatusUpdate) error {
			if u.PlatformState == PlatformStatePaused {
				pausedStatus = append(pausedStatus, u)
			}
			return nil
		}),
	}
	failures := map[string]PlatformFailure{
		"whatsapp": {Platform: "whatsapp", Attempts: 2, Retryable: true, NextRetry: now.Add(-time.Second)},
	}
	plans := map[string]PlatformStartupPlan{
		"whatsapp": {Platform: "whatsapp", Connect: func(context.Context) (Channel, error) {
			return nil, errMockPlatformDown
		}},
	}

	ReconnectFailedPlatforms(context.Background(), failures, map[string]Channel{}, plans, opts)

	got, ok := failures["whatsapp"]
	if !ok {
		t.Fatalf("paused platform must stay in the failed set, got removed")
	}
	if !got.Paused {
		t.Fatalf("attempt %d >= threshold 3 must pause the platform; Paused=false", got.Attempts)
	}
	if got.PauseReason == "" {
		t.Fatalf("paused platform must record a pause reason")
	}
	if !got.NextRetry.After(now.Add(365 * 24 * time.Hour)) {
		t.Fatalf("paused next_retry must be far in the future, got %s", got.NextRetry)
	}
	if len(pausedStatus) == 0 {
		t.Fatalf("auto-pause must emit a PlatformStatePaused runtime status update")
	}
}

func TestPlatformReconnectSkipsPausedPlatforms(t *testing.T) {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	connectCalled := false
	failures := map[string]PlatformFailure{
		"whatsapp": {
			Platform: "whatsapp", Attempts: 12, Retryable: true,
			Paused: true, PauseReason: "x", NextRetry: now.Add(-time.Hour),
		},
	}
	plans := map[string]PlatformStartupPlan{
		"whatsapp": {Platform: "whatsapp", Connect: func(context.Context) (Channel, error) {
			connectCalled = true
			return platformLifecycleTestChannel{name: "whatsapp"}, nil
		}},
	}
	ReconnectFailedPlatforms(context.Background(), failures, map[string]Channel{}, plans,
		PlatformLifecycleOptions{Now: func() time.Time { return now }})

	if connectCalled {
		t.Fatalf("reconnect watcher must skip paused platforms (connect was called)")
	}
	if f, ok := failures["whatsapp"]; !ok || !f.Paused {
		t.Fatalf("paused platform must remain queued and paused after a watcher tick")
	}
}

func TestPlatformCommandResolvesToGatewaySlashHandler(t *testing.T) {
	got := ResolveGatewayCommandDispatch("/platform pause whatsapp")
	if !got.Known {
		t.Fatalf("/platform must be a known gateway slash command")
	}
	if got.RawCommand != "platform" || got.Canonical != "platform" {
		t.Fatalf("raw/canonical = %q/%q, want platform/platform (got=%+v)", got.RawCommand, got.Canonical, got)
	}
	if got.Kind != EventPlatformControl {
		t.Fatalf("kind = %v, want EventPlatformControl", got.Kind)
	}
	if got.RawArgs != "pause whatsapp" {
		t.Fatalf("raw args = %q, want %q", got.RawArgs, "pause whatsapp")
	}
	// The pre-existing plural /platforms status command must stay distinct.
	if plural := ResolveGatewayCommandDispatch("/platforms"); plural.Kind != EventGateway {
		t.Fatalf("/platforms must remain EventGateway, got %v", plural.Kind)
	}
}

func TestPlatformCommandParseAndManagerDispatchE2E(t *testing.T) {
	kind, body := ParseInboundText("/platform list")
	if kind != EventPlatformControl || body != "/platform list" {
		t.Fatalf("ParseInboundText(/platform list) = (%v, %q), want EventPlatformControl with full body", kind, body)
	}

	ch := newFakeChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"telegram": "42"}}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(context.Background(), InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "u",
		MsgID:    "m-platform-list",
		Kind:     kind,
		Text:     body,
	}); err != nil {
		t.Fatalf("handleInbound(/platform list): %v", err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent messages = %d, want one platform command response", len(sent))
	}
	if !strings.Contains(sent[0].Text, "Gateway platforms") || !strings.Contains(sent[0].Text, "telegram") {
		t.Fatalf("platform command response missing connected platform:\n%s", sent[0].Text)
	}
	if submits := fk.submitsSnapshot(); len(submits) != 0 {
		t.Fatalf("/platform command must not submit to kernel, got submits=%+v", submits)
	}
}
