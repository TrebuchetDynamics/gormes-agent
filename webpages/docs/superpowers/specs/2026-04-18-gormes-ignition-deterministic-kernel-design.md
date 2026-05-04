# Gormes — Ignition (M0 + M1) Deterministic Kernel Design

> **⚠️ SUPERSEDED — 2026-04-18 — MERGED INTO PHASE 1 ADAPTER**
>
> This spec's **disciplines** (single-owner kernel, bounded mailboxes, pull-based provider stream, render-frame coalescing, input admission, cancellation leak-freedom, runtime-seam-not-tool-seam) have been **absorbed into** the Phase 1 frontend-adapter spec.
>
> This spec's **architectural assumptions** (Go owns OpenRouter, Go owns SQLite, Go owns the full agent loop in M1) are NOT absorbed — they conflict with the post-recon Ship-of-Theseus strategy where Python's existing `api_server.py` (port 8642) serves the LLM and owns `state.db` through Phase 3.
>
> **Merged into:** [`2026-04-18-gormes-frontend-adapter-design.md`](./2026-04-18-gormes-frontend-adapter-design.md) — sections §3.6 (single-owner kernel), §7 (pull-based Stream), §8 (RenderFrame + bounded mailboxes), §11 (local admission), §12.5 (local provenance), §13 (runtime seam stub).
>
> This document is retained as the authoritative source for Phase-4 architecture. When Phase 4 arrives, Go absorbs the agent loop / context planner / persistence ownership from Python — at which point §4 (Kernel State Machine), §6 (Context Window Physics), §7 (4-table persistence schema), §8 (Runtime seam full implementation) of this spec become the Phase-4 implementation target. **Do not delete this file.**

---

**Date:** 2026-04-18
**Author:** Xel (via Codex brainstorm)
**Status:** SUPERSEDED for Phase 1 — RETAINED as Phase 4 architectural target
**Supersedes:** `2026-04-18-gormes-ignition-design.md`
**Scope:** Milestones M0 (scaffolding) + M1 (TUI + one LLM provider) of the Gormes program.

---

## 1. Purpose

This spec replaces the earlier ignition design with a lower-entropy foundation optimized for **minimum rewrite risk into M2-M4**. The governing decision is simple:

- M1 is not a demo shell.
- M1 is the deterministic kernel of the system.
- Future milestones attach to stable seams instead of rewriting the core loop.

The prior spec had the right program shape, but it left four foundation-level questions underspecified:

1. Backpressure between provider streaming, agent orchestration, TUI rendering, and SQLite writes.
2. Context-window admission control before a request reaches the provider.
3. Provenance boundaries needed for M2 ontological memory.
4. A process-runtime seam for M4, rather than a narrow in-process tool seam.

This superseding design closes those gaps.

---

## 2. Design Decision

Three architectural paths were considered:

1. **Patch the existing channel design.**
   Keep provider channels, flat turn persistence, and per-token UI updates; add a few queue sizes and guards.
   This is the smallest diff, but it preserves the core coupling mistakes.
2. **Deterministic actor kernel.**
   Make the agent core a single-owner state machine; treat provider, storage, and platform as bounded edge adapters; add context planning and a runtime seam in M1.
   This raises M1 rigor slightly, but sharply reduces M2-M4 rewrite pressure.
3. **Full event sourcing.**
   Model everything as append-only domain events with projections.
   This is architecturally clean but too heavy for ignition.

**Chosen approach:** option 2.

---

## 3. Non-Negotiable Principles

### 3.1 Single Owner of Conversational State

The agent kernel is the only owner of in-memory conversation state for an active session. No other goroutine mutates turn buffers, run phase, or render state.

### 3.2 Channels Are Edge Mailboxes, Not the Architecture

The earlier statement "all inter-actor communication uses channels" is too vague to be safe. In this design:

- channels exist only at system edges;
- every mailbox is bounded;
- each mailbox has explicit saturation behavior;
- the agent kernel remains the authority.

### 3.3 Admission Before Execution

No provider call is allowed until the request survives local validation:

- input size bounds;
- context budget planning;
- model capability checks;
- session/run state checks.

Provider-side `context-length` errors are treated as a bug in local planning, not as a normal control path.

### 3.4 Provenance First, Features Later

M2 triple extraction, M3 platform fan-out, and M4 Python runtime integration all depend on stable provenance:

- what turns existed;
- what exact subset was sent to the model;
- which run produced which assistant output;
- what provider metadata and finish reason came back.

M1 must record that provenance directly.

### 3.5 Runtime Seam, Not Tool Seam

Python integration will not be a naked `Tool.Call(...)` bridge. The process boundary needs lifecycle, health, discovery, invocation, cancellation, and streaming semantics.

---

## 4. Deterministic Kernel Model

### 4.1 Core Topology

Gormes M1 consists of one deterministic kernel plus three edge adapters:

```text
                 +---------------------------+
                 |     Agent Kernel          |
                 |   (single owner loop)     |
                 +---------------------------+
                    ^         ^         ^
                    |         |         |
           render frames   store acks   stream events
                    |         |         |
            +-------+--+   +--+------+  +----------------+
            | Platform |   |  Store   |  | ProviderStream |
            |   TUI    |   | SQLite   |  |  OpenRouter    |
            +----------+   +----------+  +----------------+
```

The kernel owns:

- session state;
- active run state;
- assistant draft buffer;
- render snapshot state;
- phase transitions;
- cancellation of the active turn.

The kernel does **not** own:

- terminal rendering internals;
- SQLite connection internals;
- provider HTTP/SSE internals.

### 4.2 Kernel State Machine

M1 defines a closed set of phases:

```text
Idle
  -> Planning
  -> StartingRun
  -> Streaming
  -> Finalizing
  -> Idle

Streaming -> Cancelling -> Finalizing
Any phase  -> Failed      -> Idle
```

Rules:

- only one user-submitted run may be active per session in M1;
- phase transitions are serialized inside the kernel loop;
- every emitted UI frame carries a monotonically increasing `Seq`;
- every persisted run carries a durable `RunID`.

### 4.3 Kernel Responsibilities

For each submitted turn, the kernel performs exactly this sequence:

1. Validate raw user input against byte and line-count limits.
2. Persist the user turn.
3. Build a context plan.
4. Persist a `runs` row in `state='planning'`, then `state='streaming'`.
5. Open the provider stream.
6. Consume stream events and update the in-memory assistant draft.
7. Emit coalesced render frames.
8. Finalize the assistant turn and run record.
9. Return to `Idle`.

No edge adapter is allowed to mutate kernel-owned state directly.

---

## 5. Concurrency and Backpressure

### 5.1 Problem in the Earlier Spec

The superseded design streamed provider `Delta` values over a buffered channel and emitted one UI update per token. That leaves four failure modes underdefined:

1. provider outpaces UI rendering;
2. SQLite stalls during turn finalization;
3. cancellation races with a blocked channel send;
4. queues grow without a formal saturation policy.

The fix is not "more channels." The fix is **bounded mailboxes plus latest-state rendering**.

### 5.2 Edge Mailboxes

M1 uses only these bounded mailboxes:

1. **Render mailbox**
   Carries full render snapshots from kernel to TUI.
   Capacity: `1`.
   Saturation policy: replace older unsent snapshot with the newest snapshot.
2. **Store command mailbox**
   Carries persistence commands from kernel to store adapter.
   Capacity: fixed small bound, e.g. `16`.
   Saturation policy: kernel stops new work and enters `Failed` if an ack deadline is exceeded.

There are no unbounded queues in M1.

### 5.3 Provider Interface

The provider seam changes from a raw channel-returning function to a stream object:

```go
type Provider interface {
    Name() string
    Capabilities(ctx context.Context, model string) (Capabilities, error)
    OpenStream(ctx context.Context, req Request) (Stream, error)
}

type Stream interface {
    Recv(ctx context.Context) (StreamEvent, error)
    Close() error
}
```

Why this is better than `Stream(...) <-chan Delta`:

- pull-based consumption is explicit;
- shutdown responsibility is explicit;
- cancellation and EOF become regular control flow;
- the kernel can pace consumption instead of passively absorbing firehose pressure.

If a concrete provider implementation uses a background goroutine to read SSE frames ahead of `Recv`, that adapter-local queue must also be bounded with a small fixed capacity. The bound is an implementation detail, but unbounded provider-side buffering is forbidden.

### 5.4 Render Frames Replace Per-Token UI Messages

The TUI does not need every token as a discrete mailbox event. It needs the latest visible state.

Replace token-by-token UI emission with:

```go
type RenderFrame struct {
    Seq         uint64
    Phase       Phase
    DraftText   string
    Telemetry   TelemetrySnapshot
    StatusText  string
    SessionID   string
    Model       string
    LastError   string
}
```

Kernel policy:

- coalesce provider text into an in-memory draft buffer;
- flush a new `RenderFrame` at most every `16ms`;
- flush immediately on semantic edges:
  `StartingRun`, `Streaming`, `Cancelling`, `Finalizing`, `Failed`, `Idle`.

If the TUI cannot keep up, stale frames are dropped and only the newest frame survives.

### 5.5 Store Adapter Contract

SQLite is isolated behind a store adapter:

```go
type Store interface {
    Exec(ctx context.Context, cmd Command) (Ack, error)
}
```

Important rule:

- the kernel never writes to `database/sql` directly;
- the store adapter serializes writes;
- every persistence step has a deadline;
- if the store does not ack within the deadline, the kernel stops admitting new work.

This keeps SQLite latency from turning into silent goroutine accumulation.

### 5.6 Cancellation and Leak Freedom

Each active run gets its own child context:

```go
runCtx, cancelRun := context.WithCancel(rootCtx)
```

Cancellation rules:

- `Ctrl+C` during `Streaming` moves the kernel to `Cancelling`;
- the kernel cancels `runCtx`;
- provider reader exits on `runCtx.Done()`;
- store finalization uses a short dedicated deadline;
- every send on every mailbox is wrapped in `select { case ...; case <-ctx.Done(): }`.

Success criteria for M1 include:

- no goroutine leak after cancel;
- no mailbox growth after provider overrun tests;
- no process hang when store acks are delayed.

---

## 6. Context Window Physics

### 6.1 Problem in the Earlier Spec

The superseded design built `Request.Messages` from session history and allowed the provider to report context-length overflow. That is not a viable foundation. Admission control belongs inside Gormes.

### 6.2 Context Planner

M1 adds a pure planning component:

```go
type ContextPlanner interface {
    Build(req PlanInput) (PlanResult, error)
}
```

Inputs:

- provider capabilities;
- configured model;
- system prompt;
- current user turn;
- prior complete turns in reverse chronological order;
- reserved output token budget.

Outputs:

- selected turn IDs in send order;
- dropped turn count;
- estimated input tokens;
- reserved output tokens;
- truncation reason, if any.

### 6.3 Provider Capabilities

M1 requires provider capability metadata:

```go
type Capabilities struct {
    ContextWindow        int
    DefaultMaxOutput     int
    MaxInputBytes        int
    SupportsReasoning    bool
    SupportsToolCalls    bool
}
```

The exact token estimator may be approximate in M1, but it must be conservative. Underestimation is a bug.

### 6.4 Planning Policy

For every run:

1. Reserve output tokens first.
2. Include the system prompt.
3. Include the current user turn.
4. Add previous complete turns from newest to oldest while the budget remains safe.
5. Reject locally if the current user turn alone exceeds the safe budget.

M1 explicitly does **not** summarize or compress history. That remains M2 work. But M1 does deterministic truncation.

### 6.5 Input Safety Limits

The TUI path applies hard local input guards before persistence or planning:

- maximum pasted byte size;
- maximum visible line count;
- optional warning threshold before send.

If the user exceeds the hard limit, Gormes rejects the turn locally with a precise error and does not call the provider.

### 6.6 Provenance of Planning

Every completed or failed run persists planning metadata:

- `selected_turn_ids`;
- `dropped_turn_count`;
- `estimated_tokens_in`;
- `reserved_output_tokens`;
- `planner_version`;
- `system_prompt_hash`;
- `truncation_reason`.

This gives M2 and M4 replayable context provenance.

---

## 7. Persistence Model for M1 and M2

### 7.1 Problem in the Earlier Spec

A single `turns` table with `metadata` and `reasoning` columns is not enough provenance. It captures content, but not the exact run that produced that content or the exact prior turns that were sent.

The additive path to M2 is not "store more JSON on turns." The additive path is "separate conversation atoms from model invocations."

### 7.2 Schema

M1 persistence is split into four tables:

```sql
CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,
    created_at  INTEGER NOT NULL,
    model       TEXT NOT NULL,
    title       TEXT
);

CREATE TABLE turns (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id      TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role            TEXT NOT NULL CHECK (role IN ('system','user','assistant')),
    visible_text    TEXT NOT NULL,
    reasoning_text  TEXT,
    state           TEXT NOT NULL
                      CHECK (state IN ('streaming','complete','cancelled','error')),
    created_at      INTEGER NOT NULL,
    completed_at    INTEGER
);

CREATE TABLE runs (
    id                  TEXT PRIMARY KEY,
    session_id          TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    provider            TEXT NOT NULL,
    model               TEXT NOT NULL,
    state               TEXT NOT NULL
                          CHECK (state IN ('planning','streaming','complete','cancelled','error')),
    output_turn_id      INTEGER REFERENCES turns(id),
    finish_reason       TEXT,
    error_class         TEXT,
    error_text          TEXT,
    estimated_tokens_in INTEGER,
    tokens_in           INTEGER,
    tokens_out          INTEGER,
    latency_ms          INTEGER,
    planner_version     TEXT,
    request_meta        TEXT,
    response_meta       TEXT,
    started_at          INTEGER NOT NULL,
    completed_at        INTEGER
);

CREATE TABLE run_inputs (
    run_id           TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    turn_id          INTEGER NOT NULL REFERENCES turns(id) ON DELETE CASCADE,
    ordinal          INTEGER NOT NULL,
    inclusion_reason TEXT NOT NULL,
    PRIMARY KEY (run_id, ordinal)
);
```

Indexes:

- `idx_turns_session_id` on `(session_id, id)`
- `idx_runs_session_started` on `(session_id, started_at)`
- `idx_run_inputs_run_id` on `(run_id, ordinal)`

### 7.3 Why This Shape Lowers Rewrite Risk

This schema gives M2 what it actually needs:

- `turns` remain the user-facing conversation log;
- `runs` become the durable ledger of provider invocations;
- `run_inputs` records exactly which persisted conversation turns formed each context window.

The rendered system prompt for each run is persisted in `runs.request_meta` together with `system_prompt_hash`. It is not required to appear in `run_inputs`.

M2 triple extraction can now be additive:

```sql
CREATE TABLE triples (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    source_turn_id    INTEGER NOT NULL REFERENCES turns(id) ON DELETE CASCADE,
    run_id            TEXT REFERENCES runs(id) ON DELETE SET NULL,
    subject           TEXT NOT NULL,
    predicate         TEXT NOT NULL,
    object            TEXT NOT NULL,
    confidence        REAL,
    extractor_version TEXT NOT NULL,
    span_start        INTEGER,
    span_end          INTEGER,
    created_at        INTEGER NOT NULL
);
```

No destructive reinterpretation of `turns.metadata` is required because provenance already exists.

### 7.4 Assistant Turn Lifecycle

At `StartingRun`, the kernel creates an assistant turn row with:

- empty `visible_text`;
- empty `reasoning_text`;
- `state='streaming'`.

User turns are inserted directly as `state='complete'`. Only assistant turns may enter `state='streaming'`.

During streaming, the draft remains in memory.

At finalization:

- `visible_text` is updated once;
- `reasoning_text` is updated once;
- `state` becomes `complete`, `cancelled`, or `error`;
- the matching `runs.output_turn_id` points to that row.

This gives the run a stable output identity without forcing per-token database writes.

### 7.5 History Semantics

`Session.History(...)` changes meaning:

- only `turns.state='complete'` are eligible for future prompt planning;
- `cancelled` and `error` assistant turns remain visible in the transcript but are excluded from future context by default;
- the planner decides what gets sent, and `run_inputs` records the decision.

---

## 8. Python Runtime Seam

### 8.1 Problem in the Earlier Spec

The earlier `Tool` stub is sufficient for an in-process function boundary. It is not sufficient for a subprocess runtime boundary. Python integration needs process lifecycle and invocation control.

### 8.2 Runtime Interface

M1 ships a `pybridge` stub around runtime lifecycle:

```go
type Runtime interface {
    ID() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health(ctx context.Context) error
    Catalog(ctx context.Context) (ToolCatalog, error)
    Invoke(ctx context.Context, req InvocationRequest) (Invocation, error)
}

type Invocation interface {
    Events() <-chan InvocationEvent
    Wait(ctx context.Context) (InvocationResult, error)
    Cancel() error
}
```

Support structs:

- `ToolCatalog` for discovered tools and schemas;
- `InvocationRequest` for tool name, arguments, deadlines, and correlation IDs;
- `InvocationEvent` for streamed logs, partial payloads, progress, and final result;
- `InvocationResult` for final payload, stderr summary, exit classification, and duration.

### 8.3 Why Runtime Beats Tool

This seam supports future requirements without redesign:

- long-lived warm Python workers;
- cold-start and shutdown control;
- heartbeat and health checks;
- capability discovery;
- streamed tool output;
- cancellation of long-running Python jobs;
- pool-based execution in M4+.

The agent-level `Tool` abstraction can still exist later, but it should sit **above** the runtime seam, not replace it.

### 8.4 M1 Contract

M1 does not implement Python subprocesses. It only ships:

- interface definitions;
- `ErrNotImplemented`;
- clear ownership that Go remains the orchestrator and Python remains a runtime peripheral.

---

## 9. TUI Contract

The TUI becomes a platform consumer of render snapshots, not a co-owner of stream state.

Rules:

- the TUI never assembles assistant text from raw provider events;
- the TUI renders the newest `RenderFrame`;
- resize events affect layout only, never orchestration state;
- input submission enters the kernel through a typed `PlatformEvent`.

This keeps the UI replaceable for M3 without changing the agent kernel.

---

## 10. Success Criteria Additions

M1 is not complete until all of the following are true:

1. Oversize user input is rejected locally before provider invocation.
2. Provider context overflow is unreachable in normal operation because the planner enforces conservative local limits.
3. Render-path overload does not grow memory unbounded; only the newest pending frame survives.
4. Artificial SQLite delay tests do not leak goroutines or wedge shutdown.
5. Cancel mid-stream leaves zero active provider-reader goroutines after test completion.
6. Each completed or cancelled run persists a `runs` row plus ordered `run_inputs` provenance.
7. Each assistant turn created by a run has a stable `output_turn_id`.
8. `pybridge` exports lifecycle-oriented runtime interfaces, not only a function-call interface.

These criteria are in addition to the ignition criteria inherited from the earlier spec where they do not conflict.

---

## 11. Test Matrix

### 11.1 Kernel Tests

- single-turn happy path
- provider stream faster than render flush
- render consumer intentionally stalled
- store adapter delayed beyond ack deadline
- cancel during streaming with provider still attempting to emit
- provider EOF, provider protocol error, provider context cancellation

### 11.2 Planner Tests

- current user message alone exceeds limit
- history exactly fits budget
- newest-first truncation drops oldest turns
- reserved output tokens reduce admissible history
- conservative estimator never undershoots a known fixture budget

### 11.3 Persistence Tests

- `runs` and `run_inputs` written for each completed run
- `cancelled` assistant turn excluded from future planner input
- `output_turn_id` integrity
- schema accepts future additive `triples` migration without rewriting existing rows

### 11.4 Runtime Seam Tests

- `ErrNotImplemented` surface is stable
- mock runtime satisfies lifecycle contract
- invocation cancellation path is compile-checked even before M4 implementation

---

## 12. Relationship to the Superseded Spec

`2026-04-18-gormes-ignition-design.md` remains useful as historical context for:

- directory layout;
- public documentation goals;
- Bubble Tea UI intent;
- milestone framing.

But where the two specs disagree, this document wins on all kernel, persistence, planning, and runtime-boundary questions.

---

## 13. Next Step

The next artifact is an implementation plan that decomposes this deterministic kernel into reviewable tasks:

- kernel loop and state machine;
- provider stream interface;
- store adapter and schema;
- planner and capability plumbing;
- TUI render-frame integration;
- runtime seam stub;
- tests and success criteria.

This spec is now the source of truth for **what** ignition should be.
