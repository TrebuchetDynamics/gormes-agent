package inventory

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/fidelity"
)

func TestHermesContractInventoryClassifiesPluginCatalogFamilies(t *testing.T) {
	root := t.TempDir()
	hermes := filepath.Join(root, "hermes-agent")
	writePluginCatalogHermesFixture(t, hermes)
	writePluginCatalogSourcePairs(t, root)
	writePluginCatalogProgress(t, root)

	result, err := WriteHermesContractInventory(HermesContractInventoryOptions{
		Root:             root,
		CurrentHermesSHA: "abc123",
	})
	if err != nil {
		t.Fatalf("WriteHermesContractInventory: %v", err)
	}

	families := result.Report.PluginCatalog
	if got, want := catalogFamilyIDs(families), []string{
		"browser_web_search",
		"dashboard_observability",
		"google_meet",
		"image_video_generation",
		"memory_providers",
		"model_providers",
		"platform_adapters",
		"spotify",
		"teams_pipeline",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("plugin family ids = %v, want %v", got, want)
	}

	googleMeet := catalogFamilyByID(t, families, "google_meet", "plugin catalog")
	if googleMeet.Status != fidelity.StatusCovered || googleMeet.Count == 0 {
		t.Fatalf("google_meet family = %+v, want covered with examples", googleMeet)
	}
	if !containsString(googleMeet.Examples, "plugins/google_meet/meet_bot.py") {
		t.Fatalf("google_meet examples = %v, want meet_bot.py", googleMeet.Examples)
	}
	if !containsProgressRow(googleMeet.ProgressRows, "Google Meet plugin metadata") {
		t.Fatalf("google_meet progress rows = %+v, want Google Meet plugin metadata", googleMeet.ProgressRows)
	}
	if !containsSourcePair(googleMeet.SourcePairs, "plugins/google_meet/plugin.yaml") {
		t.Fatalf("google_meet source pairs = %+v, want plugin.yaml", googleMeet.SourcePairs)
	}

	media := catalogFamilyByID(t, families, "image_video_generation", "plugin catalog")
	if media.Status != fidelity.StatusPlanned {
		t.Fatalf("image/video family status = %q, want planned", media.Status)
	}
	for _, want := range []string{"plugins/image_gen/openai-codex/plugin.yaml", "plugins/video_gen/fal/plugin.yaml"} {
		if !containsString(media.Examples, want) {
			t.Fatalf("image/video examples = %v, want %s", media.Examples, want)
		}
	}

	dashboard := catalogFamilyByID(t, families, "dashboard_observability", "plugin catalog")
	if dashboard.Status != fidelity.StatusOwnedDivergence {
		t.Fatalf("dashboard family status = %q, want owned_divergence", dashboard.Status)
	}
	if !containsString(dashboard.Examples, "plugins/example-dashboard/dashboard/manifest.json") {
		t.Fatalf("dashboard examples = %v, want dashboard manifest", dashboard.Examples)
	}

	md, err := os.ReadFile(result.MarkdownPath)
	if err != nil {
		t.Fatalf("read markdown report: %v", err)
	}
	for _, want := range []string{
		"## Plugin Catalog Classification",
		"| `google_meet` | `covered` |",
		"`plugins/google_meet/meet_bot.py`",
		"| `image_video_generation` | `planned` |",
		"| `dashboard_observability` | `owned_divergence` |",
	} {
		if !strings.Contains(string(md), want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}
}

func writePluginCatalogHermesFixture(t *testing.T, hermes string) {
	t.Helper()
	for _, rel := range []string{
		"hermes_cli/main.py",
		"plugins/model-providers/openrouter/plugin.yaml",
		"plugins/model-providers/openai-codex/plugin.yaml",
		"plugins/memory/honcho/plugin.yaml",
		"plugins/memory/hindsight/README.md",
		"plugins/browser/browser_use/provider.py",
		"plugins/browser/firecrawl/plugin.yaml",
		"plugins/platforms/simplex/adapter.py",
		"plugins/platforms/teams/plugin.yaml",
		"plugins/google_meet/meet_bot.py",
		"plugins/google_meet/plugin.yaml",
		"plugins/spotify/plugin.yaml",
		"plugins/teams_pipeline/plugin.yaml",
		"plugins/teams_pipeline/pipeline.py",
		"plugins/image_gen/openai-codex/plugin.yaml",
		"plugins/video_gen/fal/plugin.yaml",
		"plugins/example-dashboard/dashboard/manifest.json",
		"plugins/hermes-achievements/dashboard/plugin_api.py",
		"plugins/hermes-achievements/docs/achievements-performance-spec.md",
		"RELEASE_v0.14.0.md",
	} {
		path := filepath.Join(hermes, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("# fixture\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
}

func writePluginCatalogSourcePairs(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "hermes-source-pairs.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir source-pairs: %v", err)
	}
	body := `{
  "schema_version": "1.0",
  "hermes_sha": "abc123",
  "pairs": [
    {
      "hermes_file": "plugins/google_meet/plugin.yaml",
      "gormes_targets": ["internal/plugins/google_meet.go"],
      "status": "covered",
      "contract": "Google Meet plugin metadata parity.",
      "tests": ["go test ./internal/plugins -run TestGoogleMeetPluginMetadata -count=1"],
      "progress_rows": ["Google Meet plugin metadata"],
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "plugins/example-dashboard/dashboard/manifest.json",
      "gormes_targets": ["internal/plugins/manifest.go"],
      "status": "owned",
      "contract": "Gormes dashboard slots are owned but must be visible in fidelity reports.",
      "tests": ["go test ./internal/plugins -run TestLoadDirParsesManifestDashboardCapabilitiesAndRequirements -count=1"],
      "progress_rows": ["Dashboard plugin slots"],
      "last_checked_hermes_sha": "abc123"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write source-pairs: %v", err)
	}
}

func writePluginCatalogProgress(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir progress: %v", err)
	}
	p := &progress.Progress{
		Meta: progress.Meta{Version: "test"},
		Phases: map[string]progress.Phase{
			"1": {
				Name:        "Plugin fixture",
				Deliverable: "Fixture",
				Subphases: map[string]progress.Subphase{
					"1.A": {
						Name: "Plugin catalog",
						Items: []progress.Item{
							catalogProgressItem("Google Meet plugin metadata", progress.StatusComplete, progress.ContractStatusValidated, "plugins", "hermes-agent/plugins/google_meet/plugin.yaml", "go test ./internal/plugins -run TestGoogleMeetPluginMetadata -count=1"),
							catalogProgressItem("Dashboard plugin slots", progress.StatusComplete, progress.ContractStatusValidated, "plugins", "hermes-agent/plugins/example-dashboard/dashboard/manifest.json", "go test ./internal/plugins -run TestLoadDirParsesManifestDashboardCapabilitiesAndRequirements -count=1"),
							catalogProgressItem("Provider plugin catalog parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "providers", "hermes-agent/plugins/model-providers/openrouter/plugin.yaml", "go test ./internal/provider -run TestProviderPluginCatalog -count=1"),
							catalogProgressItem("Memory provider plugin catalog parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "memory", "hermes-agent/plugins/memory/honcho/plugin.yaml", "go test ./internal/memory -run TestMemoryPluginCatalog -count=1"),
							catalogProgressItem("Browser and web plugin catalog parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "browser", "hermes-agent/plugins/browser/browser_use/provider.py", "go test ./internal/tools -run TestBrowserPluginCatalog -count=1"),
							catalogProgressItem("Gateway platform plugin catalog parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "channels", "hermes-agent/plugins/platforms/simplex/adapter.py", "go test ./internal/gateway -run TestPlatformPluginCatalog -count=1"),
							catalogProgressItem("Spotify plugin catalog parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "plugins", "hermes-agent/plugins/spotify/plugin.yaml", "go test ./internal/plugins -run TestSpotifyPluginMetadata -count=1"),
							catalogProgressItem("Teams pipeline plugin catalog parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "plugins", "hermes-agent/plugins/teams_pipeline/plugin.yaml", "go test ./internal/plugins -run TestTeamsPipelinePluginMetadata -count=1"),
							catalogProgressItem("Image and video plugin catalog parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "plugins", "hermes-agent/plugins/image_gen/openai-codex/plugin.yaml", "go test ./internal/plugins -run TestMediaPluginCatalog -count=1"),
						},
					},
				},
			},
		},
	}
	if err := progress.SaveProgress(path, p); err != nil {
		t.Fatalf("write progress fixture: %v", err)
	}
}
