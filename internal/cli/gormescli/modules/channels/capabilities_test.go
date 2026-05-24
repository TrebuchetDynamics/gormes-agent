package channels

import (
	"bytes"
	"encoding/json"
	"testing"

	channelcaps "github.com/TrebuchetDynamics/gormes-agent/internal/channels"
	"github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestCapabilitiesCommandUsesInjectedConfigAndBuildProvenance(t *testing.T) {
	cmd := NewCommandWithSeams(Seams{
		LoadConfig: func() (config.Config, error) {
			return config.Config{Telegram: config.TelegramCfg{BotToken: "secret-token"}}, nil
		},
		ConfiguredDetails: func(config.Config) map[string]string {
			return map[string]string{"telegram": "allowed_chat_id=42"}
		},
	}, Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{Version: "test-version", GitCommit: "test-sha"}
		},
	})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"capabilities", "--channel", "telegram", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("channels capabilities --json: %v\nstdout=%s", err, stdout.String())
	}
	var got struct {
		Build    gormescli.BuildProvenance      `json:"build"`
		Channels []channelcaps.CapabilityReport `json:"channels"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout must be JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Build.Version != "test-version" || got.Build.GitCommit != "test-sha" {
		t.Fatalf("build provenance = %+v, want injected test values", got.Build)
	}
	if len(got.Channels) != 1 || got.Channels[0].Channel != "telegram" || !got.Channels[0].Configured || got.Channels[0].ConfigDetail != "allowed_chat_id=42" {
		t.Fatalf("channels = %+v, want configured telegram detail", got.Channels)
	}
}
