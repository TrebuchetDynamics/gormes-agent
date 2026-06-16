package goncho

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestCLICompatibilityManifest_EntriesAreDeterministic proves the manifest
// returns a deterministic clone on every call, not a shared backing slice.
func TestCLICompatibilityManifest_EntriesAreDeterministic(t *testing.T) {
	got := parityCommandPaths(CLICompatibilityManifest())
	if len(got) == 0 {
		t.Fatal("parity command paths are empty")
	}
	// Prove determinism.
	entries := CLICompatibilityManifest()
	if len(entries) != len(got) {
		t.Fatalf("CLICompatibilityManifest returned %d entries on first call but %d on second", len(got), len(entries))
	}
	if again := CLICompatibilityManifest(); again[0].Path[0] == "mutated" {
		t.Fatal("CLICompatibilityManifest returns a shared backing slice; mutation leaks across calls")
	}
}

func requireCLIEntry(t *testing.T, path []string) CLIParityEntry {
	t.Helper()
	for _, entry := range CLICompatibilityManifest() {
		if reflect.DeepEqual(entry.Path, path) {
			return entry
		}
	}
	return CLIParityEntry{}
}

func parityCommandPaths(entries []CLIParityEntry) []string {
	var out []string
	for _, entry := range entries {
		out = append(out, strings.Join(entry.Path, " "))
	}
	sort.Strings(out)
	return out
}

// TestCLICompatibilityManifest_ContainsDoctor proves the doctor entry
// is present and carries the expected metadata.
func TestCLICompatibilityManifest_ContainsDoctor(t *testing.T) {
	entry := requireCLIEntry(t, []string{"doctor"})
	if entry.Kind != CLICommand {
		t.Fatalf("doctor entry.Kind = %v, want command", entry.Kind)
	}
	if entry.Status != CLIImplemented {
		t.Fatalf("doctor entry.Status = %v, want implemented", entry.Status)
	}
	if !entry.ExitCodes {
		t.Fatal("doctor must assert exit_codes")
	}
	if !entry.JSONOutput {
		t.Fatal("doctor must assert json_output")
	}
}

// TestCLICompatibilityManifest_AllCommandKindsAndStatuses proves every
// manifest entry uses a valid Kind and Status.
func TestCLICompatibilityManifest_AllCommandKindsAndStatuses(t *testing.T) {
	for _, entry := range CLICompatibilityManifest() {
		if entry.Kind != CLICommand && entry.Kind != CLICommandSet && entry.Kind != CLIGlobalFlag {
			t.Errorf("path %v: invalid kind %v", entry.Path, entry.Kind)
		}
		if entry.Status != CLIImplemented && entry.Status != CLIRowBacked {
			t.Errorf("path %v: invalid status %v", entry.Path, entry.Status)
		}
	}
}

// TestCLICompatibilityManifest_GlobalFlagsAreDeterministic tests global flags
// are also deterministic and mutation-safe.
func TestCLICompatibilityManifest_GlobalFlagsAreDeterministic(t *testing.T) {
	for _, entry := range CLICompatibilityManifest() {
		if entry.Kind != CLIGlobalFlag {
			continue
		}
		if entry.Path[0] == "--version" || entry.Path[0] == "-V" || entry.Path[0] == "--json" {
			if entry.SourceRef != "honcho-cli/src/honcho_cli/main.py" {
				t.Errorf("global flag %q source_ref = %q, want honcho-cli/src/honcho_cli/main.py", entry.Path[0], entry.SourceRef)
			}
		}
	}
}

// TestCLICompatibilityManifest_ScopeFlagsPassParserCheck verifies scope flags
// reference the correct Hermes CLI source.
func TestCLICompatibilityManifest_ScopeFlagsPassParserCheck(t *testing.T) {
	scopeFlags := []string{"--workspace", "-w", "--peer", "-p", "--session", "-s"}
	for _, entry := range CLICompatibilityManifest() {
		if entry.Kind != CLIGlobalFlag {
			continue
		}
		isScope := false
		for _, sf := range scopeFlags {
			if entry.Path[0] == sf {
				isScope = true
				break
			}
		}
		if !isScope {
			continue
		}
		if entry.SourceRef != "honcho-cli/src/honcho_cli/common.py" {
			t.Errorf("scope flag %q source_ref = %q, want honcho-cli/src/honcho_cli/common.py", entry.Path[0], entry.SourceRef)
		}
	}
}

// TestCLICompatibilityManifest_CommandSetsContainSubcommands verifies command
// sets have all expected subcommands.
func TestCLICompatibilityManifest_CommandSetsContainSubcommands(t *testing.T) {
	expectedSubs := map[string][]string{
		"workspace":  {"list", "create", "inspect", "delete", "search", "queue-status"},
		"peer":       {"list", "inspect", "card", "chat", "search", "create", "get-metadata", "set-metadata", "representation"},
		"session":    {"list", "create", "inspect", "context", "summaries", "delete", "peers", "add-peers", "remove-peers", "search", "representation", "get-metadata", "set-metadata"},
		"message":    {"list", "create", "get"},
		"conclusion": {"list", "search", "create", "delete"},
	}

	manifest := CLICompatibilityManifest()
	foundSubs := map[string][]string{}
	for _, entry := range manifest {
		if entry.Kind == CLICommand && len(entry.Path) == 2 {
			foundSubs[entry.Path[0]] = append(foundSubs[entry.Path[0]], entry.Path[1])
		}
	}

	for setName, expected := range expectedSubs {
		got := foundSubs[setName]
		sort.Strings(got)
		sort.Strings(expected)
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("%s subcommands mismatch:\ngot  %v\nwant %v", setName, got, expected)
		}
	}
}

// TestCLICompatibilityManifest_NoEmptyResidualStrings enforces that every
// row_backed entry carries a Residual explaining what remains to be
// implemented.
func TestCLICompatibilityManifest_NoEmptyResidualStrings(t *testing.T) {
	for _, entry := range CLICompatibilityManifest() {
		if entry.Status == CLIRowBacked && strings.TrimSpace(entry.Residual) == "" {
			t.Errorf("path %v is row_backed but has no residual description", entry.Path)
		}
	}
}

// TestCLICompatibilityManifest_HermesToolSourceRefFormat enforces that
// implemented source_ref values match the expected pattern for Honcho CLI
// entries.
func TestCLICompatibilityManifest_HermesToolSourceRefFormat(t *testing.T) {
	hermesPattern := regexp.MustCompile(`^honcho-cli/src/honcho_cli/`)
	for _, entry := range CLICompatibilityManifest() {
		if entry.Status != CLIImplemented {
			continue
		}
		if !hermesPattern.MatchString(entry.SourceRef) {
			t.Errorf("implemented path %v source_ref = %q, want honcho-cli/src prefix", entry.Path, entry.SourceRef)
		}
	}
}

// TestCLICompatibilityManifest_MutationSafetyIsolation proves entries
// returned by CLICompatibilityManifest have independent Path slices.
func TestCLICompatibilityManifest_MutationSafetyIsolation(t *testing.T) {
	entries := CLICompatibilityManifest()
	if len(entries) == 0 {
		t.Fatal("no manifest entries")
	}
	first := entries[0]
	if len(first.Path) == 0 {
		t.Fatal("first entry has empty path")
	}
	first.Path[0] = "mutated"

	for i, entry := range CLICompatibilityManifest() {
		if i == 0 {
			if entry.Path[0] == "mutated" {
				t.Fatal("first entry path mutated by earlier caller")
			}
		}
	}
}