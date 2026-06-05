package assets

import "testing"

func TestDashboardDistManifest(t *testing.T) {
	manifest := DashboardDistManifest()
	if manifest.EntryHTML != "index.html" {
		t.Fatalf("entry html = %q, want index.html", manifest.EntryHTML)
	}
	if manifest.AppEntry != "src/main.tsx" {
		t.Fatalf("app entry = %q, want src/main.tsx", manifest.AppEntry)
	}
	wantRoutes := []string{"/", "/chat", "/config", "/env", "/sessions", "/logs", "/cron", "/skills", "/docs", "/analytics"}
	if len(manifest.Routes) != len(wantRoutes) {
		t.Fatalf("routes = %v, want %v", manifest.Routes, wantRoutes)
	}
	for i, want := range wantRoutes {
		if manifest.Routes[i] != want {
			t.Fatalf("route[%d] = %q, want %q", i, manifest.Routes[i], want)
		}
	}
	if len(manifest.RequiredAssets) == 0 {
		t.Fatal("required assets are empty")
	}
}
