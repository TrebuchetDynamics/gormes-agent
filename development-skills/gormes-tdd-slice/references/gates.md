# TDD Slice Gates

Choose gates by touched surface.

## Minimum

```sh
go run ./cmd/progress validate
```

Confirm the selected test still matches the relevant section of
`docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md`.
If the row and map disagree, stop and route through `gormes-planner`.

## Runtime Change

```sh
go test ./... -count=1
```

## Focused Examples

```sh
go test ./internal/goncho ./internal/gonchotools ./internal/memory ./internal/persistence/session -count=1
go test ./internal/gateway ./internal/tools ./internal/llm -count=1
go test ./cmd/gormes ./internal/cli ./internal/doctor -count=1
go test ./internal/progress -count=1
```

## Parity-Focused Gates

- Goncho/Honcho: public `honcho_*` compatibility behavior, local Goncho
  storage, provenance, and degraded-mode errors should be fixture-backed.
- Provider: request/stream transcript, tool-call translation, retry/error
  classification, and credential-missing behavior should be hermetic.
- Normal turn: provider stream, tool continuation, memory hook, transcript, and
  final response should be observable through one public test path.
- API/TUI/gateway: request or frame contracts should consume kernel events
  rather than asserting private helper internals.

## Browser/Public Surface

Run docs or website tests when public docs, install flow, or visible web behavior changes:

```sh
go test ./webpages/docs -count=1
(cd webpages/landing && go test ./... -count=1)
```
