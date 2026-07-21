---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`webpages/docs/content/building-gormes/architecture_plan/progress.json`.

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
## 1. Gormes MCP server enable/disable persistence

- Phase: 5 / 5.G
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Add noninteractive `gormes mcp enable <name>` and `gormes mcp disable <name>` commands that atomically update only an existing active-profile `mcp_servers.<name>.enabled` field, preserve every other server field including tool filters and secret references, emit redacted text or JSON evidence, and state that a runtime reload is required. This slice performs no probe, network request, process launch, secret lookup, or live registry mutation.
- Trust class: operator
- Ready when: The existing typed MCP resolver continues to treat omitted enabled as true and explicit false as disabled without resolving credentials for operator output., The existing active-profile MCP store and atomic 0600 config writer are used; no second persistence path is introduced., The commands are explicitly persistence-only and report runtime_refresh_required rather than mutating an active registry.
- Not ready when: Implementation would read or print secret values, expose raw parser/writer errors, probe an MCP server, access the network, launch a process, or use a live profile/home fixture., Implementation would add interactive picker behavior, live runtime reload, stdio/OAuth transport work, durable credential cleanup, or git/bootstrap lifecycle behavior to this slice.
- Degraded mode: Invalid names, missing servers, malformed config, or atomic-write failures return stable redacted evidence and leave the config unchanged; active runtimes remain unchanged until reload or restart.
- Fixture: `t.TempDir active-profile config.toml with HTTP and stdio entries, synthetic secret-reference/header fields, explicit tool filters, injected MCPConfigPath, and no environment, process, network, OAuth, or live MCP server access`
- Write scope: `internal/config/mcpstore/`, `internal/platform/cli/gormescli/`, `internal/tools/mcp/config/`, `internal/tools/mcp/runtime/`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/config/mcpstore -run 'TestStoreSetEnabled' -count=1`, `go test ./internal/platform/cli/gormescli -run 'TestMCP(Enable\|Disable)' -count=1`, `go test ./internal/tools/mcp/config ./internal/tools/mcp/runtime -run 'Disabled\|Enabled' -count=1`, `go run ./cmd/progress validate`
- Done signal: Focused store and CLI tests prove field-preserving atomic enable/disable writes, redacted failure behavior, and explicit runtime-refresh guidance., Fresh config resolution and registry construction honor the persisted state, and the umbrella retains every non-config lifecycle gap.
- Acceptance: Enable and disable update only the exact existing server's enabled boolean for both HTTP and stdio-shaped fixtures while preserving URL/command, args, headers/env references, auth, timeouts, sampling, and tools fields., Invalid names, absent servers, malformed server maps, and injected write failures return stable redacted text/JSON evidence, a nonzero exit, and byte-unchanged config., `gormes mcp list` reflects the persisted state; a fresh resolver/registry excludes disabled servers and includes them again after enable without any live reload claim., Successful text and JSON output identifies only the sanitized server name, enabled state, and runtime_refresh_required=true; it never contains config values, paths, headers, environment values, or raw errors.
- Source refs: hermes-agent/hermes_cli/mcp_picker.py@bb5fc723:_enable_disable,_handle_row, hermes-agent/hermes_cli/mcp_catalog.py@bb5fc723:is_enabled, hermes-agent/tests/tools/test_mcp_probe.py@bb5fc723:TestProbeMCPServerTools.test_skips_disabled_servers, internal/config/mcpstore/store.go:Store.ConfigureTools,Store.Remove,Store.List, internal/tools/mcp/config/config.go:resolveMCPEnabled,disabledMCPServer, internal/platform/cli/gormescli/mcpconfig.go:newMCPConfigCommands, webpages/docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:Approved optional MCP catalog
- Unblocks: Hermes MCP catalog install/configure lifecycle
- Why now: Unblocks Hermes MCP catalog install/configure lifecycle.

<!-- PROGRESS:END -->
