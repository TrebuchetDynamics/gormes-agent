---
title: "Memory Module Roadmap"
aliases:
  - /building-gormes/modules/memory/
---

# Memory Module Roadmap

Generated from the single logical backlog. This page is a scoped review view; `progress.json` remains canonical.

**Module group:** Agent Runtime
**Module:** `memory`
**Rows:** 29
**Status counts:** `complete`: 29 · `in_progress`: 0 · `planned`: 0
**Priority counts:** `P1`: 1 · `P2`: 3 · `P3`: 1 · `unset`: 24

## Phase 3 — The Black Box (Memory)

### 3.A — SQLite + FTS5 Lattice

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `memory` | SqliteStore |
| `complete` | `unset` | `memory` | FTS5 triggers |

### 3.B — Ontological Graph + LLM Extractor

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `memory` | Extractor |
| `complete` | `unset` | `memory` | Entity/relationship upsert |
| `complete` | `unset` | `memory` | Dead-letter queue |

### 3.C — Neural Recall + Context Injection

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `memory` | RecallProvider |
| `complete` | `unset` | `memory` | 2-layer seed selection |
| `complete` | `unset` | `memory` | CTE traversal |
| `complete` | `unset` | `memory` | <memory-context> fence |

### 3.D — Semantic Fusion + Local Embeddings

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `memory` | Vector cache |
| `complete` | `unset` | `memory` | Cosine similarity recall |
| `complete` | `unset` | `memory` | Hybrid fusion |

### 3.D.5 — Memory Mirror (USER.md sync)

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `memory` | Async background export |
| `complete` | `unset` | `memory` | SQLite as source of truth |

### 3.E.4 — Extraction State Visibility

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `memory` | gormes memory status command |
| `complete` | `unset` | `memory` | Extractor queue depth + dead-letter summary |

### 3.E.5 — Insights Audit Log

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `memory` | Append-only daily usage.jsonl writer |

### 3.E.6 — Memory Decay

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `memory` | relationships.last_seen schema + backfill |
| `complete` | `unset` | `memory` | Relationship writer freshness updates |
| `complete` | `unset` | `memory` | Deterministic weight attenuation at recall time |

### 3.E.7 — Cross-Chat Synthesis

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `memory` | Same-chat default recall fence |
| `complete` | `unset` | `memory` | Opt-in user-scope recall + source filters |
| `complete` | `P2` | `memory` | Interrupted-turn memory sync suppression |
| `complete` | `P3` | `memory` | SillyTavern persona and group-chat mapping fixtures |
| `complete` | `unset` | `memory` | Cross-chat deny-path fixtures |
| `complete` | `unset` | `memory` | Cross-chat operator evidence |

## Phase 6 — The Learning Loop (Soul)

### 6.G — Structured Memory Types

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `memory` | 6 typed memory categories with confidence scoring |

### 6.I — Zero-LLM Knowledge Graph

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `memory` | Regex-based auto-link extraction + brain-first lookup |

### 6.J — Agentic Memory Lifecycle (AgeMem)

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `memory` | Agent-controlled memory retention with importance scoring |
