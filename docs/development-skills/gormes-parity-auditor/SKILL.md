---
name: gormes-parity-auditor
description: Audit upstream Hermes, Honcho, and GBrain behavior against current Gormes implementation and progress.json. Use when the user asks what is missing for full Hermes-in-Go parity, wants upstream feature mapping, wants Goncho/Honcho compatibility gaps, or needs progress.json-ready parity rows before builder work.
---

# Gormes Parity Auditor

## Mission

Find what prevents Gormes from being Hermes in Go, with Goncho as the Honcho-compatible Go port. Output builder-ready gaps, not a vague research report.

## Workflow

### 1. Bound The Audit

Choose one surface: provider routing, tools, prompt/context, sessions, Goncho memory, gateway/API, TUI, channels, cron, plugins/skills, config/secrets, observability, or packaging.

If the user asks for "everything", do one pass that produces a subsystem map and the next three audit passes.

### 2. Build A Module Map

List:

- upstream files and symbols in `../hermes-agent`, `../honcho`, or `../gbrain`;
- source classes in `docs/content/building-gormes/architecture_plan/upstream-coverage-ledger.md`;
- matching sections in `docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md`;
- current Gormes packages and commands;
- tests/fixtures proving existing behavior;
- progress rows that already cover the surface.

Use `rg`, `find`, and `jq`. Do not infer parity from package names alone.

### 3. Classify Gaps

For each upstream behavior:

- **covered**: implemented and tested in Gormes;
- **planned**: represented by a builder-ready progress row;
- **vague**: represented by an umbrella or underspecified row;
- **missing**: no useful Gormes code or progress row;
- **owned**: Gormes intentionally diverges with a better Go-native contract.

Classify against the upstream repo, the coverage ledger, and the feature map.
If upstream has a feature-bearing source class that the ledger or feature map
does not mention, the audit output must include a ledger update, feature-map
update, and progress-row proposal. If the map mentions behavior but the row is
vague, propose the smallest builder-ready split.

For whole-repo coverage claims, run:

```sh
go test ./docs -run TestUpstreamCoverageLedgerMatchesSourceClasses -count=1
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

Do not edit runtime code. Edit `progress.json` only when the user asked for the audit to apply changes.

## Validation

If editing progress:

```sh
go run ./cmd/progress write
go run ./cmd/progress validate
go test ./internal/progress -count=1
```

## Final Report

Report the subsystem audited, coverage classification, rows added/refined, unmapped areas, and the next best builder row.
