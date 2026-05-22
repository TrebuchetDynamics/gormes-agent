package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestNavivoxPairNoWaitCreatesLocalPairingHandoff(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	previousToken, hadPreviousToken := os.LookupEnv("GORMES_NAVIVOX_TOKEN")
	if err := os.Unsetenv("GORMES_NAVIVOX_TOKEN"); err != nil {
		t.Fatalf("unset GORMES_NAVIVOX_TOKEN: %v", err)
	}
	t.Cleanup(func() {
		if hadPreviousToken {
			_ = os.Setenv("GORMES_NAVIVOX_TOKEN", previousToken)
		} else {
			_ = os.Unsetenv("GORMES_NAVIVOX_TOKEN")
		}
	})

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "navivox", "pair", "--no-wait")
	if err != nil {
		t.Fatalf("navivox pair --no-wait: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Navivox pairing ready.",
		"Local bridge URL: http://127.0.0.1:8765",
		"WebSocket URL: ws://127.0.0.1:8765/v1/navivox/stream",
		"Pairing token: generated and stored",
		"Pairing QR image: ",
		"Open Navivox on Android and scan the QR.",
		"Waiting for Navivox connection skipped (--no-wait).",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Navivox.Enabled || cfg.Navivox.BindHost != "127.0.0.1" || cfg.Navivox.Port != 8765 || cfg.Navivox.AuthMode != config.NavivoxAuthPairingToken {
		t.Fatalf("navivox config = %+v, want local pairing_token config", cfg.Navivox)
	}
	if cfg.Navivox.Token == "" {
		t.Fatal("navivox token was not generated")
	}
	if strings.Contains(stdout+stderr, cfg.Navivox.Token) {
		t.Fatalf("navivox pair leaked generated token:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	qrPath := filepath.Join(home, "navivox", "pairing.png")
	info, err := os.Stat(qrPath)
	if err != nil {
		t.Fatalf("stat pairing QR image: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("pairing QR mode = %v, want 0600", got)
	}
}
