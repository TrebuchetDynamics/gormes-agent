package passthrough

import (
	"reflect"
	"testing"
)

func TestRegistryAllowsConfiguredAndRegisteredNamesButBlocksProviderCredentials(t *testing.T) {
	registry := NewRegistry([]string{"CONFIG_ONLY", "OPENAI_API_KEY"})

	blocked := registry.Register([]string{"TENOR_API_KEY", "ANTHROPIC_TOKEN"})
	if !reflect.DeepEqual(blocked, []string{"ANTHROPIC_TOKEN"}) {
		t.Fatalf("blocked = %#v, want ANTHROPIC_TOKEN", blocked)
	}
	if !registry.IsAllowed("CONFIG_ONLY") || !registry.IsAllowed("TENOR_API_KEY") {
		t.Fatalf("expected config and registered names to be allowed: %#v", registry.All())
	}
	if registry.IsAllowed("OPENAI_API_KEY") || registry.IsAllowed("ANTHROPIC_TOKEN") {
		t.Fatalf("provider credentials should be blocked")
	}
	wantAll := []string{"CONFIG_ONLY", "TENOR_API_KEY"}
	if got := registry.All(); !reflect.DeepEqual(got, wantAll) {
		t.Fatalf("All = %#v, want %#v", got, wantAll)
	}
}

func TestRegistrySnapshotsExplicitBlocklist(t *testing.T) {
	blocklist := map[string]struct{}{"INITIAL_SECRET": {}}
	registry := NewRegistryWithBlocklist(nil, blocklist)
	blocklist["LATER_SECRET"] = struct{}{}

	if blocked := registry.Register([]string{"INITIAL_SECRET", "LATER_SECRET"}); !reflect.DeepEqual(blocked, []string{"INITIAL_SECRET"}) {
		t.Fatalf("blocked = %#v, want only initial snapshot entry", blocked)
	}
	if !registry.IsAllowed("LATER_SECRET") {
		t.Fatalf("registry should not inherit blocklist mutations after construction")
	}
}
