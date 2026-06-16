package titlecmd

import (
	"strings"
	"testing"
)

func TestParseArgRejectsMalformedSlashCommand(t *testing.T) {
	for _, raw := range []string{"/title@bad-name Friendly Greeting", "//title Friendly Greeting", "／title@bot.name Friendly Greeting"} {
		if got, ok := ParseArg(raw); ok || got != "" {
			t.Fatalf("ParseArg(%q) = %q, %v; want malformed slash command rejected", raw, got, ok)
		}
	}
}

func TestParseArgHandlesSlashAndRawTitle(t *testing.T) {
	if got, ok := ParseArg("/title   Friendly Greeting  "); !ok || got != "Friendly Greeting" {
		t.Fatalf("ParseArg slash = %q, %v; want Friendly Greeting,true", got, ok)
	}
	if got, ok := ParseArg("/title@GormesBot   Friendly Greeting  "); !ok || got != "Friendly Greeting" {
		t.Fatalf("ParseArg bot mention = %q, %v; want Friendly Greeting,true", got, ok)
	}
	if got, ok := ParseArg("Friendly Greeting"); !ok || got != "Friendly Greeting" {
		t.Fatalf("ParseArg raw = %q, %v; want Friendly Greeting,true", got, ok)
	}
	if got, ok := ParseArg("title Friendly Greeting"); !ok || got != "title Friendly Greeting" {
		t.Fatalf("ParseArg raw matching word = %q, %v; want title Friendly Greeting,true", got, ok)
	}
	if got, ok := ParseArg("/title"); ok || got != "" {
		t.Fatalf("ParseArg empty = %q, %v; want empty,false", got, ok)
	}
}

func TestSanitizeCleansControlAndFormattingRunes(t *testing.T) {
	got, err := Sanitize(" Hello\x00\u009b\u200b   world\n ")
	if err != nil {
		t.Fatalf("Sanitize: %v", err)
	}
	if got != "Hello world" {
		t.Fatalf("Sanitize = %q, want Hello world", got)
	}
}

func TestSanitizeRedactsAuthorizationTitles(t *testing.T) {
	got, err := Sanitize("Release notes authorization=Bearer plain-secret-token")
	if err != nil {
		t.Fatalf("Sanitize: %v", err)
	}
	for _, forbidden := range []string{"plain-secret-token", "authorization", "Bearer", "bearer"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("Sanitize leaked authorization title field %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("Sanitize missing redaction marker in %q", got)
	}
}

func TestSanitizeRedactsSecretLikeTitles(t *testing.T) {
	got, err := Sanitize("Release notes api_key=plain-secret-token")
	if err != nil {
		t.Fatalf("Sanitize: %v", err)
	}
	for _, forbidden := range []string{"plain-secret-token", "api_key"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("Sanitize leaked secret-like title field %q in %q", forbidden, got)
		}
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("Sanitize missing redaction marker in %q", got)
	}
}

func TestSanitizeRejectsLongTitles(t *testing.T) {
	_, err := Sanitize(strings.Repeat("x", MaxSessionTitleRunes+1))
	if err == nil {
		t.Fatal("Sanitize long title err = nil, want error")
	}
}
