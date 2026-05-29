package session

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestBolt_PutGetRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, err := OpenBolt(path)
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	defer m.Close()

	ctx := context.Background()
	if err := m.Put(ctx, "telegram:42", "sess-abc"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := m.Get(ctx, "telegram:42")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "sess-abc" {
		t.Errorf("Get = %q, want %q", got, "sess-abc")
	}
}

func TestBolt_GetMissingReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, _ := OpenBolt(path)
	defer m.Close()
	got, err := m.Get(context.Background(), "does-not-exist")
	if err != nil {
		t.Errorf("Get on missing key should not error, got %v", err)
	}
	if got != "" {
		t.Errorf("Get on missing key = %q, want \"\"", got)
	}
}

func TestBolt_AutoCreatesParentDir(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "newsubdir")
	path := filepath.Join(parent, "sessions.db")
	m, err := OpenBolt(path)
	if err != nil {
		t.Fatalf("OpenBolt (missing parent dir): %v", err)
	}
	defer m.Close()

	info, err := os.Stat(parent)
	if err != nil {
		t.Fatalf("parent dir should exist after OpenBolt: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("parent is not a dir")
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("parent dir perm = %o, want 0700", perm)
	}
}

func TestBolt_CloseIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, _ := OpenBolt(path)
	if err := m.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Errorf("second Close should be no-op, got %v", err)
	}
}

func TestBolt_ConcurrentPutGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, _ := OpenBolt(path)
	defer m.Close()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _ = m.Put(ctx, "k", "v") }()
		go func() { defer wg.Done(); _, _ = m.Get(ctx, "k") }()
	}
	wg.Wait()

	got, _ := m.Get(ctx, "k")
	if got != "v" {
		t.Errorf("after concurrent writes, Get = %q, want %q", got, "v")
	}
}

func TestBolt_PutEmptyDeletes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, _ := OpenBolt(path)
	defer m.Close()
	ctx := context.Background()

	_ = m.Put(ctx, "tui:default", "sess-x")
	if got, _ := m.Get(ctx, "tui:default"); got != "sess-x" {
		t.Fatalf("setup: Get = %q, want sess-x", got)
	}

	if err := m.Put(ctx, "tui:default", ""); err != nil {
		t.Fatalf("Put(\"\") to delete: %v", err)
	}

	got, err := m.Get(ctx, "tui:default")
	if err != nil {
		t.Errorf("Get after delete: %v", err)
	}
	if got != "" {
		t.Errorf("after Put(\"\"), Get = %q, want deleted (\"\")", got)
	}
}

func TestBolt_PutEmptyOnMissingKeyIsNoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	m, _ := OpenBolt(path)
	defer m.Close()

	if err := m.Put(context.Background(), "never-existed", ""); err != nil {
		t.Errorf("Put(\"\") on missing key should be no-op, got %v", err)
	}
}

func TestBolt_LockContention(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")

	m1, err := OpenBolt(path)
	if err != nil {
		t.Fatalf("first OpenBolt: %v", err)
	}
	defer m1.Close()

	start := time.Now()
	_, err = OpenBolt(path)
	elapsed := time.Since(start)

	if !errors.Is(err, ErrDBLocked) {
		t.Errorf("second OpenBolt err = %v, want errors.Is(err, ErrDBLocked)", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("second OpenBolt took %v, should time out near %v", elapsed, openTimeout)
	}
}

func TestBolt_CorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	garbage := make([]byte, 4096)
	for i := range garbage {
		garbage[i] = byte(i)
	}
	if err := os.WriteFile(path, garbage, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := OpenBolt(path)
	if !errors.Is(err, ErrDBCorrupt) {
		t.Errorf("OpenBolt on garbage file err = %v, want errors.Is(err, ErrDBCorrupt)", err)
	}
}

func TestBolt_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests skipped as root — CI containers often run as root")
	}
	parent := filepath.Join(t.TempDir(), "blocked")
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })

	path := filepath.Join(parent, "sub", "sessions.db")
	_, err := OpenBolt(path)
	if err == nil {
		t.Fatal("OpenBolt on unwritable parent should fail")
	}
	if errors.Is(err, ErrDBLocked) || errors.Is(err, ErrDBCorrupt) {
		t.Errorf("permission error should not classify as Locked/Corrupt: %v", err)
	}
}
