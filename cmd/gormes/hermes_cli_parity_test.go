package main

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

func TestHermesCLIParityManifest(t *testing.T) {
	entries := hermesCLIParityManifest()
	if len(entries) < 120 {
		t.Fatalf("manifest has %d entries, want broad top-level/nested/slash/plugin coverage", len(entries))
	}

	wantTopLevel := []string{
		"chat", "model", "fallback", "gateway", "setup", "whatsapp", "slack", "login", "logout", "auth", "status", "cron", "webhook", "hooks", "doctor", "dump", "debug", "backup", "import", "config", "pairing", "skills", "plugins", "memory", "tools", "mcp", "sessions", "insights", "claw", "version", "update", "uninstall", "acp", "profile", "completion", "dashboard", "logs",
	}
	for _, path := range wantTopLevel {
		entry := requireHermesCLIEntry(t, []string{path})
		if entry.Kind != hermesCLICommand && entry.Kind != hermesCLICommandSet {
			t.Fatalf("top-level %q kind = %q, want command or command_set", path, entry.Kind)
		}
	}

	wantNested := [][]string{
		{"gateway", "status"}, {"gateway", "restart"}, {"gateway", "reset"}, {"gateway", "model"}, {"gateway", "profile"}, {"gateway", "usage"},
		{"fallback", "show"}, {"fallback", "set"}, {"fallback", "clear"},
		{"auth", "login"}, {"auth", "logout"}, {"auth", "status"},
		{"cron", "list"}, {"cron", "add"}, {"cron", "remove"}, {"cron", "run"},
		{"webhook", "serve"}, {"webhook", "test"},
		{"hooks", "list"}, {"hooks", "run"},
		{"debug", "doctor"}, {"debug", "share"},
		{"config", "show"}, {"config", "set"}, {"config", "check"}, {"config", "edit"}, {"config", "migrate"},
		{"pairing", "approve"}, {"pairing", "deny"}, {"pairing", "list"},
		{"skills", "list"}, {"skills", "search"}, {"skills", "install"}, {"skills", "remove"}, {"skills", "tap"}, {"skills", "snapshot"},
		{"plugins", "list"}, {"plugins", "enable"}, {"plugins", "disable"},
		{"memory", "search"}, {"memory", "add"}, {"memory", "status"},
		{"tools", "list"}, {"tools", "doctor"},
		{"mcp", "list"}, {"mcp", "call"},
		{"sessions", "list"}, {"sessions", "resume"}, {"sessions", "export"},
		{"claw", "migrate"}, {"claw", "cleanup"},
		{"profile", "show"}, {"profile", "set"},
	}
	for _, path := range wantNested {
		entry := requireHermesCLIEntry(t, path)
		if entry.SourceRef == "" || entry.Residual == "" {
			t.Fatalf("nested %v missing source/residual: %+v", path, entry)
		}
	}
}

func TestHermesCLIParityManifestNoUnknowns(t *testing.T) {
	entries := hermesCLIParityManifest()
	seen := map[string]bool{}
	for _, entry := range entries {
		key := strings.Join(entry.Path, " ")
		if key == "" {
			t.Fatalf("empty path entry: %+v", entry)
		}
		if seen[key] {
			t.Fatalf("duplicate path %q", key)
		}
		seen[key] = true
		if entry.Kind == "" || entry.Status == "" || entry.SourceRef == "" {
			t.Fatalf("entry %q missing kind/status/source: %+v", key, entry)
		}
		if entry.Status == hermesCLIImplemented && entry.Target == "" {
			t.Fatalf("implemented entry %q missing target", key)
		}
		if entry.Status == hermesCLIRowBacked && entry.Residual == "" {
			t.Fatalf("row-backed entry %q missing residual/row evidence", key)
		}
	}
}

func TestHermesCLIParityManifestCrossLinksSlashRegistry(t *testing.T) {
	var want []string
	for _, policy := range cli.CommandRegistry {
		want = append(want, "/"+policy.Name)
		for _, alias := range policy.Aliases {
			want = append(want, "/"+alias)
		}
	}
	sort.Strings(want)

	var got []string
	for _, entry := range hermesCLIParityManifest() {
		if entry.Kind == hermesCLISlashCommand || entry.Kind == hermesCLIAlias {
			got = append(got, strings.Join(entry.Path, " "))
		}
	}
	sort.Strings(got)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("slash manifest mismatch\n got: %#v\nwant: %#v", got, want)
	}
	for _, path := range [][]string{{"/steer"}, {"/usage"}, {"/bg"}, {"/set-home"}} {
		entry := requireHermesCLIEntry(t, path)
		if entry.SourceRef == "" || entry.Residual == "" {
			t.Fatalf("slash entry %v missing classification: %+v", path, entry)
		}
	}
}

func TestHermesCLIParityManifestClassifiesDynamicPluginsAndGormesDivergences(t *testing.T) {
	for _, path := range [][]string{
		{"plugins", "dynamic", "memory"},
		{"plugins", "dynamic", "disk-cleanup"},
	} {
		entry := requireHermesCLIEntry(t, path)
		if entry.Kind != hermesCLIPluginCommand || !entry.Dynamic || entry.Status == "" {
			t.Fatalf("dynamic plugin entry %v = %+v, want dynamic plugin classification", path, entry)
		}
	}
	for _, path := range [][]string{{"goncho"}, {"--offline"}, {"--remote"}} {
		entry := requireHermesCLIEntry(t, path)
		if entry.Status != hermesCLIOwned {
			t.Fatalf("Gormes-owned entry %v = %+v, want owned divergence", path, entry)
		}
	}
	oneshoot := requireHermesCLIEntry(t, []string{"-z"})
	if oneshoot.Status != hermesCLIImplemented || oneshoot.Target == "" {
		t.Fatalf("-z/--oneshot entry = %+v, want implemented Hermes parity", oneshoot)
	}
	typo := requireHermesCLIEntry(t, []string{"migrate", "ooenclaw"})
	if !strings.Contains(typo.Residual, "migrate openclaw") || typo.Status != hermesCLIRowBacked {
		t.Fatalf("typo suggestion entry = %+v, want explicit row-backed openclaw suggestion", typo)
	}
}

func requireHermesCLIEntry(t *testing.T, path []string) hermesCLIParityEntry {
	t.Helper()
	key := strings.Join(path, " ")
	for _, entry := range hermesCLIParityManifest() {
		if strings.Join(entry.Path, " ") == key {
			return entry
		}
	}
	t.Fatalf("missing Hermes CLI parity entry %q", key)
	return hermesCLIParityEntry{}
}
