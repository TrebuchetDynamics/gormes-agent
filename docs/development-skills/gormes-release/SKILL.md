---
name: gormes-release
description: Prepare, validate, PR-merge, tag, and publish a Gormes release from the existing development branch. Use when the user asks to release Gormes, bump the operator-facing version, create a GitHub release tag, or run the full development-to-main release lane.
---

# Gormes Release

## Branch Rule

Work only on the existing `development` branch. Do not create feature branches,
release branches, or worktrees. Never edit `main` directly. All source changes
reach `main` through a PR from `development`.

## Version Rule

- If the user gives an explicit version or tag, use it exactly after removing a
  leading `v` for source files.
- Otherwise, read `cmd/gormes/version.go` and bump the patch component by
  `+0.0.01`, preserving a two-digit patch width when the existing or requested
  release lane uses it.
- The tag must be `v<version>`, and the tag version must exactly match
  `cmd/gormes/version.go`; the release workflow rejects mismatches.
- Before tagging, verify the tag and GitHub release do not already exist.

## Workflow

1. Inspect state:

```sh
git branch --show-current
git status --short --untracked-files=all
git fetch origin main development --tags
```

2. Fold the latest `origin/main` into `development` if needed, resolving
   conflicts without discarding user work.
3. Apply release prep on `development`:
   - update `cmd/gormes/version.go`;
   - add a dated section to `CHANGELOG.md`;
   - regenerate progress docs if progress files changed;
   - include any current dirty runtime/docs/skill changes after scanning for
     credentials or accidental artifacts.
4. Run local gates one at a time:

```sh
timeout 20m go test ./... -count=1
timeout 5m go run ./cmd/progress validate
git diff --check
```

Run public-surface gates when `docs/`, `www.gormes.ai/`, `README.md`, or
progress mirror files changed:

```sh
timeout 20m sh -c 'cd www.gormes.ai && go test ./... -count=1'
timeout 30m sh -c 'cd www.gormes.ai && npm run test:e2e'
timeout 30m sh -c 'cd docs/www-tests && npm run test:e2e'
```

5. Commit all intended changes, push `development`, and open or update the PR
   from `development` to `main`.
6. Wait only with bounded polling. Merge the PR only after required remote
   checks pass and the PR is mergeable.
7. After merge, fetch `origin/main`, confirm `cmd/gormes/version.go` on main
   matches the release version, then create and push the annotated tag:

```sh
git tag -a "v<version>" "origin/main" -m "Release <version>"
git push origin "v<version>"
```

8. Watch the tag-triggered `Release Binaries` workflow with bounded polling.
   Confirm the GitHub release exists and contains archives plus checksums.
9. Sync `development` with `origin/main`, push `development`, and leave the
   local checkout on `development`.

## Final Evidence

Report the release version, PR URL, main merge commit, tag, release URL, local
and remote gates, and whether `development` is clean and synced.
