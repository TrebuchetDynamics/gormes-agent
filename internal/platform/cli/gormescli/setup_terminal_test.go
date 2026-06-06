package gormescli

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestSetupTerminalInteractivePersistsLocalBackend(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "local\n", "terminal")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Terminal Backend", "Current: Local", "Terminal backend set to: local"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	cfg, loadErr := config.Load(nil)
	if loadErr != nil {
		t.Fatalf("load config: %v", loadErr)
	}
	if cfg.Runtime.TerminalBackend != "local" {
		t.Fatalf("Runtime.TerminalBackend = %q, want local", cfg.Runtime.TerminalBackend)
	}
}

func TestSetupTerminalNonInteractiveRendersSectionChromeAndChoices(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "terminal", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Gormes Setup — Terminal Backend",
		"Terminal Backend",
		"Current: Local",
		"Local - run directly on this machine (default)",
		"Docker - isolated container with configurable resources",
		"Modal - serverless cloud sandbox",
		"SSH - run on a remote machine",
		"Daytona - persistent cloud development environment",
		"Singularity/Apptainer - HPC-friendly container",
		"Keep current",
		"Keeping current backend: local",
		"Terminal Backend configuration complete!",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout+stderr, "setup_section_unsupported") {
		t.Fatalf("terminal section returned unsupported evidence:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

func TestSetupTerminalInteractiveFutureBackendReportsRowBacked(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "docker\n", "terminal")
	if err == nil {
		t.Fatalf("Execute() error = nil, want row-backed backend stdout=%s stderr=%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "setup_terminal_backend_row_backed: backend=docker") {
		t.Fatalf("stderr missing row-backed evidence:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "setup_terminal_backend_row_backed: docker") {
		t.Fatalf("err = %v", err)
	}
}
