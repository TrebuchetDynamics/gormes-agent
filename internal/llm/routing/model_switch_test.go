package routing

import (
	"strings"
	"testing"
)

func TestModelAliases_ContainsKnownAliases(t *testing.T) {
	cases := []struct {
		alias  string
		vendor string
		family string
	}{
		{"sonnet", "anthropic", "claude-sonnet"},
		{"opus", "anthropic", "claude-opus"},
		{"gpt5", "openai", "gpt-5"},
		{"gemini", "google", "gemini"},
		{"deepseek", "deepseek", "deepseek-chat"},
		{"llama", "meta-llama", "llama"},
		{"grok", "x-ai", "grok"},
		{"trinity", "arcee-ai", "trinity"},
	}
	for _, tc := range cases {
		got, ok := ModelAliases[tc.alias]
		if !ok {
			t.Fatalf("ModelAliases[%q] not found", tc.alias)
		}
		if got.Vendor != tc.vendor {
			t.Errorf("ModelAliases[%q].Vendor = %q, want %q", tc.alias, got.Vendor, tc.vendor)
		}
		if got.Family != tc.family {
			t.Errorf("ModelAliases[%q].Family = %q, want %q", tc.alias, got.Family, tc.family)
		}
	}
}

func TestModelAliases_Count(t *testing.T) {
	if len(ModelAliases) < 20 {
		t.Fatalf("len(ModelAliases) = %d, want >= 20 (upstream has 22)", len(ModelAliases))
	}
}

func TestParseModelFlags_BareAlias(t *testing.T) {
	model, provider, global := ParseModelFlags("sonnet")
	if model != "sonnet" || provider != "" || global {
		t.Errorf("ParseModelFlags(simple) = (%q, %q, %v), want (sonnet, \"\", false)", model, provider, global)
	}
}

func TestParseModelFlags_WithGlobal(t *testing.T) {
	model, provider, global := ParseModelFlags("sonnet --global")
	if model != "sonnet" || provider != "" || !global {
		t.Errorf("ParseModelFlags(--global) = (%q, %q, %v), want (sonnet, \"\", true)", model, provider, global)
	}
}

func TestParseModelFlags_WithProvider(t *testing.T) {
	model, provider, global := ParseModelFlags("sonnet --provider anthropic")
	if model != "sonnet" || provider != "anthropic" || global {
		t.Errorf("ParseModelFlags(--provider) = (%q, %q, %v), want (sonnet, anthropic, false)", model, provider, global)
	}
}

func TestParseModelFlags_WithProviderAndGlobal(t *testing.T) {
	model, provider, global := ParseModelFlags("sonnet --provider anthropic --global")
	if model != "sonnet" || provider != "anthropic" || !global {
		t.Errorf("ParseModelFlags(both) = (%q, %q, %v), want (sonnet, anthropic, true)", model, provider, global)
	}
}

func TestParseModelFlags_ProviderOnly(t *testing.T) {
	model, provider, global := ParseModelFlags("--provider my-ollama")
	if model != "" || provider != "my-ollama" || global {
		t.Errorf("ParseModelFlags(provider-only) = (%q, %q, %v), want (\"\", my-ollama, false)", model, provider, global)
	}
}

func TestParseModelFlags_Empty(t *testing.T) {
	model, provider, global := ParseModelFlags("")
	if model != "" || provider != "" || global {
		t.Errorf("ParseModelFlags(empty) = (%q, %q, %v), want (\"\", \"\", false)", model, provider, global)
	}
}

func TestParseModelFlags_UnicodeDashNormalization(t *testing.T) {
	// Em dash
	_, _, global := ParseModelFlags("sonnet \u2014global")
	if !global {
		t.Errorf("ParseModelFlags(em-dash) should detect --global, got global=%v", global)
	}
	// En dash
	_, _, global = ParseModelFlags("sonnet \u2013global")
	if !global {
		t.Errorf("ParseModelFlags(en-dash) should detect --global, got global=%v", global)
	}
}

func TestModelSortKey(t *testing.T) {
	order, prefix, rest := ModelSortKey("gpt-4", "gpt")
	if order != 0 || prefix != "gpt" || rest != "-4" {
		t.Errorf("ModelSortKey(gpt-4, gpt) = (%d, %q, %q), want (0, gpt, -4)", order, prefix, rest)
	}
}

func TestModelSortKey_NoPrefix(t *testing.T) {
	order, prefix, rest := ModelSortKey("claude-sonnet-4", "")
	if order != 0 || prefix != "" || rest != "claude-sonnet-4" {
		t.Errorf("ModelSortKey(no-prefix) = (%d, %q, %q), want (0, \"\", claude-sonnet-4)", order, prefix, rest)
	}
}

func TestSortedModelAliases_Deterministic(t *testing.T) {
	a := SortedModelAliases()
	b := SortedModelAliases()
	if len(a) != len(b) {
		t.Fatalf("SortedModelAliases length changed: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("SortedModelAliases non-deterministic at index %d: %q vs %q", i, a[i], b[i])
		}
	}
}

func TestSortedModelAliases_ContainsAll(t *testing.T) {
	sorted := SortedModelAliases()
	seen := make(map[string]bool, len(sorted))
	for _, k := range sorted {
		seen[k] = true
	}
	for alias := range ModelAliases {
		if !seen[alias] {
			t.Fatalf("SortedModelAliases missing alias %q", alias)
		}
	}
}

func TestModelSwitchResult_ZeroValue(t *testing.T) {
	var r ModelSwitchResult
	if r.Success {
		t.Error("ModelSwitchResult zero value should have Success=false")
	}
	if r.IsGlobal {
		t.Error("ModelSwitchResult zero value should have IsGlobal=false")
	}
}

func TestParseModelFlags_ProviderPositionIndependent(t *testing.T) {
	// Provider before model
	_, p1, _ := ParseModelFlags("--provider openai gpt4")
	// Provider after model
	m2, p2, _ := ParseModelFlags("gpt4 --provider openai")
	if p1 != "openai" || p2 != "openai" {
		t.Errorf("provider should be resolved regardless of position: (%q,%q)", m2, p2)
	}
}

func TestDirectAlias_HasFields(t *testing.T) {
	alias := DirectAlias{
		Model:    "qwen3.5:397b",
		Provider: "custom",
		BaseURL:  "https://ollama.com/v1",
	}
	if strings.TrimSpace(alias.Model) == "" {
		t.Error("DirectAlias.Model should be set")
	}
	if alias.Provider != "custom" {
		t.Errorf("DirectAlias.Provider = %q, want custom", alias.Provider)
	}
	if alias.BaseURL != "https://ollama.com/v1" {
		t.Errorf("DirectAlias.BaseURL = %q, want ollama URL", alias.BaseURL)
	}
}
