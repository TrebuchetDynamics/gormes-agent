---
title: "Parity Workflow"
description: "How contributors turn upstream behavior into Gormes changes without overclaiming."
weight: 30
---

# Parity Workflow

Gormes uses Hermes as the active behavior contract, but parity work must stay
bounded. A parity pass compares one upstream behavior family with the closest
Gormes runtime, docs, or roadmap surface, then classifies the result with
source-backed evidence.

Use the repo-local `gormes-hermes-parity` skill for this work.

## When to run it

| Prompt | What happens |
|---|---|
| `gormes-hermes-parity` | Auto-selects one risky behavior-fidelity pair from repository evidence. |
| `gormes-hermes-parity <topic>` | Treats the topic as a discovery seed, then expands to adjacent commands, config keys, docs, tests, and progress rows before choosing one bounded pair. |

Good topics include a command, channel, provider, docs page, installer path,
visible error string, setup flow, TUI behavior, or user transcript.

## Evidence boundary

Allowed evidence:

- upstream checkouts in this repo, especially `./hermes-agent`
- sibling upstream checkouts such as `../honcho` and `../gbrain`
- checked-in Gormes code, tests, docs, generated progress data, and fixtures
- sanitized transcripts explicitly provided by an operator

Do not inspect private home directories, live credentials, channel tokens,
memory stores, or session databases as parity evidence unless the selected row
is specifically a migration or runtime-home command.

## Baseline

Start every pass from the current branch and current source state:

```bash
git status --short --branch
pwd
git rev-parse --show-toplevel
which -a gormes || true
go run ./cmd/progress validate
git rev-parse --short HEAD
```

Resolve upstream references from local checkouts first:

```bash
test -f ./hermes-agent/hermes_cli/main.py && git -C ./hermes-agent rev-parse --short HEAD
test -d ../honcho && git -C ../honcho rev-parse --short HEAD
test -d ../gbrain && git -C ../gbrain rev-parse --short HEAD
```

## Classify the pair

Each pass should name the upstream behavior, the Gormes surface, and the target
parity definition.

| Label | Meaning |
|---|---|
| `strict` | Names, inputs, outputs, errors, side effects, and registration must match upstream. |
| `functional` | The operator-visible contract is preserved even if Go internals differ. This is the default. |
| `owned` | Gormes intentionally diverges or extends Hermes, with the reason documented. |
| `excluded` | The upstream behavior is not part of Gormes, with explicit rationale. |

Then classify implementation status:

| Status | Meaning |
|---|---|
| `covered` | Implemented, tested, and source-backed. |
| `planned` | Represented by a builder-ready row with acceptance and tests. |
| `vague` | A row exists but is too broad, ambiguous, or weakly sourced. |
| `missing` | No useful Gormes code or progress row exists. |
| `stale-upstream` | Existing evidence points at old upstream behavior. |
| `blocked` | A dependency, source checkout, credential, or interface decision is absent. |

## Documentation rule

Public docs use the runtime-support labels from
[Current Status](../../parity/current-status/):

- `runtime-ready`
- `fixture-backed`
- `row-backed`
- `planned`
- `unverified`

Do not treat `progress.json` validation as proof that a product feature is
finished. It proves only that roadmap data is structurally valid. Update user
docs only for behavior that is actually available, and label fixture-backed or
row-backed behavior as such.

## Output format

Finish each pass with a compact report:

```text
scope:
source_shas:
upstream_refs:
gormes_refs:
evidence_boundary:
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

If implementation is needed, hand off a small task packet instead of a loose
TODO:

```text
task:
scope:
progress_row:
recommended_skill_chain:
source_refs:
visible_artifacts:
write_scope:
red_test_hint:
validation:
blocked_by:
```

Runtime implementation normally goes through `gormes-builder` or
`gormes-tdd-slice` after the parity pass has identified the contract.
