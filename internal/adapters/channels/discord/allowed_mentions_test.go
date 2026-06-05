package discord

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestDiscordAllowedMentionsSafeDefaults(t *testing.T) {
	got := BuildAllowedMentionsFromEnv()
	assertAllowedMentionParse(t, got, discordgo.AllowedMentionTypeUsers)
	assertAllowedMentionNotParsed(t, got, discordgo.AllowedMentionTypeEveryone)
	assertAllowedMentionNotParsed(t, got, discordgo.AllowedMentionTypeRoles)
	if !got.RepliedUser {
		t.Fatal("RepliedUser = false, want true")
	}
}

func TestDiscordAllowedMentionsEnvOverridesOnlyOwnKnobs(t *testing.T) {
	t.Setenv("DISCORD_ALLOW_MENTION_EVERYONE", "true")
	t.Setenv("DISCORD_ALLOW_MENTION_USERS", "false")

	got := BuildAllowedMentionsFromEnv()
	assertAllowedMentionParse(t, got, discordgo.AllowedMentionTypeEveryone)
	assertAllowedMentionNotParsed(t, got, discordgo.AllowedMentionTypeRoles)
	assertAllowedMentionNotParsed(t, got, discordgo.AllowedMentionTypeUsers)
	if !got.RepliedUser {
		t.Fatal("RepliedUser = false, want unchanged true")
	}
}

func TestDiscordAllowedMentionsBooleanParsing(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "true", raw: "true", want: true},
		{name: "one", raw: "1", want: true},
		{name: "yes", raw: " yes ", want: true},
		{name: "on", raw: "ON", want: true},
		{name: "false", raw: "false", want: false},
		{name: "zero", raw: "0", want: false},
		{name: "off", raw: "off", want: false},
		{name: "garbage", raw: "garbage", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DISCORD_ALLOW_MENTION_EVERYONE", tc.raw)
			got := BuildAllowedMentionsFromEnv()
			if tc.want {
				assertAllowedMentionParse(t, got, discordgo.AllowedMentionTypeEveryone)
			} else {
				assertAllowedMentionNotParsed(t, got, discordgo.AllowedMentionTypeEveryone)
			}
		})
	}
}

func TestBotSendUsesSafeAllowedMentions(t *testing.T) {
	ms := newMockSession()
	b := New(Config{AllowedChannelID: "42"}, ms, nil)

	id, err := b.Send(context.Background(), "42", "hi @everyone <@123>")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("empty id")
	}

	sent := ms.complexSnapshot()
	if len(sent) != 1 {
		t.Fatalf("complex sends = %d, want 1", len(sent))
	}
	got := sent[0].Data.AllowedMentions
	assertAllowedMentionNotParsed(t, got, discordgo.AllowedMentionTypeEveryone)
	assertAllowedMentionNotParsed(t, got, discordgo.AllowedMentionTypeRoles)
	assertAllowedMentionParse(t, got, discordgo.AllowedMentionTypeUsers)
	if !got.RepliedUser {
		t.Fatal("RepliedUser = false, want true")
	}
}

func TestBotSendMediaUsesSafeAllowedMentions(t *testing.T) {
	mediaPath := writeDiscordTempFile(t, "image.png", []byte("png"))
	ms := newMockSession()
	b := New(Config{AllowedChannelID: "42"}, ms, nil)

	_, err := b.SendMedia(context.Background(), "42", "m1", gateway.OutboundMedia{
		Path: mediaPath,
		Kind: gateway.OutboundMediaKindImage,
	})
	if err != nil {
		t.Fatal(err)
	}
	sent := ms.complexSnapshot()
	if len(sent) != 1 {
		t.Fatalf("complex sends = %d, want 1", len(sent))
	}
	got := sent[0].Data.AllowedMentions
	assertAllowedMentionNotParsed(t, got, discordgo.AllowedMentionTypeEveryone)
	assertAllowedMentionNotParsed(t, got, discordgo.AllowedMentionTypeRoles)
}

func writeDiscordTempFile(t *testing.T, name string, body []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func assertAllowedMentionParse(t *testing.T, got *discordgo.MessageAllowedMentions, want discordgo.AllowedMentionType) {
	t.Helper()
	if got == nil {
		t.Fatal("AllowedMentions is nil")
	}
	for _, typ := range got.Parse {
		if typ == want {
			return
		}
	}
	t.Fatalf("AllowedMentions.Parse = %v, want %q", got.Parse, want)
}

func assertAllowedMentionNotParsed(t *testing.T, got *discordgo.MessageAllowedMentions, banned discordgo.AllowedMentionType) {
	t.Helper()
	if got == nil {
		t.Fatal("AllowedMentions is nil")
	}
	for _, typ := range got.Parse {
		if typ == banned {
			t.Fatalf("AllowedMentions.Parse = %v, did not want %q", got.Parse, banned)
		}
	}
}
