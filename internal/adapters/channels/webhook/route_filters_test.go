package webhook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRouteFiltersMatch_CoreOperators(t *testing.T) {
	payload := map[string]any{
		"name":   "alice",
		"labels": []any{"urgent", "ops"},
		"attrs":  map[string]any{"team": "platform"},
	}
	tests := []struct {
		name string
		spec map[string]any
	}{
		{"exists", map[string]any{"field": "name", "exists": true}},
		{"not exists", map[string]any{"field": "absent", "exists": false}},
		{"missing", map[string]any{"field": "absent", "missing": true}},
		{"equals", map[string]any{"field": "name", "equals": "alice"}},
		{"not equals missing", map[string]any{"field": "absent", "not_equals": "blocked"}},
		{"contains string", map[string]any{"field": "name", "contains": "lic"}},
		{"contains list", map[string]any{"field": "labels", "contains": "urgent"}},
		{"contains map key", map[string]any{"field": "attrs", "contains": "team"}},
		{"in", map[string]any{"field": "name", "in": []any{"bob", "alice"}}},
		{"regex", map[string]any{"field": "name", "regex": `^a.*e$`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !routeFiltersMatch(tt.spec, payload, "", nil) {
				t.Fatalf("routeFiltersMatch(%v) = false, want true", tt.spec)
			}
		})
	}
}

func TestRouteFiltersMatch_InvalidSpecsFailClosed(t *testing.T) {
	payload := map[string]any{"name": "alice"}
	invalid := []any{
		"not-an-object",
		map[string]any{"all": "not-a-list"},
		map[string]any{"any": "not-a-list"},
		map[string]any{"not": "not-an-object"},
		map[string]any{"not": map[string]any{"field": "name", "unsupported": true}},
		map[string]any{"field": "name", "exists": "yes"},
		map[string]any{"field": "absent", "missing": false},
		map[string]any{"field": "name", "regex": "["},
		map[string]any{"field": "name", "in_file": "/tmp/must-not-be-read"},
		map[string]any{"field": "name", "unsupported": true},
	}
	for _, spec := range invalid {
		if routeFiltersMatch(spec, payload, "", nil) {
			t.Fatalf("invalid filter matched: %#v", spec)
		}
	}
	for _, empty := range []any{nil, map[string]any{}, []any{}} {
		if !routeFiltersMatch(empty, payload, "", nil) {
			t.Fatalf("empty filters rejected: %#v", empty)
		}
	}
}

func TestRouteFiltersMatch_InFileShapes(t *testing.T) {
	home := t.TempDir()
	tests := []struct {
		name string
		body string
	}{
		{"json list", `["chat-1","chat-2"]`},
		{"json object keys", `{"chat-1":false,"chat-2":true}`},
		{"json scalar", `"chat-2"`},
		{"trimmed lines", "chat-1\n\n chat-2 \n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(home, tt.name+".txt")
			if err := os.WriteFile(path, []byte(tt.body), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			spec := map[string]any{"field": "id", "in_file": filepath.Base(path)}
			if !routeFiltersMatchWithHome(spec, map[string]any{"id": "chat-2"}, "", nil, home) {
				t.Fatalf("in_file did not match %s watchlist", tt.name)
			}
		})
	}
}

func TestRouteFiltersMatch_InFileAliasesAndInRootSymlink(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "data"), 0o700); err != nil {
		t.Fatalf("MkdirAll(data) error = %v", err)
	}
	target := filepath.Join(home, "data", "watchlist.json")
	if err := os.WriteFile(target, []byte(`["chat-2"]`), 0o600); err != nil {
		t.Fatalf("WriteFile(watchlist) error = %v", err)
	}
	for _, path := range []string{"~/.hermes/data/watchlist.json", "~/.gormes/data/watchlist.json"} {
		spec := map[string]any{"field": "id", "in_file": path}
		if !routeFiltersMatchWithHome(spec, map[string]any{"id": "chat-2"}, "", nil, home) {
			t.Fatalf("compatibility alias %q did not match", path)
		}
	}

	link := filepath.Join(home, "watchlist-link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Logf("in-root symlink fixture unavailable: %v", err)
		return
	}
	spec := map[string]any{"field": "id", "in_file": filepath.Base(link)}
	if !routeFiltersMatchWithHome(spec, map[string]any{"id": "chat-2"}, "", nil, home) {
		t.Fatal("in-root symlink did not match")
	}
}

func TestRouteFiltersMatch_InFileRejectsUnsafeOrInvalidFiles(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "profile")
	if err := os.MkdirAll(filepath.Join(home, "directory"), 0o700); err != nil {
		t.Fatalf("MkdirAll(profile) error = %v", err)
	}
	outside := filepath.Join(root, "outside.txt")
	if err := os.WriteFile(outside, []byte("chat-2\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(outside) error = %v", err)
	}
	t.Setenv("WATCHLIST", outside)

	invalidUTF8 := filepath.Join(home, "invalid-utf8.txt")
	if err := os.WriteFile(invalidUTF8, []byte{0xff}, 0o600); err != nil {
		t.Fatalf("WriteFile(invalid UTF-8) error = %v", err)
	}
	oversizedValue := strings.Repeat("x", (1<<20)+1)
	if err := os.WriteFile(filepath.Join(home, "oversized.txt"), []byte(oversizedValue), 0o600); err != nil {
		t.Fatalf("WriteFile(oversized) error = %v", err)
	}

	tests := []struct {
		name  string
		home  string
		path  any
		value any
	}{
		{"blank home", "", "outside.txt", "chat-2"},
		{"home is not directory", outside, "~/.hermes", "chat-2"},
		{"absolute", home, outside, "chat-2"},
		{"traversal", home, "../outside.txt", "chat-2"},
		{"unsupported tilde", home, "~/outside.txt", "chat-2"},
		{"environment expansion", home, "$WATCHLIST", "chat-2"},
		{"missing", home, "missing.txt", "chat-2"},
		{"directory", home, "directory", "chat-2"},
		{"non-string path", home, 42, "chat-2"},
		{"invalid UTF-8", home, "invalid-utf8.txt", string([]byte{0xff})},
		{"oversized", home, "oversized.txt", oversizedValue},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := map[string]any{"field": "id", "in_file": tt.path}
			if routeFiltersMatchWithHome(spec, map[string]any{"id": tt.value}, "", nil, tt.home) {
				t.Fatalf("unsafe/invalid in_file matched path %#v", tt.path)
			}
		})
	}

	link := filepath.Join(home, "escape.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Logf("symlink escape fixture unavailable: %v", err)
		return
	}
	spec := map[string]any{"field": "id", "in_file": filepath.Base(link)}
	if routeFiltersMatchWithHome(spec, map[string]any{"id": "chat-2"}, "", nil, home) {
		t.Fatal("symlink escape matched")
	}
}

func TestRouteFiltersMatch_NestedGroupsAndFields(t *testing.T) {
	filters := []any{
		map[string]any{"all": []any{
			map[string]any{"field": "payload.label", "equals": "urgent"},
			map[string]any{"field": "event", "equals": "push"},
			map[string]any{"field": "event_type", "equals": "push"},
			map[string]any{"any": []any{
				map[string]any{"field": "event_type", "equals": "deployment"},
				map[string]any{"field": "headers.X-Source", "equals": "trusted"},
			}},
			map[string]any{"not": map[string]any{"field": "payload.items.0.state", "equals": "closed"}},
		}},
	}
	payload := map[string]any{"payload": map[string]any{
		"label": "urgent",
		"items": []any{map[string]any{"state": "open"}},
	}}

	if !routeFiltersMatch(filters, payload, "push", map[string]string{"x-source": "trusted"}) {
		t.Fatal("nested filters did not match payload/event/header context")
	}
}
