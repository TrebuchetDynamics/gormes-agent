package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

// Deterministic helpers ported from hermes_cli/dump.py (Phase 5.O). The
// interactive `run_dump` stdout printer stays intentionally unported —
// frontends wire their own copy/paste formatter around these pure
// primitives, and only the shape of the detected state (redacted keys,
// configured platforms, non-default overrides) is stable across
// implementations.

// RedactAPIKey mirrors hermes_cli/dump.py::_redact: an empty value stays
// empty, values under 12 characters collapse to "***", and longer values
// keep the first and last four characters around an ellipsis.
func RedactAPIKey(value string) string {
	if value == "" {
		return ""
	}
	if len(value) < 12 {
		return "***"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

// MemoryProvider mirrors _memory_provider: return
// config["memory"]["provider"], falling back to "built-in" when missing,
// empty, or wrong-shape (defensive — upstream would raise on non-dict).
func MemoryProvider(config map[string]any) string {
	mem, ok := config["memory"].(map[string]any)
	if !ok {
		return "built-in"
	}
	provider, _ := mem["provider"].(string)
	if provider == "" {
		return "built-in"
	}
	return provider
}

// GetModelAndProvider mirrors _get_model_and_provider: extract the
// active model and provider from the "model" config entry, which may be
// a dict, a string, or missing entirely.
func GetModelAndProvider(config map[string]any) (model, provider string) {
	const fallbackModel = "(not set)"
	const fallbackProvider = "(auto)"
	raw, ok := config["model"]
	if !ok {
		return fallbackModel, fallbackProvider
	}
	switch v := raw.(type) {
	case map[string]any:
		for _, key := range []string{"default", "model", "name"} {
			if s, _ := v[key].(string); s != "" {
				model = s
				break
			}
		}
		if model == "" {
			model = fallbackModel
		}
		if p, _ := v["provider"].(string); p != "" {
			provider = p
		} else {
			provider = fallbackProvider
		}
	case string:
		if v == "" {
			model = fallbackModel
		} else {
			model = v
		}
		provider = fallbackProvider
	default:
		model = fallbackModel
		provider = fallbackProvider
	}
	return model, provider
}

// CountMCPServers mirrors _count_mcp_servers: the length of
// config["mcp"]["servers"], zero for any missing/wrong-shape branch.
func CountMCPServers(config map[string]any) int {
	mcp, ok := config["mcp"].(map[string]any)
	if !ok {
		return 0
	}
	servers, ok := mcp["servers"].(map[string]any)
	if !ok {
		return 0
	}
	return len(servers)
}

// PlatformEnvProbe is one (platform name, env var) pair consulted by
// ConfiguredPlatforms.
type PlatformEnvProbe struct {
	Name   string
	EnvVar string
}

// PlatformEnvProbes mirrors hermes_cli/dump.py::_configured_platforms's
// `checks` dict. The order matches upstream's dict-insertion order so the
// rendered dump stays stable across runs.
var PlatformEnvProbes = []PlatformEnvProbe{
	{"telegram", "TELEGRAM_BOT_TOKEN"},
	{"discord", "DISCORD_BOT_TOKEN"},
	{"slack", "SLACK_BOT_TOKEN"},
	{"whatsapp", "WHATSAPP_ENABLED"},
	{"signal", "SIGNAL_HTTP_URL"},
	{"email", "EMAIL_ADDRESS"},
	{"sms", "TWILIO_ACCOUNT_SID"},
	{"matrix", "MATRIX_HOMESERVER_URL"},
	{"mattermost", "MATTERMOST_URL"},
	{"homeassistant", "HASS_TOKEN"},
	{"dingtalk", "DINGTALK_CLIENT_ID"},
	{"feishu", "FEISHU_APP_ID"},
	{"wecom", "WECOM_BOT_ID"},
	{"wecom_callback", "WECOM_CALLBACK_CORP_ID"},
	{"weixin", "WEIXIN_ACCOUNT_ID"},
	{"qqbot", "QQ_APP_ID"},
}

// ConfiguredPlatforms mirrors _configured_platforms: walk
// PlatformEnvProbes in order and return the names whose env var is
// non-empty. A nil env lookup falls back to os.Getenv so production
// callers can omit the argument; tests pass a map-backed fake.
func ConfiguredPlatforms(env func(string) string) []string {
	if env == nil {
		env = os.Getenv
	}
	out := make([]string, 0, len(PlatformEnvProbes))
	for _, p := range PlatformEnvProbes {
		if env(p.EnvVar) != "" {
			out = append(out, p.Name)
		}
	}
	return out
}

// FormatCronSummary mirrors _cron_summary: a missing file reports "0",
// an unreadable/malformed file reports "(error reading)", and a valid
// decode reports "<active> active / <total> total" where a job without
// an explicit "enabled" field counts as active (`j.get("enabled", True)`).
func FormatCronSummary(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "0"
		}
		return "(error reading)"
	}
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return "(error reading)"
	}
	rawJobs, _ := data["jobs"].([]any)
	active := 0
	for _, rj := range rawJobs {
		job, ok := rj.(map[string]any)
		if !ok {
			continue
		}
		enabled, has := job["enabled"]
		if !has {
			active++
			continue
		}
		if flag, ok := enabled.(bool); ok && flag {
			active++
		}
	}
	return fmt.Sprintf("%d active / %d total", active, len(rawJobs))
}

// CountSkillsInHome mirrors _count_skills: recursively count SKILL.md
// files under `<hermesHome>/skills/`, returning zero if that directory
// does not exist or is unreadable.
func CountSkillsInHome(hermesHome string) int {
	skillsDir := filepath.Join(hermesHome, "skills")
	info, err := os.Stat(skillsDir)
	if err != nil || !info.IsDir() {
		return 0
	}
	count := 0
	_ = filepath.WalkDir(skillsDir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && d.Name() == "SKILL.md" {
			count++
		}
		return nil
	})
	return count
}

// ConfigOverride is one reported non-default user setting. Path mirrors
// the `section.key` dotpath upstream writes; Value carries the raw user
// value so callers can stringify with their own formatter.
type ConfigOverride struct {
	Path  string
	Value any
}

// configOverrideInterestingPaths mirrors the fixed (section, key) list in
// hermes_cli/dump.py::_config_overrides, preserving upstream order so the
// rendered dump stays deterministic.
var configOverrideInterestingPaths = [][2]string{
	{"agent", "max_turns"},
	{"agent", "gateway_timeout"},
	{"agent", "tool_use_enforcement"},
	{"terminal", "backend"},
	{"terminal", "docker_image"},
	{"terminal", "persistent_shell"},
	{"browser", "allow_private_urls"},
	{"compression", "enabled"},
	{"compression", "threshold"},
	{"display", "streaming"},
	{"display", "skin"},
	{"display", "show_reasoning"},
	{"smart_model_routing", "enabled"},
	{"privacy", "redact_pii"},
	{"tts", "provider"},
}

// ConfigOverrideInterestingPaths exposes the (section, key) list scanned
// by ConfigOverrides so external renderers can freeze the upstream order
// without reaching into the package's private state.
func ConfigOverrideInterestingPaths() [][2]string {
	out := make([][2]string, len(configOverrideInterestingPaths))
	copy(out, configOverrideInterestingPaths)
	return out
}

// ConfigOverrides mirrors _config_overrides: walk the interesting-paths
// list and report any (section, key) entries where the user config
// differs from the default, then append any toolsets or
// fallback_providers top-level overrides. The returned slice preserves
// upstream's dict-insertion order so the rendered summary is stable.
func ConfigOverrides(userConfig, defaultConfig map[string]any) []ConfigOverride {
	out := make([]ConfigOverride, 0)
	for _, p := range configOverrideInterestingPaths {
		section, key := p[0], p[1]
		defaultSection, okD := defaultConfig[section].(map[string]any)
		userSection, okU := userConfig[section].(map[string]any)
		if !okD || !okU {
			continue
		}
		userVal, has := userSection[key]
		if !has || userVal == nil {
			continue
		}
		defaultVal := defaultSection[key]
		if !reflect.DeepEqual(userVal, defaultVal) {
			out = append(out, ConfigOverride{Path: section + "." + key, Value: userVal})
		}
	}

	userToolsets, hasUserTS := userConfig["toolsets"]
	defaultToolsets, hasDefTS := defaultConfig["toolsets"]
	if !hasUserTS {
		userToolsets = []any{}
	}
	if !hasDefTS {
		defaultToolsets = []any{}
	}
	if !reflect.DeepEqual(userToolsets, defaultToolsets) {
		out = append(out, ConfigOverride{Path: "toolsets", Value: userToolsets})
	}

	if rawFallbacks, ok := userConfig["fallback_providers"]; ok {
		if list, ok := rawFallbacks.([]any); ok && len(list) > 0 {
			out = append(out, ConfigOverride{Path: "fallback_providers", Value: list})
		}
	}

	return out
}
