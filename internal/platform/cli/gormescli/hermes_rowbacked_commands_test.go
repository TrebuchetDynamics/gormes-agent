package gormescli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestHermesRowBackedMiscCommandConstructorsEmitStructuredUnavailableJSON(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		wantCommand string
		wantRow     string
		destructive bool
	}{
		{
			name:        "dump",
			args:        []string{"dump", "--json"},
			wantCommand: "gormes dump",
			wantRow:     HermesDiagnosticsRow,
		},
		{
			name:        "debug_share",
			args:        []string{"debug", "share", "--json"},
			wantCommand: "gormes debug share",
			wantRow:     HermesDiagnosticsRow,
		},
		{
			name:        "debug_delete",
			args:        []string{"debug", "delete", "--json"},
			wantCommand: "gormes debug delete",
			wantRow:     HermesDiagnosticsRow,
			destructive: true,
		},
		{
			name:        "backup",
			args:        []string{"backup", "--json"},
			wantCommand: "gormes backup",
			wantRow:     "Backup/update opt-in and exclusion policy",
		},
		{
			name:        "import",
			args:        []string{"import", "--json"},
			wantCommand: "gormes import",
			wantRow:     HermesConfigRow,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newHermesRowBackedMiscRootForTest()
			var stdout bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("%s must exit non-zero\nstdout=%s", strings.Join(tc.args, " "), stdout.String())
			}
			if exit, ok := err.(interface{ ExitCode() int }); !ok || exit.ExitCode() != 2 {
				t.Fatalf("%s exit = %#v, want ExitCode() == 2", strings.Join(tc.args, " "), err)
			}

			var got RowBackedReportJSON
			if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
				t.Fatalf("%s stdout must be JSON: %v\nstdout=%s", strings.Join(tc.args, " "), err, stdout.String())
			}
			if got.Build.Version != "test-version" || got.Build.GitCommit != "test-sha" {
				t.Fatalf("build provenance = %+v, want injected test values", got.Build)
			}
			if got.Command != tc.wantCommand || got.Row != tc.wantRow || got.Status != RowBackedStatus || got.Action != "hermes_command_unavailable" {
				t.Fatalf("report = %+v, want command=%q row=%q status=%q", got, tc.wantCommand, tc.wantRow, RowBackedStatus)
			}
			if got.Destructive != tc.destructive {
				t.Fatalf("destructive = %v, want %v", got.Destructive, tc.destructive)
			}
		})
	}
}

func newHermesRowBackedMiscRootForTest() *cobra.Command {
	opts := RowBackedCommandOptions{BuildProvenance: func() BuildProvenance {
		return BuildProvenance{Version: "test-version", GitCommit: "test-sha"}
	}}
	root := &cobra.Command{Use: "gormes", SilenceUsage: true}
	root.AddCommand(
		NewDumpCommand(opts),
		NewDebugCommand(opts),
		NewBackupCommand(opts),
		NewImportCommand(opts),
	)
	return root
}
