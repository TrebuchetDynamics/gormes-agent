package tuiapp

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUILogsSlashBindingLocalModelReceivesGatewayLogTail(t *testing.T) {
	setupNativeTUITestEnv(t)
	if err := os.MkdirAll(config.GormesHome(), 0o755); err != nil {
		t.Fatalf("mkdir GORMES_HOME: %v", err)
	}
	body := "alpha line\nbeta line\ngamma line\n"
	if err := os.WriteFile(filepath.Join(config.GormesHome(), "gormes.log"), []byte(body), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}
	prevClient := logsHTTPClient
	t.Cleanup(func() { logsHTTPClient = prevClient })
	logsHTTPClient = &http.Client{Timeout: 10 * time.Millisecond}
	prevURL := logsEndpointURL
	t.Cleanup(func() { logsEndpointURL = prevURL })
	logsEndpointURL = "http://127.0.0.1:1/dead-endpoint"

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	var sawTail bool
	var tailOut string
	var tailErr error
	runOfflineTUIForTest(t, cfg, func(model tea.Model) {
		readTail := capturedTUILogTail(t, model)
		if readTail == nil {
			return
		}
		sawTail = true
		tailOut, tailErr = readTail(2)
	})

	if !sawTail {
		t.Fatal("local TUI GatewayLogTail = nil, want CLI-backed log tail reader")
	}
	if tailErr != nil {
		t.Fatalf("GatewayLogTail(2): %v\nout=%s", tailErr, tailOut)
	}
	if strings.Contains(tailOut, "alpha line") || !strings.Contains(tailOut, "beta line") || !strings.Contains(tailOut, "gamma line") {
		t.Fatalf("GatewayLogTail(2) = %q, want last two log lines", tailOut)
	}
}

func TestTUILogsSlashBindingRemoteTUIUnchanged(t *testing.T) {
	model := newPlainRemoteTUIModel()
	if readTail := capturedTUILogTail(t, model); readTail != nil {
		t.Fatal("plain/remote TUI GatewayLogTail is non-nil; only local startup should inject log tail reader")
	}
}

func capturedTUILogTail(t *testing.T, model tea.Model) tui.GatewayLogTailFunc {
	t.Helper()
	return capturedOptionalTUIModelField[tui.GatewayLogTailFunc](t, model, "gatewayLogTail")
}
