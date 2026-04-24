# Architecture Planner Loop Command

`cmd/architecture-planner-loop` improves the building-gormes architecture plan.
It is the planning counterpart to `cmd/autoloop`: autoloop executes roadmap
rows, while this command asks a planner agent to study source/reference context
and refine `docs/content/building-gormes/architecture_plan/progress.json`.

Run from the repository root:

```sh
go run ./cmd/architecture-planner-loop run --dry-run
go run ./cmd/architecture-planner-loop run --codexu
go run ./cmd/architecture-planner-loop run --claudeu
go run ./cmd/architecture-planner-loop status
go run ./cmd/architecture-planner-loop show-report
```

## Sources

The planner context includes:

- `../hermes-agent`
- `../gbrain`
- `../honcho`
- `docs/content/upstream-hermes`
- `docs/content/upstream-gbrain`
- `docs/content/building-gormes`

Override source paths with `HERMES_DIR`, `GBRAIN_DIR`, and `HONCHO_DIR`.

## Artifacts

By default artifacts are written under `.codex/architecture-planner/`:

- `context.json`
- `latest_prompt.txt`
- `latest_planner_report.md`
- `latest_planner_report.raw.md`
- `planner_state.json`
- `validation.log`

Override the artifact root with `RUN_ROOT`.

## Backends

The default backend is `codexu`. Use `--claudeu` only on hosts where `claudeu`
is installed.

## Validation

Real runs validate planner edits with:

```sh
go run ./cmd/progress-gen -write
go run ./cmd/progress-gen -validate
go test ./internal/progress -count=1
go test ./docs -count=1
```

Set `PLANNER_VALIDATE=0` only for tests or controlled local debugging.
