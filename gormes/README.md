# Gormes

Gormes is the operational moat strategy for Hermes: a Go-native agent host for the era where runtime quality matters more than demo quality. The current `cmd/gormes` build fits in a 7.9 MB static binary built with Go 1.22+, with Zero-dependencies inside the process boundary: no Python runtime, no Node runtime, and no per-host dependency stack once the binary is built.

Phase 1 is a tactical bridge, not the final shape. Today Gormes renders a Bubble Tea dashboard and talks to Python's OpenAI-compatible `api_server` on port 8642. That bridge exists to give immediate value to existing Hermes users while the long-term target stays fixed: a pure Go runtime that owns the full agent lifecycle.

## Quick Start

Start the existing Hermes backend:

```bash
API_SERVER_ENABLED=true hermes gateway start
```

Build the Go binary:

```bash
cd gormes
make build
```

Validate the local tool wiring before spending a cent on API traffic:

```bash
./bin/gormes doctor --offline
```

Run Gormes:

```bash
./bin/gormes
```

## Architectural Edge

- **Wire Doctor** — `gormes doctor` validates the local Go-native tool registry and schema shape before a live turn burns tokens.
- **Route-B reconnect** — dropped SSE streams are treated as a resilience problem to solve, not a happy-path omission to ignore.
- **16 ms coalescing mailbox** — the kernel uses a replace-latest render mailbox so stalled consumers do not trigger a thundering herd of stale frames.

## Further Reading

- [Why Gormes](docs/content/why-gormes.md)
- [Executive Roadmap](docs/ARCH_PLAN.md)
- [Phase 2.A — Tool Registry](docs/superpowers/specs/2026-04-19-gormes-phase2-tools-design.md)
- [Phase 2.B.1 — Telegram Scout](docs/superpowers/specs/2026-04-19-gormes-phase2b-telegram.md)

## License

MIT — see `../LICENSE`.
