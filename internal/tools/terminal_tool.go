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
	"regexp"
	"strings"
	"time"
)

const (
	defaultTerminalTimeout = 180 * time.Second
	maxTerminalToolTimeout = 30 * time.Minute
	defaultTerminalMaxOut  = 100 * 1024
)

// TerminalToolConfig configures the local terminal tool. This initial Go port
// supports foreground commands and Hermes-style dangerous command guardrails.
type TerminalToolConfig struct {
	Workdir        string
	ApprovalMode   string
	DefaultTimeout time.Duration
	MaxOutputBytes int
}

type TerminalTool struct {
	cfg TerminalToolConfig
}

func NewTerminalTool(cfg TerminalToolConfig) *TerminalTool {
	return &TerminalTool{cfg: cfg}
}

func (*TerminalTool) Name() string { return "terminal" }

func (*TerminalTool) Description() string {
	return "Execute a local shell command with timeout handling and dangerous-command guardrails. Reserve this for builds, git, processes, scripts, package managers, and commands that need a shell."
}

func (*TerminalTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"command":{"type":"string","description":"Command to execute."},"background":{"type":"boolean","description":"Run as a background process. This Go port currently rejects background=true; use tmux or a shell-managed daemon command when needed.","default":false},"timeout":{"type":"integer","description":"Maximum seconds to wait. Defaults to 180.","minimum":1},"workdir":{"type":"string","description":"Working directory for this command."},"pty":{"type":"boolean","description":"Accepted for Hermes schema compatibility. PTY execution is not available in this initial Go port.","default":false}},"required":["command"]}`)
}

func (t *TerminalTool) Timeout() time.Duration {
	return maxTerminalToolTimeout + 5*time.Second
}

func (t *TerminalTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Command    string `json:"command"`
		Background bool   `json:"background"`
		Timeout    int    `json:"timeout"`
		Workdir    string `json:"workdir"`
		PTY        bool   `json:"pty"`
	}
	if err := json.Unmarshal(defaultJSONArgs(args), &in); err != nil {
		return marshalToolPayload(terminalResult{Status: "error", ExitCode: -1, Error: "invalid terminal args: " + err.Error()})
	}
	if strings.TrimSpace(in.Command) == "" {
		return marshalToolPayload(terminalResult{Status: "error", ExitCode: -1, Error: "terminal command is required"})
	}
	if in.Background {
		return marshalToolPayload(terminalResult{
			Status:   "unsupported",
			ExitCode: -1,
			Error:    "background terminal processes are not ported yet; use tmux or run a foreground command with timeout",
		})
	}

	approvalMode := strings.TrimSpace(t.cfg.ApprovalMode)
	if approvalMode == "" {
		approvalMode = ApprovalModeManual
	}
	guard := GuardCommandWithApproval(ctx, "terminal", in.Command, approvalMode)
	if guard.Description != "" && !guard.Approved {
		status := "blocked"
		if guard.ApprovalRequired {
			status = "approval_required"
		}
		return marshalToolPayload(terminalResult{
			Status:      status,
			ExitCode:    -1,
			Error:       fmt.Sprintf("Command denied: %s", guard.Description),
			Command:     in.Command,
			Description: guard.Description,
		})
	}

	workdir, err := terminalWorkdir(t.cfg.Workdir, in.Workdir)
	if err != nil {
		return marshalToolPayload(terminalResult{Status: "error", ExitCode: -1, Error: err.Error(), Command: in.Command})
	}
	timeout := t.cfg.DefaultTimeout
	if timeout <= 0 {
		timeout = defaultTerminalTimeout
	}
	if in.Timeout > 0 {
		timeout = time.Duration(in.Timeout) * time.Second
	}
	maxOutput := t.cfg.MaxOutputBytes
	if maxOutput <= 0 {
		maxOutput = defaultTerminalMaxOut
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	cmd := exec.CommandContext(runCtx, "bash", "-lc", in.Command)
	cmd.Dir = workdir.Path
	cmd.Env = os.Environ()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	duration := time.Since(start)

	result := terminalResult{
		Status:     "completed",
		Command:    in.Command,
		Workdir:    workdir.Path,
		ExitCode:   0,
		Stdout:     stripANSI(stdout.String()),
		Stderr:     stripANSI(stderr.String()),
		DurationMs: duration.Milliseconds(),
	}
	if workdir.Recovered {
		result.CWDRecovered = true
		result.CWDRecovery = "terminal_cwd_recovered: working directory was missing; using nearest existing directory"
	}
	if in.PTY {
		result.PTYNote = "pty=true was accepted for schema compatibility but executed without a PTY in this Go port"
	}
	if runCtx.Err() != nil && errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		result.Status = "timeout"
		result.ExitCode = 124
		result.Error = fmt.Sprintf("Command timed out after %d seconds", int(timeout.Seconds()))
	} else if err != nil {
		result.Status = "failed"
		result.ExitCode = terminalExitCode(err)
		result.Error = err.Error()
	}
	result.Output = strings.TrimSpace(strings.TrimSpace(result.Stdout) + "\n" + strings.TrimSpace(result.Stderr))
	result.Output, result.Truncated = truncateText(result.Output, maxOutput)
	result.Stdout, _ = truncateText(result.Stdout, maxOutput)
	result.Stderr, _ = truncateText(result.Stderr, maxOutput)
	return marshalToolPayload(result)
}

type terminalResult struct {
	Status       string `json:"status"`
	Command      string `json:"command,omitempty"`
	Workdir      string `json:"workdir,omitempty"`
	Output       string `json:"output,omitempty"`
	Stdout       string `json:"stdout,omitempty"`
	Stderr       string `json:"stderr,omitempty"`
	ExitCode     int    `json:"exit_code"`
	Error        string `json:"error,omitempty"`
	Description  string `json:"description,omitempty"`
	DurationMs   int64  `json:"duration_ms,omitempty"`
	Truncated    bool   `json:"truncated,omitempty"`
	PTYNote      string `json:"pty_note,omitempty"`
	CWDRecovered bool   `json:"cwd_recovered,omitempty"`
	CWDRecovery  string `json:"cwd_recovery,omitempty"`
}

type terminalWorkdirResult struct {
	Path      string
	Recovered bool
}

func terminalWorkdir(defaultWorkdir, requested string) (terminalWorkdirResult, error) {
	workdir := strings.TrimSpace(defaultWorkdir)
	if workdir == "" || terminalCWDPlaceholder(workdir) {
		if envWorkdir := strings.TrimSpace(os.Getenv("TERMINAL_CWD")); envWorkdir != "" && !terminalCWDPlaceholder(envWorkdir) {
			workdir = envWorkdir
		}
	}
	if workdir == "" || terminalCWDPlaceholder(workdir) {
		cwd, err := os.Getwd()
		if err != nil {
			return terminalWorkdirResult{Path: os.TempDir(), Recovered: true}, nil
		}
		workdir = cwd
	}
	if strings.HasPrefix(workdir, "~") {
		expanded, err := expandUserPath(workdir)
		if err != nil {
			return terminalWorkdirResult{}, err
		}
		workdir = expanded
	} else if !filepath.IsAbs(workdir) {
		cwd, err := os.Getwd()
		if err != nil {
			workdir = filepath.Join(os.TempDir(), workdir)
		} else {
			workdir = filepath.Join(cwd, workdir)
		}
	}
	defaultMissingRecoveryAllowed := strings.TrimSpace(requested) == ""
	if strings.TrimSpace(requested) != "" {
		if filepathishDangerous(requested) {
			return terminalWorkdirResult{}, fmt.Errorf("workdir contains invalid control characters")
		}
		if strings.HasPrefix(requested, "~") {
			expanded, err := expandUserPath(requested)
			if err != nil {
				return terminalWorkdirResult{}, err
			}
			workdir = expanded
		} else if filepath.IsAbs(requested) {
			workdir = requested
		} else {
			workdir = filepath.Join(workdir, requested)
		}
	}
	info, err := os.Stat(workdir)
	if err != nil {
		if defaultMissingRecoveryAllowed {
			return terminalWorkdirResult{Path: nearestExistingTerminalDir(workdir), Recovered: true}, nil
		}
		return terminalWorkdirResult{}, fmt.Errorf("resolve working directory: %w", err)
	}
	if !info.IsDir() {
		return terminalWorkdirResult{}, fmt.Errorf("workdir %q is not a directory", workdir)
	}
	return terminalWorkdirResult{Path: workdir}, nil
}

func nearestExistingTerminalDir(path string) string {
	path = filepath.Clean(path)
	for candidate := path; candidate != "" && candidate != "."; candidate = filepath.Dir(candidate) {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			break
		}
	}
	return os.TempDir()
}

func terminalCWDPlaceholder(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case ".", "auto", "cwd":
		return true
	default:
		return false
	}
}

func terminalExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func truncateText(s string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s, false
	}
	head := maxBytes * 2 / 5
	tail := maxBytes - head
	omitted := len(s) - head - tail
	return s[:head] + fmt.Sprintf("\n\n... [OUTPUT TRUNCATED - %d bytes omitted] ...\n\n", omitted) + s[len(s)-tail:], true
}

func filepathishDangerous(s string) bool {
	return strings.ContainsRune(s, 0)
}
