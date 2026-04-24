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

Real `run` executions synchronize the three external source repos before
building planner context:

- existing git repo: `git -C <path> pull --ff-only`
- missing repo: `git clone <url> <path>`

Default clone URLs:

- Hermes: `https://github.com/NousResearch/hermes-agent.git`
- GBrain: `https://github.com/garrytan/gbrain.git`
- Honcho: `https://github.com/plastic-labs/honcho`

Override clone URLs with `HERMES_REPO_URL`, `GBRAIN_REPO_URL`, and
`HONCHO_REPO_URL`. `PLANNER_SYNC_REPOS=0` is reserved for tests and controlled
local debugging.

Dry-run mode writes planner context and prompt artifacts without pulling or
cloning external repositories.

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
