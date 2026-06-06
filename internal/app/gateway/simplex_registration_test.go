package gateway

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	runtimegateway "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestSimpleXGatewayRegistrationDisabledByDefault(t *testing.T) {
	cfg := config.Config{}
	mgr := runtimegateway.NewManagerWithSubmitter(runtimegateway.ManagerConfig{
		AllowedChats:   map[string]string{},
		AllowDiscovery: map[string]bool{},
	}, newGatewaySlackTestKernel(), slog.Default())
	factoryCalled := false
	factories := ChannelFactories{
		SimpleX: func(config.Config, *slog.Logger) (runtimegateway.Channel, error) {
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
		t.Fatal("SimpleX factory called despite missing SIMPLEX_WS_URL")
	}
}

func TestSimpleXGatewayRegistrationUsesEnvBackedPluginConfig(t *testing.T) {
	t.Setenv("SIMPLEX_WS_URL", "ws://127.0.0.1:5225")
	t.Setenv("SIMPLEX_HOME_CHANNEL", "group:ops")
	t.Setenv("SIMPLEX_ALLOWED_USERS", "contact-42,member-7")

	allowedChats, allowDiscovery, _ := GatewayPolicyMaps(config.Config{})
	if allowedChats["simplex"] != "group:ops" {
		t.Fatalf("allowedChats[simplex] = %q, want group:ops", allowedChats["simplex"])
	}
	if allowDiscovery["simplex"] {
		t.Fatal("allowDiscovery[simplex] = true, want false for opaque SimpleX IDs")
	}
	allowedUsers := GatewayAllowedUsers(config.Config{})
	if !allowedUsers["simplex"]["contact-42"] || !allowedUsers["simplex"]["member-7"] || len(allowedUsers["simplex"]) != 2 {
		t.Fatalf("allowedUsers[simplex] = %+v", allowedUsers["simplex"])
	}

	cfg := config.Config{}
	mgr := runtimegateway.NewManagerWithSubmitter(runtimegateway.ManagerConfig{
		AllowedChats:   allowedChats,
		AllowDiscovery: allowDiscovery,
		AllowedUsers:   allowedUsers,
	}, newGatewaySlackTestKernel(), slog.Default())
	fakeSimpleX := newGatewaySlackTestChannel("simplex")
	factories := ChannelFactories{
		SimpleX: func(config.Config, *slog.Logger) (runtimegateway.Channel, error) {
			return fakeSimpleX, nil
		},
	}

	registered, err := RegisterConfiguredGatewayChannels(mgr, cfg, allowedChats, allowDiscovery, factories, nil, slog.Default())
	if err != nil {
		t.Fatalf("registerConfiguredGatewayChannels: %v", err)
	}
	if registered != 1 || mgr.ChannelCount() != 1 {
		t.Fatalf("registered/channelCount = %d/%d, want SimpleX", registered, mgr.ChannelCount())
	}
}
