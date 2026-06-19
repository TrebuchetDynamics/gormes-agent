package gormescli

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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

func TestBackupCommandCreatesRestoreCompatibleZip(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "config.toml"), []byte("[hermes]\nmodel='gpt'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, "checkpoints"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "checkpoints", "skip.txt"), []byte("skip"), 0o600); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(t.TempDir(), "pre-update-test.zip")

	cmd := newHermesRowBackedMiscRootForTest()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"backup", "--source", source, "--output", outPath, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("backup --json: %v\nstdout=%s", err, stdout.String())
	}
	var got backupReportJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("backup stdout must be JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Build.Version != "test-version" || got.Action != "backup_created" || got.Path != outPath || got.Source != source || got.FileCount != 1 || got.SizeBytes <= 0 {
		t.Fatalf("backup report = %+v", got)
	}
	zr, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("open backup zip: %v", err)
	}
	defer zr.Close()
	if len(zr.File) != 1 || zr.File[0].Name != "config.toml" {
		t.Fatalf("zip files = %#v, want only config.toml", zipFileNames(zr.File))
	}
}

func TestBackupCommandDryRunJSONDoesNotCreateZip(t *testing.T) {
	source := t.TempDir()
	outPath := filepath.Join(t.TempDir(), "pre-update-test.zip")
	cmd := newHermesRowBackedMiscRootForTest()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"backup", "--source", source, "--output", outPath, "--dry-run", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("backup --dry-run --json: %v\nstdout=%s", err, stdout.String())
	}
	var got backupReportJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("dry-run stdout must be JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Action != "preview" || !got.DryRun || got.Path != outPath || got.Source != source {
		t.Fatalf("dry-run report = %+v", got)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run created zip or stat failed: %v", err)
	}
}

func zipFileNames(files []*zip.File) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.Name)
	}
	return out
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
