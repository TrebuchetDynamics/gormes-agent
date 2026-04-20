# Gormes — Ignition (M0 + M1) Design Spec

> **⚠️ SUPERSEDED — 2026-04-18**
>
> This spec described a greenfield parallel implementation where Gormes called OpenRouter directly. After a recon pass discovered that Hermes already exposes an **OpenAI-compatible HTTP+SSE server** (`gateway/platforms/api_server.py`, port 8642), the project pivoted to a Ship-of-Theseus strangler-fig architecture: Gormes is a Go *frontend adapter* that consumes Python's existing API, not a parallel rewrite.
>
> **Replaced by:** [`2026-04-18-gormes-frontend-adapter-design.md`](./2026-04-18-gormes-frontend-adapter-design.md)
>
> The content below is retained as historical record only. Do **not** implement from this document.

---

**Date:** 2026-04-18
**Author:** Xel (via Claude Code brainstorm)
**Status:** SUPERSEDED — see header
**Scope:** Milestones M0 (scaffolding) + M1 (TUI + one LLM provider) of the Gormes program.
**Parent program:** Gormes — Go port of Hermes Agent. Upstream: `NousResearch/hermes-agent`. Fork: `TrebuchetDynamics/gormes-agent`.

---

## 1. Purpose

Create the first running vertical slice of Gormes: a Go binary that (a) scaffolds the "Motherboard" architecture, (b) boots a Bubble Tea "Debug/Dashboard" TUI, and (c) completes one streaming LLM turn against OpenRouter with SQLite-backed conversation history.

The goal is **not** feature parity with Hermes. The goal is a grounded, testable foundation that subsequent milestones (M2 ontological memory, M3 multi-platform gateway, M4 Python bridge) extend rather than reinvent. A plan validated by working code is worth ten plans validated by prose.

---

## 2. Program Context — The 5-Milestone Vision

| Milestone | Focus | Scope in this spec |
|---|---|---|
| **M0 — Scaffold** | Go module, folder layout, `ARCH_PLAN.md`, interfaces, DB migrations | ✅ included |
| **M1 — TUI + LLM** | Bubble Tea Dashboard, OpenRouter provider, streaming turn | ✅ included |
| **M2 — Ontological Memory** | FTS5, fact-triples, semantic recall | ❌ out of scope |
| **M3 — Multi-Platform Gateway** | Telegram / Discord / CLI concurrent adapters | ❌ out of scope |
| **M4 — Python Bridge** | Subprocess RPC to existing Python tools | Interface stub only |

Each future milestone gets its own brainstorm → spec → plan → implementation cycle.

---

## 3. Architectural Principles

### 3.1 The Motherboard Pattern
Go is the chassis. The Go process owns state, orchestration, persistence, and platform I/O. Python is the peripheral library — complex research tools, legacy Hermes skills, and heavy ML ops will be invoked from Go as subprocess RPC in M4+. This spec does not implement the bridge but reserves the interface boundary for it.

### 3.2 Channels Over Shared State
All inter-actor communication uses channels. Shared mutable state is confined to the DB layer (SQLite serializes writes internally). No `sync.Mutex` outside `internal/db`.

### 3.3 Context Cancellation as Universal Stop
Every goroutine receives a `context.Context`. User `/stop` or `Ctrl+C` cancels the root context; every actor observes cancellation and flushes partial state before exiting.

### 3.4 CGO-Free
All Go dependencies must be pure-Go (no CGO). This makes cross-compilation trivial — a single `GOOS=linux GOARCH=amd64 go build` produces a static binary deployable to a $5 VPS or Termux.

---

## 4. Process Model

Three concurrent actors within one Go process:

```
┌─────────────────────────────────────────────────────────┐
│                     gormes (pid 1)                       │
│                                                          │
│   ┌──────────┐   UIUpdate   ┌──────────┐   Delta        │
│   │   TUI    │◄─────────────│  Agent   │◄──────┐        │
│   │(bubbleT) │──PlatformEvt►│(orchestr)│       │        │
│   └──────────┘              └──────────┘       │        │
│                                   │             │        │
│                                   ▼             │        │
│                              ┌──────────┐       │        │
│                              │ Provider │───────┘        │
│                              │(OpenRouter)│               │
│                              └──────────┘                │
│                                   │                      │
│                              SQLite (local file)         │
└─────────────────────────────────────────────────────────┘
```

- **TUI actor:** Bubble Tea program loop. Consumes `UIUpdate`; emits `PlatformEvent`.
- **Agent actor:** Single goroutine owns turn lifecycle. Serializes persistence writes. Spawns one **Provider streaming goroutine** per in-flight turn.
- **Provider actor:** Per-request goroutine; HTTP(S) streaming to OpenRouter SSE; emits `Delta` values on a buffered channel; closes channel on completion or error.

---

## 5. Directory Layout

```
gormes/
├── cmd/
│   └── gormes/
│       └── main.go              # entry point; wires actors, signals, ctx
├── internal/
│   ├── agent/
│   │   ├── agent.go             # Agent struct, Run loop
│   │   └── agent_test.go
│   ├── tui/
│   │   ├── model.go             # Bubble Tea Model
│   │   ├── view.go              # render (lipgloss)
│   │   ├── update.go            # state transitions
│   │   └── tui_test.go          # teatest
│   ├── provider/
│   │   ├── provider.go          # Provider interface, Request/Delta
│   │   ├── openrouter.go        # OpenRouter impl (SSE streaming)
│   │   ├── mock.go              # MockProvider for tests
│   │   ├── errors.go            # ErrorClass classifier
│   │   └── openrouter_test.go
│   ├── session/
│   │   ├── session.go
│   │   └── session_test.go
│   ├── db/
│   │   ├── db.go                # open, migrate, close
│   │   ├── queries.go           # prepared statements
│   │   ├── migrations/
│   │   │   └── 0001_initial.sql
│   │   └── db_test.go
│   ├── config/
│   │   ├── config.go            # env + XDG resolution
│   │   └── config_test.go
│   ├── telemetry/
│   │   └── telemetry.go         # tokens/sec, latency counters
│   └── pybridge/
│       └── pybridge.go          # M4 stub — Tool interface + ErrNotImplemented
├── pkg/
│   └── gormes/
│       └── types.go             # Provider, Platform, Tool (public re-exports)
├── docs/
│   ├── ARCH_PLAN.md             # the 5-milestone program vision doc
│   └── superpowers/
│       └── specs/
│           └── 2026-04-18-gormes-ignition-design.md   # this file
├── go.mod                       # module: github.com/TrebuchetDynamics/gormes-agent/gormes
├── go.sum
├── README.md                    # "Rosetta Stone" explainer
└── Makefile                     # build, test, test-live, lint
```

Module path: `github.com/TrebuchetDynamics/gormes-agent/gormes`. This allows `go install github.com/TrebuchetDynamics/gormes-agent/gormes/cmd/gormes@latest` to work from a single upstream.

---

## 6. Core Interfaces

Defined in `internal/provider/` and `internal/tui/`, re-exported from `pkg/gormes` for future external importers.

```go
// Provider — any LLM backend.
type Provider interface {
    // Stream issues a completion request and returns a channel of Deltas.
    // The channel is closed by the Provider when streaming ends (normal or error).
    // The final Delta before close has Done=true and, if applicable, Err set.
    Stream(ctx context.Context, req Request) (<-chan Delta, error)
    Name() string
}

type Request struct {
    Model    string
    Messages []Message     // full history including system prompt
    Params   Params        // temperature, max_tokens, etc.
}

type Delta struct {
    Token     string        // incremental content chunk (visible assistant content)
    Reasoning string        // incremental reasoning chunk (hidden thinking), empty if none
    TokensIn  int           // set on final delta only
    TokensOut int           // running count; final value on Done
    Done      bool
    FinishReason string     // set on Done: "stop" | "length" | "tool_calls" | "error" | ...
    RawEnvelope json.RawMessage // provider-specific final payload (stored verbatim in metadata)
    Err       error         // non-nil → terminal error
}

type Message struct {
    Role    string        // "system" | "user" | "assistant"
    Content string
}
```

```go
// Platform — any UI surface. M1 ships the CLI/TUI implementation;
// M3 adds Telegram, Discord, etc.
type Platform interface {
    Events() <-chan PlatformEvent
    Emit(UIUpdate) error
    Start(ctx context.Context) error
    Stop() error
}

type PlatformEvent struct {
    Kind EventKind        // Input | Cancel | Reset | Quit
    Text string
}

type UIUpdate struct {
    Kind      UpdateKind  // Token | TurnStart | TurnComplete | Telemetry | SoulEvent | Error
    Token     string
    Telemetry TelemetrySnapshot
    SoulEvent string      // "thinking" | "querying" | "streaming" | "idle" | ...
    Err       error
}
```

```go
// Tool — stub for M2+. Included in M1 to lock the boundary; no concrete impl.
type Tool interface {
    Name() string
    Call(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
}
```

`Session` is a concrete struct, not an interface:

```go
type Session struct {
    ID        string
    Model     string
    CreatedAt time.Time
    // unexported: *sql.DB handle
}

func (s *Session) AppendTurn(role, content string) (turnID int64, err error)
// History returns up to `limit` most-recent turns in chronological (ascending) order.
// Only turns with status='complete' are returned; cancelled/error turns are skipped.
func (s *Session) History(ctx context.Context, limit int) ([]Message, error)
func (s *Session) UpdateTurnStats(id int64, tokensIn, tokensOut, latencyMs int) error
func (s *Session) UpdateTurnMetadata(id int64, reasoning string, envelope json.RawMessage) error
func (s *Session) MarkTurnStatus(id int64, status string) error
```

### 6.1 System Prompt Assembly (the "Soul" for M1)

Hermes's Python `agent/prompt_builder.py` does heavy lifting — personality files, self-nudges, memory injection, tool-schema rendering. **None of that lands in M1.** The Soul logic stays in Go (not delegated to the Python bridge) because it is orchestrator territory; the bridge is for tools, not for the agent's own cognition.

**M1 contract:**
- `internal/agent` exposes `buildSystemPrompt(cfg Config) string` — pure function, no I/O.
- Default body: a short built-in string (~120 tokens) that identifies the agent as Gormes and states it has no tools yet. Lives in `internal/agent/default_prompt.go` as a `const`.
- Override via `[agent].system_prompt` in `config.toml` or `GORMES_SYSTEM_PROMPT` env var (overrides config file).
- **No file-based personality loading in M1** — that's a Hermes feature (`personality/*.md`) deferred to M2 alongside skills.
- The assembled system prompt is prepended to `Request.Messages` as `Message{Role: "system"}` on every turn.

**M2+ growth path:** `internal/agent/prompt/` becomes a package with composable sections (identity, personality, tool-schema, memory-recall, self-nudges). M1 writes the seam, M2 fills it.

---

## 7. Data Flow — One Turn

```
User types "hi" + Enter
      │
      ▼
TUI emits PlatformEvent{Kind: Input, Text: "hi"}
      │
      ▼
Agent.handleInput:
  1. session.AppendTurn("user", "hi")                 → turn_id = 42
  2. msgs, _ := session.History(ctx, N)
  3. req := Request{Model: cfg.Model, Messages: msgs}
  4. UIUpdate{Kind: SoulEvent, SoulEvent: "thinking"}
  5. UIUpdate{Kind: TurnStart}
  6. deltas, err := provider.Stream(ctx, req)
      │
      ▼
For each delta from provider:
  - append Token to assistant buffer
  - UIUpdate{Kind: Token, Token: delta.Token}
  - telemetry.Tick(delta.TokensOut)
  - UIUpdate{Kind: Telemetry, Telemetry: telemetry.Snapshot()}
      │
      ▼
Stream closes (Done=true):
  1. session.AppendTurn("assistant", fullText)        → turn_id = 43
  2. session.UpdateTurnStats(43, in, out, latency)
  3. session.UpdateTurnMetadata(43, reasoningBuf, finalDelta.RawEnvelope)
  4. UIUpdate{Kind: TurnComplete, Telemetry: final}
  5. UIUpdate{Kind: SoulEvent, SoulEvent: "idle"}
```

**Cancellation:** if ctx is cancelled mid-stream, the agent:
1. Drains remaining deltas with a tight timeout (100 ms) to allow graceful provider shutdown.
2. Persists partial assistant text with `status = 'cancelled'`.
3. Emits `UIUpdate{Kind: TurnComplete}` with a cancelled indicator.

---

## 8. Persistence

SQLite via `modernc.org/sqlite` (pure Go, no CGO). DB file: `$XDG_DATA_HOME/gormes/gormes.db` (default `~/.local/share/gormes/gormes.db`).

### 8.1 Schema — `migrations/0001_initial.sql`

```sql
CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,
    created_at  INTEGER NOT NULL,
    model       TEXT NOT NULL,
    title       TEXT
);

CREATE TABLE turns (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role        TEXT NOT NULL CHECK (role IN ('system','user','assistant')),
    content     TEXT NOT NULL,                -- visible assistant/user text
    reasoning   TEXT,                         -- provider-emitted thinking (o1, <think>, etc.); NULL if none
    metadata    TEXT,                         -- JSON blob: provider envelope, finish_reason, tool_calls, …
    tokens_in   INTEGER,
    tokens_out  INTEGER,
    latency_ms  INTEGER,
    status      TEXT NOT NULL DEFAULT 'complete'
                   CHECK (status IN ('complete','cancelled','error')),
    created_at  INTEGER NOT NULL
);

CREATE INDEX idx_turns_session ON turns(session_id, id);

CREATE TABLE schema_version (version INTEGER PRIMARY KEY);
INSERT INTO schema_version VALUES (1);
```

**Why `reasoning` and `metadata` exist in M1 even though nothing reads them:**
Adding them later would require a migration across existing user data. The cost in M1 is one nullable column each and one extra UPDATE per turn. The benefit is M2 can build the fact-triple store (subject-predicate-object) by parsing `metadata` JSON without a schema change — Triples will live in their own `triples` table, but they derive *from* `turns.metadata` and `turns.content`. Reasoning tokens also become first-class for the TUI's Soul Monitor deep-dive view (future M1.5 enhancement).

`metadata` is stored as raw JSON text, not normalized columns, because the provider envelope shape varies per provider (OpenRouter, Anthropic, OpenAI, Gemini) and normalizing it would force breaking changes every time a provider adds a field.

FTS5 virtual table deferred to M2. FTS5 will index `turns.content` and the flattened fact-triples — not `metadata` JSON.

### 8.2 Migration Strategy
On startup, `db.Open()`:
1. Opens DB file (creates if missing).
2. Reads `schema_version`; if missing, runs all migrations; otherwise runs migrations with `version > current`.
3. Wraps each migration in a transaction.

Migrations live in `internal/db/migrations/*.sql` and are embedded via `//go:embed` so the binary ships them.

---

## 9. TUI Dashboard

### 9.1 Layout

```
╭──────────────────────────────────┬──────────────────────╮
│  conversation                    │  Telemetry            │
│  (scrollable, word-wrap)         │   model: ...          │
│                                  │   tok/s: 42           │
│  > user prompt                   │   latency: 312 ms     │
│  assistant: streaming tokens...  │   in/out: 1.2k/340    │
│                                  │  ───────              │
│                                  │  Soul Monitor         │
│                                  │   [12:04:03] thinking │
│                                  │   [12:04:04] stream   │
│                                  │   ...                 │
├──────────────────────────────────┴──────────────────────┤
│  > _  (multiline editor; Enter to send)                  │
╰──────────────────────────────────────────────────────────╯
```

### 9.2 Components
- **Conversation viewport** (`bubbles/viewport`): auto-scroll on new content, `PgUp`/`PgDn` manual scroll.
- **Telemetry pane:** bare `lipgloss` block, updated from `UIUpdate{Kind: Telemetry}`.
- **Soul Monitor:** ring buffer of the last 10 `UIUpdate{Kind: SoulEvent}` entries with timestamps.
- **Editor** (`bubbles/textarea`): multiline, Enter sends, Shift+Enter newline, placeholder hint.
- **Status line** (bottom of editor): current model name, session id.

### 9.3 Responsive Rules & SIGWINCH Handling

- Width ≥ 100 cols: full layout.
- 80–99 cols: sidebar shrinks to 24 cols.
- < 80 cols: sidebar collapses; telemetry + soul compress into a one-line status strip above the editor.

**Resize mechanics:** Bubble Tea handles `SIGWINCH` internally and delivers a `tea.WindowSizeMsg` to the Model's `Update` method. The Gormes TUI Model stores the last-seen `width`/`height` and recomputes layout on every `WindowSizeMsg`. No raw signal handling in user code. No partial-render state is kept across resizes — `View()` is pure over `(Model, width, height)`. Rapid resize events (drag) are not throttled in M1; `lipgloss` re-render is fast enough.

**Explicit non-crash contract:** resizing the terminal — at any rate, to any size ≥ 20 cols × 10 rows — MUST NOT panic or exit the process. Widths below 20 cols render a single-line "terminal too narrow" banner and suppress editor input; recovery is automatic once width increases. This is covered by `tui/tui_test.go` resize tests.

### 9.4 Key Bindings
| Key | Action |
|---|---|
| `Enter` | send turn |
| `Shift+Enter` | newline in editor |
| `Ctrl+C` | if a stream is in-flight, cancel it; otherwise quit immediately |
| `Ctrl+L` | clear conversation view (does not clear DB) |
| `PgUp` / `PgDn` | scroll conversation |
| `Ctrl+D` | quit |

---

## 10. Error Handling

### 10.1 Provider Errors

Classifier in `internal/provider/errors.go`:

```go
type ErrorClass int

const (
    ClassRetryable ErrorClass = iota  // 429, 500, 502, 503, 504, network
    ClassFatal                        // 401, 403, context-length, malformed
    ClassUnknown
)
```

**M1 behaviour:** surface in Soul Monitor and mark turn as `status='error'`. **No retry loop in M1** — deferred to a dedicated retry middleware in M1.5/M2.

### 10.2 DB Errors
Writes log with `slog` at `WARN`; turn marked `status='error'`; UI shows ❌ glyph next to the partial content. Reads log at `ERROR` and return an empty history slice (the agent can still attempt to proceed).

### 10.3 TUI Panic Recovery
`cmd/gormes/main.go` wraps `tea.NewProgram().Run()` in:

```go
defer func() {
    if r := recover(); r != nil {
        dumpCrash(r, debug.Stack())  // writes ~/.local/share/gormes/crash-<ts>.log
        os.Exit(2)
    }
}()
```

### 10.4 Cancellation Contract
- Root `ctx` derived from `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)`.
- Every goroutine passes ctx down and observes `ctx.Done()`.
- Agent has a **2-second shutdown budget** after root-ctx cancel to flush pending DB writes. If the budget expires, `main` logs `"shutdown budget exceeded"` at `ERROR` and exits with status code `3` regardless of in-flight work — data loss is acceptable at that point because the user has asked twice to stop.

---

## 11. Configuration

### 11.1 Sources (precedence order)
1. CLI flags (`--model`, `--provider`, `--db-path`)
2. Environment variables (`OPENROUTER_API_KEY`, `GORMES_MODEL`, `GORMES_DB`)
3. Config file at `$XDG_CONFIG_HOME/gormes/config.toml` (default `~/.config/gormes/config.toml`)
4. Built-in defaults

### 11.2 Config File Shape (TOML)

```toml
[provider]
name  = "openrouter"
model = "anthropic/claude-opus-4-7"   # any OpenRouter slug

[storage]
db_path = ""                           # empty = XDG default

[tui]
theme = "dark"                         # "dark" | "light"; more themes in M1.5
```

### 11.3 Required for M1
Only `OPENROUTER_API_KEY` is strictly required. Everything else has sensible defaults.

---

## 12. Telemetry

In-memory counters in `internal/telemetry`:
- `tokens_in_total`, `tokens_out_total` (per session and lifetime)
- `latency_ms_last`, `latency_ms_p50`, `latency_ms_p95` (rolling 50-window)
- `tokens_per_sec` (EMA with α=0.2)

No external metrics export in M1 (no Prometheus, no OTLP). Snapshot struct is emitted via `UIUpdate{Kind: Telemetry}`.

---

## 13. Testing Strategy

### 13.1 Unit
- `provider/openrouter_test.go`: parse fixture SSE streams; validate Delta sequence.
- `provider/mock.go`: `MockProvider` exposes `Script([]Delta)` for agent tests.
- `agent/agent_test.go`: happy path, cancel mid-stream, provider error, DB-write error.
- `session/session_test.go`: append / read history, cursor pagination.
- `db/db_test.go`: migrations apply cleanly; idempotent on re-run.
- `config/config_test.go`: precedence rules.

### 13.2 TUI
- `tui/tui_test.go` using Charm's `teatest`:
  - **Type-send:** input → turn completes with scripted MockProvider.
  - **Cancel:** Ctrl+C mid-stream → cancelled turn persists.
  - **Resize:** window < 80 cols collapses sidebar.

### 13.3 Live Integration
- Build tag `//go:build live`.
- `go test -tags=live ./internal/provider/...` hits real OpenRouter if `OPENROUTER_API_KEY` is set; otherwise `t.Skip`.
- Runs in CI only on manual dispatch (not on every push).

### 13.4 Coverage Target
≥ 70 % line coverage on `internal/` (excluding `tui/`, which is exercised via `teatest` integration rather than line-by-line).

---

## 14. Build & Tooling

### 14.1 Go Version
`go 1.22` minimum (`slog`, improved range loops).

### 14.2 Makefile Targets
| Target | Action |
|---|---|
| `make build` | `go build -o bin/gormes ./cmd/gormes` |
| `make test` | `go test ./...` |
| `make test-live` | `go test -tags=live ./...` |
| `make lint` | `golangci-lint run` |
| `make fmt` | `gofmt -w . && goimports -w .` |

### 14.3 CI
Defer CI config to M1.5 to avoid coupling to repo-wide CI during initial bootstrap. Local `make test` is the bar for M1.

---

## 15. Dependency Map

| Purpose | Python (Hermes) | Go (Gormes M1) |
|---|---|---|
| LLM client | `litellm`, `instructor`, `anthropic-sdk-python` | hand-rolled `internal/provider` (OpenRouter only) |
| TUI | `rich`, custom `ui-tui/` | `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss` |
| SQLite | `sqlite3` stdlib | `modernc.org/sqlite` (CGO-free) |
| Async | `asyncio` | stdlib goroutines + channels |
| Config | various | `spf13/pflag` + stdlib `os.Getenv` + `pelletier/go-toml/v2` |
| Logging | `hermes_logging.py` | stdlib `log/slog` |
| HTTP | `httpx` | stdlib `net/http` |
| SSE parsing | `sseclient-py` | hand-rolled in `provider/openrouter.go` |

**Explicit non-dependencies for M1:** no `cobra` (pflag is enough), no `viper` (toml+env is enough), no `sqlx` (stdlib `database/sql` is enough), no generated code.

---

## 16. Relationship to Existing Python Codebase

**Hard rule:** no Python file is modified. All Gormes work lives under `gormes/`.

**Exception:** the repo-root `README.md` may be updated exactly once to add a "Go Implementation Status" section pointing at `gormes/README.md`. That README update is a separate commit from the Gormes scaffolding work and is explicitly out-of-scope for this spec (it can happen opportunistically once M1 ships).

Upstream rebases against `NousResearch/hermes-agent` will not conflict with Gormes since all new files live under `gormes/`.

---

## 17. Explicit Out-of-Scope for M0 + M1

Deferred to named future milestones or TBD; listed explicitly to prevent scope creep.

| Feature | Deferred to |
|---|---|
| Tool calling (function calls) | M2 |
| Skills system | M2 |
| FTS5 memory search | M2 |
| Ontological fact-triples | M2 |
| Session summarization / compression | M2 |
| Gateway (Telegram / Discord / Slack / WhatsApp / Signal) | M3 |
| Multiple concurrent platforms | M3 |
| Python-bridge RPC subprocess | M4 (interface stub only in M1) |
| Additional providers (Anthropic, OpenAI, Gemini, Nous Portal, NIM, …) | M1.5+ |
| Voice mode (STT/TTS) | TBD |
| MCP server integration | TBD |
| Atropos RL environments | TBD |
| Subagent spawning | TBD |
| Multi-provider smart routing | post-M1.5 |
| Prompt caching | post-M1.5 |
| Config wizard / `gormes setup` | M1.5 |
| Authentication / DM pairing | M3 |
| Cron / scheduled automations | post-M3 |
| Terminal backends (Docker / SSH / Daytona / Singularity / Modal) | post-M3 |

---

## 18. Success Criteria

The M0 + M1 slice is "ignition-complete" when **all** the following hold:

1. `go build ./cmd/gormes` succeeds on Linux, macOS, and Termux from a clean checkout.
2. `./bin/gormes` launches the Dashboard TUI with no prior config beyond `OPENROUTER_API_KEY`.
3. A typed prompt streams tokens live into the conversation pane; Soul Monitor shows `thinking → streaming → idle`.
4. Telemetry pane displays non-zero `tok/s`, `latency`, and token counts for a completed turn.
5. The conversation persists in `gormes.db`. **Session-resume rule for M1:** on launch, if any session exists in the DB, Gormes attaches to the most-recent one and loads its history into the conversation viewport; otherwise it creates a new session. A session-picker and `--new` / `--session <id>` flags are M1.5 work.
6. `Ctrl+C` mid-stream cancels cleanly; the cancelled turn is persisted with `status='cancelled'`.
7. `make test` passes with ≥ 70 % coverage on `internal/` (excluding `tui/`).
8. `gormes/docs/ARCH_PLAN.md` exists and contains all six required subsections from §21.2; the Markdown-lint test at `gormes/docs/docs_test.go` passes on `ARCH_PLAN.md` and on this spec.
9. No Python file in the repo has been modified.
10. Resizing the terminal during streaming does not crash the process (verified by `teatest` resize test).
11. Completed assistant turns persist `reasoning` (when the provider emits it) and `metadata` (raw envelope) alongside visible content — verified by a DB inspection test.

---

## 19. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| OpenRouter SSE format changes or is undocumented at edges | Pin fixtures; cover error paths with `MockProvider`; integration tests skip gracefully without key |
| Bubble Tea streaming jank at high token rates | Batch token deltas with a 16 ms coalescer before emitting `UIUpdate{Token}`; measure with scripted MockProvider firing 200 tok/s |
| `modernc.org/sqlite` performance regression vs CGO sqlite | Accept — M1 persistence load is trivially small; revisit only if M2 FTS5 benchmarks warrant |
| Scope creep pulling in tool calling early | Explicit §17 out-of-scope table; any deviation requires a new spec |
| `Tool` / `pybridge` interface design wrong in hindsight | Keep M1 stub minimal (interface only, no impl); reserve breaking changes until the M4 brainstorm |
| Upstream rebase churn against Python Hermes | All Gormes files live under `gormes/`; no overlap with upstream paths |

---

## 20. Next Step

After this spec is user-approved, the `superpowers:writing-plans` skill produces the implementation plan with concrete, reviewable tasks. The plan is the input to the `executing-plans` (or `subagent-driven-development`) skill — **not** this spec.

This spec is the source of truth for *what* M0 + M1 are. The plan is the source of truth for *how* they are built.

---

## 21. Documentation Strategy

Gormes is a public fork; documentation is a first-class deliverable, not an afterthought. Three artifacts, three audiences.

### 21.1 Artifact Inventory

| File | Audience | Purpose |
|---|---|---|
| `gormes/README.md` | GitHub visitors | 60-second pitch; install + "hello world"; links to ARCH_PLAN |
| `gormes/docs/ARCH_PLAN.md` | Forkers, Python-side contributors, public-site readers | Executive roadmap: program vision, "Why Go", milestones, hybrid manifesto |
| `gormes/docs/superpowers/specs/*.md` | Implementation agents + reviewers | Per-milestone source of truth (what each spec builds) |

`ARCH_PLAN.md` is the **Executive Roadmap** — rarely changing, high-signal, the one doc a visitor reads before deciding to fork.

### 21.2 ARCH_PLAN.md — Required Sections (M0 Deliverable)

M0 ships `gormes/docs/ARCH_PLAN.md` containing **all** of the following, in this order:

1. **Rosetta Stone Declaration.** Verbatim:
   > "The repository root is the **Reference Implementation** (Python, upstream `NousResearch/hermes-agent`). The `gormes/` directory is the **High-Performance Implementation** (Go). Neither replaces the other; they co-evolve as a translation pair."

2. **Why Go — addressed to a Python developer.** Five concrete bullet points, no hype:
   - **Binary portability.** One 15–30 MB static binary. No `uv`, `pip`, venv, or system Python on the target host. `scp`-and-run on a $5 VPS or Termux.
   - **Static types + compile-time contracts.** Tool schemas, Provider envelopes, and MCP payloads become typed structs. Schema drift is a compile error, not a silent agent-loop failure.
   - **True concurrency.** Goroutines over channels replace `asyncio`. The gateway handles 10+ platform connections without event-loop starvation.
   - **Lower idle footprint.** Target ≈ 10 MB RSS at idle vs. ≈ 80+ MB for Python Hermes. Meaningful for always-on / low-spec hosts.
   - **Explicit trade-off acknowledged.** The Python AI-library moat (`litellm`, `instructor`, heavyweight ML, research skills) stays in Python; M4's Python Bridge is the seam, not an afterthought.

3. **Hybrid Manifesto — the Motherboard Strategy.**
   - **Go = chassis:** orchestrator, state, persistence, platform I/O, agent cognition.
   - **Python = peripheral library:** research tools, legacy skills, ML heavy lifting.
   - **M4 Python Bridge** is the PCIe slot; M1 ships the interface stub (`internal/pybridge`). Delegating agent cognition to subprocess RPC was considered and rejected — every turn would pay IPC latency, and the agent's identity would couple to disk I/O.

4. **5-Milestone Live Status Table.**

   | Milestone | Status | Deliverable |
   |---|---|---|
   | M0 — Scaffold | 🔨 in progress | Go module, interfaces, migrations, ARCH_PLAN |
   | M1 — TUI + LLM | 🔨 in progress | Bubble Tea Dashboard streaming OpenRouter turn |
   | M2 — Ontological Memory | ⏳ planned | FTS5, fact-triples, semantic recall |
   | M3 — Multi-Platform Gateway | ⏳ planned | Telegram / Discord / Slack concurrent adapters |
   | M4 — Python Bridge | ⏳ planned | Subprocess RPC; skills & research tools plug in |

   Status emoji legend: 🔨 in progress · ✅ complete · ⏳ planned · ⏸ deferred. The status column is updated whenever a milestone transitions; that update is a committed change to `ARCH_PLAN.md`.

5. **Project Boundaries.** Verbatim restatement of the hard rule from §16 (no Python file modified; all Gormes work under `gormes/`; one-time README-root exception is explicitly deferred).

6. **Public site pointer.** The intended public home is `https://gormes.io`. The spec does not assume any DNS / CDN / deployment is live; site deployment is explicitly M1.5 work (see §21.3).

### 21.3 Public-Site Rendering (gormes.io)

**Honest answer on Cloudflare Pages:** Cloudflare Pages serves static files but does **not** render Markdown on its own. Dropping `.md` files in a Pages project shows them as plain text. To render `ARCH_PLAN.md` on `gormes.io`, a static-site generator must convert Markdown → HTML at build time, and Pages serves the resulting `site/` directory.

**Compatibility of the current Markdown:** `ARCH_PLAN.md` and this spec are authored in **CommonMark + GFM extensions** (tables, fenced code, task lists, strikethrough). That subset renders correctly under every mainstream SSG:

| SSG | Compatible | Deployment notes |
|---|---|---|
| **MkDocs Material** | ✅ | ⭐ Recommended. `pip install mkdocs-material`, 10-line `mkdocs.yml`, `mkdocs build` → `site/` → Cloudflare Pages |
| Hugo | ✅ | Single-binary build; matches the Gormes CGO-free portability ethos |
| Astro Starlight | ✅ | TypeScript build, best theme; heavier toolchain |
| Docusaurus | ✅ | Overkill for a docs site of this size |
| Raw Pages serving `.md` | ❌ | Not rendered — shown as text. Avoid |

**M1 contract:** docs render correctly on GitHub as plain Markdown. Public-site (`gormes.io`) deployment is **M1.5** — not blocking ignition. The M1 constraint is only: write Markdown in a dialect any SSG can consume cleanly, and avoid GitHub-only features.

**M1 Markdown avoidance list** (to keep docs SSG-portable):
- No raw HTML tags, except `<br>` inside table cells when strictly needed.
- No GitHub-only admonition blocks (`> [!NOTE]`, `> [!WARNING]`). Use a bold-header callout instead.
- Relative links are repo-relative (`./foo.md`) or absolute URLs — never root-relative (`/docs/…`), because SSGs handle the prefix differently.
- Images live at `gormes/docs/images/` and are referenced by relative paths.
- No emoji in headings (breaks some SSG slug generators).

### 21.4 Maintenance

`ARCH_PLAN.md` changes only when:
- A milestone transitions state (🔨 → ✅, ⏳ → 🔨).
- The hybrid strategy changes at a principle level.
- A new milestone is added or a deferred feature is re-scoped.

Per-milestone specs under `docs/superpowers/specs/` are append-only after approval. A spec that needs substantive change is superseded by a new dated spec; the old one is kept for history.

### 21.5 Success Criteria Additions for §18

Item 8 of §18 is extended: `gormes/docs/ARCH_PLAN.md` exists **and** contains all six required subsections from §21.2 **and** passes a lint that rejects the five items from the §21.3 avoidance list.

Lint is enforced by a simple Go test at `gormes/docs/docs_test.go` that scans `ARCH_PLAN.md` and this spec for the avoidance-list patterns and fails on any match. Adding the test is part of M0.
