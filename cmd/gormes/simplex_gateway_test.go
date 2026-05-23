package main

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestSimpleXGatewayRegistrationDisabledByDefault(t *testing.T) {
	cfg := config.Config{}
	mgr := gateway.NewManagerWithSubmitter(gateway.ManagerConfig{
		AllowedChats:   map[string]string{},
		AllowDiscovery: map[string]bool{},
	}, newGatewaySlackTestKernel(), slog.Default())
	factoryCalled := false
	factories := gatewayChannelFactories{
		SimpleX: func(config.Config, *slog.Logger) (gateway.Channel, error) {
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
		t.Fatal("SimpleX factory called despite missing SIMPLEX_WS_URL")
	}
}

func TestSimpleXGatewayRegistrationUsesEnvBackedPluginConfig(t *testing.T) {
	t.Setenv("SIMPLEX_WS_URL", "ws://127.0.0.1:5225")
	t.Setenv("SIMPLEX_HOME_CHANNEL", "group:ops")
	t.Setenv("SIMPLEX_ALLOWED_USERS", "contact-42,member-7")

	allowedChats, allowDiscovery, _ := gatewayPolicyMaps(config.Config{})
	if allowedChats["simplex"] != "group:ops" {
		t.Fatalf("allowedChats[simplex] = %q, want group:ops", allowedChats["simplex"])
	}
	if allowDiscovery["simplex"] {
		t.Fatal("allowDiscovery[simplex] = true, want false for opaque SimpleX IDs")
	}
	allowedUsers := gatewayAllowedUsers(config.Config{})
	if !allowedUsers["simplex"]["contact-42"] || !allowedUsers["simplex"]["member-7"] || len(allowedUsers["simplex"]) != 2 {
		t.Fatalf("allowedUsers[simplex] = %+v", allowedUsers["simplex"])
	}

	cfg := config.Config{}
	mgr := gateway.NewManagerWithSubmitter(gateway.ManagerConfig{
		AllowedChats:   allowedChats,
		AllowDiscovery: allowDiscovery,
		AllowedUsers:   allowedUsers,
	}, newGatewaySlackTestKernel(), slog.Default())
	fakeSimpleX := newGatewaySlackTestChannel("simplex")
	factories := gatewayChannelFactories{
		SimpleX: func(config.Config, *slog.Logger) (gateway.Channel, error) {
			return fakeSimpleX, nil
		},
	}

	registered, err := registerConfiguredGatewayChannels(mgr, cfg, allowedChats, allowDiscovery, factories, nil, slog.Default())
	if err != nil {
		t.Fatalf("registerConfiguredGatewayChannels: %v", err)
	}
	if registered != 1 || mgr.ChannelCount() != 1 {
		t.Fatalf("registered/channelCount = %d/%d, want SimpleX", registered, mgr.ChannelCount())
	}
}

func TestSimpleXGatewayStartupAllowlistEnv(t *testing.T) {
	if gatewayStartupAllowlistConfigured(config.Config{}, func(string) string { return "" }) {
		t.Fatal("empty env startup allowlist = true, want false")
	}
	if !gatewayStartupAllowlistConfigured(config.Config{}, func(key string) string {
		switch key {
		case "SIMPLEX_WS_URL":
			return "ws://127.0.0.1:5225"
		case "SIMPLEX_ALLOWED_USERS":
			return "contact-42"
		default:
			return ""
		}
	}) {
		t.Fatal("SIMPLEX_ALLOWED_USERS did not satisfy startup allowlist evidence")
	}
	if !gatewayStartupAllowAllConfigured(func(key string) string {
		if key == "SIMPLEX_ALLOW_ALL_USERS" {
			return "true"
		}
		return ""
	}) {
		t.Fatal("SIMPLEX_ALLOW_ALL_USERS did not satisfy startup allow-all evidence")
	}
}
