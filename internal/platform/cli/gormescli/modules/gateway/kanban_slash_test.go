package gateway

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func TestGatewayKanbanSlashRunnerUsesCLICommandRunner(t *testing.T) {
	setupGatewayKanbanSlashTestEnv(t)

	runner := NewKanbanSlashRunner(gormescli.KanbanCommandOptions{
		BuildProvenance: func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{Version: testGatewayVersion, GitCommit: "test-git"}
		},
		ExitCodeError: gormescli.NewExitCodeError,
	})
	if runner == nil {
		t.Fatal("NewKanbanSlashRunner returned nil; gateway /kanban would not share the CLI command runner")
	}

	out, err := runner(context.Background(), "/kanban init")
	if err != nil {
		t.Fatalf("KanbanSlashRunner(/kanban init): %v\nout=%s", err, out)
	}
	if !strings.Contains(out, "kanban initialized at") {
		t.Fatalf("KanbanSlashRunner output = %q, want init output", out)
	}
}

func setupGatewayKanbanSlashTestEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("GORMES_KANBAN_DB", "")
	t.Setenv("GORMES_KANBAN_HOME", "")
	t.Setenv("GORMES_KANBAN_TASK", "")
	t.Setenv("HERMES_KANBAN_BOARD", "")
	t.Setenv("HERMES_KANBAN_DB", "")
}
