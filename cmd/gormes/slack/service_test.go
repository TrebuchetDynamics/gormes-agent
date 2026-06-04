package slack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestPayloadSlashesOnly(t *testing.T) {
	payload, err := ManifestPayload("Gormes Test Bot", "desc", true)
	if err != nil {
		t.Fatalf("ManifestPayload: %v", err)
	}
	slashes, ok := payload.([]map[string]any)
	if !ok {
		t.Fatalf("slashes-only payload type = %T", payload)
	}
	if !hasCommand(slashes, "/hermes") || !hasCommand(slashes, "/kanban") {
		t.Fatalf("slashes-only payload missing required commands: %#v", slashes)
	}
}

func TestExpandUserPathAndWriteFileAtomic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	expanded := ExpandUserPath("~/nested/slack.json")
	if !strings.HasPrefix(expanded, home+string(os.PathSeparator)) {
		t.Fatalf("ExpandUserPath = %q, want under %q", expanded, home)
	}
	if err := WriteFileAtomic(expanded, []byte("body"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	info, err := os.Stat(expanded)
	if err != nil {
		t.Fatalf("stat written file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
	if filepath.Base(expanded) != "slack.json" {
		t.Fatalf("expanded path = %q", expanded)
	}
}

func hasCommand(rows []map[string]any, want string) bool {
	for _, row := range rows {
		if row["command"] == want {
			return true
		}
	}
	return false
}
