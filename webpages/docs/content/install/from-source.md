---
title: "From source"
description: "Build Gormes from a local Git checkout."
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
mkdir -p bin
CGO_ENABLED=0 go build -trimpath -o bin/gormes ./cmd/gormes
./bin/gormes --version
./bin/gormes doctor --offline
```

The command produces `bin/gormes` from `./cmd/gormes` without CGO or local path metadata. Running `./bin/gormes` while validating local changes prevents an older installed `gormes` from shadowing the fresh build.

To use that checkout build as `gormes` from the current shell:

```bash
export PATH="$PWD/bin:$PATH"
gormes --version
```

To run the binary directly without installing:

```bash
go run ./cmd/gormes --version
go run ./cmd/gormes doctor --offline
```

## Build profiles

| Profile | Command | Notes |
|---|---|---|
| Full (default) | `mkdir -p bin && CGO_ENABLED=0 go build -trimpath -o bin/gormes ./cmd/gormes` | All standard tools and helpers compiled in. Linux release build is ~40 MB. |
| Slim | `mkdir -p bin && CGO_ENABLED=0 go build -trimpath -tags slim -o bin/gormes-slim ./cmd/gormes` | Excludes TTS, transcription, voice mode, and image generation helpers at compile time. Smaller binary. |
| Lite | `mkdir -p bin && CGO_ENABLED=0 go build -trimpath -tags gormes_lite -o bin/gormes-lite ./cmd/gormes` | Omits audio/image helpers from the default tool registry; intended for constrained hosts. |

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
