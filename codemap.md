# Gormes repository map

Start with [AGENTS.md](AGENTS.md) for branch, skill-routing, and backlog rules.
This is a navigation map, not a second backlog or a claim of complete Hermes parity.

## Entry points

| Path | Responsibility / source evidence |
| --- | --- |
| [`cmd/gormes/main.go`](cmd/gormes/main.go) | Thin executable entry point: forwards build/version metadata to `internal/app/gormescmd.Main`. |
| [`internal/app/gormescmd/main.go`](internal/app/gormescmd/main.go) | Runtime composition: imports configuration, CLI, kernel, gateways, tools, memory, and persistence adapters. |
| [`internal/platform/cli/gormescli/`](internal/platform/cli/gormescli/) | CLI command surface and supporting command modules. Start with `root.go` and the command-domain file. |
| [`cmd/progress/main.go`](cmd/progress/main.go) | Progress validation, rendering, and `next-work` selectors; delegates to `internal/planning/progress/ctl`. |
| [`cmd/gormes-repo/`](cmd/gormes-repo/) | Repository metadata commands; see [command recipes](cmd/README.md). |

## Runtime navigation

- **Turn execution:** [`internal/kernel/kernel.go`](internal/kernel/kernel.go) owns
  the turn state machine. Its configuration wires an LLM client, tool registry,
  recall, skills, persistence, and telemetry. Tool execution lives alongside it in
  [`internal/kernel/toolexec.go`](internal/kernel/toolexec.go).
- **Providers:** [`internal/llm/`](internal/llm/) contains model transports and
  streaming implementations; [`internal/provider/`](internal/provider/) contains
  provider configuration and routing support.
- **Tools:** [`internal/tools/`](internal/tools/) contains tool implementations,
  registration, and approval/safety code. Preserve safety contracts when changing tools.
- **Chat gateways:** [`internal/gateway/channel.go`](internal/gateway/channel.go)
  defines inbound `Run` and outbound `Send` plus optional channel capabilities.
  [`internal/app/gateway/`](internal/app/gateway/) and channel application packages
  provide command/runtime integration.
- **UI:** [`internal/tui/`](internal/tui/) contains terminal UI components;
  [`internal/platform/cli/gormescli/tuiapp/`](internal/platform/cli/gormescli/tuiapp/)
  connects the CLI surface to the UI.
- **Memory:** [`internal/memory/memory.go`](internal/memory/memory.go) implements
  the SQLite-backed store through a bounded command queue and single SQL worker.
  [`internal/memory/goncho/`](internal/memory/goncho/) contains Goncho integration.
- **Persistence:** [`internal/persistence/`](internal/persistence/) groups session,
  store-contract, and transcript packages.
- **Configuration:** [`internal/config/`](internal/config/) owns configuration
  domains. For MCP persistence, start with [`internal/config/mcpstore/`](internal/config/mcpstore/).
- **Extensions and skills:** [`internal/extensibility/`](internal/extensibility/)
  groups skills, plugins, Pi-extension support, and WASI integration.

A useful source tour is executable entry point → application composition → CLI or
channel adapter → kernel → LLM/tool execution → persistence and render/delivery.
Follow the actual imports for a particular command; do not assume every command
traverses all of these layers.

## Planning, references, and documentation

- [`development-skills/`](development-skills/) is the canonical repo-local skill
  source. `.agents/skills/`, `.claude/skills/`, and `.codex/skills/` are loader views.
- [`internal/planning/progress/`](internal/planning/progress/) owns the typed
  progress model and its loader/saver. Use `cmd/progress` or the typed API, never
  hand-parse backlog JSON. The canonical backlog location and supported layouts
  are documented in [AGENTS.md](AGENTS.md).
- [`webpages/docs/content/building-gormes/`](webpages/docs/content/building-gormes/)
  contains architecture, parity, and development documentation.
- [`hermes-agent/`](hermes-agent/) is the upstream reference submodule, not the Go
  runtime. Inspect its recorded commit and remotes before making parity claims;
  do not update it as an incidental installation step.

## Development gates

Run from the repository root, using a Go version satisfying [go.mod](go.mod):

```sh
go version
go run ./cmd/progress validate
go run ./cmd/progress next-work
go run ./cmd/progress next-work --repo-only
git diff --check
```

Use the selected row's focused tests during implementation. The repository's full
pre-PR gate includes `go test ./... -count=1`; passing a selector is not equivalent
to passing the suite. Work only on `development` and preserve unrelated dirty work.

Update this map when command entry points, package boundaries, or progress tooling
move. Inspect a nested `codemap.md` when one exists; otherwise use source and tests.
