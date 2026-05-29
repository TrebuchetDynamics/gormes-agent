package discord

import "testing"

func TestDiscordAdmissionAllowedChannelsWildcardAndIgnoredWins(t *testing.T) {
	policy := AdmissionPolicy{
		AllowedChannelIDs: []string{"*"},
		IgnoredChannelIDs: []string{"ignore-me"},
	}
	ctx := AdmissionContext{
		ChannelID: "ignore-me",
		GuildID:   "guild-1",
		AuthorID:  "user-1",
		Mentioned: true,
	}

	got := EvaluateAdmission(policy, ctx)
	if got.Allowed || got.Reason != DiscordAdmissionIgnoredChannel {
		t.Fatalf("EvaluateAdmission ignored channel = %+v, want denied %q", got, DiscordAdmissionIgnoredChannel)
	}

	ctx.ChannelID = "other"
	got = EvaluateAdmission(policy, ctx)
	if !got.Allowed {
		t.Fatalf("EvaluateAdmission wildcard allowed = %+v, want allowed", got)
	}
}

func TestDiscordAdmissionMentionFreeResponseDMAndKnownThread(t *testing.T) {
	policy := AdmissionPolicy{
		RequireMention:        true,
		FreeResponseChannels:  []string{"free-parent"},
		KnownThreadBypass:     true,
		ParticipatedThreadIDs: []string{"thread-known"},
	}

	cases := []struct {
		name string
		ctx  AdmissionContext
		want bool
	}{
		{name: "guild without mention denied", ctx: AdmissionContext{ChannelID: "general", GuildID: "guild-1", AuthorID: "u"}, want: false},
		{name: "guild mention allowed", ctx: AdmissionContext{ChannelID: "general", GuildID: "guild-1", AuthorID: "u", Mentioned: true}, want: true},
		{name: "dm bypasses mention", ctx: AdmissionContext{ChannelID: "dm-1", IsDM: true, AuthorID: "u"}, want: true},
		{name: "free parent bypasses mention", ctx: AdmissionContext{ChannelID: "thread-free", ParentChannelID: "free-parent", GuildID: "guild-1", IsThread: true, AuthorID: "u"}, want: true},
		{name: "known participated thread bypasses mention", ctx: AdmissionContext{ChannelID: "thread-known", ParentChannelID: "general", GuildID: "guild-1", IsThread: true, AuthorID: "u"}, want: true},
		{name: "unknown thread still requires mention", ctx: AdmissionContext{ChannelID: "thread-new", ParentChannelID: "general", GuildID: "guild-1", IsThread: true, AuthorID: "u"}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateAdmission(policy, tc.ctx)
			if got.Allowed != tc.want {
				t.Fatalf("EvaluateAdmission = %+v, want allowed=%t", got, tc.want)
			}
		})
	}
}

func TestDiscordBotFilter(t *testing.T) {
	cases := []struct {
		name      string
		allowBots string
		ctx       AdmissionContext
		want      bool
	}{
		{name: "own messages ignored", allowBots: "all", ctx: AdmissionContext{AuthorID: "bot-self", AuthorBot: true, SelfUserID: "bot-self"}, want: false},
		{name: "default none drops bots", allowBots: "", ctx: AdmissionContext{AuthorID: "bot-2", AuthorBot: true, SelfUserID: "bot-self"}, want: false},
		{name: "all accepts bots", allowBots: "ALL", ctx: AdmissionContext{AuthorID: "bot-2", AuthorBot: true, SelfUserID: "bot-self"}, want: true},
		{name: "mentions drops unmentioned bot", allowBots: "mentions", ctx: AdmissionContext{AuthorID: "bot-2", AuthorBot: true, SelfUserID: "bot-self"}, want: false},
		{name: "mentions accepts mentioned bot", allowBots: "mentions", ctx: AdmissionContext{AuthorID: "bot-2", AuthorBot: true, SelfUserID: "bot-self", Mentioned: true}, want: true},
		{name: "humans always pass bot filter", allowBots: "none", ctx: AdmissionContext{AuthorID: "user-1", SelfUserID: "bot-self"}, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateAdmission(AdmissionPolicy{AllowBots: tc.allowBots, RequireMention: false}, tc.ctx)
			if got.Allowed != tc.want {
				t.Fatalf("EvaluateAdmission = %+v, want allowed=%t", got, tc.want)
			}
		})
	}
}

func TestDiscordAdmissionAutoThreadSkipsReplyFreeAndNoThreadChannels(t *testing.T) {
	policy := AdmissionPolicy{
		RequireMention:       true,
		AutoThread:           true,
		FreeResponseChannels: []string{"free"},
		NoThreadChannelIDs:   []string{"direct"},
	}

	cases := []struct {
		name string
		ctx  AdmissionContext
		want bool
	}{
		{name: "mentioned guild channel auto threads", ctx: AdmissionContext{ChannelID: "general", GuildID: "guild-1", AuthorID: "u", Mentioned: true}, want: true},
		{name: "reply skips auto thread", ctx: AdmissionContext{ChannelID: "general", GuildID: "guild-1", AuthorID: "u", Mentioned: true, IsReply: true}, want: false},
		{name: "free response skips auto thread", ctx: AdmissionContext{ChannelID: "free", GuildID: "guild-1", AuthorID: "u"}, want: false},
		{name: "no-thread channel skips auto thread", ctx: AdmissionContext{ChannelID: "direct", GuildID: "guild-1", AuthorID: "u", Mentioned: true}, want: false},
		{name: "thread skips auto thread", ctx: AdmissionContext{ChannelID: "thread-1", ParentChannelID: "general", GuildID: "guild-1", AuthorID: "u", Mentioned: true, IsThread: true}, want: false},
		{name: "dm skips auto thread", ctx: AdmissionContext{ChannelID: "dm-1", AuthorID: "u", IsDM: true}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateAdmission(policy, tc.ctx)
			if got.ShouldAutoThread != tc.want {
				t.Fatalf("EvaluateAdmission ShouldAutoThread = %t in %+v, want %t", got.ShouldAutoThread, got, tc.want)
			}
		})
	}
}
