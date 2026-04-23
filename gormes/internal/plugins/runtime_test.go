package plugins

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoverCatalog_ClassifiesSourcesKindsAndSelection(t *testing.T) {
	bundledRoot := t.TempDir()
	userRoot := t.TempDir()
	projectRoot := t.TempDir()

	writePluginManifest(t, filepath.Join(bundledRoot, "calculator"), `
name: calculator
version: 1.0.0
`)
	writePluginManifest(t, filepath.Join(userRoot, "weather"), `
name: weather
version: 1.0.0
requires_env:
  - WEATHER_API_KEY
`)
	writePluginManifest(t, filepath.Join(projectRoot, "team-tools"), `
name: team-tools
version: 1.0.0
`)
	writePluginManifest(t, filepath.Join(bundledRoot, "memory", "memx"), `
name: memx
version: 1.0.0
`)
	writePluginManifest(t, filepath.Join(bundledRoot, "context_engine", "compressy"), `
name: compressy
version: 1.0.0
`)

	got, err := DiscoverCatalog(DiscoveryOptions{
		BundledRoots:    []string{bundledRoot},
		UserRoots:       []string{userRoot},
		ProjectRoots:    []string{projectRoot},
		EnableProject:   true,
		DisabledGeneral: []string{"team-tools"},
		MemoryProvider:  "memx",
		ContextEngine:   "compressy",
	})
	if err != nil {
		t.Fatalf("DiscoverCatalog() error = %v", err)
	}

	if len(got) != 5 {
		t.Fatalf("len(DiscoverCatalog()) = %d, want 5", len(got))
	}

	index := make(map[string]CatalogEntry, len(got))
	for _, entry := range got {
		index[string(entry.Kind)+":"+entry.Name] = entry
	}

	if entry := index[string(KindGeneral)+":calculator"]; entry.Source != SourceBundled || entry.State != StateEnabled {
		t.Fatalf("calculator = %#v, want bundled enabled", entry)
	}
	if entry := index[string(KindGeneral)+":weather"]; entry.Source != SourceUser || entry.State != StateEnabled || !reflect.DeepEqual(entry.MissingEnv, []string{"WEATHER_API_KEY"}) {
		t.Fatalf("weather = %#v, want user enabled missing WEATHER_API_KEY", entry)
	}
	if entry := index[string(KindGeneral)+":team-tools"]; entry.Source != SourceProject || entry.State != StateDisabled {
		t.Fatalf("team-tools = %#v, want project disabled", entry)
	}
	if entry := index[string(KindMemoryProvider)+":memx"]; entry.Source != SourceBundled || entry.State != StateSelected {
		t.Fatalf("memx = %#v, want bundled selected", entry)
	}
	if entry := index[string(KindContextEngine)+":compressy"]; entry.Source != SourceBundled || entry.State != StateSelected {
		t.Fatalf("compressy = %#v, want bundled selected", entry)
	}
}

func TestDiscoverCatalog_SkipsProjectRootsUnlessEnabled(t *testing.T) {
	projectRoot := t.TempDir()
	writePluginManifest(t, filepath.Join(projectRoot, "team-tools"), `
name: team-tools
version: 1.0.0
`)

	got, err := DiscoverCatalog(DiscoveryOptions{
		ProjectRoots:  []string{projectRoot},
		EnableProject: false,
	})
	if err != nil {
		t.Fatalf("DiscoverCatalog() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(DiscoverCatalog()) = %d, want 0 when project plugins disabled", len(got))
	}
}
