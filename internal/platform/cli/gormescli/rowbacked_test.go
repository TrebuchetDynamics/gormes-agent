package gormescli

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRowBackedCommandEmitsInjectedBuildProvenanceAndExitCode(t *testing.T) {
	cmd := NewRowBackedCommand(RowBackedCommandSpec{
		Use:   "fixture",
		Short: "Fixture row-backed command",
		Row:   "Fixture row",
	}, RowBackedCommandOptions{
		BuildProvenance: func() BuildProvenance {
			return BuildProvenance{Version: "test-version", GitCommit: "test-sha"}
		},
	})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("row-backed command must exit non-zero\nstdout=%s", stdout.String())
	}
	if exit, ok := err.(interface{ ExitCode() int }); !ok || exit.ExitCode() != 2 {
		t.Fatalf("row-backed exit = %#v, want ExitCode() == 2", err)
	}

	var got RowBackedReportJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("row-backed stdout must be JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Build.Version != "test-version" || got.Build.GitCommit != "test-sha" {
		t.Fatalf("build provenance = %+v, want injected test values", got.Build)
	}
	if got.Action != "hermes_command_unavailable" || got.Command != "fixture" || got.Status != RowBackedStatus || got.Row != "Fixture row" {
		t.Fatalf("row-backed report = %+v, want fixture row-backed report", got)
	}
}
