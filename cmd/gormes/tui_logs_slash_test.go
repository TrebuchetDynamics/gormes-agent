package main

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
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
	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}

	var sawTail bool
	var tailOut string
	var tailErr error
	err = runResolvedTUIWithRuntime(cmd, tuiInvocation{Config: cfg}, rootRuntime{
		tuiProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) tuiProgram {
			return fakeTUIProgram{run: func() {
				readTail := capturedTUILogTail(t, model)
				if readTail == nil {
					return
				}
				sawTail = true
				tailOut, tailErr = readTail(2)
			}}
		},
	})
	if err != nil {
		t.Fatalf("runResolvedTUIWithRuntime: %v", err)
	}

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
	model := tui.NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, tui.Options{})
	if readTail := capturedTUILogTail(t, model); readTail != nil {
		t.Fatal("plain/remote TUI GatewayLogTail is non-nil; only local startup should inject log tail reader")
	}
}

func capturedTUILogTail(t *testing.T, model tea.Model) tui.GatewayLogTailFunc {
	t.Helper()

	m, ok := model.(tui.Model)
	if !ok {
		t.Fatalf("captured model type = %T, want tui.Model", model)
	}

	field := reflect.ValueOf(&m).Elem().FieldByName("gatewayLogTail")
	if !field.IsValid() {
		t.Fatal("tui.Model missing gatewayLogTail field")
	}
	if field.IsNil() {
		return nil
	}

	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.GatewayLogTailFunc)
}
