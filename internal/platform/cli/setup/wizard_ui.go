// Package cli — wizard_ui.go provides screen-clearing and styling helpers
// for the interactive setup wizard. All escape sequences are TTY-gated: a
// piped or redirected stdout never sees ANSI codes, so transcript captures
// stay clean and tests that pipe stdin/stdout do not break.
//
// Honored environment:
//   - NO_COLOR=1     suppresses color/bold/dim escapes (passes through plain text).
//   - GORMES_NO_CLEAR_SCREEN=1 suppresses the screen-clear escape only;
//     color escapes still apply when the writer is a TTY. Useful for users
//     who want pretty colors but don't want their scrollback wiped.
package setup

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// ansi escape sequences kept centralized so tests can assert on a stable
// surface and so the helpers can no-op them in non-TTY contexts.
const (
	ansiReset      = "\x1b[0m"
	ansiBold       = "\x1b[1m"
	ansiDim        = "\x1b[2m"
	ansiCyan       = "\x1b[36m"
	ansiYellow     = "\x1b[33m"
	ansiGreen      = "\x1b[32m"
	ansiBrightCyan = "\x1b[96m"
	ansiClearHome  = "\x1b[2J\x1b[H"
)

// IsTerminalWriter reports whether w is connected to a TTY. Returns false
// for non-*os.File writers (e.g., bytes.Buffer in tests) and for files that
// fail the term.IsTerminal probe (pipes, redirects).
func IsTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// noColorEnv reports whether the operator has opted out of ANSI color via
// the de-facto NO_COLOR convention (https://no-color.org). Empty value
// counts as opt-in to NO_COLOR per spec.
func noColorEnv() bool {
	_, set := os.LookupEnv("NO_COLOR")
	return set
}

// noClearScreenEnv reports whether the operator has opted out of screen
// clearing via GORMES_NO_CLEAR_SCREEN. Useful in environments where wiping
// scrollback hides previously-shown error messages or instructions.
func noClearScreenEnv() bool {
	v := os.Getenv("GORMES_NO_CLEAR_SCREEN")
	return v == "1" || v == "true" || v == "yes"
}

// shouldStyle reports whether color escapes should be emitted to w.
func shouldStyle(w io.Writer) bool {
	if noColorEnv() {
		return false
	}
	return IsTerminalWriter(w)
}

// ClearScreen wipes the terminal and homes the cursor when w is a TTY and
// the operator has not opted out via GORMES_NO_CLEAR_SCREEN. No-op for
// piped/redirected output and for opt-outs.
func ClearScreen(w io.Writer) {
	if noClearScreenEnv() {
		return
	}
	if !IsTerminalWriter(w) {
		return
	}
	fmt.Fprint(w, ansiClearHome)
}

// Bold wraps s in bold escapes when w supports styling, else returns s
// unchanged. Pair with the writer that will print the result so the
// styling decision tracks the destination.
func Bold(w io.Writer, s string) string {
	if !shouldStyle(w) {
		return s
	}
	return ansiBold + s + ansiReset
}

// Dim wraps s in dim escapes when w supports styling.
func Dim(w io.Writer, s string) string {
	if !shouldStyle(w) {
		return s
	}
	return ansiDim + s + ansiReset
}

// Cyan wraps s in cyan escapes when w supports styling.
func Cyan(w io.Writer, s string) string {
	if !shouldStyle(w) {
		return s
	}
	return ansiCyan + s + ansiReset
}

// BrightCyan wraps s in bright-cyan escapes when w supports styling.
// Used for the active section header glyph.
func BrightCyan(w io.Writer, s string) string {
	if !shouldStyle(w) {
		return s
	}
	return ansiBrightCyan + s + ansiReset
}

// Yellow wraps s in yellow escapes when w supports styling.
func Yellow(w io.Writer, s string) string {
	if !shouldStyle(w) {
		return s
	}
	return ansiYellow + s + ansiReset
}

// Green wraps s in green escapes when w supports styling.
func Green(w io.Writer, s string) string {
	if !shouldStyle(w) {
		return s
	}
	return ansiGreen + s + ansiReset
}

// PrintHeader renders a Hermes-style framed header for a wizard section.
// Writes a leading blank line, the framed title, and a trailing blank line
// so the section visually separates from the previous one even when screen
// clearing is opted out of.
func PrintHeader(w io.Writer, title string) {
	const minWidth = 32
	const maxWidth = 60
	width := len(title) + 4
	if width < minWidth {
		width = minWidth
	}
	if width > maxWidth {
		width = maxWidth
	}
	bar := strings.Repeat("─", width-2)
	fmt.Fprintln(w)
	fmt.Fprintln(w, BrightCyan(w, "╭"+bar+"╮"))
	pad := width - 4 - len(title)
	if pad < 0 {
		pad = 0
	}
	fmt.Fprintf(w, "%s %s%s %s\n",
		BrightCyan(w, "│"),
		Bold(w, title),
		strings.Repeat(" ", pad),
		BrightCyan(w, "│"),
	)
	fmt.Fprintln(w, BrightCyan(w, "╰"+bar+"╯"))
	fmt.Fprintln(w)
}

// PrintSectionDivider draws a thin horizontal divider, used between
// in-section sub-blocks where a full ClearScreen would be too aggressive
// (e.g., reading a confirmation after showing a summary).
func PrintSectionDivider(w io.Writer) {
	fmt.Fprintln(w, Dim(w, strings.Repeat("─", 60)))
}
