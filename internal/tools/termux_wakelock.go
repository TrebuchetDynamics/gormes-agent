package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const termuxWakeLockCommand = "termux-wake-lock"

// TermuxWakeLockManager acquires and releases the Android wake lock through
// termux-wake-lock. It is a no-op on non-Termux hosts.
type TermuxWakeLockManager struct {
	Env      map[string]string
	LookPath func(string) (string, error)
	Run      func(context.Context, string, ...string) error
}

// Acquire runs `termux-wake-lock` if on Termux and the command is available.
// Returns structured evidence; errors are non-fatal.
func (m TermuxWakeLockManager) Acquire(ctx context.Context) error {
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
func (m TermuxWakeLockManager) Release(ctx context.Context) error {
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

func (m TermuxWakeLockManager) commandPath() (string, error) {
	lp := m.LookPath
	if lp == nil {
		lp = exec.LookPath
	}
	return lp(termuxWakeLockCommand)
}

func (m TermuxWakeLockManager) run(ctx context.Context, command string, args ...string) error {
	runner := m.Run
	if runner == nil {
		runner = defaultTermuxWakeLockRunner
	}
	return runner(ctx, command, args...)
}

func (m TermuxWakeLockManager) termux() bool {
	if strings.TrimSpace(m.env("TERMUX_VERSION")) != "" {
		return true
	}
	for _, key := range []string{"PREFIX", "HOME"} {
		if strings.Contains(m.env(key), "com.termux/files") {
			return true
		}
	}
	return false
}

func (m TermuxWakeLockManager) env(key string) string {
	if m.Env != nil {
		return m.Env[key]
	}
	return os.Getenv(key)
}

func defaultTermuxWakeLockRunner(ctx context.Context, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	return cmd.Run()
}
