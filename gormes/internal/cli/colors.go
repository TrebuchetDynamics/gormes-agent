package cli

import (
	"os"
	"strings"
)

// ANSI escape codes mirrored from hermes_cli/colors.py::Colors.
const (
	ColorReset   = "\x1b[0m"
	ColorBold    = "\x1b[1m"
	ColorDim     = "\x1b[2m"
	ColorRed     = "\x1b[31m"
	ColorGreen   = "\x1b[32m"
	ColorYellow  = "\x1b[33m"
	ColorBlue    = "\x1b[34m"
	ColorMagenta = "\x1b[35m"
	ColorCyan    = "\x1b[36m"
)

// ColorEnv captures the inputs that decide whether colored output is
// appropriate, mirroring hermes_cli.colors.should_use_color() without pulling
// in global process state — tests can inject any combination.
type ColorEnv struct {
	// NoColorSet is true when the NO_COLOR environment variable is set to any
	// value (https://no-color.org/). Upstream treats presence-with-any-value as
	// a disable signal.
	NoColorSet bool
	// Term is the value of the TERM environment variable. TERM=dumb disables
	// color per upstream.
	Term string
	// IsTerminal reports whether stdout is attached to a TTY.
	IsTerminal bool
}

// ShouldUseColor reports whether colored output should be emitted for the
// given environment snapshot. Mirrors the three-rule precedence from
// hermes_cli/colors.py::should_use_color: NO_COLOR disables, TERM=dumb
// disables, and a non-TTY disables; otherwise colored output is allowed.
func ShouldUseColor(env ColorEnv) bool {
	if env.NoColorSet {
		return false
	}
	if env.Term == "dumb" {
		return false
	}
	return env.IsTerminal
}

// DetectColorSupport builds a ColorEnv from the live process environment and
// stdout TTY state. Callers that need deterministic behavior (tests, CI runs)
// should construct ColorEnv directly and call ShouldUseColor.
func DetectColorSupport() ColorEnv {
	_, noColor := os.LookupEnv("NO_COLOR")
	return ColorEnv{
		NoColorSet: noColor,
		Term:       os.Getenv("TERM"),
		IsTerminal: stdoutIsTerminal(),
	}
}

// stdoutIsTerminal reports whether os.Stdout is attached to a character
// device. This avoids pulling in golang.org/x/term for a one-line check.
func stdoutIsTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// Colorize wraps text with the given ANSI codes when useColor is true and
// returns text unchanged when useColor is false. Mirrors
// hermes_cli/colors.py::color(): when coloring is applied, all codes are
// concatenated before the text and a single ColorReset is appended.
func Colorize(text string, useColor bool, codes ...string) string {
	if !useColor {
		return text
	}
	size := len(text) + len(ColorReset)
	for _, c := range codes {
		size += len(c)
	}
	var b strings.Builder
	b.Grow(size)
	for _, c := range codes {
		b.WriteString(c)
	}
	b.WriteString(text)
	b.WriteString(ColorReset)
	return b.String()
}
