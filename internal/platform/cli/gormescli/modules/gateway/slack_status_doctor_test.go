package gateway

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	runtimegateway "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func TestGatewayStatusCommand_RendersSlackConfigAndRuntimeStates(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		setupGatewayStatusTestEnv(t)

		stdout, stderr, err := executeGatewayStatusCommand(t)
		if err != nil {
			t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
		}
		if !strings.Contains(stdout, "gateway/slack: disabled") {
			t.Fatalf("stdout missing Slack disabled state\n%s", stdout)
		}
	})

	t.Run("missing_token", func(t *testing.T) {
		setupGatewayStatusTestEnv(t)
		writeGatewayStatusConfig(t, []byte(`
[slack]
enabled = true
allowed_channel_id = "C123"
`))

		stdout, stderr, err := executeGatewayStatusCommand(t)
		if err != nil {
			t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
		}
		if !strings.Contains(stdout, "- slack: lifecycle=unknown") ||
			!strings.Contains(stdout, "target=missing_tokens=bot_token,app_token") {
			t.Fatalf("stdout missing Slack missing-token row\n%s", stdout)
		}
	})

	t.Run("startup_failed_and_running", func(t *testing.T) {
		setupGatewayStatusTestEnv(t)
		writeGatewayStatusConfig(t, []byte(`
[slack]
enabled = true
bot_token = "xoxb-test"
app_token = "xapp-test"
allowed_channel_id = "C123"
`))
		runtimeStatus := runtimegateway.NewRuntimeStatusStore(config.GatewayRuntimeStatusPath())
		if err := runtimeStatus.UpdateRuntimeStatus(context.Background(), runtimegateway.RuntimeStatusUpdate{
			Platform:      "slack",
			PlatformState: runtimegateway.PlatformStateFailed,
			ErrorMessage:  "slack: socket mode startup denied",
		}); err != nil {
			t.Fatalf("write Slack failed runtime: %v", err)
		}

		stdout, stderr, err := executeGatewayStatusCommand(t)
		if err != nil {
			t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
		}
		if !strings.Contains(stdout, "- slack: lifecycle=failed error=\"slack: socket mode startup denied\"") {
			t.Fatalf("stdout missing Slack failed row\n%s", stdout)
		}

		if err := runtimeStatus.UpdateRuntimeStatus(context.Background(), runtimegateway.RuntimeStatusUpdate{
			Platform:      "slack",
			PlatformState: runtimegateway.PlatformStateRunning,
		}); err != nil {
			t.Fatalf("write Slack running runtime: %v", err)
		}
		stdout, stderr, err = executeGatewayStatusCommand(t)
		if err != nil {
			t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
		}
		if !strings.Contains(stdout, "- slack: lifecycle=running") ||
			!strings.Contains(stdout, "target=allowed_channel_id=C123") {
			t.Fatalf("stdout missing Slack running row\n%s", stdout)
		}
	})
}

func TestDoctorSlackGatewayConfigReportsDisabledMissingFailedAndRunning(t *testing.T) {
	cases := []struct {
		name    string
		cfg     config.Config
		runtime runtimegateway.RuntimeStatus
		want    []string
	}{
		{
			name: "disabled",
			want: []string{"[WARN] Gateway Slack: disabled"},
		},
		{
			name: "missing_token",
			cfg: config.Config{Slack: config.SlackCfg{
				Enabled:          true,
				AllowedChannelID: "C123",
			}},
			want: []string{"[WARN] Gateway Slack: missing_tokens=bot_token,app_token", "allowed_channel_id=C123"},
		},
		{
			name: "startup_failed",
			cfg: config.Config{Slack: config.SlackCfg{
				Enabled:          true,
				BotToken:         "xoxb-test",
				AppToken:         "xapp-test",
				AllowedChannelID: "C123",
			}},
			runtime: runtimegateway.RuntimeStatus{Platforms: map[string]runtimegateway.PlatformRuntimeStatus{
				"slack": {State: runtimegateway.PlatformStateFailed, ErrorMessage: "slack: socket mode startup denied"},
			}},
			want: []string{"[WARN] Gateway Slack: startup_failed", "slack: socket mode startup denied"},
		},
		{
			name: "running",
			cfg: config.Config{Slack: config.SlackCfg{
				Enabled:          true,
				BotToken:         "xoxb-test",
				AppToken:         "xapp-test",
				AllowedChannelID: "C123",
			}},
			runtime: runtimegateway.RuntimeStatus{Platforms: map[string]runtimegateway.PlatformRuntimeStatus{
				"slack": {State: runtimegateway.PlatformStateRunning},
			}},
			want: []string{"[PASS] Gateway Slack: running", "allowed_channel_id=C123"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := gormescli.DoctorSlackGatewayConfig(tc.cfg, tc.runtime).Format()
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Fatalf("doctor Slack output missing %q\n%s", want, got)
				}
			}
		})
	}
}
