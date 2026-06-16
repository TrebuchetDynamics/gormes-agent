---
name: cmd-internal-refactor
description: Move one bounded cmd/gormes command domain out of root files toward root main.go only. Use when thinning cmd/gormes, burning down loose root files, preserving CLI behavior, or catching refactor regressions during cmd moves.
---

# cmd/gormes Internal Refactor

## Quick start

Move exactly one `cmd/gormes` command domain without behavior changes. A domain is not one file: when the user says "too slow", "refactor all", or supplies a file list, use accelerated same-domain mode to select the largest cohesive domain cluster from that list, move its directly owned behavior/tests/facades in one pass, validate once, then stop before a second domain. Start every move with a bug-finding oracle: capture the selected command's current observable behavior, add/keep characterization tests for risky seams, and rerun the same evidence after extraction.

## Mission

Thin `cmd/gormes` toward this final topology:

- `cmd/gormes/main.go`: executable shim and root factory delegation only. Other tracked root entries, including docs such as `codemap.md`, must be rehomed or explicitly classified before final topology claims.
- `internal/platform/cli/gormescli`: Cobra wiring, flags, args, help, command registration, exit-code seams, CLI/golden tests, root test helpers.
- `internal/app/<domain>`: command-local behavior, options/results, formatting/path/env helpers, behavior tests.
- deeper `internal/<runtime-domain>` packages: reusable gateway, provider, session, persistence, memory, TUI, channel, and tool runtime behavior.

This is refactor-only, not blind relocation. Use the move to expose regressions and preexisting defects, but preserve current Gormes behavior unless the refactor itself introduced the bug. If a slice needs intentional behavior changes, split them out and route through planner/builder/TDD.

## Entry protocol

- Resolve the repo root once (`repo=$(git rev-parse --show-toplevel)`) and run discovery from there; the current cwd may already be inside `cmd/gormes`.
- Confirm branch is `development`; do not create branches or worktrees.
- Run `git -C "$repo" status --short --branch --untracked-files=all`; preserve unrelated dirty work.
- If available, run `folder_refactor_scan` on the absolute `$repo/cmd/gormes` path before planning; otherwise list root files and subdirectories explicitly.
- Verify `hermes-knowledge-graph.json` exists and is readable, then query it only for the selected domain. If the graph is missing or unreadable, stop.
- For Hermes-backed domains, record exact node IDs/file paths. For Gormes-owned/no-Hermes-analogue domains, record the targeted zero/irrelevant graph query plus local evidence proving the no-analogue classification, such as repocheck refs, progress docs, command tests, or Gormes-owned code ownership.
- If local Go behavior conflicts with graph-backed Hermes behavior, preserve local behavior for this refactor and report the parity gap.
- Before editing selected-domain code, choose the before/after bug oracle: focused tests, CLI help/golden output, JSON smoke, noninteractive temp-home command, or app-level characterization. If no oracle exists, add one before moving behavior.
- If the user or prior handoff supplies a candidate table or file list, recheck it against the current tree and select the largest safe bounded domain cluster, not the smallest next file.
- If the domain is ambiguous, build a candidate table from evidence; do not use a fixed default. In continuation sessions, keep the table to only the supplied/listed candidates plus any directly referenced helper files.

Candidate table columns:

```text
domain | cmd files/tests | Hermes graph refs or Gormes-owned evidence | dirty/topology risk | extraction slice | decision
```

## Domain selection

Prefer, in order:

1. explicitly named bounded refactor-only domain;
2. accelerated same-domain mode when the user says "too slow", "refactor all", "keep going", or supplies a file list: pick the largest cohesive domain cluster whose behavior, facades, and directly owned tests can move together and validate together;
3. when a broad relocation is already dirty, one already-in-progress dirty domain that is independently validatable;
4. a tests-only domain when behavior/wiring already lives under `internal/` and the slice burns down root CLI or behavior tests;
5. a Gormes-owned/no-Hermes-analogue domain with a targeted graph query and strong local evidence;
6. otherwise, one clean behavior-heavy domain with matching tests and readable Hermes graph refs.

Skip domains needing live credentials, persistence migrations, gateway runtime changes, public contract changes, or multi-domain edits to validate.

## Accelerated same-domain mode

Use this mode to avoid slow one-file passes while still preserving the one-domain stop rule.

- Treat a command section/subflow as one domain cluster when all selected root files share the same user-visible command contract and focused tests can validate them together.
- Use the user's file list as the candidate universe; add only files referenced by selected-domain symbols/imports/tests. Do not wander into unrelated root domains just because they share a prefix.
- Move all directly owned root behavior, facades, marker tests, and behavior tests for that cluster in the same pass. Do not split behavior, facade, and tests into separate passes when they compile together.
- If a huge file such as `setup.go` or `setup_profiles_tui.go` contains multiple subflows, first carve out the selected subflow through narrow root adapters; do not attempt a whole-file migration unless every subflow in the file belongs to the selected domain.
- If the selected cluster proves too entangled, downgrade to the next largest same-domain tests-only or helper-only cluster from the supplied list and report the blocker.
- Run independent focused tests in parallel when the harness supports it; after a compile fix, rerun only the failed focused gate before the package/topology closeout.
- Stop after one selected domain cluster even if more listed files remain.

See [Acceleration guide](references/acceleration.md) for examples and anti-patterns.

## Bug-finding refactor oracle

Use the extraction to catch defects without smuggling in behavior changes.

- Establish a baseline before edits whenever the domain has behavior: run focused tests or capture help/output/JSON/error behavior with a temp `GORMES_HOME` and no live credentials.
- Add or keep characterization tests around bug-prone seams: flags/defaults, aliases, command freshness, env/home resolution, stdout/stderr writers, prompts/noninteractive mode, JSON fields, exit codes, and seam injection defaults.
- Treat findings explicitly: a baseline failure is a preexisting bug, a new after-move failure is an introduced regression to fix inside the refactor, and Hermes-vs-Gormes drift is a parity gap to report rather than silently change.
- Do not weaken tests, delete assertions, change golden output, or hide failures to make the move compile.

See [Bug-finding during refactor](references/bug-finding.md) for bug traps, triage, and verification examples.

## Topology contract

Read [domain-folder-topology.md](references/domain-folder-topology.md) after selecting the domain. Enforce:

```text
cmd/gormes/main.go -> internal/platform/cli/gormescli -> internal/app/<domain> -> deeper internal runtime packages
```

- App packages must not import `cmd/gormes`, `gormescli`, or other CLI platform packages.
- `gormescli` may import `internal/app/<domain>` and lower-level runtime packages.
- Prefer thin `gormescli` facades to reduce root direct-internal imports and avoid app↔CLI cycles.
- Existing `cmd/gormes/<domain>` subpackages are migration candidates, not the endpoint.
- Classify every related file by ownership; do not bulk-move by filename.
- Final root topology also excludes tracked non-Go root artifacts such as `codemap.md`; rehome/update documentation references before claiming `main.go` only.

## Workflow

1. Count/classify every current root `cmd/gormes` entry, then selected-domain files/tests.
2. Query `hermes-knowledge-graph.json` for matching Hermes command/domain files, symbols, and summaries; if none exist, classify the domain as Gormes-owned/no-Hermes-analogue only with explicit local evidence.
3. Preserve command names, aliases, flags, defaults, args, help, env/config paths, stdout/stderr, JSON shape, prompts, exit codes, and error wording.
4. Find or add characterization tests before moving code; include at least one oracle that can fail on lost flags/defaults/help/env/output/error behavior or stale seams.
5. Move Cobra wiring and CLI contract tests to `internal/platform/cli/gormescli`.
6. Move command-local behavior and behavior tests to `internal/app/<domain>`.
7. Leave reusable runtime behavior in deeper `internal/` packages.
8. Update root `main.go` factories/seams only; delete moved loose root files instead of leaving wrappers. Thin root adapters are allowed only when a larger root file still contains other domains.
9. Rerun the same bug oracle after the move, then run focused validation plus `go test ./internal/support/repochecks -run 'Cmd|Internal|Topology|Import' -count=1`.
10. Stop after one domain. Report deferred files as remaining candidates; a new domain requires a new pass.

## Verification gate

Use focused gates from the topology reference first, then `go run ./cmd/progress validate` and `git diff --check`. A passing gate must include the selected domain's before/after bug oracle when behavior moved; compile-only is insufficient unless the slice is tests-only or helper-only. Full `go test ./...` / `go vet ./...` is closeout-only when unrelated dirty/concurrent work cannot invalidate the result. Never claim full-repo green when broad unrelated dirty work is active.

## Red lines

Stop when preserving the CLI contract requires behavior changes, tests need live credentials/user state, more than one domain must move, public config/persistence/provider/gateway/TUI contracts would change, a subpackage would import root `cmd/gormes`, an app→CLI dependency cycle appears, the Hermes graph file is missing/unreadable, a graph-backed domain lacks graph evidence, a no-Hermes-analogue claim lacks local evidence, a preexisting bug fix would be needed to complete the refactor, tests/golden expectations must be weakened to pass, or unrelated dirty files make validation/commit scope unsafe.

## Output contract

Report selected domain, whether accelerated same-domain mode was used, moved/created/reused files, Hermes graph refs or Gormes-owned no-analogue evidence, preserved CLI boundary, root-file burn-down before/after, remaining root/deeper file classifications, dependency-cycle/import-budget notes, tests added or moved, bug findings classified as none/preexisting/introduced/parity-drift with evidence, validation results, skipped validation with reasons, deferred files from any supplied list, and whether root `cmd/gormes` is complete (`main.go` only) or still incomplete.

Before claiming complete topology, run `folder_refactor_audit` when available and classify every remaining root basename as `main.go` facade, intentionally out of scope, or next move/extraction candidate.

## References

- [Domain folder topology](references/domain-folder-topology.md)
- [Acceleration guide](references/acceleration.md)
- [Bug-finding during refactor](references/bug-finding.md)
