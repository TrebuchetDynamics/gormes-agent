---
name: gormes-git
description: Use when the user asks Codex to commit current Gormes changes, push development, make the branch green, finish or publish a dirty worktree, open/update/merge the development-to-main PR, or recover a rejected development push without creating branches or worktrees.
---

# Gormes Git

## Overview

Use this skill for the exact "make development green, commit everything, push it, and handle the development-to-main PR" lane. It is not a feature-building workflow and it must not create branches, worktrees, or unbounded command sessions. It also owns generated-surface reconciliation before commit so progress/docs/site mirrors do not drift.

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

## Dirty Worktree Policy

Treat the worktree as shared state.

- If the user says "commit all changes", include tracked and untracked dirty files after the safety scan. Do not silently omit unrelated files.
- If the user asks for a narrower commit, do not stage unrelated dirty files. Leave them in place and report them.
- Never revert, checkout, delete, or rewrite user/parallel-agent changes to simplify the diff.
- Before staging untracked files, scan names and quick content for secrets, credentials, home-directory dumps, vendored dependency trees, and accidental large binaries. Stop and report unsafe artifacts.
- If generated files changed, identify the source file that caused them. Generated mirror changes with no matching source change require inspection before commit.

## Workflow

### 1. Inspect The Worktree

Run:

```sh
git status --short --untracked-files=all
git diff --stat
git worktree list
```

If the user says "commit all changes", include tracked and untracked dirty files. Still scan for obvious generated secrets, credentials, or accidental large binary artifacts before staging; report and stop if committing them would be unsafe.

### 2. Reconcile Generated Surfaces

Run this before validation whenever the relevant source files changed:

```sh
go run ./cmd/progress write
node www.gormes.ai/scripts/sync-assets.mjs
```

Use `go run ./cmd/progress write` when `docs/content/building-gormes/architecture_plan/progress.json` changed. It refreshes the progress-driven docs and site progress data.

Use `node www.gormes.ai/scripts/sync-assets.mjs` when installer scripts, `benchmarks.json`, progress data, or site-served assets changed. It refreshes `www.gormes.ai/public/install.*` and generated site mirrors.

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
timeout 20m sh -c 'cd www.gormes.ai/legacy/go-renderer && go test ./... -count=1'
timeout 20m sh -c 'cd www.gormes.ai && npm run build'
timeout 30m sh -c 'cd www.gormes.ai && npm run test:e2e'
timeout 30m sh -c 'cd docs/www-tests && npm run test:e2e'
```

Use the public-surface gates when `README.md`, `docs/`, `www.gormes.ai/`, `benchmarks.json`, or `docs/content/building-gormes/architecture_plan/progress.json` changed. If a gate fails, inspect the failing test or command output, fix the real issue in the current worktree, then rerun the failing gate and any affected higher-level gate.

### 4. Stage And Recheck

After the worktree is green:

```sh
git add -A
git diff --cached --stat
git diff --cached --check
```

Confirm the staged diff matches the user's request. Do not drop unrelated dirty files when the user explicitly asked to commit all changes.

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
