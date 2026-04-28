# Retired Orchestrator Internals

The old autonomous builder/planner loop command binaries have been removed.
This directory now contains retired wrappers, watchdog compatibility checks,
systemd templates kept for historical cleanup, and notes for the old shell
entrypoints.

## Layout

- `*.sh` — retired compatibility wrappers. They now fail fast with guidance
  instead of invoking deleted loop binaries.
- `systemd/` — historical templates retained so old installed units can be
  identified and removed.
- `FROZEN.md` — freeze policy and the active Go-port exception.

## Watchdog

Install the 10-minute production watchdog with:

```sh
scripts/orchestrator/install-watchdog.sh --force
```

The watchdog is now compatibility-oriented: every tick checkpoints dirty
output, restarts `gormes-orchestrator.service` if inactive, runs
`go run ./cmd/progress validate`, and exits zero so the timer keeps firing
after recoverable failures.

## Running tests

```sh
go test ./internal/progress -count=1
```

## Legacy shell

Long-form frozen shell retained for parity lives under
`testdata/legacy-shell/scripts/` and is marked vendored for language reporting.
The root `scripts/gormes-auto-codexu-orchestrator.sh` wrapper no longer runs
the Go port because the loop command was removed.
Legacy management/resume invocations (`status`, `tail`, `abort`, `cleanup`,
`promote-commit`, `verify-gh-auth`, and `--resume`) temporarily exec the
vendored shell with the original arguments until full runtime parity lands.

The live companion scripts `scripts/gormes-architecture-planner-tasks-manager.sh`,
`scripts/documentation-improver.sh`, and `scripts/landingpage-improver.sh`
remain shell outside this cutover.

## Skill Replacement

Use `.agents/skills/gormes-skill-manager/SKILL.md` to route work, then
`gormes-planner`, `gormes-builder`, and `gormes-tdd-slice` for bounded passes.
Use `go run ./cmd/progress validate` for roadmap validation and
`go run ./cmd/progress write` for generated progress docs.

## Companion scheduling

The legacy orchestrator loop interleaves three companion scripts between cycles.
The Go port has typed companion scheduling primitives, but full runtime wiring
remains staged:

| Companion | Predicate | Typical cadence |
|---|---|---|
| `gormes-architecture-planner-tasks-manager.sh` | exhaustion (<10% candidates remain) OR cycles since last ≥ `PLANNER_EVERY_N_CYCLES` (default 4). Skipped if external systemd timer ran within `PLANNER_EVERY_N_CYCLES × LOOP_SLEEP_SECONDS × 2` seconds. | ~ every 4 cycles |
| `documentation-improver.sh` | cycles since last ≥ `DOC_IMPROVER_EVERY_N_CYCLES` (default 6) AND last cycle promoted ≥ 1 commit. | ~ every 6 productive cycles |
| `landingpage-improver.sh` | hours since last ≥ `LANDINGPAGE_EVERY_N_HOURS` (default 24). | daily |

Companions run serially on the integration worktree with `AUTO_COMMIT=1 AUTO_PUSH=0 PLANNER_INSTALL_SCHEDULE=0`, so their commits become the next cycle's `BASE_COMMIT`.

Escape hatches: `DISABLE_COMPANIONS=1` fully disables. `COMPANION_ON_IDLE=0` allows companions to run on any cycle (default `1` gates them to idle/post-promotion cycles).
