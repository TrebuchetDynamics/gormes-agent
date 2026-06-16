package doctor

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	runtimegateway "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestDoctorTelegramGatewayConfigWarnsWhenLiveRuntimeStoppedChannel(t *testing.T) {
	cfg := config.TelegramCfg{BotToken: "123456:abcdefghijklmnopqrstuvwxyzABCDEF", FirstRunDiscovery: true, AllowedUserIDs: []int64{1}}
	runtime := runtimegateway.RuntimeStatus{
		Kind:         "gormes-gateway",
		GatewayState: runtimegateway.GatewayStateRunning,
		Platforms: map[string]runtimegateway.PlatformRuntimeStatus{
			"telegram": {State: runtimegateway.PlatformStateStopped},
		},
	}

	got, ok := TelegramGatewayRuntimeWarning(cfg, runtime)
	if !ok {
		t.Fatal("doctor telegram runtime warning missing for stopped live channel")
	}
	if got.Status != StatusWarn {
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
	runtime := runtimegateway.RuntimeStatus{
		Kind:         "gormes-gateway",
		GatewayState: runtimegateway.GatewayStateRunning,
		Platforms: map[string]runtimegateway.PlatformRuntimeStatus{
			"navivox": {State: runtimegateway.PlatformStateRunning},
		},
	}

	got, ok := TelegramGatewayRuntimeWarning(cfg, runtime)
	if !ok {
		t.Fatal("doctor telegram runtime warning missing for unregistered live channel")
	}
	if got.Status != StatusWarn {
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

	if got, ok := TelegramGatewayRuntimeWarning(cfg, runtimegateway.RuntimeStatus{}); ok {
		t.Fatalf("unexpected warning for missing runtime status: %+v", got)
	}
}
