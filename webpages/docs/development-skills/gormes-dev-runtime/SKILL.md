---
name: gormes-dev-runtime
description: Use when running, installing, rebuilding, or validating local Gormes binaries, go run ./cmd/gormes, bin/gormes, install.sh, managed source clones, PATH shadowing, gateway stop/status, or sessions.db lock issues.
---

# Gormes Dev Runtime

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

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
- Shell-wide command: `install.sh`, or manual publish after a build.

`install.sh` is source-backed for final users. Its release defaults may clone
or update `$HOME/.gormes/gormes-agent` from `main`; that is not the agent
development surface. For Gormes agent work, use the current `development`
checkout or set `GORMES_BRANCH=development` when intentionally testing the
installer-managed path. It builds `./cmd/gormes`, publishes to
`$HOME/.local/bin/gormes`, and refreshes any older active `gormes` command that
appears earlier on PATH.

Root Linux defaults are `/usr/local/lib/gormes-agent` and
`/usr/local/bin/gormes`. Override with `GORMES_INSTALL_DIR`,
`GORMES_BIN_DIR`, `GORMES_INSTALL_HOME`, or `GORMES_BRANCH`.

## Dev Rules

- For active uncommitted work, prefer `go run ./cmd/gormes` or `./bin/gormes`.
- Use `go run ./cmd/gormes` to prove source changes before install; use
  `./bin/gormes` to prove the rebuilt local binary; use plain `gormes` only
  after verifying the installed command path.
- Do not ask for a push to `main` to validate development behavior. Agent work
  stays on `development`; local behavior is proven from the checkout or rebuilt
  binary.
- For side-by-side smoke tests, use an isolated home:
  `GORMES_HOME="$(mktemp -d)" go run ./cmd/gormes --offline`. Reuse the real
  home only when validating persisted sessions, migration, or operator state.
- Plain `install.sh` tests the managed branch, not the dirty checkout. For
  Gormes work, set that managed branch to `development` or use the local
  checkout; do not use it to prove uncommitted changes unless an explicit
  local-source mode is implemented.
- If using `GORMES_INSTALL_DIR` against an existing checkout, remember the
  installer may stash local changes, fetch, checkout, and pull the branch.
- After publishing binaries, verify `which -a gormes`, `readlink -f`, and
  `sha256sum` for `gormes`, `bin/gormes`, `$HOME/.gormes/bin/gormes`, and
  `$HOME/go/bin/gormes` when present.
- `gormes gateway status` identifies the live runtime owner. Use
  `gormes gateway stop` when a persisted foreground TUI needs to release
  `$HOME/.gormes/sessions.db`.
- Prefer `gormes gateway reload` or gateway `/reload` for config changes that
  the live manager can swap without reconnecting transports: allowlists,
  first-run discovery flags, display/tool-progress settings, provider/model
  client routing, skills root, and agent bindings. Invalid reloads must keep
  the last-good config active and surface `config_reload=failed` in
  `gormes gateway status`.
- Still use a gateway restart for binary changes, channel transport changes
  that need reconnecting clients, database path changes, or service-manager
  lifecycle work.
- A foreground TUI may run with in-memory session state while the gateway owns
  `sessions.db`; that is acceptable for smoke testing but not for persisted
  session validation.
- If alternating `go run`, `./bin/gormes`, and installed `gormes`, either stop
  the existing owner cleanly or give each surface a different `GORMES_HOME`.
  Do not "fix" a lock by deleting `sessions.db`.
- Gormes startup must not require Hermes `api_server` or suggest
  `hermes gateway start`. Any such output is a parity bug; fix the Gormes
  startup path or installer-published binary, not the user's Hermes process.

## Setup And Auth Regression Checks

Runtime/setup incidents often span multiple surfaces. Build a temp-home matrix
before touching real operator state:

- Run config/setup checks with isolated `GORMES_HOME`, `HERMES_HOME`,
  `CODEX_HOME`, `XDG_CONFIG_HOME`, and `XDG_DATA_HOME`.
- Use synthetic Telegram/provider tokens in tests; never echo, commit, or
  report real tokens.
- For Codex provider setup, include a synthetic `CODEX_HOME/auth.json` case:
  fresh Codex CLI auth should import into the Gormes credential pool, expired
  Codex CLI auth should fall back to device-code, and output must stay
  redacted.
- Cover the operator-facing chain that broke: `config set`, config reload,
  `onboard`, `doctor --offline`, `gateway status --json`, and the exact
  binary surface (`go run`, `./bin/gormes`, or installed `gormes`) implicated
  by the report.
- If the user provides live settings, install them only after a sanitized
  dry-run fixture passes, then verify with redacted commands. Document any
  blockers without printing secret values.

## Incident Checklist

When validating an installer/runtime report, write down these facts before
changing code:

- command run: `go run ./cmd/gormes`, `./bin/gormes`, or plain `gormes`;
- binary path and realpath from `which -a gormes` / `readlink -f`;
- source checkout used by the binary or installer;
- `GORMES_HOME` and whether it is a temp dir, `workspace-gormes`, or
  `~/.gormes`;
- whether any gateway/TUI process owns `sessions.db`;
- whether output contains Hermes-owned instructions, `~/.hermes` state reads,
  stale product labels, or API-server assumptions.

Only after this matrix is clear should a parity report move to TUI,
Telegram/channel, or provider work. Many visible UX bugs are stale binary or
wrong-home bugs until this checklist says otherwise.

## Verification

- Installer behavior: `(cd webpages/landing && go test ./internal/site -run 'TestInstallSH' -count=1)`.
- Runtime package smoke: `go test ./cmd/gormes ./internal/session -count=1`.
- Gateway reload smoke: `go test ./cmd/gormes ./internal/gateway -run 'TestGatewaySignalLoop|TestGatewayReload|TestManager_ReloadCommand' -count=1`.
- Source/binary/install matrix: run the same focused command through
  `GORMES_HOME="$(mktemp -d)" go run ./cmd/gormes ...`,
  `GORMES_HOME="$(mktemp -d)" ./bin/gormes ...`, and installed `gormes` after
  path verification when the bug involves publishing or PATH shadowing.
- Full gate before completion claims: `go test ./... -count=1`,
  `go run ./cmd/progress validate`, and `git diff --check`.

## Common Mistakes

- Assuming plain `install.sh` builds the dirty repo.
- Using a final-user `main` install path to validate agent development work.
- Testing `gormes` while `$HOME/go/bin/gormes` shadows the freshly published
  command.
- Killing random processes instead of using `gormes gateway status` and
  `gormes gateway stop`.
- Restarting the gateway for every config edit before trying `gormes gateway
  reload`, or treating a failed reload as a partially-applied config.
- Deleting lock files or `sessions.db` instead of stopping the owning Gormes
  process or using an isolated `GORMES_HOME`.
- Reintroducing Hermes commands or Hermes state reads into Gormes startup.
- Treating an in-memory `sessions.db` fallback as proof of persisted-session
  behavior.
