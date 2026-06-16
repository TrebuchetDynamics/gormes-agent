package commandline

import "testing"

func TestNameNormalizesSlashCommandTokens(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want string
	}{
		{raw: "/status", want: "status"},
		{raw: "／status", want: "status"},
		{raw: "/status@GormesBot", want: "status"},
		{raw: " /TTS@GormesBot on ", want: "tts"},
		{raw: "goal status", want: "goal"},
		{raw: "/status\u00a0now", want: "status"},
	} {
		if got := Name(tc.raw); got != tc.want {
			t.Fatalf("Name(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestNameRejectsHiddenFormattingInCommandToken(t *testing.T) {
	for _, raw := range []string{"/sta\u200btus", "／sta\u202etus", "sta\ufefftus"} {
		if got := Name(raw); got != "" {
			t.Fatalf("Name(%q) = %q, want empty for hidden formatting in command token", raw, got)
		}
	}
}

func TestNameRejectsBotMentionSuffixWithInvalidCharacters(t *testing.T) {
	for _, raw := range []string{"/status@bot-name", "／status@bot.name", "status@bot/name"} {
		if got := Name(raw); got != "" {
			t.Fatalf("Name(%q) = %q, want empty for invalid bot mention suffix", raw, got)
		}
	}
}

func TestNameRejectsMalformedBotMentionWithMultipleAtSigns(t *testing.T) {
	for _, raw := range []string{"/status@@GormesBot", "／status@@GormesBot", "status@@GormesBot"} {
		if got := Name(raw); got != "" {
			t.Fatalf("Name(%q) = %q, want empty for malformed bot mention", raw, got)
		}
	}
}

func TestNameRejectsEmptyBotMentionSuffix(t *testing.T) {
	for _, raw := range []string{"/status@", "／status@", "status@"} {
		if got := Name(raw); got != "" {
			t.Fatalf("Name(%q) = %q, want empty for malformed bot mention", raw, got)
		}
	}
}

func TestSplitSeparatesTokenAndTrimmedArgs(t *testing.T) {
	for _, tc := range []struct {
		raw       string
		wantToken string
		wantArgs  string
	}{
		{raw: "", wantToken: "", wantArgs: ""},
		{raw: " /spawn Research reviewer ", wantToken: "/spawn", wantArgs: "Research reviewer"},
		{raw: "/status", wantToken: "/status", wantArgs: ""},
		{raw: "/tts\tspeed fast", wantToken: "/tts", wantArgs: "speed fast"},
		{raw: "/status\u00a0now", wantToken: "/status", wantArgs: "now"},
	} {
		gotToken, gotArgs := Split(tc.raw)
		if gotToken != tc.wantToken || gotArgs != tc.wantArgs {
			t.Fatalf("Split(%q) = (%q, %q), want (%q, %q)", tc.raw, gotToken, gotArgs, tc.wantToken, tc.wantArgs)
		}
	}
}

func TestPayloadIfCommandRejectsEmptyCommandName(t *testing.T) {
	for _, tc := range []struct {
		raw     string
		command string
	}{
		{raw: "/", command: ""},
		{raw: "/ status", command: " "},
		{raw: "//status", command: "//"},
	} {
		if payload, ok := PayloadIfCommand(tc.raw, tc.command); ok {
			t.Fatalf("PayloadIfCommand(%q, %q) = (%q, true), want empty command rejected", tc.raw, tc.command, payload)
		}
	}
}

func TestPayloadIfCommandExposesCommandPayloadContract(t *testing.T) {
	for _, tc := range []struct {
		name        string
		raw         string
		commandName string
		wantPayload string
		wantOK      bool
	}{
		{name: "plain slash", raw: "/title Friendly Greeting", commandName: "title", wantPayload: "Friendly Greeting", wantOK: true},
		{name: "bot mention", raw: "/title@GormesBot Friendly Greeting", commandName: "/title", wantPayload: "Friendly Greeting", wantOK: true},
		{name: "unicode whitespace", raw: "/title\u00a0Friendly Greeting", commandName: "title", wantPayload: "Friendly Greeting", wantOK: true},
		{name: "bare command", raw: "/title", commandName: "title", wantPayload: "", wantOK: true},
		{name: "fullwidth slash", raw: "／title Friendly Greeting", commandName: "title", wantPayload: "Friendly Greeting", wantOK: true},
		{name: "non matching payload", raw: "Friendly Greeting", commandName: "title", wantPayload: "", wantOK: false},
		{name: "matching word without slash is raw payload", raw: "title Friendly Greeting", commandName: "title", wantPayload: "", wantOK: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotPayload, gotOK := PayloadIfCommand(tc.raw, tc.commandName)
			if gotPayload != tc.wantPayload || gotOK != tc.wantOK {
				t.Fatalf("PayloadIfCommand(%q, %q) = (%q, %v), want (%q, %v)", tc.raw, tc.commandName, gotPayload, gotOK, tc.wantPayload, tc.wantOK)
			}
		})
	}
}
