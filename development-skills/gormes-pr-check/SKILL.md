---
name: gormes-pr-check
description: Use when checking a Gormes pull request for unresolved comments, failing checks, incomplete description, missing validation evidence, or merge readiness before Greptile/release review.
---

# Gormes PR Check

Adapted from `github.com/greptileai/skills` `check-pr` for Gormes' `development`-only workflow.

Use this before `gormes-greptile-loop`, before asking for human review, or when the user asks whether a PR is ready.

## Preconditions

- Stay in the current Gormes checkout (`git rev-parse --show-toplevel`).
- Stay on `development`; do not create/switch feature branches.
- Use GitHub CLI `gh` if PR metadata is needed.

## Workflow

1. **Identify target PR**
   - Use supplied PR number, or `gh pr view --json number,url,title,headRefName,headRefOid`.
   - Confirm head branch and current local branch.

2. **Collect evidence**
   - PR title/body and checklist state.
   - Status checks and pending/failing jobs.
   - Review decisions, unresolved threads, and bot comments.
   - Current local `git status --short --branch`.
   - Local validation evidence relevant to changed files.

3. **Classify issues**

| Category | Meaning | Next skill |
|---|---|---|
| Failing check | CI or required local gate failed | `gormes-review-loop` or `gormes-tdd-slice` |
| Review feedback | Human/Greptile comment needs action | `gormes-greptile-loop` or `gormes-review-loop` |
| Missing evidence | PR lacks test/validation notes | update PR body or final report |
| Scope drift | PR mixes unrelated concerns | ask user or split commits if still local |
| Informational | no action required | record only |

4. **Do not fix unless requested**
   - Default output is an audit.
   - If the user asks to fix, route to `gormes-review-loop` or `gormes-greptile-loop`.

5. **Optional PR body update**
   - Only update the PR description when evidence is fresh and the user intent includes preparing the PR.
   - Include exact validation commands and commit hashes.

## Output Format

```text
PR readiness check
  PR: <url>
  Branch: development
  Checks: <passing/pending/failing>
  Reviews: <approved/changes/unresolved>
  Description: <complete/incomplete>
  Local status: <clean/dirty>

Actionable items:
  1. <item> — <next skill/command>

Ready to merge: <yes/no>
```

## Validation

For docs/skills-only PR checks:

- `go test ./internal/skills -count=1`
- `go run ./cmd/progress validate`
- `git diff --check`

For runtime PR checks, require the full gate before readiness:

- `go test ./... -count=1`
- `go run ./cmd/progress validate`
- `git diff --check`
