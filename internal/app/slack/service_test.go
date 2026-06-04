package slack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeManifestName(t *testing.T) {
	cases := map[string]string{
		" Model! ":                             "model",
		"---":                                  "",
		"UPPER_and-dash":                       "upper_and-dash",
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJ": "abcdefghijklmnopqrstuvwxyzabcdef",
	}
	for in, want := range cases {
		if got := SanitizeManifestName(in); got != want {
			t.Fatalf("SanitizeManifestName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestManifestSlashCommands(t *testing.T) {
	slashes := ManifestSlashCommands("https://example.test/slack/commands")
	if len(slashes) == 0 || len(slashes) > 50 {
		t.Fatalf("slash command count = %d, want 1..50", len(slashes))
	}
	if !hasCommand(slashes, "/hermes") || !hasCommand(slashes, "/model") {
		t.Fatalf("slash commands missing expected entries: %#v", slashes)
	}
	for _, row := range slashes {
		if row["url"] != "https://example.test/slack/commands" {
			t.Fatalf("slash command url = %#v", row)
		}
		if row["command"] == "/status" {
			t.Fatalf("reserved slack command leaked into manifest: %#v", row)
		}
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

func TestManifestPayloadDefaultsAndSlashesOnly(t *testing.T) {
	payload, err := ManifestPayload("", "", false)
	if err != nil {
		t.Fatal(err)
	}
	manifest := payload.(map[string]any)
	features := manifest["features"].(map[string]any)
	if _, ok := features["app_home"].(map[string]any); !ok {
		t.Fatalf("full manifest missing app_home: %#v", manifest)
	}
	if _, ok := manifest["oauth_config"].(map[string]any); !ok {
		t.Fatalf("full manifest missing oauth_config: %#v", manifest)
	}

	slashesOnly, err := ManifestPayload("", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := slashesOnly.([]map[string]any); !ok {
		t.Fatalf("slashes-only payload type = %T", slashesOnly)
	}
}

func hasCommand(slashes []map[string]any, want string) bool {
	for _, row := range slashes {
		if row["command"] == want {
			return true
		}
	}
	return false
}
