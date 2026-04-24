# Gormes Phase 5 Finish Sprint

> Execution: strict TDD, one subsystem per commit, no partial migrations without rollback path.

**Goal:** Complete Python purge by porting remaining tool/runtime surfaces to Go/WASM and finishing packaging/integration layers.

## Verified baseline
- Phase 5 status: **0/28 complete**.
- Remaining: all Phase 5 items (`5.A`..`5.Q`).

## Definition of Done
1. All Phase 5 items marked `complete`.
2. Python-dependent runtime paths removed or fully optional.
3. `go test ./... -count=1` and packaging checks pass.
4. Docs reflect pure-Go operational baseline.

## Slice P5-A — Tool and execution core
Targets:
- `5.A` tool surface port
- `5.K` sandboxed code execution
- `5.L` file ops + patches

Verify:
```bash
go test ./internal/tools ./internal/kernel ./internal/store -count=1
```

Commit:
`feat(tooling): port core tool surface and execution controls`

## Slice P5-B — Sandboxing and automation media stack
Targets:
- `5.B` sandboxing backends
- `5.C` browser automation
- `5.D` vision/image generation
- `5.E` tts/voice/transcription

Verify:
```bash
go test ./internal/... -count=1
```

Commit:
`feat(runtime): add sandbox/browser/media integration stack`

## Slice P5-C — Skills/MCP/ACP/plugins/security
Targets:
- `5.F` remaining skills system
- `5.G` MCP integration
- `5.H` ACP integration
- `5.I` plugin architecture
- `5.J` approval/security guards

Verify:
```bash
go test ./internal/skills ./internal/kernel ./cmd/gormes -count=1
```

Commit:
`feat(platform): add extensibility and guardrail surfaces`

## Slice P5-D — Coordination/operator and distribution
Targets:
- `5.M` mixture of agents
- `5.N` misc operator tools
- `5.O` hermes CLI parity
- `5.P` docker/packaging
- `5.Q` tui gateway streaming

Verify:
```bash
go test ./cmd/gormes ./internal/tui ./internal/kernel ./docs -count=1
```

Commit:
`feat(distribution): finalize operator surfaces and packaging`

## Slice P5-E — Docs and ledger closeout
Verify:
```bash
go test ./docs -count=1
go test ./... -count=1
```

Commit:
`docs(phase5): finalize final-purge closeout`
