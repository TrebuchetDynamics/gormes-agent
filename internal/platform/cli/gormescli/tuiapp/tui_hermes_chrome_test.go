package tuiapp

import (
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// TestNativeTUIStartupUsesAltScreen proves the Go Bubble Tea TUI runs as a
// coherent full-screen surface. Upstream Hermes' current Ink TUI uses an
// alternate screen unless explicitly launched in inline mode; running Gormes'
// multi-line dashboard in normal scrollback leaves stale frame fragments in
// the terminal after each render tick.
func TestNativeTUIStartupSeedsWelcomeBuildProvenance(t *testing.T) {
	cfg := loadNativeTUITestConfig(t)
	cmd := newRootCommandWithRuntime(Runtime{})
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}
	var rendered string
	err := RunResolved(cmd, Invocation{Config: cfg}, Runtime{
		Version:          "0.2.24",
		VersionDateAlias: "v2026.6.5",
		GitCommit:        "d221e369abcdef",
		ProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) Program {
			resized, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 28})
			rendered = resized.View()
			return fakeTUIProgram{}
		},
	})
	if err != nil {
		t.Fatalf("RunResolved: %v", err)
	}
	if !strings.Contains(rendered, "Gormes v0.2.24 (2026.6.5) · upstream d221e369") {
		t.Fatalf("native TUI startup did not seed Hermes-style welcome build provenance:\n%s", rendered)
	}
	wantPromptStack := strings.Repeat("─", 120) + "\n❯ \n" + strings.Repeat("─", 120)
	if !strings.Contains(rendered, wantPromptStack) {
		t.Fatalf("native TUI startup did not render Hermes-style double prompt rules around the bare composer:\n%s", rendered)
	}
	for _, stale := range []string{"main ❯ Type a message", "─ ready", "profile main ·", "Profile:", "CWD:"} {
		if strings.Contains(rendered, stale) {
			t.Fatalf("native TUI startup leaked old Gormes chrome %q:\n%s", stale, rendered)
		}
	}
}

func TestResolvedRootTUIStartupSeedsWelcomeHomeFromGormesHome(t *testing.T) {
	setupNativeTUITestEnv(t)
	expectedHome := config.GormesBaseHome()
	cmd := newRootCommandWithRuntime(Runtime{})
	if err := cmd.Flags().Set("offline", "true"); err != nil {
		t.Fatalf("set offline flag: %v", err)
	}
	invocation, err := ResolveInvocation(cmd)
	if err != nil {
		t.Fatalf("ResolveInvocation: %v", err)
	}
	if invocation.ProfileBaseHome != expectedHome {
		t.Fatalf("resolved ProfileBaseHome = %q, want %q", invocation.ProfileBaseHome, expectedHome)
	}

	var rendered string
	err = RunResolved(cmd, invocation, Runtime{
		Version:          "0.2.24",
		VersionDateAlias: "v2026.6.5",
		GitCommit:        "d221e369abcdef",
		ProgramFactory: func(model tea.Model, _ ...tea.ProgramOption) Program {
			resized, _ := model.Update(tea.WindowSizeMsg{Width: 140, Height: 52})
			rendered = resized.View()
			return fakeTUIProgram{}
		},
	})
	if err != nil {
		t.Fatalf("RunResolved: %v", err)
	}
	if !strings.Contains(rendered, expectedHome) {
		t.Fatalf("resolved native TUI welcome should render Gormes home %q instead of workspace cwd:\n%s", expectedHome, rendered)
	}
	for _, stale := range []string{"CWD:", "Profile:", "~/git/gormes/gormes-agent"} {
		if strings.Contains(rendered, stale) {
			t.Fatalf("resolved native TUI welcome leaked old workspace/profile chrome %q:\n%s", stale, rendered)
		}
	}
}

func TestNativeTUIStartupUsesAltScreen(t *testing.T) {
	cfg := loadNativeTUITestConfig(t)

	captured := captureOfflineTUIProgramOptions(t, cfg)

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

func TestNativeTUIStartupDoesNotCaptureMouseByDefault(t *testing.T) {
	cfg := loadNativeTUITestConfig(t)

	captured := captureOfflineTUIProgramOptions(t, cfg)

	mousePtr := reflect.ValueOf(tea.WithMouseAllMotion()).Pointer()
	for _, opt := range captured {
		ptr := reflect.ValueOf(opt).Pointer()
		if ptr == mousePtr {
			t.Fatal("local TUI startup enabled mouse tracking by default; terminal text selection must work without /mouse off")
		}
		if name := runtime.FuncForPC(ptr).Name(); name != "" && containsMouseAllMotion(name) {
			t.Fatalf("local TUI startup enabled mouse tracking by default via %s; terminal text selection must work without /mouse off", name)
		}
	}
}

func TestNativeTUIKernelLoggerDoesNotWriteProviderErrorsToTerminal(t *testing.T) {
	raw, err := os.ReadFile("service.go")
	if err != nil {
		t.Fatalf("read service.go: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, "}, c, store.NewNoop(), tm, slog.Default())") {
		t.Fatal("local TUI kernel still uses slog.Default(); provider/auth provenance logs can leak into the terminal")
	}
	for _, want := range []string{
		"func tuiKernelLogger() *slog.Logger",
		"slog.NewTextHandler(io.Discard, nil)",
		"}, c, store.NewNoop(), tm, tuiKernelLogger())",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("local TUI kernel log suppression missing marker %q", want)
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

func containsMouseAllMotion(name string) bool {
	for i := 0; i+14 <= len(name); i++ {
		if name[i:i+14] == "MouseAllMotion" {
			return true
		}
	}
	return false
}
