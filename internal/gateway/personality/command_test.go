package personality

import (
	"strings"
	"testing"
)

func TestParseArg(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "empty", text: "", want: ""},
		{name: "bare", text: "/personality", want: ""},
		{name: "argument", text: "/personality pirate", want: "pirate"},
		{name: "bot mention", text: "/personality@GormesBot pirate", want: "pirate"},
		{name: "already payload", text: "pirate", want: "pirate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseArg(tt.text); got != tt.want {
				t.Fatalf("ParseArg(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestRenderListSortsAndTruncates(t *testing.T) {
	got := RenderList("pirate", map[string]string{
		"zen":    "calm",
		"pirate": "ahoy there matey",
	}, 6)
	wantLines := []string{
		"**Personalities:**",
		"Active: **pirate**",
		"  • `/personality pirate` — ahoy t…",
		"  • `/personality zen` — calm",
		"",
		"Usage: `/personality <name>` or `/personality none` to clear",
	}
	want := strings.Join(wantLines, "\n")
	if got != want {
		t.Fatalf("RenderList() =\n%s\nwant\n%s", got, want)
	}
}

func TestRenderUnknownIncludesSortedHint(t *testing.T) {
	got := RenderUnknown("wizard", map[string]string{"zen": "", "pirate": ""})
	want := "Unknown personality \"wizard\". Available: pirate, zen"
	if got != want {
		t.Fatalf("RenderUnknown() = %q, want %q", got, want)
	}
}
