package gateway

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	runtimegateway "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestTeamsGatewayRegistrationDisabledByDefault(t *testing.T) {
	cfg := config.Config{}
	mgr := runtimegateway.NewManagerWithSubmitter(runtimegateway.ManagerConfig{
		AllowedChats:   map[string]string{},
		AllowDiscovery: map[string]bool{},
	}, newGatewaySlackTestKernel(), slog.Default())
	factoryCalled := false
	factories := ChannelFactories{
		Teams: func(config.Config, *slog.Logger) (runtimegateway.Channel, error) {
			factoryCalled = true
			return nil, errors.New("should not run")
		},
	}

	registered, err := RegisterConfiguredGatewayChannels(mgr, cfg, map[string]string{}, map[string]bool{}, factories, nil, slog.Default())
	if err != nil {
		t.Fatalf("registerConfiguredGatewayChannels: %v", err)
	}
	if registered != 0 || mgr.ChannelCount() != 0 {
		t.Fatalf("registered/channelCount = %d/%d, want 0/0", registered, mgr.ChannelCount())
	}
	if factoryCalled {
		t.Fatal("Teams factory called despite disabled-by-default config")
	}
}

func TestTeamsGatewayRegistrationMissingCredentialsDegradesWithoutBlockingTelegram(t *testing.T) {
	cfg := config.Config{
		Telegram: config.TelegramCfg{BotToken: "tg-token", AllowedChatID: 42},
		Teams: config.TeamsCfg{
			Enabled: true,
			Port:    3978,
		},
	}
	status := &recordingGatewayRuntimeStatus{}
	mgr := runtimegateway.NewManagerWithSubmitter(runtimegateway.ManagerConfig{
		AllowedChats:   map[string]string{},
		AllowDiscovery: map[string]bool{},
		RuntimeStatus:  status,
	}, newGatewaySlackTestKernel(), slog.Default())
	teamsFactoryCalled := false
	factories := ChannelFactories{
		Telegram: func(config.Config, *slog.Logger) (runtimegateway.Channel, error) {
			return newGatewaySlackTestChannel("telegram"), nil
		},
		Teams: func(config.Config, *slog.Logger) (runtimegateway.Channel, error) {
			teamsFactoryCalled = true
			return newGatewaySlackTestChannel("teams"), nil
		},
	}

	registered, err := RegisterConfiguredGatewayChannels(mgr, cfg, map[string]string{}, map[string]bool{}, factories, status, slog.Default())
	if err != nil {
		t.Fatalf("registerConfiguredGatewayChannels: %v", err)
	}
	if registered != 1 || mgr.ChannelCount() != 1 {
		t.Fatalf("registered/channelCount = %d/%d, want telegram only", registered, mgr.ChannelCount())
	}
	if teamsFactoryCalled {
		t.Fatal("Teams factory called despite missing credentials")
	}
	errText := status.platformError("teams")
	for _, want := range []string{"missing", "client_id", "client_secret", "tenant_id"} {
		if !strings.Contains(errText, want) {
			t.Fatalf("Teams status error %q missing %q", errText, want)
		}
	}
	if !status.hasPlatformState("teams", runtimegateway.PlatformStateFailed) {
		t.Fatal("Teams runtime status did not record failed state")
	}
}

func TestTeamsGatewayRegistrationRegistersWhenEnabledWithFakeCredentials(t *testing.T) {
	cfg := config.Config{
		Teams: config.TeamsCfg{
			Enabled:      true,
			ClientID:     "client-id",
			ClientSecret: "secret",
			TenantID:     "tenant-id",
			Port:         4444,
			AllowedUsers: []string{"aad-1"},
		},
	}
	status := &recordingGatewayRuntimeStatus{}
	mgr := runtimegateway.NewManagerWithSubmitter(runtimegateway.ManagerConfig{
		AllowedChats:  map[string]string{},
		RuntimeStatus: status,
	}, newGatewaySlackTestKernel(), slog.Default())
	fakeTeams := newGatewaySlackTestChannel("teams")
	factories := ChannelFactories{
		Teams: func(got config.Config, _ *slog.Logger) (runtimegateway.Channel, error) {
			if got.Teams.ClientID != "client-id" || got.Teams.ClientSecret != "secret" || got.Teams.TenantID != "tenant-id" || got.Teams.EffectivePort() != 4444 {
				t.Fatalf("factory saw Teams config %#v", got.Teams)
			}
			return fakeTeams, nil
		},
	}

	registered, err := RegisterConfiguredGatewayChannels(mgr, cfg, map[string]string{}, map[string]bool{}, factories, status, slog.Default())
	if err != nil {
		t.Fatalf("registerConfiguredGatewayChannels: %v", err)
	}
	if registered != 1 || mgr.ChannelCount() != 1 {
		t.Fatalf("registered/channelCount = %d/%d, want 1/1", registered, mgr.ChannelCount())
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- mgr.Run(ctx) }()
	<-fakeTeams.runStarted
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Manager Run after cancel = %v", err)
	}
}
