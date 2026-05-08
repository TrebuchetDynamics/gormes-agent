package main

import (
	"strings"
	"testing"
)

// TestPluginsUnknownSubcommand_FailsLoudly guards a UX bug found
// during install testing: `gormes plugins listt` (typo of `list`)
// previously printed the same "No plugins installed." text as the
// bare `gormes plugins` command and exited 0, so operators couldn't
// tell their typo had silently fallen through to the parent's RunE.
//
// New contract: any positional arg passed to the `plugins` parent
// must fail (cobra.NoArgs). `gormes plugins` (zero args) keeps the
// existing convenience and runs the list. Subcommands like
// `gormes plugins list` continue to work because cobra dispatches
// them BEFORE consulting the parent's Args validator.
func TestPluginsUnknownSubcommand_FailsLoudly(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	stdout, stderr, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}), "plugins", "listt",
	)
	if err == nil {
		t.Fatalf("plugins listt (typo) should error; stdout=%q stderr=%q", stdout, stderr)
	}
	combined := err.Error() + stderr
	if !strings.Contains(combined, "listt") && !strings.Contains(combined, "unknown") {
		t.Errorf("error must signal the typo (e.g. 'unknown command \"listt\"'); got %s", combined)
	}
}

// TestPluginsBareCommand_KeepsListConvenience is the regression fence:
// the existing convenience that `gormes plugins` (zero args) lists
// installed plugins must not regress.
func TestPluginsBareCommand_KeepsListConvenience(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	stdout, stderr, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}), "plugins",
	)
	if err != nil {
		t.Fatalf("bare plugins must succeed: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "No plugins installed") {
		t.Errorf("bare plugins must print the empty-list placeholder; got:\n%s", stdout)
	}
}

// TestPluginsExplicitListSubcommand_StillWorks pins the subcommand
// dispatch path so the cobra.NoArgs validator on the parent doesn't
// accidentally break the explicit `plugins list` invocation.
func TestPluginsExplicitListSubcommand_StillWorks(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	stdout, _, err := executeRootCommandForTest(
		newRootCommandWithRuntime(rootRuntime{}), "plugins", "list",
	)
	if err != nil {
		t.Fatalf("plugins list must succeed: %v\nstdout=%s", err, stdout)
	}
	if !strings.Contains(stdout, "No plugins installed") {
		t.Errorf("plugins list must print empty-list placeholder; got:\n%s", stdout)
	}
}
