---
title: "Install"
weight: 20
---

# Install

Gormes is a single static Go binary (~17 MB). Zero CGO, no Python runtime on the host.

## Recommended: source build

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
./bin/gormes --offline
./bin/gormes doctor --offline
./bin/gormes goncho doctor --json
```

This is the primary trust path while Gormes is early-stage: inspect the source
tree, build the binary locally, then verify the offline runtime and Goncho
memory diagnostics before adding provider credentials.

Requires Go 1.25+.

## Convenience: inspectable source installer

```bash
curl -fsSLO https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.sh
less install.sh
sh install.sh
```

The installer manages a source checkout, builds `gormes`, publishes a user-scoped `gormes` command, and can update in place on rerun. Convenience aliases exist at `https://gormes.ai/install.sh` and `https://gormes.ai/install.ps1`, but the inspect-first GitHub source path is preferred until signed binary releases and package-manager manifests land.

The source-backed installers can download a managed Go toolchain when local Go is missing; production release hardening is tracking Homebrew, Scoop/Winget, checksums, and signatures so operators can avoid bootstrap scripts.

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

## Verify

```bash
gormes version
gormes doctor --offline
gormes goncho doctor --json
```
