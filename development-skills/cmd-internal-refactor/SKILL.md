---
name: cmd-internal-refactor
description: Refactor one bounded cmd/gormes command domain into an internal app package. Use when asked to run cmd-internal refactor, thin cmd/gormes, or move command behavior behind an internal app service without CLI behavior changes.
---

# cmd-internal Refactor

## Mission

Gormes is the Go port of Hermes. This skill is not a generic `cmd/` cleanup workflow: every extraction must keep Gormes moving toward source-backed Hermes behavior while making `cmd/gormes` thinner and more testable.

## Quick start

Select exactly one bounded command domain from the request and repository evidence; do not rely on a fixed default topic, and do not treat `setup` as special. First study `hermes-knowledge-graph.json` for the matching Hermes command/domain contract, because Gormes' goal is to port Hermes to Go. Then characterize local CLI behavior and move only that domain's behavior into `internal/app/<domain>` while keeping `cmd/gormes` as Cobra wiring.

## Entry protocol

- Confirm the checkout is on `development`; do not create branches or worktrees.
- Check `git status --short --branch --untracked-files=all` and preserve unrelated dirty work.
- If the requested domain is ambiguous, do not stop to ask and do not use a fixed default. Build a short candidate table from current `cmd/gormes` evidence and select the smallest safe command domain: prefer a domain with clear Hermes graph refs, existing characterization tests, low dirty-worktree conflict risk, and behavior that can move without changing public CLI output. Record the candidate table and selection rationale.
- Before planning edits, study `hermes-knowledge-graph.json` at the repository root for the selected domain. Prefer targeted `jq`/`rg` queries over loading the whole 18MB file, and record the Hermes node IDs/file paths that informed the CLI boundary.
- Treat the graph as required Hermes-port evidence. If local Go behavior conflicts with graph-backed Hermes behavior, preserve local behavior for this refactor and report the parity gap; do not silently encode the drift as the new design.
- If `hermes-knowledge-graph.json` is missing or unreadable, stop and report the blocker instead of proceeding from memory or only local Go code.
- If the request includes feature changes, split them out; this skill is refactor-only.

## Rules

- One domain per PR/commit.
- No feature changes, broad cleanup, or unrelated formatting.
- No output changes unless an existing test is intentionally updated with explicit graph-backed Hermes evidence.
- Preserve CLI flags, args, help text, env vars, config paths, stdout/stderr, exit codes, JSON output, and error wording.
- Keep existing tests passing before and after.
- Do not move gateway/channel/provider/runtime behavior that belongs in an existing deeper package unless `internal/app/<domain>` is only orchestrating it.

## Target shape

```text
cmd/gormes/
  <domain>.go          thin cobra command wiring only
  <domain>_test.go    CLI contract tests only

internal/app/<domain>/
  service.go          domain behavior
  command.go          optional cobra command builder if useful
  types.go            public options/results
  *_test.go           behavior/unit/integration tests
```

## Domain selection

There are no default topics. For unspecified cmd-internal refactors, the skill must select one domain by evidence rather than by a static order. `setup` is not the default and must not be selected merely because it appears first, has an existing partial extraction, or was used in earlier examples.

Selection heuristic:

- honor an explicitly named bounded domain when it is safe and refactor-only;
- otherwise scan `cmd/gormes` for at least three candidate domains with behavior-heavy command files and matching tests;
- prefer domains with readable `hermes-knowledge-graph.json` refs and no unrelated dirty-worktree conflicts;
- prefer domains with no in-progress extraction or unrelated dirty files; if a domain already has dirty edits, skip it unless the user explicitly asked to continue that domain;
- prefer the smallest extraction that can make `cmd/gormes` thinner while preserving the CLI contract;
- avoid domains that require live credentials, persistence migrations, gateway runtime changes, or multi-domain edits to validate.

Before selecting, write a compact candidate table with columns: domain, cmd files, tests, Hermes graph refs, dirty-worktree risk, extraction slice, decision. Record the selected domain and the evidence that made it safer than nearby candidates in the final response.

## Workflow

1. Select one bounded domain using the evidence-based selection heuristic above; for ambiguous requests, compare at least three candidates before choosing.
2. Query `hermes-knowledge-graph.json` for matching Hermes command/domain files, symbols, and summaries; keep the study targeted to the selected domain and save the relevant source-reference paths in notes or the final report. Use the graph to understand the Hermes behavior Gormes is porting, not just to justify a local package move.
3. List all `cmd/gormes` files related to that domain.
4. Identify the CLI boundary:
   - flags and args;
   - env vars and config paths;
   - stdout/stderr and JSON shapes;
   - exit codes, help text, and error wording.
5. Add or find golden/contract tests before moving behavior.
6. Move pure behavior into `internal/app/<domain>`.
7. Leave `cmd/gormes` as wrapper/Cobra wiring only.
8. Move behavior tests beside `internal/app/<domain>`.
9. Keep CLI compatibility tests in `cmd/gormes`.
10. Run focused validation first, then broaden.
11. If committing, keep the commit scoped to this one domain and route final git delivery through `gormes-git` when appropriate.

## Verification gate

Run the narrowest useful tests, normally:

```bash
go test ./cmd/gormes -count=1
go test ./internal/app/<domain> -count=1
go test ./...
go vet ./...
git diff --check
```

If a command is skipped, record why and what evidence replaces it.

## Stop conditions

Stop and report a blocker before continuing when:

- preserving the CLI contract requires a behavior change;
- tests cannot characterize the domain without live credentials or user state;
- more than one domain must move to keep the build green;
- the extraction would change public config, persistence, provider, gateway, or TUI contracts;
- unrelated dirty files make validation or commit scope unsafe.

## Output contract

Final response must include:

- selected domain and moved files;
- `hermes-knowledge-graph.json` source refs studied for the domain;
- preserved CLI boundary evidence;
- tests added or moved;
- validation commands and results;
- any skipped validation with reason;
- confirmation that `cmd/gormes` is thinner and `internal/app/<domain>` owns behavior.
