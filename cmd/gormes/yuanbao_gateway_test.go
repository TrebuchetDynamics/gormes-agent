package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/yuanbao"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

type yuanbaoTestKernel struct {
	mu      sync.Mutex
	events  []kernel.PlatformEvent
	renders chan kernel.RenderFrame
}

func newYuanbaoTestKernel() *yuanbaoTestKernel {
	return &yuanbaoTestKernel{renders: make(chan kernel.RenderFrame)}
}

func (k *yuanbaoTestKernel) Submit(ev kernel.PlatformEvent) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.events = append(k.events, ev)
	return nil
}
func (k *yuanbaoTestKernel) ResetSession() error               { return nil }
func (k *yuanbaoTestKernel) Render() <-chan kernel.RenderFrame { return k.renders }

func (k *yuanbaoTestKernel) snapshot() []kernel.PlatformEvent {
	k.mu.Lock()
	defer k.mu.Unlock()
	return append([]kernel.PlatformEvent(nil), k.events...)
}

type fakeYuanbaoGatewayClient struct {
	mu       sync.Mutex
	inbound  []yuanbao.InboundPush
	sent     []yuanbao.SentMessage
	started  chan struct{}
	startOne sync.Once
}

func newFakeYuanbaoGatewayClient() *fakeYuanbaoGatewayClient {
	return &fakeYuanbaoGatewayClient{started: make(chan struct{})}
}

func (c *fakeYuanbaoGatewayClient) queue(push yuanbao.InboundPush) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inbound = append(c.inbound, push)
}

func (c *fakeYuanbaoGatewayClient) Connect(context.Context) error { return nil }

func (c *fakeYuanbaoGatewayClient) Run(ctx context.Context, deliver func(context.Context, yuanbao.InboundPush)) error {
	c.startOne.Do(func() { close(c.started) })
	for {
		c.mu.Lock()
		if len(c.inbound) == 0 {
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Millisecond):
				continue
			}
		}
		next := c.inbound[0]
		c.inbound = c.inbound[1:]
		c.mu.Unlock()
		deliver(ctx, next)
	}
}

func (c *fakeYuanbaoGatewayClient) Send(_ context.Context, conversationID, text string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, yuanbao.SentMessage{ConversationID: conversationID, Text: text})
	return "yb-msg-1", nil
}

func (c *fakeYuanbaoGatewayClient) sentSnapshot() []yuanbao.SentMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]yuanbao.SentMessage(nil), c.sent...)
}

func TestYuanbaoGateway_DisabledByDefaultIsNotRegistered(t *testing.T) {
	cfg := config.Config{}
	mgr := gateway.NewManagerWithSubmitter(gateway.ManagerConfig{
		AllowedChats:   map[string]string{},
		AllowDiscovery: map[string]bool{},
	}, newYuanbaoTestKernel(), slog.Default())
	factoryCalled := false
	factories := gatewayChannelFactories{
		Yuanbao: func(config.Config, *slog.Logger) (gateway.Channel, error) {
			factoryCalled = true
			return nil, errors.New("should not run")
		},
	}

	registered, err := registerConfiguredGatewayChannels(mgr, cfg, map[string]string{}, map[string]bool{}, factories, nil, slog.Default())
	if err != nil {
		t.Fatalf("registerConfiguredGatewayChannels: %v", err)
	}
	if registered != 0 || mgr.ChannelCount() != 0 {
		t.Fatalf("registered/channelCount = %d/%d, want 0/0 disabled-by-default", registered, mgr.ChannelCount())
	}
	if factoryCalled {
		t.Fatal("Yuanbao factory was called despite disabled-by-default config")
	}
}

func TestYuanbaoGateway_MissingCredentialsDegradeWithoutBlockingTelegram(t *testing.T) {
	cfg := config.Config{
		Telegram: config.TelegramCfg{BotToken: "tg-token", AllowedChatID: 1},
		Yuanbao: config.YuanbaoCfg{
			Enabled:               true,
			AllowedConversationID: "conv-1",
		},
	}
	status := &recordingGatewayRuntimeStatus{}
	mgr := gateway.NewManagerWithSubmitter(gateway.ManagerConfig{
		AllowedChats:   map[string]string{},
		AllowDiscovery: map[string]bool{},
		RuntimeStatus:  status,
	}, newYuanbaoTestKernel(), slog.Default())
	yuanbaoFactoryCalled := false
	factories := gatewayChannelFactories{
		Telegram: func(config.Config, *slog.Logger) (gateway.Channel, error) {
			return newGatewaySlackTestChannel("telegram"), nil
		},
		Yuanbao: func(config.Config, *slog.Logger) (gateway.Channel, error) {
			yuanbaoFactoryCalled = true
			return newGatewaySlackTestChannel("yuanbao"), nil
		},
	}

	registered, err := registerConfiguredGatewayChannels(mgr, cfg, map[string]string{}, map[string]bool{}, factories, status, slog.Default())
	if err != nil {
		t.Fatalf("registerConfiguredGatewayChannels: %v", err)
	}
	if registered != 1 || mgr.ChannelCount() != 1 {
		t.Fatalf("registered/channelCount = %d/%d, want telegram only", registered, mgr.ChannelCount())
	}
	if yuanbaoFactoryCalled {
		t.Fatal("Yuanbao factory was called despite missing credentials")
	}
	errText := status.platformError("yuanbao")
	for _, want := range []string{"missing", "login_token", "hy_source", "agent_id"} {
		if !strings.Contains(errText, want) {
			t.Fatalf("Yuanbao status error %q missing %q", errText, want)
		}
	}
	if !status.hasPlatformState("yuanbao", gateway.PlatformStateFailed) {
		t.Fatal("Yuanbao runtime status did not record failed state for missing credentials")
	}
}

func TestYuanbaoGateway_RegistersWhenEnabledWithCredentials(t *testing.T) {
	cfg := config.Config{
		Yuanbao: config.YuanbaoCfg{
			Enabled:               true,
			LoginToken:            "fake-login-token",
			HySource:              "fake-hy-source",
			AgentID:               "fake-agent",
			AllowedConversationID: "conv-99",
			CoalesceMs:            500,
			FirstRunDiscovery:     true,
		},
	}
	allowedChats := map[string]string{}
	allowDiscovery := map[string]bool{}
	status := &recordingGatewayRuntimeStatus{}
	mgr := gateway.NewManagerWithSubmitter(gateway.ManagerConfig{
		AllowedChats:   allowedChats,
		AllowDiscovery: allowDiscovery,
		RuntimeStatus:  status,
	}, newYuanbaoTestKernel(), slog.Default())

	fakeChan := newGatewaySlackTestChannel("yuanbao")
	factories := gatewayChannelFactories{
		Yuanbao: func(got config.Config, _ *slog.Logger) (gateway.Channel, error) {
			if got.Yuanbao.LoginToken != "fake-login-token" || got.Yuanbao.HySource != "fake-hy-source" || got.Yuanbao.AgentID != "fake-agent" {
				t.Fatalf("factory saw Yuanbao cfg %#v", got.Yuanbao)
			}
			return fakeChan, nil
		},
	}

	registered, err := registerConfiguredGatewayChannels(mgr, cfg, allowedChats, allowDiscovery, factories, status, slog.Default())
	if err != nil {
		t.Fatalf("registerConfiguredGatewayChannels: %v", err)
	}
	if registered != 1 || mgr.ChannelCount() != 1 {
		t.Fatalf("registered/channelCount = %d/%d, want 1/1", registered, mgr.ChannelCount())
	}
	if allowedChats["yuanbao"] != "conv-99" {
		t.Fatalf("allowedChats[yuanbao] = %q, want conv-99", allowedChats["yuanbao"])
	}
	if !allowDiscovery["yuanbao"] {
		t.Fatal("allowDiscovery[yuanbao] = false, want true")
	}
}

func TestYuanbaoGateway_FakeRuntimeDeliversInboundAndOutboundThroughManager(t *testing.T) {
	client := newFakeYuanbaoGatewayClient()
	client.queue(yuanbao.InboundPush{
		ConversationID: "conv-99",
		MessageID:      "msg-1",
		AuthorRole:     "user",
		Text:           "hi gormes",
	})

	channel := yuanbao.NewChannel(yuanbao.Config{AllowedConversationID: "conv-99"}, client, slog.Default())
	mgr := gateway.NewManagerWithSubmitter(gateway.ManagerConfig{
		AllowedChats:   map[string]string{"yuanbao": "conv-99"},
		AllowDiscovery: map[string]bool{},
	}, newYuanbaoTestKernel(), slog.Default())
	if err := mgr.Register(channel); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- mgr.Run(ctx) }()

	select {
	case <-client.started:
	case <-time.After(time.Second):
		t.Fatal("Yuanbao client did not start under shared manager")
	}

	if _, err := channel.Send(context.Background(), "conv-99", "outbound payload"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(client.sentSnapshot()) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	sent := client.sentSnapshot()
	if len(sent) != 1 || sent[0].ConversationID != "conv-99" || sent[0].Text != "outbound payload" {
		t.Fatalf("fake client did not record outbound: %#v", sent)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("manager Run = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("manager Run did not return after cancellation")
	}
}
