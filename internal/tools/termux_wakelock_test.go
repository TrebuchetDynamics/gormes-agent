package tools

import (
	"context"
	"os/exec"
	"testing"
)

func TestTermuxWakeLockManager_AcquireRelease_NonTermux(t *testing.T) {
	m := TermuxWakeLockManager{Env: map[string]string{}}
	if err := m.Acquire(context.Background()); err != nil {
		t.Fatalf("non-Termux acquire should be no-op: %v", err)
	}
	if err := m.Release(context.Background()); err != nil {
		t.Fatalf("non-Termux release should be no-op: %v", err)
	}
}

func TestTermuxWakeLockManager_AcquireRelease_TermuxMissingCommand(t *testing.T) {
	m := TermuxWakeLockManager{
		Env: map[string]string{"TERMUX_VERSION": "0.119.0"},
		LookPath: func(string) (string, error) {
			return "", exec.ErrNotFound
		},
	}
	if err := m.Acquire(context.Background()); err == nil {
		t.Fatal("Termux with missing termux-wake-lock should error on acquire")
	}
	if err := m.Release(context.Background()); err == nil {
		t.Fatal("Termux with missing termux-wake-lock should error on release")
	}
}

func TestTermuxWakeLockManager_AcquireRelease_TermuxSuccess(t *testing.T) {
	var calls []struct {
		cmd  string
		args []string
	}
	m := TermuxWakeLockManager{
		Env: map[string]string{"TERMUX_VERSION": "0.119.0"},
		LookPath: func(string) (string, error) {
			return "/data/data/com.termux/files/usr/bin/termux-wake-lock", nil
		},
		Run: func(_ context.Context, cmd string, args ...string) error {
			calls = append(calls, struct {
				cmd  string
				args []string
			}{cmd, args})
			return nil
		},
	}
	if err := m.Acquire(context.Background()); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if err := m.Release(context.Background()); err != nil {
		t.Fatalf("release: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %v", len(calls), calls)
	}
	if calls[0].cmd != "/data/data/com.termux/files/usr/bin/termux-wake-lock" || len(calls[0].args) != 0 {
		t.Fatalf("acquire call wrong: %+v", calls[0])
	}
	if calls[1].cmd != "/data/data/com.termux/files/usr/bin/termux-wake-lock" || len(calls[1].args) != 1 || calls[1].args[0] != "--release" {
		t.Fatalf("release call wrong: %+v", calls[1])
	}
}

func TestTermuxWakeLockManager_Release_OnlyReleaseFlag(t *testing.T) {
	var calls []struct {
		args []string
	}
	m := TermuxWakeLockManager{
		Env: map[string]string{"PREFIX": "/data/data/com.termux/files/usr"},
		LookPath: func(string) (string, error) {
			return "termux-wake-lock", nil
		},
		Run: func(_ context.Context, _ string, args ...string) error {
			calls = append(calls, struct{ args []string }{args})
			return nil
		},
	}
	_ = m.Release(context.Background())
	if len(calls) != 1 || len(calls[0].args) != 1 || calls[0].args[0] != "--release" {
		t.Fatalf("expected [--release], got %v", calls)
	}
}
