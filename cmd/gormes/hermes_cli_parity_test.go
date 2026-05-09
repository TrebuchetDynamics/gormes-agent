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
		"chat", "model", "gateway", "setup", "whatsapp", "slack", "login", "logout", "auth", "status", "cron", "webhook", "hooks", "doctor", "dump", "debug", "backup", "import", "config", "pairing", "skills", "plugins", "memory", "tools", "mcp", "sessions", "insights", "claw", "curator", "version", "update", "uninstall", "acp", "profile", "completion", "dashboard", "logs",
	}
	for _, path := range wantTopLevel {
		entry := requireHermesCLIEntry(t, []string{path})
		if entry.Kind != hermesCLICommand && entry.Kind != hermesCLICommandSet {
			t.Fatalf("top-level %q kind = %q, want command or command_set", path, entry.Kind)
		}
	}

	wantNested := [][]string{
		{"gateway", "run"}, {"gateway", "restart"}, {"gateway", "status"}, {"gateway", "install"}, {"gateway", "migrate-legacy"},
		{"auth", "add"}, {"auth", "list"}, {"auth", "remove"}, {"auth", "reset"}, {"auth", "status"}, {"auth", "logout"}, {"auth", "spotify"},
		{"cron", "list"}, {"cron", "create"}, {"cron", "add"}, {"cron", "edit"}, {"cron", "remove"}, {"cron", "delete"}, {"cron", "status"}, {"cron", "tick"},
		{"webhook", "subscribe"}, {"webhook", "add"}, {"webhook", "test"},
		{"hooks", "list"}, {"hooks", "ls"}, {"hooks", "revoke"}, {"hooks", "doctor"},
		{"debug", "share"}, {"debug", "delete"},
		{"config", "show"}, {"config", "set"}, {"config", "check"}, {"config", "edit"}, {"config", "migrate"}, {"config", "env-path"},
		{"pairing", "approve"}, {"pairing", "revoke"}, {"pairing", "clear-pending"}, {"pairing", "list"},
		{"skills", "browse"}, {"skills", "search"}, {"skills", "install"}, {"skills", "uninstall"}, {"skills", "tap"}, {"skills", "tap", "add"}, {"skills", "snapshot"}, {"skills", "snapshot", "export"},
		{"plugins", "list"}, {"plugins", "ls"}, {"plugins", "enable"}, {"plugins", "disable"}, {"plugins", "update"}, {"plugins", "remove"},
		{"memory", "setup"}, {"memory", "status"}, {"memory", "off"}, {"memory", "reset"},
		{"tools", "list"}, {"tools", "enable"}, {"tools", "disable"},
		{"mcp", "serve"}, {"mcp", "list"}, {"mcp", "ls"}, {"mcp", "test"}, {"mcp", "configure"}, {"mcp", "config"}, {"mcp", "login"},
		{"sessions", "list"}, {"sessions", "export"}, {"sessions", "delete"}, {"sessions", "prune"}, {"sessions", "stats"}, {"sessions", "browse"},
		{"claw", "migrate"}, {"claw", "cleanup"}, {"claw", "clean"},
		{"curator", "status"}, {"curator", "run"}, {"curator", "pause"}, {"curator", "resume"}, {"curator", "pin"}, {"curator", "unpin"}, {"curator", "backup"}, {"curator", "rollback"}, {"curator", "restore"}, {"curator", "archive"}, {"curator", "list-archived"}, {"curator", "prune"},
		{"profile", "list"}, {"profile", "use"}, {"profile", "create"}, {"profile", "show"}, {"profile", "import"},
	}
	for _, path := range wantNested {
		entry := requireHermesCLIEntry(t, path)
		if entry.SourceRef == "" || entry.Residual == "" {
			t.Fatalf("nested %v missing source/residual: %+v", path, entry)
		}
	}
}

func TestHermesCLIParityManifestNestedParserInventoryMatchesHermes(t *testing.T) {
	want := map[string][][]string{
		"gateway": {
			{"gateway", "run"}, {"gateway", "start"}, {"gateway", "stop"}, {"gateway", "restart"}, {"gateway", "status"}, {"gateway", "install"}, {"gateway", "uninstall"}, {"gateway", "setup"}, {"gateway", "migrate-legacy"}, {"gateway", "list"},
		},
		"slack": {
			{"slack", "manifest"},
		},
		"auth": {
			{"auth", "add"}, {"auth", "list"}, {"auth", "remove"}, {"auth", "reset"}, {"auth", "status"}, {"auth", "logout"}, {"auth", "spotify"},
		},
		"cron": {
			{"cron", "list"}, {"cron", "create"}, {"cron", "add"}, {"cron", "edit"}, {"cron", "pause"}, {"cron", "resume"}, {"cron", "run"}, {"cron", "remove"}, {"cron", "rm"}, {"cron", "delete"}, {"cron", "status"}, {"cron", "tick"},
		},
		"webhook": {
			{"webhook", "subscribe"}, {"webhook", "add"}, {"webhook", "list"}, {"webhook", "ls"}, {"webhook", "remove"}, {"webhook", "rm"}, {"webhook", "test"},
		},
		"hooks": {
			{"hooks", "list"}, {"hooks", "ls"}, {"hooks", "test"}, {"hooks", "revoke"}, {"hooks", "remove"}, {"hooks", "rm"}, {"hooks", "doctor"},
		},
		"debug": {
			{"debug", "share"}, {"debug", "delete"},
		},
		"config": {
			{"config", "show"}, {"config", "edit"}, {"config", "set"}, {"config", "path"}, {"config", "env-path"}, {"config", "check"}, {"config", "migrate"},
		},
		"pairing": {
			{"pairing", "list"}, {"pairing", "approve"}, {"pairing", "revoke"}, {"pairing", "clear-pending"},
		},
		"skills": {
			{"skills", "browse"}, {"skills", "search"}, {"skills", "install"}, {"skills", "inspect"}, {"skills", "list"}, {"skills", "check"}, {"skills", "update"}, {"skills", "audit"}, {"skills", "uninstall"}, {"skills", "reset"}, {"skills", "publish"}, {"skills", "snapshot"}, {"skills", "snapshot", "export"}, {"skills", "snapshot", "import"}, {"skills", "tap"}, {"skills", "tap", "list"}, {"skills", "tap", "add"}, {"skills", "tap", "remove"}, {"skills", "config"},
		},
		"plugins": {
			{"plugins", "install"}, {"plugins", "update"}, {"plugins", "remove"}, {"plugins", "rm"}, {"plugins", "uninstall"}, {"plugins", "list"}, {"plugins", "ls"}, {"plugins", "enable"}, {"plugins", "disable"},
		},
		"memory": {
			{"memory", "setup"}, {"memory", "status"}, {"memory", "off"}, {"memory", "reset"},
		},
		"tools": {
			{"tools", "list"}, {"tools", "disable"}, {"tools", "enable"},
		},
		"mcp": {
			{"mcp", "serve"}, {"mcp", "add"}, {"mcp", "remove"}, {"mcp", "rm"}, {"mcp", "list"}, {"mcp", "ls"}, {"mcp", "test"}, {"mcp", "configure"}, {"mcp", "config"}, {"mcp", "login"},
		},
		"sessions": {
			{"sessions", "list"}, {"sessions", "export"}, {"sessions", "delete"}, {"sessions", "prune"}, {"sessions", "stats"}, {"sessions", "rename"}, {"sessions", "browse"},
		},
		"claw": {
			{"claw", "migrate"}, {"claw", "cleanup"}, {"claw", "clean"},
		},
		"curator": {
			{"curator", "status"}, {"curator", "run"}, {"curator", "pause"}, {"curator", "resume"}, {"curator", "pin"}, {"curator", "unpin"}, {"curator", "restore"}, {"curator", "list-archived"}, {"curator", "archive"}, {"curator", "prune"}, {"curator", "backup"}, {"curator", "rollback"},
		},
		"profile": {
			{"profile", "list"}, {"profile", "use"}, {"profile", "create"}, {"profile", "delete"}, {"profile", "show"}, {"profile", "alias"}, {"profile", "rename"}, {"profile", "export"}, {"profile", "import"},
		},
	}
	for group, paths := range want {
		for _, path := range paths {
			entry := requireHermesCLIEntry(t, path)
			if entry.SourceRef == "" || entry.Residual == "" {
				t.Fatalf("%s parser path %v missing source/residual: %+v", group, path, entry)
			}
		}
	}

	stale := [][]string{
		{"gateway", "reset"}, {"gateway", "help"}, {"gateway", "model"}, {"gateway", "profile"}, {"gateway", "update"}, {"gateway", "approve"}, {"gateway", "deny"}, {"gateway", "voice"}, {"gateway", "usage"},
		{"cron", "enable"}, {"cron", "disable"},
		{"webhook", "serve"},
		{"hooks", "run"},
		{"debug", "doctor"}, {"debug", "paste"}, {"debug", "sweep"},
		{"pairing", "deny"}, {"pairing", "reset"},
		{"skills", "remove"},
		{"plugins", "doctor"},
		{"memory", "search"}, {"memory", "add"}, {"memory", "delete"}, {"memory", "export"},
		{"tools", "doctor"},
		{"mcp", "call"}, {"mcp", "auth"},
		{"sessions", "resume"},
		{"profile", "set"},
		{"auth", "login"}, {"auth", "refresh"},
	}
	for _, path := range stale {
		if entry, ok := findHermesCLIEntry(path); ok && entry.Kind == hermesCLICommand {
			t.Fatalf("stale nested parser path %v should not be an active parser command: %+v", path, entry)
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
		if len(entry.Path) > 0 && strings.HasPrefix(entry.Path[0], "/") && (entry.Kind == hermesCLISlashCommand || entry.Kind == hermesCLIAlias) {
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

func TestHermesCLIParityManifestClassifiesKanbanCoreAndResiduals(t *testing.T) {
	top := requireHermesCLIEntry(t, []string{"kanban"})
	if top.Status != hermesCLIRowBacked || !strings.Contains(top.Residual, "durable board core") {
		t.Fatalf("kanban top-level entry = %+v, want row-backed command set with core/residual evidence", top)
	}

	for _, path := range [][]string{
		{"kanban", "init"},
		{"kanban", "create"},
		{"kanban", "list"},
		{"kanban", "show"},
		{"kanban", "claim"},
		{"kanban", "complete"},
		{"kanban", "block"},
		{"kanban", "unblock"},
		{"kanban", "link"},
	} {
		entry := requireHermesCLIEntry(t, path)
		if entry.Status != hermesCLIImplemented || !strings.Contains(entry.Target, "cmd/gormes kanban") {
			t.Fatalf("kanban core entry %v = %+v, want implemented cmd/gormes target", path, entry)
		}
	}

	for _, path := range [][]string{
		{"kanban", "boards"},
		{"kanban", "dispatch"},
		{"kanban", "comment"},
		{"kanban", "context"},
	} {
		entry := requireHermesCLIEntry(t, path)
		if entry.Status != hermesCLIRowBacked || entry.Row == "" {
			t.Fatalf("kanban residual entry %v = %+v, want row-backed residual", path, entry)
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
	for _, path := range [][]string{{"goncho"}, {"agent"}, {"agent", "reset"}, {"--offline"}, {"--remote"}} {
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

func TestHermesCLIParityManifestClawCleanupImplemented(t *testing.T) {
	cleanup := requireHermesCLIEntry(t, []string{"claw", "cleanup"})
	if cleanup.Status != hermesCLIImplemented || cleanup.Target != "cmd/gormes claw cleanup" {
		t.Fatalf("claw cleanup entry = %+v, want implemented cmd/gormes claw cleanup target", cleanup)
	}
	clean := requireHermesCLIEntry(t, []string{"claw", "clean"})
	if clean.Status != hermesCLIImplemented || !reflect.DeepEqual(clean.AliasFor, []string{"claw", "cleanup"}) {
		t.Fatalf("claw clean entry = %+v, want implemented alias for claw cleanup", clean)
	}
}

func TestHermesCLIParityManifestProviderAuthCommandsMatchHermes(t *testing.T) {
	login := requireHermesCLIEntry(t, []string{"login"})
	if login.Status != hermesCLIExcluded || !strings.Contains(strings.ToLower(login.Residual), "removed") {
		t.Fatalf("top-level login = %+v, want excluded removed-command compatibility entry", login)
	}

	logout := requireHermesCLIEntry(t, []string{"logout"})
	if logout.Status != hermesCLIRowBacked || !logout.RedactsSecrets {
		t.Fatalf("top-level logout = %+v, want row-backed redacted provider logout", logout)
	}

	for _, path := range [][]string{
		{"auth", "add"},
		{"auth", "list"},
		{"auth", "remove"},
		{"auth", "reset"},
		{"auth", "status"},
		{"auth", "logout"},
		{"auth", "spotify"},
	} {
		entry := requireHermesCLIEntry(t, path)
		if entry.SourceRef == "" || entry.Residual == "" {
			t.Fatalf("%v missing source/residual: %+v", path, entry)
		}
		if entry.Status == hermesCLIImplemented {
			if entry.Target == "" {
				t.Fatalf("%v implemented entry missing target: %+v", path, entry)
			}
			continue
		}
		var wantRow string
		switch entry.Path[1] {
		case "add":
			wantRow = "Hermes auth OAuth provider adapters"
		case "spotify":
			wantRow = "Hermes auth Spotify service-provider subcommand"
		default:
			wantRow = "Hermes auth credential-pool command surface"
		}
		if entry.Row != wantRow {
			t.Fatalf("%v row = %q, want %q: %+v", path, entry.Row, wantRow, entry)
		}
	}
	for _, implemented := range [][]string{{"auth", "list"}, {"auth", "remove"}, {"auth", "reset"}, {"auth", "status"}, {"auth", "logout"}} {
		entry := requireHermesCLIEntry(t, implemented)
		if entry.Status != hermesCLIImplemented {
			t.Fatalf("%v status = %q, want implemented credential-pool command", implemented, entry.Status)
		}
	}
	for _, removed := range [][]string{{"auth", "login"}, {"auth", "refresh"}} {
		if entry, ok := findHermesCLIEntry(removed); ok {
			t.Fatalf("deprecated/nonexistent auth command %v should not be manifest-active: %+v", removed, entry)
		}
	}

	add := requireHermesCLIEntry(t, []string{"auth", "add"})
	if !add.RedactsSecrets {
		t.Fatalf("auth add = %+v, want secret redaction flag", add)
	}
	remove := requireHermesCLIEntry(t, []string{"auth", "remove"})
	if !remove.Destructive {
		t.Fatalf("auth remove = %+v, want destructive flag", remove)
	}
	spotify := requireHermesCLIEntry(t, []string{"auth", "spotify"})
	if !strings.Contains(spotify.Residual, "login|status|logout") {
		t.Fatalf("auth spotify residual = %q, want action inventory", spotify.Residual)
	}
}

func TestHermesCLIParityManifestGatewayStopIsImplemented(t *testing.T) {
	entry := requireHermesCLIEntry(t, []string{"gateway", "stop"})
	if entry.Status != hermesCLIImplemented {
		t.Fatalf("gateway stop status = %q, want implemented: %+v", entry.Status, entry)
	}
	if entry.Target != "cmd/gormes gateway stop" {
		t.Fatalf("gateway stop target = %q, want cmd/gormes gateway stop", entry.Target)
	}
	if !strings.Contains(entry.Residual, "validated runtime PID") {
		t.Fatalf("gateway stop residual = %q, want validated runtime PID evidence", entry.Residual)
	}
}

func TestHermesCLIParityManifestOmitsRetiredFallbackCommands(t *testing.T) {
	for _, path := range [][]string{
		{"fallback"},
		{"fallback", "list"},
		{"fallback", "ls"},
		{"fallback", "add"},
		{"fallback", "remove"},
		{"fallback", "rm"},
		{"fallback", "clear"},
		{"fallback", "show"},
		{"fallback", "set"},
	} {
		if entry, ok := findHermesCLIEntry(path); ok {
			t.Fatalf("retired fallback command %v should not remain in current Hermes parity manifest: %+v", path, entry)
		}
	}
}

func requireHermesCLIEntry(t *testing.T, path []string) hermesCLIParityEntry {
	t.Helper()
	if entry, ok := findHermesCLIEntry(path); ok {
		return entry
	}
	t.Fatalf("missing Hermes CLI parity entry %q", strings.Join(path, " "))
	return hermesCLIParityEntry{}
}

func findHermesCLIEntry(path []string) (hermesCLIParityEntry, bool) {
	key := strings.Join(path, " ")
	for _, entry := range hermesCLIParityManifest() {
		if strings.Join(entry.Path, " ") == key {
			return entry, true
		}
	}
	return hermesCLIParityEntry{}, false
}
