package gatewaytest

import (
	"strings"
	"testing"
)

// AssertContainsAll fails if got does not contain every expected substring.
func AssertContainsAll(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}
