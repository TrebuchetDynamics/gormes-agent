# Autoloop Internals

The production implementation now lives in Go under `cmd/autoloop` and
`internal/autoloop`. This directory contains transitional wrappers, systemd
templates, and historical notes for the old shell entrypoints.

## Layout

- `*.sh` — tiny compatibility wrappers that exec `go run ./cmd/autoloop ...`.
- `systemd/` — templates rendered or installed by `autoloop service ...`.
- `FROZEN.md` — freeze policy and the active Go-port exception.

## Running tests

```sh
go test ./internal/autoloop ./cmd/autoloop -count=1
```

## Legacy shell

Long-form frozen shell retained for parity lives under
`testdata/legacy-shell/scripts/orchestrator/` and is marked vendored for
language reporting.

## Backends

`internal/autoloop` owns backend adapters. `BACKEND` (env var) or the equivalent
CLI flag selects which agent CLI drives workers. The worker contract is
unchanged across backends; each backend only translates argv.

| Backend | Binary | CLI flag | Notes |
|---|---|---|---|
| `codexu` (default) | `codexu` | `--codexu` | Native codex-cli non-interactive mode. |
| `claudeu` | `claudeu` shim (PATH) | `--claudeu` | Shim translates codexu-style argv to `claude --print`. |
| `opencode` | `opencode` | `--opencode` | Uses `opencode run --no-interactive`; shape approximate. |

Switch via env (`BACKEND=claudeu $0`) or flag (`$0 --claudeu`). CLI flag wins.

## Companion scheduling

The orchestrator's forever loop interleaves three companion scripts between cycles:

| Companion | Predicate | Typical cadence |
|---|---|---|
| `gormes-architecture-planner-tasks-manager.sh` | exhaustion (<10% candidates remain) OR cycles since last ≥ `PLANNER_EVERY_N_CYCLES` (default 4). Skipped if external systemd timer ran within `PLANNER_EVERY_N_CYCLES × LOOP_SLEEP_SECONDS × 2` seconds. | ~ every 4 cycles |
| `documentation-improver.sh` | cycles since last ≥ `DOC_IMPROVER_EVERY_N_CYCLES` (default 6) AND last cycle promoted ≥ 1 commit. | ~ every 6 productive cycles |
| `landingpage-improver.sh` | hours since last ≥ `LANDINGPAGE_EVERY_N_HOURS` (default 24). | daily |

Companions run serially on the integration worktree with `AUTO_COMMIT=1 AUTO_PUSH=0 PLANNER_INSTALL_SCHEDULE=0`, so their commits become the next cycle's `BASE_COMMIT`.

Escape hatches: `DISABLE_COMPANIONS=1` fully disables. `COMPANION_ON_IDLE=0` allows companions to run on any cycle (default `1` gates them to idle/post-promotion cycles).
