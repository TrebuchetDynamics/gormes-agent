---
title: "Phase 6 — The Learning Loop (Soul)"
weight: 70
---

# Phase 6 — The Learning Loop (Soul)

**Status:** 🚧 in progress · Hermes background-review and curator parity rows
are now tracked explicitly; core automatic promotion/scoring remains planned.
6.K prompt evaluation + optimization rows are validated.

**Completion lane:** Phase 6 is [Lane 6 — Learning Loop](../lane-roadmap/#lane-6--learning-loop).
It depends on the Phase 5.F skills substrate and should not begin with live LLM
skill extraction. Ship detector, storage, extractor schema, retrieval,
feedback, and operator surfaces as separate fixture-backed rows.

The Learning Loop has two layers. Hermes now defines the compatibility floor:
after-turn background review forks can update memory/skills, and the curator can
maintain agent-created skills over time. Gormes ports those user-visible
contracts first, then adds Gormes-native evidence gates for detection, scoring,
retrieval, and promotion.

> "Agents are not prompts. They are systems. Memory + skills > raw model intelligence."

## Sub-phase outline

| Subphase | Status | Deliverable |
|---|---|---|
| 6.A — Complexity Detector | 🚧 partial | Hermes background-review fork lifecycle is row-backed; deterministic local trigger signals remain planned |
| 6.B — Skill Extractor | ⏳ planned | LLM-assisted pattern distillation from the conversation + tool-call trace, with fake-model fixtures and secret/noise rejection gates |
| 6.C — Skill Storage Format | ⏳ planned | Portable, human-editable SKILL.md with versioned metadata, provenance, review state, and atomic writes |
| 6.D — Skill Retrieval + Matching | ⏳ planned | Hybrid lexical + Phase 3 semantic lookup for relevant reviewed skills at turn start, plus optional Code Cathedral II-style code-context evidence after the base scorer is stable |
| 6.E — Feedback Loop | ⏳ planned | Hermes curator state transitions and run reports, then skill-use outcomes, explicit operator feedback, and auditable weight adjustments |
| 6.F — Skill Surface (TUI + Telegram) | 🚧 partial | Hermes curator CLI surface plus browse, edit, disable, and review skills from the TUI or messaging edge after store/feedback contracts are stable |
| 6.K — Self-Evolution Engine (GEPA) | 🚧 partial | Prompt evaluation harness and iterative prompt mutation/scoring loop are validated; behavioral pattern extraction remains planned |
| 6.L — Composable Skill Execution (Voyager) | ⏳ planned | Sandbox executable skills, dependency resolution, and validation remain future rows |

## Hermes Parity Floor

Upstream Hermes at `b816fd4e2` makes the learning loop concrete in two places:

- `run_agent.py` spawns an after-turn background review fork with active runtime
  credentials, memory+skills toolsets only, auto-deny approval behavior,
  parent-session attribution, isolated prompt history, cleanup, and one
  user-visible `Self-improvement review` summary.
- `agent/curator.py` and `hermes_cli/curator.py` add autonomous skill
  maintenance: interval/paused gates, first-run defer, activity-based
  active/stale/archived transitions, pinned-skill safeguards, dry-run reports,
  backups, rollback/restore, and `hermes curator` status/run/control commands.

Gormes already has several prerequisites: `skill_manage`, `skills_list`,
`skill_view`, validated SKILL.md storage, skill retrieval fixtures, and the
memory+skills-only background review toolset policy. The missing rows are the
background review fork lifecycle, curator state/report engine, and curator CLI.

## Why this is Phase 6 and not Phase 5.F

Phase 5.F (Skills system) was previously scoped as "port the upstream Python skills plumbing". That's mechanical. Phase 6 is the algorithm on top — detecting complexity, distilling patterns, scoring feedback. It depends on 5.F (needs the storage format), but it's not the same work.

Positioning: **Hermes-compatible self-improvement, Go-native safety gates**.
Hermes defines the background-review and curator behavior users can observe.
Gormes keeps those semantics while making the scheduler, reports, skill storage,
and operator controls testable without Python runtime assumptions.

## Hermes Skill Lessons

Skills are code-like runtime assets, not loose notes. The current skill rows show
the value of procedural knowledge with resolver checks and conformance tests.
Hermes shows the value and risk of large skill surfaces injected into prompts.
Gormes should combine the useful parts:

- active skills require valid metadata, triggers, exclusions, provenance, and
  review state;
- disabled or unreviewed skills never enter prompt injection;
- resolver routes have fixtures for confusing user phrases;
- skill selection records are tied to turn outcome and operator feedback;
- generated skill drafts are inactive until reviewed;
- updates preserve version history and source evidence;
- secret stripping and one-off task rejection are mandatory gates.

The code-context retrieval rows keep the useful shape: qualified symbols, parent-scope chunks,
call-graph edges, and two-pass retrieval. For Gormes this is a retrieval
evidence lesson, not a runtime dependency. Phase 6.D now keeps that drift as a
small blocked row: define synthetic code-context evidence and fan-out caps that
the skill scorer can explain before any tree-sitter, WASM grammar, or repo-wide
backfill decision.

The learning loop is allowed to draft and improve skills only after the storage,
resolver, review, and feedback records are testable. Otherwise "self-improving"
becomes unreviewed prompt mutation.

## TDD Execution Notes

Do not begin Phase 6 with live LLM extraction. The dependency order is:

1. **6.A background review fork lifecycle** — port Hermes runtime inheritance,
   memory+skills-only toolset restriction, summary attribution, and cleanup with
   fake review workers.
2. **6.C storage extension** — extend the Phase 2.G store with versioned
   metadata, provenance, review state, and atomic writes before generated skills
   can persist.
3. **6.E curator state/report engine** — port Hermes first-run defer,
   interval/paused gates, activity transitions, dry-run/report behavior, and
   pinned/manual safeguards before exposing the command.
4. **6.F curator CLI** — make `gormes curator` available only after it can read
   real native curator state and reports.
5. **6.A deterministic detector** — prove local trigger signals are explainable
   and replayable from transcript/tool-call fixtures.
6. **6.B extractor schema** — use fake model outputs to prove accepted/rejected
   skill drafts, secret stripping, and one-off task rejection.
7. **6.D retrieval scorer** — combine lexical and semantic signals while
   excluding disabled or unreviewed skills from prompt injection.
8. **6.E feedback records** — persist outcomes before any automatic
   promotion/demotion or weight change.
9. **6.F operator surfaces** — expose review/edit/disable flows only after the
   underlying store and feedback records are stable.

### 6.K Self-Evolution Row Status

The GEPA lane is now test-backed but remains offline and deterministic:

- **Prompt evaluation harness** is complete. `internal/hermes/prompt_evaluator.go`
  evaluates prompt variants against injected scenario runners, records
  `task_success`, `tool_accuracy`, `response_quality` on a 1-5 scale, and
  aggregates variant scores. `internal/hermes/eval_scenarios.go` provides a
  10-scenario local corpus.
- **Iterative prompt mutation and scoring loop** is complete.
  `internal/hermes/prompt_optimizer.go` generates bounded tool-selection,
  response-quality, task-decomposition, and command-safety mutations, scores
  them through the harness, and stops on convergence, perfect score, or budget.
- **Behavioral pattern extraction from session logs** is still planned. Do not
  promote prompt mutations from live logs until the extractor row has
  fixture-backed success/anti-pattern evidence and operator review rules.

## Go donor pointers

Hermes owns the background-review and curator contracts; Gormes owns the
Go-native implementation and safety evidence. Automatic scoring/promotion rows
remain Gormes-native unless a later Hermes source introduces a stricter
contract. Surrounding plumbing has donors:

| Phase 6 problem | Donor file | Notes |
|---|---|---|
| 6.A complexity detector — bounded transcript-size budget | `axe/internal/budget/budget.go` | Per-turn counter + overflow signal |
| 6.A complexity detector — append-only signal log | `engram/internal/mcp/activity.go` | Audit shape, redaction |
| 6.B extractor schema — secret stripping at ingest boundary | `nanobot/pkg/agents/truncate.go` | Sanitize/truncate before persistence |
| 6.C skill storage — versioned metadata + atomic writes | `engram/internal/store/store.go` | DDL + migration helpers |
| 6.C skill storage — sanitized artifact paths for stored evidence | `axe/internal/artifact/tracker.go` | Path-traversal guard |
| 6.D retrieval scorer — bounded fan-out cap for code-context evidence | `axe/internal/budget/budget.go` | Reset + overflow signal |
| 6.D retrieval scorer — provenance-aware ranking signals | `engram/internal/store/relations.go` | Provenance edges (`scoped`, `supersedes`) |
| 6.E feedback records — outcome ledger before promotion/demotion | `engram/internal/mcp/activity.go` | Append-only outcome log |
| 6.F operator review surfaces — workflow agent pattern | `adk-go/agent/workflowagents/...` | Loop / sequential / parallel primitives |

Route through the `gormes-references` skill
(`docs/development-skills/gormes-references/SKILL.md`) before re-deriving any
of these shapes.
