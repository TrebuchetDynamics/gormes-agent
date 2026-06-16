package gormescli

import (
	"fmt"
	"io"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
)

func RenderSetupSectionHeader(label string) string {
	return doctor.RenderDoctorHeader("Gormes Setup — " + label)
}

func SetupSectionSuppressSuccessFooter(section, out string) bool {
	if SetupSectionOutputCancelled(out) {
		return true
	}
	return section == "provider" && SetupProviderReceiptRendered(out)
}

func SetupSectionOutputCancelled(out string) bool {
	for _, sentinel := range []string{
		"Setup cancelled.",
		"setup canceled; no files were written.",
		"Setup canceled.",
	} {
		if strings.Contains(out, sentinel) {
			return true
		}
	}
	return false
}

func SetupProviderReceiptRendered(out string) bool {
	return strings.Contains(out, "\nConnection\n") &&
		strings.Contains(out, "\nAuthentication\n") &&
		strings.Contains(out, "\nNext steps\n")
}

func WriteSetupSectionUnsupported(out io.Writer, section, sectionList string) {
	fmt.Fprintf(out, "setup_section_unsupported: section=%s available=%s\n", section, sectionList)
	fmt.Fprintf(out, "Implemented sections: %s.\n", sectionList)
	fmt.Fprintln(out, "setup_section_row_backed: recommended_command=\"gormes setup\"")
}

func SetupSectionUnsupportedError(section string) error {
	return fmt.Errorf("setup_section_unsupported: %s", section)
}
