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
