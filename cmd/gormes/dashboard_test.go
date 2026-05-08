package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestDashboardHelpDoesNotPanic guards against pflag shorthand collisions
// between the root persistent flagset and the dashboard subcommand. A `-p`
// shorthand on dashboard's --port previously collided with the root
// persistent --profile flag, panicking inside cobra.mergePersistentFlags
// for `dashboard --help`, `dashboard -h`, `help dashboard`, and shell
// tab-completion (__complete dashboard ...).
func TestDashboardHelpDoesNotPanic(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"dashboard --help", []string{"dashboard", "--help"}},
		{"dashboard -h", []string{"dashboard", "-h"}},
		{"help dashboard", []string{"help", "dashboard"}},
		{"__complete dashboard", []string{"__complete", "dashboard", ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := newRootCommand()
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs(tc.args)
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Execute(%v) panicked: %v", tc.args, r)
				}
			}()
			if err := root.Execute(); err != nil {
				t.Fatalf("Execute(%v) returned error: %v", tc.args, err)
			}
			combined := stdout.String() + stderr.String()
			if tc.args[0] != "__complete" && !strings.Contains(combined, "dashboard") {
				t.Fatalf("Execute(%v) output missing 'dashboard' marker; got:\n%s", tc.args, combined)
			}
		})
	}
}
