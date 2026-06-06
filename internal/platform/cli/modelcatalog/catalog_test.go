package modelcatalog

import (
	"strings"
	"testing"
)

func TestHermesModelProviderCatalogExcludesSetupActions(t *testing.T) {
	entries := HermesModelProviderCatalog()
	if len(entries) != 37 {
		t.Fatalf("model provider entries = %d, want 37", len(entries))
	}
	for _, entry := range entries {
		switch entry.ID {
		case ProviderCatalogAuxConfig, ProviderCatalogLeaveUnchanged, "custom-endpoint":
			t.Fatalf("model provider catalog contains setup-only entry: %#v", entry)
		}
	}
}

func TestHermesProviderCatalogReturnsCopy(t *testing.T) {
	entries := HermesProviderCatalog()
	entries[0].ID = "mutated"

	if got := HermesProviderCatalog()[0].ID; got != "nous" {
		t.Fatalf("catalog first id = %q, want immutable copy", got)
	}
}

func TestHermesProviderCatalogExcludesRouterAsUpstreamProvider(t *testing.T) {
	entries := HermesProviderCatalog()
	leaveUnchanged := false
	for _, entry := range entries {
		id := strings.ToLower(strings.TrimSpace(entry.ID))
		label := strings.ToLower(strings.TrimSpace(entry.Label))
		if id == ProviderCatalogLeaveUnchanged {
			leaveUnchanged = true
		}
		if id == "router" || id == "gormes-router" || label == "gormes router" {
			t.Fatalf("provider catalog exposed Router as upstream provider: %+v", entry)
		}
	}
	if !leaveUnchanged {
		t.Fatal("provider catalog missing leave-unchanged entry")
	}
}
