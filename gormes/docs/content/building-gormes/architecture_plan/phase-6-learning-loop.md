---
title: "Phase 6 — The Learning Loop (Soul)"
weight: 70
---

# Phase 6 — The Learning Loop (Soul)

**Status:** ⏳ planned · 5/6 sub-phases

The Learning Loop is the first Gormes-original core system — not a port. It detects when a task is complex enough to be worth learning from, distills the solution into a reusable skill, stores it, and improves the skill over successive runs. Upstream Hermes alludes to self-improvement; Gormes implements it as a dedicated subsystem.

> "Agents are not prompts. They are systems. Memory + skills > raw model intelligence."

## Sub-phase outline

| Subphase | Status | Deliverable |
|---|---|---|
| 6.A — Complexity Detector | ✅ complete | Heuristic signal for "this turn was worth learning from" now ships via `internal/learning/runtime.go`, with kernel-written JSONL decisions under `${XDG_DATA_HOME}/gormes/learning/complexity.jsonl` |
| 6.B — Skill Extractor | ✅ complete | LLM-assisted pattern distillation now ships via `internal/learning/extractor.go`, gating on the 6.A signal and persisting JSONL skill candidates for 6.C |
| 6.C — Skill Storage Format | ✅ complete | Portable SKILL.md with an optional `learning:` frontmatter block (session, distilled_at, score, threshold, reasons, tool_names) ships via `internal/skills/portable.go`; legacy documents keep parsing unchanged |
| 6.D — Skill Retrieval + Matching | ✅ complete | `internal/skills.SelectHybrid` fuses the existing token-overlap lexical scorer with cosine similarity over per-skill embeddings via Reciprocal Rank Fusion |
| 6.E — Feedback Loop | ✅ complete | Per-skill outcome log + Laplace-smoothed effectiveness score now ships via `internal/learning/feedback.go` |
| 6.F — Skill Surface (TUI + Telegram) | ✅ complete (browsing only) | Shared `skills.BrowseView` now powers the gateway `/skills` command and `tui.RenderSkillsPane`; edit + disable flows remain follow-on |

## Why this is Phase 6 and not Phase 5.F

Phase 5.F (Skills system) was previously scoped as "port the upstream Python skills plumbing". That's mechanical. Phase 6 is the algorithm on top — detecting complexity, distilling patterns, scoring feedback. It depends on 5.F (needs the storage format), but it's not the same work.

Positioning: **Gormes's moat over Hermes**. Hermes has a skills directory; it does not have a native learning loop that decides what's worth writing down.

## 6.C Closeout

Phase 6.C lands the portable storage format that bridges the 6.B extractor to the 6.D / 6.F retrieval and browsing surfaces already on disk. `internal/skills/portable.go` introduces a `Provenance` struct plus `RenderPortable` / `ParsePortable` that extend the existing SKILL.md frontmatter with an optional `learning:` block carrying:

- `session_id` — the originating kernel session so operators can trace a skill back to the turn that produced it.
- `distilled_at` — RFC 3339 UTC timestamp of the distillation.
- `score` / `threshold` — the 6.A complexity score and the gate it crossed, always emitted (even at zero) so below-threshold manual promotions are legible.
- `reasons` / `tool_names` — the 6.A signal reasons and the tool trace, rendered as flow-style string sequences.

Rendering is byte-deterministic and omits blank scalars or empty lists so the file stays compact when provenance is partial. Parsing round-trips every field and — critically — the legacy `skills.Parse` seam still accepts portable docs without code changes: the existing 6.D retrieval and 6.F browsing consumers snapshot the same `Skill` struct as before, with the `learning:` block treated as an additive header that they can opt into via `ParsePortable` when they want the provenance metadata.

A missing `learning:` block parses to a nil `Provenance` pointer instead of an all-zero struct so callers can distinguish "never had provenance" from "had below-threshold provenance" without inspecting every scalar. That keeps the door open for 6.E ranking policies that weight skills by how recently they were distilled without having to special-case pre-6.C skills.

## 6.B Closeout

Phase 6.B lands the LLM-assisted extractor half of the learning loop. `internal/learning/extractor.go` introduces a narrow `LLM` seam (`Distill(ctx, prompt) (DistillResponse, error)`) and an `Extractor` that:

- Gates on the 6.A `Signal` — turns that did not cross the worth-learning threshold short-circuit before any prompt is built or any file is touched, so the extractor inherits 6.A's auditability contract.
- Renders a deterministic prompt from the `Source` — session ID, signal reasons and tool names, the user and assistant messages, and each tool event's args + result — so replaying the same turn reproduces byte-identical input to the model.
- Validates the returned `DistillResponse` (Name, Description, Body must all be non-blank) before accepting the distillation; invalid responses and LLM errors propagate upward without leaving a partial JSONL line behind.
- Appends each accepted skill proposal as a `Candidate` JSONL record alongside the scoring metadata (score, threshold, reasons, tool names, distilled-at timestamp), matching the append-only `${XDG_DATA_HOME}/gormes/learning/...` convention already set by 6.A and 6.E.

Wiring the live LLM seam into the kernel is deferred to the 6.C storage-format slice, which will resolve how `Candidate` records become SKILL.md artefacts on disk. 6.B ships the algorithm; 6.C will ship the file layout.

## 6.A Closeout

Phase 6.A now lands a deterministic heuristic detector instead of waiting for a full extractor pipeline. `internal/learning/runtime.go` scores each successful turn on five cheap signals: any tool use, multi-tool use, prompt/completion token volume, transcript size, and wall-clock duration. `internal/kernel/kernel.go` records that decision after successful turns, while the shared TUI, gateway, Telegram, ACP, and BOOT entrypoints all wire the same runtime so the detector is active everywhere the kernel runs.

The output is deliberately narrow and auditable: one append-only JSONL record per completed turn at `${XDG_DATA_HOME}/gormes/learning/complexity.jsonl`, including the score, threshold, reasons, tool names, and raw metrics. That gives Phase 6.B a stable gate for "worth learning from" without prematurely coupling this slice to skill extraction or promotion logic.

## 6.D Closeout

Phase 6.D lands the retrieval contract before the extractor (6.B) or storage format (6.C) so callers wiring future skill suggestion paths have a single deterministic ranking entrypoint to target. `internal/skills/hybrid.go` introduces `SelectHybrid(skills, embeddings, queryText, queryEmbedding, max)`, a Reciprocal Rank Fusion (RRF, k=60) combination of two independent rankings:

- **Lexical** reuses the existing `scoreSkill`/`tokenize` pair so name hits stay weighted ahead of description and body hits, preserving the legacy `Select` behavior when no embedding is provided.
- **Semantic** computes cosine similarity between the query embedding and a per-skill embedding (parallel to the skills slice). A nil entry means "not yet embedded" — the skill still participates via lexical alone instead of being penalized.

RRF is deliberately chosen over a weighted linear combination so that callers do not need to tune or normalize raw scores when introducing a new embedder: each skill simply earns `1/(60 + rank)` from each ranking it appears in, and ties are broken deterministically by skill name then path. Defensive guards (mismatched dimensions yield a zero cosine, an empty `queryEmbedding` collapses the function to lexical-only, an empty query yields nil) keep the selector safe to call before the embedder backfill catches up. Phase 6.B and 6.C can now be sequenced against this stable ranking surface instead of inventing a parallel one.

## 6.E Closeout

Phase 6.E lands the scoring half of the feedback loop ahead of the remaining extractor work. `internal/learning/feedback.go` adds a `FeedbackStore` that appends one `Outcome` record per (skill, turn) pair as JSONL, then replays that log to produce per-skill `EffectivenessScore` aggregates. The score uses Laplace smoothing — `(successes + 1) / (uses + 2)` — so a brand-new skill starts at the neutral prior `0.5` and converges to the observed success ratio as samples accumulate.

Callers who rank skills at selection time can consult `FeedbackStore.Weight(ctx, name)` and multiply the returned weight directly into relevance scores without special-casing fresh skills: unknown names, blank names, and log read errors all fall back to `0.5` instead of returning a zero that would suppress untested skills. The store is append-only and self-contained, matching the auditability contract already set by the Phase 6.A complexity log.

## 6.F Closeout (browsing)

Phase 6.F lands the browsing half of the Skill Surface. `internal/skills/browse.go` introduces a shared `BrowseView` plus `FormatBrowseSummary` helper that sorts installed and hub-available skills deterministically and paginates them into one Telegram-sized or TUI-pane-sized page. Both edges now consume the same helper so operators see identical listings:

- Gateway `/skills` is wired through `internal/gateway/skills_command.go` and a new `SkillsBrowser` seam on `ManagerConfig`; `cmd/gormes/skills_browser.go` backs it with `skills.Hub` so Telegram, Discord, and any future shared-chassis adapter deliver the same text on demand.
- TUI `internal/tui.RenderSkillsPane(view, width)` renders the same summary through `lipgloss` width-aware wrapping, keeping the TUI surface ready to reuse the Telegram payload without re-implementing formatting.

Editing and disabling skills from the TUI or messaging edge remain explicit follow-on scope — the browsing contract shipped here gives those flows a single source of truth to build against.
