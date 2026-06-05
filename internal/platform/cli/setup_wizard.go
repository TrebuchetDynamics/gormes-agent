package cli

import (
	"io"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/setup"
)

type SetupWizard = setup.SetupWizard

func NewSetupWizard() *SetupWizard { return setup.NewSetupWizard() }

func IsTerminalWriter(w io.Writer) bool { return setup.IsTerminalWriter(w) }
func ClearScreen(w io.Writer)           { setup.ClearScreen(w) }
func Bold(w io.Writer, s string) string { return setup.Bold(w, s) }
func Dim(w io.Writer, s string) string  { return setup.Dim(w, s) }
func Cyan(w io.Writer, s string) string { return setup.Cyan(w, s) }
func BrightCyan(w io.Writer, s string) string {
	return setup.BrightCyan(w, s)
}
func Yellow(w io.Writer, s string) string { return setup.Yellow(w, s) }
func Green(w io.Writer, s string) string  { return setup.Green(w, s) }
func PrintHeader(w io.Writer, title string) {
	setup.PrintHeader(w, title)
}
