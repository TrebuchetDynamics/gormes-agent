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
- **Status**: `open`.
- **Routing**: file under Phase 5.P — installer should treat
  `GORMES_BIN_DIR` as an authoritative isolation boundary and skip the
  active-PATH-command update when it points outside the sandbox prefix.

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
- **Status**: `open`.
- **Routing**: file under Phase 5.P — installer should not write to shell
  rc files when the resolved bin dir is under `/tmp/`, or when an explicit
  `--no-shell-init` flag is present, or when the sandbox prefix env vars
  are set.

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
