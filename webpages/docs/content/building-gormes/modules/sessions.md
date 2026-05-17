---
title: "Sessions Module Roadmap"
---

# Sessions Module Roadmap

Generated from the single logical backlog. This page is a scoped review view; `progress.json` remains canonical.

**Module:** `sessions`
**Rows:** 27
**Status counts:** `complete`: 27 · `in_progress`: 0 · `planned`: 0
**Priority counts:** `P1`: 8 · `P2`: 4 · `P3`: 1 · `unset`: 14

## Phase 3 — The Black Box (Memory)

### 3.E.1 — Session Index Mirror

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `sessions` | Deterministic mirror refresh without mutating session state |

### 3.E.3 — Transcript Export Command

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `sessions` | gormes session export <id> --format=markdown |

### 3.E.5 — Insights Audit Log

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `sessions` | Session, token, and cost rollups from local runtime |

### 3.E.7 — Cross-Chat Synthesis

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `sessions` | user_id concept above chat_id |

### 3.E.8 — Session Lineage + Cross-Source Search

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `sessions` | parent_session_id lineage for compression splits |
| `complete` | `unset` | `sessions` | Source-filtered session/message search core |
| `complete` | `unset` | `sessions` | Lineage-aware source-filtered search hits |
| `complete` | `P1` | `sessions` | Operator-auditable search evidence |

## Phase 4 — The Brain Transplant

### 4.B — Context Engine + Compression

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `sessions` | Long session management |
| `complete` | `unset` | `sessions` | Context compression |
| `complete` | `P1` | `sessions` | Compression token-budget trigger + summary sizing |
| `complete` | `P1` | `sessions` | Aux compression single-prompt threshold reconciliation |
| `complete` | `P2` | `sessions` | Compression protected-tail multimodal length estimator |
| `complete` | `P2` | `sessions` | Context compressor image-token budget charge |
| `complete` | `P1` | `sessions` | Context references stable-handle store |
| `complete` | `unset` | `sessions` | Manual compression feedback + context references |
| `complete` | `P1` | `sessions` | Manual compression feedback renderer + focus parser |
| `complete` | `P2` | `sessions` | ContextEngine compression-boundary callback vocabulary |
| `complete` | `P2` | `sessions` | Kernel compression-boundary callback binding |
| `complete` | `P1` | `sessions` | ContextEngine session-end hook on reset |

### 4.F — Title Generation

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `sessions` | Title prompt and truncation contract |
| `complete` | `unset` | `sessions` | Title auxiliary failure visibility |
| `complete` | `unset` | `sessions` | Auto-naming sessions |

## Phase 5 — The Final Purge

### 5.O — Hermes CLI Parity

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `sessions` | Busy command guard for compression and long CLI actions |
| `complete` | `P1` | `sessions` | Hermes sessions CLI MRU browse/delete ergonomics |

## Phase 6 — The Learning Loop (Soul)

### 6.J — Agentic Memory Lifecycle (AgeMem)

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `sessions` | Cross-session memory continuity |

### 6.K — Self-Evolution Engine (GEPA)

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P3` | `sessions` | Behavioral pattern extraction from session logs |
