---
title: "Fleet Module Roadmap"
---

# Fleet Module Roadmap

Generated from the single logical backlog. This page is a scoped review view; `progress.json` remains canonical.

**Module:** `fleet`
**Rows:** 23
**Status counts:** `complete`: 21 · `in_progress`: 0 · `planned`: 2
**Priority counts:** `P0`: 2 · `P1`: 7 · `P2`: 6 · `P3`: 3 · `unset`: 5

## Phase 1 — The Dashboard

### 1.C — Automation Reliability

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `fleet` | Orchestrator failure-row stabilization for 4-8 workers |
| `complete` | `P0` | `fleet` | Soft-success-nonzero bats coverage |
| `complete` | `unset` | `fleet` | Autoloop row health and quarantine contract |
| `complete` | `P1` | `fleet` | Watchdog checkpoint coalescing |
| `complete` | `P1` | `fleet` | PR-intake idle backoff |
| `complete` | `P1` | `fleet` | Watchdog dead-process vs slow-progress separation |

## Phase 2 — The Gateway

### 2.D — Cron / Scheduled Automations

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `fleet` | robfig/cron scheduler + bbolt job store |
| `complete` | `unset` | `fleet` | SQLite cron_runs audit + CRON.md mirror |
| `complete` | `unset` | `fleet` | Heartbeat [SYSTEM:] + [SILENT] delivery contract |
| `planned` | `P1` | `fleet` | Durable operator run report for unattended jobs |
| `planned` | `P1` | `fleet` | Scheduled briefing job emits operator run report |

### 2.E.1 — OS-AI Spine: Delegation Policy + Child Execution

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `fleet` | Durable job routing policy |

### 2.E.3 — OS-AI Spine: Durable Job Resilience

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `fleet` | Durable job backpressure + timeout audit |
| `complete` | `P3` | `fleet` | Durable worker supervisor status seam |
| `complete` | `P2` | `fleet` | Durable pause/resume intent contract |
| `complete` | `P2` | `fleet` | Durable replay and inbox message contract |
| `complete` | `P2` | `fleet` | Durable worker execution loop |
| `complete` | `P2` | `fleet` | Durable worker abort-slot recovery safety net |
| `complete` | `P3` | `fleet` | Durable worker RSS watchdog policy helper |
| `complete` | `P3` | `fleet` | Durable worker RSS drain integration |

## Phase 5 — The Final Purge

### 5.N — Misc Operator Tools

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `fleet` | Autoloop recent-failure detail excerpts |
| `complete` | `P1` | `fleet` | System Events, Heartbeat, and Presence |
| `complete` | `P0` | `fleet` | Cron no-agent script-only watchdog mode |
