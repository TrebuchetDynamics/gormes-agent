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
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | sh
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

## Inspect first

If you would rather read the script before executing it:

```bash
curl -fsSLO https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh
less install.sh
sh install.sh
```

The script is the same content served from the repository's `install.sh` at the `main` branch. The convenience one-liner and the inspect-first form produce the same install.

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
