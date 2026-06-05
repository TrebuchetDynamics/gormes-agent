---
name: cmd-internal-refactor
description: Move one bounded cmd/gormes command domain out of root files toward root main.go only. Use when asked to thin cmd/gormes, burn down loose root files, split domain_*.go files, or run cmd-internal refactor without CLI behavior changes.
---

# cmd/gormes Domain Folder Refactor

## Mission

Gormes is the Go port of Hermes. This skill is not a generic `cmd/` cleanup
workflow: every extraction must keep Gormes moving toward source-backed Hermes
behavior while making the root `cmd/gormes` package thinner and more testable.

The target topology is root `cmd/gormes/main.go` only. Cobra command
construction, flags, args, help text, command registration, and CLI/golden tests
move to `internal/platform/cli/gormescli`; command-domain behavior and behavior
tests move to `internal/app/<domain>`; reusable runtime behavior stays in deeper
`internal/<runtime-domain>` packages. Existing `cmd/gormes/<domain>` packages
created by earlier passes are migration candidates, not the desired endpoint.

## Quick start

Select exactly one bounded command domain from the request and repository
evidence. First study `hermes-knowledge-graph.json` for the matching Hermes
command/domain contract. Then read the topology reference, characterize the
local CLI boundary, classify every related root file, record the root-file
burn-down count, move Cobra/CLI contract code into `internal/platform/cli/gormescli`,
and move command-local behavior into `internal/app/<domain>`.

## Entry protocol

- Confirm the checkout is on `development`; do not create branches or worktrees.
- Check `git status --short --branch --untracked-files=all` and preserve
  unrelated dirty work.
- If the requested domain is ambiguous, do not stop to ask and do not use a
  fixed default. Build a candidate table from current `cmd/gormes` evidence and
  select the smallest safe domain.
- Before planning edits, study `hermes-knowledge-graph.json` at the repository
  root for the selected domain. Prefer targeted `jq`/`rg` queries over loading
  the whole graph, and record the Hermes node IDs/file paths that informed the
  CLI boundary.
- If local Go behavior conflicts with graph-backed Hermes behavior, preserve
  local behavior for this refactor and report the parity gap.
- If `hermes-knowledge-graph.json` is missing or unreadable, stop and report the
  blocker instead of proceeding from memory or only local Go code.
- If the request includes feature changes, split them out; this skill is
  refactor-only.

## Topology check

Read [domain-folder-topology.md](references/domain-folder-topology.md) after
selecting the domain. Enforce these core rules:

- root `cmd/gormes` owns only `main.go`, the executable shim needed for
  `go run ./cmd/gormes`; no other loose root files are accepted as the final
  topology;
- `internal/platform/cli/gormescli` owns Cobra command construction, flags,
  args, help text, command registration, exit-code wiring, CLI/golden tests,
  and root-command test helpers;
- `internal/app/<domain>` owns command-domain options/results, orchestration,
  command-local formatting/path/env helpers, and behavior tests;
- deeper `internal/` packages keep reusable gateway/channel/provider/session/
  persistence/TUI/tool runtime behavior;
- app and platform subpackages must not import root `cmd/gormes`;
- do not bulk-move files like `uninstall_*.go` by filename; classify each file
  by ownership first.

## Target shape

```text
cmd/gormes/
  main.go             executable shim only; delegates to gormescli

internal/platform/cli/gormescli/
  <domain>.go         Cobra command wiring, flags, args, help text
  <domain>_test.go   CLI contract/golden tests and command helpers

internal/app/<domain>/
  service.go          command-domain behavior/orchestration
  command.go          optional command builder/helper when useful
  types.go            public options/results
  *_test.go           behavior/unit/integration tests
```

## Domain selection

There are no default topics. For unspecified cmd-internal refactors, select one
domain by evidence rather than static order. `setup` is not special.

Selection heuristic:

- honor an explicitly named bounded domain when it is safe and refactor-only;
- otherwise scan `cmd/gormes` for at least three candidate domains with
  behavior-heavy command files and matching tests;
- prefer domains with readable Hermes graph refs and no unrelated dirty-worktree
  conflicts;
- prefer domains with no in-progress extraction; skip dirty domains unless the
  user explicitly asked to continue them;
- prefer the smallest extraction that thins root `cmd/gormes` while preserving
  the CLI contract;
- avoid domains that require live credentials, persistence migrations, gateway
  runtime changes, or multi-domain edits to validate.

Before selecting, write a compact candidate table with columns: domain, cmd
files, tests, Hermes graph refs, dirty-worktree/topology risk, extraction slice,
decision.

## Workflow

1. Select one bounded domain using the heuristic above.
2. Query `hermes-knowledge-graph.json` for matching Hermes command/domain files,
   symbols, and summaries.
3. List all related `cmd/gormes` files, including `<domain>_*.go` helpers and
   tests.
4. Identify the preserved CLI boundary: flags/args, env vars/config paths,
   stdout/stderr and JSON shapes, exit codes, help text, and error wording.
5. Add or find golden/contract tests before moving behavior.
6. Create or reuse `internal/platform/cli/gormescli` and move Cobra/CLI tests there.
7. Create or reuse `internal/app/<domain>` and move command-local behavior there.
8. Delete the moved root `cmd/gormes` files; do not leave wrapper files behind
   unless the selected slice is `main.go` itself.
9. Move behavior tests beside `internal/app/<domain>` and CLI contract tests
   beside `internal/platform/cli/gormescli`.
10. Run focused validation first, then broaden.
11. Re-scan `cmd/gormes`, update root-file burn-down evidence, and stop after
    reporting next safe candidates. A new domain requires a new pass.
12. If committing, keep the commit scoped to this one topology/refactor slice
    and route final git delivery through `gormes-git` when appropriate.

## Verification gate

Run the narrowest useful tests, normally:

```bash
go test ./cmd/gormes -count=1
go test ./internal/app/<domain> -count=1
go test ./cmd/gormes/... -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

If a command is skipped, record why and what evidence replaces it.

## Stop conditions

Stop and report a blocker before continuing when preserving the CLI contract
requires behavior changes, tests need live credentials/user state, more than one
domain must move, public config/persistence/provider/gateway/TUI contracts would
change, the subpackage would import root `cmd/gormes`, or unrelated dirty files
make validation/commit scope unsafe.

## Output contract

Final response must include the selected domain, folder created/reused and moved
files, Hermes graph refs studied, preserved CLI boundary evidence, root-file
burn-down before/after counts, files left in root/deeper packages, tests added
or moved, validation results, skipped validation with reasons, and confirmation
whether root `cmd/gormes` is complete (`main.go` only) or still incomplete.
Before claiming completion, run `folder_refactor_audit` when available and
classify every remaining root basename as the `main.go` facade, intentionally
out of scope, or a next move/extraction candidate.

## References

- [Domain folder topology](references/domain-folder-topology.md)
