package providers

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func TestInsightsCommandUsesInjectedRowBackedBuildProvenance(t *testing.T) {
	cmd := NewInsightsCommand(Options{
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

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("insights must remain row-backed and exit non-zero\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
	if exit, ok := err.(interface{ ExitCode() int }); !ok || exit.ExitCode() != 2 {
		t.Fatalf("insights exit = %#v, want ExitCode() == 2", err)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Action  string `json:"action"`
		Command string `json:"command"`
		Status  string `json:"status"`
		Row     string `json:"row"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("insights --json must emit parseable JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Build.Version != "test-version" || got.Build.GitCommit != "test-sha" {
		t.Fatalf("build provenance = %+v, want injected test values", got.Build)
	}
	if got.Action != "hermes_command_unavailable" || got.Command != "insights" || got.Status != gormescli.RowBackedStatus || got.Row != "Self-monitoring telemetry" {
		t.Fatalf("insights report = %+v, want row-backed insights telemetry report", got)
	}
	if !strings.Contains(got.Error, "row-backed") {
		t.Fatalf("error = %q, want row-backed evidence", got.Error)
	}
}
