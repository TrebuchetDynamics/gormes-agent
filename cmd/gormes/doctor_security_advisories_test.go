package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// `gormes doctor --offline` must render the ◆ Security Advisories section
// FIRST (parity intent hermes doctor.py@55c9f3206:350 — most urgent). Owned
// divergence: Gormes is a pure Go binary with no Python venv, so the faithful
// state is a clean PASS "No active security advisories" (Hermes' "silent
// otherwise"), with Gormes-owned wording only (never ~/.hermes/`hermes`).
func TestDoctorCommandRendersSecurityAdvisoriesFirstCleanPass(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline"})
	_ = cmd.Execute()

	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "◆ Security Advisories") {
		t.Fatalf("doctor must render the ◆ Security Advisories section:\n%s", out)
	}
	if !strings.Contains(out, "No active security advisories") {
		t.Fatalf("pure-Go runtime must report no active advisories:\n%s", out)
	}
	advIdx := strings.Index(out, "◆ Security Advisories")
	dirIdx := strings.Index(out, "◆ Directory Structure")
	if dirIdx >= 0 && advIdx > dirIdx {
		t.Fatalf("◆ Security Advisories must render before other sections:\n%s", out)
	}
	for _, forbidden := range []string{"~/.hermes", "hermes doctor --ack", "importlib"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("Security Advisories leaked Hermes-owned/Python wording %q:\n%s", forbidden, out)
		}
	}
}

// `gormes doctor --ack <id>` persists the dismissal to the Gormes-owned
// ~/.gormes ack store (never a Python config.yaml) and confirms it.
func TestDoctorCommandAckPersistsToGormesHome(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--ack", "shai-hulud-2026-05"})
	_ = cmd.Execute()

	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "acknowledged shai-hulud-2026-05") {
		t.Fatalf("--ack must confirm the dismissal in output:\n%s", out)
	}

	ackPath := filepath.Join(config.GormesHome(), "security", "acked_advisories.json")
	raw, err := os.ReadFile(ackPath)
	if err != nil {
		t.Fatalf("ack must persist under ~/.gormes/security: %v", err)
	}
	if !strings.Contains(string(raw), "shai-hulud-2026-05") {
		t.Fatalf("ack store must record the advisory id, got: %s", raw)
	}

	// Round-trip: a second run still finds the persisted ack store readable
	// and the section stays a clean PASS (no Python package installed).
	cmd2 := newRootCommand()
	var so2, se2 bytes.Buffer
	cmd2.SetOut(&so2)
	cmd2.SetErr(&se2)
	cmd2.SetArgs([]string{"doctor", "--offline"})
	_ = cmd2.Execute()
	out2 := so2.String() + se2.String()
	if !strings.Contains(out2, "◆ Security Advisories") || !strings.Contains(out2, "No active security advisories") {
		t.Fatalf("post-ack run must still render a clean ◆ Security Advisories section:\n%s", out2)
	}
}
