package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
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

func TestNewGatewayHermesClient_UsesConfiguredProviderTransport(t *testing.T) {
	var sawResponsesPath bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("request path = %q, want provider-native /v1/responses", r.URL.Path)
		}
		sawResponsesPath = true
		if r.Header.Get("Authorization") != "Bearer gateway-token" {
			t.Fatalf("Authorization = %q, want configured API key", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"completed","output_text":"ok from codex"}`))
	}))
	defer server.Close()

	client := newGatewayHermesClient(config.Config{Hermes: config.HermesCfg{
		Endpoint: server.URL,
		APIKey:   "gateway-token",
		Model:    "gpt-5.5",
		Provider: "openai-codex",
	}})
	stream, err := client.OpenStream(context.Background(), hermes.ChatRequest{
		Model:    "gpt-5.5",
		Messages: []hermes.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("OpenStream error = %v", err)
	}
	defer stream.Close()
	event, err := stream.Recv(context.Background())
	if err != nil {
		t.Fatalf("Recv error = %v", err)
	}
	if event.Kind != hermes.EventToken || event.Token != "ok from codex" {
		t.Fatalf("event = %+v, want codex token event", event)
	}
	done, err := stream.Recv(context.Background())
	if err != nil {
		t.Fatalf("Recv done error = %v", err)
	}
	if done.Kind != hermes.EventDone || done.FinishReason != "stop" {
		t.Fatalf("done event = %+v, want stop", done)
	}
	if _, err := stream.Recv(context.Background()); err != io.EOF {
		t.Fatalf("third Recv err = %v, want EOF", err)
	}
	if !sawResponsesPath {
		t.Fatal("provider-native responses endpoint was not called")
	}
}

func TestGatewayManagerConfig_UsageProviderInfersProviderFromConfiguredModel(t *testing.T) {
	var sawAuthorization bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wham/usage" {
			t.Fatalf("request path = %q, want /wham/usage", r.URL.Path)
		}
		sawAuthorization = r.Header.Get("Authorization") == "Bearer gateway-token"
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"plan_type":"team",
			"rate_limit":{"primary_window":{"used_percent":40}}
		}`))
	}))
	defer server.Close()

	mgrCfg := gatewayManagerConfig(
		config.Config{Hermes: config.HermesCfg{
			Endpoint: server.URL,
			APIKey:   "gateway-token",
			Model:    "gpt-5.5",
		}},
		map[string]string{},
		map[string]bool{},
		nil,
		nil,
		nil,
		gateway.RestartConfig{},
	)
	if mgrCfg.AccountUsage == nil {
		t.Fatal("AccountUsage provider is nil")
	}
	snapshot, err := mgrCfg.AccountUsage(context.Background(), gateway.InboundEvent{Platform: "telegram", ChatID: "42", Kind: gateway.EventUsage})
	if err != nil {
		t.Fatalf("AccountUsage error = %v", err)
	}
	if snapshot.Provider != "openai-codex" || snapshot.Plan != "Team" {
		t.Fatalf("snapshot = %+v, want openai-codex Team usage", snapshot)
	}
	if !sawAuthorization {
		t.Fatalf("gateway account usage request did not use configured API key")
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
