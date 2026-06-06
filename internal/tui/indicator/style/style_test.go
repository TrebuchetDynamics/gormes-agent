package style

import (
	"strings"
	"testing"
)

func TestNormalizeMatchesHermesEnum(t *testing.T) {
	cases := []struct {
		raw  string
		want Style
	}{
		{raw: "", want: Kaomoji},
		{raw: " Emoji ", want: Emoji},
		{raw: "UNICODE", want: Unicode},
		{raw: "ascii", want: ASCII},
		{raw: "rainbow", want: Kaomoji},
	}
	for _, tc := range cases {
		if got := Normalize(tc.raw); got != tc.want {
			t.Fatalf("Normalize(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestRenderFrameMatchesHermesFrames(t *testing.T) {
	if got := RenderFrame(ASCII, 0); got != "|" {
		t.Fatalf("ascii frame 0 = %q, want |", got)
	}
	if got := RenderFrame(ASCII, 1); got != "/" {
		t.Fatalf("ascii frame 1 = %q, want /", got)
	}
	if got := RenderFrame(Emoji, 0); got != "⚕ " {
		t.Fatalf("emoji frame 0 = %q, want ⚕ space", got)
	}
	if got := RenderFrame(Unicode, 0); got != "⠋" {
		t.Fatalf("unicode frame 0 = %q, want braille spinner", got)
	}
	if got := RenderFrame(Kaomoji, 0); !strings.Contains(got, "◕") {
		t.Fatalf("kaomoji frame 0 = %q, want one of the face frames", got)
	}
}
