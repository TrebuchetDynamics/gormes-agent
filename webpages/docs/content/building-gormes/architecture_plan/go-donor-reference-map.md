---
title: "Go Donor Reference Map"
weight: 17
---

# Go Donor Reference Map

Use this page when a planning row names a Go implementation shape but the
builder contract is still vague. Hermes Python remains the parity target;
these Go projects provide implementation patterns only.

The operational workflow is:

```text
Hermes contract -> smallest Go donor pattern -> progress.json row -> TDD slice
```

Do not use this page to bypass `progress.json`. If the donor study changes
implementation intent, update the canonical row with `source_refs`,
`write_scope`, `test_commands`, `acceptance`, and `provenance`.

## Permission Guardrail

| Donor | Gormes use |
|---|---|
| `goclaw` | Code may be ported with a provenance comment; Juan granted explicit Gormes permission. Preserve Hermes UX/config semantics even when the donor shape differs. |
| `nanobot` | Patterns and adapted code with attribution. Best for runtime wiring, tool host boundaries, truncation, and image token estimates. |
| `engram` | Patterns and adapted code with attribution. Best for SQLite/FTS5 memory, relation vocabulary, MCP write queues, and audit logging. |
| `trpc-agent-go` | Patterns and adapted code with attribution. Best for await-user-reply routing and callback separation. |
| `adk-go` | Patterns and adapted code with attribution. Best for loop/sequential/parallel workflow-agent primitives. |
| `plandex` | Patterns by default; verify per-file license before copying. Best for provider drift, retry, and error classification. |
| `axe` | Patterns and adapted code with attribution. Best for small token-budget and artifact-tracker utilities. |
| `agentcontrolplane`, `uzi` | Patterns only unless a separate review grants copy permission. |

When code is adapted from a donor, the receiving Go file needs a short
provenance comment naming the donor file and why the shape fits Gormes. Donor
package names must not leak into Gormes exported APIs.

## Donor Lookup By Plan Class

| Plan class | Hermes parity target | Go donor refs | What the row should freeze |
|---|---|---|---|
| Provider auth, OAuth, quota, retry, streaming errors | `../hermes-agent/agent/*`, provider setup/auth paths, gateway-visible provider errors | `references/go-agent-os/GORMES-PROVIDER-PATTERN-REFERENCES.md`, `goclaw/internal/oauth/openai.go`, `goclaw/internal/oauth/openai_quota_transport.go`, `plandex/app/server/model/model_error.go` | Resolver precedence, no empty base URLs, OAuth refresh/relogin evidence, Retry-After parsing, sanitized provider errors, hermetic `httptest` fixtures. |
| Runtime and MCP/tool host | Hermes tool registry/toolset semantics, MCP tool behavior, channel trust classes | `nanobot/pkg/runtime/runtime.go`, `nanobot/pkg/tools/service.go`, `nanobot/pkg/tools/flows.go`, `trpc-agent-go/tool/*` | Explicit options/default layering, tool declaration/filter/call boundary, before/after callbacks, unavailable-tool evidence, no adoption of Nanobot config semantics. |
| Tool-result budget and artifacts | Hermes tool-result envelopes and operator-visible truncation behavior | `nanobot/pkg/agents/truncate.go`, `nanobot/pkg/agents/tokencount.go`, `axe/internal/artifact/tracker.go`, `axe/internal/budget/budget.go` | Persist full oversized output, return bounded pointer text, sanitize paths, estimate image tokens by decoded dimensions, expose write failure as degraded evidence. |
| Context compression and prompt cache | `../hermes-agent/agent/context_compressor.py`, `agent/prompt_caching.py`, matching upstream tests | `nanobot/pkg/agents/tokencount.go`, `axe/internal/budget/budget.go`, provider-pattern notes for cache/error visibility | Token-budget tail selection, protected head messages, tool-call/result pairing, cache marker policy matrix, visible unsupported-provider stripping. |
| Goncho memory and Honcho-compatible local state | Honcho sessions/messages/search/memory contracts and Hermes memory tool behavior | `engram/internal/store/store.go`, `engram/internal/store/relations.go`, `engram/internal/mcp/write_queue.go`, `engram/internal/mcp/activity.go` | SQLite/FTS5 schema shape, relation verbs, deterministic serialized writes, panic/cancel/queue-full evidence, append-only redacted activity logs. |
| Await-user-reply tools and interruptible flows | Hermes `clarify`, approvals, TUI/gateway callbacks, cron/oneshot noninteractive behavior | `trpc-agent-go/agent/await_user_reply.go`, `trpc-agent-go/agent/callbacks.go`, `trpc-agent-go/model/callbacks.go` | One-shot resume token, route normalization, callback state split from core turn, timeout/unavailable output, route cleanup after the next user reply. |
| Workflow/subagent orchestration | Hermes delegation/subagent tool behavior, gateway active-turn policy, skill-builder rows | `adk-go/agent/workflowagents/...`, `agentcontrolplane/acp/internal/controller/*/state_machine.go`, `uzi/pkg/state/state.go` | Explicit turn/job states, local locks before distributed leases, cancellation, dependency checks, per-session state, no Kubernetes/tmux dependency in core runtime. |

## Row Checklist

Before promoting a row out of `umbrella`, answer all of these in the row:

1. Which Hermes file, symbol, command, fixture, or commit owns the user-visible
   behavior?
2. Which single Go donor file was read end to end, and what pattern is being
   adapted?
3. Which Gormes package owns the caller-facing interface?
4. What degraded evidence appears when the capability is unavailable?
5. What local fixture proves the behavior without live credentials, live
   providers, live platforms, or a real browser?
6. Does copied/adapted code require a provenance comment or a license check?

Rows that cannot answer these stay as umbrellas or route to
`gormes-interface-designer` for a package-boundary pass.
