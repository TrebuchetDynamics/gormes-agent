package shared

import (
	"errors"
	"strings"
	"testing"
)

func TestSanitizeErrorNil(t *testing.T) {
	if got := SanitizeError(nil); got != "" {
		t.Fatalf("SanitizeError(nil) = %q, want empty", got)
	}
}

func TestSanitizeErrorPreservesNormalErrors(t *testing.T) {
	err := errors.New("file not found: /home/user/config.yaml")
	got := SanitizeError(err)
	if got != err.Error() {
		t.Fatalf("SanitizeError = %q, want %q", got, err.Error())
	}
}

func TestSanitizeErrorRedactsSKSecret(t *testing.T) {
	err := errors.New("open /home/user/.gormes/sk-test123: permission denied")
	got := SanitizeError(err)
	if strings.Contains(got, "sk-test123") {
		t.Fatalf("SanitizeError leaked secret: %q", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("SanitizeError should contain [REDACTED], got %q", got)
	}
}

func TestIsAlphanum(t *testing.T) {
	cases := []struct {
		b    byte
		want bool
	}{
		{'a', true}, {'z', true}, {'A', true}, {'Z', true},
		{'0', true}, {'9', true},
		{'-', false}, {'_', false}, {'/', false}, {' ', false},
		{0, false},
	}
	for _, tc := range cases {
		got := IsAlphanum(tc.b)
		if got != tc.want {
			t.Fatalf("IsAlphanum(%q) = %v, want %v", tc.b, got, tc.want)
		}
	}
}
