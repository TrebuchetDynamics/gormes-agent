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
- `curl https://gormes.ai/install.sh | bash` non-interactive flow on a
  fresh user account (vs. cloning the repo locally and running
  `sh ./install.sh`).
- Upgrade-in-place flow when an existing `~/.gormes/` has a non-default
  `config.toml` and live `auth.json`.
- `GORMES_SKIP_SETUP=0` interactive setup wizard flow (requires a TTY
  and credentials — operator-only, not loop-driven).
