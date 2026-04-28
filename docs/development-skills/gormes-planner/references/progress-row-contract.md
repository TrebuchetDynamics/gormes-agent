# Progress Row Contract

Planner output must make builder work obvious.

## Required For Executable Rows

Use these fields when the schema allows them:

- `name`: concrete behavior, not a theme.
- `status`: normally `planned`; use `complete` only with repository evidence and validation.
- `execution_owner`: use an existing owner class such as `docs`, `gateway`, `memory`, `provider`, `tools`, `skills`, or `orchestrator`.
- `slice_size`: `small`, `medium`, or `large`; avoid `umbrella` for executable work.
- `contract`: one paragraph naming the behavior to implement.
- `contract_status`: `draft` when any prerequisite is uncertain; otherwise use the repo's established ready status.
- `ready_when`: objective prerequisites.
- `not_ready_when`: blockers that should prevent autonomous selection.
- `blocked_by` / `unblocks`: use for dependency order.
- `write_scope`: exact files/directories the builder may edit.
- `source_refs`: upstream and Gormes references, preferably exact file paths and symbols.
- `fixture`: fixture name/path when possible.
- `test_commands`: focused commands the builder must run, or `no_test_required` with a concrete reason.
- `acceptance`: observable behavior that proves done.
- `done_signal`: final evidence the builder should report.
- `provenance`: `upstream`, `gormes`, or `hybrid`; avoid fake upstream refs.

When a row comes from the Hermes/Honcho feature map, include the map section or
anchor in `source_refs`, `fixture`, or `note`; `feature_map_anchor` is a useful
planning term but is not a schema field unless the schema adds it.

## Health Preservation

Never edit an existing row's `health` block by hand. It is historical runtime
evidence and must be preserved verbatim when reshaping a row. If splitting a
row, the new rows naturally start without the old health block.

## No Parallel Backlog

Do not create sidecar TODO lists, private prompts, issue lists, or hand-written queues. All implementation intent goes into `progress.json` and generated docs.
