package mcpstore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/configwriter"
	mcpconfig "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
)

func TestStoreUpsertHTTPAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile", "config.toml")
	store := Store{Path: path}
	if err := store.UpsertHTTP("linear", HTTPRecord{URL: "https://mcp.linear.app/mcp", OAuth: true, Enabled: true}); err != nil {
		t.Fatalf("UpsertHTTP: %v", err)
	}
	resolved, err := store.Load(mcpconfig.MCPConfigOptions{LookupEnv: noEnvironment})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(resolved.Servers) != 1 {
		t.Fatalf("servers = %#v", resolved.Servers)
	}
	server := resolved.Servers[0]
	if server.Name != "linear" || server.Transport != mcpconfig.MCPTransportHTTP || server.URL != "https://mcp.linear.app/mcp" || !server.Enabled {
		t.Fatalf("server = %#v", server)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	parent, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat parent: %v", err)
	}
	if parent.Mode().Perm()&0o077 != 0 {
		t.Fatalf("parent mode = %o, want no group/world bits", parent.Mode().Perm())
	}
}

func TestStoreAddHTTPDuplicateRequiresOverwriteAndPreservesTools(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	store := Store{Path: path}
	if err := store.UpsertHTTP("custom", HTTPRecord{
		URL:          "https://old.example/mcp",
		Enabled:      true,
		ToolsInclude: []string{"search", "create"},
	}); err != nil {
		t.Fatalf("seed UpsertHTTP: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}
	err = store.AddHTTP("custom", HTTPRecord{URL: "https://new.example/private", OAuth: true, Enabled: true}, false)
	var storeErr *Error
	if !errors.As(err, &storeErr) || storeErr.Kind != ErrorAlreadyExists {
		t.Fatalf("duplicate err = %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after duplicate: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("duplicate changed config:\n%s", after)
	}
	if err := store.AddHTTP("custom", HTTPRecord{URL: "https://new.example/private", OAuth: true, Enabled: true}, true); err != nil {
		t.Fatalf("force AddHTTP: %v", err)
	}
	doc, err := configwriter.ReadTOMLDoc(path)
	if err != nil {
		t.Fatalf("ReadTOMLDoc: %v", err)
	}
	servers, _ := doc["mcp_servers"].(map[string]any)
	custom, _ := servers["custom"].(map[string]any)
	tools, _ := custom["tools"].(map[string]any)
	include, _ := tools["include"].([]any)
	if custom["url"] != "https://new.example/private" || custom["auth"] != "oauth" || len(include) != 2 || include[0] != "search" || include[1] != "create" {
		t.Fatalf("custom = %#v", custom)
	}
}

func TestStoreListStructurallyRedactsAndSorts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(path, map[string]any{
		"mcp_servers": map[string]any{
			"zeta": map[string]any{
				"command": `C:\\private\\bin\\secret-server`,
				"args":    []string{"--token", "argument-secret"},
				"env":     map[string]string{"PRIVATE_TOKEN": "environment-secret"},
				"enabled": false,
			},
			"alpha": map[string]any{
				"url":     "https://user:password@example.test/private/path?token=query-secret#fragment",
				"headers": map[string]string{"Authorization": "Bearer header-secret"},
				"enabled": true,
				"tools":   map[string]any{"include": []string{"one", "two"}},
			},
		},
	}); err != nil {
		t.Fatalf("WriteTOMLAtomic: %v", err)
	}
	rows, err := (Store{Path: path}).List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 2 || rows[0].Name != "alpha" || rows[1].Name != "zeta" {
		t.Fatalf("rows = %#v", rows)
	}
	if rows[0].Transport != "http" || rows[0].Target != "https://example.test" || rows[0].Auth != "header" || rows[0].ToolsSelected != 2 {
		t.Fatalf("alpha = %#v", rows[0])
	}
	if rows[1].Transport != "stdio" || rows[1].Target != "secret-server" || rows[1].Enabled || rows[1].Status != "disabled" {
		t.Fatalf("zeta = %#v", rows[1])
	}
	rendered := fmt.Sprintf("%#v", rows)
	for _, secret := range []string{"user", "password", "private/path", "query-secret", "fragment", "argument-secret", "PRIVATE_TOKEN", "environment-secret", "header-secret", `C:\\private`} {
		if strings.Contains(rendered, secret) {
			t.Fatalf("rows leaked %q: %s", secret, rendered)
		}
	}
}

func TestStoreServerResolvesExactRecordAndAuth(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(path, map[string]any{
		"mcp_servers": map[string]any{
			"ink": map[string]any{
				"url":     "https://example.test/mcp",
				"headers": map[string]any{"Authorization": "Bearer ${MCP_TEST_TOKEN}"},
				"enabled": true,
			},
			"oauth": map[string]any{"url": "https://oauth.example/mcp", "auth": "oauth", "enabled": true},
		},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	lookup := func(name string) (string, bool) {
		if name == "MCP_TEST_TOKEN" {
			return "private-token", true
		}
		return "", false
	}
	record, found, err := (Store{Path: path}).Server("ink", mcpconfig.MCPConfigOptions{LookupEnv: lookup})
	if err != nil || !found {
		t.Fatalf("Server ink = %+v, %v, %v", record, found, err)
	}
	if record.Status.Status != mcpconfig.MCPConfigStatusReady || record.Auth != "header" || record.Definition.Headers["Authorization"] != "Bearer private-token" {
		t.Fatalf("record = %+v", record)
	}
	oauth, found, err := (Store{Path: path}).Server("oauth", mcpconfig.MCPConfigOptions{LookupEnv: lookup})
	if err != nil || !found || oauth.Auth != "oauth" {
		t.Fatalf("Server oauth = %+v, %v, %v", oauth, found, err)
	}
	_, found, err = (Store{Path: path}).Server("missing", mcpconfig.MCPConfigOptions{LookupEnv: lookup})
	if err != nil || found {
		t.Fatalf("Server missing found=%v err=%v", found, err)
	}
	_, found, err = (Store{Path: path}).Server("bad/name", mcpconfig.MCPConfigOptions{LookupEnv: lookup})
	if err == nil || found || err.Error() != "mcp invalid input" {
		t.Fatalf("Server invalid found=%v err=%v", found, err)
	}
}

func TestStoreServerReturnsDisabledWithoutDefinition(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(path, map[string]any{
		"mcp_servers": map[string]any{"off": map[string]any{"url": "https://example.test/mcp", "enabled": false}},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	record, found, err := (Store{Path: path}).Server("off", mcpconfig.MCPConfigOptions{})
	if err != nil || !found || record.Status.Status != mcpconfig.MCPConfigStatusDisabled || record.Definition.Name != "off" || record.Definition.Enabled {
		t.Fatalf("Server off = %+v, %v, %v", record, found, err)
	}
}

func TestStoreConfigureToolsIncludeNoneAllPreservesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile", "config.toml")
	if err := configwriter.WriteTOMLAtomic(path, map[string]any{
		"hermes": map[string]any{"model": "keep-me"},
		"mcp_servers": map[string]any{
			"demo": map[string]any{
				"url":     "https://example.test/private?token=query-secret",
				"headers": map[string]any{"Authorization": "Bearer ${MCP_TEST_TOKEN}"},
				"enabled": true,
				"tools":   map[string]any{"exclude": []string{"old"}, "prompts": false},
			},
			"plain":   map[string]any{"url": "https://plain.example/mcp", "enabled": true, "tools": map[string]any{"include": []string{"old"}}},
			"sibling": map[string]any{"url": "https://sibling.example/mcp", "enabled": true},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	store := Store{Path: path}
	if err := store.ConfigureTools("demo", ToolSelection{Mode: ToolSelectionInclude, Include: []string{"search", "create"}}); err != nil {
		t.Fatalf("include: %v", err)
	}
	assertSelection := func(wantMode string, want []string) map[string]any {
		t.Helper()
		doc, err := configwriter.ReadTOMLDoc(path)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		servers, _ := doc["mcp_servers"].(map[string]any)
		demo, _ := servers["demo"].(map[string]any)
		tools, _ := demo["tools"].(map[string]any)
		if wantMode != "all" {
			gotRaw, exists := tools["include"]
			if !exists {
				t.Fatalf("include missing: %#v", tools)
			}
			got, _ := gotRaw.([]any)
			if len(got) != len(want) {
				t.Fatalf("include=%#v want=%#v", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("include=%#v want=%#v", got, want)
				}
			}
		}
		if _, exists := tools["exclude"]; exists || tools["prompts"] != false {
			t.Fatalf("tools=%#v", tools)
		}
		if demo["url"] != "https://example.test/private?token=query-secret" || demo["enabled"] != true {
			t.Fatalf("demo=%#v", demo)
		}
		if _, exists := servers["sibling"]; !exists {
			t.Fatalf("sibling lost: %#v", servers)
		}
		hermes, _ := doc["hermes"].(map[string]any)
		if hermes["model"] != "keep-me" {
			t.Fatalf("unrelated config changed: %#v", doc)
		}
		return tools
	}
	assertSelection("include", []string{"create", "search"})
	if err := store.ConfigureTools("demo", ToolSelection{Mode: ToolSelectionNone}); err != nil {
		t.Fatalf("none: %v", err)
	}
	assertSelection("none", []string{})
	if err := store.ConfigureTools("demo", ToolSelection{Mode: ToolSelectionAll}); err != nil {
		t.Fatalf("all: %v", err)
	}
	tools := assertSelection("all", nil)
	if _, exists := tools["include"]; exists {
		t.Fatalf("all retained include: %#v", tools)
	}
	if err := store.ConfigureTools("plain", ToolSelection{Mode: ToolSelectionAll}); err != nil {
		t.Fatalf("all plain: %v", err)
	}
	doc, _ := configwriter.ReadTOMLDoc(path)
	servers, _ := doc["mcp_servers"].(map[string]any)
	plain, _ := servers["plain"].(map[string]any)
	if _, exists := plain["tools"]; exists {
		t.Fatalf("empty tools table survived: %#v", plain)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode=%v", info.Mode().Perm())
	}
}

func TestStoreConfigureToolsFailuresAreNonMutating(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(path, map[string]any{
		"mcp_servers": map[string]any{
			"demo":      map[string]any{"url": "https://example.test/mcp", "enabled": true},
			"bad_tools": map[string]any{"url": "https://bad.example/mcp", "enabled": true, "tools": "private-secret"},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	store := Store{Path: path}
	tooMany := make([]string, MaxToolSelections+1)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("tool_%03d", i)
	}
	longName := strings.Repeat("x", MaxToolNameRunes+1)
	tests := []struct {
		name      string
		server    string
		selection ToolSelection
		kind      ErrorKind
	}{
		{name: "missing mode", server: "demo", selection: ToolSelection{}, kind: ErrorInvalidInput},
		{name: "missing server", server: "missing", selection: ToolSelection{Mode: ToolSelectionAll}, kind: ErrorNotFound},
		{name: "invalid server", server: "bad/name", selection: ToolSelection{Mode: ToolSelectionAll}, kind: ErrorInvalidInput},
		{name: "empty include", server: "demo", selection: ToolSelection{Mode: ToolSelectionInclude}, kind: ErrorInvalidInput},
		{name: "blank tool", server: "demo", selection: ToolSelection{Mode: ToolSelectionInclude, Include: []string{""}}, kind: ErrorInvalidInput},
		{name: "trimmed tool", server: "demo", selection: ToolSelection{Mode: ToolSelectionInclude, Include: []string{" tool"}}, kind: ErrorInvalidInput},
		{name: "duplicate tool", server: "demo", selection: ToolSelection{Mode: ToolSelectionInclude, Include: []string{"same", "same"}}, kind: ErrorInvalidInput},
		{name: "long tool", server: "demo", selection: ToolSelection{Mode: ToolSelectionInclude, Include: []string{longName}}, kind: ErrorInvalidInput},
		{name: "control tool", server: "demo", selection: ToolSelection{Mode: ToolSelectionInclude, Include: []string{"bad\nname"}}, kind: ErrorInvalidInput},
		{name: "too many", server: "demo", selection: ToolSelection{Mode: ToolSelectionInclude, Include: tooMany}, kind: ErrorInvalidInput},
		{name: "include with none", server: "demo", selection: ToolSelection{Mode: ToolSelectionNone, Include: []string{"extra"}}, kind: ErrorInvalidInput},
		{name: "malformed tools", server: "bad_tools", selection: ToolSelection{Mode: ToolSelectionAll}, kind: ErrorRejected},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before, _ := os.ReadFile(path)
			err := store.ConfigureTools(test.server, test.selection)
			var storeErr *Error
			if !errors.As(err, &storeErr) || storeErr.Kind != test.kind {
				t.Fatalf("err=%v want kind=%s", err, test.kind)
			}
			after, _ := os.ReadFile(path)
			if string(after) != string(before) {
				t.Fatalf("failure changed config:\n%s", after)
			}
			if strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), path) {
				t.Fatalf("error leaked: %v", err)
			}
		})
	}
	before, _ := os.ReadFile(path)
	writerStore := Store{Path: path, WriteDocument: func(string, map[string]any) error {
		return errors.New("private path and token-secret")
	}}
	err := writerStore.ConfigureTools("demo", ToolSelection{Mode: ToolSelectionAll})
	var storeErr *Error
	if !errors.As(err, &storeErr) || storeErr.Kind != ErrorUnavailable || strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("writer err=%v", err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatalf("writer failure changed config:\n%s", after)
	}
}

func TestStoreRemoveDeletesLastTableAndPreservesUnrelatedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(path, map[string]any{
		"hermes": map[string]any{"model": "keep-me"},
		"mcp_servers": map[string]any{
			"custom": map[string]any{"url": "https://example.test/mcp", "enabled": true},
		},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	removed, err := (Store{Path: path}).Remove("custom")
	if err != nil || !removed {
		t.Fatalf("Remove = %v, %v", removed, err)
	}
	doc, err := configwriter.ReadTOMLDoc(path)
	if err != nil {
		t.Fatalf("ReadTOMLDoc: %v", err)
	}
	if _, exists := doc["mcp_servers"]; exists {
		t.Fatalf("empty mcp_servers survived: %#v", doc)
	}
	hermes, _ := doc["hermes"].(map[string]any)
	if hermes["model"] != "keep-me" {
		t.Fatalf("unrelated config changed: %#v", doc)
	}
}

func TestStoreRemoveMissingAndWriteFailureLeaveBytesUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(path, map[string]any{
		"mcp_servers": map[string]any{
			"custom": map[string]any{"url": "https://example.test/mcp", "enabled": true},
		},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	before, _ := os.ReadFile(path)
	removed, err := (Store{Path: path}).Remove("missing")
	if err != nil || removed {
		t.Fatalf("Remove missing = %v, %v", removed, err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatalf("missing remove changed config:\n%s", after)
	}
	store := Store{Path: path, WriteDocument: func(string, map[string]any) error {
		return errors.New("private path and secret")
	}}
	removed, err = store.Remove("custom")
	if err == nil || removed || err.Error() != "mcp config unavailable" {
		t.Fatalf("Remove failed writer = %v, %v", removed, err)
	}
	after, _ = os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatalf("write failure changed config:\n%s", after)
	}
	if strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("error leaked writer detail: %q", err)
	}
}

func TestStoreWriteFailureLeavesExistingConfigUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	original := []byte("[hermes]\nmodel = 'keep-me'\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	store := Store{
		Path: path,
		WriteDocument: func(string, map[string]any) error {
			return errors.New("host path and secret must not escape")
		},
	}
	err := store.UpsertHTTP("linear", HTTPRecord{URL: "https://mcp.linear.app/mcp", OAuth: true, Enabled: true})
	if err == nil || err.Error() != "mcp config unavailable" {
		t.Fatalf("err = %v", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read config: %v", readErr)
	}
	if string(after) != string(original) {
		t.Fatalf("config changed after failed write:\n%s", after)
	}
	if strings.Contains(err.Error(), path) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("error leaked internals: %q", err)
	}
}

func TestStoreLoadRejectsMalformedServerBlockWithoutPathLeak(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private-profile", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("mcp_servers = 'wrong'\n"), 0o600); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	_, err := (Store{Path: path}).Load(mcpconfig.MCPConfigOptions{LookupEnv: noEnvironment})
	if err == nil || err.Error() != "mcp config rejected" {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(err.Error(), path) || strings.Contains(err.Error(), "wrong") {
		t.Fatalf("error leaked config details: %q", err)
	}
}

func TestStoreRejectsUnsafeHTTPRecordBeforeWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	calls := 0
	store := Store{Path: path, WriteDocument: func(string, map[string]any) error {
		calls++
		return nil
	}}
	for _, record := range []HTTPRecord{
		{URL: "file:///tmp/server", Enabled: true},
		{URL: "https:///missing-host", Enabled: true},
	} {
		if err := store.UpsertHTTP("linear", record); err == nil {
			t.Fatalf("UpsertHTTP(%q) succeeded", record.URL)
		}
	}
	if calls != 0 {
		t.Fatalf("write calls = %d, want 0", calls)
	}
	if _, err := configwriter.ReadTOMLDoc(path); err != nil {
		t.Fatalf("read absent doc: %v", err)
	}
}
