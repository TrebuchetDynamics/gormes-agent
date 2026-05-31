package gateway

import (
	"context"
	"errors"
	"testing"
	"time"
)

type platformLifecycleTestChannel struct {
	name string
}

func (c platformLifecycleTestChannel) Name() string { return c.name }

func (c platformLifecycleTestChannel) Run(ctx context.Context, _ chan<- InboundEvent) error {
	<-ctx.Done()
	return ctx.Err()
}

func (c platformLifecycleTestChannel) Send(context.Context, string, string) (string, error) {
	return c.name + "-msg", nil
}

func TestPlatformReconnectLifecycle_StartContinuesAfterTimeout(t *testing.T) {
	now := time.Date(2026, 5, 7, 1, 2, 3, 0, time.UTC)
	statuses := []RuntimeStatusUpdate{}
	opts := PlatformLifecycleOptions{
		Now:        func() time.Time { return now },
		RetryDelay: time.Minute,
		StatusSink: RuntimeStatusWriterFunc(func(_ context.Context, update RuntimeStatusUpdate) error {
			statuses = append(statuses, update)
			return nil
		}),
	}

	result := StartPlatformLifecycle(context.Background(), []PlatformStartupPlan{
		{
			Platform: "telegram",
			Timeout:  time.Millisecond,
			Connect: func(ctx context.Context) (Channel, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			},
		},
		{
			Platform: "discord",
			Timeout:  time.Second,
			Connect: func(context.Context) (Channel, error) {
				return platformLifecycleTestChannel{name: "discord"}, nil
			},
		},
	}, opts)

	if _, ok := result.Connected["discord"]; !ok {
		t.Fatalf("connected = %+v, want discord connected after telegram timeout", result.Connected)
	}
	failure, ok := result.Failed["telegram"]
	if !ok {
		t.Fatalf("failed = %+v, want telegram queued for reconnect", result.Failed)
	}
	if !failure.Retryable || failure.Attempts != 1 || !failure.NextRetry.Equal(now.Add(time.Minute)) {
		t.Fatalf("telegram failure = %+v, want retryable attempt 1 next retry", failure)
	}
	if len(statuses) < 2 {
		t.Fatalf("statuses = %+v, want startup status evidence", statuses)
	}
}

func TestPlatformReconnectLifecycle_ConnectTimeoutFromEnv(t *testing.T) {
	if got := PlatformConnectTimeoutFromEnv(func(string) string { return "" }); got != 30*time.Second {
		t.Fatalf("default timeout = %s, want 30s", got)
	}
	if got := PlatformConnectTimeoutFromEnv(func(string) string { return "0.001" }); got != time.Millisecond {
		t.Fatalf("override timeout = %s, want 1ms", got)
	}
	if got := PlatformConnectTimeoutFromEnv(func(string) string { return "-2" }); got != 0 {
		t.Fatalf("negative override timeout = %s, want disabled timeout", got)
	}
	if got := PlatformConnectTimeoutFromEnv(func(string) string { return "not-a-number" }); got != 30*time.Second {
		t.Fatalf("invalid override timeout = %s, want 30s", got)
	}
}

func TestManager_ChannelDisconnectTimeoutFromEnv(t *testing.T) {
	if got := ChannelDisconnectTimeoutFromEnv(func(string) string { return "" }); got != 5*time.Second {
		t.Fatalf("default timeout = %s, want 5s", got)
	}
	if got := ChannelDisconnectTimeoutFromEnv(func(string) string { return "0.001" }); got != time.Millisecond {
		t.Fatalf("override timeout = %s, want 1ms", got)
	}
	if got := ChannelDisconnectTimeoutFromEnv(func(string) string { return "-2" }); got != 0 {
		t.Fatalf("negative override timeout = %s, want disabled timeout", got)
	}
	if got := ChannelDisconnectTimeoutFromEnv(func(string) string { return "not-a-number" }); got != 5*time.Second {
		t.Fatalf("invalid override timeout = %s, want 5s", got)
	}
}

func TestPlatformReconnectLifecycle_ReconnectSuccessClearsFailure(t *testing.T) {
	now := time.Date(2026, 5, 7, 2, 0, 0, 0, time.UTC)
	failure := PlatformFailure{
		Platform:  "telegram",
		Attempts:  1,
		Retryable: true,
		NextRetry: now.Add(-time.Second),
		LastError: "telegram connect timed out after 30s",
	}
	tests := []struct {
		name       string
		failureKey string
	}{
		{name: "canonical queued key", failureKey: "telegram"},
		{name: "noncanonical queued key", failureKey: " Telegram "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failures := map[string]PlatformFailure{tt.failureKey: failure}
			connected := map[string]Channel{}

			ReconnectFailedPlatforms(context.Background(), failures, connected, map[string]PlatformStartupPlan{
				"telegram": {
					Platform: "telegram",
					Timeout:  time.Second,
					Connect: func(context.Context) (Channel, error) {
						return platformLifecycleTestChannel{name: "telegram"}, nil
					},
				},
			}, PlatformLifecycleOptions{Now: func() time.Time { return now }, RetryDelay: time.Minute})

			if len(failures) != 0 {
				t.Fatalf("failures = %+v, want cleared after successful reconnect", failures)
			}
			if _, ok := connected["telegram"]; !ok {
				t.Fatalf("connected = %+v, want telegram installed after reconnect", connected)
			}
		})
	}
}

func TestPlatformReconnectLifecycle_NonRetryableRemovedAndRetryableIncremented(t *testing.T) {
	now := time.Date(2026, 5, 7, 3, 0, 0, 0, time.UTC)
	failures := map[string]PlatformFailure{
		"telegram": {Platform: "telegram", Attempts: 1, Retryable: true, NextRetry: now.Add(-time.Second)},
		"discord":  {Platform: "discord", Attempts: 2, Retryable: true, NextRetry: now.Add(-time.Second)},
	}
	connected := map[string]Channel{}

	ReconnectFailedPlatforms(context.Background(), failures, connected, map[string]PlatformStartupPlan{
		"telegram": {
			Platform: "telegram",
			Timeout:  time.Second,
			Connect: func(context.Context) (Channel, error) {
				return nil, NonRetryablePlatformError(errors.New("bad token"))
			},
		},
		"discord": {
			Platform: "discord",
			Timeout:  time.Second,
			Connect: func(context.Context) (Channel, error) {
				return nil, errors.New("temporary websocket failure")
			},
		},
	}, PlatformLifecycleOptions{Now: func() time.Time { return now }, RetryDelay: 30 * time.Second})

	if _, ok := failures["telegram"]; ok {
		t.Fatalf("failures = %+v, want nonretryable telegram removed", failures)
	}
	discord := failures["discord"]
	if !discord.Retryable || discord.Attempts != 3 || !discord.NextRetry.Equal(now.Add(30*time.Second)) {
		t.Fatalf("discord failure = %+v, want retry attempt increment and next retry", discord)
	}
}
