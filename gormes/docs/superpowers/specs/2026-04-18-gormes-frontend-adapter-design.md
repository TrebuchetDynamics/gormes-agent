# Gormes — Phase 1 Frontend Adapter Design Spec

**Date:** 2026-04-18
**Author:** Xel (via Claude Code brainstorm, post-recon)
**Status:** Draft — awaiting approval
**Scope:** Phase 1 of the 5-phase Ship-of-Theseus port: the "GoCo Dashboard".
**Supersedes:** [`2026-04-18-gormes-ignition-design.md`](./2026-04-18-gormes-ignition-design.md)

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
┌──────────────────────────────────────────┐       ┌──────────────────────────────────┐
│            gormes (Go, pid N)             │       │   hermes gateway (Python, pid M) │
│                                           │       │                                  │
│   ┌──────────┐   UIUpdate   ┌──────────┐ │       │   ┌──────────────┐              │
│   │   TUI    │◄─────────────│   hub    │ │ HTTP  │   │ api_server   │              │
│   │(bubbleT) │──UserInput──►│(orch+evt)│─┼──POST ┼──►│ :8642        │              │
│   └──────────┘              └────┬─────┘ │       │   └──────┬───────┘              │
│                                  │       │       │          │                       │
│                            ┌─────▼─────┐ │ SSE   │   ┌──────▼───────┐              │
│                            │   SSE     │◄┼───────┼───│   Agent      │              │
│                            │  client   │ │       │   │  (litellm,   │              │
│                            └───────────┘ │       │   │   skills,    │              │
│                                           │       │   │   memory)    │              │
└───────────────────────────────────────────┘       │   └──────────────┘              │
                                                    │          │                       │
                                                    │     state.db (owned by Python)  │
                                                    └──────────────────────────────────┘
```

Three Go actors, all channel-based:

- **TUI actor** — Bubble Tea loop. Consumes `UIUpdate`; emits `UserInput`.
- **Hub actor** — one goroutine that owns turn lifecycle: POSTs `/v1/chat/completions`, opens the SSE body, feeds deltas into the TUI. Also owns the `/v1/runs/{id}/events` subscription when a run is in flight.
- **SSE client** — per-request goroutine that parses server-sent events and publishes them on a channel the hub consumes.

---

## 6. Directory Layout

Compared to the superseded spec, **4 packages disappear entirely** (`db`, `session`, `provider`, `pybridge`), and `agent` shrinks to a thin hub.

```
gormes/
├── cmd/gormes/main.go
├── internal/
│   ├── hermes/                    # HTTP+SSE client for api_server
│   │   ├── client.go              # POST /v1/chat/completions, /v1/runs
│   │   ├── sse.go                 # SSE frame parser
│   │   ├── events.go              # /v1/runs/{id}/events subscriber
│   │   ├── errors.go              # Classify(), HTTPError
│   │   └── client_test.go
│   ├── hub/                       # turn orchestrator (no LLM logic)
│   │   ├── hub.go
│   │   └── hub_test.go
│   ├── tui/
│   │   ├── model.go
│   │   ├── view.go
│   │   ├── update.go
│   │   └── tui_test.go
│   ├── config/
│   │   ├── config.go              # env + XDG + flags + TOML
│   │   └── config_test.go
│   └── telemetry/
│       ├── telemetry.go           # counters derived from SSE events
│       └── telemetry_test.go
├── pkg/gormes/
│   └── types.go                   # public re-exports
├── docs/
│   ├── ARCH_PLAN.md
│   ├── docs_test.go               # SSG-portability lint
│   └── superpowers/
│       ├── specs/*.md
│       └── plans/*.md
├── go.mod                         # module: github.com/XelHaku/golang-hermes-agent/gormes
├── README.md                      # "Rosetta Stone" explainer
└── Makefile
```

**Reserved-but-empty for later phases:** `internal/db`, `internal/session`, `internal/agent`, `internal/gateway`, `internal/pybridge`. Phase 1 does not create these; the directory layout anticipates them so later phases add packages without restructuring.

---

## 7. Core Interfaces

```go
// HermesClient is the Phase-1 contract with the Python api_server.
// It is the ONLY piece of code that speaks HTTP in Gormes. Everything else
// consumes the channel-based output of Stream().
type HermesClient interface {
    // Stream POSTs to /v1/chat/completions with stream=true and returns a
    // channel of Deltas. The channel is closed when streaming ends.
    // Session continuity: if req.SessionID is non-empty, it is sent as the
    // X-Hermes-Session-Id request header; the server's returned session_id
    // is surfaced on the first Delta.
    Stream(ctx context.Context, req ChatRequest) (<-chan Delta, error)

    // Health pings /health; used at startup to surface a clear error if
    // the Python api_server isn't running.
    Health(ctx context.Context) error
}

type ChatRequest struct {
    Model     string
    Messages  []Message          // role+content; Gormes only appends the latest user msg
    SessionID string             // "" for new conversation; echoed back on first Delta
    Stream    bool               // always true in Phase 1
}

type Delta struct {
    Token        string
    Reasoning    string           // populated when api_server forwards reasoning
    SessionID    string           // set on first Delta of a stream
    Done         bool
    FinishReason string
    RawEnvelope  json.RawMessage  // retained for diagnostics; NOT persisted in Phase 1
    Err          error
}

type Message struct {
    Role    string  // "system" | "user" | "assistant"
    Content string
}
```

**Run events** are a separate subscription used for the Soul Monitor (§10):

```go
// Events subscribes to /v1/runs/{run_id}/events via SSE. Used alongside
// Stream when the user opts into the /v1/runs flow (future Phase 1.5).
// For Phase 1, Gormes consumes the event stream inline with a chat completion
// when the api_server emits runs-style events in parallel.
type EventClient interface {
    Events(ctx context.Context, runID string) (<-chan RunEvent, error)
}

type RunEvent struct {
    Type    RunEventType        // see §10.2
    ToolName string             // for ToolStarted / ToolCompleted
    Preview  string             // tool args preview
    Reasoning string            // for ReasoningAvailable
    Raw      json.RawMessage
}

type RunEventType int

const (
    EventToolStarted RunEventType = iota  // SSE event: "tool.started"
    EventToolCompleted                     // SSE event: "tool.completed"
    EventReasoningAvailable                // SSE event: "reasoning.available"
    EventUnknown                           // forward-compatible: render as raw
)
```

No `Session`, no `Provider`, no `Tool` in Phase 1. Python owns those concepts.

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
