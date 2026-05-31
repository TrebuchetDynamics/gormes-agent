package identity

import "testing"

func TestProviderIDEqualIsCaseInsensitive(t *testing.T) {
	if !ProviderIDEqual("OPENAI", "openai") {
		t.Fatal("ProviderIDEqual should match provider IDs case-insensitively")
	}
	if ProviderIDEqual("openai", "anthropic") {
		t.Fatal("ProviderIDEqual matched different provider IDs")
	}
}
