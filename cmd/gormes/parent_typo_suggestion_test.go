package main

import (
	"strings"
	"testing"
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
			stdout, stderr, err := executeRootCommandForTest(
				newRootCommandWithRuntime(rootRuntime{}), tc.args...,
			)
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
