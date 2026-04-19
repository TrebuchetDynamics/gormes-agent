# Gormes — Phase 1 Frontend Adapter Design Spec

**Date:** 2026-04-18
**Author:** Xel (via Claude Code brainstorm, post-recon + kernel-discipline merge)
**Status:** Draft — awaiting approval
**Scope:** Phase 1 of the 5-phase Ship-of-Theseus port: the "GoCo Dashboard".
**Supersedes:** [`2026-04-18-gormes-ignition-design.md`](./2026-04-18-gormes-ignition-design.md) (greenfield, pre-recon)
**Absorbs:** [`2026-04-18-gormes-ignition-deterministic-kernel-design.md`](./2026-04-18-gormes-ignition-deterministic-kernel-design.md) — kernel disciplines only; its Phase-4-shaped architecture is retained in that file as the Phase-4 target.

**Architectural stance:** macro-architecture = Ship of Theseus (Python owns LLM/state; Go is a client in Phase 1). Micro-architecture = deterministic kernel (single-owner state, bounded mailboxes, pull-based streams, render-frame coalescing, admission before execution, cancellation leak-freedom). Even though Go is "just a client", its internal state machine is built with the discipline of a backend — so Phase 4 absorbs the agent loop without a rewrite.

---

## 1. Purpose

Ship a Go binary (`gormes/cmd/gormes`) that renders a cinematic Bubble Tea "Debug/Dashboard" TUI driven entirely by Hermes's existing **OpenAI-compatible HTTP+SSE server** at `gateway/platforms/api_server.py` (default `http://127.0.0.1:8642`).

Gormes is a **frontend adapter**, not an agent. Python owns LLM routing, prompt building, memory, session state, and tool execution. Go owns input capture, streaming render, soul-monitor state, and telemetry derivation. The user feels the speedup immediately; the Python core is untouched.

This is the first supplier swap in the 5-phase roadmap (§2). Every later phase expands Go's territory by absorbing a Python subsystem. The HTTP+SSE boundary between Python and Go is designed so it can survive all five phases — in the final state, Go *serves* the same contract Python does today, letting other clients (Open WebUI, LobeChat) continue working.

---

## 2. Program Context — the 5-Phase Ship of Theseus

| Phase | Name | Go takes over | Python still owns | Scope in this spec |
|---|---|---|---|---|
| **1** | **The Dashboard (Face)** | TUI + input + SSE client | Agent loop, LLM routing, memory, sessions, tools, gateway | ✅ this spec |
| 2 | The Wiring Harness (Gateway) | Multi-platform adapters (Telegram, Discord, Slack) | Agent loop, memory, tools | ❌ future |
| 3 | The Black Box (Memory) | SQLite + FTS5 + ontological graph | Agent loop, LLM routing, tools | ❌ future |
| 4 | The Powertrain (Brain Transplant) | Agent orchestrator, prompt builder, LLM client | Only tool/skill scripts (as subprocess) | ❌ future |
| 5 | The Final Purge (100% Go) | Tools ported to Go or WASM; Python contract deleted | — | ❌ future |

The HTTP+SSE boundary used in Phase 1 is the **same boundary** Phases 2–4 slowly migrate across. Phase 4's "flip" is when Gormes starts *serving* `/v1/chat/completions` instead of consuming it.

---

## 3. Architectural Principles

### 3.1 Gormes is a consumer, not an orchestrator
Every LLM call, memory lookup, tool invocation, and persistence write happens in Python. Gormes's job is: capture input → POST to Python → render the resulting SSE stream. Anything more sophisticated than that is out of scope for Phase 1.

### 3.2 Zero local state
Gormes has **no database, no cache, no session file** in Phase 1. If the user restarts Gormes, state survives because Python's `state.db` is untouched. Gormes reconnects, re-reads history via the HTTP API if needed, and resumes. SQLite ownership moves to Go only in Phase 3; before then, writing to `state.db` from Go could corrupt Python's store and is forbidden.

### 3.3 Process isolation
Python `api_server` and Go `gormes` are independent processes communicating only over HTTP. Python crash ≠ Go crash. Go reconnects automatically (§9). Python can be restarted without Go losing its render buffer.

### 3.4 CGO-free
Preserved from superseded spec: all Go dependencies are pure-Go for static cross-compilation to Linux / macOS / Termux. No CGO, no C headers.

### 3.5 HTTP contract stability
Upstream Python may refactor `api_server.py` internals freely. The OpenAI-compatible wire contract is stable by virtue of being public OpenAI-compatible — LobeChat, Open WebUI, and others depend on it. Gormes shares that moat.

### 3.6 Single-Owner Kernel (Micro-Architecture)
One goroutine — the **kernel** — owns the turn state machine, the assistant draft buffer, the current phase, and the render snapshot. Every other goroutine (TUI loop, HTTP client, SSE reader, telemetry ticker) is an **edge adapter** that communicates with the kernel through explicitly-bounded mailboxes. No cross-ownership, no shared mutexes outside well-defined edge adapters, no goroutine reaches into the kernel's fields. Phase 4 replaces the kernel's "POST and stream" body with "call the LLM and stream" without touching any adapter — that's what single-owner discipline buys.

### 3.7 Bounded Mailboxes, Explicit Saturation
Every channel has a capacity and a saturation policy (block, drop-oldest, replace-latest, or fail-fast with a deadline). There are no unbounded queues anywhere in Gormes. The LLM can produce 200 tokens/sec bursts and the TUI can stall on a resize — neither failure mode can OOM the other.

### 3.8 Admission Before Execution
The kernel rejects work at the edge of Go, not at the edge of Python. Oversized pastes, empty inputs, and in-flight-turn-collisions are rejected in-kernel before the HTTP POST fires. Python's `context-length` errors are treated as a local admission bug, not a normal control path. In Phase 1 the admission rules are lightweight (byte/line/concurrency limits); Phase 4 absorbs the full context planner from the deterministic-kernel spec.

### 3.9 Cancellation Leak-Freedom
Every active turn gets a child context. Every mailbox send is `select { case mailbox <- msg: ; case <-ctx.Done(): return }`. Every goroutine exits on ctx cancellation. Testing asserts zero goroutine leaks after cancel, zero mailbox growth under provider overrun, and zero process hangs under stalled-adapter scenarios (§15.5).

---

## 4. Prerequisites

**Runtime prerequisite — the Python api_server must be running.** Before launching `gormes`, the operator runs one of:

```bash
# Option A — enable only the API server, no messaging platforms
API_SERVER_ENABLED=true hermes gateway start

# Option B — the api_server auto-enables when an API key is set
API_SERVER_KEY=<any-string> hermes gateway start
```

Defaults: host `127.0.0.1`, port `8642`. Overridable via `API_SERVER_HOST`, `API_SERVER_PORT`, `API_SERVER_CORS_ORIGINS`, `API_SERVER_MODEL_NAME`.

Gormes reports a clear error if the api_server is unreachable and suggests the env-var command (§10.1). Auto-spawning Python is **out of scope** for Phase 1 — process lifecycle stays manual.

---

## 5. Process Model

```
┌──────────────────────────────────────────────────────┐       ┌──────────────────────────────────┐
│                 gormes (Go, pid N)                     │       │   hermes gateway (Python, pid M) │
│                                                        │       │                                  │
│   ┌──────────┐   RenderFrame   ┌──────────────┐      │       │   ┌──────────────┐              │
│   │   TUI    │◄────(cap=1)─────│              │      │       │   │ api_server   │              │
│   │(bubbleT) │────PlatformEvt─►│   KERNEL     │ HTTP │       │   │ :8642        │              │
│   └──────────┘    (cap=16)    │ (single-owner)│──POST┼──────►│   └──────┬───────┘              │
│                                │ state machine│      │       │          │                       │
│                                │              │ pull │ SSE   │          ▼                       │
│                                │   Recv() ────┼──────┼───────┼── Agent (litellm, skills,       │
│                                │              │      │       │          memory)                  │
│                                └──────────────┘      │       │          │                       │
│                                       ▲              │       │     state.db (owned by Python)   │
│                            StoreCmd   │ Ack (≤250ms) │       │                                  │
│                             (cap=16)  │              │       │                                  │
│                                       ▼              │       │                                  │
│                                (no-op store stub     │       │                                  │
│                                 in Phase 1; real     │       │                                  │
│                                 store lands Phase 3) │       │                                  │
└───────────────────────────────────────────────────────┘       └──────────────────────────────────┘
```

**Four actors. One owns state; three are edge adapters.**

- **KERNEL** (single-owner goroutine) — owns the phase state machine (§7.5), the assistant draft buffer, the render snapshot, and turn cancellation. Never touched directly by adapters.
- **TUI adapter** (Bubble Tea loop) — consumes `RenderFrame` from a capacity-1 mailbox with *replace-latest* saturation; emits `PlatformEvent` to the kernel through a capacity-16 mailbox with *fail-fast* saturation.
- **HTTP+SSE adapter** (`internal/hermes`) — per-turn goroutine. Kernel calls `client.OpenStream(ctx, req)` and pulls events via `Recv(ctx)`. The adapter exits on ctx cancel or stream EOF; the kernel owns pacing.
- **Store adapter** (no-op in Phase 1; stub interface present) — accepts persistence commands on a capacity-16 mailbox with an ack deadline. If ack is not received within 250 ms, the kernel stops admitting new turns and transitions to `Failed`. This matches the deterministic-kernel discipline from day one, even though Phase 1's store is a no-op.

---

## 6. Directory Layout

```
gormes/
├── cmd/gormes/main.go
├── internal/
│   ├── kernel/                    # single-owner state machine + render snapshot
│   │   ├── kernel.go              # Run loop; Phase enum; mailbox wiring
│   │   ├── frame.go               # RenderFrame type + coalescing (16ms flush)
│   │   ├── admission.go           # local input admission (byte/line/concurrency)
│   │   ├── provenance.go          # in-memory correlation: local RunID ↔ server IDs
│   │   └── kernel_test.go
│   ├── hermes/                    # HTTP+SSE edge adapter
│   │   ├── client.go              # Client with OpenStream() pull-based API
│   │   ├── stream.go              # Stream interface + Recv()
│   │   ├── sse.go                 # SSE frame parser (bounded read buffer)
│   │   ├── events.go              # /v1/runs/{id}/events subscriber (pull)
│   │   ├── errors.go              # Classify(), HTTPError
│   │   └── client_test.go
│   ├── store/                     # no-op store stub (Phase 1) + interface
│   │   ├── store.go               # Store interface + NoopStore impl
│   │   └── store_test.go
│   ├── pybridge/                  # runtime seam stub (Phase 5 target)
│   │   └── pybridge.go            # Runtime interface + Invocation stubs
│   ├── tui/
│   │   ├── model.go               # TUI Model — renders newest RenderFrame ONLY
│   │   ├── view.go
│   │   ├── update.go
│   │   └── tui_test.go
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   └── telemetry/
│       ├── telemetry.go           # snapshot type consumed by RenderFrame
│       └── telemetry_test.go
├── pkg/gormes/
│   └── types.go                   # public re-exports
├── docs/
│   ├── ARCH_PLAN.md
│   ├── docs_test.go               # SSG-portability lint
│   └── superpowers/
│       ├── specs/*.md
│       └── plans/*.md
├── go.mod
├── README.md
└── Makefile
```

**What changed from the pre-merge draft:**
- `hub/` → `kernel/`. The package is no longer a "hub" (a routing hub implies multiple owners); it is a kernel (single-owner state machine).
- `kernel/frame.go`, `kernel/admission.go`, `kernel/provenance.go` are new files enforcing the deterministic-kernel disciplines.
- `store/` is added as a stub. Phase 1's `NoopStore` satisfies the interface but writes nothing. Phase 3 replaces the impl behind the interface with real SQLite; no other package changes.
- `pybridge/` returns as a Phase-5 runtime-seam stub (lifecycle-oriented, not the simpler `Tool.Call` stub from the original spec).
- `hermes/stream.go` is new — it holds the pull-based `Stream` interface separate from the HTTP `Client`.

**Reserved-but-empty:** `internal/agent/` (Phase 4 — the actual LLM-routing + prompt-builder Go code), `internal/gateway/` (Phase 2). These directories are not created in Phase 1.

---

## 7. Core Interfaces

### 7.1 HTTP+SSE Client — pull-based, not channel-returning

```go
// Client is the ONLY piece of code that speaks HTTP in Gormes. It returns a
// Stream that the kernel pulls from — never a channel the kernel drains
// passively. Pull-based consumption lets the kernel pace intake against
// render-frame coalescing and cancellation.
type Client interface {
    // OpenStream POSTs /v1/chat/completions?stream=true and returns a Stream
    // whose Recv() yields token/reasoning/finalization events.
    // Session continuity: if req.SessionID is non-empty, sent as
    // X-Hermes-Session-Id; the canonical server session id is available via
    // Stream.SessionID() after the first Recv() succeeds.
    OpenStream(ctx context.Context, req ChatRequest) (Stream, error)

    // OpenRunEvents subscribes to /v1/runs/{run_id}/events. Returns nil
    // (disabled gracefully) if the server responds with 404 — non-Hermes
    // OpenAI-compatible servers (LM Studio, Open WebUI) don't implement it.
    OpenRunEvents(ctx context.Context, runID string) (RunEventStream, error)

    // Health pings /health with a 2s timeout.
    Health(ctx context.Context) error
}

type Stream interface {
    // Recv returns the next event or io.EOF when the stream ends cleanly.
    // On ctx.Done(), returns ctx.Err() without closing any shared state.
    // Caller must invoke Close() exactly once when done (successful or not).
    Recv(ctx context.Context) (Event, error)

    // SessionID returns the canonical server-assigned session id. Zero
    // value until the first successful Recv() that carries the
    // X-Hermes-Session-Id header was observed.
    SessionID() string

    // Close releases the underlying HTTP body. Safe to call multiple times.
    Close() error
}

type RunEventStream interface {
    Recv(ctx context.Context) (RunEvent, error)
    Close() error
}
```

### 7.2 Event Types

```go
type Event struct {
    Kind         EventKind
    Token        string           // when Kind == EventToken
    Reasoning    string           // when Kind == EventReasoning
    FinishReason string           // when Kind == EventDone
    TokensIn     int              // final delta only
    TokensOut    int              // running count; final on Done
    Raw          json.RawMessage  // retained for diagnostics / provenance
}

type EventKind int
const (
    EventToken EventKind = iota
    EventReasoning
    EventDone
)

type RunEvent struct {
    Type     RunEventType
    ToolName string     // for ToolStarted / ToolCompleted
    Preview  string     // tool args preview (truncated to 60 chars server-side)
    Reasoning string    // for ReasoningAvailable
    Raw      json.RawMessage
}

type RunEventType int
const (
    RunEventToolStarted RunEventType = iota
    RunEventToolCompleted
    RunEventReasoningAvailable
    RunEventUnknown
)

type ChatRequest struct {
    Model     string
    Messages  []Message
    SessionID string
    // Stream is always true in Phase 1; kept for forward-compat when non-
    // streaming single-shot calls land later.
    Stream    bool
}

type Message struct {
    Role    string  // "system" | "user" | "assistant"
    Content string
}
```

### 7.3 Store Stub (Phase 1 no-op; Phase 3 real)

```go
// Store is the persistence seam. Phase 1 ships NoopStore (silently accepts
// everything, never errors). Phase 3 replaces NoopStore with a SQLite impl
// behind the SAME interface — no kernel changes.
type Store interface {
    // Exec submits a command and blocks until ack or ctx deadline. The
    // kernel enforces a 250ms ack deadline before transitioning to Failed.
    Exec(ctx context.Context, cmd Command) (Ack, error)
}

type Command struct {
    Kind    CommandKind   // AppendUserTurn | AppendAssistantDraft | FinalizeTurn
    Payload json.RawMessage
}

type Ack struct {
    TurnID int64 // 0 from NoopStore; populated in Phase 3
}
```

### 7.4 Runtime Seam Stub (Phase 5 target; compile-checked in Phase 1)

```go
// Runtime is the lifecycle-oriented seam for Python subprocesses. Phase 1
// only declares the types so Phase 5 can plug an impl behind them. No impl
// is shipped; callers receive ErrNotImplemented.
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

var ErrNotImplemented = errors.New("gormes/pybridge: runtime lands in Phase 5")
```

(Support structs `ToolCatalog`, `InvocationRequest`, `InvocationEvent`, `InvocationResult` defined in the package; shapes match `2026-04-18-gormes-ignition-deterministic-kernel-design.md` §8.2.)

### 7.5 Kernel Phase State Machine

```
                 submit (admission pass)
   ┌───────────► Idle ─────────────────────► Connecting
   │              ▲                              │
   │              │                              │ 200 + first Recv
   │ store ack    │                              ▼
   │              │                          Streaming
   │              │                              │
   │           Finalizing ◄──────────────────────┤ Recv(EventDone)
   │              ▲                              │
   │              │ drain budget ≤2s             │ Ctrl+C
   │              │                              ▼
   │              └── Cancelling ◄───────────────┤
   │                                             │
   │                                             │ any retryable error
   │                                             ▼
   │                                      (auto-reconnect, §9.2)
   │                                             │
   │                                             │ fatal error, or
   │                                             │ retry budget exhausted
   └──────── Failed ◄────────────────────────────┘
```

Rules:
- Exactly one run active per session. A second `PlatformEventSubmit` during `Streaming | Cancelling | Finalizing` is **rejected** in-kernel with a UI error ("still processing previous turn") — not queued. Queuing is M1.5 work.
- Phase transitions happen on the kernel goroutine only. Adapters propose transitions via events; the kernel decides.
- Every `RenderFrame` carries a monotonically increasing `Seq`. The TUI renders only the newest received frame.
- Store ack timeout (250 ms) in Phase 1 is trivially met by `NoopStore`. In Phase 3 the real SQLite impl must honor the same deadline.

### 7.6 RenderFrame — the TUI's only input

```go
// RenderFrame is a complete snapshot of visible TUI state. The TUI never
// assembles assistant text from raw provider events — it renders this frame.
// Mailbox capacity is 1 with replace-latest saturation: a slow TUI drops
// stale frames instead of backpressuring the kernel.
type RenderFrame struct {
    Seq         uint64            // monotonic per Gormes process; strictly increasing
    Phase       Phase             // Idle | Connecting | Streaming | ...
    DraftText   string            // assistant buffer accumulated so far (current turn)
    History     []Message         // turns OBSERVED by THIS Gormes process since launch.
                                  // Python's state.db holds the canonical record;
                                  // Gormes does not fetch prior history in Phase 1.
                                  // On Gormes restart this field starts empty — Python
                                  // still knows the session thanks to X-Hermes-Session-Id.
    Telemetry   TelemetrySnapshot
    StatusText  string            // one-line kernel status
    SessionID   string
    Model       string
    LastError   string            // empty when no error
    SoulEvents  []SoulEntry       // ring buffer (last 10) — driven by RunEvent feed
}
```

### 7.7 Kernel Flush Policy

- Coalesce `EventToken` / `EventReasoning` into the in-memory `DraftText`.
- Flush a new `RenderFrame` at most every **16 ms** (≈60 Hz).
- Flush **immediately** on semantic edges: phase transitions, `EventDone`, any `RunEvent`, any error.
- If the render mailbox (capacity 1) already holds a pending frame, replace it with the newer frame. The TUI only ever sees the latest.

Net effect: a 200 tok/s provider burst produces ~3 render frames per second of visible UI change (because characters accumulate faster than 16 ms frame windows), while the actual-token counter in the telemetry pane ticks in real time through the same flush mechanism. The TUI cannot backpressure the kernel; the kernel cannot firehose the TUI.

### 7.8 Mailbox Catalog

| Mailbox | From | To | Capacity | Saturation policy |
|---|---|---|---|---|
| render | kernel | TUI | 1 | **replace-latest** (drop old) |
| platform-events | TUI | kernel | 16 | **fail-fast**: TUI disables input briefly if full |
| store-commands | kernel | Store | 16 | **fail-fast**: kernel transitions to `Failed` if send blocked >250 ms |
| sse-events | hermes adapter | kernel (pulled) | n/a — pull-based | kernel paces via `Recv(ctx)` |
| run-events | hermes adapter | kernel (pulled) | n/a — pull-based | kernel paces via `Recv(ctx)` |

**There are no other channels in Gormes.** If a future package adds one, it must be documented in this table with capacity and saturation policy.

No `Session`, no `Provider`, no `Tool` as Phase-1 consumer-facing types. Python owns those concepts; `Runtime` (§7.4) is the future seam.

---

## 8. Wire Protocol — exactly what crosses HTTP

### 8.1 Chat request

```http
POST http://127.0.0.1:8642/v1/chat/completions
Content-Type: application/json
X-Hermes-Session-Id: <id-or-empty>

{
  "model": "hermes-agent",
  "stream": true,
  "messages": [
    {"role": "user", "content": "hello"}
  ]
}
```

- **`model`** — for Phase 1 always `"hermes-agent"` (the Python server's model name; `API_SERVER_MODEL_NAME` overrides it server-side).
- **`messages`** — Gormes only sends the latest user turn when a session is already in play (Python re-hydrates history from its own state.db using `X-Hermes-Session-Id`). For a new conversation, messages is a single-element array.
- **System prompt assembly** — Python's `prompt_builder.py` owns the system prompt, personality files, and memory injection. **Gormes never sets the `system` role.** Attempting to do so would conflict with Python's builder.

### 8.2 Chat response (SSE)

Response headers include `X-Hermes-Session-Id: <canonical-id>`. The SSE body follows OpenAI conventions:

```
data: {"id":"...","choices":[{"delta":{"content":"hel"}}]}

data: {"id":"...","choices":[{"delta":{"content":"lo","reasoning":"thinking..."}}]}

data: {"id":"...","choices":[{"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2}}

data: [DONE]
```

Gormes parses each `data:` frame as a `Delta`. The `[DONE]` sentinel closes the stream.

### 8.3 Run events (SSE, separate connection)

Used by the Soul Monitor. Structure (per api_server.py line 1058):

```
event: tool.started
data: {"tool_call_id":"...","name":"terminal","args":{"cmd":"ls"}}

event: tool.completed
data: {"tool_call_id":"...","name":"terminal","result_preview":"..."}

event: reasoning.available
data: {"text":"I should first check..."}
```

Standard `event: <type>\ndata: <json>\n\n` framing.

---

## 9. SSE Lifecycle — connection, drop, reconnect, cancel

The TUI must survive network flakes, Python restarts, and user cancellation without crashing.

### 9.1 State machine

```
       ┌────────────────────────────────────────────────────┐
       │                                                    │
       ▼                                                    │
   [idle] ── user submits ──► [connecting] ── 200 ──► [streaming]
       ▲                              │                  │    │
       │                              │                  │    └─ ctx cancel ──┐
       │                              │                  │                    │
       │                              │                  ├─ stream EOF ──►[finalizing]
       │                              │                  │                    │
       │                              │                  │                    ▼
       │                              │                  │              [idle (turn done)]
       │                              │                  │
       │                              │                  ├─ network drop ──►[reconnecting]
       │                              │                  │                       │
       │                              │                  │                       │
       │                              ▼                  ▼                       │
       │                        [error: fatal]    [error: retryable]◄────────────┘
       │                              │                  │
       └──────────── user dismisses ──┴──────────────────┘
```

### 9.2 Reconnect strategy
- **Health check on startup** — `GET /health` with 2 s timeout. Failure → red banner in Soul Monitor + a line suggesting `API_SERVER_ENABLED=true hermes gateway start`. TUI remains usable for history browsing (nothing to browse in Phase 1, but the shell stays alive).
- **Mid-stream network drop** — the SSE body read returns `io.ErrUnexpectedEOF` or a timeout error. Gormes classifies as retryable (§12.1), tags the in-flight turn with an **in-memory** flag `interrupted` (no DB write — Phase 1 has no DB), renders a `[interrupted — reconnecting]` marker inline in the conversation, and schedules a reconnect with exponential backoff: 1 s → 2 s → 4 s → 8 s → 16 s (cap). No retry is attempted if the user has already started typing the next input — in that case the interrupted turn stays on screen with its partial content and the next POST is a fresh turn.
- **Python restart** — detected as either (a) a connection refused at reconnect, (b) an unknown `X-Hermes-Session-Id` on the next request. In case (b), Gormes clears its local session-id cache and starts a new session — Python's `state.db` still has the prior history and a future Phase-1.5 history-fetch can re-hydrate it.
- **User cancel mid-stream** — `Ctrl+C` cancels the HTTP request context. The partial assistant content stays on screen with a `[cancelled]` marker. No local persistence write happens (Phase 1 = zero local state).

### 9.3 Non-crash contract
Under **no** circumstances does a network or protocol error crash the TUI process. All errors become `UIUpdate{Kind: Error}` entries in the conversation plus a line in the Soul Monitor. The Bubble Tea panic recovery (inherited from superseded spec §10.3) catches programming bugs; network errors never reach it.

### 9.4 Timeouts
- Request header timeout: 5 s (time to first byte from api_server).
- Idle read timeout on SSE body: 60 s (Python's `CHAT_COMPLETIONS_SSE_KEEPALIVE_SECONDS = 30.0` emits a keepalive; we budget 2x).
- Total request timeout: **none** (turns may legitimately run minutes with tool use).

---

## 10. Soul Monitor — event mapping

The Soul Monitor renders live agent state in the TUI sidebar (inherited from superseded spec §9). In Phase 1 its inputs come from two streams only: the chat SSE body (for token-level state) and optional run-events SSE (for tool-level state).

### 10.1 Derived state vs. explicit events

| Soul-monitor state | Source | Trigger |
|---|---|---|
| `idle` | derived | no in-flight request |
| `connecting` | derived | POST sent, no response yet |
| `streaming` | derived | first chat-SSE delta received with non-empty `choices[0].delta.content` |
| `reasoning` | explicit | `event: reasoning.available` on run-events stream **OR** a chat delta carrying `reasoning` field |
| `tool: <name>` | explicit | `event: tool.started`, cleared on matching `tool.completed` |
| `finalizing` | derived | chat SSE has emitted `finish_reason` but stream not yet closed |
| `interrupted` | explicit | network drop during streaming |
| `cancelled` | explicit | user `Ctrl+C` |
| `error: <class>` | explicit | any `HTTPError` or parse error |

### 10.2 Event-to-UI rendering rules

- **`tool.started`** → append `[12:04:03] tool: terminal` (with arg preview if available) to the Soul Monitor ring buffer (capacity 10).
- **`tool.completed`** → append `[12:04:05] ✓ terminal` to the ring buffer.
- **`reasoning.available`** → append `[12:04:04] reasoning: thinking...` truncated to 60 chars.
- **Unknown event type** → append `[12:04:06] ? <type>` and log the raw payload at `DEBUG`. **Gormes does not crash on unknown event types.** Forward-compatibility: Python may add event kinds (`subagent.started`, `memory.write`, etc.) without breaking Gormes.

### 10.3 Thinking-event explicit non-support

api_server.py line 2078 states: `# _thinking and subagent_progress are intentionally not forwarded`. Gormes accepts that constraint — the Soul Monitor does **not** show per-token "thinking" state. `reasoning` (which IS forwarded) substitutes.

---

## 11. TUI Dashboard

Unchanged from the superseded spec §9, with one clarification: **the Soul Monitor ring buffer is driven by the event mapping in §10.2 of this spec**, not by locally synthesized `thinking`/`streaming` strings.

Responsive rules (width ≥ 100, 80–99, < 80), key bindings (`Enter`, `Shift+Enter`, `Ctrl+C`, `Ctrl+L`, `Ctrl+D`, `PgUp`/`PgDn`), SIGWINCH handling via `tea.WindowSizeMsg`, and the non-crash contract for widths ≥ 20×10 all carry forward verbatim. See superseded spec §9 for the layout diagram.

---

## 12. Error Handling

### 12.1 Classification (in `internal/hermes/errors.go`)

```go
type ErrorClass int
const (
    ClassUnknown ErrorClass = iota
    ClassRetryable    // network, 429, 500, 502, 503, 504, read timeout
    ClassFatal        // 401, 403, context-length, malformed SSE, 404
)
```

Phase 1 behavior:
- **Retryable** — auto-reconnect per §9.2, surface in Soul Monitor.
- **Fatal** — no retry; show a red banner + terminate turn; TUI stays alive.

### 12.2 Panic recovery
Same as superseded spec §10.3: `defer recover()` in `main.go` dumps to `$XDG_DATA_HOME/gormes/crash-<ts>.log`.

### 12.3 Cancellation contract
Same as superseded spec §10.4: signal-derived root ctx, 2-second shutdown budget, exit code 3 if budget exceeded.

### 12.4 Local Input Admission

Kernel rejects a `PlatformEventSubmit` **before** any HTTP call fires if any of these hold:

| Rule | Default | Override |
|---|---|---|
| Empty / whitespace-only | reject | — |
| Byte length > `GORMES_MAX_INPUT_BYTES` | reject with "input too large (N bytes, limit M)" | env or `[input]` in config.toml |
| Line count > `GORMES_MAX_INPUT_LINES` | reject | same |
| A run is already active in the session | reject with "still processing previous turn" | — |

Defaults: 200 KB, 10 000 lines. These are deliberately generous — the real purpose of the admission rule is to make pathological paste-the-whole-file-by-accident inputs a clear failure with a precise message, not a 413 from Python that takes ten seconds to time out. Phase 4 replaces this with the full context planner from the deterministic-kernel spec §6.

A rejection produces a `RenderFrame` with `LastError` set and `Phase` remaining `Idle`. The TUI re-focuses the editor with the user's text preserved.

### 12.5 Local Provenance & Correlation IDs

Every admitted turn is assigned a **local run ID** (UUIDv7, 30 bytes hex). The kernel logs via `slog` at `INFO` the correlation:

```
turn admitted   local_run_id=01JH... session_hint=...
turn POST sent  local_run_id=01JH... endpoint=http://127.0.0.1:8642/v1/chat/completions
turn SSE start  local_run_id=01JH... server_session_id=... model=hermes-agent
turn done       local_run_id=01JH... server_session_id=... finish=stop tokens=3/5 latency=312ms
turn error      local_run_id=01JH... class=retryable err=...
```

The correlation is in-memory (a map[local_run_id]Correlation) and log-backed — **no DB writes in Phase 1**. Phase 3 promotes this to a `runs` table (schema from the deterministic-kernel spec §7.2) through the Store seam. In Phase 1 the log stream is the audit trail.

Log output goes to `$XDG_DATA_HOME/gormes/gormes.log` with daily rotation (stdlib `log/slog` + a custom rotating `io.Writer`). Setting `GORMES_LOG_LEVEL=DEBUG` adds SSE-chunk-level granularity for debugging.

### 12.6 Leak Freedom Assertions

Every test that exercises cancel or error paths asserts:
- Zero goroutine leak 200 ms after cancel (baseline-subtract comparison).
- No mailbox remains with pending items at kernel exit.
- HTTP body Close() was called on every opened Stream (verified by a counting `http.Transport` wrapper in tests).

See §15.5 for the discipline-specific test list.

---

## 13. Configuration

### 13.1 Sources (precedence order)
1. CLI flags (`--endpoint`, `--model`) — **no `--api-key` flag**; secrets must not appear in process argument lists
2. Environment variables (`GORMES_ENDPOINT`, `GORMES_MODEL`, `GORMES_API_KEY`)
3. Config file at `$XDG_CONFIG_HOME/gormes/config.toml`
4. Built-in defaults

### 13.2 Config file shape

```toml
[hermes]
endpoint = "http://127.0.0.1:8642"   # Python api_server base URL
api_key  = ""                         # matches API_SERVER_KEY on the Python side (empty if unset)
model    = "hermes-agent"             # served model name

[tui]
theme = "dark"                        # "dark" | "light"
```

### 13.3 Required for Phase 1
Nothing. Gormes works against the default local api_server with no config. If the Python side has `API_SERVER_KEY` set, the user sets `GORMES_API_KEY` to the same value.

---

## 14. Telemetry

Derived from SSE events — **no DB**.

Counters (in-memory only, reset on Gormes restart):
- `tokens_in_total`, `tokens_out_total` — from the final chat-SSE `usage` field.
- `latency_ms_last` — from request-start → `finish_reason` delta.
- `tokens_per_sec` — EMA (α = 0.2) of chat-SSE delta rate.
- `model` — set from `ChatRequest.Model` at request time; overridden if any SSE chunk payload carries a `model` field (OpenAI chunks sometimes do).

Renderer unchanged from superseded spec §12.

---

## 15. Testing Strategy

### 15.1 Unit
- `hermes/client_test.go` — fake `httptest.Server` scripts: SSE happy path, reasoning field, unknown event type, `[DONE]` sentinel, mid-stream `io.ErrUnexpectedEOF`, 401, 429, 503, 404.
- `hermes/events_test.go` — run-events SSE parser for `tool.started` / `tool.completed` / `reasoning.available` / unknown.
- `hub/hub_test.go` — orchestrator with a scripted `HermesClient` fake: happy path, reconnect after drop, cancel.
- `config/config_test.go` — flag > env > toml > defaults precedence (carried over from superseded spec).
- `telemetry/telemetry_test.go` — EMA math, counter increments.

### 15.2 TUI
- `tui/tui_test.go` via `teatest`:
  - **type-send** — input → fake client emits 3 deltas → assistant buffer renders `"hello"`.
  - **cancel** — `Ctrl+C` mid-stream → `[cancelled]` marker appears, turn exits cleanly.
  - **reconnect** — drop mid-stream → Soul Monitor shows `interrupted`, then `connecting`, then `streaming` again.
  - **resize** — sequence of `tea.WindowSizeMsg` at widths 200/80/50/10/200 does not panic.
  - **unknown event** — run-events fake emits `event: subagent.started` → Soul Monitor shows `? subagent.started`; no panic.

### 15.3 Live integration
- `hermes/live_test.go` with `//go:build live`. Requires `API_SERVER_ENABLED=true hermes gateway start` to be running locally. Skips with `t.Skip` if `http://127.0.0.1:8642/health` returns any non-2xx status **or** the connection is refused.

### 15.4 Coverage target
≥ 70 % line coverage on `internal/` excluding `tui/`. `tui/` is validated by `teatest` behavioral tests rather than line coverage.

### 15.5 Kernel-Discipline Tests (required)

The following tests MUST pass. Each asserts one of the deterministic-kernel invariants under stress.

1. **Provider outpaces TUI.** `MockStream` emits 2000 `EventToken`s in 10 ms. Render mailbox capacity is 1. TUI `teatest` harness deliberately stalls 100 ms on `Update`. Assert: (a) no goroutine leak, (b) final `DraftText` equals the concatenation of all tokens, (c) TUI rendered fewer than 200 frames (coalescing worked), (d) kernel did not block.

2. **TUI permanently stalled.** Render mailbox consumer stops receiving. Kernel continues to receive SSE events and accumulates them into `DraftText`. Assert: memory use remains bounded (one frame in mailbox); kernel completes the turn; on ctx cancel, kernel exits cleanly.

3. **Store ack delayed.** `NoopStore` is wrapped with a `SlowStore` that delays ack by 500 ms (beyond the 250 ms deadline). Kernel submits one turn. Assert: kernel transitions to `Failed`, emits a final `RenderFrame{Phase: Failed, LastError: "store ack timeout"}`, and does not accept further submissions until reset.

4. **Cancel mid-stream, zero goroutine leak.** User submits turn. SSE emits 5 tokens. User sends `Ctrl+C`. Assert: no goroutine from `hermes`, `kernel`, or `store` is leaked 200 ms after cancel (counted by `runtime.NumGoroutine()` delta). HTTP body Close was called.

5. **HTTP drop mid-stream with auto-reconnect.** Fake server closes SSE mid-stream. Assert: kernel emits `interrupted` Soul event, schedules reconnect, tries up to 5 attempts with backoff, then transitions to `Failed` if all fail.

6. **Unknown RunEvent type.** Fake server emits `event: subagent.started`. Assert: kernel records `RunEventUnknown`, Soul Monitor renders `? subagent.started`, no panic.

7. **Input admission rejection.** User submits 300 KB of text (above the 200 KB default). Assert: kernel returns `RenderFrame{LastError: "input too large..."}`, no HTTP POST fires, `Phase` stays `Idle`.

8. **Second submit during streaming.** User submits turn 1. Before turn 1 completes, user submits turn 2. Assert: turn 2 is rejected with "still processing previous turn", turn 1 completes normally.

9. **Health check failure on startup.** `gormes` launched with no `api_server` running. Assert: startup completes, TUI renders, Soul Monitor shows actionable error (`API_SERVER_ENABLED=true hermes gateway start`), further submits rejected until next successful health re-check.

10. **Render frame Seq monotonicity.** 50 rapid turns. Assert: every emitted `RenderFrame.Seq` is strictly greater than the previous one observed by the TUI.

These ten tests, combined with the standard teatest scenarios (§15.2), are the production-readiness bar for Phase 1.

---

## 16. Build & Tooling

Unchanged from superseded spec §14 — Go 1.22+, Makefile targets (`build`, `test`, `test-live`, `lint`, `fmt`), `modernc.org/sqlite` **no longer needed** in Phase 1 (reserved for Phase 3).

---

## 17. Dependency Map

| Purpose | Python (Hermes) | Go (Gormes Phase 1) |
|---|---|---|
| LLM client | `litellm`, `instructor`, etc. | **none — consumed via HTTP** |
| System-prompt building | `agent/prompt_builder.py` | **none — Python builds the prompt** |
| Session storage | `hermes_state.py` / state.db | **none — Python owns state** |
| HTTP client | n/a | stdlib `net/http` |
| SSE parsing | n/a | hand-rolled in `internal/hermes/sse.go` |
| TUI | `rich` / custom | `charmbracelet/bubbletea`, `bubbles`, `lipgloss` |
| Config | various | `spf13/pflag` + `pelletier/go-toml/v2` |
| Logging | `hermes_logging.py` | stdlib `log/slog` |

**Dependencies removed vs. superseded spec:** `modernc.org/sqlite`, any provider-side SDK. Gormes's `go.mod` is now much smaller.

---

## 18. Relationship to the Python Codebase

Same hard rule as superseded spec §16: **no Python file is modified**. All Gormes work lives under `gormes/`. The one-time repo-root README update for "Go Implementation Status" is still deferred until after Phase 1 ships.

The HTTP boundary means Gormes does not need to import anything from Python, and Python does not need to know Gormes exists. Python's `api_server.py` treats Gormes the same as Open WebUI or LobeChat.

---

## 19. Explicit Out-of-Scope for Phase 1

| Feature | Deferred to |
|---|---|
| Local SQLite store, FTS5, history cursor | Phase 3 |
| Ontological fact-triples | Phase 3 |
| Gateway (Telegram / Discord / Slack / WhatsApp / Signal adapters in Go) | Phase 2 |
| Native Go agent orchestrator, direct OpenRouter calls | Phase 4 |
| Python bridge RPC (subprocess) | Phase 5 (inverted: Go calls Python tools as subprocess) |
| System-prompt builder in Go | Phase 4 |
| Tool execution in Go | Phase 5 |
| Auto-spawning Python (`gormes up` = `hermes gateway start` + `gormes`) | Phase 1.5 |
| Session picker / `--new` / `--session <id>` flags | Phase 1.5 |
| Voice, MCP, Atropos, subagents | TBD |
| Prompt caching | Phase 4 (Python owns until then) |
| Multi-provider routing UI | n/a in Go — Python's concern |
| `/v1/responses` API consumption (stateful chaining) | Phase 1.5 (Phase 1 uses `/v1/chat/completions` + `X-Hermes-Session-Id`) |

---

## 20. Success Criteria

Phase 1 is "dashboard-live" when **all** hold:

1. `go build ./cmd/gormes` succeeds on Linux, macOS, and Termux from a clean checkout.
2. With `API_SERVER_ENABLED=true hermes gateway start` running, `./bin/gormes` renders the Dashboard with no additional config.
3. A typed prompt produces live token streaming into the conversation pane within 500 ms of the first byte.
4. Soul Monitor visibly transitions through `connecting → streaming → idle` on a typical turn.
5. When the Python agent invokes a tool (verified by the live smoke test prompting one), `tool: <name>` appears in the Soul Monitor and clears when the tool completes.
6. Killing the Python process mid-stream causes the turn to exit with `status: interrupted` and an auto-reconnect attempt — no Go process crash.
7. `Ctrl+C` mid-stream cancels cleanly; partial content remains visible with a `[cancelled]` marker.
8. Resizing the terminal during streaming does not crash the process (verified by the `teatest` resize test).
9. `make test` passes with ≥ 70 % coverage on `internal/` (excluding `tui/`).
10. `gormes/docs/ARCH_PLAN.md` exists and contains the 5-phase Ship-of-Theseus roadmap verbatim from §2.
11. The Markdown lint at `gormes/docs/docs_test.go` passes on ARCH_PLAN.md, this spec, and the Phase 1 plan.
12. **No Python file in the repo has been modified.**
13. **`gormes.db`, `state.db`, or any SQLite file under Go control does not exist.** (verified by a post-build find: `find ~/.local/share/gormes -name '*.db' -print` returns empty).
14. **All ten kernel-discipline tests from §15.5 pass.** Specifically: provider-outpaces-TUI coalescing works, ctx cancellation leaves zero goroutine leak, store-ack-timeout transitions to Failed, render-frame Seq is strictly monotonic.
15. **No unbounded channel exists in the Go code.** Enforced by a Go-test-based lint at `gormes/internal/discipline_test.go` that uses `go/ast` to walk every `make(chan ...)` call in `internal/` and asserts a capacity literal > 0 is present, OR the channel is on the whitelist (documented in §7.8's mailbox catalog).

---

## 21. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| api_server's SSE keepalive interval (30 s) shorter than our idle timeout (60 s) — acceptable, but if Python bumps keepalive we may falsely timeout | Document the 2x ratio in a comment; if Python's keepalive changes, adjust via config, not code |
| Session-id header not returned (old Python version) | Fall back to stateless turns with full history on each request (higher token cost, still correct) |
| api_server's event set grows (new `subagent.started` etc.) | §10.2 unknown-event rule handles it without a Gormes change |
| Users run Gormes without starting api_server first | `/health` startup check produces a clear actionable error with the exact env-var command |
| Python-side schema changes to `/v1/chat/completions` response (breaking) | Unlikely — it's OpenAI-compatible; breakage affects every third-party client and Hermes won't ship it. If it happens, Gormes pins a response schema version later |
| Tool-call preview contains binary / very long content | Truncate to 60 chars in Soul Monitor; store full preview in `RunEvent.Raw` for debugging |
| Gormes running against a non-Hermes OpenAI server (Open WebUI, LM Studio) — run-events endpoint absent | Feature-detect: `GET /v1/runs` 404 → disable Soul Monitor tool/reasoning lines gracefully, keep token streaming working |

---

## 22. Documentation Strategy

Carry forward §21 of the superseded spec with one change: the `ARCH_PLAN.md` milestone table is replaced by the 5-phase Ship-of-Theseus roadmap from §2 of this spec. The Markdown SSG-portability lint (`docs_test.go`) now checks ARCH_PLAN.md + this spec + the Phase 1 plan.

`ARCH_PLAN.md` required-subsections change slightly:
1. Rosetta Stone Declaration — unchanged.
2. Why Go — unchanged (the five concrete bullets still apply).
3. Hybrid Manifesto — **updated**: the hybrid is temporary. The long-term state is 100% Go. Every phase shrinks Python's footprint.
4. 5-phase live-status table — replaces the old 5-milestone table. Status emoji (🔨 / ✅ / ⏳ / ⏸) and legend unchanged.
5. Project Boundaries — unchanged.
6. Public site pointer (`https://gormes.io`) — unchanged.

---

## 23. Next Step

After this spec is approved, `superpowers:writing-plans` produces the Phase 1 implementation plan. The new plan will be **significantly smaller** than the discarded 21-task plan — estimated 10–12 tasks — because 4 packages are gone and the agent-cognition layer becomes a thin HTTP hub.

This spec is the source of truth for *what* Phase 1 is. The plan is the source of truth for *how* it gets built.
