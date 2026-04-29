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
## 1. Gormes-native MCP host runtime boundary

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

<!-- PROGRESS:END -->
