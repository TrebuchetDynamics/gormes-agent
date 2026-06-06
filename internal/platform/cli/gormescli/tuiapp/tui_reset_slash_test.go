package tuiapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUIResetSlashBindingLocalModelReceivesSessionResetSeam(t *testing.T) {
	setupNativeTUITestEnv(t)

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}

	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}

	var captured tea.Model
	err = RunResolved(cmd, Invocation{Config: cfg}, Runtime{
		ProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) Program {
			captured = model
			reset := capturedTUISessionReset(t, model)
			if reset == nil {
				t.Fatal("local TUI SessionReset = nil, want kernel-backed reset seam")
			}
			if err := reset(); err != nil {
				t.Fatalf("SessionResetFunc: %v", err)
			}
			return fakeTUIProgram{}
		},
	})
	if err != nil {
		t.Fatalf("RunResolved: %v", err)
	}
	if captured == nil {
		t.Fatal("did not capture local TUI model")
	}
}

func TestTUIResetSlashBindingRemoteTUIUnchanged(t *testing.T) {
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

	if reset := capturedTUISessionReset(t, captured); reset != nil {
		t.Fatal("remote TUI SessionReset is non-nil; remote startup must not receive local kernel seam")
	}
}

func capturedTUISessionReset(t *testing.T, model tea.Model) tui.SessionResetFunc {
	t.Helper()
	m := capturedTUIModel(t, model)
	field := reflect.ValueOf(&m).Elem().FieldByName("sessionReset")
	if !field.IsValid() {
		t.Fatal("tui.Model missing sessionReset field")
	}
	if field.IsNil() {
		return nil
	}
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.SessionResetFunc)
}
