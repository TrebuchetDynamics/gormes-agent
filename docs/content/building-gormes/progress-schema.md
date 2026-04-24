---
title: "Progress Schema"
weight: 38
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

## Validation Rules

- `docs/data/progress.json` must not exist.
- `in_progress` rows cannot use `slice_size: umbrella`.
- item-level `P0` and `in_progress` rows must include full contract metadata.
- contract rows must declare `slice_size`, `execution_owner`, and `ready_when`.
- blocked rows must declare `ready_when`.
- `fixture_ready` rows must name a concrete fixture package or path.
- complete rows with contract metadata must use `contract_status: validated`.

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
  "fixture": "internal/hermes/testdata/provider_transcripts",
  "source_refs": ["docs/content/upstream-hermes/source-study.md"],
  "ready_when": ["Anthropic transcript fixtures replay without live credentials."],
  "acceptance": ["All provider transcript fixtures pass under go test ./internal/hermes."]
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
