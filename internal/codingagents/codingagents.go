// Package codingagents provides scaffolding for delegating coding tasks to
// external agent binaries such as codex, claude-code, and opencode.
//
// Phase 1 of the coding-agent delegation feature: shared interface and
// supporting types only. Adapter implementations and tool registration land
// in later phases.
package codingagents

import (
	"context"
	"time"
)

// Mode identifies how a coding agent should approach a task. The mode is
// passed to adapters so they can apply backend-specific flags (for example,
// codex review vs. apply modes).
type Mode string

const (
	// ModePlan asks the worker to outline an approach without making edits.
	ModePlan Mode = "plan"
	// ModeEdit asks the worker to make code edits in the workspace.
	ModeEdit Mode = "edit"
	// ModeTest asks the worker to add or update tests for the workspace.
	ModeTest Mode = "test"
	// ModeReview asks the worker to review code without modifying it.
	ModeReview Mode = "review"
	// ModeExplain asks the worker to explain code without modifying it.
	ModeExplain Mode = "explain"
)

// String returns the string form of the mode.
func (m Mode) String() string { return string(m) }

// Valid reports whether the mode is one of the recognized constants.
func (m Mode) Valid() bool {
	switch m {
	case ModePlan, ModeEdit, ModeTest, ModeReview, ModeExplain:
		return true
	}
	return false
}

// CodingAgentRequest captures the inputs the kernel sends to a coding worker.
// Workspace is the absolute, resolved directory the worker may modify (or
// inspect for read-only modes). Prompt is the task description.
type CodingAgentRequest struct {
	// Workspace is the resolved absolute path where the worker runs. It must
	// already have been vetted by WorkspaceGuard.
	Workspace string
	// Prompt is the human-language task description for the worker.
	Prompt string
	// Mode selects the worker's posture (plan/edit/test/review/explain).
	Mode Mode
	// AllowEdits indicates whether the worker may modify files in the
	// workspace. Plan/review/explain modes typically pass false.
	AllowEdits bool
	// Timeout bounds the worker invocation. Zero means the adapter chooses a
	// reasonable default.
	Timeout time.Duration
	// ExtraArgs allows callers to pass adapter-specific arguments without
	// forcing them through the structured fields.
	ExtraArgs []string
}

// CodingAgentResult is the structured response from a coding worker. The
// shape is intentionally backend-agnostic so the kernel can compare runs
// across codex, claude-code, and opencode.
type CodingAgentResult struct {
	// Backend names the worker that produced this result (for example
	// "codex", "claude-code", "opencode").
	Backend string
	// ExitCode is the worker's process exit code.
	ExitCode int
	// Stdout is the captured standard output of the worker.
	Stdout string
	// Stderr is the captured standard error of the worker.
	Stderr string
	// FilesChanged lists workspace-relative paths that the worker modified,
	// derived from git diff output.
	FilesChanged []string
	// GitDiff is the unified diff produced by comparing the snapshot before
	// and after the worker ran. Empty when no edits occurred.
	GitDiff string
	// Duration is how long the worker invocation took.
	Duration time.Duration
}

// CodingAgent is the contract every backend adapter implements. Adapters are
// expected to honor the request's workspace, mode, edit permissions, and
// timeout, and to populate the result with stdout, stderr, files changed,
// and a git diff captured by the shared snapshot helper.
type CodingAgent interface {
	// Name returns the backend identifier (matches CodingAgentResult.Backend).
	Name() string
	// Run executes the worker against the request and returns the structured
	// result. It must not modify files outside req.Workspace.
	Run(ctx context.Context, req CodingAgentRequest) (*CodingAgentResult, error)
}
