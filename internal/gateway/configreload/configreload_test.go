package configreload

import (
	"errors"
	"strings"
	"testing"
)

func TestSanitizeErrorRedactsSecretsAndBoundsOutput(t *testing.T) {
	if got := SanitizeError(errors.New("api_key=abc123 failed")); got != "[redacted]" {
		t.Fatalf("secret error = %q, want redacted", got)
	}
	long := strings.Repeat("x", 300)
	if got := SanitizeError(errors.New(long)); len(got) != 240 {
		t.Fatalf("bounded length = %d, want 240", len(got))
	}
	if got := SanitizeError(errors.New(" temporary config missing ")); got != "temporary config missing" {
		t.Fatalf("sanitized normal error = %q", got)
	}
}

func TestCloneNestedBoolMapIsDeepCopy(t *testing.T) {
	input := map[string]map[string]bool{"telegram": {"u1": true}}
	clone := CloneNestedBoolMap(input)
	input["telegram"]["u1"] = false
	input["telegram"]["u2"] = true
	if !clone["telegram"]["u1"] || clone["telegram"]["u2"] {
		t.Fatalf("clone changed with input mutation: %+v", clone)
	}
}
