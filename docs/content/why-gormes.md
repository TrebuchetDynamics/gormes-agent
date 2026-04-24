---
title: "Why Gormes"
description: "The Go-native philosophy behind Gormes: operational moat, wire doctor, chaos resilience, and surgical binaries."
weight: 1
---

# Why Gormes

Gormes exists because the bottleneck has moved. Model quality keeps improving; operational friction is what now breaks agent systems in the field. The goal is not to wrap Hermes in another shell. The goal is to move the runtime surfaces that matter most into a Go-native host that is easier to ship, easier to reason about, and harder to kill by accident.

## Operational Moat

The current `cmd/gormes` build fits in a 7.9 MB static binary built with Go 1.22+. That matters because deployment friction is architecture, not cosmetics. A single binary with Zero-dependencies inside the process boundary is easier to copy to a VPS, easier to audit, and easier to recover in the middle of a bad day than a wrapper that drags a Python or Node runtime into every host.

## Wire Doctor

`gormes doctor` exists to catch wiring mistakes before a live turn burns tokens. The Go-native tool registry and schema surface can be validated locally, including the `--offline` path, before the model ever sees a tool definition. That is not a convenience flag. It is a control point that turns schema drift into a local failure instead of a paid production failure.

## Chaos Resilience

Real systems drop streams. Gormes treats that as a first-class architectural problem. Route-B reconnect keeps a turn alive when the SSE stream goes sideways, and the 16 ms coalescing mailbox prevents a stalled renderer from creating a thundering herd of stale frames. The kernel pushes the latest useful state, not every intermediate twitch.

## Surgical Architecture

Gormes is deliberately split into focused binaries. `gormes` stays small and terminal-first. `gormes-telegram` exists as a separate artifact because platform adapters should not bloat the TUI binary or couple unrelated dependency graphs. This is a surgical-strike architecture: clear ownership, smaller binaries, cleaner crash boundaries, and less hidden weight.

## Further Reading

- [Quick Start on GitHub](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/README.md)
- [Executive Roadmap](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/docs/ARCH_PLAN.md)
- [Phase 2.A — Tool Registry](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/docs/superpowers/specs/2026-04-19-gormes-phase2-tools-design.md)
- [Phase 2.B.1 — Telegram Scout](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/docs/superpowers/specs/2026-04-19-gormes-phase2b-telegram.md)
