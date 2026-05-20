---
name: gormes-planner
description: Use when an agent must inspect upstream hermes-agent, honcho, docs, or progress.json; map Hermes/Honcho features into Go phases; refine or split roadmap rows; plan Goncho as the Honcho-compatible Go port inside Gormes; or update planning/docs/progress surfaces without implementing runtime feature code.
---

# Gormes Planner

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

## Mission

Plan Gormes until it is Hermes in Go, with no lesser definition of done. Goncho is the in-repo Go port of Honcho and must preserve Honcho-compatible external contracts where users/tools depend on them.

Hermes Agent is the Python upstream/father implementation for Gormes. Prefer
the in-repo checkout at `./hermes-agent`; fall back to `../hermes-agent` only
when the in-repo checkout is absent. Resolve it as `$HERMES_SRC` before citing
paths or row source refs. It is behavior evidence, not a Gormes runtime
dependency.

This skill is for bounded planner passes by Codex, Claude, or another agent.
It replaces the deleted autonomous planner-loop command.

Hard rule: do not recreate or invoke `cmd/planner-loop`. Inspect the
repository, upstream sources, docs, ledgers, and `progress.json` directly.
Repository evidence is the source of truth.

If the task might instead be parity audit, implementation, TDD, interface design, or skill creation, route through `gormes-skill-manager` first.

## Source Order

1. Read `AGENTS.md` and preserve the skill-driven progress contract.
2. Treat `docs/content/building-gormes/architecture_plan/progress.json` as the only backlog — a single *logical* backlog accessed via `internal/progress.Load`/`SaveProgress` or `cmd/progress`, which transparently handle either the monolithic file or a split/per-module directory layout (one logical backlog, physically one or many files; the on-disk form is the monolith today, split-capable). Never hand-parse member files. Before row selection or row reshaping, name the feature module boundary and use `go run ./cmd/progress list --module <module>` only as a scoped view over that one backlog.
3. Treat `docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md` as the canonical Hermes/Honcho-to-Go map.
4. Treat `docs/content/building-gormes/architecture_plan/upstream-coverage-ledger.md` as the completeness check for whether all feature-bearing Hermes/Honcho source classes are mapped.
5. Treat `docs/development-skills/<name>/SKILL.md` as the canonical skill source; `.agents/skills/`, `.claude/skills/`, and `.codex/skills/` are symlink loader views.
6. Use in-repo upstream references when present, otherwise sibling checkouts:
   - `$HERMES_SRC`: `./hermes-agent`, then `../hermes-agent`
   - `$HONCHO_SRC`: `./honcho`, then `../honcho`
7. Use existing Gormes code under `cmd/`, `internal/`, `docs/`, and `www.gormes.ai` as implementation evidence.
8. Read `references/parity-map.md` as a quick trigger checklist when deciding whether a subsystem is fully mapped.
9. Read `references/progress-row-contract.md` before editing progress rows.
10. For CLI/config/migration parity, inspect `$HERMES_SRC/hermes_cli/main.py`,
    `$HERMES_SRC/hermes_cli/commands.py`,
    `$HERMES_SRC/hermes_cli/config.py`, `$HERMES_SRC/hermes_cli/claw.py`,
    `$HERMES_SRC/gateway/run.py`, and the matching Gormes `cmd/gormes`
    and `internal/cli` surfaces before changing rows.

For TUI and channel UX parity, choose the active upstream implementation
explicitly. `$HERMES_SRC/ui-tui` is authoritative for the current
full-screen terminal app; `$HERMES_SRC/cli.py` remains evidence
for legacy prompt-toolkit behavior and command semantics. If a progress row
points only at stale upstream files, refine the row before handing it to a
builder.

## Workflow

### 1. Bound The Pass

State the pass goal before editing. Examples:

- "Map Hermes provider routing gaps into Phase 4 rows."
- "Map Honcho session/message/memory contracts into Goncho rows."
- "Split broad gateway rows into worker-ready TDD slices."

Reject vague expansion. If the user says "finish Gormes", choose one subsystem pass and explain why it is next.

Keep passes small enough to finish. A planner pass should produce builder-ready tracer-bullet rows, not a grand essay.

For full-map requests, update the feature map first, then reshape
`progress.json`. The docs map explains the destination; the rows are the only
executable queue.

To answer "is everything mapped?", update or audit the upstream coverage ledger.
No planner pass should claim full Hermes/Honcho coverage unless the ledger has
no unmapped feature-bearing source class and
`go test ./webpages/docs -run TestUpstreamCoverageLedgerMatchesSourceClasses -count=1`
passes when sibling upstream checkouts are available.

### 2. Inventory Current Reality

Run lightweight discovery first. These commands validate the shared progress
contract and inspect the backlog; they do not start an autonomous planner loop:

```sh
go run ./cmd/progress validate
go run ./cmd/progress list --module progress
go run ./cmd/progress emit | jq -r '.phases | to_entries[] | [.key, (.value.name // ""), ([.value.subphases[]?.items[]?] | length)] | @tsv'
find cmd internal -maxdepth 2 -type f -name '*.go' | sort
```

Use `rg` against upstream and Gormes rather than guessing. Compare contracts, command names, interfaces, fixtures, and tests.

Before changing rows, zoom out one level: name the relevant modules, callers, public contracts, tests, and upstream files. If this map is unclear, do more discovery instead of planning from vibes.

### 3. Map Upstream To Gormes

For each subsystem in scope:

1. Identify upstream behavior in Hermes/Honcho.
2. Identify the closest Gormes package or missing package.
3. Decide if the subphase is `porting`, `converged`, or `owned`.
4. Create or refine small rows only when there is an implementation gap.
5. Add or update the feature-map anchor when the behavior changes the Hermes/Honcho finish line.
6. Add or update the upstream coverage ledger when a new source class, SDK surface, endpoint family, or public document changes the map.
7. Prefer exact file paths, type names, command names, fixtures, and test commands.

Do not mark a row shipped unless repository evidence proves the behavior exists and tests cover it.

For large areas, design vertical slices. Each row should deliver one narrow behavior through all required layers rather than one horizontal layer of a future system.

### CLI, Config, And Migration Parity

When planning Hermes command or config parity:

- Treat `Hermes CLI command-tree parity manifest` as the gate before claiming
  broad command parity or assigning handler ports.
- Separate native config lifecycle from cross-product imports:
  `gormes config migrate` updates Gormes' own schema, while
  `gormes migrate hermes` and `gormes migrate openclaw` import external state.
- Classify Gormes-owned additions such as `goncho`, `--offline`, `--remote`,
  and XDG/TOML config as `owned` with source-backed rationale. Hermes-owned
  `-z/--oneshot` is removed-command guidance in Gormes; `gormes chat -q` is
  the canonical scripted-chat surface.
- Do not accept broad globs (`hermes_cli/**`, `gateway/**`, `_handle_*`) as
  sole evidence. Pair them with exact commands, symbols, fixtures, or tests.
- Preserve the public command spelling `openclaw`; typo-like requests such as
  `ooenclaw` should be planned as deterministic suggestions to
  `gormes migrate openclaw`, not silent aliases, unless a dedicated
  compatibility row explicitly changes that API policy.

### Persona, Templates, Skills, And Reset Defaults

When planning agent-default or "Gormes bot persona" parity, inspect the
upstream sources that actually seed and inject identity:
`$HERMES_SRC/hermes_cli/default_soul.py`,
`$HERMES_SRC/hermes_cli/config.py`,
`$HERMES_SRC/hermes_cli/profiles.py`,
`$HERMES_SRC/agent/prompt_builder.py`, and
`$HERMES_SRC/docker/SOUL.md` when container defaults matter. Rows must say
whether Gormes is porting Hermes' default `SOUL.md`, intentionally replacing
the brand text with Gormes identity, or preserving user-owned local
customization.

For skills parity, use exact sources such as
`$HERMES_SRC/agent/skill_commands.py`,
`$HERMES_SRC/agent/skill_preprocessing.py`,
`$HERMES_SRC/agent/skill_utils.py`,
`$HERMES_SRC/tools/skills_tool.py`,
`$HERMES_SRC/tools/skill_manager_tool.py`, and
`$HERMES_SRC/tools/skills_sync.py`. Plan user-visible skill behavior:
load order, template files, linked references, enabled/disabled/platform
filtering, reset/sync commands, and how skill tool calls appear to the model.

For reset behavior, separate session reset from development-environment reset.
Hermes session reset evidence lives in files such as
`$HERMES_SRC/tests/gateway/test_session_boundary_hooks.py`,
`$HERMES_SRC/tests/gateway/test_session_reset_notify.py`,
`$HERMES_SRC/tests/gateway/test_session_model_reset.py`, and
`$HERMES_SRC/tests/run_agent/test_session_reset_fix.py`. A Gormes
development reset follow-up must first inspect the completed `Gormes agent
template reset command` row, `internal/agenttemplate`, and
`cmd/gormes/agent_reset_test.go`. Extend or correct that surface with explicit
write_scope, fixture state, and acceptance; do not hide reset behavior inside
installer or generic runtime rows, and do not create a duplicate reset row.

Workspace-specific paths such as `workspace-gormes` and `workspace-mineru` are
planning evidence, not product defaults. Rows may require discovery or
fixture-backed behavior around those layouts, but must reject hard-coded
operator paths in Gormes code.

### 4. Rewrite Rows For Builders

Every executable row must be one worker-sized slice and include enough detail for a builder agent to start TDD immediately:

- `execution_owner`
- `slice_size`
- `contract`
- `contract_status`
- `ready_when`
- `not_ready_when`
- `write_scope`
- `test_commands` or `no_test_required`
- `acceptance`
- `source_refs`
- `done_signal`

Rows derived from the feature map should also identify the map section in
`source_refs`, `fixture`, or the row note, plus the Go package target and
public interface the builder should test.

Broad parity goals become umbrella inventory rows until split. Do not create private TODO files or side queues.

Apply the deep-module test when planning Go interfaces: a good Gormes module should expose a small caller-facing interface while hiding meaningful provider, gateway, session, memory, or persistence complexity. If a proposed row only moves shallow plumbing around, refine it before handing it to a builder.

### 5. Preserve Runtime Boundaries

Planner passes may edit planning surfaces:

- `progress.json`
- generated building-gormes docs
- upstream study docs
- public progress/web copy when roadmap messaging changes

Planner passes must not implement runtime feature code under `cmd/**/*.go` or `internal/**/*.go`. If runtime code is required, create a builder-ready row.

### 6. Iterate Deliberately

Use several bounded passes, not one unbounded prompt:

1. Inventory pass.
2. Parity-gap pass.
3. Row-readiness pass.
4. Dependency/order pass.
5. Validation/docs pass.

Stop after validation and report what remains. Do not keep planning until tokens run out.

When an interface shape is uncertain, create an interface-design task or use the `gormes-interface-designer` skill before writing implementation rows. When parity coverage is uncertain, use `gormes-parity-auditor` first.

### 7. Validate

After progress/doc edits, run only non-loop validation:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
go test ./internal/progress -count=1
go test ./webpages/docs -count=1
```

Do not recreate `cmd/planner-loop` here. If validation exposes stale loop
assumptions, fix the docs/rows/tests that encode those assumptions.

If `www.gormes.ai` content/data changed, also run:

```sh
(cd webpages/landing && go test ./... -count=1)
```

## Final Report

Report:

1. Scope scanned
2. Upstream features mapped
3. Progress rows added/refined/split
4. Goncho/Honcho implications
5. Feature-map sections changed
6. Builder-ready next rows
7. Validation evidence
8. Remaining unmapped Hermes/Honcho areas
9. Risks or blockers
