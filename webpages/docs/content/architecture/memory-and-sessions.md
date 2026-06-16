---
title: "Memory and Sessions"
description: "Session state, memory stores, and the difference between implementation evidence and user-visible recall."
weight: 40
---

# Memory and Sessions

Gormes stores native runtime state under `GORMES_HOME`, defaulting to `~/.gormes`.

Memory and session pages must distinguish:

- persisted session metadata;
- searchable session history;
- extracted facts or summaries;
- user-visible recall quality;
- Honcho/Goncho compatibility work.

Progress rows can prove a storage or fixture contract without proving that memory feels good in a live conversation. User docs should say what operators can inspect and how to debug bad recall.

## Gormes memory spine

The local memory spine is SQLite-first and prompt-facing:

- `internal/memory.OpenSqlite` creates the SQLite/FTS5 store and exposes the
  `store.Store` queue used by the kernel.
- `internal/memory.NewExtractor` turns completed non-cron turns into validated
  entities and relationships, then writes them transactionally into the graph.
- `internal/memory.NewRecall` builds scoped recall from exact-name, FTS5,
  semantic, graph traversal, decay, and RRF layers before formatting a sanitized
  `<memory-context>` block.
- `internal/gonchotools.NewTurnIntegration` adapts the in-binary Goncho service
  to the kernel's recall interface and registers Honcho-compatible memory tools
  when a local Goncho service is configured.

## Learning-loop study boundary

Hermes remains the compatibility source for the learning loop. The active
source refs are `hermes-agent/README.md`, `hermes-agent/run_agent.py`,
`hermes-agent/agent/curator.py`, `hermes-agent/toolsets.py`, and
`hermes-agent/cli-config.yaml.example`: they describe curated memory,
background memory/skill review, restricted memory-and-skills toolsets, auxiliary
curator model routing, and memory nudge settings.

OpenClaw is useful donor evidence, not a parity contract. Relevant source refs
include `openclaw/qa/new-scenarios-2026-04.md`,
`openclaw/webpages/docs/install/migrating-hermes.md`, and active-memory entries in
`openclaw/CHANGELOG.md`; use them for optional Gormes-owned ideas such as
explicit channel memory-search QA, graceful memory-plugin degradation,
per-conversation recall filters, timeout partial summaries, and operator
memory diagnostics.

## Validation handles

Use `go test ./internal/memory -count=1` for memory spine changes and
`go run ./cmd/progress validate` before changing memory, Goncho, or learning
loop progress rows. Documentation-only changes still require `git diff --check`.
