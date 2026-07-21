---
name: gormes-planner
description: Shape source-backed Gormes progress rows. Use for decision=plan builder handoffs, row splitting, feature maps, or parity planning. Do not use when decision=build or for runtime code.
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
If Juan explicitly asks for a planner-loop style subsystem, plan it as a
first-class Gormes feature with clear interfaces, progress rows, validation
gates, and operator controls. Otherwise inspect the repository, upstream
sources, docs, ledgers, progress rows, and parity evidence directly. Repository
evidence is the source of truth.

If the task might instead be parity audit, implementation, TDD, interface design, or skill creation, route through `gormes-skill-manager` first.

## Source Order

1. Read `AGENTS.md` and preserve the skill-driven evidence contract.
2. Treat the logical progress backlog as canonical for implementation intent and TODO intake. Use `cmd/progress` and `internal/planning/progress` (`Load`/`SaveProgress`) for progress data; never hand-parse split-layout members or create side queues. Do not append active work to `TODO.md`.
3. Treat `webpages/docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md` as the canonical Hermes/Honcho-to-Go map for subsystem context.
4. Treat `webpages/docs/content/building-gormes/architecture_plan/upstream-coverage-ledger.md` as the completeness check for whether all feature-bearing Hermes/Honcho source classes are mapped.
5. Treat `webpages/docs/hermes-releases/FEATURE-MATRIX.md` as a release-note study aid for choosing high-signal improvement lanes; it is not an executable queue and never replaces source refs or behavior atoms.
6. Use `./hermes-knowledge-graph.json` or `$HERMES_SRC/.understand-anything/knowledge-graph.json` as topology/routing accelerators before broad source searches; graph nodes and layers are navigation hints, not coverage proof.
7. Treat `development-skills/<name>/SKILL.md` as the canonical skill source; `.agents/skills/`, `.claude/skills/`, and `.codex/skills/` are symlink loader views.
8. Use in-repo upstream references when present, otherwise sibling checkouts:
   - `$HERMES_SRC`: `./hermes-agent`, then `../hermes-agent`
   - `$HONCHO_SRC`: `./honcho`, then `../honcho`
9. Use existing Gormes code under `cmd/`, `internal/`, `webpages/docs/`, and `www.gormes.ai` as implementation evidence.
10. Read existing parity evidence docs before creating or reshaping progress rows; they are source-backed classification evidence, not the backlog itself.
11. For CLI/config/migration parity, inspect `$HERMES_SRC/hermes_cli/main.py`,
    `$HERMES_SRC/hermes_cli/commands.py`,
    `$HERMES_SRC/hermes_cli/config.py`, `$HERMES_SRC/hermes_cli/claw.py`,
    `$HERMES_SRC/gateway/run.py`, and the matching Gormes `cmd/gormes`
    and `internal/platform/cli` surfaces before changing rows/evidence.

For TUI and channel UX parity, choose the active upstream implementation
explicitly. `$HERMES_SRC/ui-tui` is authoritative for the current
full-screen terminal app; `$HERMES_SRC/cli.py` remains evidence
for legacy prompt-toolkit behavior and command semantics. If a parity evidence
entry points only at stale upstream files, refine the row/evidence
before handing it to a builder.

## Workflow

### 1. Bound The Pass

State the pass goal before editing. Examples:

- "Map Hermes provider routing gaps into Phase 4 rows."
- "Map Honcho session/message/memory contracts into Goncho rows."
- "Split broad gateway rows into worker-ready TDD slices."

Reject vague expansion. If the user says "finish Gormes", choose one subsystem pass and explain why it is next. If the user asks to add TODOs or missing features, first route the raw idea through `gormes-progress-slicer`; planner edits only rows that can be made concrete with source refs, write scope, readiness, and validation.

Keep passes small enough to finish. A planner pass should produce builder-ready tracer-bullet rows, not a grand essay.

For full-map requests, update the feature map first, then reshape
the progress rows. The docs map explains the destination; the logical progress backlog is the
executable queue. If the request is seeded by `webpages/docs/hermes-releases/FEATURE-MATRIX.md`
or `hermes-knowledge-graph.json`, use those artifacts to choose the smallest
release family or source topology to inspect, then prove the active contract in
`$HERMES_SRC` before editing rows or evidence.

To answer "is everything mapped?", update or audit the upstream coverage ledger.
No planner pass should claim full Hermes/Honcho coverage unless the ledger has
no unmapped feature-bearing source class and
`go test ./webpages/docs -run TestUpstreamCoverageLedgerMatchesSourceClasses -count=1`
passes when sibling upstream checkouts are available.

### 2. Inventory Current Reality

Run lightweight discovery first:

```sh
# Inspect the canonical logical backlog / TODO control plane
go run ./cmd/progress next-work
go run ./cmd/progress next-work --repo-only
go run ./cmd/progress list --module <module>

# Consult parity evidence as source-backed classification, not as the queue
rg -n "missing|partial|covered|<topic>" webpages/docs/parity-evidence webpages/docs/content/building-gormes

# Discover Gormes code surface
find cmd internal -maxdepth 2 -type f -name '*.go' | sort
```

Use `rg` against upstream and Gormes rather than guessing. When
`hermes-knowledge-graph.json` exists, query its layers or tour before broad
`rg` sweeps to pick likely upstream files, then read those files directly.
Compare contracts, command names, interfaces, fixtures, and tests.

For "actually implemented features" or stale-`missing` sweeps, treat the pass as evidence reconciliation before row creation:

- Start from progress rows or evidence entries marked `missing` or broad `partial` in the requested surface.
- Search `cmd/` and `internal/` by row/evidence name, upstream symbol, command/tool name, channel name, and likely package nouns.
- If checked-in code plus focused tests prove the behavior already exists, update the progress row and/or parity evidence to `covered` or `partial` with exact Go files/tests and remaining gaps.
- Do not create a builder row for already implemented behavior. Create/refine a row only for the unimplemented remainder after evidence is corrected.
- If the reconciliation reveals a concrete remaining runtime gap, leave the row/evidence entry `partial` and make the builder-facing gap explicit enough for `gormes-builder` to select only that remainder.

Before changing rows, zoom out one level: name the relevant modules, callers, public contracts, tests, and upstream files. If this map is unclear, do more discovery instead of planning from vibes.

### Decision=Plan Fast Path

Use this directly when a builder reports `decision=plan`; do not send an exact,
source-backed packet through another discovery loop.

1. Re-run both selectors in the live checkout:
   `go run ./cmd/progress next-work` and
   `go run ./cmd/progress next-work --repo-only`.
2. If repo-only now says `decision=build`, stop planning, name that row, and
   route to `gormes-builder`; never add a duplicate.
3. If it still says `decision=plan`, search existing rows first. Refine one
   vague planned row, or choose one proven `partial`/`missing` behavior atom
   with exact upstream refs and a real in-repo gap.
4. Run a stale-classification preflight against current Go code/tests. If the
   behavior exists, correct evidence instead of creating runtime work.
5. Create one repo-scoped vertical row through
   `internal/planning/progress.Load` / `SaveProgress`. Include every field in
   [the row contract](references/progress-row-contract.md).
6. Regenerate, validate, and rerun repo-only selection. The pass succeeds only
   when the intended row is selected, or a higher-ranked row is selected and
   explicitly reported.

A builder-blocked packet should follow the canonical
[builder-to-planner handoff](../gormes-builder/references/planner-handoff.md):
name the candidate atom/row, exact upstream symbols/tests, current Go insertion
point, intended module, exclusions, and first hermetic RED fixture; preserve
`unknown` rather than inventing evidence. Keep classifications in
`HERMES-BEHAVIOR-ATOMS.md`; implementation intent belongs in the logical
backlog. Do not create `TODO.md`, issue lists, prompt queues, or private
ledgers.

Dirty planning/docs work is a hygiene constraint, not a refusal reason. Record
the pre-pass dirty paths before editing and follow the split-layout rules below.

### 3. Map Upstream To Gormes

For each subsystem in scope:

1. Identify upstream behavior in Hermes/Honcho.
2. Identify the closest Gormes package or missing package.
3. Decide if the subphase is `porting`, `converged`, or `owned`.
4. Create or refine small rows only when there is an implementation gap.
5. Add or update the feature-map anchor when the behavior changes the Hermes/Honcho finish line.
6. Add or update the upstream coverage ledger when a new source class, SDK surface, endpoint family, or public document changes the map.
7. Prefer exact file paths, type names, command names, fixtures, and test commands.

Do not mark a row or evidence entry covered unless repository evidence proves the behavior exists and tests cover it. Do not leave it `missing` when repository evidence proves a narrower `partial`; stale `missing` classifications cause `gormes-builder` to duplicate already-shipped code.

For large areas, design vertical slices. Each row should deliver one narrow behavior through all required layers rather than one horizontal layer of a future system.

### Specialized Lanes

Load [specialized planner lanes](references/specialized-lanes.md) only when the
request concerns CLI/config/migration parity, persona/templates/skills/reset
defaults, or exact external review feedback. Keep those rare rules out of the
default planning path.

### 4. Rewrite Rows For Builders

Every executable row must be one worker-sized slice and include enough detail for a builder agent to start TDD immediately. Apply the row contract's security-sensitive readiness gate before using `fixture_ready`; unresolved filesystem, process, network, secret, or persistence policy stays `draft`.

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

After editing, prove the row is actually selectable:

```sh
go run ./cmd/progress validate
go run ./cmd/progress next-work --repo-only
```

If the selected next-work row is not the one you just repaired, explain why
(the other row has higher priority/unblocks more work) and do not claim the
builder remains blocked.

Rows derived from the feature map should also identify the map section in
`source_refs`, `fixture`, or the row note, plus the Go package target and
public interface the builder should test.

Broad parity goals become umbrella inventory rows until split. Do not create private TODO files or side queues.

Apply the deep-module test when planning Go interfaces: a good Gormes module should expose a small caller-facing interface while hiding meaningful provider, gateway, session, memory, or persistence complexity. If a proposed row only moves shallow plumbing around, refine it before handing it to a builder.

### 5. Preserve Runtime Boundaries

Planner passes may edit planning surfaces:

- logical progress data under `webpages/docs/content/building-gormes/architecture_plan/progress.json` or its split layout, through `cmd/progress` / `internal/planning/progress`
- parity evidence docs when source-backed classifications change
- generated building-gormes docs
- upstream study docs
- public progress/web copy when roadmap messaging changes

Planner passes must not implement runtime feature code under `cmd/**/*.go` or `internal/**/*.go`. Do not edit runtime during a planner pass, even when the insertion point is obvious. If runtime code is required, create a builder-ready row.

#### Split-Layout Dirty-Tree Hygiene

- Load and save the logical backlog only through
  `internal/planning/progress.Load` / `SaveProgress` or `cmd/progress`; never
  hand-parse or directly mutate module JSON.
- Snapshot dirty paths before saving. Afterwards inspect changed files under
  `progress.json/modules/`; expected row-module changes are allowed.
- If canonical serialization touches an unrelated module that was clean before
  the pass, restore only that incidental clean-file change. Never restore,
  overwrite, or normalize a file that was already dirty.
- Run `go run ./cmd/progress write` after the logical row is correct. Reconcile
  the hand-maintained completion ledger when row totals/open counts changed.

### 6. Iterate Deliberately

Use several bounded passes, not one unbounded prompt:

1. Inventory pass.
2. Parity-gap pass.
3. Atom-readiness pass.
4. Dependency/order pass.
5. Validation/docs pass.

Stop after validation and report what remains. Do not keep planning until tokens run out.

When an interface shape is uncertain, create an interface-design task or use the `gormes-interface-designer` skill before writing implementation rows. When parity coverage is uncertain, use `gormes-parity-auditor` first.

### 7. Validate

After evidence/edit updates, run non-loop validation:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
go run ./cmd/progress next-work --repo-only
go test ./internal/planning/progress -count=1
go test ./webpages/docs -run TestCompletionPlanCurrentFinishLedgerMatchesProgress -count=1
go test ./webpages/docs -count=1
git diff --check
```

Run `write` only when progress data changed. Report unrelated full-docs
failures separately; focused progress/ledger gates must still pass.

If this pass introduces planner-loop behavior, keep it explicitly scoped,
progress-row-backed, and validated. If validation exposes stale loop
assumptions, fix the docs/rows/tests that encode those assumptions.

If `www.gormes.ai` content/data changed, also run:

```sh
(cd webpages/landing && go test ./... -count=1)
```

## Example

Builder packet: `decision=plan`; the webhook filter atom is `partial`; upstream
names `_load_filter_file_values`; current Go filters omit `in_file`.

Planner outcome: rerun selectors, prove the gap, and define injected home,
absolute/symlink, size, and degraded policy with hermetic fixtures. If any
owner decision remains, keep the row `draft`; otherwise prove repo-only selects
it. Script execution stays excluded.

Behavioral evaluation: this scenario is pinned for deterministic review, but
no paid paired model run was approved; the skill improvement is unreplicated.

## Output Contract

```text
scope: <bounded subsystem/pass>
upstream_mapped: <exact symbols/tests>
rows_changed: <phase/subphase/name and before→after>
feature_map_evidence: <changed anchor or unchanged reason>
selector_result: <exact decision/build row or decision/plan reason>
validation: <commands and outcomes>
remaining: <named gaps, including Goncho/Honcho implications>
risks: <security/owner blockers or none>
next_skill: <gormes-builder or evidence/design skill>
```

Do not claim `builder_ready` unless fresh repo-only selection proves it. Include
exact full-docs failures when unrelated gates remain red.
