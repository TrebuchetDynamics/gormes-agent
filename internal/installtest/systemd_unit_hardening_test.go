package installtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallSystemdUnitHasRestartHardening guards against silent
// regressions of the gateway-service crash-loop guardrails.
//
// Background: install.sh writes
// ~/.config/systemd/user/gormes-gateway.service with `Restart=on-failure`.
// systemd's defaults (StartLimitIntervalSec=10s, StartLimitBurst=5) plus
// `RestartSec=5` allow ~2 restarts per 10s window, which never trips the
// burst limit — a misconfigured unit (e.g. missing [hermes].endpoint) can
// crash-loop indefinitely. Production saw 13,151 accumulated restarts.
//
// The unit heredoc must therefore include:
//   - RestartSec >= 30s so two restarts cannot fit in the default window.
//   - StartLimitIntervalSec >= 300s so the burst counter spans a meaningful
//     interval.
//   - StartLimitBurst preserved (default 5 is fine, but pinning it makes the
//     contract explicit and survives systemd default churn).
func TestInstallSystemdUnitHasRestartHardening(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatalf("read install.sh: %v", err)
	}
	src := string(body)

	// Locate the gateway unit heredoc (between [Service] and [Install]).
	servicePos := strings.Index(src, "Description=Gormes Gateway")
	if servicePos < 0 {
		t.Fatal("install.sh: systemd unit heredoc not found")
	}
	tail := src[servicePos:]
	endPos := strings.Index(tail, "SYSTEMDUNIT")
	if endPos < 0 {
		t.Fatal("install.sh: SYSTEMDUNIT terminator not found in unit heredoc")
	}
	unit := tail[:endPos]

	mustContain := []string{
		"Restart=on-failure",
		"RestartSec=30",
		"StartLimitIntervalSec=300",
		"StartLimitBurst=5",
	}
	for _, want := range mustContain {
		if !strings.Contains(unit, want) {
			t.Errorf("systemd unit heredoc missing %q\nrendered unit body:\n%s", want, unit)
		}
	}
}
