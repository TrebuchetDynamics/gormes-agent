package contextwindow

import (
	"errors"
	"strings"
	"testing"
)

func TestModelContextResolverSanitizesProviderLookupError(t *testing.T) {
	resolver := NewModelContextResolver(ModelContextLookupFunc(func(ModelContextQuery) (int, bool, error) {
		return 0, false, errors.New("lookup failed\nAuthorization: Bearer sk-context-window-secret\n**Injected:** yes")
	}))

	got := resolver.Resolve(ModelContextQuery{Provider: "openai", Model: "gpt-test"})
	for _, forbidden := range []string{"sk-context-window-secret", "Bearer sk", "\n", "**Injected:**"} {
		if strings.Contains(got.ProviderLookupError, forbidden) {
			t.Fatalf("ProviderLookupError leaked %q in %q", forbidden, got.ProviderLookupError)
		}
	}
	if !strings.Contains(got.ProviderLookupError, "[redacted]") {
		t.Fatalf("ProviderLookupError = %q, want redaction marker", got.ProviderLookupError)
	}
}
