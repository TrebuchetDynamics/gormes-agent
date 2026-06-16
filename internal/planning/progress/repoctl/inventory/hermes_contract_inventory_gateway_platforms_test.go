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

func TestHermesContractInventoryClassifiesGatewayPlatformSurface(t *testing.T) {
	root := t.TempDir()
	hermes := filepath.Join(root, "hermes-agent")
	writeGatewayPlatformHermesFixture(t, hermes)
	writeGatewayPlatformSourcePairs(t, root)
	writeGatewayPlatformProgress(t, root)

	result, err := WriteHermesContractInventory(HermesContractInventoryOptions{
		Root:             root,
		CurrentHermesSHA: "abc123",
	})
	if err != nil {
		t.Fatalf("WriteHermesContractInventory: %v", err)
	}

	families := result.Report.GatewayPlatformCatalog
	if got, want := catalogFamilyIDs(families), []string{
		"api_server_surface",
		"builtin_platform_connectors",
		"bundled_platform_plugins",
		"gateway_runtime_lifecycle",
		"generated_artifacts",
		"platform_docs",
		"platform_enum_config",
		"platform_helpers",
		"tui_gateway_bridge",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("gateway platform family ids = %v, want %v", got, want)
	}

	enum := catalogFamilyByID(t, families, "platform_enum_config", "gateway platform")
	if enum.Status != fidelity.StatusCovered || enum.Count == 0 {
		t.Fatalf("platform enum family = %+v, want covered with examples", enum)
	}
	if !containsString(enum.Examples, "gateway/config.py") {
		t.Fatalf("platform enum examples = %v, want gateway/config.py", enum.Examples)
	}
	if !containsProgressRow(enum.ProgressRows, "Hermes gateway platform registry manifest") {
		t.Fatalf("platform enum progress rows = %+v, want manifest row", enum.ProgressRows)
	}
	if !containsSourcePair(enum.SourcePairs, "gateway/config.py") {
		t.Fatalf("platform enum source pairs = %+v, want gateway/config.py", enum.SourcePairs)
	}

	connectors := catalogFamilyByID(t, families, "builtin_platform_connectors", "gateway platform")
	for _, want := range []string{
		"gateway/platforms/telegram.py",
		"gateway/platforms/msgraph_webhook.py",
		"gateway/platforms/qqbot/adapter.py",
		"gateway/platforms/yuanbao.py",
	} {
		if !containsString(connectors.Examples, want) {
			t.Fatalf("builtin connector examples = %v, want %s", connectors.Examples, want)
		}
	}

	plugins := catalogFamilyByID(t, families, "bundled_platform_plugins", "gateway platform")
	if plugins.Status != fidelity.StatusCovered {
		t.Fatalf("bundled platform plugin status = %q, want covered", plugins.Status)
	}
	for _, want := range []string{"plugins/platforms/google_chat/plugin.yaml", "plugins/platforms/simplex/adapter.py"} {
		if !containsString(plugins.Examples, want) {
			t.Fatalf("bundled platform plugin examples = %v, want %s", plugins.Examples, want)
		}
	}

	helpers := catalogFamilyByID(t, families, "platform_helpers", "gateway platform")
	if helpers.Status != fidelity.StatusPlanned {
		t.Fatalf("platform helper status = %q, want planned", helpers.Status)
	}
	for _, want := range []string{"gateway/platforms/base.py", "gateway/platforms/helpers.py", "gateway/platforms/_http_client_limits.py"} {
		if !containsString(helpers.Examples, want) {
			t.Fatalf("platform helper examples = %v, want %s", helpers.Examples, want)
		}
	}

	tui := catalogFamilyByID(t, families, "tui_gateway_bridge", "gateway platform")
	if tui.Status != fidelity.StatusCovered {
		t.Fatalf("tui gateway bridge status = %q, want covered", tui.Status)
	}
	for _, want := range []string{"tui_gateway/server.py", "tui_gateway/render.py", "tui_gateway/ws.py"} {
		if !containsString(tui.Examples, want) {
			t.Fatalf("tui gateway examples = %v, want %s", tui.Examples, want)
		}
	}

	generated := catalogFamilyByID(t, families, "generated_artifacts", "gateway platform")
	if generated.Status != fidelity.StatusExcluded {
		t.Fatalf("generated artifact status = %q, want excluded", generated.Status)
	}
	if !containsString(generated.Examples, "gateway/platforms/generated_platforms.pyc") {
		t.Fatalf("generated examples = %v, want generated pyc", generated.Examples)
	}

	md, err := os.ReadFile(result.MarkdownPath)
	if err != nil {
		t.Fatalf("read markdown report: %v", err)
	}
	for _, want := range []string{
		"## Gateway Platform Classification",
		"| `platform_enum_config` | `covered` |",
		"`gateway/config.py`",
		"| `platform_helpers` | `planned` |",
		"| `tui_gateway_bridge` | `covered` |",
		"| `generated_artifacts` | `excluded` |",
	} {
		if !strings.Contains(string(md), want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}
}

func writeGatewayPlatformHermesFixture(t *testing.T, hermes string) {
	t.Helper()
	files := map[string]string{
		"hermes_cli/main.py":                        "# fixture\n",
		"gateway/config.py":                         "class Platform:\n    TELEGRAM = 'telegram'\n",
		"gateway/run.py":                            "def start_gateway():\n    pass\n",
		"gateway/platforms/ADDING_A_PLATFORM.md":    "# Adding a platform\n",
		"gateway/platforms/base.py":                 "class BasePlatformAdapter: pass\n",
		"gateway/platforms/helpers.py":              "# helpers\n",
		"gateway/platforms/_http_client_limits.py":  "# limits\n",
		"gateway/platforms/api_server.py":           "# api server\n",
		"gateway/platforms/telegram.py":             "# telegram\n",
		"gateway/platforms/discord.py":              "# discord\n",
		"gateway/platforms/slack.py":                "# slack\n",
		"gateway/platforms/whatsapp.py":             "# whatsapp\n",
		"gateway/platforms/msgraph_webhook.py":      "# msgraph\n",
		"gateway/platforms/qqbot/adapter.py":        "# qqbot\n",
		"gateway/platforms/yuanbao.py":              "# yuanbao\n",
		"gateway/platforms/yuanbao_media.py":        "# yuanbao media\n",
		"gateway/platforms/generated_platforms.pyc": "generated\n",
		"plugins/platforms/google_chat/plugin.yaml": "name: google_chat\n",
		"plugins/platforms/google_chat/adapter.py":  "# google chat\n",
		"plugins/platforms/irc/plugin.yaml":         "name: irc\n",
		"plugins/platforms/line/adapter.py":         "# line\n",
		"plugins/platforms/simplex/adapter.py":      "# simplex\n",
		"plugins/platforms/teams/adapter.py":        "# teams\n",
		"tui_gateway/server.py":                     "# server\n",
		"tui_gateway/render.py":                     "# render\n",
		"tui_gateway/ws.py":                         "# ws\n",
		"RELEASE_v0.14.0.md":                        "# release\n",
	}
	for rel, body := range files {
		path := filepath.Join(hermes, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
}

func writeGatewayPlatformSourcePairs(t *testing.T, root string) {
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
      "hermes_file": "gateway/config.py",
      "gormes_targets": ["internal/gateway/platform_manifest.go"],
      "status": "covered",
      "contract": "Hermes Platform enum and configured platform registry are covered by the Go manifest.",
      "tests": ["go test ./internal/gateway -run TestHermesGatewayPlatformManifestCoversUpstream -count=1"],
      "progress_rows": ["Hermes gateway platform registry manifest"],
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "plugins/platforms/google_chat/plugin.yaml",
      "gormes_targets": ["internal/gateway/platform_manifest.go"],
      "status": "covered",
      "contract": "Bundled platform plugin manifest evidence is represented in the Go platform manifest.",
      "tests": ["go test ./internal/gateway -run TestHermesGatewayPlatformManifestCoversBundledPluginPlatforms -count=1"],
      "progress_rows": ["Bundled platform plugin manifest drift guard"],
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "tui_gateway/server.py",
      "gormes_targets": ["internal/tuigateway/gateway_mux.go"],
      "status": "covered",
      "contract": "TUI gateway websocket bridge is covered by native Gormes websocket fixture evidence.",
      "tests": ["go test ./internal/tuigateway -run TestGatewayMuxWebSocket_SessionSubmitAndFrameEvent -count=1"],
      "progress_rows": ["TUI websocket attach transport"],
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "gateway/platforms/helpers.py",
      "gormes_targets": ["internal/gateway/platform_manifest.go"],
      "status": "planned",
      "contract": "Helper/base adapter semantics remain row-backed evidence until each helper contract is fixture-covered.",
      "progress_rows": ["Hermes gateway helper module parity"],
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "gateway/platforms/generated_platforms.pyc",
      "status": "excluded",
      "contract": "Generated Python bytecode artifacts are excluded from runtime parity claims.",
      "last_checked_hermes_sha": "abc123"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write source-pairs: %v", err)
	}
}

func writeGatewayPlatformProgress(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir progress: %v", err)
	}
	p := &progress.Progress{
		Meta: progress.Meta{Version: "test"},
		Phases: map[string]progress.Phase{
			"1": {
				Name:        "Gateway platform fixture",
				Deliverable: "Fixture",
				Subphases: map[string]progress.Subphase{
					"1.A": {
						Name: "Gateway platforms",
						Items: []progress.Item{
							catalogProgressItem("Hermes gateway platform registry manifest", progress.StatusComplete, progress.ContractStatusValidated, "channels", "hermes-agent/gateway/config.py", "go test ./internal/gateway -run TestHermesGatewayPlatformManifestCoversUpstream -count=1"),
							catalogProgressItem("Bundled platform plugin manifest drift guard", progress.StatusComplete, progress.ContractStatusValidated, "channels", "hermes-agent/plugins/platforms/google_chat/plugin.yaml", "go test ./internal/gateway -run TestHermesGatewayPlatformManifestCoversBundledPluginPlatforms -count=1"),
							catalogProgressItem("TUI websocket attach transport", progress.StatusComplete, progress.ContractStatusValidated, "gateway", "hermes-agent/tui_gateway/server.py", "go test ./internal/tuigateway -run TestGatewayMuxWebSocket_SessionSubmitAndFrameEvent -count=1"),
							catalogProgressItem("Hermes gateway helper module parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "channels", "hermes-agent/gateway/platforms/helpers.py", "go test ./internal/gateway -run TestPlatformConnectedCheckersCoverManifest -count=1"),
							catalogProgressItem("Gateway API server surface parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "gateway", "hermes-agent/gateway/platforms/api_server.py", "go test ./internal/apiserver -run TestDashboardContract -count=1"),
							catalogProgressItem("Gateway platform docs evidence", progress.StatusPlanned, progress.ContractStatusFixtureReady, "channels", "hermes-agent/gateway/platforms/ADDING_A_PLATFORM.md", "go test ./internal/progress/repoctl -run TestHermesContractInventoryClassifiesGatewayPlatformSurface -count=1"),
							catalogProgressItem("Gateway runtime lifecycle parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "gateway", "hermes-agent/gateway/run.py", "go test ./internal/gateway -run TestGateway -count=1"),
							catalogProgressItem("Gateway builtin connector parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "channels", "hermes-agent/gateway/platforms/telegram.py", "go test ./internal/gateway -run TestHermesGatewayPlatformManifest -count=1"),
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
