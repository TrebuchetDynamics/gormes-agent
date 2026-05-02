---
name: gormes-git
description: Commit, validate, push, and PR-merge the existing Gormes development branch safely. Use when the user asks Codex to commit all current changes, push development, make the branch green, finish/publish the dirty worktree, open/update/merge the development-to-main PR, or recover a rejected development push without creating branches or worktrees.
---

# Gormes Git

## Overview

Use this skill for the exact "make development green, commit everything, push it, and handle the development-to-main PR" lane. It is not a feature-building workflow and it must not create branches or worktrees.

## Branch Rule

Work only on the existing `development` branch.

1. Run `git branch --show-current`.
2. If it is not `development`, stop before editing or staging. Switch only when it is safe and the user has not asked to preserve another branch state.
3. Do not create feature branches, short-lived branches, or git worktrees.
4. Do not use destructive commands such as `git reset --hard` or `git checkout -- <path>` unless the user explicitly asks for that exact operation.
5. Never edit `main` directly. Changes reach `main` only through a GitHub pull request from `development`.

## Workflow

### 1. Inspect The Worktree

Run:

```sh
git status --short --untracked-files=all
git diff --stat
git worktree list
```

If the user says "commit all changes", include tracked and untracked dirty files. Still scan for obvious generated secrets, credentials, or accidental large binary artifacts before staging; report and stop if committing them would be unsafe.

### 2. Make It Green Before Commit

Run the repository green gate:

```sh
go test ./... -count=1
go run ./cmd/progress validate
git diff --check
```

Run focused public-surface gates when relevant:

```sh
(cd www.gormes.ai && go test ./... -count=1)
(cd www.gormes.ai && npm run test:e2e)
(cd docs/www-tests && npm run test:e2e)
```

Use the public-surface gates when `README.md`, `docs/`, `www.gormes.ai/`, `benchmarks.json`, or `docs/content/building-gormes/architecture_plan/progress.json` changed. If a gate fails, inspect the failing test or command output, fix the real issue in the current worktree, then rerun the failing gate and any affected higher-level gate.

### 3. Stage And Recheck

After the worktree is green:

```sh
git add -A
git diff --cached --stat
git diff --cached --check
```

Confirm the staged diff matches the user's request. Do not drop unrelated dirty files when the user explicitly asked to commit all changes.

### 4. Commit

Use a concise commit subject that reflects the staged diff. Prefer a broad `chore:` subject when the commit spans docs, runtime, tests, and skills. Example:

```sh
git commit -m "chore: update gormes docs and workflow"
```

If hooks fail, fix the failure, restage, rerun the relevant green gate, and commit again.

### 5. Push Development

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

### 6. PR Merge Rules

Use this section only when the user asks to open, update, or merge the PR to `main`.

1. Fetch current refs:

```sh
git fetch origin main development
```

2. Confirm local `development` is clean, pushed, and matches `origin/development`.
3. Ensure `development` includes current `origin/main`. If not, merge `origin/main` into `development`, rerun the green gate, commit the merge if needed, and push `origin development`.
4. Open or reuse the PR with base `main` and head `development`. Do not open PRs from any other branch.
5. Before merging, require clean local gates and clean remote checks for the latest `origin/development` SHA:

```sh
gh pr checks --watch
```

6. Merge only after required checks pass and the PR is mergeable. Prefer the repository's configured merge method. Do not force-merge, bypass branch protection, delete `development`, or push directly to `main`.
7. After merge, fetch refs and confirm `origin/main` contains the merged `development` head. Leave the local worktree on `development`.

### 7. Final Evidence

Report:

- branch name;
- commit hash and subject;
- push destination;
- validation commands that passed;
- PR URL and merge result, when PR work was requested;
- remaining dirty files, if any.

Do not claim the branch is green without command output from the relevant gates.
