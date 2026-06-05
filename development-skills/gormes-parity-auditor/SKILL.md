---
name: gormes-parity-auditor
description: Map missing Hermes-in-Go parity. Use for upstream Hermes/Honcho feature audits, Goncho compatibility gaps, and source-backed progress-row evidence before builder work.
---

# Gormes Parity Auditor

## Repository Branch Rule

For Gormes work, stay on the existing `development` branch. Do not create or
use feature branches, short-lived branches, or git worktrees. If the checkout
is not on `development`, stop before editing and switch safely or report the
blocker.

## Mission

Find what prevents Gormes from being Hermes in Go, with Goncho as the Honcho-compatible Go port. Output builder-ready gaps, not a vague research report.

Hermes Agent is the Python upstream/father implementation for Gormes. Prefer
the in-repo checkout at `./hermes-agent` and fall back to `../hermes-agent`
only when it is absent. Resolve it as `$HERMES_SRC` before citing source refs.
It is behavior evidence, not a runtime dependency.

## Workflow

### 1. Bound The Audit

Choose one surface: provider routing, tools, prompt/context, sessions, Goncho memory, gateway/API, TUI, channels, cron, plugins/skills, CLI command tree, config/secrets/migration, observability, or packaging.

If the user asks for "everything", do one pass that produces a subsystem map and the next three audit passes.

For concrete operator reports like "TUI looks bad", "tool calls are exposed",
"hourglass is wrong", "installer runs Hermes", or "answer duplicated", audit
that visible behavior first. Hidden UX details are parity surfaces, not polish.
Preserve the user's transcript or terminal output as evidence and classify the
exact artifact: extra message, wrong label, wrong process owner, stale binary,
wrong home directory, leaked tool call, or missing reset/template state.

### 2. Build A Module Map

List:

- upstream files and symbols in `$HERMES_SRC` or `$HONCHO_SRC`;
- source classes in `docs/content/building-gormes/architecture_plan/upstream-coverage-ledger.md`;
- matching sections in `docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md`;
- current Gormes packages and commands;
- tests/fixtures proving existing behavior;
- progress rows that already cover the surface.

Use `rg`, `find`, and `jq`. Do not infer parity from package names alone.

Resolve upstream roots before reading refs:

```sh
HERMES_SRC="$(for p in ./hermes-agent ../hermes-agent references/hermes-agent; do [ -d "$p" ] && [ -f "$p/hermes_cli/main.py" ] && { printf '%s\n' "$p"; break; }; done)"
HONCHO_SRC="$(for p in ./honcho ../honcho references/honcho; do [ -d "$p" ] && { printf '%s\n' "$p"; break; }; done)"
```

Pick the active upstream implementation before classifying. For current
full-screen TUI UX, inspect `$HERMES_SRC/ui-tui/src/components` and
`$HERMES_SRC/ui-tui/src/__tests__` before falling back to older `cli.py`
prompt_toolkit code. For installer/runtime behavior, route implementation
questions through `gormes-dev-runtime` after recording the upstream evidence.
For prompt identity, memory, skills, and reset defaults, include
`hermes_cli/default_soul.py`, `agent/prompt_builder.py`, `tools/memory_tool.py`,
`agent/memory_manager.py`, `agent/skill_commands.py`, and `tools/skills_tool.py`
as applicable, then compare against `internal/agenttemplate`,
`internal/gateway/live_turn_prompt.go`, `internal/tools`, and `internal/skills`.
For tool-loop reports, include `$HERMES_SRC/run_agent.py` max-iteration
handling, then compare against the completed Gormes `Kernel tool loop` row,
`internal/kernel/kernel.go`, and `internal/kernel/tools_test.go` before
opening new work.

For CLI/config/migration audits, include these exact upstream anchors when in
scope: `hermes_cli/main.py` parser commands, `hermes_cli/commands.py` slash
registry, `gateway/run.py` command handlers, `hermes_cli/config.py` config
subcommands, and `hermes_cli/claw.py` plus
`optional-skills/migration/openclaw-migration/**` for OpenClaw migration.

### 3. Classify Gaps

For each upstream behavior:

- **covered**: implemented and tested in Gormes;
- **planned**: represented by a builder-ready progress row;
- **vague**: represented by an umbrella or underspecified row;
- **missing**: no useful Gormes code or progress row;
- **stale-upstream**: existing Gormes evidence or progress refs target retired
  upstream behavior while a newer Hermes source defines the active contract;
- **owned**: Gormes intentionally diverges with a better Go-native contract.

Classify against the upstream repo, the coverage ledger, and the feature map.
If upstream has a feature-bearing source class that the ledger or feature map
does not mention, the audit output must include a ledger update, feature-map
update, and progress-row proposal. If the map mentions behavior but the row is
vague, propose the smallest builder-ready split.

When the user asks for all Hermes commands or functions, do not jump straight
to handler implementation. First require a source-backed command-tree manifest
that classifies every top-level command, nested command, slash command, alias,
and Gormes-owned divergence as covered, planned, row-backed, owned, excluded,
or missing.

When a request misspells OpenClaw as `ooenclaw`, audit it as an operator typo
path. The parity plan should require a deterministic suggestion to
`gormes migrate openclaw`; do not count the typo as a Hermes/OpenClaw command
or propose it as a silent alias without a dedicated compatibility row.

For whole-repo coverage claims, run:

```sh
go test ./webpages/docs -run TestUpstreamCoverageLedgerMatchesSourceClasses -count=1
```

The test skips absent sibling upstream repos, but when they exist it must pass
before saying Hermes/Honcho source classes are fully mapped.

Read `references/audit-output.md` for the report shape.

### 4. Produce Progress-Ready Work

For every `vague` or `missing` item, propose a tracer-bullet row:

- one observable behavior;
- exact source refs;
- coverage-ledger row or missing source class;
- feature-map section or anchor;
- target Go package and public interface;
- exact write scope;
- focused test command;
- acceptance and done signal;
- dependencies.

Do not edit runtime code. Edit progress rows or parity evidence only when the user asked for the audit to apply changes; progress rows remain the backlog.

When the audit discovers stale refs on a row that is otherwise complete, update
the row's source refs, acceptance, and note instead of opening a duplicate row.
When runtime behavior is wrong too, hand one observable behavior to
`gormes-tdd-slice` with the active upstream refs included.

If a row is already complete, prefer adding correction evidence or a small
follow-up row over reopening the umbrella. In particular, do not duplicate the
completed `Gormes agent template reset command`; audit its actual files and
tests first.

Also avoid duplicating the completed `Kernel tool loop` row. If its focused
tests pass, classify live raw iteration-limit output as stale runtime,
provider-stream, or channel-rendering follow-up with a transcript fixture.

## Validation

If editing progress:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
go test ./internal/planning/progress -count=1
```

## Final Report

Report the subsystem audited, coverage classification, rows added/refined, unmapped areas, and the next best builder row.
