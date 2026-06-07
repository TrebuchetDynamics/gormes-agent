package tuiapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUIModelSlashBindingLocalModelReceivesSessionModelSeamAndCatalog(t *testing.T) {
	setupNativeTUITestEnv(t)

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	cfg.Hermes.Provider = "anthropic"
	cfg.Hermes.Model = "claude-sonnet-test"

	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}

	var captured tea.Model
	err = RunResolved(cmd, Invocation{Config: cfg}, Runtime{
		ProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) Program {
			captured = model
			setModel := capturedTUISetSessionModel(t, model)
			if setModel == nil {
				t.Fatal("local TUI SetSessionModelFunc = nil, want kernel-backed model switch seam")
			}
			if err := setModel("anthropic", "claude-opus-test"); err != nil {
				t.Fatalf("SetSessionModelFunc: %v", err)
			}
			return fakeTUIProgram{}
		},
	})
	if err != nil {
		t.Fatalf("RunResolved: %v", err)
	}

	if got := capturedTUIModelProvider(t, captured); got != "anthropic" {
		t.Fatalf("captured model provider = %q, want anthropic", got)
	}
	catalog := capturedTUIModelPickerCatalog(t, captured)
	if catalog == nil {
		t.Fatal("local TUI ModelPickerCatalog = nil, want provider/model catalog seam")
	}
	providers, err := catalog()
	if err != nil {
		t.Fatalf("ModelPickerCatalog(): %v", err)
	}
	if len(providers) == 0 {
		t.Fatal("ModelPickerCatalog returned no providers")
	}
}

func TestTUIModelSlashBindingRemoteTUIUnchanged(t *testing.T) {
	setupNativeTUITestEnv(t)

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}

	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)
		frame := kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1, Model: "remote-model"}
		data, _ := json.Marshal(frame)
		fmt.Fprintf(w, "event: frame\ndata: %s\n\n", data)
		flusher.Flush()
		<-r.Context().Done()
	}))
	defer gateway.Close()

	var captured tea.Model
	err = RunResolved(newRootCommand(), Invocation{
		Config:    cfg,
		RemoteURL: gateway.URL,
	}, Runtime{
		ProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) Program {
			captured = model
			return fakeTUIProgram{}
		},
	})
	if err != nil {
		t.Fatalf("RunResolved(remote): %v", err)
	}

	if setModel := capturedTUISetSessionModel(t, captured); setModel != nil {
		t.Fatal("remote TUI SetSessionModelFunc is non-nil; remote startup must not receive local kernel seam")
	}
	if catalog := capturedTUIModelPickerCatalog(t, captured); catalog != nil {
		t.Fatal("remote TUI ModelPickerCatalog is non-nil; remote startup must not receive local model catalog")
	}
}

func capturedTUISetSessionModel(t *testing.T, model tea.Model) tui.SetSessionModelFunc {
	t.Helper()
	return capturedOptionalTUIModelField[tui.SetSessionModelFunc](t, model, "setSessionModel")
}

func capturedTUIModelPickerCatalog(t *testing.T, model tea.Model) tui.ModelPickerCatalogFunc {
	t.Helper()
	return capturedOptionalTUIModelField[tui.ModelPickerCatalogFunc](t, model, "modelPickerCatalog")
}

func capturedTUIModelProvider(t *testing.T, model tea.Model) string {
	t.Helper()
	return capturedRequiredTUIModelField[string](t, model, "modelProvider")
}
