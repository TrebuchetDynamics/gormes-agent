package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
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

func TestGatewayTelegramDynamicCommands_LoadsActiveSkillCommands(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "active", "media")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: jellyfin-jellystat-24h-summary
description: Summarize media stats
---

Run the report.
`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	commands := gatewayTelegramDynamicCommands(context.Background(), config.Config{Skills: config.SkillsCfg{Root: root}})
	for _, cmd := range commands {
		if cmd.Name == "jellyfin-jellystat-24h-summary" && cmd.Description == "Summarize media stats" {
			return
		}
	}
	t.Fatalf("gatewayTelegramDynamicCommands() = %#v, want active skill command", commands)
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

	client, err := newGatewayHermesClient(config.Config{Hermes: config.HermesCfg{
		Endpoint: server.URL,
		APIKey:   "gateway-token",
		Model:    "gpt-5.5",
		Provider: "openai-codex",
	}})
	if err != nil {
		t.Fatalf("newGatewayHermesClient error = %v", err)
	}
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

func TestGatewayManagerConfig_LiveTurnMetadataProductionWiring(t *testing.T) {
	mgrCfg := gatewayManagerConfig(
		config.Config{Hermes: config.HermesCfg{
			Model:    "gpt-5.5",
			Provider: "openai-codex",
		}},
		map[string]string{},
		map[string]bool{},
		nil,
		nil,
		nil,
		nil,
		gateway.RestartConfig{},
	)
	if mgrCfg.LiveTurnNow == nil {
		t.Fatal("LiveTurnNow is nil; production gateway metadata block would omit timestamp")
	}
	if got := mgrCfg.LiveTurnActiveModel; got == nil || got() != "gpt-5.5" {
		if got == nil {
			t.Fatal("LiveTurnActiveModel is nil")
		}
		t.Fatalf("LiveTurnActiveModel() = %q, want configured model", got())
	}
	if got := mgrCfg.LiveTurnActiveProvider; got == nil || got() != "openai-codex" {
		if got == nil {
			t.Fatal("LiveTurnActiveProvider is nil")
		}
		t.Fatalf("LiveTurnActiveProvider() = %q, want configured provider", got())
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

// TestGatewayManagerConfig_TitleModelNonNilWithBoltMap is a production smoke
// fixture. It constructs a ManagerConfig using a real *session.BoltMap and a
// hermetic HTTP stub client, then asserts that TitleModel and TitleStore are
// non-nil — proving the gateway seam is wired in production builds without
// invoking any live LLM.
func TestGatewayManagerConfig_TitleModelNonNilWithBoltMap(t *testing.T) {
	dbPath := t.TempDir() + "/title_seam_smoke.db"
	smap, err := session.OpenBolt(dbPath)
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer smap.Close()

	// Stub provider — never called; we only assert the seam exists.
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer stub.Close()

	hc := hermes.NewHTTPClientWithProvider(stub.URL, "stub-key", "openai")
	mgrCfg := gatewayManagerConfig(
		config.Config{Hermes: config.HermesCfg{Endpoint: stub.URL, Model: "stub-model"}},
		map[string]string{},
		map[string]bool{},
		smap,
		hc,
		nil,
		nil,
		gateway.RestartConfig{},
	)

	if mgrCfg.TitleModel == nil {
		t.Error("TitleModel is nil; production gateway cannot auto-title sessions")
	}
	if mgrCfg.TitleStore == nil {
		t.Error("TitleStore is nil; production gateway cannot persist auto-titles")
	}
}
