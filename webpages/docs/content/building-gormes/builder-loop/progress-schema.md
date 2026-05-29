---
title: "Progress Schema"
weight: 60
aliases:
  - /building-gormes/progress-schema/
---

# Progress Schema

This page is generated from the Go progress model and validation rules.

<!-- PROGRESS:START kind=progress-schema -->
## Item Fields

| Field | Required when | Meaning |
|---|---|---|
| `name` | every item | Human-readable roadmap row name. |
| `status` | every item | `planned`, `in_progress`, or `complete`. |
| `priority` | optional | `P0` through `P4`. Item-level `P0` rows require contract metadata. |
| `contract` | active/P0 handoffs | The upstream behavior or Gormes-native behavior being preserved. |
| `contract_status` | contract rows | `missing`, `draft`, `fixture_ready`, or `validated`. |
| `slice_size` | contract rows and umbrella rows | `small`, `medium`, `large`, or `umbrella`. |
| `execution_owner` | contract rows and umbrella rows | `docs`, `gateway`, `memory`, `provider`, `tools`, `skills`, or `orchestrator`. |
| `trust_class` | active/P0 handoffs | Allowed caller classes: `operator`, `gateway`, `child-agent`, `system`. |
| `degraded_mode` | active/P0 handoffs | How partial capability is visible in doctor, status, audit, logs, or generated docs. |
| `fixture` | active/P0 handoffs | Local package/path/fixture set proving compatibility without live credentials. |
| `source_refs` | active/P0 handoffs | Docs or code references used to derive the contract. |
| `blocked_by` | optional | Roadmap rows or conditions blocking this slice. Requires `ready_when`. |
| `unblocks` | optional | Downstream rows enabled by this slice. |
| `ready_when` | contract rows and blocked rows | Concrete condition that makes the row assignable. |
| `not_ready_when` | umbrella rows, optional elsewhere | Conditions that make the row unsafe or too broad to assign. |
| `acceptance` | active/P0 handoffs | Testable done criteria. |
| `write_scope` | contract rows | Files, directories, or packages a builder skill may edit for this slice. |
| `test_commands` | contract rows | Commands that prove the slice without live provider or platform credentials. Required for skill-builder selection unless `no_test_required` is present. |
| `no_test_required` | rare testless contract rows | Explicit reason a row has no focused executable test command. Rows without `test_commands` or this field are not worker-ready. |
| `done_signal` | contract rows | Observable evidence that the row can move forward or close. |

## Meta Fields

| Field | Required when | Meaning |
|---|---|---|
| `meta.builder_loop.entrypoint` | skill handoff metadata is declared | Primary skill-routing entrypoint. Historical field name retained for schema compatibility. |
| `meta.builder_loop.plan` | skill handoff metadata is declared | Canonical completion plan for skill-driven work. |
| `meta.builder_loop.agent_queue` | skill handoff metadata is declared | Generated queue page for assignable rows. |
| `meta.builder_loop.progress_schema` | skill handoff metadata is declared | This schema reference. |
| `meta.builder_loop.candidate_source` | skill handoff metadata is declared | Canonical progress file consumed by skills. |
| `meta.builder_loop.unit_test` | skill handoff metadata is declared | Fast verification command for progress docs/schema behavior. |
| `meta.builder_loop.candidate_policy` | skill handoff metadata is declared | Shared selection rules used by builder skills. |

## Validation Rules

- `docs/data/progress.json` must not exist.
- if `meta.builder_loop` is declared, entrypoint, plan, candidate source, generated docs, unit test, and candidate policy must all be present.
- `in_progress` rows cannot use `slice_size: umbrella`.
- item-level `P0` and `in_progress` rows must include full contract metadata.
- contract rows must declare `slice_size`, `execution_owner`, `ready_when`, `write_scope`, `test_commands` (or explicit `no_test_required`), and `done_signal`.
- blocked rows must declare `ready_when`.
- `fixture_ready` rows must name a concrete fixture package or path.
- complete rows with contract metadata must use `contract_status: validated`.

## Planning Metrics

Progress is measured from derived status counts, not from free-form narrative.
`Progress.Stats()` walks phases, subphases, and items and tallies
`complete`, `in_progress`, and `planned`. A subphase is
`complete` only when every item is complete, `in_progress` when any
item has started, and `planned` when no item has started. README and the
architecture-plan index use those derived counts for shipped/subphase totals.

Future work is measured from contract-bearing rows. A row becomes assignable
when it is not `complete`, has no `blocked_by` dependency, is not
`slice_size: umbrella`, and declares the handoff fields builder skills need:
`source_refs`, `write_scope`, `test_commands` or `no_test_required`,
`acceptance`, `ready_when`, `not_ready_when`, and
`done_signal` whenever applicable. `agent-queue.md` is the
assignable-work view; `blocked-slices.md` is the deferred-work view; and
`umbrella-cleanup.md` is the work that must be split before assignment.

Planner quality is measured by reducing ambiguity for builder skills:
exact upstream refs, local file paths, fixture names, validation commands,
dependency edges, and degraded-mode behavior count as useful planning;
generic notes without bounded tests or write scope do not.

## Generated Agent Surfaces

- `docs/content/building-gormes/builder-loop/builder-loop-handoff.md` lists shared skill entrypoint, plan, candidate source, generated docs, test command, and candidate policy.
- `docs/content/building-gormes/builder-loop/agent-queue.md` lists only unblocked, non-umbrella contract rows with owner, size, readiness, degraded mode, fixture, write scope, test commands or a no-test-required reason, done signal, acceptance, and source references.
- `docs/content/building-gormes/builder-loop/blocked-slices.md` keeps blocked rows out of the execution queue while preserving their unblock condition.
- `docs/content/building-gormes/builder-loop/umbrella-cleanup.md` lists broad inventory rows that must be split before assignment.
- `docs/content/building-gormes/modules/` contains generated module-scoped roadmap review pages. These are views over the single logical backlog, not side queues.

## Good Row

```json
{
  "name": "Provider transcript harness",
  "status": "planned",
  "priority": "P1",
  "contract": "Provider-neutral request and stream event transcript harness",
  "contract_status": "fixture_ready",
  "slice_size": "medium",
  "execution_owner": "provider",
  "trust_class": ["system"],
  "degraded_mode": "Provider status reports missing fixture coverage before routing can select the adapter.",
  "fixture": "internal/llm/testdata/provider_transcripts",
  "source_refs": ["docs/content/upstream-hermes/source-study.md"],
  "ready_when": ["Anthropic transcript fixtures replay without live credentials."],
  "write_scope": ["internal/llm/"],
  "test_commands": ["go test ./internal/llm -count=1"],
  "done_signal": ["Provider transcript replay passes from captured fixtures."],
  "acceptance": ["All provider transcript fixtures pass under go test ./internal/llm."]
}
```

## Bad Row

```json
{
  "name": "Port CLI",
  "status": "in_progress",
  "slice_size": "umbrella"
}
```

This is invalid because an active execution row cannot be an umbrella, and it
does not explain the contract, fixture, caller trust class, degraded mode, or
acceptance criteria.
<!-- PROGRESS:END -->
