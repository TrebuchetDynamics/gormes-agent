package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Tests freeze the Go port of the deterministic helpers inside
// hermes_cli/dump.py (Phase 5.O). The interactive `run_dump` stdout printer
// stays intentionally unported — frontends wire their own copy/paste
// formatter around these pure primitives.

// -----------------------------------------------------------------------------
// RedactAPIKey — mirrors _redact
// -----------------------------------------------------------------------------

func TestRedactAPIKey_EmptyReturnsEmpty(t *testing.T) {
	if got := RedactAPIKey(""); got != "" {
		t.Fatalf("RedactAPIKey(\"\") = %q, want empty string", got)
	}
}

func TestRedactAPIKey_ShortValueReturnsMask(t *testing.T) {
	cases := []string{
		"a",
		"short",
		"elevench12", // 10 chars
		"elevench123", // 11 chars — still below the <12 boundary
	}
	for _, v := range cases {
		t.Run(v, func(t *testing.T) {
			if got := RedactAPIKey(v); got != "***" {
				t.Fatalf("RedactAPIKey(%q) = %q, want %q", v, got, "***")
			}
		})
	}
}

func TestRedactAPIKey_LongValueKeepsFirstAndLastFour(t *testing.T) {
	// 12 chars is the first length that takes the "keep ends" branch.
	got := RedactAPIKey("sk-0123456789abcdef")
	want := "sk-0..." + "cdef"
	if got != want {
		t.Fatalf("RedactAPIKey(long) = %q, want %q", got, want)
	}
}

func TestRedactAPIKey_ExactBoundaryAt12(t *testing.T) {
	// Exactly 12 chars → first branch where "keep ends" fires.
	got := RedactAPIKey("abcdefghijkl")
	want := "abcd...ijkl"
	if got != want {
		t.Fatalf("RedactAPIKey(12-char) = %q, want %q", got, want)
	}
}

// -----------------------------------------------------------------------------
// MemoryProvider — mirrors _memory_provider
// -----------------------------------------------------------------------------

func TestMemoryProvider_MissingSectionReturnsBuiltIn(t *testing.T) {
	got := MemoryProvider(map[string]any{})
	if got != "built-in" {
		t.Fatalf("MemoryProvider(empty) = %q, want %q", got, "built-in")
	}
}

func TestMemoryProvider_EmptyProviderReturnsBuiltIn(t *testing.T) {
	got := MemoryProvider(map[string]any{"memory": map[string]any{"provider": ""}})
	if got != "built-in" {
		t.Fatalf("MemoryProvider(empty provider) = %q, want %q", got, "built-in")
	}
}

func TestMemoryProvider_SetProviderReturnsName(t *testing.T) {
	got := MemoryProvider(map[string]any{"memory": map[string]any{"provider": "mem0"}})
	if got != "mem0" {
		t.Fatalf("MemoryProvider(mem0) = %q, want %q", got, "mem0")
	}
}

func TestMemoryProvider_NonMapMemoryFallsBack(t *testing.T) {
	// Upstream `config.get("memory", {})` would return a non-dict, then
	// `.get("provider", "")` would crash. The Go port defensively falls
	// back to "built-in" instead of panicking.
	got := MemoryProvider(map[string]any{"memory": "nonsense"})
	if got != "built-in" {
		t.Fatalf("MemoryProvider(non-map) = %q, want %q", got, "built-in")
	}
}

// -----------------------------------------------------------------------------
// GetModelAndProvider — mirrors _get_model_and_provider
// -----------------------------------------------------------------------------

func TestGetModelAndProvider_MissingReturnsFallbacks(t *testing.T) {
	model, provider := GetModelAndProvider(map[string]any{})
	if model != "(not set)" || provider != "(auto)" {
		t.Fatalf("GetModelAndProvider(empty) = (%q, %q), want (%q, %q)", model, provider, "(not set)", "(auto)")
	}
}

func TestGetModelAndProvider_StringEmpty(t *testing.T) {
	model, provider := GetModelAndProvider(map[string]any{"model": ""})
	if model != "(not set)" || provider != "(auto)" {
		t.Fatalf("GetModelAndProvider(\"\") = (%q, %q), want (%q, %q)", model, provider, "(not set)", "(auto)")
	}
}

func TestGetModelAndProvider_StringValue(t *testing.T) {
	model, provider := GetModelAndProvider(map[string]any{"model": "openai/gpt-4o"})
	if model != "openai/gpt-4o" || provider != "(auto)" {
		t.Fatalf("GetModelAndProvider(string) = (%q, %q), want (openai/gpt-4o, (auto))", model, provider)
	}
}

func TestGetModelAndProvider_DictDefaultPreferred(t *testing.T) {
	model, provider := GetModelAndProvider(map[string]any{
		"model": map[string]any{
			"default":  "gpt-4o-mini",
			"model":    "ignored",
			"name":     "also-ignored",
			"provider": "openai",
		},
	})
	if model != "gpt-4o-mini" || provider != "openai" {
		t.Fatalf("GetModelAndProvider(default key) = (%q, %q), want (gpt-4o-mini, openai)", model, provider)
	}
}

func TestGetModelAndProvider_DictModelFallback(t *testing.T) {
	// Upstream precedence: default -> model -> name. With "default" missing
	// but "model" set, that's the value chosen.
	model, provider := GetModelAndProvider(map[string]any{
		"model": map[string]any{
			"model":    "claude-3-5-sonnet",
			"name":     "fallback-name",
			"provider": "anthropic",
		},
	})
	if model != "claude-3-5-sonnet" || provider != "anthropic" {
		t.Fatalf("GetModelAndProvider(model key) = (%q, %q), want (claude-3-5-sonnet, anthropic)", model, provider)
	}
}

func TestGetModelAndProvider_DictNameFallback(t *testing.T) {
	// With both "default" and "model" missing/empty, "name" wins.
	model, provider := GetModelAndProvider(map[string]any{
		"model": map[string]any{
			"default":  "",
			"model":    "",
			"name":     "kimi-k2",
			"provider": "kimi",
		},
	})
	if model != "kimi-k2" || provider != "kimi" {
		t.Fatalf("GetModelAndProvider(name key) = (%q, %q), want (kimi-k2, kimi)", model, provider)
	}
}

func TestGetModelAndProvider_DictMissingAllKeysFallsBack(t *testing.T) {
	model, provider := GetModelAndProvider(map[string]any{
		"model": map[string]any{},
	})
	if model != "(not set)" || provider != "(auto)" {
		t.Fatalf("GetModelAndProvider(empty dict) = (%q, %q), want ((not set), (auto))", model, provider)
	}
}

func TestGetModelAndProvider_UnknownTypeFallsBack(t *testing.T) {
	// Upstream's else branch handles anything non-dict/non-string by
	// returning the pair of fallbacks — matches Go's default branch.
	model, provider := GetModelAndProvider(map[string]any{"model": 42})
	if model != "(not set)" || provider != "(auto)" {
		t.Fatalf("GetModelAndProvider(int) = (%q, %q), want ((not set), (auto))", model, provider)
	}
}

// -----------------------------------------------------------------------------
// CountMCPServers — mirrors _count_mcp_servers
// -----------------------------------------------------------------------------

func TestCountMCPServers_MissingMCPSectionReturnsZero(t *testing.T) {
	if got := CountMCPServers(map[string]any{}); got != 0 {
		t.Fatalf("CountMCPServers(empty) = %d, want 0", got)
	}
}

func TestCountMCPServers_MissingServersKeyReturnsZero(t *testing.T) {
	if got := CountMCPServers(map[string]any{"mcp": map[string]any{}}); got != 0 {
		t.Fatalf("CountMCPServers(no servers key) = %d, want 0", got)
	}
}

func TestCountMCPServers_CountsMapKeys(t *testing.T) {
	config := map[string]any{
		"mcp": map[string]any{
			"servers": map[string]any{
				"context7":   map[string]any{},
				"filesystem": map[string]any{},
				"grpc-gw":    map[string]any{},
			},
		},
	}
	if got := CountMCPServers(config); got != 3 {
		t.Fatalf("CountMCPServers(3 servers) = %d, want 3", got)
	}
}

func TestCountMCPServers_NonMapMCPSectionReturnsZero(t *testing.T) {
	if got := CountMCPServers(map[string]any{"mcp": "garbage"}); got != 0 {
		t.Fatalf("CountMCPServers(non-map mcp) = %d, want 0", got)
	}
}

// -----------------------------------------------------------------------------
// ConfiguredPlatforms — mirrors _configured_platforms
// -----------------------------------------------------------------------------

func TestPlatformEnvProbes_PinnedOrder(t *testing.T) {
	want := []PlatformEnvProbe{
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
	if !reflect.DeepEqual(PlatformEnvProbes, want) {
		t.Fatalf("PlatformEnvProbes drifted:\n got=%+v\nwant=%+v", PlatformEnvProbes, want)
	}
}

func TestConfiguredPlatforms_NoEnvReturnsEmpty(t *testing.T) {
	got := ConfiguredPlatforms(func(string) string { return "" })
	if got == nil {
		t.Fatalf("ConfiguredPlatforms(no env) = nil, want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("ConfiguredPlatforms(no env) = %v, want empty", got)
	}
}

func TestConfiguredPlatforms_DeterministicOrderMatchesProbeList(t *testing.T) {
	env := map[string]string{
		"TELEGRAM_BOT_TOKEN":  "t",
		"DISCORD_BOT_TOKEN":   "d",
		"SLACK_BOT_TOKEN":     "s",
		"WHATSAPP_ENABLED":    "yes",
		"EMAIL_ADDRESS":       "ops@example.com",
		"MATRIX_HOMESERVER_URL": "https://matrix.example.org",
		"QQ_APP_ID":           "qq",
	}
	got := ConfiguredPlatforms(func(k string) string { return env[k] })
	want := []string{"telegram", "discord", "slack", "whatsapp", "email", "matrix", "qqbot"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ConfiguredPlatforms order = %v, want %v", got, want)
	}
}

func TestConfiguredPlatforms_NilLookupFallsBackToProcessEnv(t *testing.T) {
	// A nil lookup falls back to os.Getenv so production callers can use the
	// zero value. Isolate the variables we touch by clearing + restoring.
	probe := PlatformEnvProbes[0]
	orig, had := os.LookupEnv(probe.EnvVar)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(probe.EnvVar, orig)
		} else {
			_ = os.Unsetenv(probe.EnvVar)
		}
	})
	for _, p := range PlatformEnvProbes {
		_ = os.Unsetenv(p.EnvVar)
	}
	_ = os.Setenv(probe.EnvVar, "set-via-env")

	got := ConfiguredPlatforms(nil)
	if len(got) == 0 || got[0] != probe.Name {
		t.Fatalf("ConfiguredPlatforms(nil) first = %v, want first element %q", got, probe.Name)
	}
}

// -----------------------------------------------------------------------------
// FormatCronSummary — mirrors _cron_summary
// -----------------------------------------------------------------------------

func TestFormatCronSummary_MissingFileReturnsZero(t *testing.T) {
	tmp := t.TempDir()
	got := FormatCronSummary(filepath.Join(tmp, "absent.json"))
	if got != "0" {
		t.Fatalf("FormatCronSummary(missing) = %q, want %q", got, "0")
	}
}

func TestFormatCronSummary_MalformedJSONReturnsErrorReading(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "jobs.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got := FormatCronSummary(path)
	if got != "(error reading)" {
		t.Fatalf("FormatCronSummary(malformed) = %q, want %q", got, "(error reading)")
	}
}

func TestFormatCronSummary_NoJobsArrayReturnsZeroOfZero(t *testing.T) {
	// Upstream: `jobs = data.get("jobs", [])` — missing key yields an empty
	// list, so `0 active / 0 total`.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "jobs.json")
	if err := os.WriteFile(path, []byte(`{"version": 2}`), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got := FormatCronSummary(path)
	want := "0 active / 0 total"
	if got != want {
		t.Fatalf("FormatCronSummary(no jobs key) = %q, want %q", got, want)
	}
}

func TestFormatCronSummary_EnabledDefaultsTrue(t *testing.T) {
	// Upstream: `j.get("enabled", True)` — a job with no `enabled` field
	// counts as active.
	tmp := t.TempDir()
	path := filepath.Join(tmp, "jobs.json")
	body := `{
        "jobs": [
            {"name": "a", "enabled": true},
            {"name": "b", "enabled": false},
            {"name": "c"}
        ]
    }`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got := FormatCronSummary(path)
	want := "2 active / 3 total"
	if got != want {
		t.Fatalf("FormatCronSummary = %q, want %q", got, want)
	}
}

// -----------------------------------------------------------------------------
// CountSkillsInHome — mirrors _count_skills
// -----------------------------------------------------------------------------

func TestCountSkillsInHome_NoSkillsDirReturnsZero(t *testing.T) {
	tmp := t.TempDir()
	if got := CountSkillsInHome(tmp); got != 0 {
		t.Fatalf("CountSkillsInHome(no skills dir) = %d, want 0", got)
	}
}

func TestCountSkillsInHome_CountsRecursiveSkillMarkers(t *testing.T) {
	tmp := t.TempDir()
	skills := filepath.Join(tmp, "skills")
	// Two skills at varying depths, plus a decoy file that must NOT count.
	for _, rel := range []string{
		"foo/SKILL.md",
		"bar/baz/SKILL.md",
		"decoy/README.md",
	} {
		full := filepath.Join(skills, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", full, err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	if got := CountSkillsInHome(tmp); got != 2 {
		t.Fatalf("CountSkillsInHome(2 skills) = %d, want 2", got)
	}
}

// -----------------------------------------------------------------------------
// ConfigOverrides — mirrors _config_overrides
// -----------------------------------------------------------------------------

func TestConfigOverrides_InterestingPathsPinnedOrder(t *testing.T) {
	want := [][2]string{
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
	if !reflect.DeepEqual(ConfigOverrideInterestingPaths(), want) {
		t.Fatalf("ConfigOverrideInterestingPaths drifted:\n got=%v\nwant=%v", ConfigOverrideInterestingPaths(), want)
	}
}

func TestConfigOverrides_EmptyReturnsEmptySlice(t *testing.T) {
	// Defaults == user → no overrides reported.
	def := map[string]any{
		"agent":    map[string]any{"max_turns": 50.0},
		"terminal": map[string]any{"backend": "local"},
		"toolsets": []any{"hermes-cli"},
	}
	user := map[string]any{
		"agent":    map[string]any{"max_turns": 50.0},
		"terminal": map[string]any{"backend": "local"},
		"toolsets": []any{"hermes-cli"},
	}
	got := ConfigOverrides(user, def)
	if len(got) != 0 {
		t.Fatalf("ConfigOverrides(equal) = %+v, want empty slice", got)
	}
}

func TestConfigOverrides_DifferentValueReported(t *testing.T) {
	def := map[string]any{
		"agent": map[string]any{"max_turns": 50.0},
	}
	user := map[string]any{
		"agent": map[string]any{"max_turns": 200.0},
	}
	got := ConfigOverrides(user, def)
	if len(got) != 1 {
		t.Fatalf("ConfigOverrides len = %d, want 1", len(got))
	}
	if got[0].Path != "agent.max_turns" {
		t.Fatalf("ConfigOverrides path = %q, want agent.max_turns", got[0].Path)
	}
	if !reflect.DeepEqual(got[0].Value, 200.0) {
		t.Fatalf("ConfigOverrides value = %v, want 200.0", got[0].Value)
	}
}

func TestConfigOverrides_NilUserValueIsIgnored(t *testing.T) {
	// Upstream: `if user_val is not None and user_val != default_val:` —
	// an explicit null at the user level is filtered out.
	def := map[string]any{
		"agent": map[string]any{"max_turns": 50.0},
	}
	user := map[string]any{
		"agent": map[string]any{"max_turns": nil},
	}
	got := ConfigOverrides(user, def)
	if len(got) != 0 {
		t.Fatalf("ConfigOverrides(nil user value) = %+v, want empty", got)
	}
}

func TestConfigOverrides_ToolsetsOverrideReported(t *testing.T) {
	def := map[string]any{
		"toolsets": []any{"hermes-cli"},
	}
	user := map[string]any{
		"toolsets": []any{"hermes-cli", "telegram"},
	}
	got := ConfigOverrides(user, def)
	if len(got) != 1 {
		t.Fatalf("ConfigOverrides len = %d, want 1", len(got))
	}
	if got[0].Path != "toolsets" {
		t.Fatalf("ConfigOverrides path = %q, want toolsets", got[0].Path)
	}
	want := []any{"hermes-cli", "telegram"}
	if !reflect.DeepEqual(got[0].Value, want) {
		t.Fatalf("ConfigOverrides value = %v, want %v", got[0].Value, want)
	}
}

func TestConfigOverrides_FallbackProvidersReportedWhenNonEmpty(t *testing.T) {
	def := map[string]any{}
	user := map[string]any{
		"fallback_providers": []any{"openrouter", "openai"},
	}
	got := ConfigOverrides(user, def)
	if len(got) != 1 {
		t.Fatalf("ConfigOverrides len = %d, want 1", len(got))
	}
	if got[0].Path != "fallback_providers" {
		t.Fatalf("ConfigOverrides path = %q, want fallback_providers", got[0].Path)
	}
	want := []any{"openrouter", "openai"}
	if !reflect.DeepEqual(got[0].Value, want) {
		t.Fatalf("ConfigOverrides value = %v, want %v", got[0].Value, want)
	}
}

func TestConfigOverrides_EmptyFallbackListSuppressed(t *testing.T) {
	// Upstream: `if fallbacks:` — an empty list is falsy, not reported.
	def := map[string]any{}
	user := map[string]any{
		"fallback_providers": []any{},
	}
	got := ConfigOverrides(user, def)
	if len(got) != 0 {
		t.Fatalf("ConfigOverrides(empty fallback list) = %+v, want empty", got)
	}
}

func TestConfigOverrides_NonMapSectionSkipped(t *testing.T) {
	// Upstream: `if not isinstance(default_section, dict) or not
	// isinstance(user_section, dict): continue` — a non-dict section value
	// is skipped without raising.
	def := map[string]any{
		"agent": map[string]any{"max_turns": 50.0},
	}
	user := map[string]any{
		"agent": "nonsense",
	}
	got := ConfigOverrides(user, def)
	if len(got) != 0 {
		t.Fatalf("ConfigOverrides(non-map user section) = %+v, want empty", got)
	}
}

func TestConfigOverrides_CombinedOrderMatchesUpstream(t *testing.T) {
	// Multiple overrides must surface in the upstream insertion order:
	// interesting-paths first, then toolsets, then fallback_providers.
	def := map[string]any{
		"agent":    map[string]any{"max_turns": 50.0, "gateway_timeout": 60.0},
		"display":  map[string]any{"streaming": true},
		"toolsets": []any{"hermes-cli"},
	}
	user := map[string]any{
		"agent":              map[string]any{"max_turns": 200.0, "gateway_timeout": 120.0},
		"display":            map[string]any{"streaming": false},
		"toolsets":           []any{"hermes-cli", "telegram"},
		"fallback_providers": []any{"openrouter"},
	}
	got := ConfigOverrides(user, def)
	wantPaths := []string{
		"agent.max_turns",
		"agent.gateway_timeout",
		"display.streaming",
		"toolsets",
		"fallback_providers",
	}
	if len(got) != len(wantPaths) {
		t.Fatalf("ConfigOverrides len = %d, want %d: %+v", len(got), len(wantPaths), got)
	}
	for i, w := range wantPaths {
		if got[i].Path != w {
			t.Fatalf("ConfigOverrides[%d].Path = %q, want %q", i, got[i].Path, w)
		}
	}
}

// -----------------------------------------------------------------------------
// Manifest promotion — dump.py must now be tracked as in_progress with the
// dump.go destination, not planned.
// -----------------------------------------------------------------------------

func TestHermesCLIManifest_DumpPromotedToInProgress(t *testing.T) {
	tree := loadManifestOrFatal(t)
	var found *HermesCLIFile
	for i := range tree.Files {
		if tree.Files[i].Source == "dump.py" {
			found = &tree.Files[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("manifest missing dump.py entry")
	}
	if found.Status != HermesCLIPortStatusInProgress {
		t.Fatalf("dump.py status = %q, want %q", found.Status, HermesCLIPortStatusInProgress)
	}
	wantGo := "internal/cli/dump.go"
	var hasGo bool
	for _, g := range found.Go {
		if g == wantGo {
			hasGo = true
			break
		}
	}
	if !hasGo {
		t.Fatalf("dump.py Go destinations = %v, want to include %q", found.Go, wantGo)
	}
}
