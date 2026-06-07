package environment

import (
	"reflect"
	"sync"
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

func TestEnvPassthroughRegistryRejectsMalformedNames(t *testing.T) {
	registry := NewEnvPassthroughRegistry([]string{"CONFIG_ONLY=value", " CONFIG_OK ", "BAD NAME", "1BAD"})

	blocked := registry.Register([]string{"TENOR_API_KEY=secret", "VALID_NAME", "BAD\nNAME", "HAS-DASH"})
	if len(blocked) != 0 {
		t.Fatalf("malformed names should be ignored, not reported as provider credential blocks: %#v", blocked)
	}

	for _, name := range []string{"CONFIG_ONLY=value", "TENOR_API_KEY=secret", "BAD NAME", "BAD\nNAME", "1BAD", "HAS-DASH"} {
		if registry.IsAllowed(name) {
			t.Fatalf("malformed candidate %q should not be allowed", name)
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

func TestEnvPassthroughRegistryBlocksAWSCredentialTriplet(t *testing.T) {
	registry := NewEnvPassthroughRegistry([]string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"SAFE_KEY",
	})

	blocked := registry.Register([]string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
		"TENOR_API_KEY",
	})
	wantBlocked := []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN"}
	if !reflect.DeepEqual(blocked, wantBlocked) {
		t.Fatalf("blocked = %#v, want %#v", blocked, wantBlocked)
	}

	for _, name := range wantBlocked {
		if registry.IsAllowed(name) {
			t.Fatalf("AWS provider credential %q should not be allowed", name)
		}
	}
	gotAll := registry.All()
	wantAll := []string{"SAFE_KEY", "TENOR_API_KEY"}
	if !reflect.DeepEqual(gotAll, wantAll) {
		t.Fatalf("All = %#v, want %#v", gotAll, wantAll)
	}
}

func TestEnvPassthroughRegistrySnapshotsProviderCredentialBlocklist(t *testing.T) {
	registry := NewEnvPassthroughRegistry(nil)

	ProviderCredentialEnvBlocklist["TEST_ONLY_PROVIDER_SECRET"] = struct{}{}
	defer delete(ProviderCredentialEnvBlocklist, "TEST_ONLY_PROVIDER_SECRET")

	if blocked := registry.Register([]string{"TEST_ONLY_PROVIDER_SECRET"}); len(blocked) != 0 {
		t.Fatalf("existing registry should not inherit later global blocklist mutations: %#v", blocked)
	}
	if !registry.IsAllowed("TEST_ONLY_PROVIDER_SECRET") {
		t.Fatal("existing registry should keep the construction-time blocklist snapshot")
	}

	fresh := NewEnvPassthroughRegistry(nil)
	if blocked := fresh.Register([]string{"TEST_ONLY_PROVIDER_SECRET"}); !reflect.DeepEqual(blocked, []string{"TEST_ONLY_PROVIDER_SECRET"}) {
		t.Fatalf("fresh registry should see current blocklist, blocked = %#v", blocked)
	}
}

func TestEnvPassthroughRegistryCanonicalizesCustomBlocklistEntries(t *testing.T) {
	ProviderCredentialEnvBlocklist[" custom_provider_token "] = struct{}{}
	defer delete(ProviderCredentialEnvBlocklist, " custom_provider_token ")

	registry := NewEnvPassthroughRegistry([]string{"CUSTOM_PROVIDER_TOKEN"})
	if registry.IsAllowed("CUSTOM_PROVIDER_TOKEN") {
		t.Fatal("configured allowlist should not bypass custom provider credentials with noncanonical blocklist spelling")
	}

	blocked := registry.Register([]string{"custom_provider_token"})
	wantBlocked := []string{"custom_provider_token"}
	if !reflect.DeepEqual(blocked, wantBlocked) {
		t.Fatalf("blocked = %#v, want %#v", blocked, wantBlocked)
	}
}

func TestEnvPassthroughRegistryConcurrentSessionAccess(t *testing.T) {
	registry := NewEnvPassthroughRegistry([]string{"CONFIG_ONLY"})

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				registry.Register([]string{"TENOR_API_KEY", "OPENAI_API_KEY"})
				_ = registry.IsAllowed("TENOR_API_KEY")
				_ = registry.All()
			}
		}()
	}
	wg.Wait()

	if !registry.IsAllowed("CONFIG_ONLY") {
		t.Fatal("configured allowlist should remain visible after concurrent access")
	}
	if registry.IsAllowed("OPENAI_API_KEY") {
		t.Fatal("provider credentials should remain blocked after concurrent access")
	}
}
