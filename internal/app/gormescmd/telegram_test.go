package gormescmd

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	apptelegram "github.com/TrebuchetDynamics/gormes-agent/internal/app/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	gatewaymodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestTelegramStartupFallsBackWhenSessionDBLocked(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	locked, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("lock session DB: %v", err)
	}
	defer locked.Close()

	smap, boltMap, notice, err := apptelegram.OpenTelegramSessionMap()
	if err != nil {
		t.Fatalf("openTelegramSessionMap: %v", err)
	}
	defer smap.Close()
	if boltMap != nil {
		t.Fatalf("boltMap = %#v, want nil fallback while sessions.db is locked", boltMap)
	}
	if _, ok := smap.(*session.MemMap); !ok {
		t.Fatalf("session map = %T, want *session.MemMap fallback", smap)
	}
	for _, want := range []string{"telegram session state: in-memory", "sessions.db locked"} {
		if !strings.Contains(notice, want) {
			t.Fatalf("notice missing %q: %q", want, notice)
		}
	}
}

func TestTelegramManagerConfig_LiveTurnMetadataProductionWiring(t *testing.T) {
	mgrCfg := apptelegram.TelegramManagerConfig(
		config.Config{Hermes: config.HermesCfg{
			Model:    "gpt-5.5",
			Provider: "openai-codex",
		}},
		nil,
		func(cfg config.Config, allowedChats map[string]string, allowDiscovery map[string]bool, allowedWhitelists map[string]gateway.WhitelistConfig, smap session.Map) gateway.ManagerConfig {
			return gatewayManagerConfig(cfg, allowedChats, allowDiscovery, allowedWhitelists, smap, nil, nil, nil, gateway.RestartConfig{})
		},
	)
	if mgrCfg.LiveTurnNow == nil {
		t.Fatal("LiveTurnNow is nil; production telegram metadata block would omit timestamp")
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

func TestTelegramProductionProviderPayloadIncludesOperatorContext(t *testing.T) {
	operatorRoot := t.TempDir()
	writeTelegramFixtureFile(t, operatorRoot, "SOUL.md", "You are Gormes, not ChatGPT.")
	writeTelegramFixtureFile(t, operatorRoot, "USER.md", "# User\nName: Juan")
	writeTelegramFixtureFile(t, operatorRoot, "MEMORY.md", "# Memory\nGormes identity must persist.")
	workdir := filepath.Join(operatorRoot, "gormes-agent")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	t.Setenv("TERMINAL_CWD", workdir)
	t.Setenv("GORMES_HOME", filepath.Join(t.TempDir(), "empty-gormes-home"))
	t.Setenv("HERMES_HOME", "")

	provider := llm.NewMockClient()
	provider.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "I am Gormes."},
		{Kind: llm.EventDone, FinishReason: "stop"},
	}, "sess-provider")
	k := kernel.New(kernel.Config{
		Model:     "gpt-5.5",
		Endpoint:  "http://mock-provider",
		Admission: kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, provider, store.NewNoop(), telemetry.New(), slog.Default())

	ch := newTelegramTestChannel()
	smap := session.NewMemMap()
	mgrCfg := apptelegram.TelegramManagerConfig(config.Config{
		Hermes: config.HermesCfg{
			Model:    "gpt-5.5",
			Provider: "openai-codex",
		},
		Telegram: config.TelegramCfg{AllowedChatID: 42},
	}, smap, func(cfg config.Config, allowedChats map[string]string, allowDiscovery map[string]bool, allowedWhitelists map[string]gateway.WhitelistConfig, smap session.Map) gateway.ManagerConfig {
		return gatewayManagerConfig(cfg, allowedChats, allowDiscovery, allowedWhitelists, smap, nil, nil, nil, gateway.RestartConfig{})
	})
	// Stabilize the golden metadata while still exercising the production
	// telegramManagerConfig path for model/provider and context discovery.
	mgrCfg.LiveTurnNow = func() time.Time { return time.Date(2026, 4, 29, 16, 55, 0, 0, time.UTC) }

	m := gateway.NewManager(mgrCfg, k, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() { _ = k.Run(ctx) }()
	go func() { _ = m.Run(ctx) }()

	ch.push(gateway.InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "user-juan",
		MsgID:    "tg-msg-1",
		Kind:     gateway.EventSubmit,
		Text:     "What's your name?",
	})

	waitForTelegramProviderRequest(t, time.Second, func() bool {
		return len(provider.Requests()) == 1
	})
	req := provider.Requests()[0]
	if req.Model != "gpt-5.5" {
		t.Fatalf("provider request model = %q, want configured model", req.Model)
	}
	if len(req.Messages) < 2 || req.Messages[0].Role != "system" {
		t.Fatalf("provider request messages = %#v, want leading system context before user", req.Messages)
	}
	system := req.Messages[0].Content
	for _, want := range []string{
		"You are Gormes, not ChatGPT.",
		"# User\nName: Juan",
		"# Memory\nGormes identity must persist.",
		"Conversation started:",
		"Model: gpt-5.5",
		"Provider: openai-codex",
		"## Current Session Context",
		"**Source:** telegram chat `42`",
		"**User ID:** `user-juan`",
	} {
		if !strings.Contains(system, want) {
			t.Fatalf("provider system prompt missing %q in:\n%s", want, system)
		}
	}
	last := req.Messages[len(req.Messages)-1]
	if last.Role != "user" || last.Content != "What's your name?" {
		t.Fatalf("provider final user message = %+v, want Telegram submit", last)
	}
	for _, msg := range req.Messages {
		if msg.Role == "assistant" && strings.Contains(msg.Content, "Gormes") {
			t.Fatalf("provider request unexpectedly contains assistant identity postprocessing: %#v", req.Messages)
		}
	}
}

func TestTelegramProductionProviderPayloadUsesConfiguredTerminalWorkspace(t *testing.T) {
	root := t.TempDir()
	sourceCheckout := filepath.Join(root, "workspace-mineru", "gormes-agent")
	if err := os.MkdirAll(sourceCheckout, 0o755); err != nil {
		t.Fatalf("mkdir source checkout: %v", err)
	}
	workspace := filepath.Join(root, "workspace-gormes")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	writeTelegramFixtureFile(t, sourceCheckout, "AGENTS.md", "Historical source checkout: workspace-mineru")
	writeTelegramFixtureFile(t, workspace, "AGENTS.md", "Runtime workspace: workspace-gormes")
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	if err := os.Chdir(sourceCheckout); err != nil {
		t.Fatalf("Chdir(sourceCheckout): %v", err)
	}
	t.Setenv("TERMINAL_CWD", "")
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes-home"))
	t.Setenv("HERMES_HOME", "")

	provider := llm.NewMockClient()
	provider.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "workspace ok"},
		{Kind: llm.EventDone, FinishReason: "stop"},
	}, "sess-provider")
	k := kernel.New(kernel.Config{
		Model:     "gpt-5.5",
		Endpoint:  "http://mock-provider",
		Admission: kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, provider, store.NewNoop(), telemetry.New(), slog.Default())

	ch := newTelegramTestChannel()
	smap := session.NewMemMap()
	mgrCfg := apptelegram.TelegramManagerConfig(config.Config{
		Hermes: config.HermesCfg{
			Model:    "gpt-5.5",
			Provider: "openai-codex",
		},
		Telegram: config.TelegramCfg{AllowedChatID: 42},
		Terminal: config.TerminalCfg{CWD: workspace},
	}, smap, func(cfg config.Config, allowedChats map[string]string, allowDiscovery map[string]bool, allowedWhitelists map[string]gateway.WhitelistConfig, smap session.Map) gateway.ManagerConfig {
		return gatewayManagerConfig(cfg, allowedChats, allowDiscovery, allowedWhitelists, smap, nil, nil, nil, gateway.RestartConfig{})
	})
	mgrCfg.LiveTurnNow = func() time.Time { return time.Date(2026, 5, 5, 6, 20, 0, 0, time.UTC) }

	m := gateway.NewManager(mgrCfg, k, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() { _ = k.Run(ctx) }()
	go func() { _ = m.Run(ctx) }()

	ch.push(gateway.InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "user-juan",
		MsgID:    "tg-msg-workspace",
		Kind:     gateway.EventSubmit,
		Text:     "What's your workspace?",
	})

	waitForTelegramProviderRequest(t, time.Second, func() bool {
		return len(provider.Requests()) == 1
	})
	req := provider.Requests()[0]
	if len(req.Messages) == 0 || req.Messages[0].Role != "system" {
		t.Fatalf("provider request messages = %#v, want leading system context", req.Messages)
	}
	system := req.Messages[0].Content
	for _, want := range []string{
		"Active workspace: `" + workspace + "`",
		"Current working directory: `" + workspace + "`",
		"Runtime workspace: workspace-gormes",
	} {
		if !strings.Contains(system, want) {
			t.Fatalf("provider system prompt missing %q in:\n%s", want, system)
		}
	}
	if strings.Contains(system, "Active workspace: `"+filepath.Dir(sourceCheckout)+"`") {
		t.Fatalf("provider system prompt used source checkout workspace instead of configured terminal cwd:\n%s", system)
	}
}

func TestTelegramProductionProviderPayloadUsesRuntimeSeededAgentTemplates(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace-gormes")
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes-home"))
	t.Setenv("HERMES_HOME", "")

	cfg := config.Config{
		Hermes: config.HermesCfg{
			Model:    "gpt-5.5",
			Provider: "openai-codex",
		},
		Telegram: config.TelegramCfg{AllowedChatID: 42},
		Terminal: config.TerminalCfg{CWD: workspace},
	}
	if _, err := gatewaymodule.EnsureAgentTemplates(cfg, discardLogger()); err != nil {
		t.Fatalf("ensureGatewayAgentTemplates: %v", err)
	}

	provider := llm.NewMockClient()
	provider.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "seeded context ok"},
		{Kind: llm.EventDone, FinishReason: "stop"},
	}, "sess-provider")
	k := kernel.New(kernel.Config{
		Model:     "gpt-5.5",
		Endpoint:  "http://mock-provider",
		Admission: kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, provider, store.NewNoop(), telemetry.New(), slog.Default())

	ch := newTelegramTestChannel()
	smap := session.NewMemMap()
	mgrCfg := apptelegram.TelegramManagerConfig(cfg, smap, func(cfg config.Config, allowedChats map[string]string, allowDiscovery map[string]bool, allowedWhitelists map[string]gateway.WhitelistConfig, smap session.Map) gateway.ManagerConfig {
		return gatewayManagerConfig(cfg, allowedChats, allowDiscovery, allowedWhitelists, smap, nil, nil, nil, gateway.RestartConfig{})
	})
	mgrCfg.LiveTurnNow = func() time.Time { return time.Date(2026, 5, 5, 7, 0, 0, 0, time.UTC) }

	m := gateway.NewManager(mgrCfg, k, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() { _ = k.Run(ctx) }()
	go func() { _ = m.Run(ctx) }()

	ch.push(gateway.InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "user-juan",
		MsgID:    "tg-msg-seeded-context",
		Kind:     gateway.EventSubmit,
		Text:     "Do you have a SOUL.md?",
	})

	waitForTelegramProviderRequest(t, time.Second, func() bool {
		return len(provider.Requests()) == 1
	})
	req := provider.Requests()[0]
	if len(req.Messages) == 0 || req.Messages[0].Role != "system" {
		t.Fatalf("provider request messages = %#v, want leading system context", req.Messages)
	}
	system := req.Messages[0].Content
	for _, want := range []string{
		"Active workspace: `" + workspace + "`",
		llm.DefaultSoulMD,
		"## AGENTS.md",
		"## IDENTITY.md",
		"## TOOLS.md",
		"# Durable User Context",
		"# User",
		"# Memory",
	} {
		if !strings.Contains(system, want) {
			t.Fatalf("provider system prompt missing seeded template %q in:\n%s", want, system)
		}
	}
}

func writeTelegramFixtureFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

type telegramTestChannel struct {
	in chan gateway.InboundEvent
}

func newTelegramTestChannel() *telegramTestChannel {
	return &telegramTestChannel{in: make(chan gateway.InboundEvent, 1)}
}

func (c *telegramTestChannel) Name() string { return "telegram" }

func (c *telegramTestChannel) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-c.in:
			select {
			case inbox <- ev:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func (c *telegramTestChannel) Send(ctx context.Context, chatID, text string) (string, error) {
	return "telegram-test-msg", nil
}

func (c *telegramTestChannel) push(ev gateway.InboundEvent) { c.in <- ev }

func waitForTelegramProviderRequest(t *testing.T, timeout time.Duration, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
