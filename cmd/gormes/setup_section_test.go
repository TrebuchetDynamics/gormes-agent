package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

// `gormes setup <section>` must present the same boxed section chrome as the
// gormes full wizard and upstream `hermes setup <section>`
// (hermes_cli/setup.py@55c9f3206:3199), instead of the ad-hoc plain
// cli.PrintHeader text. The header is the shared 59-wide box reused from the
// shipped doctor RenderDoctorHeader pattern, titled `Gormes Setup — <Label>`
// with the gormes-owned section→label map.
func TestSetupSectionRendersBoxedHeader(t *testing.T) {
	for _, tc := range []struct {
		section string
		label   string
	}{
		{"model", "Model"},
		{"gateway", "Messaging Gateway"},
	} {
		fake := &setupCommandFakeSeams{isTTY: true}
		fake.runSetupGateway = func(*cobra.Command, bool) error { return nil }
		stdout, stderr, err := runSetupTestCommand(t, fake.seams(), tc.section)
		if err != nil {
			t.Fatalf("setup %s: Execute() error = %v stderr=%s", tc.section, err, stderr)
		}
		wantTitle := "Gormes Setup — " + tc.label
		if !strings.Contains(stdout, wantTitle) {
			t.Fatalf("setup %s: missing boxed header title %q:\n%s", tc.section, wantTitle, stdout)
		}
		if !strings.Contains(stdout, "┌") || !strings.Contains(stdout, "│") || !strings.Contains(stdout, "└") {
			t.Fatalf("setup %s: header is not the shared box chrome (┌ │ └):\n%s", tc.section, stdout)
		}
		for _, forbidden := range []string{"hermes setup", "~/.hermes", "⚕ Hermes"} {
			if strings.Contains(stdout, forbidden) {
				t.Fatalf("setup %s: leaked Hermes-owned wording %q:\n%s", tc.section, forbidden, stdout)
			}
		}
	}
}

// On clean success a section ends with a uniform `<Label> configuration
// complete!` footer (parity with hermes setup.py:3199
// `print_success(f"{label} configuration complete!")`).
func TestSetupSectionPrintsCompletionFooterOnSuccess(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "model")
	if err != nil {
		t.Fatalf("setup model: Execute() error = %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Model configuration complete!") {
		t.Fatalf("successful section must end with the uniform completion footer:\n%s", stdout)
	}
}

// Degraded: a section that cleanly cancels (returns nil but prints its own
// cancel line) must NOT get a false `… configuration complete!` footer.
func TestSetupSectionCancelSuppressesCompletionFooter(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: true}
	seams := fake.seams()
	seams.RunModelPicker = func(*cobra.Command) error {
		return fmt.Errorf("gormes model: %w", cli.ErrModelPickerCancelled)
	}
	stdout, stderr, err := runSetupTestCommand(t, seams, "model")
	if err != nil {
		t.Fatalf("setup model cancel: Execute() error = %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "Setup cancelled.") {
		t.Fatalf("cancel path should still print its clean cancel line:\n%s", stdout)
	}
	if strings.Contains(stdout, "configuration complete!") {
		t.Fatalf("cancelled section must NOT print a false completion footer:\n%s", stdout)
	}
}

// Degraded: a TTY-required section with no TTY returns errSetupRequiresTTY
// and must NOT print a completion footer.
func TestSetupSectionTTYRefusedSuppressesCompletionFooter(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, _, err := runSetupTestCommand(t, fake.seams(), "agent")
	if err == nil {
		t.Fatalf("setup agent without TTY should error (errSetupRequiresTTY), got nil\n%s", stdout)
	}
	if strings.Contains(stdout, "configuration complete!") {
		t.Fatalf("TTY-refused section must NOT print a completion footer:\n%s", stdout)
	}
}

// An unknown section keeps the existing unsupported behavior and prints no
// boxed setup header.
func TestSetupUnknownSectionNoBoxedHeader(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, _ := runSetupTestCommand(t, fake.seams(), "definitely-not-a-section")
	combined := stdout + stderr
	if strings.Contains(combined, "Gormes Setup — ") {
		t.Fatalf("unknown section must not render the boxed setup header:\n%s", combined)
	}
}
