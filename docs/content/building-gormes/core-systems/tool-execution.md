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
    Execute(ctx context.Context, input string) (string, error)
}
```

Every tool lives behind this interface. Schemas are Go structs — schema drift is a compile error, not a silent agent-loop failure.

## What you get

- **Deterministic execution** — no subprocess spawning for in-process tools
- **Bounded side effects** — ctx cancels; deadlines respected
- **Wire Doctor** — `gormes doctor --offline` validates the registry before a live turn burns tokens

## Status

✅ Shipped (Phase 2.A), with Phase 5.K now extending the registry to include a guarded `execute_code` tool. The current Go tool set still avoids broad terminal/file mutation surfaces: `execute_code` runs local `sh`/`python` snippets with timeout/output caps and pre-exec filesystem/network blocking, while the wider sandbox backend matrix stays in Phase 5.B and dangerous-action approval remains Phase 5.J. Cron job management remains an operator-tool parity task in Phase 5.N even though the Phase 2.D scheduler/audit bridge is shipped. See [Phase 2](../../architecture_plan/phase-2-gateway/) and [Phase 5](../../architecture_plan/phase-5-final-purge/).

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
