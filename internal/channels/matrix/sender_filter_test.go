package matrix

import "testing"

func TestMatrixSelfSenderFilter_CaseInsensitiveAndTrimmed(t *testing.T) {
	filter := NewSenderFilter("@Bot:Example.ORG")

	for _, sender := range []string{
		"@bot:example.org",
		"@BOT:EXAMPLE.ORG",
		"  @bot:example.org  ",
	} {
		if !filter.IsSelfSender(sender) {
			t.Fatalf("IsSelfSender(%q) = false, want true", sender)
		}
	}
	if filter.IsSelfSender("@alice:example.org") {
		t.Fatal("IsSelfSender(@alice:example.org) = true, want false")
	}
}

func TestMatrixSelfSenderFilter_UnresolvedUserIDDropsAll(t *testing.T) {
	filter := NewSenderFilter(" ")

	for _, sender := range []string{"@alice:example.org", ""} {
		if !filter.IsSelfSender(sender) {
			t.Fatalf("IsSelfSender(%q) = false, want true when own user_id is unresolved", sender)
		}
		if !filter.ShouldDropSender(sender) {
			t.Fatalf("ShouldDropSender(%q) = false, want true when own user_id is unresolved", sender)
		}
	}
}

func TestMatrixBridgeSenderFilter_RejectsLeadingUnderscoreLocalpart(t *testing.T) {
	filter := NewSenderFilter("@bot:example.org")

	for _, sender := range []string{
		"@_telegram_123:bridge.example",
		"@_discord_999:example",
		"@_slackbridge_puppet:example",
		"@:server.example",
		"",
		"   ",
	} {
		if !filter.IsSystemOrBridgeSender(sender) {
			t.Fatalf("IsSystemOrBridgeSender(%q) = false, want true", sender)
		}
		if !filter.ShouldDropSender(sender) {
			t.Fatalf("ShouldDropSender(%q) = false, want true", sender)
		}
	}
}

func TestMatrixBridgeSenderFilter_AllowsRegularUsers(t *testing.T) {
	filter := NewSenderFilter("@daemon:example.org")

	for _, sender := range []string{
		"@alice:example",
		"@alice_smith:example",
		"@daemon:nerdworks.casa",
	} {
		if filter.IsSystemOrBridgeSender(sender) {
			t.Fatalf("IsSystemOrBridgeSender(%q) = true, want false", sender)
		}
	}
	if filter.ShouldDropSender("@alice:example") {
		t.Fatal("ShouldDropSender(@alice:example) = true, want false")
	}
}
