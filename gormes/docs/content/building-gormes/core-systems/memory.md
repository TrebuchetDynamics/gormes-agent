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

## Remaining Phase 3 queue

- **Session and tool mirrors** (3.E.1, 3.E.2) — human-readable session index plus append-only tool audit JSONL.
- **Transcript export + extractor status** (3.E.3, 3.E.4) — operator commands for exporting a session and inspecting queue/dead-letter health.
- **Insights, decay, and cross-chat identity** (3.E.5, 3.E.6, 3.E.7) — lightweight usage log, `last_seen`-driven decay, and one-user-many-chats graph unification.
- **Session lineage + cross-source search** (3.E.8) — the remaining `SessionDB` donor seam, paired with later compression work.

## Why this is not just "chat logs"

Chat logs are append-only. Memory has schema. You query it, derive from it, inject it back into the context window. The SQLite + FTS5 combination gives you ACID durability *and* full-text search in a single ~100 KB binary dependency.

See [Phase 3](../architecture_plan/phase-3-memory/) for the full sub-status.
