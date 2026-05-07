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

### [msg-systemd-fiction] Install reports systemd user service installed even when no service was created

- **First seen**: 2026-05-07T17:27Z (same pass).
- **Surface touched**: install transcript only (no actual systemd state
  modified — verified via `systemctl --user list-unit-files 'gormes*'`
  returning `0 unit files listed`).
- **Trigger**: any successful `install.sh` run.
- **Symptom**: install transcript ends with the block:
  ```
  systemd user service installed:
    systemctl --user start gormes-gateway    # start now
    systemctl --user status gormes-gateway   # check status
    journalctl --user -u gormes-gateway -f   # follow logs
    (auto-starts on login; survives reboots)
  ```
  but `systemctl --user list-unit-files 'gormes*'` reports zero files.
  The message is unconditional and misleads operators into believing a
  service exists; running the suggested `systemctl --user start` fails.
- **Status**: `open`.
- **Routing**: file under Phase 5.P — wrap the systemd block in a
  `command -v systemctl >/dev/null && [ "$XDG_RUNTIME_DIR" ]` (or the
  installer's existing systemd-detection check) before printing, and
  ensure the unit file is actually written when the block prints.

---

## Mitigated Issues

(none yet)

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
