package testenv

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TrueColor forces Lip Gloss to render true-color escapes for deterministic
// color-sensitive TUI tests and restores the previous profile at cleanup.
func TrueColor(t testing.TB) {
	t.Helper()
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })
}
