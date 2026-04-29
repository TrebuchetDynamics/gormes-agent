package main

import (
	"reflect"
	"runtime"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// TestNativeTUIStartupDoesNotForceAltScreen proves runResolvedTUIWithRuntime
// no longer wraps the Bubble Tea program in tea.WithAltScreen. Hermes runs
// prompt_toolkit Application(full_screen=False), and the Gormes bottom-pinned
// chrome ports that contract: the program must live in normal scrollback so
// history persists after exit. A future explicit config flag may opt back in,
// but the default startup path must not raise alt-screen mode.
func TestNativeTUIStartupDoesNotForceAltScreen(t *testing.T) {
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
	for _, opt := range captured {
		ptr := reflect.ValueOf(opt).Pointer()
		if ptr == altScreenPtr {
			t.Fatal("local TUI startup forced tea.WithAltScreen; bottom-pinned Hermes chrome must run in normal scrollback")
		}
		if name := runtime.FuncForPC(ptr).Name(); name != "" && containsAltScreen(name) {
			t.Fatalf("local TUI startup forced alt-screen-like option %q; bottom-pinned Hermes chrome must run in normal scrollback", name)
		}
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
