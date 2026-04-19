# Gormes

Go frontend adapter for [Hermes Agent](../README.md). Phase 1 of a 5-phase Ship-of-Theseus port.

Gormes renders a Bubble Tea Dashboard TUI and talks to Python's OpenAI-compatible `api_server` on port 8642. Python owns the agent loop, LLM routing, memory, and `state.db`. Gormes owns rendering, input, and the deterministic-kernel state machine.

## Install

```bash
cd gormes
make build
./bin/gormes
```

Requires Go 1.22+ and a running Python `api_server`:

```bash
API_SERVER_ENABLED=true hermes gateway start
```

## Architecture

See [`docs/ARCH_PLAN.md`](docs/ARCH_PLAN.md) for the 5-phase roadmap and the Ship-of-Theseus strategy.

## License

MIT — see `../LICENSE`.
