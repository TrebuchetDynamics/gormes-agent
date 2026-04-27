package main

import (
	"context"
	"log/slog"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

type fakeShutdownManager struct {
	called  chan struct{}
	release chan struct{}
}

func (f *fakeShutdownManager) Shutdown(context.Context) error {
	close(f.called)
	<-f.release
	return nil
}

func TestGatewayFreshFinalAfter_TelegramOnly(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.Config
		want time.Duration
	}{
		{
			name: "telegram default threshold",
			cfg: config.Config{
				Telegram: config.TelegramCfg{
					BotToken:               "telegram-token",
					FreshFinalAfterSeconds: 60,
				},
			},
			want: time.Minute,
		},
		{
			name: "telegram explicit zero disables",
			cfg: config.Config{
				Telegram: config.TelegramCfg{
					BotToken:               "telegram-token",
					FreshFinalAfterSeconds: 0,
				},
			},
			want: 0,
		},
		{
			name: "discord only stays disabled",
			cfg: config.Config{
				Telegram: config.TelegramCfg{FreshFinalAfterSeconds: 60},
				Discord: config.DiscordCfg{
					Token:            "discord-token",
					AllowedChannelID: "C123",
				},
			},
			want: 0,
		},
		{
			name: "slack only stays disabled",
			cfg: config.Config{
				Telegram: config.TelegramCfg{FreshFinalAfterSeconds: 60},
				Slack: config.SlackCfg{
					Enabled:          true,
					BotToken:         "xoxb-token",
					AppToken:         "xapp-token",
					AllowedChannelID: "C123",
				},
			},
			want: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mgrCfg := gatewayManagerConfig(
				tc.cfg,
				map[string]string{},
				map[string]bool{},
				nil,
				nil,
				nil,
				gateway.RestartConfig{},
			)
			if mgrCfg.FreshFinalAfter != tc.want {
				t.Fatalf("FreshFinalAfter = %s, want %s", mgrCfg.FreshFinalAfter, tc.want)
			}
		})
	}
}

func TestGatewaySignalLoopDrainsBeforeCancel(t *testing.T) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	mgr := &fakeShutdownManager{
		called:  make(chan struct{}),
		release: make(chan struct{}),
	}

	done := make(chan struct{})
	forceExit := make(chan int, 1)
	go func() {
		defer close(done)
		runGatewaySignalLoop(sigCh, 200*time.Millisecond, mgr, cancel, slog.Default(), func(code int) {
			forceExit <- code
		})
	}()

	sigCh <- syscall.SIGTERM

	select {
	case <-mgr.called:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Shutdown was not called after signal")
	}

	select {
	case <-rootCtx.Done():
		t.Fatal("root context canceled before shutdown drain completed")
	default:
	}

	close(mgr.release)

	select {
	case <-rootCtx.Done():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("root context not canceled after shutdown drain completed")
	}

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("signal loop did not return")
	}

	select {
	case code := <-forceExit:
		t.Fatalf("unexpected force exit: %d", code)
	default:
	}
}
