package gormescli

import (
	"strings"
	"testing"
)

// `gormes doctor --offline` must now render the ◆ Directory Structure
// section (previously empty because no check fed it). Parity with hermes
// doctor.py@55c9f3206:812, Gormes-owned layout/wording.
func TestDoctorCommandRendersDirectoryStructureSection(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	stdout, stderr, _ := executeCobraCommandForTest(newRootCommand(), cobraCommandExecutionOptions{}, "doctor", "--offline")

	out := stdout + stderr
	if !strings.Contains(out, "◆ Directory Structure") {
		t.Fatalf("doctor must render the ◆ Directory Structure section:\n%s", out)
	}
	if !strings.Contains(out, "some Gormes directories/files not yet present") {
		t.Fatalf("doctor must emit a Directory Structure summary line:\n%s", out)
	}
	if !strings.Contains(out, "~/.gormes") || !strings.Contains(out, "sessions/") {
		t.Fatalf("Directory Structure must list the Gormes-owned home + subdir layout:\n%s", out)
	}
	for _, forbidden := range []string{"~/.hermes", "hermes setup", "memories/"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("Directory Structure leaked Hermes-owned wording %q:\n%s", forbidden, out)
		}
	}
}
