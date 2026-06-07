package environment

import (
	"reflect"
	"testing"
)

func TestEnvPassthroughRegistryRegisterCheckUnionAndClear(t *testing.T) {
	registry := NewEnvPassthroughRegistry([]string{"CONFIG_ONLY", "OPENAI_API_KEY"})

	blocked := registry.Register([]string{
		" TENOR_API_KEY ",
		"NOTION_TOKEN",
		"OPENAI_API_KEY",
		"ANTHROPIC_TOKEN",
		"",
	})
	wantBlocked := []string{"OPENAI_API_KEY", "ANTHROPIC_TOKEN"}
	if !reflect.DeepEqual(blocked, wantBlocked) {
		t.Fatalf("blocked = %#v, want %#v", blocked, wantBlocked)
	}

	for _, name := range []string{"TENOR_API_KEY", "NOTION_TOKEN", "CONFIG_ONLY"} {
		if !registry.IsAllowed(name) {
			t.Fatalf("%s should be allowed", name)
		}
	}
	for _, name := range []string{"OPENAI_API_KEY", "ANTHROPIC_TOKEN", "MISSING"} {
		if registry.IsAllowed(name) {
			t.Fatalf("%s should not be allowed", name)
		}
	}

	gotAll := registry.All()
	wantAll := []string{"CONFIG_ONLY", "NOTION_TOKEN", "TENOR_API_KEY"}
	if !reflect.DeepEqual(gotAll, wantAll) {
		t.Fatalf("All = %#v, want %#v", gotAll, wantAll)
	}

	registry.ClearRegistered()
	if registry.IsAllowed("TENOR_API_KEY") || registry.IsAllowed("NOTION_TOKEN") {
		t.Fatalf("registered vars should be cleared, all=%#v", registry.All())
	}
	if !registry.IsAllowed("CONFIG_ONLY") {
		t.Fatalf("config allowlist should remain after ClearRegistered")
	}
}

func TestEnvPassthroughDefaultRegistryIsSessionScoped(t *testing.T) {
	first := NewEnvPassthroughRegistry(nil)
	second := NewEnvPassthroughRegistry(nil)

	first.Register([]string{"TENOR_API_KEY"})
	if !first.IsAllowed("TENOR_API_KEY") {
		t.Fatal("first registry should allow registered var")
	}
	if second.IsAllowed("TENOR_API_KEY") {
		t.Fatal("second registry should not see first registry registrations")
	}
}

func TestEnvPassthroughRegistryRejectsAssignments(t *testing.T) {
	registry := NewEnvPassthroughRegistry([]string{"CONFIG_ONLY=value", " CONFIG_OK "})

	blocked := registry.Register([]string{"TENOR_API_KEY=secret", "VALID_NAME"})
	if len(blocked) != 0 {
		t.Fatalf("assignment-like names should be ignored, not reported as provider credential blocks: %#v", blocked)
	}

	for _, name := range []string{"CONFIG_ONLY=value", "TENOR_API_KEY=secret"} {
		if registry.IsAllowed(name) {
			t.Fatalf("assignment-like candidate %q should not be allowed", name)
		}
	}
	gotAll := registry.All()
	wantAll := []string{"CONFIG_OK", "VALID_NAME"}
	if !reflect.DeepEqual(gotAll, wantAll) {
		t.Fatalf("All = %#v, want %#v", gotAll, wantAll)
	}
}

func TestEnvPassthroughRegistryBlocksProviderCredentialsCaseInsensitively(t *testing.T) {
	registry := NewEnvPassthroughRegistry([]string{" openai_api_key ", "SAFE_KEY"})

	blocked := registry.Register([]string{"anthropic_token", "TENOR_API_KEY"})
	wantBlocked := []string{"anthropic_token"}
	if !reflect.DeepEqual(blocked, wantBlocked) {
		t.Fatalf("blocked = %#v, want %#v", blocked, wantBlocked)
	}

	for _, name := range []string{"openai_api_key", "OPENAI_API_KEY", "anthropic_token", "ANTHROPIC_TOKEN"} {
		if registry.IsAllowed(name) {
			t.Fatalf("provider credential %q should not be allowed", name)
		}
	}
	gotAll := registry.All()
	wantAll := []string{"SAFE_KEY", "TENOR_API_KEY"}
	if !reflect.DeepEqual(gotAll, wantAll) {
		t.Fatalf("All = %#v, want %#v", gotAll, wantAll)
	}
}
