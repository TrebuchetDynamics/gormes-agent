package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const noMCPToolsetSentinel = "no_mcp"
const homeAssistantToolset = "homeassistant"
const hassTokenEnv = "HASS_TOKEN"

var platformDefaultToolsets = map[string]string{
	"api_server":     "hermes-api-server",
	"bluebubbles":    "hermes-bluebubbles",
	"cli":            "hermes-cli",
	"cron":           "hermes-cron",
	"dingtalk":       "hermes-dingtalk",
	"discord":        "hermes-discord",
	"email":          "hermes-email",
	"feishu":         "hermes-feishu",
	"homeassistant":  "hermes-homeassistant",
	"mattermost":     "hermes-mattermost",
	"matrix":         "hermes-matrix",
	"qqbot":          "hermes-qqbot",
	"signal":         "hermes-signal",
	"slack":          "hermes-slack",
	"telegram":       "hermes-telegram",
	"webhook":        "hermes-webhook",
	"wecom":          "hermes-wecom",
	"wecom_callback": "hermes-wecom-callback",
	"weixin":         "hermes-weixin",
	"whatsapp":       "hermes-whatsapp",
}

var defaultRuntimeToolsets = []string{
	"browser",
	"clarify",
	"code_execution",
	"cronjob",
	"delegation",
	"file",
	"memory",
	"messaging",
	"session_search",
	"skills",
	"terminal",
	"todo",
	"tts",
	"vision",
	"web",
}

// PlatformToolsetIssueKind identifies a config-status degraded-mode finding.
type PlatformToolsetIssueKind string

const (
	PlatformToolsetIssueIgnoredDefaultSuperset    PlatformToolsetIssueKind = "ignored_default_superset"
	PlatformToolsetIssueHomeAssistantTokenMissing PlatformToolsetIssueKind = "homeassistant_token_missing"
	PlatformToolsetIssueNumericKeyNormalized      PlatformToolsetIssueKind = "numeric_key_normalized"
	PlatformToolsetIssueNumericEntryNormalized    PlatformToolsetIssueKind = "numeric_entry_normalized"
	PlatformToolsetIssueNoMCPSuppression          PlatformToolsetIssueKind = "no_mcp_suppression"
	PlatformToolsetIssueRestrictedToolset         PlatformToolsetIssueKind = "restricted_toolset"
)

// PlatformToolsetIssue records a normalization or degraded-mode decision.
type PlatformToolsetIssue struct {
	Kind     PlatformToolsetIssueKind
	Platform string
	Toolset  string
	Detail   string
}

// PlatformToolsetReport is the pure helper status surface used by future CLI
// config/setup commands.
type PlatformToolsetReport struct {
	Platform          string
	RuntimeToolsets   []string
	PersistedToolsets []string
	Issues            []PlatformToolsetIssue
}

// PlatformToolsetConfig is a YAML-shaped read/write model for Hermes-compatible
// platform_toolsets and mcp_servers config sections.
type PlatformToolsetConfig struct {
	PlatformToolsets map[string][]string
	MCPServers       map[string]MCPServerConfig

	parseIssues []PlatformToolsetIssue
}

// MCPServerConfig captures only the fields needed for platform toolset
// resolution. Unknown MCP server names still round-trip through
// PlatformToolsets as passthrough entries.
type MCPServerConfig struct {
	Enabled bool
}

// ParsePlatformToolsetConfig normalizes the config sections touched by
// platform toolset setup without performing file I/O.
func ParsePlatformToolsetConfig(raw any) (PlatformToolsetConfig, PlatformToolsetReport) {
	cfg := PlatformToolsetConfig{
		PlatformToolsets: make(map[string][]string),
		MCPServers:       make(map[string]MCPServerConfig),
	}
	report := PlatformToolsetReport{}

	root := asMap(raw)
	if root == nil {
		cfg.parseIssues = report.Issues
		return cfg, report
	}

	if platformToolsets, ok := lookup(root, "platform_toolsets"); ok {
		for platformKey, entries := range asMap(platformToolsets) {
			platform, issue := normalizeName(platformKey)
			if issue != nil {
				issue.Kind = PlatformToolsetIssueNumericKeyNormalized
				issue.Platform = platform
				report.Issues = append(report.Issues, *issue)
			}
			names, issues := normalizeNameList(entries, platform)
			report.Issues = append(report.Issues, issues...)
			cfg.PlatformToolsets[platform] = names
		}
	}

	if mcpServers, ok := lookup(root, "mcp_servers"); ok {
		for serverKey, serverRaw := range asMap(mcpServers) {
			serverName, issue := normalizeName(serverKey)
			if issue != nil {
				issue.Kind = PlatformToolsetIssueNumericKeyNormalized
				issue.Toolset = serverName
				report.Issues = append(report.Issues, *issue)
			}
			serverMap := asMap(serverRaw)
			if serverMap == nil {
				continue
			}
			cfg.MCPServers[serverName] = MCPServerConfig{
				Enabled: parseEnabledFlag(mapValue(serverMap, "enabled"), true),
			}
		}
	}

	cfg.parseIssues = append([]PlatformToolsetIssue(nil), report.Issues...)
	return cfg, report
}

// PlatformStatus returns the runtime-visible toolset names for one platform.
func (cfg PlatformToolsetConfig) PlatformStatus(platform string) (PlatformToolsetReport, error) {
	manifest, err := tools.LoadUpstreamToolParityManifest()
	if err != nil {
		return PlatformToolsetReport{}, err
	}

	report := PlatformToolsetReport{
		Platform: platform,
		Issues:   append([]PlatformToolsetIssue(nil), cfg.parseIssues...),
	}
	selected, configured := cfg.PlatformToolsets[platform]
	if !configured {
		selected = append([]string(nil), defaultRuntimeToolsets...)
		if homeAssistantDefaultPlatform(platform) {
			if !hasHomeAssistantToken() {
				report.Issues = append(report.Issues, PlatformToolsetIssue{
					Kind:     PlatformToolsetIssueHomeAssistantTokenMissing,
					Platform: platform,
					Toolset:  homeAssistantToolset,
					Detail:   "HASS_TOKEN is not set; Home Assistant stays default-off",
				})
			} else {
				selected = append(selected, homeAssistantToolset)
			}
		}
	}

	runtime := make(map[string]struct{})
	suppressMCP := false
	for _, name := range selected {
		switch {
		case name == noMCPToolsetSentinel:
			suppressMCP = true
			report.Issues = append(report.Issues, PlatformToolsetIssue{
				Kind:     PlatformToolsetIssueNoMCPSuppression,
				Platform: platform,
				Toolset:  name,
				Detail:   "MCP servers suppressed for this platform",
			})
		case isPlatformDefaultSuperset(name):
			report.Issues = append(report.Issues, PlatformToolsetIssue{
				Kind:     PlatformToolsetIssueIgnoredDefaultSuperset,
				Platform: platform,
				Toolset:  name,
				Detail:   "platform default supersets are not runtime toolsets",
			})
		case !toolsetAllowedForPlatform(manifest, name, platform):
			report.Issues = append(report.Issues, PlatformToolsetIssue{
				Kind:     PlatformToolsetIssueRestrictedToolset,
				Platform: platform,
				Toolset:  name,
				Detail:   "toolset is not allowed for this platform",
			})
		default:
			runtime[name] = struct{}{}
		}
	}

	if !suppressMCP {
		for name, server := range cfg.MCPServers {
			if server.Enabled {
				runtime[name] = struct{}{}
			}
		}
	}

	report.RuntimeToolsets = sortedKeys(runtime)
	return report, nil
}

// SavePlatformSelection persists one platform's selected toolsets while
// preserving unknown MCP server names already present in that platform list.
func (cfg *PlatformToolsetConfig) SavePlatformSelection(platform string, selected []string) (PlatformToolsetReport, error) {
	if cfg.PlatformToolsets == nil {
		cfg.PlatformToolsets = make(map[string][]string)
	}
	manifest, err := tools.LoadUpstreamToolParityManifest()
	if err != nil {
		return PlatformToolsetReport{}, err
	}
	knownToolsets := knownManifestToolsets(manifest)

	report := PlatformToolsetReport{
		Platform: platform,
		Issues:   append([]PlatformToolsetIssue(nil), cfg.parseIssues...),
	}

	persisted := make(map[string]struct{})
	selectedNames, issues := normalizeNameList(selected, platform)
	report.Issues = append(report.Issues, issues...)
	for _, name := range selectedNames {
		if shouldPersistSelection(manifest, platform, name, &report) {
			persisted[name] = struct{}{}
		}
	}

	for _, existing := range cfg.PlatformToolsets[platform] {
		if existing == noMCPToolsetSentinel {
			report.Issues = append(report.Issues, PlatformToolsetIssue{
				Kind:     PlatformToolsetIssueNoMCPSuppression,
				Platform: platform,
				Toolset:  existing,
				Detail:   "saving from the picker clears no_mcp",
			})
			continue
		}
		if isPlatformDefaultSuperset(existing) {
			report.Issues = append(report.Issues, PlatformToolsetIssue{
				Kind:     PlatformToolsetIssueIgnoredDefaultSuperset,
				Platform: platform,
				Toolset:  existing,
				Detail:   "platform default supersets are stripped before persistence",
			})
			continue
		}
		if !toolsetAllowedForPlatform(manifest, existing, platform) {
			report.Issues = append(report.Issues, PlatformToolsetIssue{
				Kind:     PlatformToolsetIssueRestrictedToolset,
				Platform: platform,
				Toolset:  existing,
				Detail:   "toolset is not allowed for this platform",
			})
			continue
		}
		if _, known := knownToolsets[existing]; known {
			continue
		}
		persisted[existing] = struct{}{}
	}

	cfg.PlatformToolsets[platform] = sortedKeys(persisted)
	report.PersistedToolsets = append([]string(nil), cfg.PlatformToolsets[platform]...)
	return report, nil
}

func shouldPersistSelection(manifest tools.UpstreamToolParityManifest, platform string, name string, report *PlatformToolsetReport) bool {
	if name == noMCPToolsetSentinel {
		report.Issues = append(report.Issues, PlatformToolsetIssue{
			Kind:     PlatformToolsetIssueNoMCPSuppression,
			Platform: platform,
			Toolset:  name,
			Detail:   "no_mcp is a config sentinel, not a persisted runtime toolset",
		})
		return false
	}
	if isPlatformDefaultSuperset(name) {
		report.Issues = append(report.Issues, PlatformToolsetIssue{
			Kind:     PlatformToolsetIssueIgnoredDefaultSuperset,
			Platform: platform,
			Toolset:  name,
			Detail:   "platform default supersets are stripped before persistence",
		})
		return false
	}
	if !toolsetAllowedForPlatform(manifest, name, platform) {
		report.Issues = append(report.Issues, PlatformToolsetIssue{
			Kind:     PlatformToolsetIssueRestrictedToolset,
			Platform: platform,
			Toolset:  name,
			Detail:   "toolset is not allowed for this platform",
		})
		return false
	}
	return true
}

func toolsetAllowedForPlatform(manifest tools.UpstreamToolParityManifest, name string, platform string) bool {
	row, ok := manifest.Toolset(name)
	if !ok || len(row.PlatformRestrictions.AllowedPlatforms) == 0 {
		return true
	}
	for _, allowed := range row.PlatformRestrictions.AllowedPlatforms {
		if allowed == platform {
			return true
		}
	}
	return false
}

func homeAssistantDefaultPlatform(platform string) bool {
	switch platform {
	case "cron", "cli":
		return true
	default:
		return false
	}
}

func hasHomeAssistantToken() bool {
	return strings.TrimSpace(os.Getenv(hassTokenEnv)) != ""
}

func knownManifestToolsets(manifest tools.UpstreamToolParityManifest) map[string]struct{} {
	known := make(map[string]struct{}, len(manifest.Toolsets))
	for _, row := range manifest.Toolsets {
		known[row.Name] = struct{}{}
	}
	return known
}

func isPlatformDefaultSuperset(name string) bool {
	for _, platformDefault := range platformDefaultToolsets {
		if name == platformDefault {
			return true
		}
	}
	return false
}

func asMap(value any) map[any]any {
	switch typed := value.(type) {
	case map[any]any:
		return typed
	case map[string]any:
		out := make(map[any]any, len(typed))
		for key, value := range typed {
			out[key] = value
		}
		return out
	default:
		return nil
	}
}

func lookup(values map[any]any, name string) (any, bool) {
	for key, value := range values {
		if keyName, _ := normalizeName(key); keyName == name {
			return value, true
		}
	}
	return nil, false
}

func mapValue(values map[any]any, name string) any {
	value, _ := lookup(values, name)
	return value
}

func normalizeNameList(value any, platform string) ([]string, []PlatformToolsetIssue) {
	var raw []any
	switch typed := value.(type) {
	case []any:
		raw = typed
	case []string:
		raw = make([]any, len(typed))
		for i, item := range typed {
			raw[i] = item
		}
	default:
		return nil, nil
	}

	out := make([]string, 0, len(raw))
	var issues []PlatformToolsetIssue
	for _, item := range raw {
		name, issue := normalizeName(item)
		if name == "" {
			continue
		}
		if issue != nil {
			issue.Kind = PlatformToolsetIssueNumericEntryNormalized
			issue.Platform = platform
			issues = append(issues, *issue)
		}
		out = append(out, name)
	}
	return out, issues
}

func normalizeName(value any) (string, *PlatformToolsetIssue) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case fmt.Stringer:
		name := typed.String()
		return name, &PlatformToolsetIssue{Toolset: name, Detail: "non-string name normalized to string"}
	default:
		name := fmt.Sprint(value)
		return name, &PlatformToolsetIssue{Toolset: name, Detail: "non-string name normalized to string"}
	}
}

func parseEnabledFlag(value any, fallback bool) bool {
	switch typed := value.(type) {
	case nil:
		return fallback
	case bool:
		return typed
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		default:
			return fallback
		}
	default:
		return fallback
	}
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
