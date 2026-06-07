package capabilities

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	channelcaps "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func TestChannelsCommandDefaultRendersCapabilities(t *testing.T) {
	cmd := NewCommandWithSeams(Seams{
		LoadConfig: func() (config.Config, error) { return config.Config{}, nil },
		ConfiguredDetails: func(config.Config) map[string]string {
			return map[string]string{"whatsapp": "mode=bot"}
		},
	}, Options{})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--channel", "whatsapp"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("channels: %v\nstdout=%s", err, stdout.String())
	}
	for _, want := range []string{"WhatsApp (whatsapp)", "Status: configured (mode=bot)", "Support:"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("channels printed help instead of capability summary:\n%s", stdout.String())
	}
}

func TestChannelsCommandAcceptsPositionalChannel(t *testing.T) {
	cmd := NewCommandWithSeams(Seams{
		LoadConfig: func() (config.Config, error) { return config.Config{}, nil },
		ConfiguredDetails: func(config.Config) map[string]string {
			return map[string]string{"telegram": "allowed_chat_id=42"}
		},
	}, Options{})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"telegram"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("channels telegram: %v\nstdout=%s", err, stdout.String())
	}
	for _, want := range []string{"Telegram (telegram)", "Status: configured (allowed_chat_id=42)"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	if strings.Contains(stdout.String(), "WhatsApp (whatsapp)") {
		t.Fatalf("positional channel should filter to telegram only:\n%s", stdout.String())
	}
}

func TestChannelsSetupSubcommandPrintsSetupGuidance(t *testing.T) {
	cmd := NewCommandWithSeams(Seams{}, Options{})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"setup", "telegram"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("channels setup telegram: %v\nstdout=%s", err, stdout.String())
	}
	for _, want := range []string{
		"Channel setup commands for Telegram (telegram):",
		"gormes setup telegram",
		"gormes setup --quick --target telegram",
		"gormes setup gateway --plan",
		"gormes channels telegram",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestChannelsSetupSubcommandSupportsEveryManifestChannel(t *testing.T) {
	reports, err := channelcaps.BuildCapabilityReports(channelcaps.CapabilityOptions{})
	if err != nil {
		t.Fatalf("BuildCapabilityReports: %v", err)
	}
	for _, report := range reports {
		t.Run(report.Channel, func(t *testing.T) {
			cmd := NewCommandWithSeams(Seams{}, Options{})
			var stdout bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			cmd.SetArgs([]string{"setup", report.Channel})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("channels setup %s: %v\nstdout=%s", report.Channel, err, stdout.String())
			}
			for _, want := range []string{"Channel setup commands for", "gormes setup gateway --plan", "gormes channels " + report.Channel} {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
				}
			}
			if !IsQuickSetupChannel(report.Channel) && strings.Contains(stdout.String(), "gormes setup --quick --target "+report.Channel) {
				t.Fatalf("row-backed channel %s should not advertise unsupported quick target:\n%s", report.Channel, stdout.String())
			}
		})
	}
}

func TestChannelsSetupSubcommandListsAllSetupEntrypoints(t *testing.T) {
	cmd := NewCommandWithSeams(Seams{}, Options{})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"setup"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("channels setup: %v\nstdout=%s", err, stdout.String())
	}
	for _, want := range []string{
		"Channel setup commands:",
		"telegram: gormes setup telegram",
		"discord: gormes setup discord",
		"slack: gormes setup slack",
		"whatsapp: gormes setup whatsapp",
		"navivox: gormes setup navivox",
		"Other channels: run `gormes channels setup <channel>` for row-backed guidance.",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestChannelsCommandTypoStillSuggestsCapabilities(t *testing.T) {
	cmd := NewCommandWithSeams(Seams{}, Options{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"capabilites"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("channels capabilites returned nil error, want typo suggestion")
	}
	combined := strings.ToLower(err.Error())
	for _, want := range []string{"did you mean", "capabilities"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}

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
