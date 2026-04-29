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

<!-- PROGRESS:START kind=agent-queue -->
## 1. Tool output budget persisted artifact pointer

- Phase: 5 / 5.A
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Native tool execution bounds large tool results by persisting full output as a session artifact and returning a short text pointer to the model/channel, preserving Hermes operator readability and channel safety.
- Trust class: gateway, operator, system
- Ready when: The builder restates the Hermes parity contract and confirms no dependency on hermes-agent runtime services before editing., A small pure package can be tested without live providers, channels, or filesystem outside temp dirs., Artifact path policy uses Gormes data/session dirs, not reference repo paths.
- Not ready when: The row changes individual tool handlers before a shared result-budget helper exists., The row sends full oversized output to Telegram or provider context., The row stores artifacts outside a sanitized Gormes session/run directory.
- Degraded mode: If artifact persistence fails, the result is still bounded and includes a safe warning without exposing raw oversized payloads to external channels.
- Fixture: `internal/tools/result_budget_test.go`
- Write scope: `internal/tools/`, `internal/hermes/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools ./internal/hermes -run 'Tool.*Budget\|Truncat\|Artifact' -count=1`, `go test ./internal/tools -count=1`, `go run ./cmd/progress validate`
- Done signal: Tool-result budget tests prove oversized outputs become safe artifact pointers with sanitized paths and no channel/provider flooding.
- Acceptance: Text output over budget is truncated and full output is written to a sanitized artifact path., JSON/non-text output is persisted as JSON and represented by a short pointer., Callers receive evidence for truncated, persisted, and persistence_failed cases.
- Source refs: references/go-agent-os/GORMES-REUSE-AUDIT.md#2-tool-output-truncation, references/go-agent-os/nanobot/pkg/agents/truncate.go, references/go-agent-os/axe/internal/artifact/tracker.go, internal/tools/, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md#tools-sandboxes-and-security
- Unblocks: 61-tool registry port, Native runtime provider gateway binding, MCP stdio transport + tool/list discovery
- Why now: Unblocks 61-tool registry port, Native runtime provider gateway binding, MCP stdio transport + tool/list discovery.

## 2. Gormes-native MCP host runtime boundary

- Phase: 5 / 5.G
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Gormes exposes a native MCP/tool host boundary with explicit tool declarations, filtering, audit evidence, and channel/runtime-safe execution without adopting a non-Hermes config surface.
- Trust class: gateway, operator, system
- Ready when: The builder restates the Hermes parity contract and confirms no dependency on hermes-agent runtime services before editing., The interface design identifies caller-facing tool declaration/call/filter types before transport implementation., Hermes toolset/config semantics remain the source of truth for what tools are enabled.
- Not ready when: The row imports Nanobot config semantics or changes Hermes config.yaml precedence., The row vendors a full MCP framework instead of creating a tested Gormes boundary., The row bypasses channel/tool trust classes.
- Degraded mode: Unavailable MCP servers/tools produce structured unavailable/unauthorized evidence while core Hermes-parity tools and channel commands continue to work.
- Fixture: `internal/tools/mcp_host_boundary_test.go`
- Write scope: `internal/tools/`, `internal/plugins/`, `internal/gateway/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools ./internal/plugins -run 'ToolDeclaration\|ToolFilter\|MCPHost\|Audit' -count=1`, `go test ./internal/tools ./internal/plugins -count=1`, `go run ./cmd/progress validate`
- Done signal: A native Gormes tool/MCP boundary exists behind Hermes-compatible toolset/config semantics with filter and audit tests.
- Acceptance: A Gormes tool declaration interface can render provider JSON schema and MCP metadata from one source., Include/exclude filters can restrict tools by channel, trust class, and configured toolset., Audit evidence records server/tool name, arguments redaction status, result status, and unavailable errors.
- Source refs: references/go-agent-os/GORMES-REUSE-AUDIT.md#3-runtime-service-wiring-via-explicit-optionsmergecomplete, references/go-agent-os/nanobot/pkg/tools/service.go, references/go-agent-os/nanobot/pkg/runtime/runtime.go, references/go-agent-os/trpc-agent-go/tool/tool.go, references/go-agent-os/trpc-agent-go/tool/filter.go, internal/tools/, internal/plugins/, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md#tools-sandboxes-and-security
- Unblocks: MCP stdio transport + tool/list discovery, Managed tool gateway bridge, Tool output budget persisted artifact pointer
- Why now: Unblocks MCP stdio transport + tool/list discovery, Managed tool gateway bridge, Tool output budget persisted artifact pointer.

## 3. Goncho serialized write queue + relation candidates

- Phase: 5 / 5.N
- Owner: `memory`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Goncho serializes memory/conclusion writes and records pending relation candidates for possible conflicts or supersession without blocking the originating memory write.
- Trust class: operator, system
- Ready when: The builder restates the Hermes parity contract and confirms no dependency on hermes-agent runtime services before editing., Existing Goncho storage/tests identify the write entrypoints to serialize., The relation vocabulary is mapped to Honcho-compatible public behavior or explicitly kept internal.
- Not ready when: The row exposes Engram-specific API names as public Gormes/Honcho names., The row fails the original memory write because candidate detection is unavailable., The row adds an LLM judge before pending relation storage is deterministic.
- Degraded mode: If candidate search or relation insertion fails, the memory write still succeeds with degraded evidence; queue-full returns a retryable typed error.
- Fixture: `internal/goncho/write_queue_relation_test.go`
- Write scope: `internal/goncho/`, `internal/memory/`, `internal/gonchotools/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/goncho ./internal/memory -run 'WriteQueue\|Relation\|Conflict\|Supersede' -count=1`, `go test ./internal/goncho ./internal/memory -count=1`, `go run ./cmd/progress validate`
- Done signal: Goncho tests prove serialized writes and nonblocking pending relation candidates without changing Honcho-compatible external names.
- Acceptance: Concurrent memory writes execute in deterministic queue order under test., Queued cancellation before start does not mutate storage; started writes complete deterministically., Saving a memory can create pending relation candidates with verbs such as related, conflicts_with, supersedes, compatible, scoped, or not_conflict for later judgment.
- Source refs: references/go-agent-os/GORMES-REUSE-AUDIT.md#5-deterministic-serialized-mcpmemory-write-queue, references/go-agent-os/engram/internal/mcp/write_queue.go, references/go-agent-os/engram/internal/store/relations.go, references/go-agent-os/engram/internal/store/store.go, internal/goncho/, internal/memory/, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md#honcho-feature-map-for-goncho
- Unblocks: Goncho memory integration into normal agent turn, Goncho operator diagnostics contract
- Why now: Unblocks Goncho memory integration into normal agent turn, Goncho operator diagnostics contract.

<!-- PROGRESS:END -->
