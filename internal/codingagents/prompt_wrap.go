package codingagents

import (
	"fmt"
	"strings"
)

// gormesRepoMarker is the substring used to detect a gormes-agent workspace
// so the wrapper can inject repo-specific rules.
const gormesRepoMarker = "/gormes-agent"

// WrapPrompt produces the standard wrapper sent to every coding-agent
// worker. The wrapper restates the workspace, mode, and task and injects
// repo-specific rules when running against the gormes-agent checkout so the
// worker honors the development-branch invariant.
func WrapPrompt(req CodingAgentRequest) string {
	mode := req.Mode
	if mode == "" {
		mode = ModeExplain
	}
	var b strings.Builder
	b.WriteString("You are being run by gormes as a coding worker.\n")
	fmt.Fprintf(&b, "Workspace: %s\n", req.Workspace)
	fmt.Fprintf(&b, "Mode: %s\n", mode)
	fmt.Fprintf(&b, "Edits allowed: %t\n", req.AllowEdits)
	b.WriteString("Task:\n")
	b.WriteString(strings.TrimSpace(req.Prompt))
	b.WriteString("\n\nRules:\n")
	b.WriteString(commonRules())
	if strings.Contains(req.Workspace, gormesRepoMarker) {
		b.WriteString("\n")
		b.WriteString(gormesRepoRules())
	}
	return b.String()
}

// commonRules lists the guardrails every coding-agent worker must respect
// regardless of which repository is under edit.
func commonRules() string {
	return strings.Join([]string{
		"- Stay inside the Workspace directory; never touch files above it.",
		"- Do not run destructive commands (rm -rf, force-push, history rewrites).",
		"- Do not exfiltrate secrets, credentials, or .env contents.",
		"- Honor the Mode: planning/review/explain modes must not modify files.",
		"- Prefer surgical edits; avoid unrelated refactors.",
	}, "\n")
}

// gormesRepoRules is appended only when the workspace is a gormes-agent
// checkout. It mirrors the repository-wide development-branch invariant so
// the worker cannot create stray branches or worktrees.
func gormesRepoRules() string {
	return strings.Join([]string{
		"Gormes-repo rules:",
		"- Work directly on the existing development branch; do not create branches or worktrees.",
		"- Run go test ./... -count=1 and go run ./cmd/progress validate before declaring done.",
		"- Keep changes inside the write scope declared by the progress.json row.",
		"- Never edit files that grant your worker shell elevation.",
	}, "\n")
}
