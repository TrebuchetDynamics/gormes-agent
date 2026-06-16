# Bug-Finding During cmd/gormes Refactors

Use this reference after selecting one `cmd/gormes` command domain. The goal is to make refactors expose bugs while preserving behavior unless the refactor introduced the regression.

## Target

One selected command-domain extraction from root `cmd/gormes` into `internal/platform/cli/gormescli`, `internal/app/<domain>`, or deeper reusable packages.

## Baseline oracle

Before moving behavior, choose at least one oracle that can fail for the selected contract:

```bash
# Focused tests if they already exist.
go test ./cmd/gormes -run '<Domain|Command>' -count=1

# App/CLI packages when the domain already has internal owners.
go test ./internal/app/<domain> -count=1
go test ./internal/platform/cli/gormescli -run '<Domain|Command>' -count=1

# CLI smoke when output/help/JSON/error behavior is the contract.
GORMES_HOME="$(mktemp -d)" go run ./cmd/gormes <command> --help
```

Capture the exact command, selector, and result. If the selector matches no tests, it is not a behavior oracle; run the package without `-run` or add a characterization test.

## Common refactor bug traps

Look for these before and after the move:

- lost Cobra flags, aliases, defaults, annotations, examples, shell completion, or help wording;
- command factory freshness bugs where persistent flags, writers, or seams leak between tests;
- env/home/profile paths resolved at package init instead of command execution;
- stdout/stderr writers, color settings, JSON output, or error wording routed to the wrong stream;
- prompts, TTY checks, or noninteractive paths accidentally requiring live input;
- nil/default seams not copied from root factories into `gormescli` facades;
- app packages importing `cmd/gormes`, `gormescli`, or platform CLI packages;
- stale root symbols/wrappers hiding dead behavior after the move;
- tests weakened because moved code exposed an old bad assumption.

Useful stale-symbol check:

```bash
rg -n '<movedSymbol|oldRootHelper|newFacade>' cmd/gormes internal/platform/cli/gormescli internal/app/<domain>
```

## Triage findings

Use explicit labels in the final report:

- **none**: the before/after oracle and focused gates agree.
- **preexisting bug**: the baseline already fails or a characterization test reveals behavior that was already wrong. Preserve current behavior for the refactor and hand off to `gormes-tdd-slice` unless the user explicitly changes scope.
- **introduced regression**: the baseline passes before the move and fails after it. Fix within the selected refactor slice before claiming done.
- **parity drift**: Gormes behavior differs from graph-backed Hermes evidence. Preserve local behavior during the refactor and report the parity gap for planner/parity routing.

Do not convert a preexisting bug into an unreviewed behavior fix just because the files are open.

## Fix

For introduced regressions only:

1. restore the lost contract in the new owner package;
2. keep or add the characterization test that caught it;
3. delete obsolete root wrappers only after all callers use the new facade;
4. rerun the same oracle that failed, then the package/topology closeout gates.

If the fix requires changing public output, config, persistence, provider/gateway/TUI behavior, or another command domain, stop and hand off instead.

## Verify

A refactor closeout should include:

```text
bug oracle before: <command/test> -> <result>
bug oracle after:  <same command/test> -> <result>
bug findings: none|preexisting|introduced|parity-drift, with evidence
focused gates: <internal app/CLI/root tests>
topology gate: go test ./internal/support/repochecks -run 'Cmd|Internal|Topology|Import' -count=1
```

Then run `go run ./cmd/progress validate` and `git diff --check` unless unrelated dirty work makes those results misleading; if skipped, say why.
