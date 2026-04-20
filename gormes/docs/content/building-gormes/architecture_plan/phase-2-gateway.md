---
title: "Phase 2 — The Gateway"
weight: 30
---

# Phase 2 — The Gateway (Wiring Harness)

**Status:** 🔨 in progress

**Deliverable:** Go-native wiring harness: tools, Telegram, and thin session resume land before the wider gateway surface.

## Phase 2 Ledger

| Subphase | Status | Priority | Deliverable |
|---|---|---|---|
| Phase 2.A — Tool Registry | ✅ complete | P0 | In-process Go tool registry, streamed `tool_calls` accumulation, kernel tool loop, and doctor verification |
| Phase 2.B.1 — Telegram Scout | ✅ complete | P1 | Telegram adapter over the existing kernel, long-poll ingress, edit coalescing at the messaging edge |
| Phase 2.C — Thin Mapping Persistence | ✅ complete | P0 | bbolt-backed `(platform, chat_id) -> session_id` resume; no transcript ownership moved into Go |
| **Phase 2.E — Subagent System** | ⏳ planned | **P0** | **Execution isolation model:** spawn parallel workstreams with resource boundaries, context isolation, cancellation scopes, and failure containment. NOT a port of Python's loose process model—Gormes implements real subagents with deterministic lifecycle management |
| Phase 2.D — Cron / Scheduled Automations | ⏳ planned | P2 | Port `cron/scheduler.py` + `cron/jobs.py` to a Go ticker + bbolt job store; natural-language cron parsing via the brain (Phase 4) once available |
| Phase 2.B.2+ — Wider Gateway Surface | ⏳ planned | P1 | Additional platform adapters (Discord, Slack, WhatsApp, Signal, Email, SMS, etc.) |
| Phase 2.F — Hooks + Lifecycle | ⏳ planned | P2 | Port `gateway/hooks.py`, `builtin_hooks/`, `restart.py`, `pairing.py`, `status.py`, `mirror.py`, `sticker_cache.py`; per-event extension points and managed restarts |
| **Phase 2.G — Skills System** | ⏳ planned | **P0** | **The Learning Loop:** detect complex tasks, extract reusable patterns, save as versioned skills, improve over time. This is THE differentiation—without it, Gormes is just a chatbot with tools. See Phase 6 for the full learning loop architecture |

Phase 2.C is intentionally not Phase 3. It stores only session handles in bbolt. Python still owns transcript memory, transcript search, and prompt assembly; the SQLite + FTS5 memory lattice is Phase 3 (now substantially implemented).

> **Note on binary size:** The static CGO-free binary currently builds at **~17 MB** (measured: `bin/gormes` from `make build` with `-trimpath -ldflags="-s -w"` at commit `4a25542c`, post-3.D). This reflects all Phase 3 additions (extractor, recall, mirror, Embedder, semantic fusion) atop the original TUI + Telegram base. Remains well within the 25 MB hard moat with 8 MB headroom.
