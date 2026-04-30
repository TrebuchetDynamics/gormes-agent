package main

import (
	"reflect"
	"runtime"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// TestNativeTUIStartupUsesAltScreen proves the Go Bubble Tea TUI runs as a
// coherent full-screen surface. Upstream Hermes' current Ink TUI uses an
// alternate screen unless explicitly launched in inline mode; running Gormes'
// multi-line dashboard in normal scrollback leaves stale frame fragments in
// the terminal after each render tick.
func TestNativeTUIStartupUsesAltScreen(t *testing.T) {
	setupNativeTUITestEnv(t)

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}

	cmd := newRootCommand()
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}

	var captured []tea.ProgramOption
	err = runResolvedTUIWithRuntime(cmd, tuiInvocation{Config: cfg}, rootRuntime{
		tuiProgramFactory: func(_ tea.Model, opts ...tea.ProgramOption) tuiProgram {
			captured = append(captured, opts...)
			return fakeTUIProgram{}
		},
	})
	if err != nil {
		t.Fatalf("runResolvedTUIWithRuntime: %v", err)
	}

	altScreenPtr := reflect.ValueOf(tea.WithAltScreen()).Pointer()
	foundAltScreen := false
	for _, opt := range captured {
		ptr := reflect.ValueOf(opt).Pointer()
		if ptr == altScreenPtr {
			foundAltScreen = true
		}
		if name := runtime.FuncForPC(ptr).Name(); name != "" && containsAltScreen(name) {
			foundAltScreen = true
		}
	}
	if !foundAltScreen {
		t.Fatal("local TUI startup did not enable tea.WithAltScreen; full-screen Bubble Tea renders must not smear stale frames into normal scrollback")
	}
}

func containsAltScreen(name string) bool {
	for i := 0; i+8 <= len(name); i++ {
		if name[i:i+8] == "AltScree" {
			return true
		}
	}
	return false
}
