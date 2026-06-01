# Gormes Builder Delivery Gates

Use the narrowest gate that proves the row, then broaden for shared behavior.

## Always Consider

```sh
go run ./cmd/progress validate
```

For rows derived from `hermes-honcho-feature-map.md`, verify the row still
matches the map section, Go package target, degraded mode, and proof gate. If
the implementation changes the claim, update the map or send the work back to
`gormes-planner`.

For runtime code:

```sh
go test ./... -count=1
```

## Package-Focused Gates

- Gateway/provider/tooling: `go test ./internal/gateway ./internal/tools ./internal/llm -count=1`
- Goncho/memory/session: `go test ./internal/goncho ./internal/gonchotools ./internal/memory ./internal/persistence/session ./internal/persistence/store -count=1`
- CLI/doctor/config: `go test ./cmd/gormes ./internal/cli ./internal/doctor ./internal/config -count=1`
- Progress schema/docs: `go test ./internal/progress -count=1`
- TUI/API/server: `go test ./internal/tui ./internal/tuigateway ./internal/apiserver -count=1`

Adjust package lists to the actual touched files.

## Feature Map Gates

- Phase 4.I normal-turn work must prove provider stream, tool continuation,
  session transcript, memory hook, final response, and audit evidence with
  hermetic fixtures as rows become available.
- Phase 3.G Goncho work must preserve public Honcho-compatible `honcho_*`
  tool/client names while keeping internal package names as Goncho.
- Provider rows should include transcript fixtures and error-classification
  fixtures before any live-provider smoke.
- API/TUI/gateway rows should consume typed turn events instead of duplicating
  kernel behavior.
- When a future `internal/e2e` package exists, normal-turn rows should include
  its focused command before broad `go test ./...`.

## Docs And Web Gates

If `docs/` changed:

```sh
go test ./webpages/docs -count=1
```

If generated progress docs changed:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
```

If `www.gormes.ai` changed:

```sh
(cd webpages/landing && go test ./... -count=1)
```

Run Playwright e2e when page layout, install flows, progress surfaces, or user-visible browser behavior changed.

## Hermetic Test Policy

- Prefer fixtures and fakes over live providers.
- Do not require network credentials for row-local tests.
- If a live-provider test is necessary, gate it behind an explicit env var and add a hermetic test for default CI.

## Completion Standard

A row is done only when:

- implementation matches the row contract;
- tests prove the contract and fail meaningfully without the implementation;
- progress validation still passes;
- docs/web surfaces are synchronized when public behavior changed;
- no unrelated user changes were reverted.
