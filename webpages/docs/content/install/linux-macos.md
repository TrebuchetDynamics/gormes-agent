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
- runs the post-install setup wizard when stdin is a terminal.

## Termux on Android

Termux is supported as a no-root Android arm64/aarch64 runtime for PC-like Gormes operator workflows. Use the same release-first installer command as Linux and macOS:

```bash
curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh
```

On Termux, repo-root `install.sh` detects `TERMUX_VERSION` or the standard Termux `$PREFIX` path and publishes the command to `$PREFIX/bin/gormes`. The release asset is `android-arm64`, not `linux-arm64`; source build is a fallback for unsupported architectures, unavailable release assets, non-main branches, or contributor workflows.

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
gormes --oneshot "hello from Termux"
```

For long gateway sessions, run Gormes inside `tmux`. `termux-wake-lock` and Android battery-optimization settings can improve uptime, but Android can still stop background processes. Local CLI/TUI, provider calls, SQLite/Goncho, and foreground gateway work are in scope; Docker, GPU/local LLM inference, heavy browser automation, and large builds should run on a remote machine and be controlled from Termux over SSH.

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
| `--skip-setup` | Skip the post-install setup wizard. Equivalent env var: `GORMES_SKIP_SETUP=1`. |
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

# Skip the post-install setup wizard
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

To configure providers and channel credentials, run `gormes setup` once the offline doctor passes.
