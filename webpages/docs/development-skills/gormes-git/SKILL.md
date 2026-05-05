---
name: gormes-git
description: Use when the user asks Codex to commit current Gormes changes, push development, make the branch green, finish or publish a dirty worktree, open/update/merge the development-to-main PR, or recover a rejected development push without creating branches or worktrees.
---

# Gormes Git

## Overview

Use this skill for the exact "make development green, commit everything, push it, and handle the development-to-main PR" lane. It is not a feature-building workflow and it must not create branches, worktrees, or unbounded command sessions. It also owns generated-surface reconciliation before commit so progress/docs/site mirrors do not drift.

If the user asks to improve `gormes-git` itself, treat that as skill
maintenance first: use `gormes-skill-manager` plus system `skill-creator`, edit
only the skill/routing docs needed, validate the skill shape, and do not commit
or push unless the user explicitly asks for that git operation.

## Branch Rule

Work only on the existing `development` branch.

1. Run `git branch --show-current`.
2. If it is not `development`, stop before editing or staging. Switch only when it is safe and the user has not asked to preserve another branch state.
3. Do not create feature branches, short-lived branches, or git worktrees.
4. Do not use destructive commands such as `git reset --hard` or `git checkout -- <path>` unless the user explicitly asks for that exact operation.
5. Never edit `main` directly. Changes reach `main` only through a GitHub pull request from `development`.

## Session Safety

Keep the agent session responsive while still preserving validation integrity.

- Run expensive commands one at a time. Do not parallelize full repo tests, e2e tests, pushes, merges, or PR checks.
- Use non-interactive commands only. Never use watch mode, browser-opening flags, or commands that wait indefinitely.
- Put an explicit timeout around any gate that can hang. If GNU `timeout` is unavailable, use the command runner's own timeout mechanism.
- Keep output bounded. Start with summary commands, and if a full gate fails with huge output, rerun the smallest failing package, test, or spec for readable evidence.
- If a command times out, report it as timed out. Do not claim the gate passed, do not keep retrying the same full command, and do not continue to commit or merge until the timeout is understood.
- If remote checks are pending, report the pending state or poll in a small bounded loop. Do not leave a watch command running.

## Intent Classification

Do not treat naming this skill as permission to commit or push. Classify the
user's exact intent before staging:

| User intent | Action |
|---|---|
| Improve `gormes-git` or another skill | Skill-maintenance lane; validate skill docs only; no commit/push unless explicitly requested. |
| Commit all/current dirty changes | Include tracked and untracked files after safety scan and generated-surface reconciliation. |
| Commit this/narrow change | Stage only owned paths for the requested slice; leave unrelated dirty files untouched and report them. |
| Push `development` | Require a clean committed branch, then push only `origin development`. |
| Open/update/merge PR | Use the PR merge rules only after local and remote gates are clean. |

If wording like "finish this" or "ship it" is ambiguous in a dirty worktree,
state the inferred scope before staging. Ask only when the wrong scope could
commit unrelated user work.

## Dirty Worktree Policy

Treat the worktree as shared state.

- Capture the initial dirty set before editing or staging. Treat later
  unrelated changes as user/parallel-agent work unless you created them.
- If the user says "commit all changes", include tracked and untracked dirty files after the safety scan. Do not silently omit unrelated files.
- If the user asks for a narrower commit, do not stage unrelated dirty files. Leave them in place and report them.
- Never revert, checkout, delete, or rewrite user/parallel-agent changes to simplify the diff.
- Before staging untracked files, scan names and quick content for secrets, credentials, home-directory dumps, vendored dependency trees, and accidental large binaries. Stop and report unsafe artifacts.
- If generated files changed, identify the source file that caused them. Generated mirror changes with no matching source change require inspection before commit.

## Workflow

### 1. Inspect The Worktree

Run:

```sh
git branch --show-current
git status --short --untracked-files=all
git diff --stat
git worktree list
```

If the user says "commit all changes", include tracked and untracked dirty files. Still scan for obvious generated secrets, credentials, or accidental large binary artifacts before staging; report and stop if committing them would be unsafe.

### 2. Reconcile Generated Surfaces

Run this before validation whenever the relevant source files changed:

```sh
go run ./cmd/progress write
node webpages/landing/scripts/sync-assets.mjs
```

Use `go run ./cmd/progress write` when `docs/content/building-gormes/architecture_plan/progress.json` changed. It refreshes the progress-driven docs and site progress data.

Use `node webpages/landing/scripts/sync-assets.mjs` when installer scripts, `benchmarks.json`, progress data, or site-served assets changed. It refreshes `webpages/landing/public/install.*` and generated site mirrors.

After either command, rerun `git status --short --untracked-files=all` and review the newly changed files. Do not hand-edit generated mirrors unless the generator is broken and that blocker is reported.

### 3. Make It Green Before Commit

Run the repository green gate one command at a time with timeouts:

```sh
timeout 20m go test ./... -count=1
timeout 5m go run ./cmd/progress validate
git diff --check
```

Run focused public-surface gates when relevant:

```sh
timeout 20m sh -c 'cd webpages/landing/legacy/go-renderer && go test ./... -count=1'
timeout 20m sh -c 'cd webpages/landing && npm run build'
timeout 30m sh -c 'cd webpages/landing && npm run test:e2e'
timeout 30m sh -c 'cd docs/www-tests && npm run test:e2e'
```

Use the public-surface gates when `README.md`, `docs/`, `webpages/landing/`, `benchmarks.json`, or `docs/content/building-gormes/architecture_plan/progress.json` changed. If a gate fails, inspect the failing test or command output, fix the real issue in the current worktree, then rerun the failing gate and any affected higher-level gate.

For skill-maintenance edits that are not being committed in this turn, run the
skill validation and the light repo contracts instead of the full commit gate:

```sh
python3 /home/xel/.codex/skills/.system/skill-creator/scripts/quick_validate.py docs/development-skills/<skill-name>
find -L .agents/skills .claude/skills .codex/skills -maxdepth 2 -name SKILL.md -print | sort
go run ./cmd/progress validate
git diff --check
```

### 4. Stage And Recheck

After the worktree is green:

```sh
# For commit-all intent only:
git add -A

# For narrow commit intent, stage only owned paths:
git add -- <path> [<path>...]

git diff --cached --stat
git diff --cached --check
git diff --cached --name-status
```

Confirm the staged diff matches the user's request. If the user asked for a
narrow commit, `git diff --cached --name-status` must not include unrelated
initial dirty files. If the user explicitly asked to commit all changes, do not
drop unrelated dirty files after the safety scan.

### 5. Commit

Use a concise commit subject that reflects the staged diff. Prefer a broad `chore:` subject when the commit spans docs, runtime, tests, and skills. Example:

```sh
git commit -m "chore: update gormes docs and workflow"
```

If hooks fail, fix the failure, restage, rerun the relevant green gate, and commit again.

### 6. Push Development

Push only `development`:

```sh
git push origin development
```

If the push is rejected because the remote moved, run:

```sh
git fetch origin development
git merge --no-edit origin/development
```

Resolve any conflicts without discarding local work, rerun the green gate, then push `origin development` again. If the merge becomes unsafe or ambiguous, stop and report the blocker instead of forcing.

### 7. PR Merge Rules

Use this section only when the user asks to open, update, or merge the PR to `main`.

1. Fetch current refs:

```sh
git fetch origin main development
```

2. Confirm local `development` is clean, pushed, and matches `origin/development`.
3. Ensure `development` includes current `origin/main`. If not, merge `origin/main` into `development`, rerun the green gate, commit the merge if needed, and push `origin development`.
4. Open or reuse the PR with base `main` and head `development`. Do not open PRs from any other branch.
5. Before merging, require clean local gates and clean remote checks for the latest `origin/development` SHA. Use a bounded, non-watch check:

```sh
gh pr checks --json name,bucket,state,workflow,link \
  --jq '.[] | [.bucket, .state, .name, .workflow] | @tsv'
```

If `gh pr checks` exits with code 8, checks are pending. That is not a crash and not a pass; poll only in a bounded loop or report the pending checks and stop.

6. Merge only after required checks pass and the PR is mergeable. Prefer the repository's configured merge method. Do not force-merge, bypass branch protection, delete `development`, or push directly to `main`.
7. After merge, fetch refs and confirm `origin/main` contains the merged `development` head. Leave the local worktree on `development`.

### 8. Final Evidence

Report:

- branch name;
- commit hash and subject;
- push destination;
- validation commands that passed;
- PR URL and merge result, when PR work was requested;
- remaining dirty files, if any.

Do not claim the branch is green without command output from the relevant gates.
