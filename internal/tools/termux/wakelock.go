package termux

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

const wakeLockCommand = "termux-wake-lock"

// WakeLockManager acquires and releases the Android wake lock through
// termux-wake-lock. It is a no-op on non-Termux hosts.
type WakeLockManager struct {
	Env      map[string]string
	LookPath func(string) (string, error)
	Run      func(context.Context, string, ...string) error
}

// Acquire runs `termux-wake-lock` if on Termux and the command is available.
// Returns structured evidence; errors are non-fatal.
func (m WakeLockManager) Acquire(ctx context.Context) error {
	if !m.termux() {
		return nil
	}
	cmd, err := m.commandPath()
	if err != nil {
		return fmt.Errorf("termux-wake-lock unavailable: %w", err)
	}
	if err := m.run(ctx, cmd); err != nil {
		return fmt.Errorf("termux-wake-lock failed: %w", err)
	}
	return nil
}

// Release runs `termux-wake-lock --release` if on Termux and the command is
// available. Returns structured evidence; errors are non-fatal.
func (m WakeLockManager) Release(ctx context.Context) error {
	if !m.termux() {
		return nil
	}
	cmd, err := m.commandPath()
	if err != nil {
		return fmt.Errorf("termux-wake-lock unavailable: %w", err)
	}
	if err := m.run(ctx, cmd, "--release"); err != nil {
		return fmt.Errorf("termux-wake-lock --release failed: %w", err)
	}
	return nil
}

func (m WakeLockManager) commandPath() (string, error) {
	lp := m.LookPath
	if lp == nil {
		lp = exec.LookPath
	}
	return lp(wakeLockCommand)
}

func (m WakeLockManager) run(ctx context.Context, command string, args ...string) error {
	runner := m.Run
	if runner == nil {
		runner = defaultWakeLockRunner
	}
	return runner(ctx, command, args...)
}

func (m WakeLockManager) termux() bool {
	return IsEnvironment(m.env)
}

func (m WakeLockManager) env(key string) string {
	if m.Env != nil {
		return m.Env[key]
	}
	return os.Getenv(key)
}

func defaultWakeLockRunner(ctx context.Context, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	return cmd.Run()
}
