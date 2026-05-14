package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

type fakeReloadShutdownManager struct {
	*fakeShutdownManager
	reloads   chan struct{}
	reloadErr error
}

func (f *fakeReloadShutdownManager) Reload(context.Context) error {
	f.reloads <- struct{}{}
	return f.reloadErr
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
				map[string]bool{}, nil,
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

func TestGatewayVerbosePersistsGormesDisplayPlatformOnly(t *testing.T) {
	root := t.TempDir()
	gormesHome := filepath.Join(root, "gormes")
	hermesHome := filepath.Join(root, "hermes")
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("HERMES_HOME", hermesHome)
	if err := os.MkdirAll(hermesHome, 0o755); err != nil {
		t.Fatal(err)
	}
	hermesPath := filepath.Join(hermesHome, "config.yaml")
	if err := os.WriteFile(hermesPath, []byte("display:\n  tool_progress: off\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	mgrCfg := gatewayManagerConfig(
		config.Config{Display: config.DisplayCfg{ToolProgress: "all", ToolProgressCommand: true}},
		map[string]string{},
		map[string]bool{},
		nil,
		nil,
		nil,
		nil,
		nil,
		gateway.RestartConfig{},
	)
	if mgrCfg.PersistToolProgressMode == nil {
		t.Fatal("PersistToolProgressMode = nil, want production native persistence hook")
	}
	if err := mgrCfg.PersistToolProgressMode("Telegram", "new"); err != nil {
		t.Fatalf("PersistToolProgressMode: %v", err)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Display.Platforms["telegram"].ToolProgress; got != "new" {
		t.Fatalf("Display.Platforms[telegram].ToolProgress = %q, want native persisted new", got)
	}
	hermesBody, err := os.ReadFile(hermesPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(hermesBody), "Telegram") || strings.Contains(string(hermesBody), "new") {
		t.Fatalf("Hermes config.yaml mutated by gateway /verbose persistence:\n%s", hermesBody)
	}
}

func TestSQLOpenGonchoUsesRegisteredSQLiteDriver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "goncho.db")
	db, err := sqlOpenGoncho(path)
	if err != nil {
		t.Fatalf("sqlOpenGoncho() error = %v, want registered sqlite driver", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE smoke (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("sqlite smoke table: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "?") {
			t.Fatalf("sqlOpenGoncho created literal query filename %q", entry.Name())
		}
	}
}

func TestGatewayManagerConfig_ToolProgressDisplayConfig(t *testing.T) {
	mgrCfg := gatewayManagerConfig(
		config.Config{
			Display: config.DisplayCfg{
				ToolProgress:        "new",
				ToolProgressCommand: true,
				Platforms: map[string]config.DisplayPlatformCfg{
					"telegram": {ToolProgress: "off"},
					"slack":    {ToolProgress: "verbose"},
				},
			},
		},
		map[string]string{},
		map[string]bool{}, nil,
		nil,
		nil,
		nil,
		nil,
		gateway.RestartConfig{},
	)
	if mgrCfg.ToolProgressMode != "new" {
		t.Fatalf("ToolProgressMode = %q, want global display.tool_progress", mgrCfg.ToolProgressMode)
	}
	if !mgrCfg.ToolProgressCommandEnabled {
		t.Fatal("ToolProgressCommandEnabled = false, want display.tool_progress_command")
	}
	if mgrCfg.PersistToolProgressMode == nil {
		t.Fatal("PersistToolProgressMode = nil, want production /verbose persistence hook")
	}
	if got := mgrCfg.ToolProgressModes["telegram"]; got != "off" {
		t.Fatalf("ToolProgressModes[telegram] = %q, want per-platform override", got)
	}
	if got := mgrCfg.ToolProgressModes["slack"]; got != "verbose" {
		t.Fatalf("ToolProgressModes[slack] = %q, want per-platform override", got)
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
		map[string]bool{}, nil,
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
		map[string]bool{}, nil,
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

func TestGatewaySignalLoopReloadsOnSIGHUPWithoutCancel(t *testing.T) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 2)
	mgr := &fakeReloadShutdownManager{
		fakeShutdownManager: &fakeShutdownManager{
			called:  make(chan struct{}),
			release: make(chan struct{}),
		},
		reloads: make(chan struct{}, 1),
	}

	done := make(chan struct{})
	forceExit := make(chan int, 1)
	go func() {
		defer close(done)
		runGatewaySignalLoop(sigCh, 200*time.Millisecond, mgr, cancel, slog.Default(), func(code int) {
			forceExit <- code
		})
	}()

	sigCh <- syscall.SIGHUP

	select {
	case <-mgr.reloads:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Reload was not called after SIGHUP")
	}

	select {
	case <-rootCtx.Done():
		t.Fatal("root context canceled after reload signal")
	default:
	}

	select {
	case <-mgr.called:
		t.Fatal("Shutdown was called for reload signal")
	default:
	}

	sigCh <- syscall.SIGTERM
	select {
	case <-mgr.called:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Shutdown was not called after SIGTERM")
	}
	close(mgr.release)

	select {
	case <-rootCtx.Done():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("root context not canceled after shutdown signal")
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

func TestGatewaySignalLoopDoesNotLogReloadFailureSecrets(t *testing.T) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 2)
	mgr := &fakeReloadShutdownManager{
		fakeShutdownManager: &fakeShutdownManager{
			called:  make(chan struct{}),
			release: make(chan struct{}),
		},
		reloads:   make(chan struct{}, 1),
		reloadErr: errors.New("parse config.toml: api_key=plain-secret-token"),
	}
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))

	done := make(chan struct{})
	go func() {
		defer close(done)
		runGatewaySignalLoop(sigCh, 200*time.Millisecond, mgr, cancel, log, func(int) {})
	}()

	sigCh <- syscall.SIGHUP
	select {
	case <-mgr.reloads:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Reload was not called after SIGHUP")
	}
	if strings.Contains(logs.String(), "plain-secret-token") || strings.Contains(logs.String(), "api_key") {
		t.Fatalf("reload failure log leaked secret material:\n%s", logs.String())
	}
	select {
	case <-rootCtx.Done():
		t.Fatal("root context canceled after failed reload signal")
	default:
	}

	sigCh <- syscall.SIGTERM
	select {
	case <-mgr.called:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Shutdown was not called after SIGTERM")
	}
	close(mgr.release)
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("signal loop did not return")
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
		nil,
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

func TestRegisterConfiguredGatewayChannels_TelegramPerAccountTokens(t *testing.T) {
	var registered []string
	factories := gatewayChannelFactories{
		Telegram: func(cfg config.Config, _ *slog.Logger) (gateway.Channel, error) {
			registered = append(registered, cfg.Telegram.BotToken+":"+cfg.Telegram.AccountID)
			return &telegramPerAccountFakeChannel{name: "telegram", accountID: cfg.Telegram.AccountID}, nil
		},
	}

	cfg := config.Config{
		Telegram: config.TelegramCfg{
			BotToken: "global:token",
			Accounts: map[string]config.TelegramAccountCfg{
				"main":   {BotToken: "main:token", AllowedChatID: 111},
				"mineru": {BotToken: "mineru:token", AllowedChatID: 222},
			},
		},
	}

	mgr := gateway.NewManager(gateway.ManagerConfig{}, nil, slog.Default())
	allowedChats := map[string]string{}
	allowDiscovery := map[string]bool{}
	status := gateway.NewRuntimeStatusStore(filepath.Join(t.TempDir(), "status.json"))

	count, err := registerConfiguredGatewayChannels(mgr, cfg, allowedChats, allowDiscovery, factories, status, slog.Default())
	if err != nil {
		t.Fatalf("registerConfiguredGatewayChannels: %v", err)
	}

	if count != 2 {
		t.Errorf("registered count = %d, want 2", count)
	}
	if len(registered) != 2 {
		t.Fatalf("factory calls = %d, want 2", len(registered))
	}

	want := map[string]bool{"main:token:main": true, "mineru:token:mineru": true}
	for _, r := range registered {
		if !want[r] {
			t.Errorf("unexpected factory call %q, want one of %v", r, want)
		}
		delete(want, r)
	}
	if len(want) > 0 {
		t.Errorf("missing expected factory calls: %v", want)
	}
}

func TestRegisterConfiguredGatewayChannels_NaviboxOnlyWhenEnabled(t *testing.T) {
	calls := 0
	factories := gatewayChannelFactories{
		Navibox: func(cfg config.Config, _ *slog.Logger) (gateway.Channel, error) {
			calls++
			return &telegramPerAccountFakeChannel{name: "navibox"}, nil
		},
	}
	mgr := gateway.NewManager(gateway.ManagerConfig{}, nil, slog.Default())

	registered, err := registerConfiguredGatewayChannels(mgr, config.Config{}, map[string]string{}, map[string]bool{}, factories, nil, slog.Default())
	if err != nil {
		t.Fatalf("register disabled navibox: %v", err)
	}
	if registered != 0 || calls != 0 {
		t.Fatalf("disabled navibox registered=%d calls=%d, want 0/0", registered, calls)
	}

	cfg := config.Config{
		Navibox: config.NaviboxCfg{
			Enabled:      true,
			BindHost:     "127.0.0.1",
			Port:         8765,
			ExposureMode: "local",
			AuthMode:     "pairing_token",
			Token:        "nvbx_test_token",
		},
	}
	registered, err = registerConfiguredGatewayChannels(mgr, cfg, map[string]string{}, map[string]bool{}, factories, nil, slog.Default())
	if err != nil {
		t.Fatalf("register enabled navibox: %v", err)
	}
	if registered != 1 || calls != 1 {
		t.Fatalf("enabled navibox registered=%d calls=%d, want 1/1", registered, calls)
	}
}

func TestGatewayAllowedUsersIncludesNaviboxSyntheticUser(t *testing.T) {
	allowed := gatewayAllowedUsers(config.Config{Navibox: config.NaviboxCfg{Enabled: true}})
	if !allowed["navibox"]["navibox"] {
		t.Fatalf("gatewayAllowedUsers() = %#v, want navibox synthetic user", allowed)
	}
}

type telegramPerAccountFakeChannel struct {
	name      string
	accountID string
}

func (c *telegramPerAccountFakeChannel) Name() string {
	if c.accountID != "" {
		return c.name + ":" + c.accountID
	}
	return c.name
}
func (c *telegramPerAccountFakeChannel) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	<-ctx.Done()
	return ctx.Err()
}
func (c *telegramPerAccountFakeChannel) Send(ctx context.Context, chatID, text string) (string, error) {
	return "", nil
}

func TestRegisterConfiguredGatewayChannels_DiscordPerAccountTokens(t *testing.T) {
	var registered []string
	factories := gatewayChannelFactories{
		Discord: func(cfg config.Config, _ *slog.Logger) (gateway.Channel, error) {
			registered = append(registered, cfg.Discord.Token+":"+cfg.Discord.AccountID)
			return &discordPerAccountFakeChannel{name: "discord", accountID: cfg.Discord.AccountID}, nil
		},
	}

	cfg := config.Config{
		Discord: config.DiscordCfg{
			Token: "global:token",
			Accounts: map[string]config.DiscordAccountCfg{
				"main":   {Token: "main:token", AllowedChannelID: "111"},
				"mineru": {Token: "mineru:token", AllowedChannelID: "222"},
			},
		},
	}

	mgr := gateway.NewManager(gateway.ManagerConfig{}, nil, slog.Default())
	allowedChats := map[string]string{}
	allowDiscovery := map[string]bool{}
	status := gateway.NewRuntimeStatusStore(filepath.Join(t.TempDir(), "status.json"))

	count, err := registerConfiguredGatewayChannels(mgr, cfg, allowedChats, allowDiscovery, factories, status, slog.Default())
	if err != nil {
		t.Fatalf("registerConfiguredGatewayChannels: %v", err)
	}

	if count != 2 {
		t.Errorf("registered count = %d, want 2", count)
	}
	if len(registered) != 2 {
		t.Fatalf("factory calls = %d, want 2", len(registered))
	}

	want := map[string]bool{"main:token:main": true, "mineru:token:mineru": true}
	for _, r := range registered {
		if !want[r] {
			t.Errorf("unexpected factory call %q, want one of %v", r, want)
		}
		delete(want, r)
	}
	if len(want) > 0 {
		t.Errorf("missing expected factory calls: %v", want)
	}
}

type discordPerAccountFakeChannel struct {
	name      string
	accountID string
}

func (c *discordPerAccountFakeChannel) Name() string {
	if c.accountID != "" {
		return c.name + ":" + c.accountID
	}
	return c.name
}
func (c *discordPerAccountFakeChannel) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	<-ctx.Done()
	return ctx.Err()
}
func (c *discordPerAccountFakeChannel) Send(ctx context.Context, chatID, text string) (string, error) {
	return "", nil
}

func TestRegisterConfiguredGatewayChannels_SlackPerAccountTokens(t *testing.T) {
	var registered []string
	factories := gatewayChannelFactories{
		Slack: func(cfg config.Config, _ *slog.Logger) (gateway.Channel, error) {
			registered = append(registered, cfg.Slack.BotToken+":"+cfg.Slack.AppToken+":"+cfg.Slack.AccountID)
			return &slackPerAccountFakeChannel{name: "slack", accountID: cfg.Slack.AccountID}, nil
		},
	}

	cfg := config.Config{
		Slack: config.SlackCfg{
			Enabled:  true,
			BotToken: "global:bot",
			AppToken: "global:app",
			Accounts: map[string]config.SlackAccountCfg{
				"main":   {BotToken: "main:bot", AppToken: "main:app", AllowedChannelID: "C111"},
				"mineru": {BotToken: "mineru:bot", AppToken: "mineru:app", AllowedChannelID: "C222"},
			},
		},
	}

	mgr := gateway.NewManager(gateway.ManagerConfig{}, nil, slog.Default())
	allowedChats := map[string]string{}
	allowDiscovery := map[string]bool{}
	status := gateway.NewRuntimeStatusStore(filepath.Join(t.TempDir(), "status.json"))

	count, err := registerConfiguredGatewayChannels(mgr, cfg, allowedChats, allowDiscovery, factories, status, slog.Default())
	if err != nil {
		t.Fatalf("registerConfiguredGatewayChannels: %v", err)
	}

	if count != 2 {
		t.Errorf("registered count = %d, want 2", count)
	}
	if len(registered) != 2 {
		t.Fatalf("factory calls = %d, want 2", len(registered))
	}

	want := map[string]bool{"main:bot:main:app:main": true, "mineru:bot:mineru:app:mineru": true}
	for _, r := range registered {
		if !want[r] {
			t.Errorf("unexpected factory call %q, want one of %v", r, want)
		}
		delete(want, r)
	}
	if len(want) > 0 {
		t.Errorf("missing expected factory calls: %v", want)
	}
}

type slackPerAccountFakeChannel struct {
	name      string
	accountID string
}

func (c *slackPerAccountFakeChannel) Name() string {
	if c.accountID != "" {
		return c.name + ":" + c.accountID
	}
	return c.name
}
func (c *slackPerAccountFakeChannel) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	<-ctx.Done()
	return ctx.Err()
}
func (c *slackPerAccountFakeChannel) Send(ctx context.Context, chatID, text string) (string, error) {
	return "", nil
}

func TestRegisterConfiguredGatewayChannels_MissingAccountTokenReportsDegraded(t *testing.T) {
	var registered []string
	factories := gatewayChannelFactories{
		Telegram: func(cfg config.Config, _ *slog.Logger) (gateway.Channel, error) {
			registered = append(registered, cfg.Telegram.BotToken+":"+cfg.Telegram.AccountID)
			return &telegramPerAccountFakeChannel{name: "telegram", accountID: cfg.Telegram.AccountID}, nil
		},
	}

	cfg := config.Config{
		Telegram: config.TelegramCfg{
			BotToken: "global:token",
			Accounts: map[string]config.TelegramAccountCfg{
				"main":   {BotToken: "main:token", AllowedChatID: 111},
				"mineru": {BotToken: "", AllowedChatID: 222},
			},
		},
	}

	mgr := gateway.NewManager(gateway.ManagerConfig{}, nil, slog.Default())
	allowedChats := map[string]string{}
	allowDiscovery := map[string]bool{}
	status := gateway.NewRuntimeStatusStore(filepath.Join(t.TempDir(), "status.json"))

	count, err := registerConfiguredGatewayChannels(mgr, cfg, allowedChats, allowDiscovery, factories, status, slog.Default())
	if err != nil {
		t.Fatalf("registerConfiguredGatewayChannels: %v", err)
	}

	if count != 1 {
		t.Errorf("registered count = %d, want 1", count)
	}
	if len(registered) != 1 {
		t.Fatalf("factory calls = %d, want 1", len(registered))
	}
	if registered[0] != "main:token:main" {
		t.Errorf("factory call = %q, want main:token:main", registered[0])
	}

	snap, err := status.ReadRuntimeStatusSnapshot(context.Background())
	if err != nil {
		t.Fatalf("ReadRuntimeStatusSnapshot: %v", err)
	}

	foundDegraded := false
	for _, p := range snap.Status.Platforms {
		if p.State == gateway.PlatformStateFailed && strings.Contains(p.ErrorMessage, "mineru") {
			foundDegraded = true
			break
		}
	}
	if !foundDegraded {
		t.Errorf("expected degraded status for missing mineru token, got platforms=%+v", snap.Status.Platforms)
	}
}
