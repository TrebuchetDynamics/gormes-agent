package main

import (
	"strings"
	"testing"
)

// TestParentCommands_RejectUnknownSubcommands sweeps every parent
// command that exposes only subcommands (no positional Run/RunE
// payload) and proves they reject typos at the cobra layer instead
// of silently printing help and exiting 0.
//
// Background: during install testing, `gormes plugins listt` (typo
// of `list`) printed "No plugins installed." and exited 0 — the
// operator's typo silently fell through to the parent's RunE. The
// fix on `plugins` was to add `cobra.NoArgs` so any positional arg
// at the parent level errors with cobra's "unknown command"
// message. This test extends that contract to every other parent
// command in the same shape.
//
// New parents added in the future inherit this test by name —
// just add an entry below.
func TestParentCommands_RejectUnknownSubcommands(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"channels", []string{"channels", "listt"}},
		{"mcp", []string{"mcp", "logiin"}},
		{"goncho", []string{"goncho", "doctorr"}},
		{"memory", []string{"memory", "statuss"}},
		{"session", []string{"session", "browesee"}},
		{"claw", []string{"claw", "migrateee"}},
		{"profile", []string{"profile", "showw"}},
		{"kanban", []string{"kanban", "listt"}},
		{"skills", []string{"skills", "listt"}},
		{"secrets", []string{"secrets", "audit-typo"}},
		{"security", []string{"security", "audit-typo"}},
		{"agent", []string{"agent", "resett"}},
		{"navivox", []string{"navivox", "pairr"}},
		{"curator", []string{"curator", "pausee"}},
		{"acp", []string{"acp", "servee"}},
		{"setup", []string{"setup", "providerr-typo"}},
		{"gateway", []string{"gateway", "staus"}},
		// `migrate` deliberately omitted: the shared parent guard
		// preserves SuggestionsMinimumDistance-based typo guidance like
		// `did you mean "openclaw"?`. Cobra still rejects those typos
		// (TestHermesCommandAliasFidelity_RootUnknownAndTypoSuggestions
		// covers it), just through a suggestion-specific path.
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GORMES_HOME", t.TempDir())
			stdout, stderr, err := executeRootCommandForTest(
				newRootCommandWithRuntime(rootRuntime{}), tc.args...,
			)
			if err == nil {
				t.Fatalf("`%s` (typo) must error; instead got stdout=%q stderr=%q",
					strings.Join(tc.args, " "), stdout, stderr)
			}
		})
	}
}
