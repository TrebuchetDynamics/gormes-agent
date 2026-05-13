---
title: "From source"
description: "Build Gormes from a local Git checkout, or install the latest commit with go install."
weight: 30
aliases:
  - /using-gormes/quickstart/
  - /using-gormes/hardware/
---

# From source

The bootstrap installers (`install.sh`, `install.ps1`) already fall back to a source build when a release binary is unavailable. This page covers building Gormes directly when you want the inspectable, hands-on path.

## When to choose source over the installer

- **Air-gapped or offline networks** where the GitHub Releases API is not reachable.
- **Audit and review** before running any installer — clone, read the tree, build locally, then put the fresh binary first on PATH.
- **Custom build flags or tags** such as `-tags slim` or `-tags gormes_lite` (see the [hardware/build-profile notes](#build-profiles)).
- **Non-main branches or feature work** where release archives are not published.
- **Unsupported platforms** that do not yet have a release asset.

Source builds require Git and Go 1.26+. The installer can fetch a managed Go for you; for a hand-built source clone, install Go from your distribution or [go.dev](https://go.dev/dl/).

## Clone and build

```bash
git clone https://github.com/TrebuchetDynamics/gormes-agent.git
cd gormes-agent
make build
export PATH="$PWD/bin:$PATH"
gormes --version
gormes doctor --offline
```

`make build` produces `bin/gormes` from `./cmd/gormes` using `CGO_ENABLED=0 go build -trimpath` with build-time version metadata. Putting `$PWD/bin` first on PATH while validating local changes prevents an older installed `gormes` from shadowing the fresh build.

To run the binary directly without installing:

```bash
go run ./cmd/gormes --version
go run ./cmd/gormes doctor --offline
```

## go install

If you only need the latest tagged release on the default branch, `go install` works without a manual clone:

```bash
go install github.com/TrebuchetDynamics/gormes-agent/cmd/gormes@latest
```

This drops the binary in `$(go env GOBIN)` (usually `$HOME/go/bin`). Put that directory on PATH if it is not already.

## Build profiles

| Profile | Command | Notes |
|---|---|---|
| Full (default) | `make build` | All standard tools and helpers compiled in. Linux release build is ~40 MB. |
| Slim | `make build-slim` | `go build -tags slim`. Excludes TTS, transcription, voice mode, and image generation helpers at compile time. Smaller binary. |
| Lite | `go build -tags gormes_lite ./cmd/gormes` | Omits audio/image helpers from the default tool registry; intended for constrained hosts. |

## Pre-compiled release archives

Tagged releases publish a per-platform archive plus checksum sidecar to GitHub Releases:

- `gormes-${version}-${os}-${arch}.tar.gz`
- `gormes-${version}-${os}-${arch}.tar.gz.sha256`

Supported `${os}-${arch}` slugs: `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`, `windows-amd64`, `windows-arm64`, `android-arm64`.

The bootstrap installers download both files and verify the archive SHA-256 before extracting `gormes` (or `gormes.exe`). If you prefer not to run the installer, download and verify the archive manually, extract it, and place the binary on PATH yourself.

## Verify

After any source or release install:

```bash
gormes --version
gormes doctor --offline
```

`gormes doctor --offline` runs the local runtime, TUI, gateway, and memory diagnostics without any network call, so it is the right check to use before adding provider credentials.
