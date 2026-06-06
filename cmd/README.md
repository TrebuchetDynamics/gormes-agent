# Command Entry Points

This directory contains the remaining runnable command folders for Gormes.
Run commands from the repository root.

## Active Commands

| Command | Role | Typical invocation |
|---|---|---|
| `gormes` | User-facing runtime and TUI. | `go run ./cmd/gormes --offline` |
| `progress` | Validates `progress.json` and regenerates progress-driven docs/site data. | `go run ./cmd/progress validate` |
| `gormes-repo` | Updates repo metadata such as benchmark and README data. | `go run ./cmd/gormes-repo benchmark record` |

The old autonomous `builder-loop` and `planner-loop` command directories were
removed. Planning and building now happen through repo-local skills under
`.agents/skills/`, with `progress.json` as the only backlog.

## Common Recipes

Validate the canonical progress file without changing files:

```sh
make validate-progress
```

Regenerate progress-driven Markdown and site data:

```sh
make generate-progress
```

Build the main Gormes binary and run it in offline UI mode:

```sh
make build
./bin/gormes --offline
```

Update README benchmark text from `benchmarks.json`:

```sh
go run ./cmd/gormes-repo readme update
```

## Build Integration

The root `Makefile` wires these commands into normal contributor workflows:

```sh
make validate-progress  # go run ./cmd/progress validate
make generate-progress  # go run ./cmd/progress write
make build              # build gormes, record repo metrics, refresh generated docs
```

Use `.agents/skills/gormes-skill-manager/SKILL.md` to route planner, builder,
TDD, parity-audit, and interface-design work.
