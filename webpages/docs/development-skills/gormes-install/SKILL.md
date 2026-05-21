---
name: gormes-install
description: Use when verifying Gormes install or setup paths, exercising install.sh or install.ps1, installing the current development checkout with --local, refreshing the live gateway binary, finding install-time issues that do not surface in CI, documenting install regressions, or proving a fresh user can reach a working gormes command without disturbing existing state.
---

# Gormes Install

## Mission

Exercise the operator install + setup paths against a real Gormes release and surface any issues that pure CI cannot catch: PATH leaks, shell-rc edits, symlink hijacking, misleading status messages, hidden side effects on existing installs, and the gap between the script's promised plan and what actually lands on the host.

`go test` validates code. `go run ./cmd/progress validate` validates plans. **This skill validates the operator's first ten minutes** — the part of the product where a stranger decides whether to keep using Gormes.

The output is one of:
- A clean install transcript with confirmed sandbox isolation (release is install-clean).
- A clean development-workstation install from the existing `development` checkout, with binary hashes and gateway freshness verified.
- A list of source-backed install regressions, each with a reproducible recipe and the exact production-state surface it touched.
- New rows added to `progress.json` Phase 5.P (installer) or Phase 5.O (CLI) when an issue needs builder follow-up.

## When To Use

| Trigger | Pass type |
|---|---|
| New release tagged | Full pre-publish install rehearsal in isolation. |
| `install.sh` or `install.ps1` changed on `development` | Pre-merge install regression check. |
| User asks to install/build Gormes on this PC from `development` | Development workstation install from the current checkout. |
| User says "install gormes bin in this pc", "build and install development", or "refresh my live gormes" | Production workstation install with gateway restart/freshness verification. |
| User reports broken install / `gormes` command not found / shell rc clutter | Targeted regression repro, then route to `gormes-builder`. |
| Sandbox-isolation env vars added or changed (`GORMES_INSTALL_HOME`, `GORMES_BIN_DIR`, etc.) | Verify each isolation surface. |
| New target environment (Termux, WSL, locked-down corp Linux, fresh container) | Coverage extension; capture environment-specific findings. |
| New rows in Phase 5.P or 5.O | Downstream verification once the row lands. |

Do **not** use this skill for runtime feature implementation. If install reveals a bug, route the fix through `gormes-builder` and `gormes-tdd-slice`. This skill is for finding and documenting, not patching.

## Repository Branch Rule

Stay on the existing `development` branch. Do not create release branches, feature branches, or worktrees. If install testing surfaces a fix, the fix lands through a normal `gormes-git` push to `development` plus a future PR; never edit `main` directly.

## Hard Constraints

These are non-negotiable because the test machine is usually the operator's real workstation:

1. **Never run `install.sh` directly against production state without explicit operator agreement**. Default to a sandbox HOME under `/tmp/gormes-install-test/<UTC-stamp>/` and route `GORMES_INSTALL_HOME` + `GORMES_BIN_DIR` there. Exception: if the user explicitly asks to install/build Gormes on this PC from `development`, use the development workstation path below and report the production surfaces touched.
2. **Capture pre-state before every test pass** so post-state diffs prove what install actually changed. Pre-state must include:
   - `~/.local/bin/gormes` symlink target (or absence) and inode.
   - `~/.bashrc`, `~/.profile`, `~/.zshrc`, `~/.zprofile`, `~/.config/fish/config.fish` last modified time + grep for `GORMES`/`gormes` lines.
   - **systemd unit ground truth via THREE independent signals** (filing
     a "no service installed" issue from a single negative `systemctl`
     query has produced a false positive in the past — see
     `references/known-issues.md` Withdrawn section):
     - `ls -la ~/.config/systemd/user/gormes-gateway.service` (file
       existence is ground truth);
     - `systemctl --user is-enabled gormes-gateway` (manager-loaded state);
     - `systemctl --user list-unit-files` without a glob pattern.
   - `~/.gormes/config.toml`, `~/.gormes/auth.json` mtimes (must NOT change in a sandbox pass).
   - `crontab -l` lines containing `gormes-codexu-builder-loop` or similar.
3. **Restore production state after every sandbox pass** if the install touched any of the above. Restoration recipes live in `references/test-recipes.md`.
4. **Never commit credentials, captured logs containing tokens, or `~/.gormes/auth.json` snippets** to the repo. The sandbox lives in `/tmp/` deliberately.
5. **The autonomous builder loop** (`~/.local/bin/gormes-codexu-builder-loop`) **is a separate concern from the gormes binary**. Pause it before destructive testing if there's any chance of a race; a sandbox install should never SIGTERM, restart, or touch the loop's gateway.

## Workflow

### 1. Bound The Pass

State which install path you are testing:
- fresh-user `curl … | bash` non-interactive flow;
- existing-user upgrade in place against production paths;
- development workstation install from the existing `development` checkout;
- `--local` mode (build from current checkout instead of cloning);
- `--dry-run` plan accuracy vs. real-install outcome;
- `--uninstall` flow;
- `--branch <branch>` non-default branch flow;
- environment-specific (Termux, WSL2, root install, no-systemd minimal Linux, locked-down filesystem).

If the user asks "test the install" without scoping, default to the **fresh-user sandboxed flow** and surface coverage gaps in the final report.

### 1A. Development Workstation Install From `development`

Use this path only when the operator explicitly asks to install on this PC from
`development`. This is a production-state install, not a sandbox pass.

Success criteria:
- current branch is `development`;
- no feature branch, release branch, git worktree, stash, reset, or checkout of
  `main`;
- install builds from the current checkout with `--local`;
- installed command, managed binary, and repo-local `bin/gormes` have the same
  sha256;
- default gateway and any running profile-scoped gateway units report `fresh`
  after restart.

Preflight:

```sh
git branch --show-current
git status --short --branch
which -a gormes || true
if command -v gormes >/dev/null 2>&1; then
  readlink -f "$(command -v gormes)" || true
  gormes --version || true
  gormes gateway status --json 2>/dev/null || true
fi
```

If the branch is not `development`, stop before installing. Switch to
`development` only when it is safe and consistent with the repo rule; otherwise
report the blocker. If the user asks for exact `origin/development`, pull only
with a clean worktree and a fast-forward:

```sh
git fetch origin development
git pull --ff-only origin development
```

Do not stash, reset, rebase, or discard user/agent edits to make the pull work.
When the tree is dirty and the user still asks to install "this PC", build the
current dirty checkout and report that fact.

Install and align binary surfaces:

```sh
./install.sh --local --skip-setup --restart-gateway auto

version=$(sed -n 's/^var Version = "\(.*\)"/\1/p' cmd/gormes/version.go)
commit=$(git rev-parse --short HEAD)
dirty=false
if [ -n "$(git status --porcelain)" ]; then dirty=true; fi
CGO_ENABLED=0 go build -trimpath \
  -ldflags "-s -w -X main.Version=${version} -X main.GitCommit=${commit} -X main.GitDirty=${dirty}" \
  -o bin/gormes ./cmd/gormes
```

Verify:

```sh
which -a gormes
readlink -f "$(command -v gormes)"
sha256sum "$(command -v gormes)" "$HOME/.gormes/bin/gormes" bin/gormes
gormes --version
./bin/gormes --version
gormes gateway status --json | jq -r '.build.version, .build.git_commit, .runtime.gateway_state, .runtime.stale_code.status, (.runtime.platforms.telegram.state // "telegram-missing")'

profile_root="$HOME/.gormes/profiles"
if [ -d "$profile_root" ]; then
  find "$profile_root" -mindepth 1 -maxdepth 1 -type d -print | sort |
    while IFS= read -r profile_home; do
      profile_id=$(basename "$profile_home")
      printf 'profile=%s\n' "$profile_id"
      GORMES_HOME="$profile_home" \
        gormes gateway status --json |
        jq -r '.build.version, .build.git_commit, .runtime.gateway_state, .runtime.stale_code.status, (.runtime.platforms.telegram.state // "telegram-missing")'
    done
fi
```

If `stale_code.status` is `stale` because `development` moved during the pass,
rerun the local install from the new HEAD. If any sha256 differs, rebuild the
mismatched surface before reporting success.

### 2. Capture Pre-State

```sh
TS=$(date -u +%Y%m%dT%H%M%SZ)
SANDBOX=/tmp/gormes-install-test/$TS
mkdir -p "$SANDBOX/home" "$SANDBOX/bin" "$SANDBOX/pre-state"

ls -la ~/.local/bin/gormes 2>/dev/null > "$SANDBOX/pre-state/local-bin-gormes.txt" || true
grep -n -E "GORMES|gormes" ~/.bashrc ~/.profile ~/.zshrc 2>/dev/null > "$SANDBOX/pre-state/shell-rc.txt" || true
systemctl --user list-unit-files 'gormes*' 2>/dev/null > "$SANDBOX/pre-state/systemd.txt" || true
stat -c '%y %n' ~/.gormes/config.toml ~/.gormes/auth.json 2>/dev/null > "$SANDBOX/pre-state/gormes-home.txt" || true
crontab -l 2>/dev/null | grep -E "gormes-codexu|gormes-opencode" > "$SANDBOX/pre-state/cron.txt" || true
```

### 3. Run The Plan First (Dry Run)

Always start with `--dry-run` against the sandbox and read the resolved plan. The plan tells you exactly which paths the install intends to touch.

```sh
GORMES_INSTALL_HOME="$SANDBOX/home" \
GORMES_BIN_DIR="$SANDBOX/bin" \
GORMES_SKIP_SETUP=1 \
GORMES_RESTART_GATEWAY=never \
sh ./install.sh --dry-run > "$SANDBOX/dryrun.log" 2>&1
```

Verify the plan stays inside `$SANDBOX/`. If the plan mentions `~/.local/bin/gormes`, `~/.bashrc`, `/etc/`, `systemctl`, or `~/.gormes/` outside the sandbox, that is an isolation gap — record it before proceeding.

### 4. Run The Real Install With Verbose Diagnostics

```sh
GORMES_INSTALL_HOME="$SANDBOX/home" \
GORMES_BIN_DIR="$SANDBOX/bin" \
GORMES_SKIP_SETUP=1 \
GORMES_RESTART_GATEWAY=never \
GORMES_INSTALL_VERBOSE=1 \
sh ./install.sh --verbose --skip-setup --restart-gateway never \
  > "$SANDBOX/install.log" 2>&1
```

Always pass `--skip-setup` and `--restart-gateway never` for sandbox passes. The setup wizard prompts for credentials; restart-gateway interferes with any production gateway.

### 5. Verify Post-State And Diff Against Pre-State

```sh
"$SANDBOX/bin/gormes" version
"$SANDBOX/bin/gormes" doctor --offline > "$SANDBOX/doctor.log" 2>&1

ls -la ~/.local/bin/gormes > "$SANDBOX/post-state/local-bin-gormes.txt"
grep -n -E "GORMES|gormes" ~/.bashrc ~/.profile ~/.zshrc 2>/dev/null > "$SANDBOX/post-state/shell-rc.txt" || true
systemctl --user list-unit-files 'gormes*' 2>/dev/null > "$SANDBOX/post-state/systemd.txt" || true
stat -c '%y %n' ~/.gormes/config.toml ~/.gormes/auth.json 2>/dev/null > "$SANDBOX/post-state/gormes-home.txt" || true

diff "$SANDBOX/pre-state/local-bin-gormes.txt" "$SANDBOX/post-state/local-bin-gormes.txt" || true
diff "$SANDBOX/pre-state/shell-rc.txt"         "$SANDBOX/post-state/shell-rc.txt"         || true
diff "$SANDBOX/pre-state/systemd.txt"          "$SANDBOX/post-state/systemd.txt"          || true
diff "$SANDBOX/pre-state/gormes-home.txt"      "$SANDBOX/post-state/gormes-home.txt"      || true
```

Any non-empty diff for a sandbox-isolated pass is an issue.

### 6. Functional Smoke

After the binary lands, run the operator's first commands and capture exit codes + last 20 lines of output for each:

```sh
"$SANDBOX/bin/gormes" version
"$SANDBOX/bin/gormes" --help
"$SANDBOX/bin/gormes" doctor --offline
"$SANDBOX/bin/gormes" config get hermes.api_key 2>&1 || true   # should fail closed in fresh sandbox
GORMES_HOME="$SANDBOX/home" "$SANDBOX/bin/gormes" --offline --help
```

### 7. Restore Production If Anything Leaked

When the sandbox pass touched production state (see `references/test-recipes.md` for restoration recipes):

- If `~/.local/bin/gormes` was hijacked, re-run `install.sh` against production defaults to restore the proper symlink.
- If `~/.bashrc`/`~/.profile` were edited, `sed -i.preinstall-bak '/Gormes installer.*\/tmp\/gormes-install-test/d; /export PATH=\"\/tmp\/gormes-install-test/d' ~/.bashrc ~/.profile`.
- If `~/.gormes/auth.json` mtime changed, **stop and report** — that file is sensitive and should never be modified by a sandbox pass.

### 8. Document Findings

For each issue, capture:
- Surface touched (file path or system seam, with absolute path).
- Pre-state vs post-state diff.
- The install.sh log line(s) that announced the change.
- The env vars and flags that were supposed to prevent it.
- Reproducibility (timestamped sandbox dir is the witness).

Append confirmed issues to `references/known-issues.md` in this skill. Each entry gets a stable header, a one-line summary, the recipe, and a status (`open`, `mitigated`, `fixed-in-vX.X.XX`).

### 9. Route Fixes

This skill stops at find + document. Route fixes via:
- `gormes-builder` for the implementation slice.
- `gormes-planner` to add a row in `progress.json` Phase 5.P (installer) or 5.O (CLI surfaces) when no row exists.
- `gormes-tdd-slice` for fixes that need test coverage (most install issues do — they're behavioral).

## Validation

After updating this skill or its references:

```sh
python3 /home/xel/.codex/skills/.system/skill-creator/scripts/quick_validate.py docs/development-skills/gormes-install
find -L .agents/skills .claude/skills .codex/skills -maxdepth 2 -name SKILL.md -print | sort | grep gormes-install
go run ./cmd/progress validate
git diff --check
```

If a real install pass landed test artifacts under `/tmp/gormes-install-test/`, scrub them before committing — they may contain machine paths, tokens (if `--skip-setup` was forgotten), or environment fingerprints not safe to publish.

## Final Report

Report:

1. Pass type (fresh sandbox, prod upgrade, --local, --dry-run, --uninstall, environment-specific).
2. Sandbox path (or "production" with operator agreement noted).
3. Resolved plan from `--dry-run`.
4. Install exit status + last 30 lines of `install.log`.
5. Pre-state vs post-state diffs (one block per surface).
6. Functional smoke results (one line per command + exit code).
7. New issues found, with the entry now in `references/known-issues.md`.
8. Production restoration steps run, if any.
9. Coverage gaps (environments / flags / paths not exercised this pass).

For development-workstation installs, report instead:

1. Branch and commit installed, including dirty-tree status.
2. `which -a gormes` and active command realpath.
3. sha256 comparison for active command, `$HOME/.gormes/bin/gormes`, and `bin/gormes`.
4. Default gateway freshness and each profile-scoped gateway freshness.
5. Any profile gateway units restarted by the installer.
6. Verification commands run and their pass/fail result.
