# Gormes Builder Delivery Gates

Use the narrowest gate that proves the row, then broaden for shared behavior.

## Always Consider

```sh
go run ./cmd/progress validate
go run ./cmd/progress next-work --repo-only
```

Before edits, record `git status --short`. If the working tree is already dirty,
preserve those paths and capture known broad-gate failures when affordable.
After edits, inspect status again so untracked implementation/test files are not
omitted from review.

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
- Goncho/memory/session: `go test ./internal/goncho ./internal/tools/goncho ./internal/memory ./internal/persistence/session ./internal/persistence/store -count=1`
- CLI/doctor/config: `go test ./cmd/gormes ./internal/platform/cli ./internal/platform/doctor ./internal/config -count=1`
- Progress schema/docs: `go test ./internal/planning/progress -count=1`
- TUI/API/server: `go test ./internal/tui ./internal/adapters/tuigateway ./internal/adapters/apiserver -count=1`

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

If `webpages/docs/` changed:

```sh
go test ./webpages/docs -count=1
```

If generated progress docs changed:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
go test ./internal/planning/progress -count=1
go test ./webpages/docs -run TestCompletionPlanCurrentFinishLedgerMatchesProgress -count=1
```

Use typed `internal/planning/progress.Load` / `SaveProgress`; inspect split
module changes and never overwrite a module that was dirty before the pass.
When a narrow row completes but its evidence atom still names adjacent gaps,
keep that atom `partial`.

If `www.gormes.ai` changed:

```sh
(cd webpages/landing && go test ./... -count=1)
```

Run Playwright e2e when page layout, install flows, progress surfaces, or user-visible browser behavior changed.

## RED Receipt

For test-backed rows, retain the exact focused command, failing test name, and
observable reason before implementation; rerun the same command after GREEN and
refactor. Harness setup failures do not count as behavioral RED.

## Security-Sensitive Policy

Filesystem, process, network, secret, and durable-state rows must carry explicit
trust ownership plus applicable root/symlink/cap/timeout/redaction/atomicity
policy before implementation. Use `t.TempDir`, fake transports, and inert
process fixtures. Missing policy is a planner repair, not a builder default.

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
- no unrelated user changes were reverted;
- `git status --short` is reviewed and every builder-owned tracked/untracked
  file is accounted for; `git diff --check` passes for tracked changes;
- unrelated full-suite failures are reported by exact test/command and are not
  failures in touched packages.
