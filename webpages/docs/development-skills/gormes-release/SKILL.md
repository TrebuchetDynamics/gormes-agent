---
name: gormes-release
description: Prepare, validate, PR-merge, tag, and publish a Gormes release from the existing development branch. Use when the user asks to release Gormes, bump the operator-facing version, create a GitHub release tag, or run the full development-to-main release lane.
---

# Gormes Release

## Role

Use this skill only for the public release lane: version prep, local and remote
validation, development-to-main PR merge, annotated tag creation, GitHub release
artifact verification, and post-release development sync.

If the user asks to improve `gormes-release` itself, treat that as skill
maintenance first: use `gormes-skill-manager` plus system `skill-creator`, edit
only the skill/routing docs needed, validate the skill shape, and do not bump
versions, tag, merge, publish, commit, or push unless the user explicitly asks
for that release operation.

## Branch Rule

Work only on the existing `development` branch. Do not create feature branches,
release branches, or worktrees. Never edit `main` directly. All source changes
reach `main` through a PR from `development`.

`gormes-release` may call `gormes-git` as a subroutine for the dirty-worktree
green/commit/push/PR lane. Do not duplicate or bypass `gormes-git` safeguards
for staging, generated-surface reconciliation, push rejection recovery, or PR
checks.

## Intent And Stop Conditions

Do not treat naming this skill as permission to publish a release. Classify the
request before changing files:

| User intent | Action |
|---|---|
| Improve `gormes-release` or another skill | Skill-maintenance lane; validate docs only; no release actions. |
| Prepare release only | Version/changelog/docs prep may happen; no PR merge, tag, or publish unless later requested. |
| Release/publish Gormes | Run the full release lane after all preflight checks pass. |
| Recover failed release | First classify failure point: before tag, tag workflow before GitHub release, GitHub release exists, or artifact verification failed. |

Stop and report instead of continuing when:

- current branch is not `development`;
- the requested version/tag is ambiguous;
- `git status` contains dirty work the user has not agreed to include or leave
  out of the release commit;
- `go test ./...`, `go run ./cmd/progress validate`, `git diff --check`, or
  required remote checks fail;
- tag `v<version>` or GitHub release `v<version>` already exists;
- `origin/main` is red or the development-to-main PR is not mergeable;
- artifact publication created a GitHub release, but checksums or archives are
  missing. Do not delete public release assets without explicit recovery
  instructions.

## Version Rule

- If the user gives an explicit version or tag, use it exactly after removing a
  leading `v` for source files.
- Otherwise, read `cmd/gormes/version.go` and bump the patch component by
  `+0.0.01`, preserving a two-digit patch width when the existing or requested
  release lane uses it.
- The tag must be `v<version>`, and the tag version must exactly match
  `cmd/gormes/version.go`; the release workflow rejects mismatches.
- Before tagging, verify the tag and GitHub release do not already exist.
- After version prep but before commit, run a focused check that the source
  version, changelog heading, tag string, and release workflow expectations all
  agree.

## Preflight Commands

Run these before editing release files:

```sh
git branch --show-current
git status --short --untracked-files=all
git fetch origin main development --tags
git rev-parse --short HEAD
git rev-parse --short origin/development
git rev-parse --short origin/main
gh auth status
```

After resolving `<version>` and `v<version>`:

```sh
git ls-remote --tags origin "refs/tags/v<version>"
gh release view "v<version>" --json tagName,url,isDraft,isPrerelease
```

Empty `git ls-remote` output and a failing `gh release view` because the
release does not exist are expected before a new release. Any existing tag or
release is a stop condition unless the user explicitly asked for recovery.

## Workflow

1. Inspect state and classify intent:

```sh
git branch --show-current
git status --short --untracked-files=all
git fetch origin main development --tags
```

2. Resolve release scope:
   - version/tag;
   - whether current dirty work is included in this release or intentionally
     left dirty;
   - whether this is prep-only, full publish, or recovery.
3. Fold the latest `origin/main` into `development` if needed, resolving
   conflicts without discarding user work. If conflicts are ambiguous, stop.
4. Apply release prep on `development`:
   - update `cmd/gormes/version.go`;
   - add a dated section to `CHANGELOG.md`;
   - regenerate progress docs if progress files changed;
   - keep `.github/workflows/release.yml` validate prerequisites aligned with
     `.github/workflows/ci.yml`; `go test ./...` includes docs tests that need
     `docs` npm dependencies installed;
   - include any current dirty runtime/docs/skill changes after scanning for
     credentials or accidental artifacts.
5. Run local gates one at a time:

```sh
timeout 20m go test ./... -count=1
timeout 5m go run ./cmd/progress validate
git diff --check
```

Run public-surface gates when `docs/`, `www.gormes.ai/`, `README.md`, or
progress mirror files changed:

```sh
timeout 20m sh -c 'cd webpages/landing && go test ./... -count=1'
timeout 30m sh -c 'cd webpages/landing && npm run test:e2e'
timeout 30m sh -c 'cd docs/www-tests && npm run test:e2e'
```

6. Use `gormes-git` for commit, push, and PR setup once the release diff is
   green. Stage only the intended release scope unless the user explicitly
   asked to include all dirty work.
7. Wait only with bounded polling. Merge the PR only after required remote
   checks pass and the PR is mergeable.
8. After merge, fetch `origin/main`, confirm `cmd/gormes/version.go` on main
   matches the release version, then create and push the annotated tag:

```sh
git tag -a "v<version>" "origin/main" -m "Release <version>"
git push origin "v<version>"
```

9. Watch the tag-triggered `Release Binaries` workflow with bounded polling.
   Confirm the GitHub release exists and contains archives plus checksums.
   If the tag workflow fails before a GitHub release is created, fix through
   the `development` PR path. Delete a failed local/remote tag only when all
   are true: the tag was created in this run, no GitHub release exists for it,
   the user has authorized tag recovery, and the fixed `main` commit is green.
   If a GitHub release already exists, stop and report the recovery options
   instead of mutating public artifacts.
10. Sync `development` with `origin/main`, push `development`, and leave the
   local checkout on `development`.

## Recovery Classification

When recovering a release, first identify the failure point:

| Failure point | Recovery posture |
|---|---|
| Before version commit | Fix on `development`, rerun local gates, then continue. |
| After version commit but before PR merge | Fix on `development`, rerun `gormes-git` gates, update PR. |
| After merge but before tag push | Fix through a new `development` PR; tag only the corrected `origin/main`. |
| Tag pushed, workflow failed, no GitHub release exists | With explicit recovery authorization, delete the just-created tag, fix through PR, recreate same tag on corrected `main`. |
| GitHub release exists but artifacts/checksums are wrong | Do not delete or overwrite public assets automatically; report exact missing/bad artifacts and ask for recovery direction. |

## Final Evidence

Report the release version, PR URL, main merge commit, tag, release URL, local
and remote gates, artifact/checksum list, post-release `development` sync state,
and any dirty files left out of the release.
