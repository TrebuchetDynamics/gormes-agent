package providers

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli"
)

func TestLogoutCommandUsesInjectedAuthSeamsAndBuildProvenance(t *testing.T) {
	var loggedOut []string
	var reset []string
	cmd := NewLogoutCommandWithSeams(LogoutSeams{
		NormalizeAuthProvider: func(provider string) string {
			return strings.ToLower(strings.TrimSpace(provider))
		},
		ConfiguredProvider: func() (string, error) {
			return "openai-codex", nil
		},
		RunAuthLogout: func(cmd *cobra.Command, provider string) error {
			loggedOut = append(loggedOut, provider)
			return WriteAuthLifecycleJSON(cmd.OutOrStdout(), AuthLifecycleReportJSON{
				Build:    gormescli.BuildProvenance{Version: "auth-version", GitCommit: "auth-sha"},
				Action:   "logged_out",
				Provider: provider,
				Redacted: true,
			})
		},
		ResetProviderIfMatching: func(provider string) error {
			reset = append(reset, provider)
			return nil
		},
	}, Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{Version: "test-version", GitCommit: "test-sha"}
		},
	})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("logout: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if len(loggedOut) != 1 || loggedOut[0] != "openai-codex" {
		t.Fatalf("loggedOut = %v, want [openai-codex]", loggedOut)
	}
	if len(reset) != 1 || reset[0] != "openai-codex" {
		t.Fatalf("reset = %v, want [openai-codex]", reset)
	}

	var got AuthLifecycleReportJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("logout stdout must be JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Build.Version != "auth-version" || got.Build.GitCommit != "auth-sha" {
		t.Fatalf("auth seam build provenance = %+v, want auth seam JSON to remain authoritative", got.Build)
	}
	if got.Action != "logged_out" || got.Provider != "openai-codex" || !got.Redacted {
		t.Fatalf("logout report = %+v, want logged_out openai-codex redacted", got)
	}
}

func TestLogoutCommandMissingDefaultWritesInjectedAbsentReport(t *testing.T) {
	cmd := NewLogoutCommandWithSeams(LogoutSeams{
		NormalizeAuthProvider: func(provider string) string {
			return strings.ToLower(strings.TrimSpace(provider))
		},
		ConfiguredProvider: func() (string, error) {
			return "", nil
		},
		RunAuthLogout: func(*cobra.Command, string) error {
			t.Fatal("RunAuthLogout must not run when no explicit or configured provider exists")
			return nil
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
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("logout missing default: %v\nstdout=%s", err, stdout.String())
	}
	var got AuthLifecycleReportJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("logout absent stdout must be JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Build.Version != "test-version" || got.Build.GitCommit != "test-sha" {
		t.Fatalf("build provenance = %+v, want injected test values", got.Build)
	}
	if got.Action != "absent" || got.Provider != "auto" || !got.Redacted {
		t.Fatalf("logout absent report = %+v, want absent auto redacted", got)
	}
}
