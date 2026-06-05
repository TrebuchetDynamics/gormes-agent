# Gormes Native Update Command Design

**Status:** Draft
**Author:** Codex
**Date:** 2026-05-15

## Context

Hermes `hermes update` is an operator-facing full update flow: it checks for
remote commits, protects local state, updates source, refreshes runtime
dependencies, syncs bundled skills, handles config migrations, and restarts
running gateways with drain evidence.

Gormes already has a native `gormes update` command, but it is not yet a full
operator update. It updates a managed source checkout and emits structured
evidence, while the installed `gormes` command is still refreshed by rerunning
`install.sh`. That split is misleading: an operator can see "update complete"
while the command on `PATH` still points at the previous binary.

Gormes should keep the Hermes update contract where it matters to operators,
but remain Go-native. It must not add Hermes' Python dependency installation,
`uv pip install`, or `__pycache__` cleanup phases.

## Goals

1. Make `gormes update` a self-contained managed-install updater.
2. Rebuild and atomically republish the installed `gormes` binary after a
   successful source update.
3. Refresh a shadowing active `gormes` command on `PATH` when safe, matching
   the existing installer policy.
4. Make `gormes update --check` contact the remote and report exact update
   state without mutating the checkout or binaries.
5. Restart live gateways only through validated Gormes gateway/service state.
6. Preserve structured human and JSON evidence for every meaningful phase.
7. Keep the command safe for dirty checkouts, failed builds, failed publishes,
   and sandbox/custom install boundaries.

## Non-goals

- Installing or updating Python packages, Node packages outside existing web UI
  build behavior, or any Hermes runtime dependency.
- Replacing `install.sh` as the bootstrap path for fresh machines.
- Rewriting the whole installer in Go.
- Discovering and killing arbitrary `gormes gateway` processes.
- Changing Gormes runtime data layout, profile layout, or provider setup.
- Implementing Windows Scheduled Task gateway lifecycle beyond the existing
  gateway restart surfaces.

## Accepted Contract

1. `gormes update` should rebuild and atomically republish the installed binary
   when operating on a managed install.
2. The build/publish pipeline should be native Go code, not a shell-out to
   `install.sh`.
3. The active `PATH` command should be refreshed when it shadows the managed
   binary, but only under the default managed-install safety boundary.
4. `GORMES_BIN_DIR` and `GORMES_PREFIX` mean "custom/sandbox boundary"; update
   must not rewrite unrelated active `PATH` commands in that mode.
5. `--restart-gateway=auto` should restart only gateways proven by Gormes
   runtime/service records.
6. `--restart-gateway=always` should fail the update report when restart
   validation fails.
7. `--restart-gateway=never` should publish the binary and skip restart, with
   evidence when a live gateway still needs refresh.
8. `--check` should fetch the configured branch and report current commit,
   remote commit, and commits behind, with no checkout, stash, pull, build, or
   publish.

## Architecture

The updater keeps four execution surfaces distinct:

```text
managed source checkout  ->  managed build output  ->  published command
                                                        active PATH command
```

- **Managed source checkout:** resolved from `GORMES_INSTALL_DIR`, otherwise
  `$GORMES_INSTALL_HOME/gormes-agent`.
- **Managed build output:** `$GORMES_INSTALL_HOME/bin/gormes` on Unix-like
  systems, and the equivalent managed bin path on Windows.
- **Published command:** the configured install bin path, normally
  `$HOME/.local/bin/gormes` for non-root Unix installs.
- **Active PATH command:** any earlier `gormes` command found by PATH lookup
  that would shadow the freshly published binary.

Normal update flow:

1. Resolve managed checkout and install paths.
2. Resolve backup policy and create an optional pre-update backup.
3. Verify the managed checkout is a git worktree.
4. Switch to the configured branch if needed.
5. Autostash dirty local changes.
6. Fetch and fast-forward pull, with reset fallback evidence for divergence.
7. Restore or preserve autostash deterministically.
8. Build a new `gormes` binary from the updated checkout.
9. Verify the new binary by running `gormes version`.
10. Atomically publish the binary to the managed/published command path.
11. Refresh a shadowing active `PATH` command when allowed by path policy.
12. Sync bundled skills, build web UI, and check config migrations.
13. Apply the requested gateway restart policy.

Check flow:

1. Verify the checkout is a git worktree.
2. Fetch the configured remote branch.
3. Resolve `HEAD`, `origin/<branch>`, and `HEAD..origin/<branch>` count.
4. Emit a human or JSON report and exit without mutation.

## Components

### `internal/cli.UpdateCheckRunner`

Fetches remote state and returns:

- branch
- current commit
- remote commit
- commits behind
- evidence kind and detail on failure

The production adapter uses git. Tests use a fake runner.

### `internal/cli.UpdateBinaryPublisher`

Builds, verifies, and publishes the new binary. It owns:

- build command construction
- temp binary path selection
- version verification
- atomic replacement
- rollback of a previously published binary when verification fails after
  replacement
- evidence for build, publish, verify, and rollback results

### `internal/cli.UpdatePathPolicy`

Encodes when it is safe to refresh a shadowing active `PATH` command. It should
mirror `install.sh`:

- Default managed install: refresh active shadowing command.
- `GORMES_BIN_DIR` set: skip active PATH refresh.
- `GORMES_PREFIX` set: skip active PATH refresh.
- Missing active command: no-op.
- Active command already equals published command: no-op.

### `cmd/gormes` Adapters

The command layer resolves real paths and wires production dependencies:

- managed checkout resolver
- managed bin resolver
- published bin resolver
- git runner
- Go build runner
- binary version verifier
- active PATH lookup
- gateway restart/status seams

The command layer should remain thin; most behavior belongs in `internal/cli`
so it can be tested without touching the real machine.

## Evidence Model

Add or use these evidence kinds:

- `update_check`
- `update_check_available`
- `update_check_current`
- `update_build_completed`
- `update_build_failed`
- `update_publish_completed`
- `update_publish_failed`
- `update_verify_failed`
- `update_publish_rollback_completed`
- `update_publish_rollback_failed`
- `update_active_path_refreshed`
- `update_active_path_skipped`
- `update_active_path_failed`
- `update_gateway_restart_needed`

Existing evidence kinds for git, backup, skill sync, web build, config
migration, and gateway restart should remain stable.

JSON output must include the same evidence as human output. Human output can be
compact; JSON should preserve enough detail for cron/fleet consumers.

## Error Handling

- **Remote check failure:** fail nonzero with `update_network_error`,
  `update_auth_error`, or `update_git_error`; do not mutate.
- **Git update failure after autostash:** preserve stash and emit manual
  recovery guidance.
- **Build failure:** leave published binaries unchanged and emit
  `update_build_failed`.
- **Publish failure:** leave the previous published binary in place when
  possible and emit `update_publish_failed`.
- **Verify failure after replacement:** attempt rollback, then emit
  `update_verify_failed` plus rollback evidence.
- **Active PATH refresh failure:** do not roll back the main published binary;
  emit warning evidence because the managed install succeeded.
- **Gateway restart failure under `auto`:** warn, keep update successful.
- **Gateway restart failure under `always`:** fail the update report.
- **Gateway restart under `never`:** skip the attempt and emit
  `update_gateway_restart_needed` when a live gateway is detected.
- **Backup failure:** warn and continue, matching current behavior.

No evidence may include secrets, API keys, auth tokens, or raw private runtime
home contents.

## Testing Plan

### Pure Lifecycle Tests

Use fake git/build/publish/path/restart seams to prove ordering:

```text
backup -> git update -> build -> publish -> active PATH refresh
       -> skill sync -> web build -> config check -> gateway restart
```

Also prove failure short-circuits at the correct boundary.

### Check Tests

Prove `--check`:

- fetches remote state
- reports current and remote commit
- reports commits behind
- exits with no checkout, stash, pull, build, publish, skill sync, web build, or
  restart calls
- returns nonzero on network/auth/git errors
- preserves JSON shape

### Publisher Tests

Use temp directories and fake executables where possible:

- successful build and publish
- failed build leaves old binary unchanged
- failed verify rolls back
- failed rollback is reported
- active PATH refresh happens under default policy
- active PATH refresh is skipped under `GORMES_BIN_DIR` and `GORMES_PREFIX`

### Command Tests

Prove:

- flags wire to lifecycle options
- real path resolvers do not fall back to `os.Getwd`
- `--json` suppresses human banners
- human output includes build/publish evidence
- failed update exits nonzero while preserving parseable JSON

### Focused Verification

Run focused tests first:

```sh
go test ./cmd/gormes ./internal/cli -run 'TestUpdate|Test.*Publish|Test.*Check' -count=1
go run ./cmd/progress validate
git diff --check
```

Run `go test ./... -count=1` before claiming full repository readiness or
before opening/updating a PR.

## Implementation Slices

### Slice 1: Remote Check

Implement real `--check` semantics with typed evidence and JSON output. No
build or publish behavior changes.

### Slice 2: Native Build And Publish

Add build/publish seams and wire them into normal update after the git update
phase and before skill/web/config phases. Include atomic publish and rollback.

### Slice 3: Active PATH Refresh

Add path policy and refresh active shadowing `gormes` commands under default
managed-install rules.

### Slice 4: Gateway Restart Wiring

Wire production update restart policy through existing validated gateway
restart/status paths. Keep process discovery conservative.

### Slice 5: Update Log And Hangup Protection

Wire existing update-log mirroring and SIGHUP helpers into production update
without changing JSON output semantics.

## Open Follow-ups

- Windows publish and Scheduled Task restart behavior may need a dedicated
  slice if the Unix-like publisher cannot be expressed portably.
- Add a lightweight automatic pre-update runtime snapshot, analogous to Hermes'
  quick snapshot, after binary publish semantics are safe.
- Docs should be updated after Slice 2 so public CLI docs no longer imply that
  source-only update is the full operator update.
