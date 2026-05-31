package model

import "testing"

func TestEntryGuildAndNormalizeGuildQueryShareDiscordGuildIdentity(t *testing.T) {
	entry := Entry{Guild: " Sages "}
	if got := EntryGuild(entry); got != "Sages" {
		t.Fatalf("EntryGuild = %q, want trimmed display guild", got)
	}
	if got := NormalizeGuildQuery(" SAGES "); got != "sages" {
		t.Fatalf("NormalizeGuildQuery = %q, want lowercase trimmed match key", got)
	}
}
