---
name: gormes-review-loop
description: Use when Gormes work has PR feedback, CI failures, reviewer comments, static-analysis findings, or a need to iterate until a bounded validation target is green.
---

# Gormes Review Loop

## Overview

Review feedback is a loop, not a vibes pass. Fix one verified issue at a time, prove it, and stop only when the agreed gate is green or a blocker is documented.

## When to Use

Use for:
- PR review comments, GitHub checks, CI failures, `git diff --check`, or reviewer confidence issues;
- “keep going until green”, “address feedback”, “make this mergeable”, or “review loop” requests;
- self-review before `gormes-git` or `gormes-release`.

Do not use to expand scope. Large PRs should be split before looping.

## Workflow

1. Capture the target gate: review comments, Greptile score, GitHub check, CI annotation, test command, progress validation, or diff check.
2. Read the exact failing evidence; do not infer from summaries. Preserve reviewer text, PR/check URL, file path, line/symbol, command output, and commit SHA when available.
3. Pick one issue and classify it: bug, test drift, docs drift, style, blocker, or row-shaping needed.
4. If the feedback is a broad external-review finding that must become canonical backlog work, route to `gormes-planner` with the exact evidence. The planner must refine an existing progress row or add one builder-ready row; do not create side queues.
5. For code behavior that is already builder-ready, use `gormes-tdd-slice`: failing test first, minimal fix, rerun focused test.
6. Re-run the target gate.
7. Repeat until green, or document the blocker and pivot per workspace rules.

## Quick Reference

| Evidence | Next step |
|---|---|
| Test failure | Reproduce focused failure before editing |
| Review comment | Map to file and exact requested behavior |
| Greptile/Grep-style review finding that changes backlog shape | Route exact evidence to `gormes-planner` |
| Diff whitespace | Run `git diff --check` after edit |
| Progress docs drift | Run `go run ./cmd/progress validate` |
| Full merge gate | Run `go test ./... -count=1`, progress validate, diff check |

## Stop Conditions

- Target gate is green with command output captured.
- Same command failed twice with the same blocker.
- Fix requires scope expansion; ask Juan or route to planner.

## Common Mistakes

- Letting an automated loop rewrite broad areas without focused evidence.
- Turning external review findings into private TODOs instead of progress-row refinements.
- Marking review “done” after edits but before rerunning the gate.
- Treating a confidence score as proof; command output and reviewer comments are evidence.
- Continuing after a blocker without documenting it in required locations.
