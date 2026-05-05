---
title: "Install"
weight: 20
---

# Install

Gormes is a single static Go binary. The current Linux benchmark build is ~38 MB; size varies by OS and build profile. Zero CGO, no Python runtime on the host.

## Method 1: source build

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
export PATH="$PWD/bin:$PATH"
gormes doctor --offline
gormes --offline
gormes goncho doctor --json
```

This is the primary trust path while Gormes is early-stage: inspect the source
tree, build the command locally, put the fresh build first on `PATH`, then
verify the offline runtime and Goncho memory diagnostics before adding provider
credentials.

Requires Go 1.25+.

## Method 2: install.sh

```bash
curl -fsSLO https://gormes.ai/install.sh
less install.sh
sh install.sh
gormes doctor --offline
```

The Unix installer mirrors Hermes' source-backed user flow for Gormes. It
clones or updates a managed checkout, builds `gormes`, publishes a stable
command, verifies `gormes version`, runs `gormes doctor --offline`, and starts
`gormes setup` when a terminal is available. Rerun the same command to update
the managed checkout and rebuild the command.

If you intentionally want the one-line convenience form:

```bash
curl -fsSL https://gormes.ai/install.sh | bash
```

Use `sh install.sh --skip-setup` or set `GORMES_SKIP_SETUP=1` when you want to
install first and run `gormes setup` later.

The installer can download a managed Go toolchain when local Go is missing;
production release hardening is tracking Homebrew, Scoop/Winget, and signatures
so operators can avoid bootstrap scripts.

## Windows PowerShell

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 -OutFile install.ps1
Get-Content .\install.ps1
powershell -ExecutionPolicy Bypass -File .\install.ps1
```

## Go install directly

```bash
go install github.com/TrebuchetDynamics/gormes-agent/cmd/gormes@latest
```

## Pre-compiled release artifacts

Tagged releases build `gormes-${version}-${os}-${arch}.tar.gz` archives for
Linux, macOS, and Windows on amd64 and arm64, each with a `.sha256` checksum.
The Unix bootstrap installer intentionally remains source-backed for Hermes
install-experience parity; release artifacts are for package managers,
air-gapped mirrors, and manual verification paths.

## Platform matrix

| Platform | Status |
|---|---|
| Linux x86_64 | ✅ tested |
| Linux arm64 | ✅ tested |
| macOS arm64 (Apple Silicon) | ✅ tested |
| macOS Intel | 🟡 should work, not regression-tested |
| Windows (native) | 🟡 source-backed PowerShell installer, unsigned binary |
| Windows WSL2 | ✅ tested |
| Termux (Android) | ✅ tested |

See the [hardware matrix](../hardware/) for release-by-release binary size,
RSS, build profile, and device evidence.

## Verify

```bash
gormes version
gormes doctor --offline
gormes goncho doctor --json
```
