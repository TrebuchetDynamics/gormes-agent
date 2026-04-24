package cli

import "testing"

// Tests freeze the Go port of hermes_cli/colors.py (Phase 5.O). The upstream
// Python module is tiny and self-contained: a fixed ANSI palette, a
// should_use_color() helper that mirrors https://no-color.org/ plus a
// TERM=dumb check plus a TTY check, and a color() wrapper that applies codes
// only when coloring is appropriate.

func TestShouldUseColor_Matrix(t *testing.T) {
	cases := []struct {
		name string
		env  ColorEnv
		want bool
	}{
		{"tty with normal TERM and no NO_COLOR → colored", ColorEnv{IsTerminal: true, Term: "xterm-256color"}, true},
		{"NO_COLOR set wins over everything", ColorEnv{NoColorSet: true, IsTerminal: true, Term: "xterm"}, false},
		{"NO_COLOR set even when stdout is not a tty", ColorEnv{NoColorSet: true, IsTerminal: false}, false},
		{"TERM=dumb disables even on a TTY", ColorEnv{Term: "dumb", IsTerminal: true}, false},
		{"non-tty disables", ColorEnv{IsTerminal: false, Term: "xterm"}, false},
		{"empty TERM on a TTY is allowed", ColorEnv{IsTerminal: true, Term: ""}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ShouldUseColor(tc.env); got != tc.want {
				t.Fatalf("ShouldUseColor(%+v) = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

func TestColorize_NoColorReturnsPlain(t *testing.T) {
	got := Colorize("hi", false, ColorRed, ColorBold)
	if got != "hi" {
		t.Fatalf("Colorize(useColor=false) = %q, want %q", got, "hi")
	}
}

func TestColorize_SingleCodeWrapsWithReset(t *testing.T) {
	got := Colorize("hi", true, ColorRed)
	want := ColorRed + "hi" + ColorReset
	if got != want {
		t.Fatalf("Colorize single code = %q, want %q", got, want)
	}
}

func TestColorize_MultipleCodesConcatenated(t *testing.T) {
	got := Colorize("x", true, ColorBold, ColorCyan)
	want := ColorBold + ColorCyan + "x" + ColorReset
	if got != want {
		t.Fatalf("Colorize multi codes = %q, want %q", got, want)
	}
}

func TestColorize_ZeroCodesWithUseColorStillAppendsReset(t *testing.T) {
	// Mirrors the upstream color(text) behavior with no codes: the text is
	// still bracketed by the (empty) prefix and Colors.RESET, which "".join(())
	// yields "" for. The observable difference is a trailing RESET.
	got := Colorize("x", true)
	want := "x" + ColorReset
	if got != want {
		t.Fatalf("Colorize zero codes useColor=true = %q, want %q", got, want)
	}
}

func TestColorPaletteMatchesUpstream(t *testing.T) {
	// Exact ANSI byte sequences from hermes_cli/colors.py::Colors.
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"Reset", ColorReset, "\x1b[0m"},
		{"Bold", ColorBold, "\x1b[1m"},
		{"Dim", ColorDim, "\x1b[2m"},
		{"Red", ColorRed, "\x1b[31m"},
		{"Green", ColorGreen, "\x1b[32m"},
		{"Yellow", ColorYellow, "\x1b[33m"},
		{"Blue", ColorBlue, "\x1b[34m"},
		{"Magenta", ColorMagenta, "\x1b[35m"},
		{"Cyan", ColorCyan, "\x1b[36m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}
