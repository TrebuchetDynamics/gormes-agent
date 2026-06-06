package tuiapp

import (
	"context"
	"reflect"
	"testing"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUIUsageSlashBindingLocalModelReceivesAccountUsageAdapter(t *testing.T) {
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	cfg.Hermes.Provider = "custom-provider"
	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}

	var sawAccountUsage bool
	var snapshot llm.AccountUsageSnapshot
	var accountErr error
	err = RunResolved(cmd, Invocation{Config: cfg}, Runtime{
		ProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) Program {
			return fakeTUIProgram{run: func() {
				accountUsage := capturedTUIAccountUsage(t, model)
				if accountUsage == nil {
					return
				}
				sawAccountUsage = true
				snapshot, accountErr = accountUsage(context.Background())
			}}
		},
	})
	if err != nil {
		t.Fatalf("RunResolved: %v", err)
	}
	if !sawAccountUsage {
		t.Fatal("local TUI AccountUsage = nil, want provider-backed usage adapter")
	}
	if accountErr != nil {
		t.Fatalf("AccountUsage: %v", accountErr)
	}
	if snapshot.Provider != "custom-provider" {
		t.Fatalf("AccountUsage provider = %q, want custom-provider", snapshot.Provider)
	}
	if snapshot.Unavailable == nil || snapshot.Unavailable.Reason != llm.AccountUsageReasonUnsupportedProvider {
		t.Fatalf("AccountUsage unavailable = %+v, want unsupported provider evidence", snapshot.Unavailable)
	}
}

func TestTUIUsageSlashBindingRemoteTUIUnchanged(t *testing.T) {
	model := tui.NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, tui.Options{})
	if accountUsage := capturedTUIAccountUsage(t, model); accountUsage != nil {
		t.Fatal("plain/remote TUI AccountUsage is non-nil; only local startup should inject provider adapter")
	}
}

func capturedTUIAccountUsage(t *testing.T, model tea.Model) tui.AccountUsageFunc {
	t.Helper()

	m, ok := model.(tui.Model)
	if !ok {
		t.Fatalf("captured model type = %T, want tui.Model", model)
	}

	field := reflect.ValueOf(&m).Elem().FieldByName("accountUsage")
	if !field.IsValid() {
		t.Fatal("tui.Model missing accountUsage field")
	}
	if field.IsNil() {
		return nil
	}

	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(tui.AccountUsageFunc)
}
