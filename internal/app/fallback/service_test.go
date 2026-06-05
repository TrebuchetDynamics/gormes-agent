package fallback

import "testing"

func TestFormatFallbackEntryIncludesProviderAndBaseURL(t *testing.T) {
	got := formatFallbackEntry(FallbackEntry{Provider: "anthropic", Model: "claude", BaseURL: "https://api.example"})
	want := "claude  (via anthropic)  [https://api.example]"
	if got != want {
		t.Fatalf("formatFallbackEntry = %q, want %q", got, want)
	}
}
