---
title: "Agent Operating Model"
weight: 17
---

# Agent Operating Model

This page is the day-to-day operating model for Codex, Claude, claudeu,
codexu, and future backends working on Gormes.

Every substantial pass starts with a repo-local skill and ends with evidence.

## Skill Selection

Use `gormes-skill-manager` when the correct skill is not obvious. Otherwise
select the narrowest skill that matches the pass:

| Pass | Skill |
|---|---|
| Decide direction with the operator | `grill-me` |
| Route substantial work or improve repo-local skills | `gormes-skill-manager` |
| Reconcile hidden Hermes/Gormes UX details or recurring parity reports | `gormes-hermes-parity` |
| Audit upstream parity | `gormes-parity-auditor` |
| Refine phases, rows, docs, or dependency order | `gormes-planner` |
| Validate installer, `go run`, local binary, PATH, runtime home, gateway owner, or `sessions.db` behavior | `gormes-dev-runtime` |
| Compare package/API shapes | `gormes-interface-designer` |
| Implement a `progress.json` row | `gormes-builder` |
| Drive one behavior through red-green-refactor | `gormes-tdd-slice` |
| Create or improve skills | `gormes-skill-manager` plus skill-creation workflow |

Do not load every skill. A good pass normally uses one skill; a complex pass
uses at most three.

For live dogfood reports, route the surface first. Installer/PATH/home/session
ownership bugs use `gormes-dev-runtime`; visible Hermes-vs-Gormes UX bugs use
`gormes-hermes-parity`; implementation of one proven behavior uses
`gormes-tdd-slice`. Do not let a stale installed binary or wrong `GORMES_HOME`
turn into speculative TUI/provider work.

## Planner Pass

A planner pass must:

1. name the lane and subsystem;
2. inspect upstream and current Gormes code;
3. classify gaps as covered, planned, vague, missing, or owned;
4. update docs or `progress.json` only where needed;
5. preserve `health` blocks;
6. validate docs/progress;
7. stop with a short next-builder-row list.

Planner passes do not implement runtime feature code.

`gormes-planner` replaces the deleted `cmd/planner-loop` command. During a
planner-skill pass, do not recreate or invoke loop binaries. Manual repository
evidence wins over stale loop assumptions.

Minimum validation:

```sh
go run ./cmd/progress validate
go test ./docs -count=1
```

If progress rows or generated progress docs changed:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
go test ./internal/progress -count=1
```

Planner output packet:

| Field | Required content |
|---|---|
| Lane/subsystem | One lane and one subsystem only. |
| Evidence | Upstream paths, Gormes paths, tests, and current progress rows. |
| Classification | Covered, planned, vague, missing, or owned for each behavior. |
| Row change | Exact row additions/splits/refinements, or explicit "docs-only". |
| Builder handoff | Next one to three rows, write scope, and test commands. |
| Validation | Commands run and failures, if any. |

Planner passes may update generated docs only through `go run ./cmd/progress
write`. They must not edit content between `PROGRESS` markers by hand.

When a planner pass adds a phase, subphase, or closure row, it must also update
the hand-written planning pages that explain why the structure exists:
`completion-plan.md`, `lane-roadmap.md`, and any subsystem study page that owns
the new rows. A phase restructure is not complete until Hugo-generated pages
and the human-readable plan tell the same story.

## Builder Pass

A builder pass must:

1. select exactly one row unless the user explicitly asks for a batch;
2. restate the contract, write scope, tests, and done signal;
3. write one failing behavior test where feasible;
4. implement the smallest green path;
5. repeat vertically only while scope remains the same row;
6. run row-local commands and focused package gates;
7. update docs/progress evidence when public behavior changed.

Builder passes do not invent backlog. If a row is vague, send it back through
`gormes-planner`.

Builder selection without loop binaries:

1. Read `progress.json` and the generated Agent Queue.
2. Prefer rows with `fixture_ready` or validated contracts, no `blocked_by`,
   non-umbrella `slice_size`, explicit `write_scope`, and `test_commands`.
3. Prefer Lane 1/2 closure rows unless the user names a row or a dependency
   requires another lane.
4. Restate the row. If the restatement cannot name behavior, tests, and done
   signal, stop and run `gormes-planner`.
5. Build one row only.

## Parity Audit Pass

A parity audit pass must compare exact upstream behavior against exact Gormes
evidence. It should not be a list of file names.

For each behavior, record:

- upstream path and symbol;
- Gormes package or missing package;
- tests/fixtures;
- progress row coverage;
- classification;
- recommended row if vague or missing.

For operator-visible parity, preserve the user's transcript or terminal output
before summarizing. Include extra hourglass/status messages, duplicate replies,
wrong product labels, raw tool-iteration errors, leaked tool-call text, and
which runtime surface was running. These details are the contract future tests
must freeze.

## Interface Design Pass

Use this before implementation when a package boundary is unclear.

Output at least two materially different designs and compare:

- caller simplicity;
- Hermes/Honcho compatibility;
- module depth;
- testability;
- migration cost;
- risk of shallow pass-through abstraction.

The result should either update a row or unblock one builder pass.

## Stop Conditions

Stop and report instead of continuing when:

- validation fails for a reason outside the current pass;
- progress row scope is missing or contradictory;
- upstream behavior cannot be verified locally;
- the pass would require runtime code during a planner pass;
- the builder would need to edit outside `write_scope`;
- live credentials are required for row-local proof.

## Evidence Format

Every final report should include:

1. skill used;
2. scope scanned;
3. files changed;
4. validation run;
5. next row or next pass;
6. blockers.

The user should be able to resume from the report without reading the entire
conversation.
