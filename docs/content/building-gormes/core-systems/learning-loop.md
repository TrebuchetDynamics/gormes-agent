---
title: "Learning Loop"
weight: 20
---

# The Learning Loop (The Soul)

Detects when a task was complex enough to learn from, distills the solution into a reusable skill, stores it, and improves the skill over successive runs.

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

Upstream Hermes has a `skills/` directory with hand-authored SKILL.md files. It does not have an algorithm that decides what's worth writing down. That's what Phase 6 delivers.

## Current status

⏳ Planned — see [Phase 6](../../architecture_plan/phase-6-learning-loop/) for the sub-phase breakdown.

Execution should be TDD-first and local-signal-first:

- Start with deterministic complexity signals from transcript length, tool-call count, retries, edits, and operator feedback.
- Extend the Phase 2.G SKILL.md store with versioned metadata, provenance, review state, and atomic writes before generated skills persist.
- Use fake-model extraction fixtures to prove secret stripping and one-off task rejection before live LLM generation.
- Keep disabled or unreviewed skills out of prompt injection until retrieval, feedback, and operator review surfaces are all test-covered.
- Treat GBrain Code Cathedral II as optional retrieval evidence: parent-scope and call-edge context may improve skill matching later, but the base learning loop must not require a TypeScript indexer, tree-sitter WASM, or repository-wide backfill.

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

Skill extraction itself has no clean donor — Hermes' Python `skills/` is
hand-authored, and no Go reference ships an automated extractor. Record this
explicitly on Phase 6 rows as `provenance.origin_type: gormes` and write the
contract from scratch with TDD.
