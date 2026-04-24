# Gormes Phase 4 Finish Sprint

> Execution: strict TDD, provider adapters behind stable interfaces, no hidden side effects.

**Goal:** Deliver native Go orchestrator foundations (providers, context engine, prompt builder, routing, telemetry, credentials, resilience) and close all Phase 4 ledger items.

## Verified baseline
- Phase 4 status: **0/15 complete**.
- Remaining: all Phase 4 items (`4.A`..`4.H`).

## Definition of Done
1. All Phase 4 items marked `complete`.
2. Provider adapter contract tests pass.
3. `go test ./internal/hermes ./internal/kernel ./internal/config ./docs -count=1` passes.
4. `go test ./... -count=1` passes.

## Slice P4-A — Provider adapter framework + first providers
Targets:
- `4.A` adapters: Anthropic, Bedrock, Gemini, OpenRouter, Google Code Assist, Codex

Files:
- `internal/hermes/*`
- `internal/config/*`
- `internal/hermes/*_test.go`

Verify:
```bash
go test ./internal/hermes ./internal/config -count=1
```

Commit:
`feat(provider): add multi-provider adapter framework and implementations`

## Slice P4-B — Context engine + compression
Targets:
- `4.B` long-session management and compression

Files:
- `internal/kernel/*`
- `internal/memory/*` (only integration seams)

Verify:
```bash
go test ./internal/kernel -count=1 -race
```

Commit:
`feat(kernel): add long-session context engine and compression`

## Slice P4-C — Native prompt builder + routing
Targets:
- `4.C` prompt assembly
- `4.D` per-turn model routing

Files:
- `internal/kernel/*`
- `internal/config/*`

Verify:
```bash
go test ./internal/kernel ./internal/config -count=1
```

Commit:
`feat(kernel): add native prompt builder and model router`

## Slice P4-D — Telemetry/title/credentials/resilience
Targets:
- `4.E` trajectory + insights
- `4.F` title generation
- `4.G` credentials + oauth
- `4.H` rate/retry/caching

Files:
- `internal/telemetry/*`
- `internal/session/*`
- `internal/config/*`
- `cmd/gormes/*`

Verify:
```bash
go test ./internal/telemetry ./internal/session ./cmd/gormes -count=1
```

Commit:
`feat(runtime): finalize phase4 operator intelligence and resilience`

## Slice P4-E — Docs and ledger closeout
Verify:
```bash
go test ./docs -count=1
go test ./... -count=1
```

Commit:
`docs(phase4): finalize brain-transplant phase closeout`
