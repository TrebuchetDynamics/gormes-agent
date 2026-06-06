package setupterminal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setup"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestRunNonInteractivePrintsCurrentBackendAndChoices(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	var out bytes.Buffer
	err := Run(&out, &bytes.Buffer{}, true, Runtime{ShouldPrintStaticChoiceMenu: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{
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
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunInteractiveLocalPersistsBackend(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	var out, errOut bytes.Buffer
	err := Run(&out, &errOut, false, Runtime{
		PromptChoice: func(title, linePrompt, defaultValue string, choices []setup.Choice) (string, error) {
			if title != "Select terminal backend" || linePrompt != "Select terminal backend [keep]: " || defaultValue != "keep" {
				t.Fatalf("prompt = %q %q %q", title, linePrompt, defaultValue)
			}
			return "local", nil
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v stdout=%s stderr=%s", err, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "Terminal backend set to: local") {
		t.Fatalf("stdout missing set receipt:\n%s", out.String())
	}
	cfg, loadErr := config.Load(nil)
	if loadErr != nil {
		t.Fatalf("load config: %v", loadErr)
	}
	if cfg.Runtime.TerminalBackend != "local" {
		t.Fatalf("Runtime.TerminalBackend = %q, want local", cfg.Runtime.TerminalBackend)
	}
}

func TestRunInteractiveFutureBackendReportsRowBacked(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	var out, errOut bytes.Buffer
	err := Run(&out, &errOut, false, Runtime{
		PromptChoice: func(string, string, string, []setup.Choice) (string, error) { return "docker", nil },
	})
	if err == nil {
		t.Fatal("Run() error = nil, want row-backed backend")
	}
	coded, ok := err.(interface{ ExitCode() int })
	if !ok || coded.ExitCode() != 2 {
		t.Fatalf("ExitCode = %v, want 2 (err=%v)", coded, err)
	}
	if !strings.Contains(errOut.String(), "setup_terminal_backend_row_backed: backend=docker") {
		t.Fatalf("stderr missing row-backed evidence:\n%s", errOut.String())
	}
	if !strings.Contains(err.Error(), "setup_terminal_backend_row_backed: docker") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunInteractiveKeepDoesNotWrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	var writes int
	var out bytes.Buffer
	err := Run(&out, &bytes.Buffer{}, false, Runtime{
		PromptChoice: func(string, string, string, []setup.Choice) (string, error) { return "keep", nil },
		WriteTOMLValue: func(string, string, string) error {
			writes++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if writes != 0 {
		t.Fatalf("writes = %d, want 0", writes)
	}
	if !strings.Contains(out.String(), "Keeping current backend: local") {
		t.Fatalf("stdout missing keep receipt:\n%s", out.String())
	}
}
