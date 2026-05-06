// Package doctor runs diagnostic checks against a constructed Gormes runtime.
// Each Check returns a CheckResult that cmd/gormes/doctor renders to stdout.
package doctor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Status enumerates the possible outcomes of a diagnostic check.
type Status int

const (
	StatusPass Status = iota
	StatusFail
	StatusWarn
)

func (s Status) String() string {
	switch s {
	case StatusPass:
		return "PASS"
	case StatusFail:
		return "FAIL"
	case StatusWarn:
		return "WARN"
	}
	return "UNKNOWN"
}

// Symbol returns a compact glyph for console output.
func (s Status) Symbol() string {
	switch s {
	case StatusPass:
		return "✓"
	case StatusFail:
		return "✗"
	case StatusWarn:
		return "!"
	}
	return "?"
}

// CheckResult is the output of one diagnostic check.
type CheckResult struct {
	Name    string // short label, e.g. "Toolbox"
	Status  Status
	Summary string     // one-line headline
	Items   []ItemInfo // optional per-entry detail
}

// ItemInfo is a per-tool (or per-entry) row rendered under the headline.
type ItemInfo struct {
	Name   string
	Status Status
	Note   string // description on pass; error detail on fail
}

// ErrGitHubCLIMissing identifies a missing gh binary without exposing PATH or
// shell error text in user-visible doctor output.
var ErrGitHubCLIMissing = errors.New("github cli missing")

// GitHubAuthStatusResult is the sanitized result of `gh auth status`.
type GitHubAuthStatusResult struct {
	ExitCode int
	Err      error
	TimedOut bool
}

// GitHubAuthStatusRunner probes the GitHub CLI auth state. Tests inject this
// seam so doctor never touches the host gh binary.
type GitHubAuthStatusRunner func(context.Context) GitHubAuthStatusResult

// GitHubAuthOptions contains the environment and command seam used by
// CheckGitHubAuth.
type GitHubAuthOptions struct {
	Env             map[string]string
	RunGHAuthStatus GitHubAuthStatusRunner
}

// CheckGitHubAuth mirrors Hermes doctor behavior: an env token is sufficient,
// otherwise an authenticated gh CLI session is sufficient, otherwise the check
// warns with redacted degraded evidence.
func CheckGitHubAuth(ctx context.Context, opts GitHubAuthOptions) CheckResult {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(opts.Env["GITHUB_TOKEN"]) != "" {
		return githubAuthResult(StatusPass, "token configured", "github_token_env", "source=GITHUB_TOKEN")
	}
	if strings.TrimSpace(opts.Env["GH_TOKEN"]) != "" {
		return githubAuthResult(StatusPass, "token configured", "github_token_env", "source=GH_TOKEN")
	}

	runner := opts.RunGHAuthStatus
	if runner == nil {
		runner = DefaultGitHubAuthStatusRunner
	}
	probe := runner(ctx)
	switch {
	case probe.Err == nil && probe.ExitCode == 0:
		return githubAuthResult(StatusPass, "authenticated via gh CLI", "github_cli_authenticated", "command=gh auth status --json authenticated")
	case errors.Is(probe.Err, ErrGitHubCLIMissing):
		return githubAuthResult(StatusWarn, "No GITHUB_TOKEN and gh CLI missing", "github_cli_missing", "set GITHUB_TOKEN/GH_TOKEN or authenticate gh")
	case probe.TimedOut:
		return githubAuthResult(StatusWarn, "No GITHUB_TOKEN and gh auth timed out", "github_cli_timeout", "set GITHUB_TOKEN/GH_TOKEN or retry gh auth")
	case probe.Err != nil:
		return githubAuthResult(StatusWarn, "No GITHUB_TOKEN and gh auth status failed", "github_cli_status_failed", "set GITHUB_TOKEN/GH_TOKEN or authenticate gh")
	default:
		return githubAuthResult(StatusWarn, "No GITHUB_TOKEN and gh CLI is unauthenticated", "github_cli_unauthenticated", "set GITHUB_TOKEN/GH_TOKEN or run gh auth login")
	}
}

// DefaultGitHubAuthStatusRunner checks gh auth without capturing stdout/stderr;
// doctor only needs the success/failure class and must not print gh output.
func DefaultGitHubAuthStatusRunner(ctx context.Context) GitHubAuthStatusResult {
	if ctx == nil {
		ctx = context.Background()
	}
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(probeCtx, "gh", "auth", "status", "--json", "authenticated")
	err := cmd.Run()
	if probeCtx.Err() == context.DeadlineExceeded {
		return GitHubAuthStatusResult{ExitCode: -1, Err: probeCtx.Err(), TimedOut: true}
	}
	if err == nil {
		return GitHubAuthStatusResult{ExitCode: 0}
	}
	var execErr *exec.Error
	if errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound) {
		return GitHubAuthStatusResult{ExitCode: -1, Err: ErrGitHubCLIMissing}
	}
	if errors.Is(err, exec.ErrNotFound) {
		return GitHubAuthStatusResult{ExitCode: -1, Err: ErrGitHubCLIMissing}
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return GitHubAuthStatusResult{ExitCode: exitErr.ExitCode(), Err: err}
	}
	return GitHubAuthStatusResult{ExitCode: -1, Err: err}
}

func githubAuthResult(status Status, summary, evidence, note string) CheckResult {
	return CheckResult{
		Name:    "GitHub auth",
		Status:  status,
		Summary: summary + " evidence=" + evidence,
		Items: []ItemInfo{{
			Name:   "github",
			Status: status,
			Note:   "evidence=" + evidence + " " + note,
		}},
	}
}

// Format renders the CheckResult as a multi-line string suitable for
// `gormes doctor` stdout.
func (r CheckResult) Format() string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %s: %s\n", r.Status.String(), r.Name, r.Summary)
	if len(r.Items) == 0 {
		return b.String()
	}
	// Two-column formatting: widest name + status column.
	nameW := 0
	for _, it := range r.Items {
		if n := len(it.Name); n > nameW {
			nameW = n
		}
	}
	for _, it := range r.Items {
		fmt.Fprintf(&b, "  %s %-*s  %s\n", it.Status.Symbol(), nameW, it.Name, it.Note)
	}
	return b.String()
}
