package configreload

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
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

func TestSanitizeErrorBoundsOutputWithoutSplittingUTF8(t *testing.T) {
	long := strings.Repeat("x", 239) + "é" + strings.Repeat("y", 20)
	got := SanitizeError(errors.New(long))
	if len(got) > 240 {
		t.Fatalf("bounded UTF-8 length = %d, want <= 240", len(got))
	}
	if !utf8.ValidString(got) {
		t.Fatalf("bounded UTF-8 output is invalid: %q", got)
	}
}

func TestCloneMapsKeepHistoricalEmptyMapBehavior(t *testing.T) {
	if got := CloneStringMap(nil); got == nil || len(got) != 0 {
		t.Fatalf("CloneStringMap(nil) = %#v, want empty non-nil map", got)
	}
	if got := CloneBoolMap(nil); got == nil || len(got) != 0 {
		t.Fatalf("CloneBoolMap(nil) = %#v, want empty non-nil map", got)
	}
	if got := CloneNestedBoolMap(nil); got == nil || len(got) != 0 {
		t.Fatalf("CloneNestedBoolMap(nil) = %#v, want empty non-nil map", got)
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
