# Gormes Install — Known Issues

Living register of install-time issues found by `gormes-install` passes.
Each entry stays here until a corresponding `progress.json` row marks the
fix complete, then the entry's `status` is updated to
`fixed-in-v<version>` and the recipe stays as a regression-prevention
witness.

## Format

```
### [<short-id>] <one-line summary>

- **First seen**: <UTC timestamp> against `<install.sh path>@<sha>`
- **Surface touched**: <absolute path or seam>
- **Sandbox env vars set**: <list>
- **Trigger**: <minimal repro>
- **Symptom**: <what the pre/post diff showed>
- **Status**: `open` | `mitigated` | `fixed-in-v<version>`
- **Routing**: `gormes-planner` row id, if filed
```

---

## Open Issues

### [iso-bin-hijack] Sandbox install hijacks production `~/.local/bin/gormes` symlink

- **First seen**: 2026-05-07T17:27Z against `install.sh` at commit `08706c6d3`
  (development before v0.1.06 squash-merge; same behavior on `1e9c0a026`).
- **Surface touched**: `/home/<user>/.local/bin/gormes` (symlink target).
- **Sandbox env vars set**: `GORMES_INSTALL_HOME=$SANDBOX/home`,
  `GORMES_BIN_DIR=$SANDBOX/bin`, `GORMES_SKIP_SETUP=1`,
  `GORMES_RESTART_GATEWAY=never`.
- **Trigger**: `sh ./install.sh --verbose --skip-setup --restart-gateway never`
  on a host that already has `gormes` on `PATH` outside the sandbox.
- **Symptom**: install log emits
  `updating active PATH command /home/<user>/.local/bin/gormes`
  and replaces the existing symlink with one pointing at the **sandbox**
  build (`/tmp/gormes-install-test/<ts>/home/bin/gormes`). After the
  sandbox dir is cleaned (e.g., `/tmp` reaped on reboot), the production
  symlink dangles.
- **Status**: `fixed-in-2026-05-07`.
- **Fix**: `install.sh` now defines `sandbox_bin_dir_set()`
  (`[ -n "$GORMES_BIN_DIR" ] || [ -n "$GORMES_PREFIX" ]`).
  `update_active_command()` early-returns when true with the log line
  `skipping active PATH command update (sandbox bin dir set via
  GORMES_BIN_DIR; respecting boundary)`. `print_install_plan_body` and
  `print_verbose_plan` both surface the decision as
  `update_active_path_command: skipped|yes`.
- **Regression fence**: `internal/installtest/iso_bin_dir_test.go` covers
  the skipped path for `GORMES_BIN_DIR`, the skipped path for
  `GORMES_PREFIX`, the default-yes path, and the verbose-includes-reason
  path — all via fast `--dry-run` plan inspection (no git clone, no
  `go build`). End-to-end manual verification on 2026-05-07 confirmed a
  real sandbox install with `GORMES_BIN_DIR` set leaves the production
  `~/.local/bin/gormes` symlink target unchanged.

### [iso-shellrc-leak] Sandbox install permanently edits `~/.bashrc` and `~/.profile`

- **First seen**: 2026-05-07T17:27Z (same pass as iso-bin-hijack).
- **Surface touched**: `/home/<user>/.bashrc`, `/home/<user>/.profile`.
- **Sandbox env vars set**: same as above.
- **Trigger**: same as above. Triggered even though `GORMES_BIN_DIR` was
  set to a sandbox path.
- **Symptom**: install log emits
  `added /tmp/gormes-install-test/<ts>/bin to PATH in: ~/.bashrc ~/.profile`
  and writes two lines to each rc:
  ```sh
  # Gormes installer — added /tmp/gormes-install-test/<ts>/bin to PATH
  export PATH="/tmp/gormes-install-test/<ts>/bin:$PATH"
  ```
  After the sandbox dir is cleaned, every new login shell has a dangling
  PATH entry. There is no installer-emitted cleanup path — the operator
  has to `sed` it out by hand.
- **Status**: `fixed-in-2026-05-07`.
- **Fix**: `install.sh`'s `ensure_path_in_shell_config()` now early-returns
  with `PATH_CONFIG_RESULT=sandbox_skipped` when `sandbox_bin_dir_set`
  is true, after exporting the bin dir into the current install run's
  `PATH` (so downstream verification still works). It logs
  `skipping shell rc PATH edits (sandbox bin dir set via
  GORMES_BIN_DIR; respecting boundary — ~/.bashrc, ~/.profile,
  ~/.zshrc, fish config left untouched)`. `print_install_plan_body` and
  `print_verbose_plan` both surface the decision as
  `edit_shell_rc_files: skipped|yes`.
- **Regression fence**: `internal/installtest/iso_shellrc_test.go` covers
  the skipped path for `GORMES_BIN_DIR`, the skipped path for
  `GORMES_PREFIX`, and the default-yes regression fence — all via fast
  `--dry-run` plan inspection. End-to-end manual verification on
  2026-05-07 confirmed a real sandbox install with `GORMES_BIN_DIR` set
  left the production `~/.bashrc` and `~/.profile` `gormes`-line counts
  byte-identical.
- **Note on the original heuristic**: the planner row originally proposed
  a "bin dir under `/tmp/`" detection heuristic and a separate
  `--no-shell-init` flag. The shipped fix uses the operator-explicit
  `sandbox_bin_dir_set` boundary instead because that is the actual
  user-intent signal and is consistent with the iso-bin-hijack and
  iso-systemd-hijack fixes (one boundary, three protected surfaces).

### [iso-systemd-hijack] Sandbox install rewrites production user systemd unit to point at sandbox binary

- **First seen**: 2026-05-07T18:35Z during the SECOND sandbox pass (the
  one that verified iso-bin-hijack stays fixed).
- **Surface touched**:
  `/home/<user>/.config/systemd/user/gormes-gateway.service`.
- **Sandbox env vars set**: `GORMES_INSTALL_HOME=$SANDBOX/home`,
  `GORMES_BIN_DIR=$SANDBOX/bin`, `GORMES_SKIP_SETUP=1`,
  `GORMES_RESTART_GATEWAY=never`.
- **Trigger**: `sh ./install.sh --skip-setup --restart-gateway never` on
  a Linux host with systemd-user available
  (`systemctl --user >/dev/null 2>&1` succeeds) and an existing
  production `gormes-gateway.service` already enabled.
- **Symptom**: install log prints
  `systemd user service installed:` and the production unit file's
  `ExecStart`, `ExecReload`, and `Environment=GORMES_HOME=…` are all
  rewritten to point at the sandbox path:
  ```ini
  ExecStart=/tmp/gormes-install-test/<ts>/home/bin/gormes gateway
  Environment=GORMES_HOME=/tmp/gormes-install-test/<ts>/home
  ```
  After the next `/tmp` reap or reboot, the gateway service fails to
  start because its binary path no longer exists. Worse than
  iso-bin-hijack: the failure is invisible until the operator tries to
  use the gateway. The unit file mtime jumps and `systemctl --user
  daemon-reload` is also issued during the install, so the broken state
  is loaded into the manager immediately.
- **Status**: `fixed-in-2026-05-07`.
- **Fix**: `install.sh`'s `print_service_instructions()` dispatcher now
  early-returns when `sandbox_bin_dir_set` is true, with the log line
  `skipping system service install (sandbox bin dir set via
  GORMES_BIN_DIR; respecting boundary — ~/.config/systemd/user/ and
  ~/Library/LaunchAgents/ left untouched)`. `print_install_plan_body`
  and `print_verbose_plan` both surface the decision as
  `install_system_service: skipped|yes`. The same fix protects the
  macOS launchd plist path because both run through the same
  dispatcher.
- **Regression fence**: `internal/installtest/iso_systemd_dir_test.go`
  covers the skipped path for `GORMES_BIN_DIR`, the skipped path for
  `GORMES_PREFIX`, and the default-yes regression fence — all via fast
  `--dry-run` plan inspection. End-to-end manual verification on
  2026-05-07 confirmed a real sandbox install with `GORMES_BIN_DIR` set
  left the production `~/.config/systemd/user/gormes-gateway.service`
  sha256 byte-identical.

### [termux-publish-symlink-noexec] Termux published command symlinks into exec-blocked `$HOME`, verify fails, rolls back

- **First seen**: 2026-05-15 against the published `v0.2.12` release
  `install.sh` (latest-release `binary-fetch`), on a real Termux/Android
  device. Witness: operator transcript (binary-fetch android-arm64,
  SHA-256 verified, extracted to `~/.gormes/bin/gormes`, then
  `✗ published command verification failed for
  /data/data/com.termux/files/usr/bin/gormes; rolled back`).
- **Surface touched**: `/data/data/com.termux/files/usr/bin/gormes`
  (`$PREFIX/bin`, the PATH-published command) — left **absent** after
  rollback, so a fresh Termux user has no working `gormes`.
- **Environment**: Termux on Android (API 29+), `is_termux` true,
  `pick_bin_dir` → `$PREFIX/bin`, `managed_bin_dir` → `$HOME/.gormes/bin`.
- **Root cause**: `install.sh:publish_built_binary` prefers
  `ln -s "$build_bin" "$tmp"` (install.sh:1523). On Termux `ln -s`
  succeeds, so `$PREFIX/bin/gormes` becomes a **symlink to
  `$HOME/.gormes/bin/gormes`**. The post-publish verify
  `"$published_bin" version` (install.sh:1530) execs the symlink, which
  the kernel resolves+execs at the target path under
  `/data/data/com.termux/files/home` (app-private writable storage).
  Android 10+ blocks `execve()` of files in app-writable data dirs
  (W^X); Termux only makes `$PREFIX` exec-capable. The exec is denied,
  `version` fails, and the rollback removes the published command.
- **Not a build defect**: the android-arm64 asset is a
  `CGO_ENABLED=0 GOOS=android` static Go binary (`release.yml:95-96,148`);
  it runs fine when executed from an exec-permitted path such as
  `$PREFIX/bin`. The defect is install.sh placing a `$HOME`-targeting
  symlink instead of a real executable in `$PREFIX/bin`.
- **Reproduction (source-backed; needs a real Termux device to run)**:
  `curl …/releases/latest/download/install.sh | sh` on Termux/Android
  arm64 with no pre-existing `$PREFIX/bin/gormes`.
- **Status**: `fixed-in-development` for the next release. `install.sh`
  now skips the symlink path when `is_termux` is true and copies the real
  binary into `$PREFIX/bin` before running the existing published-command
  verification. Non-Termux hosts keep the symlink-preferred behavior.
- **Regression fence**:
  `internal/installtest/termux_publish_test.go` covers forced-Termux
  publication as a regular non-`$HOME` file and covers the unchanged
  non-Termux symlink path.

### [termux-exec-args-injection] Termux `termux-exec` injects binary path into `os.Args`, breaking Cobra parsing

- **First seen**: 2026-05-18 against operator transcript on Termux/Android
  16 (Samsung SM-S, API 36). Witness: after install rollback, running
  `gormes --version` produced
  `Error: unknown command "/data/data/com.termux/files/usr/bin/gormes" for "gormes"`.
- **Surface touched**: `cmd/gormes/main.go` argument parsing — every
  invocation on affected Termux builds is broken because the first
  positional argument is the binary's absolute path.
- **Environment**: Termux on Android 10+ with `termux-exec` active
  (`LD_PRELOAD` intercepts `exec()` and delegates through
  `/system/bin/linker64`). `termux-app` issue #4630 documents that
  compiled Go programs receive their own full path as `os.Args[1]`.
- **Root cause**: `termux-exec` inserts the absolute path to the
  executable as an extra argument in `argv`. When the operator runs
  `gormes --version`, the Go runtime sees
  `os.Args = ["gormes", "/data/data/com.termux/files/usr/bin/gormes", "--version"]`
  (or similar, depending on invocation path). `main.go` passes
  `os.Args[1:]` to Cobra, so Cobra receives
  `["/data/data/com.termux/files/usr/bin/gormes", "--version"]` and
  treats the absolute path as an unknown subcommand name, failing before
  it ever processes `--version`.
- **Not a build defect**: the android-arm64 binary is correct; this is a
  Termux runtime `exec()` family wrapper behavior that affects Go programs
  specifically.
- **Reproduction**: Build or install `gormes` on a Termux device where
  `termux-exec` is loaded, then run any command such as
  `gormes --version`, `gormes version`, or `gormes doctor --offline`.
- **Status**: `fixed-in-development` for the next release; not fixed in
  public latest `v0.2.20`. Evidence on 2026-05-21: `v0.2.20` points at
  tag commit `c27835f25d32`, `origin/development` points at
  `df1c011d61c3`, and no `v0.2.21` tag exists yet. Operators should
  treat the latest public Termux installer as affected until a follow-up
  release is published.
  `cmd/gormes/main.go:sanitizeTermuxExecArgs` detects the injected path
  by comparing `os.Args[1]` (and `os.Args[2]` in the less-common shift
  case) against `os.Executable()`. When they match on `GOOS=android`,
  the duplicate is stripped before the args reach Cobra. A 2026-05-21
  latest-release transcript showed v0.2.20 still failed when Android
  reported the executable through `/data/user/0/com.termux/...` while
  `termux-exec` injected the equivalent `/data/data/com.termux/...` path.
  Development now normalizes that Termux package-path alias before
  comparing, so `gormes --version`, `gormes version`, installer
  publish-verification, and all subcommands work correctly under Termux
  without affecting Linux, macOS, or Windows. The installer also treats a
  Termux binary-fetch publish-verification failure as a recoverable release
  binary runtime failure: it rolls back the bad copy, falls back to
  source-build once, and republishes a verified command.
- **Regression fence**:
  `cmd/gormes/termux_exec_args_test.go` covers both injection positions
  (args[0] and args[1]), `/data/data` versus `/data/user/0` Termux path
  aliases, normal args passthrough, and the empty-exe no-op path.
  `internal/installtest/termux_publish_recovery_test.go` covers the
  binary-fetch publish-verification fallback to source-build and the final
  rolled-back error when the source-build retry also cannot run `version`.

---

## Mitigated Issues

(none yet)

---

## Withdrawn (False Positives)

### [msg-systemd-fiction] Install reports systemd user service installed even when no service was created — RETRACTED 2026-05-07

- **Originally filed**: 2026-05-07T17:27Z under "Open Issues".
- **Withdrawn**: 2026-05-07T17:38Z after re-verification.
- **Why retracted**: the original verification command
  `systemctl --user list-unit-files 'gormes*'` returned `0 unit files listed`
  at the moment the original sandbox pass checked, which was misread as
  "no unit was written." Re-checking shortly after the same install pass
  with the same command (and with `systemctl --user is-enabled gormes-gateway`,
  and with `ls ~/.config/systemd/user/gormes-gateway.service`) all
  confirm the unit file IS present and IS enabled. The install transcript
  was truthful.
- **Root cause of the misdiagnosis**: `systemctl --user list-unit-files`
  may transiently miss a just-`daemon-reload`'d unit until systemd
  re-scans its manager state. A single negative result is not authoritative.
- **Verification lesson** (added to the workflow): when checking whether
  a systemd user unit was actually written, ALWAYS verify with at least
  two independent signals before concluding "no unit":
  1. `ls -la ~/.config/systemd/user/gormes-gateway.service` (file existence
     is the ground truth).
  2. `systemctl --user is-enabled gormes-gateway` (manager-loaded state).
  3. Optionally `systemctl --user list-unit-files` without a glob pattern
     (the pattern flag occasionally misses recently-loaded units).
- **Filing note**: the corresponding `progress.json` row "Install transcript:
  only print systemd block when unit file actually written" was withdrawn
  in the same retraction. install.sh's existing gating
  (`if has systemctl && systemctl --user >/dev/null 2>&1; then
  install_systemd_user_service; fi`) was already correct; no fix needed.

---

## Fixed Issues

(none yet)

---

## Coverage Gaps Worth Closing

These have not been exercised by a `gormes-install` pass yet. File
follow-up rows when one becomes urgent.

- `--uninstall` flow on a sandbox install (does it leave `~/.gormes`,
  shell rc, systemd, cron entries behind?).
- `--local` flag (build from current checkout instead of cloning).
- Fresh-machine flow (no Go, no git, no previous `~/.gormes`) — likely
  needs a container fixture.
- Termux on Android (different `HOME`, no systemd, no `/usr/local/bin`).
- WSL2 (mixed Windows/Linux PATH semantics).
- Root install on RHEL/Rocky/Alma (the FHS layout branch in `install.sh`).
- `install.ps1` on native Windows (separate skill pass — out of scope
  for this Linux/macOS skill until a Windows test host is available).
- `curl https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | bash` non-interactive flow on a
  fresh user account (vs. cloning the repo locally and running
  `sh ./install.sh`).
- Upgrade-in-place flow when an existing `~/.gormes/` has a non-default
  `config.toml` and live `auth.json`.
- `GORMES_SKIP_SETUP=0` interactive setup wizard flow (requires a TTY
  and credentials — operator-only, not loop-driven).
