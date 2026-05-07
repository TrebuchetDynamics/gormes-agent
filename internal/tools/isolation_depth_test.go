package tools

import (
	"testing"
)

// Row acceptance criteria:
// - Process-level isolation is the default and requires zero setup
// - Docker/gVisor isolation selectable via config
// - Firecracker VM isolation selectable via config
// - Isolation depth is per-session configurable
// - Deeper isolation correctly fails if backend not available

func TestIsolationDepth_DefaultIsProcessLevel(t *testing.T) {
	cfg := DefaultIsolationConfig()
	if cfg.Level != IsolationProcess {
		t.Fatalf("default isolation level = %s, want process", cfg.Level)
	}
	if !cfg.IsAvailable() {
		t.Fatal("process-level isolation must always be available (zero setup)")
	}
	if cfg.RequiresSetup() {
		t.Fatal("process-level isolation must not require setup")
	}
}

func TestIsolationDepth_ProcessLevelAlwaysAvailable(t *testing.T) {
	// Process-level must work standalone — no Docker, no VM socket.
	cfg := IsolationConfig{Level: IsolationProcess}
	if !cfg.IsAvailable() {
		t.Fatal("process isolation must be available with no container image or VM socket")
	}
}

func TestIsolationDepth_ContainerSelectableViaConfig(t *testing.T) {
	cfg, err := NewIsolationConfigFromMode("container", "ubuntu:22.04", "")
	if err != nil {
		t.Fatalf("NewIsolationConfigFromMode error: %v", err)
	}
	if cfg.Level != IsolationContainer {
		t.Fatalf("level = %s, want container", cfg.Level)
	}
	if !cfg.IsAvailable() {
		t.Fatal("container with image should be available")
	}
	if !cfg.RequiresSetup() {
		t.Fatal("container isolation should require setup")
	}
}

func TestIsolationDepth_ContainerUnavailableWithoutImage(t *testing.T) {
	cfg, err := NewIsolationConfigFromMode("container", "", "")
	if err != nil {
		t.Fatalf("NewIsolationConfigFromMode error: %v", err)
	}
	if cfg.IsAvailable() {
		t.Fatal("container without image should NOT be available")
	}
}

func TestIsolationDepth_VMSelectableViaConfig(t *testing.T) {
	cfg, err := NewIsolationConfigFromMode("vm", "", "/var/run/firecracker.sock")
	if err != nil {
		t.Fatalf("NewIsolationConfigFromMode error: %v", err)
	}
	if cfg.Level != IsolationVM {
		t.Fatalf("level = %s, want vm", cfg.Level)
	}
	if !cfg.IsAvailable() {
		t.Fatal("VM with socket should be available")
	}
	if !cfg.RequiresSetup() {
		t.Fatal("VM isolation should require setup")
	}
}

func TestIsolationDepth_VMUnavailableWithoutSocket(t *testing.T) {
	cfg, err := NewIsolationConfigFromMode("vm", "", "")
	if err != nil {
		t.Fatalf("NewIsolationConfigFromMode error: %v", err)
	}
	if cfg.IsAvailable() {
		t.Fatal("VM without socket should NOT be available")
	}
}

func TestIsolationDepth_InvalidModeFallsBackToProcess(t *testing.T) {
	cfg, err := NewIsolationConfigFromMode("quantum", "", "")
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if cfg.Level != IsolationProcess {
		t.Fatalf("invalid mode should fall back to process, got %s", cfg.Level)
	}
}

func TestIsolationDepth_PerSessionConfigurable(t *testing.T) {
	// Prove different sessions can have different isolation configs.
	sessionA := IsolationConfig{Level: IsolationProcess}
	sessionB := IsolationConfig{Level: IsolationContainer, ContainerImage: "gvisor:latest"}

	if sessionA.Level == sessionB.Level {
		t.Fatal("per-session isolation configs must differ")
	}
	if !sessionA.IsAvailable() {
		t.Fatal("session A (process) must be available")
	}
	if !sessionB.IsAvailable() {
		t.Fatal("session B (container with image) must be available")
	}
}

func TestIsolationDepth_StringRepresentation(t *testing.T) {
	tests := []struct {
		level IsolationLevel
		want  string
	}{
		{IsolationProcess, "process"},
		{IsolationContainer, "container"},
		{IsolationVM, "vm"},
		{IsolationLevel(99), "unknown"},
	}
	for _, tt := range tests {
		got := tt.level.String()
		if got != tt.want {
			t.Errorf("IsolationLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestIsolationDepth_ParseIsolationLevel(t *testing.T) {
	tests := []struct {
		input string
		want  IsolationLevel
		ok    bool
	}{
		{"process", IsolationProcess, true},
		{"container", IsolationContainer, true},
		{"vm", IsolationVM, true},
		{"docker", IsolationProcess, false},
		{"", IsolationProcess, false},
		{"PROCESS", IsolationProcess, false},
	}
	for _, tt := range tests {
		got, ok := ParseIsolationLevel(tt.input)
		if ok != tt.ok {
			t.Errorf("ParseIsolationLevel(%q) ok=%v, want %v", tt.input, ok, tt.ok)
		}
		if ok && got != tt.want {
			t.Errorf("ParseIsolationLevel(%q) = %s, want %s", tt.input, got, tt.want)
		}
	}
}
