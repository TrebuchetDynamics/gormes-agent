package cli

import (
	"reflect"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPlatformToolsetConfigSaveStripsDefaultsAndPreservesMCP(t *testing.T) {
	cfg, parseReport := parseToolsetConfigYAML(t, `
platform_toolsets:
  cli:
    - web
    - terminal
    - hermes-cli
    - custom-mcp-server
    - github-tools
`)
	assertNoIssue(t, parseReport, PlatformToolsetIssueRestrictedToolset)

	report, err := cfg.SavePlatformSelection("cli", []string{"web", "browser"})
	if err != nil {
		t.Fatalf("SavePlatformSelection: %v", err)
	}

	want := []string{"browser", "custom-mcp-server", "github-tools", "web"}
	if got := cfg.PlatformToolsets["cli"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("saved cli toolsets = %v, want %v", got, want)
	}
	assertIssue(t, report, PlatformToolsetIssueIgnoredDefaultSuperset, "hermes-cli")
	assertNoIssue(t, report, PlatformToolsetIssueRestrictedToolset)
}

func TestPlatformToolsetConfigSaveStripsTelegramDefault(t *testing.T) {
	cfg, _ := parseToolsetConfigYAML(t, `
platform_toolsets:
  telegram:
    - browser
    - file
    - hermes-telegram
    - terminal
    - web
    - my-mcp-server
`)

	report, err := cfg.SavePlatformSelection("telegram", []string{"browser", "file", "terminal", "web"})
	if err != nil {
		t.Fatalf("SavePlatformSelection: %v", err)
	}

	want := []string{"browser", "file", "my-mcp-server", "terminal", "web"}
	if got := cfg.PlatformToolsets["telegram"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("saved telegram toolsets = %v, want %v", got, want)
	}
	assertIssue(t, report, PlatformToolsetIssueIgnoredDefaultSuperset, "hermes-telegram")
}

func TestPlatformToolsetConfigNormalizesNumericYAMLKeysAndEntries(t *testing.T) {
	cfg, parseReport := parseToolsetConfigYAML(t, `
platform_toolsets:
  cli:
    - web
    - 12306
mcp_servers:
  12306:
    url: https://example.test/mcp
  normal-server:
    url: https://example.test/normal
`)

	status, err := cfg.PlatformStatus("cli")
	if err != nil {
		t.Fatalf("PlatformStatus: %v", err)
	}

	for _, name := range status.RuntimeToolsets {
		if reflect.TypeOf(name).Kind() != reflect.String {
			t.Fatalf("runtime toolset name %v is not a string", name)
		}
	}
	if got := append([]string(nil), status.RuntimeToolsets...); !sort.StringsAreSorted(got) {
		t.Fatalf("runtime toolsets are not sorted: %v", got)
	}
	assertContains(t, status.RuntimeToolsets, "12306")
	assertContains(t, status.RuntimeToolsets, "normal-server")
	assertIssue(t, parseReport, PlatformToolsetIssueNumericEntryNormalized, "12306")
	assertIssue(t, parseReport, PlatformToolsetIssueNumericKeyNormalized, "12306")

	report, err := cfg.SavePlatformSelection("cli", []string{"web", "browser"})
	if err != nil {
		t.Fatalf("SavePlatformSelection: %v", err)
	}
	assertContains(t, cfg.PlatformToolsets["cli"], "12306")
	assertContains(t, report.PersistedToolsets, "12306")
}

func TestPlatformToolsetConfigNoMCPSentinelIsPlatformScopedAndNotRuntimeToolset(t *testing.T) {
	cfg, _ := parseToolsetConfigYAML(t, `
platform_toolsets:
  api_server:
    - web
    - terminal
    - no_mcp
mcp_servers:
  exa:
    url: https://mcp.exa.ai/mcp
`)

	apiStatus, err := cfg.PlatformStatus("api_server")
	if err != nil {
		t.Fatalf("PlatformStatus(api_server): %v", err)
	}
	assertNotContains(t, apiStatus.RuntimeToolsets, "exa")
	assertNotContains(t, apiStatus.RuntimeToolsets, "no_mcp")
	assertIssue(t, apiStatus, PlatformToolsetIssueNoMCPSuppression, "no_mcp")

	cliStatus, err := cfg.PlatformStatus("cli")
	if err != nil {
		t.Fatalf("PlatformStatus(cli): %v", err)
	}
	assertContains(t, cliStatus.RuntimeToolsets, "exa")
	assertNotContains(t, cliStatus.RuntimeToolsets, "no_mcp")

	saveReport, err := cfg.SavePlatformSelection("api_server", []string{"web", "browser"})
	if err != nil {
		t.Fatalf("SavePlatformSelection: %v", err)
	}
	assertNotContains(t, cfg.PlatformToolsets["api_server"], "no_mcp")
	assertIssue(t, saveReport, PlatformToolsetIssueNoMCPSuppression, "no_mcp")
}

func TestPlatformToolsetConfigRejectsDiscordToolsetsOutsideDiscord(t *testing.T) {
	cfg, _ := parseToolsetConfigYAML(t, `platform_toolsets: {}`)

	report, err := cfg.SavePlatformSelection("telegram", []string{"web", "terminal", "discord", "discord_admin"})
	if err != nil {
		t.Fatalf("SavePlatformSelection: %v", err)
	}

	want := []string{"terminal", "web"}
	if got := cfg.PlatformToolsets["telegram"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("saved telegram toolsets = %v, want %v", got, want)
	}
	assertIssue(t, report, PlatformToolsetIssueRestrictedToolset, "discord")
	assertIssue(t, report, PlatformToolsetIssueRestrictedToolset, "discord_admin")

	discordReport, err := cfg.SavePlatformSelection("discord", []string{"web", "discord"})
	if err != nil {
		t.Fatalf("SavePlatformSelection(discord): %v", err)
	}
	assertContains(t, cfg.PlatformToolsets["discord"], "discord")
	assertNoIssue(t, discordReport, PlatformToolsetIssueRestrictedToolset)
}

func TestPlatformToolsetStatusKeepsHomeAssistantWhenHASSTokenSet(t *testing.T) {
	t.Setenv("HASS_TOKEN", "fake-test-token")

	cfg, _ := parseToolsetConfigYAML(t, `platform_toolsets: {}`)

	for _, platform := range []string{"cron", "cli"} {
		status, err := cfg.PlatformStatus(platform)
		if err != nil {
			t.Fatalf("PlatformStatus(%s): %v", platform, err)
		}
		assertContains(t, status.RuntimeToolsets, "homeassistant")
		assertNotContains(t, status.RuntimeToolsets, "moa")
		assertNotContains(t, status.RuntimeToolsets, "rl")
	}

	explicitEmpty, _ := parseToolsetConfigYAML(t, `
platform_toolsets:
  cli: []
`)
	status, err := explicitEmpty.PlatformStatus("cli")
	if err != nil {
		t.Fatalf("PlatformStatus(cli): %v", err)
	}
	if len(status.RuntimeToolsets) != 0 {
		t.Fatalf("explicit empty cli runtime toolsets = %v, want empty", status.RuntimeToolsets)
	}
}

func TestPlatformToolsetStatusLeavesHomeAssistantOffWithoutHASSToken(t *testing.T) {
	t.Setenv("HASS_TOKEN", "")

	cfg, _ := parseToolsetConfigYAML(t, `platform_toolsets: {}`)

	for _, platform := range []string{"cron", "cli"} {
		status, err := cfg.PlatformStatus(platform)
		if err != nil {
			t.Fatalf("PlatformStatus(%s): %v", platform, err)
		}
		assertNotContains(t, status.RuntimeToolsets, "homeassistant")
		assertIssue(t, status, PlatformToolsetIssueHomeAssistantTokenMissing, "homeassistant")
	}
}

func parseToolsetConfigYAML(t *testing.T, body string) (PlatformToolsetConfig, PlatformToolsetReport) {
	t.Helper()
	var raw any
	if err := yaml.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}
	return ParsePlatformToolsetConfig(raw)
}

func assertIssue(t *testing.T, report PlatformToolsetReport, kind PlatformToolsetIssueKind, toolset string) {
	t.Helper()
	for _, issue := range report.Issues {
		if issue.Kind == kind && issue.Toolset == toolset {
			return
		}
	}
	t.Fatalf("missing issue kind=%s toolset=%s in %#v", kind, toolset, report.Issues)
}

func assertNoIssue(t *testing.T, report PlatformToolsetReport, kind PlatformToolsetIssueKind) {
	t.Helper()
	for _, issue := range report.Issues {
		if issue.Kind == kind {
			t.Fatalf("unexpected issue kind=%s in %#v", kind, report.Issues)
		}
	}
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("%v does not contain %q", values, want)
}

func assertNotContains(t *testing.T, values []string, unwanted string) {
	t.Helper()
	for _, value := range values {
		if value == unwanted {
			t.Fatalf("%v unexpectedly contains %q", values, unwanted)
		}
	}
}
