# Gormes Install — Test Recipes

Reusable recipes for the workflow steps in `SKILL.md`. Copy-paste shaped.

## Sandbox Setup

```sh
TS=$(date -u +%Y%m%dT%H%M%SZ)
SANDBOX=/tmp/gormes-install-test/$TS
mkdir -p "$SANDBOX/home" "$SANDBOX/bin" "$SANDBOX/pre-state" "$SANDBOX/post-state"
echo "sandbox=$SANDBOX"
```

The sandbox lives in `/tmp/` so an inadvertent `--restart-gateway auto`
or stray credential file will not leak into a backed-up directory.

## Pre-State Capture

```sh
{
  echo "# pre-state $TS"
  echo "## ~/.local/bin/gormes"
  ls -la ~/.local/bin/gormes 2>&1 || true
  echo
  echo "## shell rc files"
  for rc in ~/.bashrc ~/.profile ~/.zshrc ~/.zprofile ~/.config/fish/config.fish; do
    [ -f "$rc" ] && grep -nE "GORMES|gormes" "$rc" 2>/dev/null && echo "($rc end)"
  done
  echo
  echo "## systemd unit ground truth (three signals — single negative is not authoritative)"
  echo "### file existence"
  ls -la ~/.config/systemd/user/gormes-gateway.service 2>&1 || true
  echo "### is-enabled"
  systemctl --user is-enabled gormes-gateway 2>&1 || true
  echo "### list-unit-files (no pattern; pattern flag occasionally misses recently-loaded units)"
  systemctl --user list-unit-files 2>&1 | grep -i gormes || true
  echo
  echo "## ~/.gormes config + auth mtimes"
  stat -c '%y %n' ~/.gormes/config.toml ~/.gormes/auth.json 2>&1 || true
  echo
  echo "## crontab gormes lines"
  crontab -l 2>/dev/null | grep -E "gormes-codexu|gormes-opencode" || true
} > "$SANDBOX/pre-state/snapshot.txt"
```

## Dry Run

```sh
GORMES_INSTALL_HOME="$SANDBOX/home" \
GORMES_BIN_DIR="$SANDBOX/bin" \
GORMES_SKIP_SETUP=1 \
GORMES_RESTART_GATEWAY=never \
sh ./install.sh --dry-run > "$SANDBOX/dryrun.log" 2>&1

cat "$SANDBOX/dryrun.log"
```

Expected plan keys:

```
dry run
  branch: main
  source: managed git checkout of main
  checkout: <SANDBOX>/home/gormes-agent
  install_home: <SANDBOX>/home
  managed_binary: <SANDBOX>/home/bin/gormes
  published_binary: <SANDBOX>/bin/gormes
  restart_gateway: never
  setup_wizard: false
```

If any path printed in the dry-run plan falls outside `$SANDBOX/`,
record an isolation gap and reconsider whether to run the real install.

## Real Install (Sandboxed)

```sh
GORMES_INSTALL_HOME="$SANDBOX/home" \
GORMES_BIN_DIR="$SANDBOX/bin" \
GORMES_SKIP_SETUP=1 \
GORMES_RESTART_GATEWAY=never \
GORMES_INSTALL_VERBOSE=1 \
sh ./install.sh --verbose --skip-setup --restart-gateway never \
  > "$SANDBOX/install.log" 2>&1

echo "exit=$?"
test -x "$SANDBOX/bin/gormes" && "$SANDBOX/bin/gormes" version
```

The real install prints `updating active PATH command …` and
`added <path> to PATH in: ~/.bashrc ~/.profile` if the
`iso-bin-hijack` and `iso-shellrc-leak` issues are still open in
`known-issues.md`. Treat the next-step diff as the proof.

## Post-State Capture + Diff

```sh
{
  echo "# post-state $TS"
  echo "## ~/.local/bin/gormes"
  ls -la ~/.local/bin/gormes 2>&1 || true
  echo
  echo "## shell rc files"
  for rc in ~/.bashrc ~/.profile ~/.zshrc ~/.zprofile ~/.config/fish/config.fish; do
    [ -f "$rc" ] && grep -nE "GORMES|gormes" "$rc" 2>/dev/null && echo "($rc end)"
  done
  echo
  echo "## systemd unit ground truth (three signals)"
  echo "### file existence"
  ls -la ~/.config/systemd/user/gormes-gateway.service 2>&1 || true
  echo "### is-enabled"
  systemctl --user is-enabled gormes-gateway 2>&1 || true
  echo "### list-unit-files (no pattern)"
  systemctl --user list-unit-files 2>&1 | grep -i gormes || true
  echo
  echo "## ~/.gormes config + auth mtimes"
  stat -c '%y %n' ~/.gormes/config.toml ~/.gormes/auth.json 2>&1 || true
} > "$SANDBOX/post-state/snapshot.txt"

diff -u "$SANDBOX/pre-state/snapshot.txt" "$SANDBOX/post-state/snapshot.txt" \
  > "$SANDBOX/state-diff.txt" || true
cat "$SANDBOX/state-diff.txt"
```

Any non-empty diff in a sandbox-isolated pass is an issue. Record the
exact diff in `known-issues.md`.

## Functional Smoke

```sh
GBIN="$SANDBOX/bin/gormes"

echo "## version"
"$GBIN" version

echo "## help"
"$GBIN" --help | head -20

echo "## doctor --offline"
"$GBIN" doctor --offline > "$SANDBOX/doctor.log" 2>&1
echo "exit=$?"
tail -20 "$SANDBOX/doctor.log"

echo "## offline TUI help (no actual TTY)"
GORMES_HOME="$SANDBOX/home" "$GBIN" --offline --help | head -10

echo "## config get on fresh sandbox (should fail closed)"
GORMES_HOME="$SANDBOX/home" "$GBIN" config get hermes.api_key 2>&1 || true
```

## Production Restoration After Sandbox Leak

If `iso-bin-hijack` triggered (production `~/.local/bin/gormes` was
hijacked), re-run the installer against production defaults so the
real production install paths are restored:

```sh
GORMES_SKIP_SETUP=1 \
sh ./install.sh --skip-setup --restart-gateway auto
```

This is safe because the operator's `~/.gormes/config.toml` and
`auth.json` are not modified by `install.sh` (only the binary +
managed source checkout are rebuilt).

If `iso-shellrc-leak` triggered (`~/.bashrc`/`~/.profile` got sandbox
PATH lines):

```sh
sed -i.preinstall-bak \
  '/Gormes installer.*\/tmp\/gormes-install-test/d; /export PATH=\"\/tmp\/gormes-install-test/d' \
  ~/.bashrc ~/.profile
```

The `.preinstall-bak` file is the rollback witness; keep it until the
next test pass confirms the strip stuck.

If `~/.gormes/auth.json` mtime changed: **stop, do not auto-restore**.
That file is sensitive. Stash with `cp -p` and inspect the diff before
deciding whether the change is benign (e.g., the installer corrected a
file mode) or hostile.

## Cleanup

```sh
# Optional — keep the sandbox for retrospective analysis or the next pass.
# Run when you are sure you do not want to refer to the captured logs again.
rm -rf "$SANDBOX"
```

## Production Upgrade Pass (Operator-Agreed Only)

When the operator explicitly agrees to upgrade their production install,
the sandbox-flag dance is replaced with the simple form. The operator
must confirm that `~/.gormes/config.toml` and `auth.json` are backed up
or their loss is acceptable (they are not normally touched, but a
broken release could regress that):

```sh
sh ./install.sh --skip-setup --restart-gateway auto
```

Record the exit code and the post-install transcript. If the operator
runs this path, append the outcome under "Mitigated Issues" in
`known-issues.md` if it surfaces a regression that the sandboxed pass
missed.

## Uninstall Pass

```sh
GORMES_INSTALL_HOME="$SANDBOX/home" \
sh ./install.sh --uninstall --dry-run
# review plan, then if happy:
GORMES_INSTALL_HOME="$SANDBOX/home" \
sh ./install.sh --uninstall --yes
```

After uninstall, capture the post-state snapshot again and verify the
sandbox dir, the binary, the systemd unit (if any), the shell rc edits,
and the cron entries are all removed. Anything left behind is an
uninstall completeness gap.

## Branch / Tag Pass

To exercise a non-default branch or a tagged release reference:

```sh
GORMES_BRANCH=v0.1.06 \
GORMES_INSTALL_HOME="$SANDBOX/home" \
GORMES_BIN_DIR="$SANDBOX/bin" \
GORMES_SKIP_SETUP=1 \
GORMES_RESTART_GATEWAY=never \
sh ./install.sh --branch v0.1.06 --dry-run
```

Note: `install.sh` is source-clone-and-build by design and ignores the
release-workflow-published binaries. The published archives are useful
for a separate "binary-fetch" install path (not yet implemented; track
under Phase 5.P).
