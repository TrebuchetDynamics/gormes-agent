package webhook

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWebhookDynamicRoutes_LoadMergeMtimeAndRemoval(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, DynamicRoutesFilename)
	static := map[string]RouteConfig{
		"static":   {Secret: "static-secret", Prompt: "static"},
		"conflict": {Secret: "static-secret", Prompt: "static wins"},
	}
	routes := NewDynamicRouteSet(home, static)

	if err := routes.Reload(); err != nil {
		t.Fatalf("Reload(no file) error = %v", err)
	}
	if _, ok := routes.Route("static"); !ok {
		t.Fatal("static route missing before dynamic file exists")
	}

	writeDynamicRoutes(t, path, `{
		"dynamic": {"secret": "dynamic-secret", "prompt": "dyn", "events": ["push"]},
		"conflict": {"secret": "dynamic-secret", "prompt": "dynamic loses"}
	}`)
	if err := routes.Reload(); err != nil {
		t.Fatalf("Reload(dynamic file) error = %v", err)
	}
	if got, ok := routes.Route("dynamic"); !ok || got.Secret != "dynamic-secret" {
		t.Fatalf("dynamic route = %#v ok:%v, want loaded route", got, ok)
	}
	if got, _ := routes.Route("conflict"); got.Secret != "static-secret" {
		t.Fatalf("conflict route secret = %q, want static-secret", got.Secret)
	}

	routes.dynamic["injected"] = RouteConfig{Secret: "test"}
	if err := routes.Reload(); err != nil {
		t.Fatalf("Reload(same mtime) error = %v", err)
	}
	if _, ok := routes.Route("injected"); !ok {
		t.Fatal("same mtime reload dropped dynamic route; want mtime gate")
	}

	next := time.Now().Add(2 * time.Second)
	writeDynamicRoutes(t, path, `{"replacement": {"secret": "next", "prompt": "next"}}`)
	if err := os.Chtimes(path, next, next); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}
	if err := routes.Reload(); err != nil {
		t.Fatalf("Reload(updated file) error = %v", err)
	}
	if _, ok := routes.Route("replacement"); !ok {
		t.Fatal("replacement dynamic route missing after changed mtime")
	}
	if _, ok := routes.Route("dynamic"); ok {
		t.Fatal("old dynamic route still present after changed file")
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove(dynamic file) error = %v", err)
	}
	if err := routes.Reload(); err != nil {
		t.Fatalf("Reload(removed file) error = %v", err)
	}
	if _, ok := routes.Route("replacement"); ok {
		t.Fatal("dynamic route still present after file removal")
	}
	if _, ok := routes.Route("static"); !ok {
		t.Fatal("static route missing after dynamic file removal")
	}
}

func TestWebhookDynamicRoutes_CorruptFileKeepsStaticRoutes(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, DynamicRoutesFilename)
	routes := NewDynamicRouteSet(home, map[string]RouteConfig{
		"static": {Secret: "static-secret", Prompt: "static"},
	})

	writeDynamicRoutes(t, path, `not json`)
	if err := routes.Reload(); err == nil {
		t.Fatal("Reload(corrupt file) error = nil, want non-nil degraded evidence")
	}
	if _, ok := routes.Route("static"); !ok {
		t.Fatal("static route missing after corrupt dynamic file")
	}
	if got := routes.DynamicCount(); got != 0 {
		t.Fatalf("DynamicCount = %d, want 0 after corrupt first load", got)
	}
}

func writeDynamicRoutes(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
