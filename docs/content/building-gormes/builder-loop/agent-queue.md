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
## 1. Tool descriptor layer (OperationSpec)

- Phase: 5 / 5.A
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: Every tool in the registry carries a declarative descriptor (OperationSpec) that generates model schemas, CLI commands, gateway slash commands, doctor checks, and audit taxonomy from one source
- Trust class: operator, gateway, child-agent, system
- Ready when: Tool registry inventory + schema parity harness is complete., Hardline command pattern table + DetectHardline function is validated on main.
- Not ready when: The slice ports handler logic instead of adding descriptors around existing handlers., The slice changes the existing Tool interface contract., The slice wires descriptors into live prompt assembly or gateway dispatch before the descriptor schema is fixture-backed.
- Degraded mode: If descriptors are missing, doctor reports tool_descriptor_incomplete and the tool is hidden from gateway/child-agent callers until the descriptor is present.
- Fixture: `internal/tools/operation_spec_test.go`
- Write scope: `internal/tools/operation_spec.go`, `internal/tools/operation_spec_test.go`, `internal/tools/registry.go`, `internal/tools/registry_test.go`, `internal/tools/executor.go`, `internal/tools/executor_test.go`, `internal/doctor/tool_descriptors.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run 'TestOperationSpec\|TestTrustClass\|TestExecutor' -count=1`, `go test ./internal/tools -count=1`, `go test ./internal/doctor -run 'TestToolDescriptor' -count=1`, `go run ./cmd/progress validate`
- Done signal: OperationSpec fixtures prove descriptor validation, trust-class rejection, and doctor checks. Core tools carry descriptors. Executor rejects disallowed callers before handler entry.
- Acceptance: OperationSpec struct declares name, description, schema, mutating, idempotent, prompt_safe, allowed trust classes, timeout, and audit kind., Tool registry accepts tools with or without descriptors; tools without descriptors report descriptor_missing in doctor., Shared tool executor rejects disallowed trust classes before a handler runs, with explicit trust_class_denied evidence., Doctor validates every registered tool descriptor for required fields and schema validity., Default toolset assigns descriptors to all core tools (read_file, search_files, write_file, patch, terminal, todo, session_search)., Descriptor schema generation produces valid JSON Schema for model consumption without handler changes.
- Source refs: gbrain:src/core/operations.ts (contract-first operation catalog), mercury-agent:permission manifest (trust-class enforcement), docs/content/building-gormes/must-have-features.md, docs/content/building-gormes/architecture_plan/phase-5-final-purge.md, docs/content/building-gormes/upstream-lessons.md
- Why now: P0 handoff; needs contract proof before closeout.

## 2. Regex-based auto-link extraction + brain-first lookup

- Phase: 6 / 6.I
- Owner: `memory`
- Size: `large`
- Status: `planned`
- Priority: `P2`
- Contract: Markdown links, wikilinks, qualified wikilinks auto-extracted; typed inference; brain-first 5-step lookup
- Trust class: operator, system
- Ready when: Goncho page storage exists
- Not ready when: No local page/slug storage in Goncho
- Degraded mode: Links not auto-extracted; lookup skips local DB/graph and goes directly to LLM/external API
- Fixture: `internal/goncho/auto_link_test.go`
- Write scope: `internal/goncho/auto_link.go`, `internal/goncho/brain_first.go`
- Test commands: `go test ./internal/goncho -run TestAutoLink -count=1`
- Done signal: Auto-link tests prove markdown/wikilink/typed extraction; brain-first tests prove 5-step lookup
- Acceptance: Markdown link syntax is auto-extracted, Wikilinks [[slug]] and [[source:dir/slug]] auto-extracted, Typed inference: FOUNDED, INVESTED, ADVISES, WORKS_AT, Brain-first lookup: local DB → graph → cache → LLM → external API
- Source refs: gbrain/src/core/link-extraction.ts, gbrain/src/core/search/hybrid.ts, hermes-agent/agent/context_references.py
- Unblocks: Compiled truth pattern, Tiered enrichment
- Why now: Unblocks Compiled truth pattern, Tiered enrichment.

## 3. SKILL.md metadata.when/loaded/placement schema

- Phase: 6 / 6.H
- Owner: `skills`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: YAML frontmatter supports metadata.when (conditional activation), metadata.loaded (auto-load), metadata.placement (system/onscreen/admin); hierarchical routing
- Trust class: operator, system
- Ready when: SKILL.md parser exists
- Not ready when: SKILL.md format is still undefined
- Degraded mode: Skills load without metadata placement; all skills treated as system scope
- Fixture: `internal/skills/metadata_test.go`
- Write scope: `internal/skills/metadata.go`, `internal/skills/routing.go`
- Test commands: `go test ./internal/skills -run TestMetadata -count=1`
- Done signal: Metadata tests prove conditional activation, auto-load, placement, and routing
- Acceptance: metadata.when supports conditional activation, metadata.loaded supports auto-load flag, metadata.placement supports system/onscreen/admin, Hierarchical routing skill routes to sub-skills
- Source refs: space-agent/src/skills/schema.js, hermes-agent/agent/skill_utils.py
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
