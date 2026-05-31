package commandline

import "testing"

func TestNameNormalizesSlashCommandTokens(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want string
	}{
		{raw: "/status", want: "status"},
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
