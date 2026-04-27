---
title: "Memory"
weight: 30
---

# Memory

Persistent, searchable state that outlives the process. Structured enough for graph traversal; flat enough for `grep`.

## Components shipped today

- **SQLite + FTS5 lattice** (3.A) — `internal/memory/SqliteStore`. Schema migrations, fire-and-forget worker, lexical search.
- **Ontological graph** (3.B) — entities, relationships, LLM-assisted extractor with dead-letter queue.
- **Neural recall** (3.C) — 2-layer seed selection, CTE traversal, `<memory-context>` fence injection matching Hermes's `build_memory_context_block`.
- **Semantic fusion** (3.D) — Ollama embeddings, cosine recall, and hybrid lexical+semantic seed fusion.
- **USER.md mirror** (3.D.5) — async export of entity/relationship graph to human-readable Markdown. Gormes-original; no upstream equivalent.
- **Tool audit JSONL** (3.E.2) — append-only JSONL from kernel and `delegate_task` tool execution with timing, outcome, and error fields.
- **Transcript export** (3.E.3) — `gormes session export <id> --format=markdown` renders SQLite turns, timestamps, and tool calls for operator sharing.
- **Operator visibility** (3.E.4, 3.E.5) — `gormes memory status` is shipped, and the local insights layer now persists append-only daily `usage.jsonl` records from `telemetry.Snapshot` rollups.
- **GONCHO compatibility seam** — internal memory work lives behind the `goncho` service, while the exported tool surface remains Honcho-compatible (`honcho_*`).

## Phase 3 closeout queue

- **Shipped visibility spine** (3.E.1–3.E.5) — session index mirror, tool audit, transcript export, memory status, and daily insights logging are landed.
- **`last_seen` closeout** (3.E.6) — shipped: schema v3g backfills `relationships.last_seen`, repeated relationship observations advance it without rewriting legacy `updated_at`, and recall attenuation uses `COALESCE(NULLIF(last_seen, 0), updated_at)`.
- **Cross-chat identity closeout** (3.E.7) — shipped: GONCHO identity hierarchy is `user_id > chat_id > session_id`; `internal/session` persists canonical chat-to-user bindings, and `internal/memory`, `internal/goncho`, and `internal/gonchotools` cover same-chat default fencing, opt-in canonical user/source-filtered recall, Honcho-compatible schemas, host mappings, SillyTavern persona/group-chat mapping, deny paths, and operator-readable evidence.
- **Session lineage + cross-source search closeout** (3.E.8) — shipped: source-filtered search spans one canonical `user_id` across chats inside `internal/memory` and the internal GONCHO service; `parent_session_id`, compression-continuation resume, lineage-aware hits, and operator-auditable search evidence are validated.
- **Goncho/Honcho parity** (3.F) — shipped and converged: context representation options, typed search filters, directional peer cards, queue status, summary budgeting, dialectic chat, file import, topology fixtures, operator diagnostics, streaming persistence, `[goncho]` config, and dream-scheduler intent are validated while public tools remain `honcho_*`.

## Identity + lineage contract

- **GONCHO identity hierarchy** — `user_id > chat_id > session_id`.
- **Recall fence** — same-chat by default; opt-in cross-chat only when a canonical `user_id` resolves.
- **Tool boundary** — `honcho_search` and `honcho_context` preserve the external Honcho-compatible tool names and now advertise `scope` / `sources` while the implementation stays in the internal `goncho` package.
- **Lineage rule** — `parent_session_id` is append-only metadata on descendants, not a rewrite of ancestor history.
- **Implementation plan** — `docs/superpowers/plans/2026-04-22-gormes-phase3-identity-lineage-plan.md`.
- **Execution plan** — `docs/superpowers/plans/2026-04-22-gormes-phase3-identity-lineage-execution-plan.md`; the identity and lineage sequence it described has now landed. Future memory work should add new small 3.F or Phase 6 rows rather than reopening the shipped cross-chat spine.

## Why this is not just "chat logs"

Chat logs are append-only. Memory has schema. You query it, derive from it, inject it back into the context window. The SQLite + FTS5 combination gives you ACID durability *and* full-text search in a single ~100 KB binary dependency.

See [Phase 3](../../architecture_plan/phase-3-memory/) for the full sub-status.
