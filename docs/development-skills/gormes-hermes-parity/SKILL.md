---
name: gormes-hermes-parity
description: Use when running a recurring or periodic Gormes-vs-Hermes parity sweep, checking overall Hermes-in-Go progress, refreshing parity definitions, safely renaming or restructuring parity taxonomy, delegating follow-up tasks through gormes-skill-manager, or recording source-backed parity evidence into progress.json.
---

# Gormes Hermes Parity

## Mission

Run a bounded recurring parity sweep that keeps Gormes pointed at the real
finish line: Hermes in Go, with Goncho as the Honcho-compatible Go port. This
skill coordinates audit and planning work. It records progress in the canonical
roadmap and hands implementation to builder skills; it does not create a second
queue.

Use this when the user says `gormes-hermes-parity`, asks for a periodic parity
check, asks "what is left for full parity?", or wants ambiguous parity goals
turned into source-backed progress rows. It may also reshape parity taxonomy,
feature-map sections, progress row structure, and public progress wording when
the current names or grouping hide the real Hermes contract.

## Skill Chain

Default chain:

```text
gormes-hermes-parity
  -> gormes-skill-manager for routing follow-up tasks
  -> gormes-parity-auditor
  -> gormes-planner
  -> gormes-builder / gormes-tdd-slice for selected implementation rows
```

Keep each run bounded. If the user says "everything", produce a subsystem map
and the next three concrete passes instead of trying to audit the whole repo in
one turn.

## Delegation Through Skill Manager

Use `gormes-skill-manager` whenever the sweep discovers work that is bigger
than the current bounded pass, crosses subsystem boundaries, or requires a
different delivery mode.

Delegate by producing small task packets, not loose TODOs:

```text
task:
scope:
feature_map_area:
progress_row:
recommended_skill_chain:
source_refs:
write_scope:
validation:
blocked_by:
```

Route typical follow-ups this way:

| Follow-up | Ask skill manager to route to |
|---|---|
| Need more upstream comparison | `gormes-parity-auditor` |
| Need progress rows, taxonomy, or docs reshaped | `gormes-planner` |
| Need one row implemented | `gormes-builder` |
| Need tests-first runtime behavior | `gormes-tdd-slice` |
| Need package/API boundary design | `gormes-interface-designer` |
| Need provider/auth/model behavior | `gormes-provider-parity` |
| Need browser automation parity | `gormes-browser-harness` |
| Need donor Go implementation shape | `gormes-references` |

When the agent runtime supports parallel workers and the user has authorized
delegation, independent task packets may run in parallel. Keep write scopes
disjoint, tell workers they are not alone in the codebase, and make each worker
report changed files and validation. Otherwise, record the packets as the next
builder-ready work and stop.

## Default Periodic Prompt

Use this as the general-purpose recurring prompt:

```text
Use $gormes-hermes-parity to run a bounded Hermes/Gormes parity sweep. If I do
not name a scope, choose the highest-risk or stalest incomplete surface from
progress.json and recent upstream evidence. Compare ../hermes-agent, ../honcho,
and ../gbrain against current Gormes, classify each behavior, update canonical
progress evidence and rows when needed, run validation, and finish with the next
builder-ready rows. If the existing taxonomy is misleading, safely rename or
restructure it with source-wide reference checks and compatibility notes. Use
gormes-skill-manager to route or delegate follow-up task packets.
```

## Parity Definitions

Use one of these labels for every behavior in scope:

| Label | Meaning |
|---|---|
| `strict` | Gormes must match upstream names, inputs, outputs, errors, side effects, and registration exactly. |
| `functional` | Gormes preserves the user/operator contract, but the Go internals or provider shape may differ. This is the default target. |
| `owned` | Gormes intentionally diverges or extends Hermes. The row must explain why and how compatibility is preserved or why it is not required. |
| `excluded` | Upstream behavior is intentionally not part of Gormes. This needs explicit source-backed rationale and user-visible risk noted. |

Use these progress classifications during the sweep:

| Classification | Record it as |
|---|---|
| `covered` | Implemented, tested, and source-backed. Mark complete only with repository evidence. |
| `planned` | Represented by a builder-ready row with acceptance and tests. |
| `vague` | A row exists but is too broad, ambiguous, missing tests, or missing source refs. Refine or split it. |
| `missing` | No useful Gormes code or progress row exists. Add the smallest source-backed row. |
| `stale-upstream` | Existing evidence points at old upstream behavior. Refresh refs and acceptance. |
| `blocked` | Cannot proceed because a dependency, source checkout, credential, or interface decision is absent. Record the blocker explicitly. |

When strict and functional parity conflict, prefer functional parity only when
the difference is documented as `owned` or the public Hermes contract is still
preserved.

## Periodic Workflow

### 1. Bound The Sweep

Pick one surface unless the user named several independent scopes:

- web/tools and native tool descriptors;
- provider/auth/model routing and usage;
- CLI/config/migration command tree;
- gateway, channels, and operator flows;
- sessions, memory, and Goncho/Honcho compatibility;
- prompt/context/runtime loop behavior;
- plugins, skills, browser automation, packaging, docs, or public progress.

If no scope is named, choose the highest-value surface by checking incomplete
rows, stale source refs, recent upstream changes, and user-visible risk.

### 2. Establish Baseline

Run lightweight discovery before editing:

```sh
git status --short --branch
go run ./cmd/progress validate
git rev-parse --short HEAD
git -C ../hermes-agent rev-parse --short HEAD || true
git -C ../honcho rev-parse --short HEAD || true
git -C ../gbrain rev-parse --short HEAD || true
```

Then inspect the relevant feature-map section, upstream coverage ledger,
matching `progress.json` rows, and current Gormes packages. Use `rg`, `find`,
and `jq`; do not infer parity from file names alone.

### 3. Inventory Upstream Behavior

For the scoped surface, list exact upstream files, symbols, commands, tests,
docs, request/response contracts, fixtures, and registration points from:

- `../hermes-agent`
- `../honcho`
- `../gbrain`

If a sibling checkout is missing, record the missing source as a blocker. Do
not replace upstream evidence with memory or guesses.

### 4. Compare To Gormes

For each upstream behavior:

1. Identify the closest Gormes package, command, tool, fixture, or missing area.
2. Classify the parity state as `covered`, `planned`, `vague`, `missing`,
   `stale-upstream`, `blocked`, or `owned`.
3. Assign the parity definition: `strict`, `functional`, `owned`, or `excluded`.
4. Link exact source refs and current Gormes evidence.
5. Decide whether to update the feature map, coverage ledger, or progress row.

Use `gormes-parity-auditor` for the detailed source comparison when the surface
is not already clear. Use `gormes-planner` before editing rows or planning docs.

### 5. Record Progress

Use only canonical progress surfaces. Do not create side TODO files, private
ledgers, or prompt-only backlog lists.

For each gap that needs work, update or create a row with:

- source refs from upstream and Gormes;
- parity definition and classification;
- observable contract;
- acceptance and done signal;
- focused test commands;
- write scope;
- dependencies and blockers;
- `ready_when` and `not_ready_when`.

If a row is complete, only mark it complete when code, tests, and docs evidence
prove the behavior. If a behavior is `owned`, document the divergence and the
compatibility boundary in the row.

If parity docs or coordinator briefs still describe a row as `regressed`,
`planner refinement`, or a top-P0 blocker after `progress.json` and source/tests
show the row is already complete, do a docs/progress reconciliation slice instead
of creating new runtime work. Verify the completed row's focused tests first,
update the parity matrix/detail sections and coordinator next-slice ordering to
remove stale blocker language, run the progress/docs gates, and commit only the
docs/progress surfaces. Example: after `Telegram MarkdownV2 parse-mode rendering
closeout` was complete, stale matrix/brief docs still advertised it as a
regressed planner-refinement P0; the correct bounded pass was to reconcile those
docs and promote the next P0 rows, not reimplement ParseMode wiring.

### 6. Safe Taxonomy And Restructure Mode

Use this mode when parity labels, feature-map headings, phase/subphase names,
row names, public progress wording, or package-level terminology no longer
match the upstream contract.

Allowed in the same parity sweep:

- rename or split parity labels and classifications;
- rename feature-map headings or coverage-ledger categories;
- split, merge, or regroup progress rows and subphases;
- update generated progress docs and `www.gormes.ai` progress data;
- update skill-routing docs when the workflow taxonomy changes.

For runtime Go identifiers, commands, tool names, config keys, database fields,
or public APIs, first decide whether the rename is internal, compatibility
preserving, or breaking. Internal renames can be handled as refactors. Public
renames need aliases, migration notes, or a builder-ready compatibility row.

Follow this safety loop:

1. Write a mapping table before editing: `old name -> new name`, scope,
   source refs, and compatibility decision.
2. Use `rg -n` to find every current reference across `cmd`, `internal`,
   `docs`, `www.gormes.ai`, skills, tests, and generated data.
3. Update structured data with structured tools when practical. Preserve
   `progress.json` schema fields and row identity unless the row split/merge is
   the point of the refactor.
4. Keep user-facing aliases for any public name unless a source-backed decision
   says the compatibility break is intentional.
5. Regenerate derived docs with `go run ./cmd/progress write`.
6. Re-run `rg -n` for old terms. Remaining references must be intentional
   history, migration, or compatibility notes.
7. Run the full validation set for the touched surfaces.

Large restructures should land as one no-behavior-change taxonomy migration,
then separate builder rows for runtime behavior. Do not mix a broad rename with
new feature implementation unless the user explicitly asks for that combined
slice and the tests prove both.

### 7. Validate

After progress or docs edits, run:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
go test ./internal/progress -count=1
go test ./docs -count=1
git diff --check
```

If only this skill or routing docs changed, validate the skill shape and run
`git diff --check`.

If upstream coverage claims changed, also run:

```sh
go test ./docs -run TestUpstreamCoverageLedgerMatchesSourceClasses -count=1
```

If `www.gormes.ai` data changed, also run:

```sh
(cd www.gormes.ai && go test ./... -count=1)
```

If runtime Go identifiers, commands, tools, config, persistence, or public APIs
changed as part of a taxonomy refactor, also run the focused package tests and
then:

```sh
go test ./... -count=1
```

### 8. Report The Sweep

Finish with this compact report:

```text
scope:
upstream_refs:
gormes_refs:
parity_definition:
classification_summary:
taxonomy_changes:
rows_changed:
compatibility_notes:
delegated_task_packets:
validation:
next_builder_rows:
blockers:
```

Include exact file paths and commands. Do not claim "full parity" unless the
feature map, coverage ledger, progress rows, and validation all support it.

## Guardrails

- Do not implement runtime code in a parity sweep. Create or refine builder-ready
  rows, then use `gormes-builder` and `gormes-tdd-slice` for implementation.
- Do not treat a passing unit test as parity unless upstream behavior was
  compared and source refs are recorded.
- Do not mark vague umbrella rows complete.
- Do not silently accept owned divergences. Name them and explain the boundary.
- Do not perform broad taxonomy renames without an old-to-new mapping, `rg`
  reference checks before and after, generated-doc refresh, and validation.
- Preserve dirty user work and unrelated pending changes.
