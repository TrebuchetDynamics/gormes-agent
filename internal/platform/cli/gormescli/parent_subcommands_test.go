package gormescli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestParentCommands_TypoEmitsDidYouMean pins the next layer of UX
// the bare `cobra.NoArgs` guard misses: cobra's `NoArgs` helper
// returns a flat "unknown command X for Y" error WITHOUT invoking
// `findSuggestions`. So an operator who typed `gormes session lst`
// (1-edit-distance from `list`) sees only the bare error, with no
// hint that `list` was 1 character away — even though `gormes confg`
// (1-edit-distance from `config`) at the root surfaces the full
// "Did you mean: config" path. The two paths should agree.
//
// This contract is a layer above the existing
// TestParentCommands_RejectUnknownSubcommands sweep — that sweep
// only proves typos error. This sweep proves they error WITH a
// suggestion when one is in cobra's edit-distance window.
func TestParentCommands_TypoEmitsDidYouMean(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		wantSuggest string
	}{
		{"session", []string{"session", "lst"}, "list"},
		{"session", []string{"session", "expor"}, "export"},
		{"memory", []string{"memory", "statuss"}, "status"},
		{"goncho", []string{"goncho", "doctorr"}, "doctor"},
		{"kanban", []string{"kanban", "lst"}, "list"},
		{"profile", []string{"profile", "lst"}, "list"},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			t.Setenv("GORMES_HOME", t.TempDir())
			stdout, stderr, err := executeParentGuardRootForTest(newParentGuardRootForTest(), tc.args...)
			if err == nil {
				t.Fatalf("`%s` must error; got stdout=%q stderr=%q",
					strings.Join(tc.args, " "), stdout, stderr)
			}
			combined := err.Error() + stderr
			// Match the case-insensitive form to keep the test stable
			// across cobra's `Did you mean this?` format and Gormes'
			// inline `did you mean "X"?` form.
			if !strings.Contains(strings.ToLower(combined), "did you mean") {
				t.Fatalf("typo `%s` must include `did you mean…`; got:\n%s",
					strings.Join(tc.args, " "), combined)
			}
			if !strings.Contains(combined, tc.wantSuggest) {
				t.Fatalf("typo `%s` must suggest %q; got:\n%s",
					strings.Join(tc.args, " "), tc.wantSuggest, combined)
			}
		})
	}
}

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
		// (TestMigrateOpenClawDryRun_RejectsMissingDryRunAndTypo covers it),
		// just through a suggestion-specific path.
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GORMES_HOME", t.TempDir())
			stdout, stderr, err := executeParentGuardRootForTest(newParentGuardRootForTest(), tc.args...)
			if err == nil {
				t.Fatalf("`%s` (typo) must error; instead got stdout=%q stderr=%q",
					strings.Join(tc.args, " "), stdout, stderr)
			}
		})
	}
}

func TestParentCommands_JSONSubcommandRequired(t *testing.T) {
	stdout, stderr, err := executeParentGuardRootForTest(newParentGuardRootForTest(), "session", "--json")
	if err == nil {
		t.Fatalf("session --json error = nil; stdout=%s stderr=%s", stdout, stderr)
	}
	for _, want := range []string{`"action": "subcommand_required"`, `"parent": "gormes session"`, `"list"`, `"export"`} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("session --json stdout missing %s:\nstdout=%s\nstderr=%s", want, stdout, stderr)
		}
	}
}

func newParentGuardRootForTest() *cobra.Command {
	factories := stubRootFactories()
	fixtures := map[string][]string{
		"channels": {"list"},
		"mcp":      {"login"},
		"goncho":   {"doctor"},
		"memory":   {"status"},
		"session":  {"list", "export", "browse"},
		"claw":     {"migrate"},
		"profile":  {"list", "show"},
		"kanban":   {"list"},
		"skills":   {"list"},
		"secrets":  {"audit"},
		"security": {"audit"},
		"agent":    {"reset"},
		"navivox":  {"pair"},
		"curator":  {"pause"},
		"acp":      {"serve"},
		"setup":    {"provider"},
	}
	for parent, children := range fixtures {
		parent, children := parent, children
		factories[parent] = func() *cobra.Command {
			cmd := &cobra.Command{Use: parent}
			for _, child := range children {
				cmd.AddCommand(&cobra.Command{Use: child, Run: func(*cobra.Command, []string) {}})
			}
			return cmd
		}
	}
	factories["gateway"] = func() *cobra.Command {
		cmd := &cobra.Command{Use: "gateway", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error { return nil }}
		cmd.AddCommand(&cobra.Command{Use: "status", Run: func(*cobra.Command, []string) {}})
		return cmd
	}
	return NewRootCommand(RootOptions{
		Version: "test-version",
		Finalizers: []func(*cobra.Command){
			func(cmd *cobra.Command) {
				InstallParentUnknownSubcommandGuards(cmd, ParentUnknownSubcommandGuardOptions{
					BuildProvenance: testBuildProvenance,
					ExitCodeError:   NewExitCodeError,
				})
			},
		},
	}, factories)
}

func executeParentGuardRootForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}
