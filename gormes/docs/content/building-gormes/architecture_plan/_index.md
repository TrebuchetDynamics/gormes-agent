---
title: "Architecture Plan"
weight: 10
---

# Gormes — Executive Roadmap

**Public site:** https://gormes.ai
**Source:** https://github.com/TrebuchetDynamics/gormes-agent
**Upstream reference:** https://github.com/NousResearch/hermes-agent

**Positioning:** Production runtime for self-improving agents. Hermes-class capabilities without Python.

---

## 0. Operational Moat Thesis

When intelligence becomes abundant, operational friction becomes the bottleneck.

That is the reason Gormes exists.

**Gormes is not an AI agent. It is the operating system for agents**—infrastructure that manages agent lifecycles, memory, tool execution, and platform gateways. Your real competitors aren't other chatbots. They're Docker + Python stacks (complex orchestration), serverless workflows (cold start latency), and workflow engines (limited autonomy). Gormes wins by being simpler, faster, and more reliable than all of them.

The strategic target is not "a Go wrapper around Hermes." The strategic target is a single-binary runtime that stays alive 24/7 without Python, environments, or orchestration overhead.

---

## 1. Rosetta Stone Declaration

The repository root is the **Reference Implementation** (Python, upstream `NousResearch/hermes-agent`). The `gormes/` directory is the **High-Performance Implementation** (Go). Neither replaces the other during Phases 1–4; they co-evolve as a translation pair until Phase 5's final purge completes the migration.

---

## Milestone Status

| Phase | Status | Deliverable |
|---|---|---|
| Phase 1 — The Dashboard | ✅ complete | Tactical Go TUI bridge over Python's `api_server` |
| Phase 2 — The Gateway | 🔨 in progress | Go-native tools + Telegram + session resume + wider adapters |
| Phase 3 — The Black Box (Memory) | 🔨 substantially complete | SQLite + FTS5 + graph + recall + USER.md mirror |
| Phase 4 — The Brain Transplant | ⏳ planned | Native Go agent orchestrator + prompt builder (Hermes-off) |
| Phase 5 — The Final Purge | ⏳ planned | 100% Go — Python tool scripts ported |
| **Phase 6 — The Learning Loop (Soul)** | ⏳ planned | Native skill extraction. Compounding intelligence. The feature Hermes doesn't have. |

Legend: 🔨 in progress · ✅ complete · ⏳ planned.

Each phase has a dedicated page below with sub-phase breakdowns and current state.
