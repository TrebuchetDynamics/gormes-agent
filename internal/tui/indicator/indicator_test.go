package indicator

import "testing"

func TestFacadePreservesStyleAndSlashContracts(t *testing.T) {
	if got := NormalizeStyle(" Emoji "); got != StyleEmoji {
		t.Fatalf("NormalizeStyle() = %q, want %q", got, StyleEmoji)
	}
	if got := RenderFrame(StyleASCII, 1); got != "/" {
		t.Fatalf("RenderFrame() = %q, want /", got)
	}
	if got := Frames(StyleUnicode)[0]; got != "⠋" {
		t.Fatalf("Frames(StyleUnicode)[0] = %q, want braille spinner", got)
	}

	got := ParseSlash("/indicator unicode", StyleEmoji)
	if got.Style != StyleUnicode || got.Status != "indicator → unicode" || !got.Apply {
		t.Fatalf("ParseSlash() = %#v, want unicode apply result", got)
	}
	if got := ParseSlash("/indicator sparkle", StyleEmoji); got.Status != SlashUsage || got.Apply {
		t.Fatalf("ParseSlash(invalid) = %#v, want usage without apply", got)
	}
}
