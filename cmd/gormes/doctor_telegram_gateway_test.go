package main

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/doctor"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestDoctorTelegramGatewayConfigWarnsWhenLiveRuntimeStoppedChannel(t *testing.T) {
	cfg := config.TelegramCfg{BotToken: "123456:abcdefghijklmnopqrstuvwxyzABCDEF", FirstRunDiscovery: true, AllowedUserIDs: []int64{1}}
	runtime := gateway.RuntimeStatus{
		Kind:         "gormes-gateway",
		GatewayState: gateway.GatewayStateRunning,
		Platforms: map[string]gateway.PlatformRuntimeStatus{
			"telegram": {State: gateway.PlatformStateStopped},
		},
	}

	got, ok := doctorTelegramGatewayRuntimeWarning(cfg, runtime)
	if !ok {
		t.Fatal("doctor telegram runtime warning missing for stopped live channel")
	}
	if got.Status != doctor.StatusWarn {
		t.Fatalf("status = %s, want warn", got.Status)
	}
	for _, want := range []string{"lifecycle=stopped", "gormes gateway restart"} {
		if !strings.Contains(got.Summary, want) {
			t.Fatalf("summary missing %q: %s", want, got.Summary)
		}
	}
}

func TestDoctorTelegramGatewayConfigWarnsWhenLiveRuntimeDidNotRegisterChannel(t *testing.T) {
	cfg := config.TelegramCfg{BotToken: "123456:abcdefghijklmnopqrstuvwxyzABCDEF", FirstRunDiscovery: true, AllowedUserIDs: []int64{1}}
	runtime := gateway.RuntimeStatus{
		Kind:         "gormes-gateway",
		GatewayState: gateway.GatewayStateRunning,
		Platforms: map[string]gateway.PlatformRuntimeStatus{
			"navivox": {State: gateway.PlatformStateRunning},
		},
	}

	got, ok := doctorTelegramGatewayRuntimeWarning(cfg, runtime)
	if !ok {
		t.Fatal("doctor telegram runtime warning missing for unregistered live channel")
	}
	if got.Status != doctor.StatusWarn {
		t.Fatalf("status = %s, want warn", got.Status)
	}
	for _, want := range []string{"not registered", "restart"} {
		if !strings.Contains(got.Summary, want) {
			t.Fatalf("summary missing %q: %s", want, got.Summary)
		}
	}
}

func TestDoctorTelegramGatewayConfigDoesNotWarnWhenRuntimeIsUnknown(t *testing.T) {
	cfg := config.TelegramCfg{BotToken: "123456:abcdefghijklmnopqrstuvwxyzABCDEF", FirstRunDiscovery: true}

	if got, ok := doctorTelegramGatewayRuntimeWarning(cfg, gateway.RuntimeStatus{}); ok {
		t.Fatalf("unexpected warning for missing runtime status: %+v", got)
	}
}
