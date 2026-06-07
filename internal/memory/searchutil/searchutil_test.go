package searchutil

import (
	"strings"
	"testing"
)

func TestSanitizeFTS5PatternStripsOperators(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Acme progress?", "Acme progress"},
		{"what's this* about?", "what s this about"},
		{"(hello) world", "hello world"},
		{"Acme-daily", "Acme daily"},
		{"name_with_underscore", "name_with_underscore"},
	}
	for _, c := range cases {
		if got := SanitizeFTS5Pattern(c.in); got != c.want {
			t.Fatalf("SanitizeFTS5Pattern(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if strings.Contains(SanitizeFTS5Pattern("tell me about Acme?"), "?") {
		t.Fatal("question mark survived sanitization")
	}
}

func TestSameChatKeyComparesSourceCaseInsensitively(t *testing.T) {
	if !SameChatKey(" Telegram :42", "telegram:42 ") {
		t.Fatal("SameChatKey returned false for same source/chat with source case and whitespace differences")
	}
	if SameChatKey("telegram:42", "telegram:0042") {
		t.Fatal("SameChatKey returned true for different chat IDs")
	}
}
