package redaction

import (
	"strings"
	"testing"
)

func TestRedactLiteralsLongestFirstSkipsEmptyValues(t *testing.T) {
	got := RedactLiterals("token=abcdef short=abc", []string{"", "abc", "abcdef"}, "[X]")
	want := "token=[X] short=[X]"
	if got != want {
		t.Fatalf("RedactLiterals() = %q, want %q", got, want)
	}
	if strings.Contains(got, "abcdef") {
		t.Fatalf("RedactLiterals leaked longer literal: %q", got)
	}
}

func TestRedactLiteralsDefaultMarker(t *testing.T) {
	got := RedactLiterals("api=secret", []string{"secret"}, "")
	want := "api=[redacted]"
	if got != want {
		t.Fatalf("RedactLiterals default marker = %q, want %q", got, want)
	}
}
