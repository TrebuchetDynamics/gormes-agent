# Builder To Planner Handoff

Use this only after fresh `next-work` and `next-work --repo-only` selectors do
not return a buildable repo row, or when the selected row has a precise contract
defect. The builder must stop runtime edits before emitting it.

```text
decision: plan
selector_output: <exact fresh repo-only output>
selected_row: <name, or none>
repair_reason: <no row | stale classification | missing field | broad scope | blocker>
candidate_atom_or_row: <exact evidence entry/row, or unknown>
source_refs: <upstream symbols/tests already verified>
gormes_insertion_point: <current package/public boundary, if known>
first_red_fixture: <smallest hermetic observable failure, if known>
required_row_fields: <specific missing/incorrect fields>
exclusions: <adjacent behavior that must remain separate>
planner_action: <one concrete refine/split/create action>
```

Rules:

- Preserve exact selector output; do not paraphrase `decision=build` as blocked.
- Do not invent upstream refs or an insertion point. Use `unknown` and let the
  planner run evidence discovery.
- Prefer one already-proven `partial`/`missing` atom over a broad parity sweep.
- If the packet already names exact source refs, insertion point, exclusions,
  and RED fixture, planner should shape the row directly without routing
  through `gormes-progress-slicer`.
- The packet is a handoff, not a side backlog; implementation intent belongs in
  the logical progress backlog once planner accepts it.

## Pinned Scenario

Fresh repo-only selection says `decision=plan`. The webhook route-filter atom
is `partial`; Hermes names `_load_filter_file_values`; current Go filters omit
`in_file`. Builder emits one packet for a profile-backed `in_file` row and
excludes route scripts. It does not edit runtime or create a TODO file.
