package gateway

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	runtimegateway "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestGatewayNavivoxRegistrationLogsDeprecationWarning(t *testing.T) {
	cfg := config.Config{Navivox: config.NavivoxCfg{Enabled: true}}
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	mgr := runtimegateway.NewManagerWithSubmitter(runtimegateway.ManagerConfig{
		AllowedChats:   map[string]string{},
		AllowDiscovery: map[string]bool{},
	}, newGatewaySlackTestKernel(), logger)

	factories := ChannelFactories{
		Navivox: func(config.Config, *slog.Logger) (runtimegateway.Channel, error) {
			return newGatewaySlackTestChannel("navivox"), nil
		},
	}

	registered, err := RegisterConfiguredGatewayChannels(mgr, cfg, map[string]string{}, map[string]bool{}, factories, &recordingGatewayRuntimeStatus{}, logger)
	if err != nil {
		t.Fatalf("RegisterConfiguredGatewayChannels: %v", err)
	}
	if registered != 1 {
		t.Fatalf("registered = %d, want 1", registered)
	}
	if !strings.Contains(logs.String(), "navivox channel is deprecated and will be removed") {
		t.Fatalf("gateway log missing navivox deprecation warning:\n%s", logs.String())
	}
}
