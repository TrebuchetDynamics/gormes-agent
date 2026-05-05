---
name: gormes-git
description: Use when the user invokes gormes-git to commit every current Gormes worktree change in coherent split commits, make development green, and push only the existing development branch.
---

# Gormes Git

## Mission

`gormes-git` has one job: commit all dirty Gormes changes on the existing
`development` branch, split the commits by coherent scope, validate the branch,
and push `origin development`.

Do not use this skill for feature implementation, PR opening, PR merging,
release tagging, branch creation, worktrees, or narrow/partial commits. If the
user invokes this skill in a dirty worktree, treat that as permission to commit
every tracked and untracked change after the safety scan.

## Branch Rule

Work only on the existing `development` branch.

1. Run `git branch --show-current`.
2. If it is not `development`, stop before editing or staging. Switch only when it is safe and the user has not asked to preserve another branch state.
3. Do not create feature branches, short-lived branches, or git worktrees.
4. Do not use destructive commands such as `git reset --hard` or `git checkout -- <path>` unless the user explicitly asks for that exact operation.
5. Never edit `main` directly.

## Session Safety

- Run expensive commands one at a time. Do not parallelize full repo tests, e2e tests, pushes, merges, or PR checks.
- Use non-interactive commands only. Never use watch mode, browser-opening flags, or commands that wait indefinitely.
- Put an explicit timeout around any gate that can hang. If GNU `timeout` is unavailable, use the command runner's own timeout mechanism.
- Keep output bounded. Start with summary commands, and if a full gate fails with huge output, rerun the smallest failing package, test, or spec for readable evidence.
- If a command times out, report it as timed out. Do not claim the gate passed, do not keep retrying the same full command, and do not continue to commit or merge until the timeout is understood.

## Dirty Worktree Policy

Treat the worktree as shared state.

- Capture the dirty set before staging. Treat later changes as part of the
  commit-all request unless they are unsafe.
- Include tracked and untracked dirty files after the safety scan. Do not
  silently omit unrelated files.
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

If untracked files exist, scan them before staging:

```sh
git ls-files --others --exclude-standard
find <untracked-paths> -maxdepth 2 -type f -size +5M -print
```

Stop if a file looks like credentials, a home-directory dump, vendored
dependencies, build artifacts, or accidental large binary output.

### 2. Reconcile Generated Surfaces

Run this before validation whenever the relevant source files changed:

```sh
go run ./cmd/progress write
node webpages/landing/scripts/sync-assets.mjs
```

Use `go run ./cmd/progress write` when `docs/content/building-gormes/architecture_plan/progress.json` changed. It refreshes the progress-driven docs and site progress data.

Use `node webpages/landing/scripts/sync-assets.mjs` when installer scripts, `benchmarks.json`, progress data, or site-served assets changed. It refreshes `webpages/landing/public/install.*` and generated site mirrors.

After either command, rerun `git status --short --untracked-files=all` and review the newly changed files. Do not hand-edit generated mirrors unless the generator is broken and that blocker is reported.

### 3. Make It Green

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

Run skill validation when a skill changed:

```sh
python3 /home/xel/.codex/skills/.system/skill-creator/scripts/quick_validate.py webpages/docs/development-skills/<skill-name>
find -L .agents/skills .claude/skills .codex/skills -maxdepth 2 -name SKILL.md -print | sort
```

### 4. Split Commits

Commit all changes in coherent slices. Prefer these groups when present:

| Scope | Typical paths | Subject shape |
|---|---|---|
| Runtime/CLI behavior | `cmd/`, `internal/`, runtime tests | `fix:` or `feat:` |
| Docs, parity, skills | `webpages/docs/`, development skills, progress rows, generated progress docs | `docs:` |
| Landing/public site | `assets/`, `webpages/landing/`, landing tests | `web:` |
| Repository metadata | root `README.md`, benchmarks, install mirrors, generated site data not tied to another slice | `chore:` |

Keep generated files in the same commit as the source change that caused them
when the relationship is clear. If a generated file spans multiple scopes, put
it with the progress/docs commit unless that would leave a later commit
unbuildable.

For each split commit:

```sh
git add -- <paths-for-this-slice>
git diff --cached --stat
git diff --cached --check
git diff --cached --name-status
git commit -m "<type>: <summary>"
```

If a commit hook changes files, review them, rerun the relevant validation,
stage the hook output, and retry the commit.

After the final split commit:

```sh
git status --short --branch --untracked-files=all
```

The worktree must be clean before pushing.

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

### 6. Final Evidence

Report:

- branch name;
- each commit hash and subject;
- push destination and remote result;
- validation commands that passed;
- remaining dirty files, if any.

Do not claim the branch is green without command output from the relevant gates.
