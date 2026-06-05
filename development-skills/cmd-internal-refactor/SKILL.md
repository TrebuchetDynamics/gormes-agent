---
name: cmd-internal-refactor
description: Move one bounded cmd/gormes command domain out of root files toward root main.go only. Use when thinning cmd/gormes, burning down loose root files, or preserving CLI behavior during cmd refactors.
---

# cmd/gormes Internal Refactor

## Quick start

Move exactly one `cmd/gormes` command domain without behavior changes. Confirm `development`, preserve dirty work, scan every root `cmd/gormes` entry, study `hermes-knowledge-graph.json`, read the topology reference, move CLI wiring to `internal/platform/cli/gormescli`, move command behavior to `internal/app/<domain>`, validate, then stop.

## Mission

Thin `cmd/gormes` toward this final topology:

- `cmd/gormes/main.go`: executable shim and root factory delegation only. Other tracked root entries, including docs such as `codemap.md`, must be rehomed or explicitly classified before final topology claims.
- `internal/platform/cli/gormescli`: Cobra wiring, flags, args, help, command registration, exit-code seams, CLI/golden tests, root test helpers.
- `internal/app/<domain>`: command-local behavior, options/results, formatting/path/env helpers, behavior tests.
- deeper `internal/<runtime-domain>` packages: reusable gateway, provider, session, persistence, memory, TUI, channel, and tool runtime behavior.

This is refactor-only. If a slice needs behavior changes, split them out and route through planner/builder/TDD.

## Entry protocol

- Confirm branch is `development`; do not create branches or worktrees.
- Run `git status --short --branch --untracked-files=all`; preserve unrelated dirty work.
- If available, run `folder_refactor_scan` on `cmd/gormes` before planning; otherwise list root files and subdirectories explicitly.
- Study `hermes-knowledge-graph.json` for the selected domain; record exact node IDs/file paths. If the graph is missing or unreadable, stop.
- If local Go behavior conflicts with graph-backed Hermes behavior, preserve local behavior for this refactor and report the parity gap.
- If the user or prior handoff supplies a candidate table, recheck it against the current tree and select only the first safe bounded domain.
- If the domain is ambiguous, build a candidate table from evidence; do not use a fixed default.

Candidate table columns:

```text
domain | cmd files/tests | Hermes graph refs | dirty/topology risk | extraction slice | decision
```

## Domain selection

Prefer, in order:

1. explicitly named bounded refactor-only domain;
2. when a broad relocation is already dirty, one already-in-progress dirty domain that is small and independently validatable;
3. a tests-only domain when behavior/wiring already lives under `internal/` and the slice burns down root CLI or behavior tests;
4. otherwise, one clean behavior-heavy domain with matching tests and readable Hermes graph refs.

Skip domains needing live credentials, persistence migrations, gateway runtime changes, public contract changes, or multi-domain edits to validate.

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
2. Query `hermes-knowledge-graph.json` for matching Hermes command/domain files, symbols, and summaries.
3. Preserve command names, aliases, flags, defaults, args, help, env/config paths, stdout/stderr, JSON shape, prompts, exit codes, and error wording.
4. Find or add characterization tests before moving code.
5. Move Cobra wiring and CLI contract tests to `internal/platform/cli/gormescli`.
6. Move command-local behavior and behavior tests to `internal/app/<domain>`.
7. Leave reusable runtime behavior in deeper `internal/` packages.
8. Update root `main.go` factories/seams only; delete moved loose root files instead of leaving wrappers.
9. Run focused validation plus `go test ./internal/support/repochecks -run 'Cmd|Internal|Topology|Import' -count=1`.
10. Stop after one domain. Report next safe candidates; a new domain requires a new pass.

## Verification gate

Use focused gates from the topology reference first, then `go run ./cmd/progress validate` and `git diff --check`. Full `go test ./...` / `go vet ./...` is closeout-only when unrelated dirty/concurrent work cannot invalidate the result. Never claim full-repo green when broad unrelated dirty work is active.

## Red lines

Stop when preserving the CLI contract requires behavior changes, tests need live credentials/user state, more than one domain must move, public config/persistence/provider/gateway/TUI contracts would change, a subpackage would import root `cmd/gormes`, an app→CLI dependency cycle appears, Hermes graph evidence is missing, or unrelated dirty files make validation/commit scope unsafe.

## Output contract

Report selected domain, moved/created/reused files, Hermes graph refs, preserved CLI boundary, root-file burn-down before/after, remaining root/deeper file classifications, dependency-cycle/import-budget notes, tests added or moved, validation results, skipped validation with reasons, and whether root `cmd/gormes` is complete (`main.go` only) or still incomplete.

Before claiming complete topology, run `folder_refactor_audit` when available and classify every remaining root basename as `main.go` facade, intentionally out of scope, or next move/extraction candidate.

## References

- [Domain folder topology](references/domain-folder-topology.md)
