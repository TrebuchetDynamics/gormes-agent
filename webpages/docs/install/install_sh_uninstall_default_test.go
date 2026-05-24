package install_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallSh_UninstallDefaultsToApply pins the asymmetry fix found
// during the v0.2.0 fresh-install e2e probe:
//
//	sh install.sh                        → ACTUALLY installs (no preview)
//	sh install.sh --uninstall            → ACTUALLY uninstalls (no preview)
//	sh install.sh --uninstall --dry-run  → preview (caller opted in)
//
// Before the fix, `sh install.sh --uninstall` (no flags) ran
// `gormes uninstall` which defaults to dry-run, leaving the operator
// looking at a preview and assuming the cleanup happened. Asymmetric
// vs. install which actually installs by default.
//
// The test runs install.sh in a sandbox PATH containing a fake
// `gormes` shim that records the flags it was called with, then
// verifies install.sh prepended `--yes --dry-run=false` to the args.
func TestInstallSh_UninstallDefaultsToApply(t *testing.T) {
	scriptPath := repoFile(t, "install.sh")

	cases := []struct {
		name         string
		extraArgs    []string
		wantContains []string
		notContains  []string
	}{
		{
			name:         "no flags → apply by default",
			extraArgs:    nil,
			wantContains: []string{"uninstall", "--yes", "--dry-run=false"},
		},
		{
			name:         "explicit --dry-run is preserved",
			extraArgs:    []string{"--dry-run"},
			wantContains: []string{"uninstall", "--dry-run"},
			notContains:  []string{"--dry-run=false"},
		},
		{
			name:         "passthrough flags survive (--keep-config)",
			extraArgs:    []string{"--keep-config"},
			wantContains: []string{"uninstall", "--yes", "--dry-run=false", "--keep-config"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			binDir := t.TempDir()
			recordPath := filepath.Join(t.TempDir(), "shim-args.log")
			shim := filepath.Join(binDir, "gormes")
			shimBody := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + recordPath + "\n"
			if err := os.WriteFile(shim, []byte(shimBody), 0o755); err != nil {
				t.Fatal(err)
			}

			args := []string{scriptPath, "--bin-dir", binDir, "--uninstall"}
			args = append(args, tc.extraArgs...)
			cmd := exec.Command("sh", args...)
			cmd.Env = append(os.Environ(),
				"PATH="+binDir+":"+os.Getenv("PATH"),
				"GORMES_BIN_DIR="+binDir,
			)
			out, runErr := cmd.CombinedOutput()
			// install.sh may exit non-zero from various unrelated
			// paths in a sandbox, but the shim either ran or didn't.
			recorded, readErr := os.ReadFile(recordPath)
			if readErr != nil {
				t.Fatalf("shim was never invoked (no record file). install.sh exit=%v output=%s", runErr, string(out))
			}
			line := strings.TrimSpace(string(recorded))
			for _, want := range tc.wantContains {
				if !strings.Contains(line, want) {
					t.Errorf("shim args missing %q\nrecorded: %s\ninstall.sh output:\n%s", want, line, string(out))
				}
			}
			for _, banned := range tc.notContains {
				if strings.Contains(line, banned) {
					t.Errorf("shim args must NOT contain %q (caller's intent must win)\nrecorded: %s", banned, line)
				}
			}
		})
	}
}
