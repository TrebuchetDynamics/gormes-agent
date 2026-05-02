# Releasing Gormes

This document describes how maintainers ship a new Gormes-Agent release.

## Release Philosophy

- **Source-first**: The recommended install path is building from source.
- **Signed artifacts**: Every release includes SHA-256 checksums and build provenance attestations.
- **Automated**: Tagging triggers the release pipeline; manual steps are minimal and documented.
- **Green-main only**: Release tags are created from `main` only, after the
  `development -> main` pull request is merged and required CI is green on the
  exact `main` commit that will be tagged. If CI is red, pending, skipped, or
  unknown, do not tag and do not publish a release.

## Versioning

Gormes uses SemVer. The project is still in the `0.x` compatibility window;
older internal preview lines used a `-scout` suffix, but public release tags
should use the exact version string from `cmd/gormes/version.go`.

Examples:
- `0.1.0` — first source-first public release line
- `0.1.1` — patch release
- `0.2.0-scout` — optional future preview line

## Release Methods

### Method 1: Automated (Recommended)

1. Go to **Actions → Release Preparation** in GitHub.
2. Click **Run workflow**.
3. Select bump type (`patch`, `minor`, `major`).
4. Optionally enter a prerelease suffix (e.g., `scout`).
5. The workflow will:
   - Run the full test suite
   - Bump `cmd/gormes/version.go`
   - Update `CHANGELOG.md`
   - Regenerate progress docs
   - Open a pull request to `main`

6. Review the PR. Do not merge until required CI is green.
7. Merge the PR into `main`.
8. Confirm `main` is on the merged commit and required CI is green for that
   exact commit.
9. After the green-main check, create and push the tag:
   ```bash
   git checkout main
   git pull --ff-only origin main
   gh run list --branch main --commit "$(git rev-parse HEAD)" --limit 5
   git tag -a v0.X.Y-scout -m "Release 0.X.Y-scout"
   git push origin v0.X.Y-scout
   ```

10. The **Release Binaries** workflow triggers automatically and publishes:
   - Six platform archives (linux/darwin/windows × amd64/arm64)
   - SHA-256 checksums
   - SPDX JSON SBOMs
   - Build provenance attestations

### Method 2: Local Bump Script

```bash
./scripts/bump-version.sh patch scout
# Review changes
git add cmd/gormes/version.go CHANGELOG.md docs/
git commit -m "release: bump version to 0.X.Y-scout"
git push origin main
```

Then create and push the tag as in Method 1.

### Method 3: Manual Workflow Dispatch

For hotfixes or custom builds:

1. Go to **Actions → Release Binaries**.
2. Click **Run workflow**.
3. Enter the version string (without leading `v`).
4. The workflow builds and publishes artifacts without requiring a tag.

## Artifact Verification

Users can verify releases:

```bash
# Verify SHA-256
curl -LO https://github.com/TrebuchetDynamics/gormes-agent/releases/download/v0.1.0/gormes-0.1.0-linux-amd64.tar.gz
curl -LO https://github.com/TrebuchetDynamics/gormes-agent/releases/download/v0.1.0/gormes-0.1.0-linux-amd64.tar.gz.sha256
sha256sum -c gormes-0.1.0-linux-amd64.tar.gz.sha256

# Verify build provenance (requires gh CLI)
gh attestation verify gormes-0.1.0-linux-amd64.tar.gz \
  --repo TrebuchetDynamics/gormes-agent
```

## Pre-Release Checklist

- [ ] `go test ./... -count=1` passes
- [ ] `go run ./cmd/progress validate` passes
- [ ] `git diff --check` is clean
- [ ] `CHANGELOG.md` is updated
- [ ] `cmd/gormes/version.go` matches the intended tag
- [ ] README progress rollup is regenerated
- [ ] `development -> main` PR is merged
- [ ] Required CI is green on the exact `main` commit to tag
