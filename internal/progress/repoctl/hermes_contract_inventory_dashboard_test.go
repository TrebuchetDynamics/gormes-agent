package repoctl

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/progress/fidelity"
)

func TestHermesContractInventoryClassifiesWebDashboardSurface(t *testing.T) {
	root := t.TempDir()
	hermes := filepath.Join(root, "hermes-agent")
	writeDashboardHermesFixture(t, hermes)
	writeDashboardSourcePairs(t, root)
	writeDashboardProgress(t, root)

	result, err := WriteHermesContractInventory(HermesContractInventoryOptions{
		Root:             root,
		CurrentHermesSHA: "abc123",
	})
	if err != nil {
		t.Fatalf("WriteHermesContractInventory: %v", err)
	}

	families := result.Report.WebDashboardCatalog
	if got, want := dashboardFamilyIDs(families), []string{
		"cron_admin_jobs",
		"gateway_client_events",
		"i18n_catalog",
		"model_picker",
		"oauth_provider_panels",
		"plugin_pages_slots",
		"profiles_config",
		"sessions_page",
		"terminal_chat_pty",
		"theme_system",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("web dashboard family ids = %v, want %v", got, want)
	}

	gateway := dashboardFamilyByID(t, families, "gateway_client_events")
	if gateway.Status != fidelity.StatusCovered || gateway.Count == 0 {
		t.Fatalf("gateway client family = %+v, want covered with examples", gateway)
	}
	if !containsDashboardString(gateway.Examples, "web/src/lib/gatewayClient.ts") {
		t.Fatalf("gateway client examples = %v, want gatewayClient.ts", gateway.Examples)
	}
	if !containsDashboardProgressRow(gateway.ProgressRows, "TUI gateway dashboard client events") {
		t.Fatalf("gateway client progress rows = %+v, want TUI gateway row", gateway.ProgressRows)
	}
	if !containsDashboardSourcePair(gateway.SourcePairs, "web/src/lib/gatewayClient.ts") {
		t.Fatalf("gateway client source pairs = %+v, want gatewayClient.ts", gateway.SourcePairs)
	}

	plugins := dashboardFamilyByID(t, families, "plugin_pages_slots")
	if plugins.Status != fidelity.StatusCovered {
		t.Fatalf("plugin pages status = %q, want covered", plugins.Status)
	}
	for _, want := range []string{"web/src/pages/PluginsPage.tsx", "web/src/plugins/slots.ts", "web/src/plugins/registry.ts"} {
		if !containsDashboardString(plugins.Examples, want) {
			t.Fatalf("plugin page examples = %v, want %s", plugins.Examples, want)
		}
	}

	cron := dashboardFamilyByID(t, families, "cron_admin_jobs")
	if cron.Status != fidelity.StatusCovered {
		t.Fatalf("cron/admin status = %q, want covered", cron.Status)
	}
	if !containsDashboardString(cron.Examples, "web/src/pages/CronPage.tsx") {
		t.Fatalf("cron examples = %v, want CronPage.tsx", cron.Examples)
	}

	theme := dashboardFamilyByID(t, families, "theme_system")
	if theme.Status != fidelity.StatusOwnedDivergence {
		t.Fatalf("theme status = %q, want owned_divergence", theme.Status)
	}
	for _, want := range []string{"web/src/themes/context.tsx", "web/src/themes/presets.ts", "web/src/components/ThemeSwitcher.tsx"} {
		if !containsDashboardString(theme.Examples, want) {
			t.Fatalf("theme examples = %v, want %s", theme.Examples, want)
		}
	}

	oauth := dashboardFamilyByID(t, families, "oauth_provider_panels")
	if oauth.Status != fidelity.StatusPlanned {
		t.Fatalf("oauth/provider panel status = %q, want planned", oauth.Status)
	}
	for _, want := range []string{"web/src/components/OAuthProvidersCard.tsx", "web/src/components/OAuthLoginModal.tsx"} {
		if !containsDashboardString(oauth.Examples, want) {
			t.Fatalf("oauth examples = %v, want %s", oauth.Examples, want)
		}
	}

	chat := dashboardFamilyByID(t, families, "terminal_chat_pty")
	if chat.Status != fidelity.StatusPlanned {
		t.Fatalf("terminal chat status = %q, want planned", chat.Status)
	}
	for _, want := range []string{"web/src/pages/ChatPage.tsx", "web/src/components/ToolCall.tsx", "web/src/components/SlashPopover.tsx"} {
		if !containsDashboardString(chat.Examples, want) {
			t.Fatalf("chat examples = %v, want %s", chat.Examples, want)
		}
	}

	md, err := os.ReadFile(result.MarkdownPath)
	if err != nil {
		t.Fatalf("read markdown report: %v", err)
	}
	for _, want := range []string{
		"## Web Dashboard Classification",
		"| `gateway_client_events` | `covered` |",
		"`web/src/lib/gatewayClient.ts`",
		"| `plugin_pages_slots` | `covered` |",
		"| `oauth_provider_panels` | `planned` |",
		"| `theme_system` | `owned_divergence` |",
	} {
		if !strings.Contains(string(md), want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}
}

func writeDashboardHermesFixture(t *testing.T, hermes string) {
	t.Helper()
	files := map[string]string{
		"hermes_cli/main.py":                         "# fixture\n",
		"web/src/App.tsx":                            "export function App() { return null }\n",
		"web/src/lib/gatewayClient.ts":               "export class GatewayClient {}\n",
		"web/src/lib/api.ts":                         "export async function api() {}\n",
		"web/src/pages/ChatPage.tsx":                 "export function ChatPage() { return null }\n",
		"web/src/components/ChatSidebar.tsx":         "export function ChatSidebar() { return null }\n",
		"web/src/components/ToolCall.tsx":            "export function ToolCall() { return null }\n",
		"web/src/components/SlashPopover.tsx":        "export function SlashPopover() { return null }\n",
		"web/src/pages/SessionsPage.tsx":             "export function SessionsPage() { return null }\n",
		"web/src/pages/ProfilesPage.tsx":             "export function ProfilesPage() { return null }\n",
		"web/src/pages/ConfigPage.tsx":               "export function ConfigPage() { return null }\n",
		"web/src/pages/PluginsPage.tsx":              "export function PluginsPage() { return null }\n",
		"web/src/plugins/slots.ts":                   "export const KNOWN_SLOT_NAMES = []\n",
		"web/src/plugins/registry.ts":                "export function exposePluginSDK() {}\n",
		"web/src/plugins/usePlugins.ts":              "export function usePlugins() {}\n",
		"web/src/pages/CronPage.tsx":                 "export function CronPage() { return null }\n",
		"web/src/pages/ModelsPage.tsx":               "export function ModelsPage() { return null }\n",
		"web/src/components/ModelPickerDialog.tsx":   "export function ModelPickerDialog() { return null }\n",
		"web/src/components/ModelInfoCard.tsx":       "export function ModelInfoCard() { return null }\n",
		"web/src/components/OAuthProvidersCard.tsx":  "export function OAuthProvidersCard() { return null }\n",
		"web/src/components/OAuthLoginModal.tsx":     "export function OAuthLoginModal() { return null }\n",
		"web/src/i18n/context.tsx":                   "export function I18nProvider() {}\n",
		"web/src/i18n/en.ts":                         "export default {}\n",
		"web/src/i18n/es.ts":                         "export default {}\n",
		"web/src/themes/context.tsx":                 "export function ThemeProvider() {}\n",
		"web/src/themes/presets.ts":                  "export const presets = []\n",
		"web/src/components/ThemeSwitcher.tsx":       "export function ThemeSwitcher() { return null }\n",
		"web/src/components/LanguageSwitcher.tsx":    "export function LanguageSwitcher() { return null }\n",
		"web/src/components/SidebarStatusStrip.tsx":  "export function SidebarStatusStrip() { return null }\n",
		"web/src/components/DeleteConfirmDialog.tsx": "export function DeleteConfirmDialog() { return null }\n",
		"web/src/components/ui/input.tsx":            "export function Input() { return null }\n",
		"web/src/index.css":                          "body {}\n",
		"web/package.json":                           "{\"scripts\":{}}\n",
		"RELEASE_v0.14.0.md":                         "# release\n",
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

func writeDashboardSourcePairs(t *testing.T, root string) {
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
      "hermes_file": "web/src/lib/gatewayClient.ts",
      "gormes_targets": ["internal/tuigateway/gateway_mux.go"],
      "status": "covered",
      "contract": "Hermes dashboard GatewayClient event shapes are covered by Gormes TUI gateway fixtures.",
      "tests": ["go test ./internal/tuigateway -run TestGatewayMuxWebSocket_SessionSubmitAndFrameEvent -count=1"],
      "progress_rows": ["TUI gateway dashboard client events"],
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "web/src/pages/SessionsPage.tsx",
      "gormes_targets": ["internal/apiserver/dashboard_contract.go"],
      "status": "covered",
      "contract": "Hermes dashboard session page data contract is covered by native dashboard endpoint fixtures.",
      "tests": ["go test ./internal/apiserver -run TestDashboardContract_CoversNativeDashboardEndpoints -count=1"],
      "progress_rows": ["Dashboard sessions endpoint contract"],
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "web/src/plugins/slots.ts",
      "gormes_targets": ["internal/apiserver/plugins.go"],
      "status": "covered",
      "contract": "Hermes dashboard page-scoped plugin slot metadata is covered by Gormes plugin endpoint fixtures.",
      "tests": ["go test ./internal/apiserver -run TestDashboardPluginsEndpointPreservesPageScopedSlotMetadata -count=1"],
      "progress_rows": ["Dashboard page-scoped plugin slots"],
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "web/src/pages/CronPage.tsx",
      "gormes_targets": ["internal/apiserver/cron_admin.go"],
      "status": "covered",
      "contract": "Hermes dashboard cron admin page contract is covered by Gormes cron admin fixtures.",
      "tests": ["go test ./internal/apiserver -run TestAPIServerCronAdmin -count=1"],
      "progress_rows": ["Dashboard cron admin endpoints"],
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "web/src/i18n/context.tsx",
      "gormes_targets": ["internal/i18n/catalog.go"],
      "status": "covered",
      "contract": "Hermes dashboard i18n catalog is represented by Gormes static-message locale fixtures.",
      "tests": ["go test ./internal/i18n -count=1"],
      "progress_rows": ["Dashboard i18n locale catalog"],
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "web/src/themes/context.tsx",
      "gormes_targets": ["internal/apiserver/dashboard_extensions.go"],
      "status": "owned",
      "contract": "Gormes dashboard themes are owned extension metadata but must remain visible in strict-fidelity reports.",
      "tests": ["go test ./internal/apiserver -run TestDashboardExtensionStatusDistinguishesThemesPluginsAndBackendRoutes -count=1"],
      "progress_rows": ["Gormes dashboard theme extension status"],
      "last_checked_hermes_sha": "abc123"
    },
    {
      "hermes_file": "web/src/components/OAuthProvidersCard.tsx",
      "gormes_targets": ["internal/apiserver/capabilities.go"],
      "status": "planned",
      "contract": "Hermes dashboard OAuth/provider panels remain planned dashboard parity evidence.",
      "progress_rows": ["Dashboard OAuth provider panels"],
      "last_checked_hermes_sha": "abc123"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write source-pairs: %v", err)
	}
}

func writeDashboardProgress(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir progress: %v", err)
	}
	p := &progress.Progress{
		Meta: progress.Meta{Version: "test"},
		Phases: map[string]progress.Phase{
			"1": {
				Name:        "Dashboard fixture",
				Deliverable: "Fixture",
				Subphases: map[string]progress.Subphase{
					"1.A": {
						Name: "Web dashboard",
						Items: []progress.Item{
							dashboardProgressItem("TUI gateway dashboard client events", progress.StatusComplete, progress.ContractStatusValidated, "gateway", "hermes-agent/web/src/lib/gatewayClient.ts", "go test ./internal/tuigateway -run TestGatewayMuxWebSocket_SessionSubmitAndFrameEvent -count=1"),
							dashboardProgressItem("Dashboard sessions endpoint contract", progress.StatusComplete, progress.ContractStatusValidated, "gateway", "hermes-agent/web/src/pages/SessionsPage.tsx", "go test ./internal/apiserver -run TestDashboardContract_CoversNativeDashboardEndpoints -count=1"),
							dashboardProgressItem("Dashboard page-scoped plugin slots", progress.StatusComplete, progress.ContractStatusValidated, "plugins", "hermes-agent/web/src/plugins/slots.ts", "go test ./internal/apiserver -run TestDashboardPluginsEndpointPreservesPageScopedSlotMetadata -count=1"),
							dashboardProgressItem("Dashboard cron admin endpoints", progress.StatusComplete, progress.ContractStatusValidated, "gateway", "hermes-agent/web/src/pages/CronPage.tsx", "go test ./internal/apiserver -run TestAPIServerCronAdmin -count=1"),
							dashboardProgressItem("Dashboard i18n locale catalog", progress.StatusComplete, progress.ContractStatusValidated, "gateway", "hermes-agent/web/src/i18n/context.tsx", "go test ./internal/i18n -count=1"),
							dashboardProgressItem("Gormes dashboard theme extension status", progress.StatusComplete, progress.ContractStatusValidated, "gateway", "hermes-agent/web/src/themes/context.tsx", "go test ./internal/apiserver -run TestDashboardExtensionStatusDistinguishesThemesPluginsAndBackendRoutes -count=1"),
							dashboardProgressItem("Dashboard OAuth provider panels", progress.StatusPlanned, progress.ContractStatusFixtureReady, "providers", "hermes-agent/web/src/components/OAuthProvidersCard.tsx", "go test ./internal/apiserver -run TestAPIServerCapabilitiesEndpoint_AdvertisesHermesCompatibleContract -count=1"),
							dashboardProgressItem("Dashboard terminal chat PTY parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "gateway", "hermes-agent/web/src/pages/ChatPage.tsx", "go test ./internal/apiserver -run TestDashboardContract_CoversNativeDashboardEndpoints -count=1"),
							dashboardProgressItem("Dashboard model picker dialog parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "gateway", "hermes-agent/web/src/components/ModelPickerDialog.tsx", "go test ./internal/apiserver -run TestAPIServerCapabilitiesEndpoint_AdvertisesHermesCompatibleContract -count=1"),
							dashboardProgressItem("Dashboard profiles config parity", progress.StatusPlanned, progress.ContractStatusFixtureReady, "profiles", "hermes-agent/web/src/pages/ProfilesPage.tsx", "go test ./internal/apiserver -run TestDashboardContract_CoversNativeDashboardEndpoints -count=1"),
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

func dashboardProgressItem(name string, status progress.Status, contractStatus progress.ContractStatus, module, sourceRef, testCommand string) progress.Item {
	return progress.Item{
		Name:           name,
		Priority:       "P1",
		Status:         status,
		ContractStatus: contractStatus,
		Module:         module,
		Contract:       name + " contract.",
		SourceRefs:     []string{sourceRef},
		TestCommands:   []string{testCommand},
	}
}

func dashboardFamilyIDs(families []fidelity.CatalogFamilyReport) []string {
	ids := make([]string, 0, len(families))
	for _, family := range families {
		ids = append(ids, family.ID)
	}
	return ids
}

func dashboardFamilyByID(t *testing.T, families []fidelity.CatalogFamilyReport, id string) fidelity.CatalogFamilyReport {
	t.Helper()
	for _, family := range families {
		if family.ID == id {
			return family
		}
	}
	t.Fatalf("web dashboard family %q missing from %+v", id, families)
	return fidelity.CatalogFamilyReport{}
}

func containsDashboardString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsDashboardProgressRow(rows []fidelity.ProgressRowEvidence, want string) bool {
	for _, row := range rows {
		if row.Name == want {
			return true
		}
	}
	return false
}

func containsDashboardSourcePair(pairs []fidelity.SourcePairEvidence, want string) bool {
	for _, pair := range pairs {
		if pair.HermesFile == want {
			return true
		}
	}
	return false
}
