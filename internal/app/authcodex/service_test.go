package authcodex

import "testing"

func TestSanitizeCommandErrorRedactsOAuthSecrets(t *testing.T) {
	for _, input := range []string{
		"access_token=abc",
		"refresh_token=abc",
		"Authorization: Bearer abc",
		"client_secret=abc",
	} {
		if got := SanitizeCommandError(input); got != "[redacted]" {
			t.Fatalf("SanitizeCommandError(%q) = %q", input, got)
		}
	}
}

func TestSanitizeCommandErrorTruncatesLongErrors(t *testing.T) {
	input := ""
	for len(input) <= 200 {
		input += "x"
	}
	if got := SanitizeCommandError(input); len(got) != 160 {
		t.Fatalf("len(SanitizeCommandError(long)) = %d", len(got))
	}
}
