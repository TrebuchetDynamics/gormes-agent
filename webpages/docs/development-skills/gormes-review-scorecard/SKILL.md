---
name: gormes-review-scorecard
description: Use when Greptile is unavailable or before requesting external review to assign a conservative 1-5 production-readiness score to a Gormes change from source evidence and validation results.
---

# Gormes Review Scorecard

This is a local, conservative version of the Greptile-style 5/5 quality gate. It does not replace Greptile or CI; it tells the agent whether the work deserves external review yet.

## Use When

- The user asks for a “5/5 review” but Greptile cannot be queried.
- You need a preflight before `gormes-greptile-loop` or `gormes-release`.
- A change has green tests but may still be weak on UX, scope, safety, or evidence.

## Scoring

Start at 5. Subtract for evidence-backed problems:

| Deduction | Problem |
|---|---|
| -2 | failing required test/check, known runtime bug, data-loss/security risk |
| -1 | missing focused test for changed behavior |
| -1 | unclear user-visible contract or parity evidence |
| -1 | mixed concerns that should be split |
| -1 | weak validation notes or unverified manual claim |
| -1 | docs/progress/skill routing drift |

Minimum score is 1. Never award 5/5 with dirty uncommitted production code unless the review scope is explicitly local/uncommitted.

## Required Evidence

1. `git status --short --branch`
2. Changed-file summary: `git diff --stat` or `git show --stat` for committed work.
3. Validation commands appropriate to the change:
   - docs/skills: `go test ./internal/skills -count=1`, `go run ./cmd/progress validate`, `git diff --check`
   - runtime: `go test ./... -count=1`, `go run ./cmd/progress validate`, `git diff --check`
4. Relevant user-visible evidence: test fixture, CLI output, screenshot description, transcript, PR comment, or source reference.

## Workflow

1. Define the reviewed scope: uncommitted diff, commit range, or PR.
2. Gather evidence; do not score from memory.
3. Apply deductions mechanically.
4. For each deduction, name the exact file/command/comment causing it.
5. If score is under 5, route fixes:
   - behavior/test issue → `gormes-tdd-slice`
   - PR/reviewer issue → `gormes-review-loop` or `gormes-greptile-loop`
   - broad missing work → `gormes-progress-slicer`
   - unclear architecture → `gormes-architecture-zoomout`

## Output Format

```text
Gormes review scorecard
  Scope: <diff/commit/PR>
  Score: <n>/5
  Evidence: <commands/files>

Deductions:
  - <points>: <reason + exact evidence>

Required to reach 5/5:
  1. <smallest next action + skill>
```

## Guardrails

- This is not a vibes score. No evidence means no 5/5.
- Do not use the scorecard to excuse skipping CI or Greptile.
- Do not keep looping forever; if the same class of issue repeats, improve the relevant repo-local skill.
