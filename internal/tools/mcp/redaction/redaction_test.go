package redaction

import (
	"strings"
	"testing"
)

func TestStringRedactsFullHyphenatedSecretTokens(t *testing.T) {
	input := "provider returned sk-live-secret and token=abc-def-ghi"
	got := String(input)

	for _, leaked := range []string{"sk-live", "-secret", "abc-def-ghi"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("String(%q) = %q, leaked token fragment %q", input, got, leaked)
		}
	}
	if strings.Count(got, Value) != 2 {
		t.Fatalf("String(%q) = %q, want two redaction markers", input, got)
	}
}

func TestFormatStringMapRedactsSecretKeyValues(t *testing.T) {
	got := FormatStringMap(map[string]string{
		"Authorization": "plain-secret-value",
		"X-Trace":       "trace-1",
	})

	if strings.Contains(got, "plain-secret-value") {
		t.Fatalf("FormatStringMap leaked secret-key value: %q", got)
	}
	if !strings.Contains(got, "Authorization="+Value) || !strings.Contains(got, "X-Trace=trace-1") {
		t.Fatalf("FormatStringMap = %q, want secret key redacted and non-secret preserved", got)
	}
}
