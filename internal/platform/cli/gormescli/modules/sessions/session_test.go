package sessions

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewSessionCommandOwnsSessionTreeAndFlags(t *testing.T) {
	var called string
	cmd := NewSessionCommandWithSeams(SessionCommandSeams{
		RunList: func(cmd *cobra.Command, args []string) error {
			called = "list"
			return nil
		},
	})
	if cmd.Use != "session" || len(cmd.Aliases) != 1 || cmd.Aliases[0] != "sessions" {
		t.Fatalf("session command identity = use %q aliases %#v", cmd.Use, cmd.Aliases)
	}
	for _, path := range []string{"list", "export", "delete", "prune", "browse", "stats", "rename"} {
		if child, _, err := cmd.Find([]string{path}); err != nil || child == nil || child.Name() != path {
			t.Fatalf("missing session child %q: child=%v err=%v", path, child, err)
		}
	}
	assertFlag(t, cmd, []string{"list"}, "source")
	assertFlag(t, cmd, []string{"list"}, "limit")
	assertFlag(t, cmd, []string{"list"}, "json")
	assertFlag(t, cmd, []string{"export"}, "format")
	assertFlag(t, cmd, []string{"delete"}, "yes")
	assertFlag(t, cmd, []string{"prune"}, "older-than")
	assertFlag(t, cmd, []string{"browse"}, "no-curses")

	cmd.SetArgs([]string{"list"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute session list: %v", err)
	}
	if called != "list" {
		t.Fatalf("seam called = %q, want list", called)
	}
}

func TestNewSessionCommandUsesUnavailableSeam(t *testing.T) {
	var rows []string
	cmd := NewSessionCommandWithSeams(SessionCommandSeams{
		UnavailableCommand: func(spec UnavailableCommandSpec) *cobra.Command {
			rows = append(rows, spec.Row)
			return &cobra.Command{Use: spec.Use, Short: spec.Short}
		},
	})
	if _, _, err := cmd.Find([]string{"stats"}); err != nil {
		t.Fatalf("find stats: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("unavailable rows = %#v, want 2", rows)
	}
}

func assertFlag(t *testing.T, root *cobra.Command, path []string, name string) {
	t.Helper()
	cmd, _, err := root.Find(path)
	if err != nil {
		t.Fatalf("find %v: %v", path, err)
	}
	if cmd == nil || cmd.Flags().Lookup(name) == nil {
		t.Fatalf("%v flag %q missing", path, name)
	}
}
