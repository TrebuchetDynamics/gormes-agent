package sandbox

import (
	"context"
	"testing"
)

// TestSandboxProviderInterface ensures the SandboxProvider interface
// contract compiles and can be satisfied by a LocalSandboxProvider.
func TestSandboxProviderInterfaceContract(t *testing.T) {
	// Compile-time check: *localProvider implements SandboxProvider
	var _ SandboxProvider = (*LocalSandboxProvider)(nil)

	ctx := context.Background()
	p := NewLocalSandboxProvider(LocalSandboxConfig{
		BaseDir: t.TempDir(),
	})

	// Acquire should return a sandbox
	sb, err := p.Acquire(ctx, "test-session")
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	if sb == nil {
		t.Fatal("Acquire returned nil sandbox")
	}

	// Get should return the same sandbox for the same session
	sb2, err := p.Get(ctx, "test-session")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if sb2 != sb {
		t.Fatal("Get returned a different sandbox for the same session")
	}

	// Release should succeed
	if err := p.Release(ctx, "test-session"); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// After release, Get should return an error
	_, err = p.Get(ctx, "test-session")
	if err == nil {
		t.Fatal("expected error after releasing sandbox")
	}

	// Shutdown should succeed
	if err := p.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

// TestSandboxInterfaceContract ensures the Sandbox interface compiles.
func TestSandboxInterfaceContract(t *testing.T) {
	// Compile-time check: *sandbox implements Sandbox
	var _ Sandbox = (*sandbox)(nil)
}

func TestLocalSandboxProvider_Acquire_ReusesExisting(t *testing.T) {
	ctx := context.Background()
	p := NewLocalSandboxProvider(LocalSandboxConfig{
		BaseDir: t.TempDir(),
	})

	sb1, err := p.Acquire(ctx, "session-1")
	if err != nil {
		t.Fatalf("first Acquire failed: %v", err)
	}

	sb2, err := p.Acquire(ctx, "session-1")
	if err != nil {
		t.Fatalf("second Acquire failed: %v", err)
	}

	if sb1 != sb2 {
		t.Fatal("Acquire should return the same sandbox for an existing session")
	}
}

func TestLocalSandboxProvider_Acquire_IsolatedSessions(t *testing.T) {
	ctx := context.Background()
	p := NewLocalSandboxProvider(LocalSandboxConfig{
		BaseDir: t.TempDir(),
	})

	sb1, err := p.Acquire(ctx, "session-a")
	if err != nil {
		t.Fatalf("Acquire session-a failed: %v", err)
	}

	sb2, err := p.Acquire(ctx, "session-b")
	if err != nil {
		t.Fatalf("Acquire session-b failed: %v", err)
	}

	if sb1 == sb2 {
		t.Fatal("different sessions should have different sandboxes")
	}

	// Each sandbox should have its own workspace directory
	if sb1.WorkspaceDir() == sb2.WorkspaceDir() {
		t.Fatal("different sessions should have different workspace directories")
	}
}

func TestLocalSandboxProvider_Shutdown_ReleasesAll(t *testing.T) {
	ctx := context.Background()
	p := NewLocalSandboxProvider(LocalSandboxConfig{
		BaseDir: t.TempDir(),
	})

	_, _ = p.Acquire(ctx, "session-a")
	_, _ = p.Acquire(ctx, "session-b")

	if err := p.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// After shutdown, all sessions should be released
	_, err := p.Get(ctx, "session-a")
	if err == nil {
		t.Fatal("expected error after shutdown")
	}
	_, err = p.Get(ctx, "session-b")
	if err == nil {
		t.Fatal("expected error after shutdown")
	}
}
