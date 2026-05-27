package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// `gormes doctor --offline` must render the ◆ Profiles section from the real
// Gormes profile seam. With only the default profile it is a single clean
// PASS "default profile only" (never WARN — Gormes always has a usable
// default). Parity intent hermes doctor.py@55c9f3206:1768; owned divergence:
// Gormes ~/.gormes/profiles wording only, never ~/.hermes-<name> / wrapper /
// per-profile gateway-running.
func TestDoctorCommandRendersProfilesDefaultOnly(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline"})
	_ = cmd.Execute()

	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "◆ Profiles") {
		t.Fatalf("doctor must render the ◆ Profiles section:\n%s", out)
	}
	if !strings.Contains(out, "✓ default profile only") {
		t.Fatalf("doctor must emit a Profiles check line:\n%s", out)
	}
	if !strings.Contains(out, "default profile only") {
		t.Fatalf("default-only must report 'default profile only':\n%s", out)
	}
	for _, forbidden := range []string{"~/.hermes", "hermes profile", "wrapper", "gateway running"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("Profiles leaked Hermes-owned/fabricated wording %q:\n%s", forbidden, out)
		}
	}
}

// A named profile directory under ~/.gormes/profiles/<name> must be enumerated
// by the ◆ Profiles section via the real seam (no hardcoded glob), with a
// computed "N profile(s) found" summary — never narrated.
func TestDoctorCommandEnumeratesNamedProfile(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	profileDir := filepath.Join(config.GormesHome(), "profiles", "work")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatalf("create named profile dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "config.toml"), []byte("[hermes]\nprovider = 'nous'\nmodel = 'moonshotai/kimi-k2.6'\n"), 0o600); err != nil {
		t.Fatalf("write profile config: %v", err)
	}
	status := gateway.NewRuntimeStatusStore(filepath.Join(profileDir, "runtime", "gateway_state.json"))
	if err := status.UpdateRuntimeStatus(context.Background(), gateway.RuntimeStatusUpdate{GatewayState: gateway.GatewayStateRunning}); err != nil {
		t.Fatalf("write profile gateway state: %v", err)
	}

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline"})
	_ = cmd.Execute()

	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "◆ Profiles") {
		t.Fatalf("doctor must render the ◆ Profiles section:\n%s", out)
	}
	if !strings.Contains(out, "2 profile(s) found") {
		t.Fatalf("named profile must produce a computed count summary:\n%s", out)
	}
	if !strings.Contains(out, "work") {
		t.Fatalf("the named profile must be enumerated by name:\n%s", out)
	}
	if !strings.Contains(out, "provider=nous") || !strings.Contains(out, "model=moonshotai/kimi-k2.6") {
		t.Fatalf("named profile must show effective local provider/model:\n%s", out)
	}
	if !strings.Contains(out, "gateway=recorded running") {
		t.Fatalf("named profile must show recorded gateway state:\n%s", out)
	}
	if strings.Contains(out, "~/.hermes") || strings.Contains(out, "wrapper") {
		t.Fatalf("Profiles leaked Hermes-owned wording:\n%s", out)
	}
}
