package gormescli

import (
	"bytes"
	"strings"
	"testing"
)

func TestSetupSectionRendersBoxedHeader(t *testing.T) {
	for _, tc := range []struct {
		label string
	}{
		{label: "Model"},
		{label: "Messaging Gateway"},
	} {
		stdout := RenderSetupSectionHeader(tc.label)
		wantTitle := "Gormes Setup — " + tc.label
		if !strings.Contains(stdout, wantTitle) {
			t.Fatalf("missing boxed header title %q:\n%s", wantTitle, stdout)
		}
		if !strings.Contains(stdout, "┌") || !strings.Contains(stdout, "│") || !strings.Contains(stdout, "└") {
			t.Fatalf("header is not the shared box chrome (┌ │ └):\n%s", stdout)
		}
		for _, forbidden := range []string{"hermes setup", "~/.hermes", "⚕ Hermes"} {
			if strings.Contains(stdout, forbidden) {
				t.Fatalf("header leaked Hermes-owned wording %q:\n%s", forbidden, stdout)
			}
		}
	}
}

func TestSetupSectionSuccessFooterSuppression(t *testing.T) {
	if SetupSectionSuppressSuccessFooter("model", "Model saved\n") {
		t.Fatal("successful model output should not suppress the uniform footer")
	}
	for _, output := range []string{
		"Setup cancelled.\n",
		"setup canceled; no files were written.\n",
		"Setup canceled.\n",
	} {
		if !SetupSectionSuppressSuccessFooter("model", output) {
			t.Fatalf("cancel output should suppress success footer: %q", output)
		}
	}
	providerReceipt := "\nConnection\nAuthentication\nNext steps\n"
	if !SetupSectionSuppressSuccessFooter("provider", providerReceipt) {
		t.Fatal("provider receipt output should suppress duplicate success footer")
	}
	if SetupSectionSuppressSuccessFooter("model", providerReceipt) {
		t.Fatal("provider receipt suppression must not apply to unrelated sections")
	}
}

func TestSetupSectionUnsupportedGuidance(t *testing.T) {
	var out bytes.Buffer
	WriteSetupSectionUnsupported(&out, "browser", "provider|model|gateway")
	artifact := out.String() + SetupSectionUnsupportedError("browser").Error()
	for _, want := range []string{
		"setup_section_unsupported: section=browser available=provider|model|gateway",
		"Implemented sections: provider|model|gateway.",
		"recommended_command=\"gormes setup\"",
		"setup_section_row_backed",
		"setup_section_unsupported: browser",
	} {
		if !strings.Contains(artifact, want) {
			t.Fatalf("artifact missing %q:\n%s", want, artifact)
		}
	}
	if strings.Contains(artifact, `unknown command "browser"`) {
		t.Fatalf("artifact leaked raw Cobra unknown-command text:\n%s", artifact)
	}
}
