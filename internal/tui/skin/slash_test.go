package skin

import "testing"

func TestSlashName(t *testing.T) {
	if got := SlashName("/skin ocean blue"); got != "ocean blue" {
		t.Fatalf("SlashName = %q, want full arg tail", got)
	}
	if got := SlashName("/skin"); got != "" {
		t.Fatalf("SlashName without arg = %q, want empty", got)
	}
}

func TestDisplayName(t *testing.T) {
	if got := DisplayName("  "); got != "default" {
		t.Fatalf("DisplayName empty = %q, want default", got)
	}
	if got := DisplayName(" poseidon "); got != "poseidon" {
		t.Fatalf("DisplayName trimmed = %q", got)
	}
}
