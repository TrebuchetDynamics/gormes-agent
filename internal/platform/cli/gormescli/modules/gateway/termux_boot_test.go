package gateway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGatewayBootInstall_Uninstall_Termux(t *testing.T) {
	if !TermuxDetected() {
		t.Skip("not on Termux")
	}
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("TERMUX_VERSION", "0.119.0")

	cmd := NewBootInstallCommand()
	var out strings.Builder
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("boot-install: %v", err)
	}
	script := filepath.Join(tmpHome, ".termux", "boot", "gormes-gateway.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatalf("script not created: %v", err)
	}
	data, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	if !strings.Contains(string(data), "tmux new-session") {
		t.Fatalf("script missing tmux command: %s", data)
	}

	out.Reset()
	cmd = NewBootUninstallCommand()
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("boot-uninstall: %v", err)
	}
	if _, err := os.Stat(script); !os.IsNotExist(err) {
		t.Fatalf("script still exists after uninstall")
	}
}

func TestGatewayBootInstall_RequiresTermux(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "")
	t.Setenv("PREFIX", "")
	t.Setenv("HOME", "/tmp")
	cmd := NewBootInstallCommand()
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error on non-Termux host")
	}
}

func TestGatewayBootUninstall_Idempotent(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("TERMUX_VERSION", "0.119.0")

	cmd := NewBootUninstallCommand()
	var out strings.Builder
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("boot-uninstall on missing script: %v", err)
	}
	if !strings.Contains(out.String(), "does not exist") {
		t.Fatalf("expected 'does not exist' message, got: %s", out.String())
	}
}
