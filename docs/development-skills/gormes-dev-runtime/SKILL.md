---
name: gormes-dev-runtime
description: Use when running, installing, rebuilding, or validating local Gormes binaries, go run ./cmd/gormes, bin/gormes, install.sh, managed source clones, PATH shadowing, gateway stop/status, or sessions.db lock issues.
---

# Gormes Dev Runtime

## Mission

Keep three execution surfaces distinct: the dirty development checkout, the local
compiled binary, and the shell-wide installed command. Never let Hermes or a
stale PATH entry become part of the Gormes dev loop.

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

## Verification

- Installer behavior: `(cd www.gormes.ai && go test ./internal/site -run 'TestInstallSH' -count=1)`.
- Runtime package smoke: `go test ./cmd/gormes ./internal/session -count=1`.
- Full gate before completion claims: `go test ./... -count=1`,
  `go run ./cmd/progress validate`, and `git diff --check`.

## Common Mistakes

- Assuming plain `install.sh` builds the dirty repo.
- Forgetting that final-user installs default to `main`.
- Testing `gormes` while `$HOME/go/bin/gormes` shadows the freshly published
  command.
- Killing random processes instead of using `gormes gateway status` and
  `gormes gateway stop`.
- Reintroducing Hermes commands or Hermes state reads into Gormes startup.
