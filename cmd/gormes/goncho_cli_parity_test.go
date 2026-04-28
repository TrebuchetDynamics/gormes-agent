package main

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestGonchoCLIParityManifestMatchesUpstreamCommandTree(t *testing.T) {
	want := readUpstreamHonchoCLICommandPaths(t)
	got := parityCommandPaths(gonchoCLICompatibilityManifest())
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CLI command tree mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestGonchoCLIParityManifestClassifiesEveryEntry(t *testing.T) {
	entries := gonchoCLICompatibilityManifest()
	if len(entries) != 53 {
		t.Fatalf("entries len = %d, want 53 command/flag parity entries", len(entries))
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		key := strings.Join(entry.Path, " ")
		if key == "" {
			t.Fatalf("empty path entry: %+v", entry)
		}
		if seen[key] {
			t.Fatalf("duplicate CLI parity path %q", key)
		}
		seen[key] = true
		if entry.Kind == "" || entry.Status == "" || entry.SourceRef == "" || entry.Residual == "" {
			t.Fatalf("entry %q missing classification/source/residual: %+v", key, entry)
		}
		if entry.Status == gonchoCLIImplemented && entry.Target == "" {
			t.Fatalf("implemented entry %q missing target", key)
		}
		if entry.Status == gonchoCLIRowBacked && entry.Target != "" {
			t.Fatalf("row-backed entry %q unexpectedly has target %q", key, entry.Target)
		}
	}
}

func TestGonchoCLIParityManifestMapsImplementedDoctorToCobraCommand(t *testing.T) {
	entry := requireGonchoCLIEntry(t, []string{"doctor"})
	if entry.Status != gonchoCLIImplemented || entry.Target != "cmd/gormes goncho doctor" || !entry.JSONOutput || !entry.ExitCodes {
		t.Fatalf("doctor entry = %+v, want implemented JSON/exit-code target", entry)
	}

	root := newRootCommand()
	goncho, _, err := root.Find([]string{"goncho"})
	if err != nil || goncho == nil {
		t.Fatalf("find goncho command err=%v cmd=%v", err, goncho)
	}
	doctor, _, err := goncho.Find([]string{"doctor"})
	if err != nil || doctor == nil {
		t.Fatalf("find goncho doctor err=%v cmd=%v", err, doctor)
	}
}

func TestGonchoCLIParityManifestPreservesRiskContracts(t *testing.T) {
	for _, path := range [][]string{
		{"workspace", "delete"},
		{"session", "delete"},
		{"conclusion", "delete"},
	} {
		entry := requireGonchoCLIEntry(t, path)
		if !entry.Confirm || !entry.ExitCodes {
			t.Fatalf("%v entry = %+v, want destructive confirm and exit-code evidence", path, entry)
		}
	}
	for _, path := range [][]string{
		{"--json"},
		{"--version"},
		{"-V"},
		{"--workspace"},
		{"-w"},
		{"--peer"},
		{"-p"},
		{"--session"},
		{"-s"},
	} {
		entry := requireGonchoCLIEntry(t, path)
		if entry.Kind != gonchoCLIGlobalFlag {
			t.Fatalf("%v entry = %+v, want global flag", path, entry)
		}
	}
	for _, path := range [][]string{
		{"workspace", "list"},
		{"workspace", "search"},
		{"message", "get"},
	} {
		entry := requireGonchoCLIEntry(t, path)
		if !entry.JSONOutput {
			t.Fatalf("%v entry = %+v, want JSON-output evidence", path, entry)
		}
	}
}

func TestGonchoCLIParityManifestReturnsDefensiveCopies(t *testing.T) {
	entries := gonchoCLICompatibilityManifest()
	entries[0].Path[0] = "mutated"
	if again := gonchoCLICompatibilityManifest(); again[0].Path[0] == "mutated" {
		t.Fatalf("manifest path was mutated through returned slice: %+v", again[0])
	}
}

func requireGonchoCLIEntry(t *testing.T, path []string) gonchoCLIParityEntry {
	t.Helper()
	key := strings.Join(path, " ")
	for _, entry := range gonchoCLICompatibilityManifest() {
		if strings.Join(entry.Path, " ") == key {
			return entry
		}
	}
	t.Fatalf("missing CLI parity entry %q", key)
	return gonchoCLIParityEntry{}
}

func parityCommandPaths(entries []gonchoCLIParityEntry) []string {
	var out []string
	for _, entry := range entries {
		if entry.Kind == gonchoCLICommand || entry.Kind == gonchoCLICommandSet {
			out = append(out, strings.Join(entry.Path, " "))
		}
	}
	sort.Strings(out)
	return out
}

func readUpstreamHonchoCLICommandPaths(t *testing.T) []string {
	t.Helper()
	root := filepath.Clean("../../../honcho/honcho-cli/src/honcho_cli")
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			t.Skipf("upstream Honcho CLI source not present at %s", root)
		}
		t.Fatalf("stat upstream Honcho CLI: %v", err)
	}

	paths := []string{"init", "doctor", "help", "config"}
	for _, group := range []struct {
		name string
		file string
	}{
		{"workspace", "workspace.py"},
		{"peer", "peer.py"},
		{"session", "session.py"},
		{"message", "message.py"},
		{"conclusion", "conclusion.py"},
	} {
		paths = append(paths, group.name)
		for _, command := range parseTyperCommandNames(t, filepath.Join(root, "commands", group.file)) {
			paths = append(paths, group.name+" "+command)
		}
	}
	sort.Strings(paths)
	return paths
}

func parseTyperCommandNames(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	lines := strings.Split(string(raw), "\n")
	explicit := regexp.MustCompile(`@app\.command\("([^"]+)"\)`)
	implicit := regexp.MustCompile(`@app\.command\(\)`)
	defLine := regexp.MustCompile(`def ([A-Za-z_][A-Za-z0-9_]*)\(`)
	var out []string
	for i, line := range lines {
		if match := explicit.FindStringSubmatch(line); match != nil {
			out = append(out, match[1])
			continue
		}
		if !implicit.MatchString(line) {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			if match := defLine.FindStringSubmatch(lines[j]); match != nil {
				out = append(out, strings.ReplaceAll(match[1], "_", "-"))
				break
			}
		}
	}
	sort.Strings(out)
	return out
}
