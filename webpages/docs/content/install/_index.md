---
title: "Install Gormes"
description: "Install Gormes on Linux, macOS, or Windows — release-first with a source fallback. One Go binary, no Python or Node runtime."
weight: 5
---

# Install Gormes

Gormes ships as a single static Go binary. The Linux release build measures ~40 MB (recorded in `benchmarks.json`); size varies by OS and build profile. There is no Python runtime, no Node runtime, and no Docker daemon to install. Neither bootstrap installer requires root or admin rights; a non-root install stays fully user-scoped, while a root Linux install uses FHS paths like `/usr/local/bin/gormes`.

The Unix and Windows installers are release-first: they fetch the latest signed release archive for the host platform, then fall back to a managed source build only when a release asset is unavailable, when verification fails, when you ask for it, or when the target branch is not `main`.

## Pick your path

- [Linux and macOS](./linux-macos/) — `install.sh` one-liner, inspect-first variant, customization flags, and `gormes doctor --offline` verification.
- [Windows native](./windows/) — `install.ps1` one-liner, inspect-first variant, PowerShell parameters, and verification.
- [From source](./from-source/) — `git clone` plus `CGO_ENABLED=0 go build -trimpath -o bin/gormes ./cmd/gormes`, and when to prefer source over the installer (advanced, air-gapped, custom build flags).

## Platform support

| Platform | Release binary | Source build | Notes |
|---|---|---|---|
| Linux x86_64 | yes | yes | Primary development target; release archive `gormes-${version}-linux-amd64.tar.gz`. |
| Linux arm64 | yes | yes | Raspberry Pi-class and ARM server target. |
| macOS Apple Silicon | yes | yes | Release archive `gormes-${version}-darwin-arm64.tar.gz`. |
| macOS Intel | yes | yes | Release archive `gormes-${version}-darwin-amd64.tar.gz`. |
| Windows native | yes | yes | `install.ps1` publishes `gormes.exe` under `%LOCALAPPDATA%\gormes\bin`. |
| WSL2 | yes | yes | Treated as Linux; uses `install.sh`. |
| Termux (Android) | yes | yes | Release archive `gormes-${version}-android-arm64.tar.gz`. |

Each tagged release publishes `gormes-${version}-${os}-${arch}.tar.gz` together with a `.sha256` checksum file. The installer downloads both and verifies the archive SHA-256 before extracting the binary.
