---
title: "Phase 6 — The Learning Loop (Soul)"
weight: 70
---

# Phase 6 — The Learning Loop (Soul)

**Status:** ⏳ planned · 0/6 sub-phases

The Learning Loop is the first Gormes-original core system — not a port. It detects when a task is complex enough to be worth learning from, distills the solution into a reusable skill, stores it, and improves the skill over successive runs. Upstream Hermes alludes to self-improvement; Gormes implements it as a dedicated subsystem.

> "Agents are not prompts. They are systems. Memory + skills > raw model intelligence."

## Sub-phase outline

| Subphase | Status | Deliverable |
|---|---|---|
| 6.A — Complexity Detector | ⏳ planned | Heuristic (or LLM-scored) signal for "this turn was worth learning from" |
| 6.B — Skill Extractor | ⏳ planned | LLM-assisted pattern distillation from the conversation + tool-call trace |
| 6.C — Skill Storage Format | ⏳ planned | Portable, human-editable Markdown (SKILL.md) with structured metadata |
| 6.D — Skill Retrieval + Matching | ⏳ planned | Hybrid lexical + semantic lookup for relevant skills at turn start |
| 6.E — Feedback Loop | ⏳ planned | Did the skill help? Adjust weight. Surface usage stats to operator |
| 6.F — Skill Surface (TUI + Telegram) | ⏳ planned | Browse, edit, disable skills from the CLI or messaging edge |

## Why this is Phase 6 and not Phase 5.F

Phase 5.F (Skills system) was previously scoped as "port the upstream Python skills plumbing". That's mechanical. Phase 6 is the algorithm on top — detecting complexity, distilling patterns, scoring feedback. It depends on 5.F (needs the storage format), but it's not the same work.

Positioning: **Gormes's moat over Hermes**. Hermes has a skills directory; it does not have a native learning loop that decides what's worth writing down.
