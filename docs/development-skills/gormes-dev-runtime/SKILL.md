---
name: gormes-dev-runtime
description: Use when running, installing, rebuilding, or validating local Gormes binaries, go run ./cmd/gormes, bin/gormes, install.sh, managed source clones, PATH shadowing, gateway stop/status, or sessions.db lock issues.
---

# Gormes Dev Runtime

## Mission

Keep three execution surfaces distinct: the dirty development checkout, the local
compiled binary, and the shell-wide installed command. Never let Hermes or a
stale PATH entry become part of the Gormes dev loop.

Start by proving which surface the user is actually running. Use `pwd`,
`git rev-parse --show-toplevel`, `which -a gormes`, and `readlink -f` before
changing installers, binaries, or config. If the user names a development
workspace, honor that path; do not silently fall back to a sibling checkout or
the installer-managed clone.

Treat source checkout, runtime home, installer-managed checkout, and PATH
binary as separate facts. For this workspace family, `workspace-mineru` may be
the editing workspace while `workspace-gormes` or `GORMES_HOME` is the Gormes
development/runtime home. Discover and pass those paths through environment or
fixtures; do not hard-code `/home/xel/...` paths into product code or tests.

## Choose The Surface

- Current checkout, including uncommitted edits: `go run ./cmd/gormes`.
- Compiled binary from this checkout: `go build -o bin/gormes ./cmd/gormes`,
  then `./bin/gormes`.
- Shell-wide command: `scripts/install.sh`, or manual publish after a build.

`install.sh` is source-backed for final users. By default it clones or updates
`$HOME/.gormes/gormes-agent`, uses `GORMES_BRANCH=main`, builds
`./cmd/gormes`, publishes to `$HOME/.local/bin/gormes`, and refreshes any older
active `gormes` command that appears earlier on PATH.

Root Linux defaults are `/usr/local/lib/gormes-agent` and
`/usr/local/bin/gormes`. Override with `GORMES_INSTALL_DIR`,
`GORMES_BIN_DIR`, `GORMES_INSTALL_HOME`, or `GORMES_BRANCH`.

## Dev Rules

- For active uncommitted work, prefer `go run ./cmd/gormes` or `./bin/gormes`.
- Use `go run ./cmd/gormes` to prove source changes before install; use
  `./bin/gormes` to prove the rebuilt local binary; use plain `gormes` only
  after verifying the installed command path.
- For side-by-side smoke tests, use an isolated home:
  `GORMES_HOME="$(mktemp -d)" go run ./cmd/gormes --offline`. Reuse the real
  home only when validating persisted sessions, migration, or operator state.
- Plain `install.sh` tests the managed branch, not the dirty checkout. Do not
  use it to prove uncommitted changes unless an explicit local-source mode is
  implemented.
- If using `GORMES_INSTALL_DIR` against an existing checkout, remember the
  installer may stash local changes, fetch, checkout, and pull the branch.
- After publishing binaries, verify `which -a gormes`, `readlink -f`, and
  `sha256sum` for `gormes`, `bin/gormes`, `$HOME/.gormes/bin/gormes`, and
  `$HOME/go/bin/gormes` when present.
- `gormes gateway status` identifies the live runtime owner. Use
  `gormes gateway stop` when a persisted foreground TUI needs to release
  `$HOME/.gormes/sessions.db`.
- A foreground TUI may run with in-memory session state while the gateway owns
  `sessions.db`; that is acceptable for smoke testing but not for persisted
  session validation.
- If alternating `go run`, `./bin/gormes`, and installed `gormes`, either stop
  the existing owner cleanly or give each surface a different `GORMES_HOME`.
  Do not "fix" a lock by deleting `sessions.db`.
- Gormes startup must not require Hermes `api_server` or suggest
  `hermes gateway start`. Any such output is a parity bug; fix the Gormes
  startup path or installer-published binary, not the user's Hermes process.

## Verification

- Installer behavior: `(cd www.gormes.ai && go test ./internal/site -run 'TestInstallSH' -count=1)`.
- Runtime package smoke: `go test ./cmd/gormes ./internal/session -count=1`.
- Source/binary/install matrix: run the same focused command through
  `GORMES_HOME="$(mktemp -d)" go run ./cmd/gormes ...`,
  `GORMES_HOME="$(mktemp -d)" ./bin/gormes ...`, and installed `gormes` after
  path verification when the bug involves publishing or PATH shadowing.
- Full gate before completion claims: `go test ./... -count=1`,
  `go run ./cmd/progress validate`, and `git diff --check`.

## Common Mistakes

- Assuming plain `install.sh` builds the dirty repo.
- Forgetting that final-user installs default to `main`.
- Testing `gormes` while `$HOME/go/bin/gormes` shadows the freshly published
  command.
- Killing random processes instead of using `gormes gateway status` and
  `gormes gateway stop`.
- Deleting lock files or `sessions.db` instead of stopping the owning Gormes
  process or using an isolated `GORMES_HOME`.
- Reintroducing Hermes commands or Hermes state reads into Gormes startup.
- Treating an in-memory `sessions.db` fallback as proof of persisted-session
  behavior.
