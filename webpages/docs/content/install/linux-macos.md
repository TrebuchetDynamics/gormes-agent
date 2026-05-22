---
title: "Linux and macOS"
description: "Install Gormes on Linux, macOS, or WSL2 with the release-first install.sh bootstrap."
weight: 10
aliases:
  - /getting-started/installation/
  - /using-gormes/install/
---

# Linux and macOS

`install.sh` is the supported install path for Linux, macOS, and WSL2. It is release-first: by default it downloads the latest signed release archive for the host platform, verifies its SHA-256, and publishes `gormes` to a user-scoped bin directory. It falls back to a managed source build only when needed.

## 60-second install

```bash
curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh
```

After the installer finishes:

```bash
gormes --version
gormes doctor --offline
```

`gormes doctor --offline` validates the local runtime, TUI, gateway, and memory layout without making any network calls.

By default the installer:

- publishes `gormes` to `$HOME/.local/bin/gormes` for non-root installs, or `/usr/local/bin/gormes` for root installs on Linux;
- keeps managed state under `$HOME/.gormes`, including a managed source checkout at `$HOME/.gormes/gormes-agent` when a source build is used;
- records install events to `$HOME/.gormes/install.log.jsonl`;
- prints a Navivox-first setup menu instead of forcing terminal setup.

The default post-install path is:

1. `gormes navivox pair` — recommended. Pair the Android app and continue setup there.
2. `gormes setup` — continue fully in the terminal.

## Termux on Android

Termux is supported as a no-root Android arm64/aarch64 runtime for PC-like Gormes operator workflows. Use the same release-first installer command as Linux and macOS:

```bash
curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh
```

On Termux, repo-root `install.sh` detects `TERMUX_VERSION` or the standard Termux `$PREFIX` path and publishes the command to `$PREFIX/bin/gormes`. The release asset is `android-arm64`, not `linux-arm64`; source build is a fallback for unsupported architectures, unavailable release assets, non-main branches, or contributor workflows.

> **Termux/Android caveat:** The live `v0.2.20` latest-release installer can still fail publish verification on affected Termux devices, then report `unknown command /data/data/com.termux/files/usr/bin/gormes for gormes`. The fix is committed on `development`, but Termux latest install should not be called repaired until a follow-up release is published.

Only source fallback or contributor builds need the build toolchain:

```bash
pkg update
pkg install git golang clang tmux openssh curl jq sqlite
```

The rule for this project is explicit: install.sh stays in the repository root as the canonical Unix installer. Do not use a separate Unix installer mirror for Termux.

After install:

```bash
gormes version
gormes doctor --offline --json
gormes config check
```

If provider credentials are configured, run a one-turn smoke:

```bash
gormes chat -q "hello from Termux"
```

For long gateway sessions, use the foreground/tmux model:

```bash
tmux new -s gormes-gateway
termux-wake-lock  # optional best-effort aid
gormes gateway
```

Use `gormes gateway status` and `gormes gateway stop` from another Termux shell to inspect or stop the runtime. Android battery-optimization settings and `termux-wake-lock` can improve uptime, but Android can still stop background processes. Termux:Boot can launch your own tmux or foreground wrapper after reboot; Gormes does not install or manage Android services automatically. Local CLI/TUI, provider calls, SQLite/Goncho, and foreground gateway work are in scope; Docker, GPU/local LLM inference, heavy browser automation, and large builds should run on a remote machine and be controlled from Termux over SSH.

### Termux as controller, remote host as executor

For heavier work, the phone is the Gormes controller and the remote host is the heavy executor. Keep the Android side responsible for setup, short CLI/TUI turns, gateway control, notes, and operator review. Put remote browser automation, GPU/local model inference, Docker builds, and large `go test ./...` runs on a workstation or server reached over SSH.

Set a shell shortcut for the host you use most:

```bash
export GORMES_REMOTE_HOST=workstation
ssh workstation 'gormes doctor --offline'
```

Use tmux on the remote machine for long-running builds and browser sessions:

```bash
ssh -t "$GORMES_REMOTE_HOST" 'tmux new -A -s gormes-build'
cd ~/code/gormes-agent
go test ./...
```

Run one-off remote agent turns from Termux with the existing scripted-chat surface:

```bash
ssh "$GORMES_REMOTE_HOST" 'cd ~/code/gormes-agent && gormes chat -q "summarize the current failing tests"'
```

Run persistent remote gateways explicitly on the remote host, not as hidden Android services:

```bash
ssh -t "$GORMES_REMOTE_HOST" 'tmux new -A -s gormes-gateway "gormes gateway"'
```

The rule is simple: do not add a new top-level `gormes run` command for this workflow. Use normal shell, SSH, tmux, `gormes chat -q`, and `gormes gateway` surfaces so the same commands remain inspectable on Termux, Linux, macOS, and WSL2. If no remote host is configured, Termux remains a local CLI/TUI/gateway runtime with the heavy-workload boundaries above.

## Inspect first

If you would rather read the script before executing it:

```bash
curl -fsSLO https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh
less install.sh
sh install.sh
```

The script attached to the latest GitHub Release is the canonical public installer for the current release line. The convenience one-liner and the inspect-first form produce the same install.

## Customize

`install.sh` accepts flags and environment variables for common operator needs.

| Flag | Purpose |
|---|---|
| `--build` (alias `--from-source`) | Build from source instead of fetching the release binary. Slower; needed for unsupported platforms or non-main branches. Equivalent env var: `GORMES_INSTALL_FROM_SOURCE=1`. |
| `--local` | Build from the current working directory's checkout. Use when developing against a local clone. |
| `--dry-run` | Print the resolved install plan (method, source, install home, published binary path) and exit without changing the machine. |
| `--skip-setup` | Skip the post-install setup recommendation. Equivalent env var: `GORMES_SKIP_SETUP=1`. |
| `--uninstall [args]` | Delegate to `gormes uninstall` to remove Gormes. Flags after `--uninstall` pass through, for example `install.sh --uninstall --dry-run`. |
| `--branch NAME` | Target a non-default branch. Triggers a source build because release binaries are only published from `main`. |
| `--home DIR` | Override the managed install home (default: `$HOME/.gormes`). |
| `--bin-dir DIR` | Override the published command directory. |
| `--restart-gateway auto\|always\|never` | Control whether the installer restarts a live gateway after upgrade. |
| `-v`, `--verbose` | Print resolved paths, platform details, and step diagnostics. |

Equivalent install variants:

```bash
# Preview the plan, do not write anything to disk
sh install.sh --dry-run

# Skip the post-install setup recommendation
sh install.sh --skip-setup

# Force a source build even if a release binary is available
sh install.sh --build

# Build from the current source checkout
cd ~/code/gormes-agent
sh install.sh --local

# Use environment variables instead of flags
GORMES_INSTALL_FROM_SOURCE=1 sh install.sh
GORMES_SKIP_SETUP=1 sh install.sh

# Remove Gormes (review the dry-run plan first)
sh install.sh --uninstall --dry-run
sh install.sh --uninstall --yes
```

When the installer falls back to a source build it ensures Git and a supported Go toolchain (1.26+) are available, downloading a managed Go to `$HOME/.gormes/go` if no suitable system Go is found.

## Verify

```bash
gormes --version
gormes doctor --offline
```

If `gormes` is not found after install, open a new shell so the updated PATH is picked up, or add the installer's published bin directory to PATH manually (default: `$HOME/.local/bin`).

Recommended setup path after install is `gormes navivox pair` so the Android app can continue configuration. To configure providers and channel credentials entirely in the terminal, run `gormes setup` once the offline doctor passes.
