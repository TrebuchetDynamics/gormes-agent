---
name: gormes-release
description: Use when the user asks to release Gormes, bump the operator-facing version, create a GitHub release tag, publish artifacts, recover a release, or run the full development-to-main release lane.
---

# Gormes Release

## Role

Use this skill only for the public release lane: version prep, commit-all
development cleanup, local and remote validation, development-to-main PR merge,
post-merge CI/CD validation on `main`, README/public-status freshness,
annotated tag creation, GitHub release artifact verification, and post-release
development sync.

If the user asks to improve `gormes-release` itself, treat that as skill
maintenance first: use `gormes-skill-manager` plus system `skill-creator`, edit
only the skill/routing docs needed, validate the skill shape, and do not bump
versions, tag, merge, publish, commit, or push unless the user explicitly asks
for that release operation.

## Branch Rule

Work only on the existing `development` branch. Do not create feature branches,
release branches, or worktrees. Never edit `main` directly.

The intended release path is always `development` -> GitHub PR -> `main` ->
tag. This is not an optional strategy or a fallback: every release, including a
"small" or "already green" release, must flow through a PR from `development`
to `main`. Do not fast-forward, cherry-pick, push directly to `main`, tag from
`development`, or tag before the PR has merged.

`gormes-release` calls `gormes-git` as the subroutine for dirty-worktree
safety scanning, generated-surface reconciliation, coherent split commits,
local green gates, and pushing `origin development`. The release skill owns the
GitHub PR, merge-to-main, tag, workflow, release artifact, and post-release
development sync steps.

## Intent And Stop Conditions

Do not treat naming this skill as permission to publish a release. Classify the
request before changing files:

| User intent | Action |
|---|---|
| Improve `gormes-release` or another skill | Skill-maintenance lane; validate docs only; no release actions. |
| Prepare release only | Version/changelog/docs prep may happen; no PR merge, tag, or publish unless later requested. |
| Release/publish Gormes | Run the full lane: include all safe dirty work, make `development` green, commit, push, open/update the `development` -> `main` PR, wait for green PR CI, merge that PR, wait for all post-merge `main` CI/CD workflows to finish green, tag the resulting `origin/main` commit, publish, then sync and validate `development`. |
| Recover failed release | First classify failure point: before tag, tag workflow before GitHub release, GitHub release exists, or artifact verification failed. |

Stop and report instead of continuing when:

- current branch is not `development`;
- the requested version/tag is ambiguous;
- dirty work contains credentials, accidental artifacts, generated drift with no
  source, or changes too ambiguous to include safely. For an explicit full
  release/publish request, treat current dirty work as agreed for inclusion
  after the safety scan;
- `README.md`, `CHANGELOG.md`, `webpages/landing/src/data/release.json`, or
  benchmark-derived public status claims still mention the previous release,
  previous date alias, stale release URL, or stale binary size after version
  prep;
- `go test ./...`, `go run ./cmd/progress validate`, `git diff --check`, or
  required remote checks fail;
- any post-merge `main` workflow for the release merge commit fails or remains
  inconclusive after bounded polling, including `CI`, `OCI Image`,
  `www mirror sync check`, `Deploy gormes.ai`, or `Deploy docs.gormes.ai`;
- tag `v<version>` or GitHub release `v<version>` already exists;
- the operator asks to bypass the `development` -> `main` PR path, tag from
  `development`, push directly to `main`, or merge without green PR checks;
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
   - whether this is a full release/publish request, which means all safe dirty
     work must be included through `gormes-git`;
   - for prep-only requests, which dirty files are intentionally included or
     left dirty;
   - whether this is prep-only, full publish, or recovery.
3. Fold the latest `origin/main` into `development` if needed, resolving
   conflicts without discarding user work. If conflicts are ambiguous, stop.
4. Apply release prep on `development`:
   - update `cmd/gormes/version.go`;
   - add a dated section to `CHANGELOG.md`;
   - refresh `README.md` public status so latest release, date alias, release
     URL, CI/CD wording, artifact matrix, and benchmark-derived binary size
     match the release being prepared;
   - regenerate public release metadata with `node webpages/landing/scripts/sync-assets.mjs`
     when `cmd/gormes/version.go`, installer mirrors, benchmarks, or landing
     data changed;
   - regenerate progress docs if progress files changed;
   - keep `.github/workflows/release.yml` validate prerequisites aligned with
     `.github/workflows/ci.yml`; `go test ./...` includes docs tests that need
     `docs` npm dependencies installed;
   - include any current dirty runtime/docs/skill changes after scanning for
     credentials or accidental artifacts.
5. Run local gates one at a time, including a public-status freshness check:

```sh
python3 - <<'PY'
from pathlib import Path
version = '<version>'
tag = f'v{version}'
date_alias = '<date-alias>'
readme = Path('README.md').read_text()
release_json = Path('webpages/landing/src/data/release.json').read_text()
assert tag in readme and date_alias in readme, 'README release status is stale'
assert tag in release_json and date_alias in release_json, 'landing release metadata is stale'
assert '<previous-tag>' not in readme, 'README still names previous release'
PY
timeout 20m go test ./... -count=1
timeout 5m go run ./cmd/progress validate
git diff --check
```

Run public-surface gates when `webpages/docs/`, `www.gormes.ai/`, `README.md`, or
progress mirror files changed:

```sh
timeout 20m sh -c 'cd webpages/landing/legacy/go-renderer && go test ./... -count=1'
timeout 30m sh -c 'cd webpages/landing && npm run test:e2e'
timeout 30m sh -c 'cd webpages/docs/www-tests && npm run test:e2e'
```

6. For a full release/publish lane, run `gormes-git` once the release diff is
   locally green. That subroutine must:
   - include all tracked and untracked dirty work after the safety scan;
   - split commits by coherent scope;
   - rerun the local green gate;
   - push only `origin development`.

   For prep-only release work, commit only the intended prep scope and leave
   out-of-scope dirty files unstaged.

7. Open or update the `development` to `main` PR, then wait only with bounded
   polling. This PR is the intended always-path for release integration. Do not
   tag before this PR is merged, even if `development` is already green and
   even if the release diff is version-only.

   ```sh
   gh pr list --head development --base main --state open --json number,url,title
   gh pr create --base main --head development --title "Release v<version>" --body "<body>"
   gh pr checks <number> --watch --interval 30
   gh pr view <number> --json state,isDraft,mergeStateStatus,reviewDecision,statusCheckRollup
   ```

   If an open `development` -> `main` PR already exists, update/reuse it rather
   than creating another branch or PR. If checks fail or the PR is not
   mergeable, fix through `development`, rerun local gates, push `development`,
   and wait for CI again.

8. Merge the PR only after required remote
    checks pass and the PR is mergeable.

    **Merge strategy detection:** GitHub repos may restrict allowed merge
    strategies. Detect which strategy works before merging:

    ```sh
    # Check repo settings for allowed merge strategies
    gh api repos/:owner/:repo --jq '.allow_merge_commit, .allow_squash_merge, .allow_rebase_merge'
    ```

    Then merge with the appropriate strategy:

    ```sh
    # Strategy 1: Merge commit (default when allowed)
    gh pr merge <number> --merge --subject "Release v<VERSION>" --body "<body>"

    # Strategy 2: Squash (use when merge commits are not allowed)
    gh pr merge <number> --squash --subject "Release v<VERSION>" 2>&1

    # Strategy 3: Rebase (use when only rebase is allowed)
    gh pr merge <number> --rebase --subject "Release v<VERSION>" 2>&1
    ```

    If `--merge` fails with `GraphQL: Merge commits are not allowed`, retry
    with `--squash`. If `--squash` also fails, try `--rebase`.

    After the merge, verify the result:

    ```sh
    git fetch origin main
    git log --oneline origin/main -3
    gh pr view <number> --json state,mergedBy,mergedAt
    ```

9. After merge, fetch `origin/main`, confirm `cmd/gormes/version.go` on main
   matches the release version, and confirm the merge came from the
   `development` -> `main` PR. Then wait for every workflow triggered by that
   exact `origin/main` commit to complete. This is the CI/CD release gate, not
   just PR CI. At minimum inspect all runs for the merge SHA, including:
   `CI`, `OCI Image`, `www mirror sync check`, `Deploy gormes.ai`, and
   `Deploy docs.gormes.ai` when they are triggered.

   ```sh
   main_sha=$(git rev-parse origin/main)
   gh run list --branch main --limit 20 --json workflowName,headSha,status,conclusion,url
   gh run view <failed-run-id> --log
   ```

   Do not tag while any post-merge `main` workflow is failing, in progress, or
   missing from the expected surface for changed files. If any main CI/CD run
   fails, fix through a new `development` -> `main` PR, wait for that PR and
   post-merge main workflows to pass, then tag only the corrected `origin/main`
   commit.

10. After the post-merge `main` CI/CD gate is green, create and push the
    annotated tag from the merged `origin/main` commit only:

```sh
git tag -a "v<version>" "origin/main" -m "Release <version>"
git push origin "v<version>"
```

11. Watch the tag-triggered `Release Binaries` workflow with bounded polling.
   Confirm the GitHub release exists and contains archives plus checksums.
   If the tag workflow fails before a GitHub release is created, fix through
   the `development` PR path. Delete a failed local/remote tag only when all
   are true: the tag was created in this run, no GitHub release exists for it,
   the user has authorized tag recovery, and the fixed `main` commit is green.
   If a GitHub release already exists, stop and report the recovery options
   instead of mutating public artifacts.
12. Sync `development` with `origin/main`, push `development`, and wait for the
   resulting `development` branch workflows to finish green. Leave the local
   checkout on `development`.

## Recovery Classification

When recovering a release, first identify the failure point:

| Failure point | Recovery posture |
|---|---|
| Before version commit | Fix on `development`, rerun local gates, then continue. |
| After version commit but before PR merge | Fix on `development`, rerun `gormes-git` gates, update PR. |
| After merge but before tag push | Wait for post-merge `main` CI/CD. If any main workflow fails, fix through a new `development` PR; tag only the corrected `origin/main` after all main CI/CD is green. |
| Tag pushed, workflow failed, no GitHub release exists | With explicit recovery authorization, delete the just-created tag, fix through PR, recreate same tag on corrected `main`. |
| GitHub release exists but artifacts/checksums are wrong | Do not delete or overwrite public assets automatically; report exact missing/bad artifacts and ask for recovery direction. |

## Final Evidence

Report the release version, development commit hashes, PR URL, main merge
commit, tag, release URL, local gates, PR checks, post-merge `main` CI/CD
workflow results, tag-triggered release workflow result, artifact/checksum
list, post-release `development` sync workflow results, and any dirty files
left out of the release.
