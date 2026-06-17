package gormescli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// TestBuildDashboardTurnLoopWiresPersistence proves the dashboard turn-loop
// factory constructs a working loop and opens the SQLite transcript store, so
// dashboard chats persist to memory.db. It runs fully offline (the configured
// endpoint is never dialed during construction).
func TestBuildDashboardTurnLoopWiresPersistence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_ENDPOINT", "http://127.0.0.1:9/v1")
	t.Setenv("GORMES_API_KEY", "test-key")
	t.Setenv("GORMES_MODEL", "test-model")

	loop, cleanup, err := buildDashboardTurnLoop(context.Background())
	if err != nil {
		t.Fatalf("buildDashboardTurnLoop: %v", err)
	}
	if loop == nil {
		t.Fatal("expected a non-nil turn loop")
	}
	if cleanup == nil {
		t.Fatal("expected a non-nil cleanup")
	}
	// Cleanup must close the kernel run loop and stores without panicking.
	defer cleanup()

	memPath := config.MemoryDBPath()
	if _, err := os.Stat(memPath); err != nil {
		t.Fatalf("expected transcript store at %s: %v", memPath, err)
	}
	if filepath.Dir(memPath) == "" {
		t.Fatal("memory db path resolved empty")
	}
}

// TestBuildDashboardTurnLoopErrorsWithoutProvider proves chat construction fails
// cleanly (rather than panicking) when no provider/endpoint is configured, so
// the command can degrade to display-only.
func TestBuildDashboardTurnLoopErrorsWithoutProvider(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_API_KEY", "")
	t.Setenv("GORMES_MODEL", "")

	loop, cleanup, err := buildDashboardTurnLoop(context.Background())
	if err == nil {
		if cleanup != nil {
			cleanup()
		}
		t.Fatalf("expected an error when no provider is configured, got loop=%v", loop)
	}
}
