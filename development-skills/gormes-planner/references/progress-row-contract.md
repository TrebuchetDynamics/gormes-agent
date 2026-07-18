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

Allowed schema values must come from `internal/planning/progress`, not planner
invention:

- `trust_class`: `operator`, `gateway`, `child-agent`, `system`.
- `execution_owner`: `docs`, `gateway`, `memory`, `provider`, `tools`,
  `skills`, `orchestrator`, `tui`, or `goncho`.
- `contract_status`: `missing`, `draft`, `fixture_ready`, or `validated`.
- `slice_size`: `small`, `medium`, `large`, or non-executable `umbrella`.

A repo builder row must keep every `write_scope` path under the current repo.
External paths, placeholders such as `(separate repo)`, or optional alternatives
make the row non-repo-scoped until split.

## Security-Sensitive Row Readiness

Rows that read files, resolve profile paths, execute processes, make outbound
requests, expose secrets, or mutate durable state stay `draft` until policy is
explicit. Put unresolved choices in `not_ready_when`; use
`gormes-interface-designer` when the caller-facing API is unclear.

Before `fixture_ready`, the row must state as applicable:

- who controls the config/input and its `trust_class`;
- injected root/home and allowed relative, absolute, tilde, and environment
  expansion forms;
- traversal and symlink policy, file type/size/encoding limits, and whether
  reads may escape the injected root;
- executable allow/root policy, argv (no shell interpolation), cwd,
  environment scrubbing, timeout, output cap, and cancellation;
- network allowlist/redirect/timeout/body-cap policy;
- secret redaction and durable-write/atomicity behavior;
- fail-closed degraded evidence and hermetic `t.TempDir`/fake-transport
  fixtures, including escape, missing, malformed, and oversized cases.

Do not use a developer's live `~/.gormes`, `~/.hermes`, credentials, network,
or process environment as the fixture. If compatibility requires a broader
upstream policy such as absolute paths, record the risk and owner decision
instead of silently narrowing or broadening it.

## Inventory And Umbrella Parent Rows

If you add `contract` or `contract_status` to a broad `slice_size: umbrella`
row for source-backed inventory continuity, the progress validator treats it as
a contract row. Include the same schema-completeness fields even though it is
not builder-selectable:

- `ready_when`: say the parent is inventory-only and child rows remain present.
- `not_ready_when`: explicitly block assigning the umbrella as one combined
  implementation slice.
- `write_scope`: normally just `webpages/docs/content/building-gormes/architecture_plan/progress.json`
  unless generated docs or site data are expected.
- `no_test_required`: explain that executable child rows carry the tests.
- `done_signal`: state that the parent remains non-selectable and source-backed
  while child rows provide runnable implementation and tests.

Do not add a contract to a parent row without these fields; `go run
./cmd/progress write` will fail before generating docs.

When a row comes from the Hermes/Honcho feature map, include the map section or
anchor in `source_refs`, `fixture`, or `note`; `feature_map_anchor` is a useful
planning term but is not a schema field unless the schema adds it.

Do not use broad wildcards such as `agent/**`, `tools/**`, `gateway/**`,
`hermes_cli/**`, `src/**`, `sdks/**`, `mcp/**`, or `_handle_*_command` as the
only evidence for a feature-bearing row. Broad source classes are acceptable as
companion context only when the row also cites exact files, symbols, commands,
fixtures, or tests.

For CLI/config/migration rows, keep the contracts distinct:
`Hermes CLI command-tree parity manifest` inventories commands,
`gormes config migrate` updates only native Gormes schema/defaults, and
`gormes migrate hermes` / `gormes migrate openclaw` import external state.
Do not merge these into one broad config row. If `ooenclaw` appears in a
request, represent it as a tested typo-suggestion path for
`gormes migrate openclaw`, not a second migration command, unless a dedicated
compatibility row explicitly owns that alias.

For full-map or parity-register rows, cite exact nested upstream refs when
symbol-level mapping is feasible. If the subsystem is too large for symbol
proof, explicitly classify it as `mapped-by-contract`, `owned`, `excluded`, or
`still row-backed` in `contract`, `acceptance`, or `note`; classification is
not currently a schema field. Do not leave `unknown/gap` in a completed
planning row.

When a row is `owned` or `excluded`, the same row must record three things or
it must be downgraded to `still row-backed` and split:

1. the upstream behavior in `source_refs`,
2. the divergence rationale in `contract`, `acceptance`, or `note`, and
3. observable proof — either a Go test name, a fixture path, or a
   `no_test_required` reason — pinned in `test_commands` (or `note`).

When a row is `mapped-by-contract`, the row must name the fake-transport
fixture, golden file, or compatibility harness that stands in for the
upstream symbol; otherwise downgrade to `still row-backed`.

When a row originates in a swarm pass, prefix the swarm origin in `note`
(e.g. `swarm_origin: 2026-04-27/gap-classifier`) so reconcilers can bisect
drift between the feature map, this register, and the runtime plan.

When splitting a row, copy the parity classification, rationale, replacement
test, fixture pointer, and swarm origin onto each child row before narrowing
them. Do not copy `health` (see Health Preservation below); the new rows
start without health evidence and earn it from runtime.

Fix stale upstream refs in the same planner pass that discovers them. Docs-only
map rows may be `complete`/`validated` only after docs link coverage, progress
validation, and the relevant docs tests pass.

## Health Preservation

Never edit an existing row's `health` block by hand. It is historical runtime
evidence and must be preserved verbatim when reshaping a row. If splitting a
row, the new rows naturally start without the old health block.

## Split-Layout Mutation Receipt

Before a typed save, record `git status --short`. Use
`internal/planning/progress.Load` / `SaveProgress`; do not decode or edit module
JSON directly. After saving:

1. inspect changed `progress.json/modules/*` files;
2. keep the intended module and any module already dirty before the pass;
3. restore only incidental serialization changes to modules that were clean and
   unrelated before the pass;
4. run `go run ./cmd/progress write` and validate;
5. when totals changed, update
   `architecture_plan/completion-plan.md` and run
   `TestCompletionPlanCurrentFinishLedgerMatchesProgress`.

Never use a broad checkout/reset command over pre-existing dirty planning work.

## No Parallel Backlog

Do not create sidecar TODO lists, private prompts, issue lists, or hand-written queues. All implementation intent goes into `progress.json` and generated docs.
