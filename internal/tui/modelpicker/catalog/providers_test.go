package catalog

import (
	"strings"
	"testing"
)

func TestHermesProvidersUsesHermesProviderCatalog(t *testing.T) {
	providers := HermesProviders()
	if len(providers) != 37 {
		t.Fatalf("providers = %d, want 37", len(providers))
	}
	for _, want := range []struct {
		index int
		id    string
		label string
	}{
		{0, "nous", "Nous Portal (Nous Research subscription)"},
		{5, "openai-codex", "OpenAI Codex"},
		{36, "custom", "custom (direct API)"},
	} {
		got := providers[want.index]
		if got.ID != want.id || got.Label != want.label {
			t.Fatalf("provider[%d] = %#v, want id=%q label=%q", want.index, got, want.id, want.label)
		}
	}
	for _, provider := range providers {
		if strings.Contains(provider.Label, "(oauth_external)") || strings.Contains(provider.Label, "(api_key)") {
			t.Fatalf("provider leaked raw auth taxonomy: %#v", provider)
		}
	}
}
