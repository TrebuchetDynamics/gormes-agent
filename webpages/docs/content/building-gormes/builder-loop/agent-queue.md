---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
skill-driven implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared skill handoff facts live in [Skill Builder Handoff](../builder-loop-handoff/):
the main skill entrypoint, plan, candidate source, generated docs, tests, and
candidate policy. Keep those control-plane facts in `meta.builder_loop`, and
keep row-specific execution facts in `progress.json`.

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Internal session search tool package rehome

- Phase: 8 / 8.F
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Move the model-facing session_search adapter from internal/sessionsearchtool to internal/tools/sessionsearch while preserving descriptor text, JSON schema, argument normalization, memory/session catalog execution, degraded evidence, and cmd/gormes registry wiring. This is the next behavior-preserving Tool Adapter Enclave package move after compact and trace.
- Trust class: -
- Ready when: internal topology guard is shipped and can activate a row-scoped forbidden-root entry, internal tool compact and trace helper rehome rows are complete so this move starts from the next clean tool-enclave baseline, the slice moves only internal/sessionsearchtool, not internal/kanbantools or internal/gonchotools
- Not ready when: the row attempts to consolidate multiple tool helper families in one move, session_search descriptor text, JSON schema, argument defaults/limits, current-session binding, same-chat/user scope rules, or degraded evidence changes, a compatibility shim keeps internal/sessionsearchtool alive after the move
- Degraded mode: If the rehome is incomplete, builds fail on the old import path or topology validation reports stale internal/sessionsearchtool references; no runtime compatibility shim is allowed.
- Fixture: `internal/tools/sessionsearch/session_search_tool_schema_test.go and internal/tools/sessionsearch/session_search_tool_execution_test.go after the move, plus cmd/gormes registry tests.`
- Write scope: `internal/internal_topology_test.go`, `internal/sessionsearchtool/`, `internal/tools/sessionsearch/`, `cmd/gormes/registry.go`, `cmd/gormes/registry_test.go`, `internal/tools/codemap.md`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal -run TestInternalTopology -count=1`, `go test ./internal/tools/sessionsearch ./cmd/gormes -run 'SessionSearch\|Registry' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: internal/tools/sessionsearch exists and internal/sessionsearchtool is absent, rg 'internal/sessionsearchtool\|github.com/.*/internal/sessionsearchtool' cmd internal --glob '*.go' --glob '!internal/internal_topology_test.go' returns no matches, row test_commands pass
- Acceptance: internal/internal_topology_test.go has an active tools migration entry that forbids internal/sessionsearchtool and requires internal/tools/sessionsearch, internal/tools/sessionsearch preserves the session_search tool descriptor, schema, argument validation, limit clamping, and unavailable-store degraded evidence from the old package tests, session_search execution tests still prove same-chat default recall, explicit user-scope widening evidence, source filters, recent/search mode, lineage shaping, and hidden tool-source filtering through the public tool interface, cmd/gormes default registry imports internal/tools/sessionsearch and still registers session_search with the configured SQLite memory DB and SessionSearchDirectory, no active Go import or path literal under cmd/ or internal/ points at internal/sessionsearchtool after the move, except the intentional topology migration OldRoots entry
- Source refs: internal/REFACTOR-CMD-PLAN.md:Tool Adapter Enclave, internal/REFACTOR-CMD-PLAN.md:Package Move Playbook, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:Tool registry and toolsets, hermes-agent/tools/session_search_tool.py, hermes-agent/tests/tools/test_session_search.py, internal/sessionsearchtool/session_search_tool.go:NewSessionSearchTool, internal/sessionsearchtool/session_search_tool_schema_test.go, internal/sessionsearchtool/session_search_tool_execution_test.go, cmd/gormes/registry.go:buildDefaultRegistry
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
