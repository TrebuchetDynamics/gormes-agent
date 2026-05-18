---
title: "Learning Loop"
weight: 20
---

# The Learning Loop (The Soul)

Detects when a task was complex enough to learn from, distills the solution
into a reusable skill, stores it, improves the skill over successive runs, and
keeps agent-created skills maintainable through Hermes-compatible background
review and curator flows.

## Simplified flow

```go
if taskComplexity(turn) > threshold {
    skill := extractSkill(conversation, toolCalls)
    store.Save(skill)
}
```

## Why this is load-bearing

Without a learning loop you lose:

- **Compounding intelligence** — the bot doesn't get smarter at *your* workflows over time
- **Differentiation** — every agent looks the same at turn zero
- **Long-term value** — you pay the same token tax on turn 1000 as on turn 1

Upstream Hermes now has two concrete learning-loop contracts: an after-turn
background review fork that can save memory/skill updates, and a curator that
maintains agent-created skills through activity-based lifecycle transitions,
dry-run reports, backups, rollback/restore, and `hermes curator` commands. The
curator also has a dedicated `auxiliary.curator` model slot, so review work can
use a cheaper or slower side model without changing the main chat model. Phase
6 ports those contracts first, including the `skill_manage` support-file actions
the curator prompt uses to demote narrow content into references/templates/scripts
or assets, then adds Gormes-native evidence for detectors, scoring, retrieval,
and promotion.

## Current status

🚧 Partial — see [Phase 6](../../architecture_plan/phase-6-learning-loop/) for
the sub-phase breakdown. Gormes already has SKILL.md validation, skill
metadata/retrieval fixtures, base `skill_manage` create/edit/patch/delete,
`skills_list`, `skill_view`, and the background-review memory+skills-only
toolset policy. Missing parity rows now track Hermes `skill_manage`
support-file and curator-intent actions, the background review fork lifecycle,
curator state/report engine, curator auxiliary model routing, and `gormes
curator` command surface.

Execution should be TDD-first and local-signal-first:

- Start with Hermes `skill_manage` support-file and curator-intent actions:
  support-file write/patch/remove, `absorbed_into` deletes, pinned-skill
  refusal, optional guard rollback, and background-review provenance.
- Then port the Hermes background-review fork lifecycle: active runtime
  inheritance, memory+skills-only toolsets, attributed summaries, and cleanup.
- Add the Hermes `auxiliary.curator` slot before native curator runs so model
  routing, fallback, timeout, and credential behavior are visible and testable.
- Port the Hermes curator state/report engine before enabling `gormes curator`.
- Extend the Phase 2.G SKILL.md store with versioned metadata, provenance,
  review state, and atomic writes before generated skills persist.
- Add deterministic complexity signals from transcript length, tool-call count,
  retries, edits, and operator feedback.
- Use fake-model extraction fixtures to prove secret stripping and one-off task
  rejection before live LLM generation.
- Keep disabled or unreviewed skills out of prompt injection until retrieval, feedback, and operator review surfaces are all test-covered.
- Treat Gormes-owned Code Cathedral II as optional retrieval evidence: parent-scope and call-edge context may improve skill matching later, but the base learning loop must not require a TypeScript indexer, tree-sitter WASM, or repository-wide backfill.

## Donor pointers

When implementing a Phase 6 slice, route through the
`gormes-references` skill (`docs/development-skills/gormes-references/SKILL.md`)
before inventing a new shape. Useful donors:

| Learning-loop problem | Donor file |
|---|---|
| Audit/append-only activity log for skill events (provenance, redaction) | `engram/internal/mcp/activity.go` |
| Bounded token-budget for transcript-size complexity signals | `axe/internal/budget/budget.go` |
| Artifact tracker for stored extraction evidence (sanitized paths) | `axe/internal/artifact/tracker.go` |
| Truncation policy for large transcripts before extractor hand-off | `nanobot/pkg/agents/truncate.go` |
| Token-count estimation for reasoning/extraction batching | `nanobot/pkg/agents/tokencount.go` |

Hermes is now the source of truth for background review and curator behavior.
Use `provenance.origin_type: upstream` for those rows. Gormes-native detector,
scoring, and promotion rows should keep `provenance.origin_type: gormes` unless
a later Hermes source introduces a stricter public contract.

## Operator Proof

This is the deterministic local proof for the learning loop. It does not claim
that every repeated task improves automatically, and it does not require a live
model, hosted memory service, or network call. The proof is five linked
surfaces that an operator can inspect after a real or fixture-backed task.

| Surface | Command | What it proves |
|---|---|---|
| Task evidence | `gormes session list` and the audit log for the turn | The task had a traceable session, tool calls, edits, failures, and operator feedback instead of an ungrounded story. |
| Skill creation or improvement | `gormes skills list` | Reusable procedure knowledge lives in a SKILL.md surface, separate from raw chat history. New or changed skills stay reviewable before prompt injection. |
| Memory recall | `gormes memory status` | Durable facts and extractor state are visible as memory state, not hidden inside the assistant's latest answer. |
| Curator maintenance | `gormes curator status` | Agent-created skills have lifecycle state, review reports, archive/restore paths, and operator review boundaries. |
| Repeated-task proof | Run the same fixture after the skill or memory update | The second run should use the durable skill or recalled memory instead of rediscovering the same procedure from scratch. |

Read the loop as a trust boundary:

1. A task produces evidence: transcript, tool audit, edit diff, attachment
   metadata, or explicit operator correction.
2. The background review may propose memory or skill changes, but durable
   changes must remain attributable to the source session.
3. Skill creation or improvement records reusable "how" knowledge; memory
   records durable "what" facts.
4. Curator maintenance keeps agent-created skills reviewable by archiving,
   restoring, pinning, or consolidating them under operator review.
5. Repeated-task proof is only accepted when a deterministic local fixture or
   transcript shows the next run using the durable artifact.

The failure mode to watch for is a fake learning loop: an assistant says it
will remember something, but `gormes memory status` shows no durable state,
`gormes skills list` has no reusable procedure, and `gormes curator status`
has no review trail. That is a product bug, not just a wording bug, because the
operator cannot distinguish durable learning from session-local narration.
