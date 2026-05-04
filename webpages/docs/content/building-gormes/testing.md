---
title: "Testing"
weight: 50
---

# Testing

Testing is the proof that a `progress.json` row is complete. The default is
hermetic fixture evidence; live providers and live platforms are optional smoke
checks, not row-local proof.

Use the [Completion Lane Roadmap](../architecture_plan/lane-roadmap/) to choose
the lane gate for broad changes.

## Row-local TDD

Use `gormes-tdd-slice` for implementation rows:

1. Write one failing public-behavior test.
2. Make the smallest implementation pass.
3. Repeat vertically inside the same row.
4. Refactor only while green.
5. Run the row's `test_commands`.

Tests should verify behavior through public interfaces: CLI output, HTTP
responses, gateway events, tool schemas/results, memory recall, provider
transcripts, or package APIs meant for callers. Avoid tests that only prove a
private helper's current shape unless the row explicitly defines that helper as
the public contract.

## Baseline layers

## Go tests

`go test ./...` from the repository root. Covers kernel, memory, tools, telegram adapter, session resume. Integration tests are tag-gated:

```bash
go test -tags=live ./...         # requires local Ollama
```

## Landing + docs smoke (Playwright)

`npm run test:e2e` from `www.gormes.ai/` and `docs/www-tests/`. Parametrized over mobile viewports (320 / 360 / 390 / 430 / 768 / 1024 px). Asserts:

- No horizontal overflow
- Section copy matches the locked strings in `content.go`
- Starlight navigation, TOC, and prev/next links render on mobile and desktop

## Astro/Starlight build

`go test ./docs/... -run TestAstroBuild`. Shells out to `npm run build`, verifies generated Starlight pages and redirect aliases, checks for broken internal links, and confirms the canonical `progress.json` is copied into the static export.

## Architecture fixtures

Subsystem ports need tests that freeze contracts, not just happy-path code.
Borrow these fixture classes from the Hermes and GBrain studies:

- **Command registry parity:** one registry drives parsing, help text, platform
  command exposure, aliases, and active-turn policy.
- **Provider transcripts:** replay request/stream fixtures for text,
  reasoning, tool-call deltas, finish reasons, usage, and retryable errors.
- **Tool schema parity:** compare upstream tool names, toolsets, JSON schemas,
  result envelopes, trust classes, availability/degraded status, and default
  registry exposure. A renderer fixture for `📖 read_file: "x"` is not proof
  that the model can call `read_file`; `cmd/gormes` registry tests must prove
  the descriptor is present.
- **Memory scope negatives:** prove same-chat default recall, opt-in user-scope
  widening, source allowlists, deny paths, and no cross-chat leakage.
- **Compression replay:** prove head/tail preservation, tool call/result pair
  integrity, summary lineage, and JSON-valid shrunken tool arguments.
- **Durable job replay:** prove cron/subagent jobs can be claimed, renewed,
  completed, failed, cancelled, retried, and inspected after restart.
- **Skill resolver conformance:** prove triggers, exclusions, disabled state,
  review state, and confusing phrases route to the expected skill or no skill.
- **Normal-turn e2e:** prove a Python-free Go agent turn crosses provider,
  tool execution, Goncho memory recall/persistence, final response, and
  audit/status evidence with fake providers and temp state.

## Operator Transcript Fixtures

When the bug came from live dogfood, preserve the exact visible artifact in the
test. A final answer alone is not sufficient proof.

- **Gateway/channel UX:** assert outbound send/edit/delete order, placeholder
  lifetime, typing-action fallback, final-message count, duplicate suppression,
  and whether tool progress is visible.
- **Tool progress:** assert the correct surface. Gateway progress uses
  `emoji tool_name: "preview"` with `new/all/verbose` modes, bounded previews,
  and `(×N)` collapse. Current TUI shelves use Title Case tool calls. Do not
  mix the two contracts in one fixture. Pair these with default-registry
  assertions when the bug is "Gormes collapsed all work into execute_code."
- **Local task tools:** for `read_file`, `search_files`, `write_file`,
  `patch`, and `terminal`, use temp roots and fake/small processes. Cover
  outside-root denial, symlink escape denial, duplicate-read status behavior,
  dangerous-command approval, timeout/error envelopes, and explicit
  unsupported evidence for background/PTY gaps.
- **Runtime surface:** for installer or lock reports, run the same focused
  command through the intended surface: `go run ./cmd/gormes`, `./bin/gormes`,
  or installed `gormes`, with an explicit `GORMES_HOME`. Record binary path and
  source checkout in the test name or helper comments.
- **Persona/reset:** ask the provider-captured prompt or reset fixture what
  files were injected or reset. Do not patch the assistant's final string after
  the provider as proof of identity.

## Discipline

Every PR must keep the relevant row-local, package, docs, and lane gates green.
The `deploy-gormes-landing.yml` and `deploy-gormes-docs.yml` workflows run the
Go + Playwright subsets on every push to `main`.

Minimum completion proof for runtime rows:

```bash
go run ./cmd/progress validate
go test ./... -count=1
```

Planner/doc-only rows normally run:

```bash
go run ./cmd/progress validate
go test ./docs -count=1
```

## Lane Gate Cheat Sheet

| Lane | Focused gate |
|---|---|
| Native agent spine | `go test ./internal/hermes ./internal/kernel ./internal/gateway ./internal/telemetry -count=1` plus `go test ./internal/e2e -count=1` after Phase 4.I creates it |
| Goncho/memory | `go test ./internal/goncho ./internal/gonchotools ./internal/memory ./internal/session ./internal/store -count=1` |
| Tools/security/skills | `go test ./internal/tools ./internal/plugins ./internal/skills ./internal/doctor -count=1` |
| Gateway/channels/cron | `go test ./internal/gateway ./internal/cron ./internal/channels/... ./internal/slack ./internal/discord -count=1` |
| CLI/API/TUI/release | `go test ./cmd/gormes ./internal/cli ./internal/apiserver ./internal/tui ./internal/tuigateway -count=1` |
| Docs/progress | `go run ./cmd/progress validate && go test ./internal/progress ./docs -count=1` |

When a row touches more than one lane, run all relevant focused gates before
the broad `go test ./... -count=1`.
