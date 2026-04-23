package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	defaultExecuteCodeTimeout   = 30 * time.Second
	defaultExecuteCodeMode      = "strict"
	defaultExecuteCodeStdoutCap = 50 * 1024
	defaultExecuteCodeStderrCap = 10 * 1024
	projectExecutionScratchDir  = ".gormes/code-execution"
	truncatedOutputSentinel     = "\n[output truncated]"
)

type RunState string

const (
	RunStateRunning   RunState = "running"
	RunStateCompleted RunState = "completed"
	RunStateFailed    RunState = "failed"
	RunStateTimedOut  RunState = "timed_out"
)

type ExecuteResult struct {
	Status          string  `json:"status"`
	ExitCode        int     `json:"exit_code"`
	Stdout          string  `json:"stdout,omitempty"`
	Stderr          string  `json:"stderr,omitempty"`
	Output          string  `json:"output,omitempty"`
	DurationSeconds float64 `json:"duration_seconds"`
	Mode            string  `json:"mode"`
	Workspace       string  `json:"workspace,omitempty"`
}

type ProcessSession struct {
	ID         string         `json:"id"`
	Mode       string         `json:"mode"`
	Workspace  string         `json:"workspace"`
	State      RunState       `json:"state"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	Result     *ExecuteResult `json:"result,omitempty"`
}

type ProcessRegistry struct {
	mu       sync.RWMutex
	sessions map[string]ProcessSession
}

func NewProcessRegistry() *ProcessRegistry {
	return &ProcessRegistry{sessions: make(map[string]ProcessSession)}
}

func (r *ProcessRegistry) Start(mode, workspace string) ProcessSession {
	if r == nil {
		return ProcessSession{}
	}
	session := ProcessSession{
		ID:        uuid.NewString(),
		Mode:      mode,
		Workspace: workspace,
		State:     RunStateRunning,
		StartedAt: time.Now().UTC(),
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = session
	return session
}

func (r *ProcessRegistry) Finish(id string, result ExecuteResult) {
	if r == nil || strings.TrimSpace(id) == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[id]
	if !ok {
		return
	}
	finishedAt := time.Now().UTC()
	session.FinishedAt = &finishedAt
	session.Result = &result
	session.State = stateForExecuteResult(result)
	r.sessions[id] = session
}

func (r *ProcessRegistry) Session(id string) (ProcessSession, bool) {
	if r == nil {
		return ProcessSession{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[id]
	return session, ok
}

func stateForExecuteResult(result ExecuteResult) RunState {
	switch result.Status {
	case "success":
		return RunStateCompleted
	case "timeout":
		return RunStateTimedOut
	default:
		return RunStateFailed
	}
}

type ExecuteCodeTool struct {
	Registry *ProcessRegistry
	TimeoutD time.Duration
	Getwd    func() (string, error)

	mu              sync.Mutex
	defaultRegistry *ProcessRegistry
}

func (*ExecuteCodeTool) Name() string { return "execute_code" }

func (*ExecuteCodeTool) Description() string {
	return "Run a self-contained Go program in a scrubbed strict or project workspace. Use this for multi-step local processing without recursive tool calls."
}

func (*ExecuteCodeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"code":{"type":"string","description":"Go source for a package main program."},"mode":{"type":"string","enum":["strict","project"],"description":"strict = temp sandbox; project = run from the current working tree root."}},"required":["code"]}`)
}

func (t *ExecuteCodeTool) Timeout() time.Duration {
	if t == nil || t.TimeoutD <= 0 {
		return defaultExecuteCodeTimeout
	}
	return t.TimeoutD
}

func (t *ExecuteCodeTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Code string `json:"code"`
		Mode string `json:"mode"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("execute_code: invalid args: %w", err)
	}
	if strings.TrimSpace(in.Code) == "" {
		return nil, errors.New("execute_code: code is required")
	}

	mode, err := normalizeExecutionMode(in.Mode)
	if err != nil {
		return nil, fmt.Errorf("execute_code: %w", err)
	}

	cwd, err := t.getwd()
	if err != nil {
		return nil, fmt.Errorf("execute_code: getwd: %w", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, t.Timeout())
	defer cancel()

	workspace, runDir, cmdArgs, cleanup, err := prepareExecutionWorkspace(mode, cwd, in.Code)
	if err != nil {
		return nil, fmt.Errorf("execute_code: prepare workspace: %w", err)
	}
	defer cleanup()

	registry := t.registry()
	session := registry.Start(mode, workspace)

	startedAt := time.Now()
	stdout := newCappedBuffer(defaultExecuteCodeStdoutCap)
	stderr := newCappedBuffer(defaultExecuteCodeStderrCap)

	cmd := exec.CommandContext(runCtx, "go", cmdArgs...)
	cmd.Dir = runDir
	cmd.Env = scrubbedExecEnv()
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	result := ExecuteResult{
		Status:    "success",
		ExitCode:  0,
		Mode:      mode,
		Workspace: workspace,
	}

	runErr := cmd.Run()
	result.DurationSeconds = time.Since(startedAt).Seconds()
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.Output = combineExecutionOutput(result.Stdout, result.Stderr)

	switch {
	case runErr == nil:
		// success path already seeded above
	case errors.Is(runCtx.Err(), context.DeadlineExceeded):
		result.Status = "timeout"
		result.ExitCode = -1
		if result.Stderr == "" {
			result.Stderr = "execution timed out"
		}
		result.Output = combineExecutionOutput(result.Stdout, result.Stderr)
	case exitErr(runErr) != nil:
		result.Status = "error"
		result.ExitCode = exitErr(runErr).ExitCode()
	default:
		return nil, fmt.Errorf("execute_code: run go: %w", runErr)
	}

	registry.Finish(session.ID, result)
	return json.Marshal(result)
}

func (t *ExecuteCodeTool) registry() *ProcessRegistry {
	if t != nil && t.Registry != nil {
		return t.Registry
	}
	if t == nil {
		return NewProcessRegistry()
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.defaultRegistry == nil {
		t.defaultRegistry = NewProcessRegistry()
	}
	return t.defaultRegistry
}

func (t *ExecuteCodeTool) getwd() (string, error) {
	if t != nil && t.Getwd != nil {
		return t.Getwd()
	}
	return os.Getwd()
}

func normalizeExecutionMode(mode string) (string, error) {
	if strings.TrimSpace(mode) == "" {
		return defaultExecuteCodeMode, nil
	}
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "strict":
		return "strict", nil
	case "project":
		return "project", nil
	default:
		return "", fmt.Errorf("unsupported mode %q", mode)
	}
}

func prepareExecutionWorkspace(mode, cwd, code string) (workspace string, runDir string, cmdArgs []string, cleanup func() error, err error) {
	switch mode {
	case "project":
		root := filepath.Join(cwd, projectExecutionScratchDir)
		workspace = filepath.Join(root, uuid.NewString())
		if err := os.MkdirAll(workspace, 0o755); err != nil {
			return "", "", nil, nil, err
		}
		mainPath := filepath.Join(workspace, "main.go")
		if err := os.WriteFile(mainPath, []byte(code), 0o600); err != nil {
			_ = os.RemoveAll(workspace)
			return "", "", nil, nil, err
		}
		return workspace, cwd, []string{"run", mainPath}, func() error {
			return os.RemoveAll(workspace)
		}, nil
	default:
		workspace, err = os.MkdirTemp("", "gormes-code-exec-*")
		if err != nil {
			return "", "", nil, nil, err
		}
		mainPath := filepath.Join(workspace, "main.go")
		if err := os.WriteFile(mainPath, []byte(code), 0o600); err != nil {
			_ = os.RemoveAll(workspace)
			return "", "", nil, nil, err
		}
		return workspace, workspace, []string{"run", "main.go"}, func() error {
			return os.RemoveAll(workspace)
		}, nil
	}
}

func scrubbedExecEnv() []string {
	env := os.Environ()
	out := make([]string, 0, len(env))
	for _, kv := range env {
		name, _, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if isSecretEnvName(name) {
			continue
		}
		out = append(out, kv)
	}
	sort.Strings(out)
	return out
}

func isSecretEnvName(name string) bool {
	upper := strings.ToUpper(strings.TrimSpace(name))
	for _, marker := range []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "CREDENTIAL", "PASSWD", "AUTH"} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}

func combineExecutionOutput(stdout, stderr string) string {
	switch {
	case stdout == "":
		return stderr
	case stderr == "":
		return stdout
	default:
		return stdout + "\n" + stderr
	}
}

func exitErr(err error) *exec.ExitError {
	var target *exec.ExitError
	if errors.As(err, &target) {
		return target
	}
	return nil
}

type cappedBuffer struct {
	limit     int
	buf       bytes.Buffer
	truncated bool
}

func newCappedBuffer(limit int) *cappedBuffer {
	return &cappedBuffer{limit: limit}
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		return len(p), nil
	}
	remaining := b.limit - b.buf.Len()
	if remaining > 0 {
		if len(p) > remaining {
			_, _ = b.buf.Write(p[:remaining])
			b.truncated = true
			return len(p), nil
		}
		_, _ = b.buf.Write(p)
		return len(p), nil
	}
	b.truncated = true
	return len(p), nil
}

func (b *cappedBuffer) String() string {
	if b == nil {
		return ""
	}
	if !b.truncated {
		return b.buf.String()
	}
	return b.buf.String() + truncatedOutputSentinel
}
