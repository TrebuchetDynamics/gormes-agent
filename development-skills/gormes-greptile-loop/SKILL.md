---
name: gormes-greptile-loop
description: Use when a Gormes pull request has Greptile review feedback or a sub-5/5 Greptile confidence score and the user wants an autonomous fix-review loop until 5/5 or a documented blocker.
---

# Gormes Greptile Loop

Adapted from `github.com/greptileai/skills` `greploop` for the Gormes repository rules: stay on `development`, keep changes bounded, validate locally before every push, and never hide blockers.

## Preconditions

- Current repo is the Gormes checkout reported by `git rev-parse --show-toplevel`.
- Branch is the existing `development` branch. Do not create branches or worktrees.
- `gh` is installed and authenticated.
- A GitHub PR exists for `development` or the user supplied a PR number.
- Greptile is installed/enabled for the repository.

If any precondition fails, stop and report the exact blocker. Do not fake a 5/5 result.

## Loop Contract

Goal: Greptile confidence `5/5` and zero unresolved Greptile comments.

Hard caps:

- Max 5 iterations per invocation.
- Stop after the same command fails twice with the same first stderr line.
- Stop if a fix requires scope expansion outside the reviewed change.

## Workflow

1. **Identify PR**
   - Prefer explicit PR number from the user.
   - Otherwise run `gh pr view --json number,headRefName,headRefOid`.
   - Confirm the PR head is `development` or explain the mismatch before continuing.

2. **Trigger or wait for Greptile**
   - Push current committed changes only when the working tree is clean or the current slice has just been committed.
   - If Greptile is not pending/in-progress, comment `@greptile review`.
   - Poll until the Greptile check/review completes.

3. **Fetch review evidence**
   - Read the most recent Greptile score from PR body and review comments.
   - Fetch unresolved review threads and inline comments.
   - Keep exact evidence: author, file, line, comment body, score, commit SHA.

4. **Exit if perfect**
   - Stop only when score is `5/5` and unresolved Greptile comments are `0`.

5. **Fix actionable feedback**
   - For behavior changes, route through `gormes-tdd-slice`: failing test first when practical, minimal fix, focused test.
   - For docs/skill/process comments, make the smallest documentation edit.
   - For false positives, write a short evidence-backed reply and resolve only if the claim is demonstrably false.

6. **Validate locally**
   - Required before push: `go test ./... -count=1`, `go run ./cmd/progress validate`, `git diff --check`.
   - If the change is docs-only, at minimum run `go test ./internal/skills -count=1`, progress validate, and diff check.

7. **Commit and push**
   - Use coherent commits per concern.
   - Push `origin development`.
   - Resolve addressed Greptile threads.
   - Repeat from step 2.

## Report Format

```text
Greptile loop result
  PR: <number/url>
  Iterations: <n>/5
  Final confidence: <x>/5
  Greptile comments resolved: <n>
  Remaining comments: <n>
  Local validation: <commands>
  Commits pushed: <hashes>
  Blocker: <none or exact blocker>
```

## Safety Notes

- Do not claim Greptile returned 5/5 without fetched PR evidence.
- Do not resolve comments that were not addressed or disproven.
- Do not broaden feature scope to satisfy review style preferences; ask or route to `gormes-progress-slicer`.
