package credentials

import (
	"slices"
	"strings"
	"testing"
)

func resetCredentialSanitizerWarningsForTest() {
	ResetDefaultCredentialSanitizerWarnings()
}

func TestCredentialSanitizerStripsUnicodeLookalikes(t *testing.T) {
	var warnings []CredentialSanitizerWarning
	sanitizer := NewCredentialSanitizer(func(w CredentialSanitizerWarning) {
		warnings = append(warnings, w)
	})

	got := sanitizer.Sanitize("OPENROUTER_API_KEY", "sk-proj-abc\u028bdef")

	if got != "sk-proj-abcdef" {
		t.Fatalf("sanitized credential = %q, want %q", got, "sk-proj-abcdef")
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings len = %d, want 1: %#v", len(warnings), warnings)
	}
	warning := warnings[0]
	if warning.Code != CredentialSanitizerEvidenceNonASCIIStripped {
		t.Fatalf("warning code = %q, want %q", warning.Code, CredentialSanitizerEvidenceNonASCIIStripped)
	}
	if warning.Key != "OPENROUTER_API_KEY" || !warning.WarningOnce || !warning.Redacted {
		t.Fatalf("warning metadata = %#v", warning)
	}
	if !slices.Contains(warning.CodePoints, "U+028B") {
		t.Fatalf("warning code points = %#v, want U+028B", warning.CodePoints)
	}
	if strings.Contains(warning.RedactedPreview, "sk-proj") || strings.Contains(warning.Message, "abc") || strings.Contains(warning.Message, "\u028b") {
		t.Fatalf("warning leaked credential material: %#v", warning)
	}
}

func TestCredentialSanitizerWarnsOncePerKey(t *testing.T) {
	var warnings []CredentialSanitizerWarning
	sanitizer := NewCredentialSanitizer(func(w CredentialSanitizerWarning) {
		warnings = append(warnings, w)
	})

	first := sanitizer.Sanitize("GEMINI_API_KEY", "AIza\u028bbad")
	second := sanitizer.Sanitize("GEMINI_API_KEY", "AIza\u028bbad2")

	if first != "AIzabad" || second != "AIzabad2" {
		t.Fatalf("sanitized values = %q, %q; want %q, %q", first, second, "AIzabad", "AIzabad2")
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings len = %d, want one warning per key: %#v", len(warnings), warnings)
	}
}

func TestCredentialSanitizerLeavesNonCredentialUnicode(t *testing.T) {
	var warnings []CredentialSanitizerWarning
	sanitizer := NewCredentialSanitizer(func(w CredentialSanitizerWarning) {
		warnings = append(warnings, w)
	})

	got := sanitizer.Sanitize("MY_UNICODE_VAR", "hello caf\u00e9")

	if got != "hello caf\u00e9" {
		t.Fatalf("non-credential value = %q, want Unicode preserved", got)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
}

func TestCredentialSanitizerPreservesASCIIControlBytes(t *testing.T) {
	var warnings []CredentialSanitizerWarning
	sanitizer := NewCredentialSanitizer(func(w CredentialSanitizerWarning) {
		warnings = append(warnings, w)
	})

	got := sanitizer.Sanitize("ANTHROPIC_API_KEY", "sk-ant\x1bapi-key")

	if got != "sk-ant\x1bapi-key" {
		t.Fatalf("ASCII control credential = %q, want unchanged", got)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}
}
