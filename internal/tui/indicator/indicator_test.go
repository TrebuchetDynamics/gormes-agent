package indicator

import (
	"strings"
	"testing"
)

func TestStyleMatchesHermesEnumAndFrames(t *testing.T) {
	cases := []struct {
		raw  string
		want Style
	}{
		{raw: "", want: StyleKaomoji},
		{raw: " Emoji ", want: StyleEmoji},
		{raw: "UNICODE", want: StyleUnicode},
		{raw: "ascii", want: StyleASCII},
		{raw: "rainbow", want: StyleKaomoji},
	}
	for _, tc := range cases {
		if got := NormalizeStyle(tc.raw); got != tc.want {
			t.Fatalf("NormalizeStyle(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}

	if got := RenderFrame(StyleASCII, 0); got != "|" {
		t.Fatalf("ascii frame 0 = %q, want |", got)
	}
	if got := RenderFrame(StyleASCII, 1); got != "/" {
		t.Fatalf("ascii frame 1 = %q, want /", got)
	}
	if got := RenderFrame(StyleEmoji, 0); got != "⚕ " {
		t.Fatalf("emoji frame 0 = %q, want ⚕ space", got)
	}
	if got := RenderFrame(StyleUnicode, 0); got != "⠋" {
		t.Fatalf("unicode frame 0 = %q, want braille spinner", got)
	}
	if got := RenderFrame(StyleKaomoji, 0); !strings.Contains(got, "◕") {
		t.Fatalf("kaomoji frame 0 = %q, want one of the face frames", got)
	}
}
