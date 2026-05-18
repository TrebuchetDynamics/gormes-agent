package sessions

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewCheckpointsCommandOwnsTreeFlagsAndBareStatus(t *testing.T) {
	var called string
	cmd := NewCheckpointsCommandWithSeams(CheckpointsCommandSeams{
		RunStatus: func(cmd *cobra.Command, args []string) error {
			called = "status"
			return nil
		},
	})
	for _, path := range []string{"status", "list", "prune", "clear", "clear-legacy"} {
		if child, _, err := cmd.Find([]string{path}); err != nil || child == nil || child.Name() != path {
			t.Fatalf("missing checkpoints child %q: child=%v err=%v", path, child, err)
		}
	}
	assertFlag(t, cmd, []string{"status"}, "limit")
	assertFlag(t, cmd, []string{"list"}, "json")
	assertFlag(t, cmd, []string{"prune"}, "retention-days")
	assertFlag(t, cmd, []string{"prune"}, "dry-run")
	assertFlag(t, cmd, []string{"clear"}, "force")
	assertFlag(t, cmd, []string{"clear-legacy"}, "json")

	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute bare checkpoints: %v", err)
	}
	if called != "status" {
		t.Fatalf("seam called = %q, want status", called)
	}
}

func TestNewCheckpointsCommandUnknownSubcommandSuggestion(t *testing.T) {
	cmd := NewCheckpointsCommandWithSeams(CheckpointsCommandSeams{})
	cmd.SetArgs([]string{"statsu"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("unknown checkpoints command error = nil")
	}
	if !strings.Contains(err.Error(), `unknown command "statsu"`) || !strings.Contains(err.Error(), "status") {
		t.Fatalf("error = %q, want status suggestion", err)
	}
}
