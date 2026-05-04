---
title: "Hardware Matrix"
weight: 25
---

# Hardware Matrix

Gormes should earn the "runs anywhere" claim with measured release evidence,
not aspirational copy. Record each tested target here when a release candidate
is built and smoke-tested.

## Release Measurements

| Target | Status | Build | Smoke Gate | Binary Size | Idle RSS | Notes |
|---|---|---|---|---:|---:|---|
| Linux x86_64 | tested | full | `gormes doctor --offline` | ~34 MB | pending | Primary development path. |
| Linux arm64 | tested | full | `gormes doctor --offline` | pending | pending | Raspberry Pi-class and ARM server target. |
| macOS arm64 | tested | full | `gormes doctor --offline` | pending | pending | Apple Silicon target. |
| macOS amd64 | candidate | full | `gormes doctor --offline` | pending | pending | Cross-compiled release artifact. |
| Windows amd64 | candidate | full | `gormes doctor --offline` | pending | pending | Native PowerShell installer exists; signed release still pending. |
| Windows arm64 | candidate | full | `gormes doctor --offline` | pending | pending | Cross-compiled release artifact. |
| WSL2 | tested | full | `gormes doctor --offline` | ~34 MB | pending | Treat as Linux x86_64/arm64 depending on host. |
| Termux / Android | experimental | source | `gormes doctor --offline` | pending | pending | Source-backed until mobile release packaging is proven. |
| Raspberry Pi 4/5 | planned | lite | `gormes doctor --offline` | pending | pending | Track boot time and gateway RSS. |
| Low-memory Linux VPS | planned | lite | `gormes gateway status` | pending | pending | Track persistent gateway RSS. |

## Build Profiles

| Profile | Command | Intended Use |
|---|---|---|
| full | `go build ./cmd/gormes` | Default parity/productivity build with all standard tools. |
| lite | `go build -tags gormes_lite ./cmd/gormes` | Constrained build that omits audio/image helpers from the default registry. |
| slim | `go build -tags slim ./cmd/gormes` | Experimental smallest binary profile; audio, transcription, voice, and image helpers are excluded at compile time. |
| gateway | `gormes gateway` | Persistent chat runtime using the installed binary. |
| dashboard | `gormes dashboard` | Local launcher/control plane; opens the browser unless `--no-open` is set. |

## Release Gate

Before a release row can claim a target as tested, record:

- exact artifact name and version;
- `gormes version`;
- `gormes doctor --offline`;
- binary size from `ls -lh`;
- idle RSS after 60 seconds;
- gateway RSS after 60 seconds when the target is a persistent gateway host.
