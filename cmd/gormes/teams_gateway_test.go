package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestTeamsGatewayRegistrationDisabledByDefault(t *testing.T) {
	cfg := config.Config{}
	mgr := gateway.NewManagerWithSubmitter(gateway.ManagerConfig{
		AllowedChats:   map[string]string{},
		AllowDiscovery: map[string]bool{},
	}, newGatewaySlackTestKernel(), slog.Default())
	factoryCalled := false
	factories := gatewayChannelFactories{
		Teams: func(config.Config, *slog.Logger) (gateway.Channel, error) {
			factoryCalled = true
			return nil, errors.New("should not run")
		},
	}

	registered, err := registerConfiguredGatewayChannels(mgr, cfg, map[string]string{}, map[string]bool{}, factories, nil, slog.Default())
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
	mgr := gateway.NewManagerWithSubmitter(gateway.ManagerConfig{
		AllowedChats:   map[string]string{},
		AllowDiscovery: map[string]bool{},
		RuntimeStatus:  status,
	}, newGatewaySlackTestKernel(), slog.Default())
	teamsFactoryCalled := false
	factories := gatewayChannelFactories{
		Telegram: func(config.Config, *slog.Logger) (gateway.Channel, error) {
			return newGatewaySlackTestChannel("telegram"), nil
		},
		Teams: func(config.Config, *slog.Logger) (gateway.Channel, error) {
			teamsFactoryCalled = true
			return newGatewaySlackTestChannel("teams"), nil
		},
	}

	registered, err := registerConfiguredGatewayChannels(mgr, cfg, map[string]string{}, map[string]bool{}, factories, status, slog.Default())
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
	if !status.hasPlatformState("teams", gateway.PlatformStateFailed) {
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
	mgr := gateway.NewManagerWithSubmitter(gateway.ManagerConfig{
		AllowedChats:  map[string]string{},
		RuntimeStatus: status,
	}, newGatewaySlackTestKernel(), slog.Default())
	fakeTeams := newGatewaySlackTestChannel("teams")
	factories := gatewayChannelFactories{
		Teams: func(got config.Config, _ *slog.Logger) (gateway.Channel, error) {
			if got.Teams.ClientID != "client-id" || got.Teams.ClientSecret != "secret" || got.Teams.TenantID != "tenant-id" || got.Teams.EffectivePort() != 4444 {
				t.Fatalf("factory saw Teams config %#v", got.Teams)
			}
			return fakeTeams, nil
		},
	}

	registered, err := registerConfiguredGatewayChannels(mgr, cfg, map[string]string{}, map[string]bool{}, factories, status, slog.Default())
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

func TestGatewayStatusCommand_RendersTeamsConfiguredUnavailable(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	writeGatewayStatusConfig(t, []byte(`
[teams]
enabled = true
port = 3978
allowed_users = ["aad-1"]
`))

	stdout, stderr, err := executeGatewayStatusCommand(t)
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if !strings.Contains(stdout, "- teams: lifecycle=unknown") ||
		!strings.Contains(stdout, "target=missing_credentials=client_id,client_secret,tenant_id port=3978 allowed_users=1") {
		t.Fatalf("stdout missing Teams configured-unavailable row\n%s", stdout)
	}
}
