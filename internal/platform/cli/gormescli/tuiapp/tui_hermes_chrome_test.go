package tuiapp

import (
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestNativeTUIStartupUsesAltScreen proves the Go Bubble Tea TUI runs as a
// coherent full-screen surface. Upstream Hermes' current Ink TUI uses an
// alternate screen unless explicitly launched in inline mode; running Gormes'
// multi-line dashboard in normal scrollback leaves stale frame fragments in
// the terminal after each render tick.
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
