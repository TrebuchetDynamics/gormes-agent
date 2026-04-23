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
- **Sandboxed code execution** — `execute_code` stages self-contained Go snippets in `strict` temp workspaces or `project` workspaces under `.gormes/code-execution`, strips secret-bearing env vars, and returns structured stdout/stderr results
- **Bounded side effects** — ctx cancels; deadlines respected
- **Fail-closed dangerous action gate** — command-like tool args are scanned for destructive shell payloads before `Tool.Execute` runs
- **Wire Doctor** — `gormes doctor --offline` validates the registry before a live turn burns tokens

## Status

✅ Shipped (Phase 2.A + Phase 5.K tracked scope). The registry still executes most tools in-process, and the broader execution surface now includes Go-native `execute_code` with scrubbed `strict`/`project` workspaces, timeout/output caps, and structured results flowing through the same kernel tool loop. Dangerous shell payloads in command-like JSON fields still fail closed before execution in both the kernel and the in-process executor. Interactive terminal-style debug sessions and recursive in-script tool RPC remain future follow-on scope. See [Phase 2](../architecture_plan/phase-2-gateway/).
