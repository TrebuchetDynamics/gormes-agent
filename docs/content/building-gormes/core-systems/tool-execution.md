---
title: "Tool Execution"
weight: 40
---

# Tool Execution

Typed Go interfaces. In-process registry. No Python bounce.

## The contract

```go
type Tool interface {
    Name() string
    Description() string
    Schema() json.RawMessage
    Timeout() time.Duration
    Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
}
```

Every model-visible native tool lives behind this interface. The registry owns
the descriptor sent to the provider, the timeout used by the kernel, and the
JSON payload returned as a role=tool result.

## What you get

- **Deterministic execution** — no subprocess spawning for in-process tools
- **Bounded side effects** — ctx cancels; deadlines respected
- **Hermes-visible progress** — started tool calls produce channel/TUI progress
  through the shared tool-trace renderer, separate from assistant final text
- **Registry proof** — default tool exposure is asserted in `cmd/gormes`
  registry tests before a model can rely on it
- **Wire Doctor** — `gormes doctor --offline` validates the registry before a live turn burns tokens

## Status

✅ Shipped in layers:

- Phase 2.A established the typed in-process registry and kernel tool-call
  handshake.
- Phase 5.K added guarded, shell-only `execute_code` snippets with
  timeout/output caps, pre-exec filesystem/network blocking, and Python runtime
  hardline blocking at the terminal boundary.
- Phase 5.L now exposes the first Hermes-style local task tools by default:
  `read_file`, `search_files`, `write_file`, `patch`, and a foreground-only
  guarded `terminal`.
- Phase 5.N has native `todo` and `session_search` implementations, but their
  default live-turn registration still needs session-aware binding before they
  should be treated like stateless registry entries.

The important parity distinction is that **tool-progress rendering and tool
execution inventory are different contracts**. Gormes can render
`📖 read_file: "..."` only if a tool-start event exists, but the model can
actually choose `read_file` only when the descriptor is registered in the
runtime registry. Both sides need tests.

Remaining gaps are intentionally narrower: terminal background process
registry, PTY execution, sandbox backends, fuzzy patch/lint/checkpoint restore,
session-scoped `todo` registration, and session-bound `session_search`
registration. See [Phase 2](../../architecture_plan/phase-2-gateway/) and
[Phase 5](../../architecture_plan/phase-5-final-purge/).

## Native Tool Families

| Family | Current state | Remaining parity |
|---|---|---|
| `execute_code` | Guarded local shell snippets with caps and blocking policy; Python runtimes are not exposed | project/strict modes, helper parity without a Python child runtime, broader sandbox backends |
| `read_file` | Workspace-rooted text read, line-numbered pagination, duplicate-read status suppression, symlink escape denial | full Hermes file-state integration and checkpoint restore linkage |
| `search_files` | Workspace-rooted content and file-name search | ripgrep-level output modes, backend cwd tracking, large-result artifact storage |
| `write_file` / `patch` | Workspace-rooted full writes and replace-mode patch with read-status guardrails | fuzzy matching strategies, V4A multi-file patch mode, syntax/lint hooks, rollback |
| `terminal` | Foreground shell command with dangerous-command guardrails, timeouts, and output caps | background process registry, PTY, persistent cwd/env snapshots, Docker/Modal/SSH/Daytona/Singularity backends |
| `todo` | Native stateful tool and prompt-injection formatter | live per-session store binding in default registry |
| `session_search` | Native descriptor and execution wrapper over memory/session stores | current-session binding and safe default registration |

## Donor pointers

When implementing a new tool, tool-output policy, or runtime seam, route
through the `gormes-references` skill (`docs/development-skills/gormes-references/SKILL.md`)
before re-deriving a shape. The most useful Go donors for tools/runtime work:

| Tool/runtime problem | Donor file |
|---|---|
| Tool registry with before/after hooks and filter gates | `nanobot/pkg/tools/service.go`, `nanobot/pkg/tools/flows.go` |
| Truncate large tool outputs while persisting full bytes (artifact pointer + short text) | `nanobot/pkg/agents/truncate.go` |
| Estimate image tokens by decoded dimensions (per-provider conservative fallback) | `nanobot/pkg/agents/tokencount.go` |
| Wire a runtime with explicit dependency layering (`Options.Merge` + `Complete` defaults) | `nanobot/pkg/runtime/runtime.go` |
| Tool filtering by channel/trust/toolset (declarative pre-call gate) | `nanobot/pkg/tools/flows.go`, helpers under `axe/internal/tool/` |
| Per-turn token-budget tracker with reset and overflow signal | `axe/internal/budget/budget.go` |
| Artifact tracker for tool outputs (sanitized paths, append-only registry) | `axe/internal/artifact/tracker.go` |
| Loop / sequential / parallel workflow agents (no kernel rewrite) | `adk-go/agent/workflowagents/...`, examples under `adk-go/examples/workflowagents/...` |

Nanobot is Apache 2.0 and adk-go is Apache 2.0 — both permitted as patterns +
adapted code with attribution. Axe is MIT, same. Always add a
`// Adapted from <donor>/...::Symbol` comment on the receiving Gormes file when
porting code.
