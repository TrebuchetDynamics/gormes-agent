package main

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestKanbanTailCommandStreamsTaskEventsUntilContextCancel(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	task := runKanbanJSONTask(t, "create", "Tail task", "--json")
	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "block", task.ID, "waiting"); err != nil {
		t.Fatalf("kanban block: %v\nstderr=%s", err, stderr)
	}

	run := startKanbanTailCommand(t, task.ID, "--interval", "0")
	output := run.waitForOutput(t, "] blocked waiting")
	stdout, stderr, err := run.stop(t)
	if err != nil {
		t.Fatalf("kanban tail after cancel: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("kanban tail stderr = %q, want empty", stderr)
	}
	if !strings.Contains(output, "Tailing events for "+task.ID+". Ctrl-C to stop.") {
		t.Fatalf("kanban tail output = %q, want Hermes-shaped header", output)
	}
	if !strings.Contains(output, "] created") || !strings.Contains(output, "] blocked waiting") {
		t.Fatalf("kanban tail output = %q, want created and blocked task event lines", output)
	}
}

func TestKanbanTailCommandPrintsNewEvents(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	task := runKanbanJSONTask(t, "create", "Live tail task", "--json")
	run := startKanbanTailCommand(t, task.ID, "--interval", "0")
	run.waitForOutput(t, "Tailing events for "+task.ID+". Ctrl-C to stop.")

	if _, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "block", task.ID, "new event"); err != nil {
		t.Fatalf("kanban block while tailing: %v\nstderr=%s", err, stderr)
	}
	output := run.waitForOutput(t, "] blocked new event")
	stdout, stderr, err := run.stop(t)
	if err != nil {
		t.Fatalf("kanban tail after new event cancel: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("kanban tail stderr = %q, want empty", stderr)
	}
	if got := strings.Count(output, "] blocked new event"); got != 1 {
		t.Fatalf("blocked event count = %d in output %q, want exactly once", got, output)
	}
}

func TestKanbanTailCommandClampsFastIntervals(t *testing.T) {
	for _, seconds := range []float64{-1, 0, 0.05} {
		if got := kanbanTailPollInterval(seconds); got != 100*time.Millisecond {
			t.Fatalf("kanbanTailPollInterval(%v) = %s, want 100ms", seconds, got)
		}
	}
	if got := kanbanTailPollInterval(0.25); got != 250*time.Millisecond {
		t.Fatalf("kanbanTailPollInterval(0.25) = %s, want 250ms", got)
	}
}

type kanbanTailRun struct {
	cancel context.CancelFunc
	done   chan error
	stdout *lockedBuffer
	stderr *lockedBuffer
}

func startKanbanTailCommand(t *testing.T, args ...string) *kanbanTailRun {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := newRootCommandWithRuntime(rootRuntime{})
	cmd.SetContext(ctx)
	run := &kanbanTailRun{
		cancel: cancel,
		done:   make(chan error, 1),
		stdout: &lockedBuffer{},
		stderr: &lockedBuffer{},
	}
	cmd.SetOut(run.stdout)
	cmd.SetErr(run.stderr)
	go func() {
		run.done <- executeRootCommand(cmd, append([]string{"kanban", "tail"}, args...)...)
	}()
	return run
}

func (r *kanbanTailRun) waitForOutput(t *testing.T, want string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		out := r.stdout.String()
		if strings.Contains(out, want) {
			return out
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q\nstdout=%s\nstderr=%s", want, r.stdout.String(), r.stderr.String())
	return ""
}

func (r *kanbanTailRun) stop(t *testing.T) (string, string, error) {
	t.Helper()
	r.cancel()
	select {
	case err := <-r.done:
		return r.stdout.String(), r.stderr.String(), err
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for kanban tail to stop\nstdout=%s\nstderr=%s", r.stdout.String(), r.stderr.String())
		return r.stdout.String(), r.stderr.String(), nil
	}
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
