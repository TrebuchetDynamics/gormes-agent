package logout

import "testing"

func TestConfiguredProviderAppliesNormalizerToEmptyDefault(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	got, err := ConfiguredProvider(func(provider string) string {
		if provider == "" {
			return "auto"
		}
		return provider
	})
	if err != nil {
		t.Fatalf("ConfiguredProvider: %v", err)
	}
	if got != "auto" {
		t.Fatalf("ConfiguredProvider = %q, want auto", got)
	}
}
