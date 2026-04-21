# Gormes Honcho-Parity Implementation Plan

> **For Hermes:** Use `subagent-driven-development` to execute this plan task-by-task with TDD gates.

**Goal:** Replicate the practical Honcho feature surface inside Gormes so memory is fully local, provider-free, and accessible through Honcho-like tools.

**Architecture:** Build a local `internal/honcho` service layer on top of existing `internal/memory` (3.A–3.D), then expose Honcho-compatible tool contracts: `honcho_profile`, `honcho_search`, `honcho_context`, `honcho_reasoning`, `honcho_conclude`.

**Tech Stack:** Go 1.25, SQLite (`internal/memory`), existing Hermes HTTP client for optional reasoning synthesis, tool registry in `internal/tools`.

---

## Current Baseline (already done)

Gormes already has the hard substrate needed for Honcho parity:

- Local turn persistence + FTS5 (`internal/memory/memory.go`, schema 3a)
- Entity/relationship extraction (`internal/memory/extractor.go`, schema 3b)
- Recall context injection (`internal/memory/recall.go`, schema 3c)
- Semantic embeddings (`internal/memory/embedder.go`, schema 3d)

This means we are not starting from zero; we need interface and peer-model parity.

---

## Task 1: Define Honcho parity contract (spec + acceptance tests)

**Objective:** Freeze exact behavior for each Honcho-like tool before writing implementation.

**Files:**
- Create: `gormes/docs/superpowers/specs/2026-04-21-gormes-honcho-parity-design.md`
- Create: `gormes/internal/honcho/contracts_test.go`

**Steps:**
1. Specify input/output envelopes for:
   - `honcho_profile(peer)` read/update card
   - `honcho_search(query, max_tokens, peer)` raw retrieval
   - `honcho_context(query, peer)` structured context snapshot
   - `honcho_reasoning(query, reasoning_level, peer)` synthesized answer
   - `honcho_conclude(conclusion|delete_id, peer)` fact write/delete
2. Add contract tests with fixture payloads and strict JSON shape assertions.
3. Run: `go test ./internal/honcho -run Contract -v`.

**Done when:** contracts are copy-pasteable into tool schemas and tests pass.

---

## Task 2: Add peer-card and conclusion tables to memory schema

**Objective:** Persist peer cards + durable conclusions locally in SQLite.

**Files:**
- Modify: `gormes/internal/memory/schema.go` (new migration `3d -> 3e` or next free tag)
- Modify: `gormes/internal/memory/migrate.go`
- Create: `gormes/internal/memory/honcho_sql.go`
- Create: `gormes/internal/memory/honcho_sql_test.go`

**Steps:**
1. Add schema tables:
   - `peer_cards(peer TEXT PRIMARY KEY, card_json TEXT NOT NULL, updated_at INTEGER NOT NULL)`
   - `peer_conclusions(id INTEGER PK, peer TEXT NOT NULL, conclusion TEXT NOT NULL, created_at INTEGER NOT NULL)`
2. Add migration test from prior schema versions.
3. Add CRUD SQL helpers with parameterized queries only.
4. Run: `go test ./internal/memory -run 'Migrate|HonchoSQL' -v`.

**Done when:** schema migrates cleanly and CRUD tests pass.

---

## Task 3: Implement internal Honcho service layer

**Objective:** Implement peer-scoped memory operations independent of tool transport.

**Files:**
- Create: `gormes/internal/honcho/service.go`
- Create: `gormes/internal/honcho/types.go`
- Create: `gormes/internal/honcho/service_test.go`

**Steps:**
1. Implement service methods:
   - `GetProfile(peer)` / `SetProfile(peer, []string)`
   - `Search(peer, query, maxTokens)` using turns + entities + relationships
   - `Context(peer, query)` structured snapshot
   - `Conclude(peer, text)` and `DeleteConclusion(peer, id)`
2. Keep retrieval deterministic and bounded by token budget.
3. Unit-test peer isolation and deterministic ordering.
4. Run: `go test ./internal/honcho -v`.

**Done when:** all service methods pass tests without tool-layer dependencies.

---

## Task 4: Expose Honcho-compatible tools in registry

**Objective:** Add Honcho tools to Gormes tool surface with stable schemas.

**Files:**
- Create: `gormes/internal/tools/honcho_tools.go`
- Modify: `gormes/cmd/gormes/registry.go`
- Create: `gormes/internal/tools/honcho_tools_test.go`

**Steps:**
1. Add tool structs implementing existing `tools.Tool` interface.
2. Register tools in default registry (guarded by config flag if needed).
3. Validate schema compatibility with contract tests from Task 1.
4. Run: `go test ./internal/tools -run Honcho -v`.

**Done when:** tools appear in descriptors and execute with expected JSON outputs.

---

## Task 5: Implement `honcho_reasoning` synthesis path

**Objective:** Provide optional higher-level synthesis using local context + model call.

**Files:**
- Create: `gormes/internal/honcho/reasoning.go`
- Create: `gormes/internal/honcho/reasoning_test.go`
- Modify: `gormes/cmd/gormes/telegram.go` (dependency wiring)

**Steps:**
1. Map reasoning levels (`minimal|low|medium|high|max`) to token/temperature budgets.
2. Build prompt from `Context()` result; no raw DB dumping.
3. Add fallback when model unavailable: return deterministic summary from context blocks.
4. Run: `go test ./internal/honcho -run Reasoning -v`.

**Done when:** reasoning works with and without live model endpoint.

---

## Task 6: Operator UX and diagnostics

**Objective:** Add visibility and debuggability for Honcho parity status.

**Files:**
- Modify: `gormes/cmd/gormes/doctor.go`
- Create: `gormes/internal/doctor/honcho_check.go`
- Modify: `gormes/README.md`

**Steps:**
1. Add doctor checks for Honcho tables + tool registration + peer-card read/write.
2. Add docs section: “Honcho-compatible local memory in Gormes”.
3. Run full verification:
   - `go test ./...`
   - `go test -race ./... -count=1 -timeout 300s`

**Done when:** doctor reports PASS and docs show exact usage.

---

## Immediate Execution Slice (start now)

Start with **Task 1 + Task 2 only** (contract + schema), then commit:

- Commit 1: `docs: add honcho parity design contract`
- Commit 2: `feat(memory): add peer cards and conclusions schema`

This de-risks the rest and gives stable interfaces before tool wiring.

---

## Success Criteria

1. Gormes can answer Honcho-style reads/writes with local state only.
2. Peer cards + conclusions persist across restarts.
3. Tool outputs are schema-stable and test-covered.
4. No Python or external Honcho dependency required for core functionality.
5. `go test ./...` passes after integration.
