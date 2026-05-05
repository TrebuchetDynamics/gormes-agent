---
name: gormes-references
description: When stuck implementing a Go feature for Gormes, jump to the right donor file under references/go-agent-os/ before re-deriving from scratch. Hermes Python defines the parity contract; Go donors (goclaw, nanobot, plandex, engram, trpc-agent-go, adk-go, axe) supply working Go shapes. Goclaw code is permitted in Gormes with provenance; other donors stay patterns-only unless individually authorized.
---

# Gormes References

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

Use this skill whenever you are about to write Gormes Go code and you are not sure which shape to use, which library to consult, or how a similar agent system solved the same problem. It is a research/lookup skill — not a planner, builder, or TDD skill. After consulting references, hand off to `gormes-builder` / `gormes-tdd-slice` for implementation.

## When To Use

Trigger this skill when:

- you are about to call a Go stdlib API and want to see how a similar agent system structured it (HTTP client, OAuth, SSE, SQLite, MCP);
- a Gormes test or production failure surfaces a behavior gap and the implementation shape is unclear;
- the implementation seam needs OAuth / token-pool / provider-error / retry / streaming / tool-runtime / memory-store / await-user-reply / artifact-tracking;
- you are tempted to invent a new abstraction and there is probably already a working version in a donor;
- the failure mode looks like one of the **Recurring Pitfalls** in `references/go-agent-os/GORMES-PROVIDER-PATTERN-REFERENCES.md`.

Skip this skill when:

- the problem is upstream Hermes Python parity (use `gormes-parity-auditor` first);
- the problem is provider/auth/runtime specifically (use `gormes-provider-parity` — it is the focused superset);
- you already know the donor file and just need to write the test/code (use `gormes-tdd-slice`);
- the question is about external library/framework documentation (use Context7 MCP per repo policy).

## Mission

Reduce token waste, copy-paste errors, and re-invented abstractions by routing every Gormes implementation question to the smallest set of donor files that answer it.

Hermes Python remains the parity contract. Go donors are how Gormes _shapes_ Hermes parity in idiomatic Go.
The local Hermes source is normally `./hermes-agent` inside this repository;
fall back to `../hermes-agent` only when absent. Resolve it as `$HERMES_SRC`
before reading upstream behavior.

## Permission Map (2026-04-29)

| Donor | Permission for Gormes |
|---|---|
| `goclaw` (CC BY-NC 4.0 in upstream) | **Code permitted** — Juan granted explicit permission. Port with provenance comment. |
| `nanobot` (Apache 2.0) | Patterns + adapted code with attribution |
| `plandex` (MIT/AGPL — verify per file) | Patterns; copy code only after per-file license check |
| `engram` (MIT) | Patterns + adapted code with attribution |
| `trpc-agent-go` (Apache 2.0) | Patterns + adapted code with attribution |
| `adk-go` (Apache 2.0) | Patterns + adapted code with attribution |
| `axe` (MIT) | Patterns + adapted code with attribution |
| `agentcontrolplane`, `uzi` | Patterns only; no code copy without review |

When porting code (any donor), add a provenance comment on the receiving Gormes file:

```go
// Adapted from <donor>/path/to/file.go::SymbolName
// Reason: <why we needed this shape>
```

Convert types and imports to Gormes-native names; never let donor symbol names leak into Gormes' public API.

## Workflow

### 1. Identify The Problem Class

Pick exactly one and continue:

- **Provider / auth / streaming / quota / retry** → use `gormes-provider-parity` (focused superset). The provider table in `references/go-agent-os/GORMES-PROVIDER-PATTERN-REFERENCES.md` already maps the donor files.
- **MCP / tool host / runtime wiring / tool-result truncation** → see "Runtime & tools" donor map below.
- **Memory / Goncho / SQLite + FTS5 / write-queue / relations** → see "Memory & Goncho" donor map.
- **Channels / await-user-reply / callbacks / workflow primitives** → see "Channels & workflow" donor map.
- **Small utilities (token budget, artifact tracker, path sanitization)** → see "Utility primitives" donor map.
- **None of the above** → use the fallback grep below.

### 2. Read Hermes Python First

Even when Gormes ships its own Go implementation, the user-facing behavior must match Hermes. Anchor the parity target before reading donors:

- `$HERMES_SRC/AGENTS.md`
- relevant subsystem under `$HERMES_SRC/`

If Hermes has no analog (e.g. a brand-new Gormes-only seam), record this explicitly in the receiving row's `provenance.origin_type: gormes`.

### 3. Read The Donor File End-To-End

Open the donor file once and skim it whole before extracting any pattern. Donor files often hide invariants in helpers (e.g. path sanitization, header construction, jittered backoff) that are easy to miss when grepping.

### 4. Adapt, Don't Adopt

Translate the donor pattern into Gormes-native names, types, and tests. Never:

- import donor packages into Gormes;
- keep donor symbol names in public Gormes APIs;
- skip Gormes tests because the donor is tested;
- adopt donor UX/config choices that contradict Hermes parity.

Always:

- add a provenance comment naming the donor file/function;
- write Gormes RED tests covering the ported behavior;
- run focused gates per row + `go test ./...` + `go run ./cmd/progress validate`.

### 5. Hand Off

After lookup, hand off to:

- `gormes-tdd-slice` for the red-green-refactor loop;
- `gormes-builder` to land the row;
- `gormes-interface-designer` if the package boundary is now clearer but not finalized.

## Donor Maps By Subsystem

The provider/auth/streaming map already lives in `references/go-agent-os/GORMES-PROVIDER-PATTERN-REFERENCES.md`. The maps below cover everything else.

### Runtime & tools

| Gormes problem | Donor file | Pattern |
|---|---|---|
| Wire a runtime with explicit dependency layering | `nanobot/pkg/runtime/runtime.go` | `Options.Merge` + `Complete` defaults + explicit wiring of LLM client, tool registry, sampler |
| Build a tool service with before/after hooks | `nanobot/pkg/tools/service.go`, `nanobot/pkg/tools/flows.go` | Tool registry + filter gates + hook ordering |
| Truncate large tool outputs while persisting full bytes | `nanobot/pkg/agents/truncate.go` | `maxToolResultSize`, sanitized session-local artifact path, short text pointer |
| Estimate image tokens by decoded dimensions | `nanobot/pkg/agents/tokencount.go` | WebP decode, per-provider estimate, conservative fallback |
| Loop / sequential / parallel workflow agents | `adk-go/agent/workflowagents/...`, examples under `adk-go/examples/workflowagents/...` | Workflow primitives without rewriting kernel |
| Tool filtering (include/exclude by channel/trust/toolset) | `axe/internal/...` (filter helpers), `nanobot/pkg/tools/flows.go` | Pre-call gate, declarative filter |

### Memory & Goncho

| Gormes problem | Donor file | Pattern |
|---|---|---|
| SQLite + FTS5 memory store | `engram/internal/store/store.go` | DDL, indexes, migration helpers |
| Memory relation/conflict vocabulary | `engram/internal/store/relations.go` | `related`, `conflicts_with`, `supersedes`, `compatible`, `scoped`, `not_conflict` |
| Deterministic serialized MCP write queue | `engram/internal/mcp/write_queue.go` | Mutex-serialized queue, cancel-before-start, evidence vocabulary |
| MCP activity/audit logging | `engram/internal/mcp/activity.go` | Audit shape, redaction, append-only file |

### Channels & workflow

| Gormes problem | Donor file | Pattern |
|---|---|---|
| Await-user-reply / pause-and-resume | `trpc-agent-go/agent/await_user_reply.go` | Resume-token contract, callback-context shape, no-leak design |
| Provider/agent callback pipeline | `trpc-agent-go/model/callbacks.go`, `trpc-agent-go/agent/callbacks.go` | Before/after split from core turn |
| Per-session worker fan-out | `uzi` (patterns only) | Tmux fanout, isolated session state; do not use git worktrees for Gormes work |

### Utility primitives

| Gormes problem | Donor file | Pattern |
|---|---|---|
| Bounded token-budget tracker | `axe/internal/budget/budget.go` | Per-turn counter, reset, overflow signal |
| Artifact tracker with sanitized paths | `axe/internal/artifact/tracker.go` | Path-traversal guard, append-only registry |
| Test-budget / agent-step counter | `axe/internal/budget/budget_test.go` | Test shape Gormes can mirror |

### Provider / auth / streaming

The full table lives in `references/go-agent-os/GORMES-PROVIDER-PATTERN-REFERENCES.md` ("Quick Lookup: Problem → Donor File"). Always consult that note for OAuth, token refresh, error classification, retry/backoff, drift, and stream-error mapping.

## Fallback Grep

When the table doesn't list the problem:

```sh
# Substring-search every donor implementation. Adjust the pattern to match
# your problem (function name, header, error code, struct field).
grep -rn "<symbol-or-concept>" \
  /home/xel/git/sages-openclaw/workspace-mineru/references/go-agent-os/{goclaw,nanobot,plandex,engram,trpc-agent-go,adk-go,axe} \
  2>/dev/null | head -20
```

If the grep returns nothing useful, escalate to `gormes-skill-manager` to decide whether the missing donor is a real gap (file a new reference clone request) or whether Gormes needs a from-scratch interface (route to `gormes-interface-designer`).

## Validation

This is a research skill, so validation focuses on the implementation that follows:

- Every ported snippet has a `// Adapted from …` provenance comment.
- The receiving Gormes test exists before the implementation lands (RED).
- `go test ./<receiving-package> -count=1` is green.
- `go test ./... -count=1` and `go run ./cmd/progress validate` pass before commit.
- No donor symbol name appears in Gormes' exported API (`go doc ./<package> | grep -iE "(goclaw|nanobot|plandex|engram|trpc|adk|axe)"` returns nothing).

## Final Report Shape

```text
problem_class:
hermes_parity_target:
donor_files_consulted:
adaptation_applied:
provenance_comment:
gormes_tests_added:
hand_off_skill: gormes-tdd-slice | gormes-builder | gormes-interface-designer
```
