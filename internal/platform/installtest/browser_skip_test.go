package installtest

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestInstall_DryRunSkipBrowserFlagsAreAccepted(t *testing.T) {
	for _, flag := range []string{"--skip-browser", "--no-playwright"} {
		t.Run(flag, func(t *testing.T) {
			sb := t.TempDir()
			out := runInstallDryRun(t, map[string]string{
				"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
				"GORMES_SKIP_SETUP":      "1",
				"GORMES_RESTART_GATEWAY": "never",
			}, flag)

			if !strings.Contains(out, "browser    skip") {
				t.Fatalf("dry-run plan should report browser_setup as skipped for %s; got:\n%s", flag, out)
			}
			if !strings.Contains(out, "no Playwright install needed") {
				t.Fatalf("dry-run plan should explain %s is a no-op for Gormes' no Playwright install needed install; got:\n%s", flag, out)
			}
		})
	}
}
