---
title: "Next Slices"
weight: 30
aliases:
  - /building-gormes/next-slices/
---

# Next Slices

This page is generated from the canonical progress file and lists the highest
leverage contract-bearing roadmap rows to execute next.

The ordering is:

1. unblocked `P0` handoffs;
2. active `in_progress` rows;
3. `fixture_ready` rows;
4. unblocked rows that unblock other slices;
5. remaining `draft` contract rows.

Use this page when choosing implementation work. If a row is too broad, split
the row in `progress.json` before assigning it.

If no slices are listed, the next correct action is planner work: choose one
planned row from `progress.json` or a phase page and add enough contract detail
for it to appear here. Do not infer that an empty generated list means the
roadmap is complete.

<!-- PROGRESS:START kind=next-slices -->
| Phase | Slice | Contract | Trust class | Fixture | Why now |
|---|---|---|---|---|---|
| 3 / 3.F | GONCHO local-first markdown MCP memory requirement | GONCHO must support a local-first memory mode that answers the OpenClaw community pain point: no cloud dependency, no mandatory API key, user-readable/editable markdown memory files, MCP-compatible access from any agent framework, optional local embeddings via Ollama, and restart-persistent storage. | operator, system | `internal/goncho/local_markdown_mcp_test.go; internal/gonchotools/mcp_catalog_test.go; internal/memory/markdown_store_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.J | Secrets Runtime Controls | Port OpenClaw's secrets runtime control surface: secrets apply for deploying previously generated plans, secrets audit to detect plaintext secrets/unresolved refs/precedence drift, secrets configure for interactive provider setup with SecretRef mapping and preflight validation, and secrets reload to re-resolve secret references and atomically swap the runtime snapshot. | operator | `internal/tools/secrets_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.J | Security Audit Command | Port OpenClaw's security audit: gormes security audit --deep --fix --json. Deep mode includes live gateway probe checks. Fix mode applies safe remediations and file-permission fixes. JSON mode produces machine-readable output. Audit categories: gateway auth status, state integrity, channel security warnings, shell blocklist coverage, filesystem scoping, credential redaction. | operator | `internal/tools/security_audit_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.O | CLI contextual first-touch onboarding hint renderers | internal/cli exposes pure constants and renderers for Hermes-compatible contextual onboarding hints: BusyInputPromptFlag = `busy_input_prompt`, ToolProgressPromptFlag = `tool_progress_prompt`, BusyInputHint(surface, mode string) string for interrupt/queue/steer modes, and ToolProgressHint(surface string) string for long-running tool progress. CLI text is plain ASCII and gateway text may use channel-friendly wording, but both preserve the operator contract: explain what just happened, name `/busy` or `/verbose` follow-up commands, and state that the tip only shows once. | operator, gateway, system | `internal/cli/onboarding_hints_test.go` | Unblocks Busy-input first-touch hint binding, Tool-progress first-touch hint binding. |
| 5 / 5.O | Gormes onboard interactive action runner | `gormes onboard --wizard` in an interactive TTY turns the existing deterministic first-run plan into an action runner. It renders the same model -> provider -> auth -> gateway -> browser/CDP -> skills -> dashboard steps, shows each step's configured/missing status and skip warning before any action, and lets the operator run, skip, or review each step. Selected actions delegate through fakeable command seams to existing setup/model/auth/gateway/browser/skills/dashboard surfaces; tests must never start live providers, gateways, browsers, dashboards, TTS downloads, or vendor probes. | operator, system | `cmd/gormes/onboard_wizard_test.go; internal/cli/onboard_test.go::TestOnboardPlan*` | Unblocks First-run user experience, Interactive Onboarding. |
| 5 / 5.H | ACP Client Bridge Mode | Complete the ACP integration with client bridge mode: gormes acp client connects to the Go-native ACP server (5.H server side is validated) with session key/label resolution, reset-session capability, require-existing guard, provenance modes (off/meta/meta+receipt), and --no-prefix-cwd flag. Match OpenClaw's ACP bridge surface. | operator, system | `internal/acp/client_test.go` | Unblocks Multi-agent interoperability, Editor integrations. |
| 5 / 5.I | Extension Lifecycle Hook System | Port agent-zero extension lifecycle hook system: register extensions at 8+ lifecycle points (agent_init, monologue_start/end, message_loop_start/end, before_main_llm_call, prompt_before/after, stream_chunk, tool_before/after, context_deleted). Extension chain executes in registration order with per-extension timeout and panic isolation. | operator, system | `internal/kernel/extensions_test.go` | Unblocks Plugin ecosystem, Skill injection pipeline. |
| 5 / 5.N | System Events, Heartbeat, and Presence | Port OpenClaw's system event surface: gormes system event enqueues a system event and optionally triggers a heartbeat; gormes system heartbeat shows and controls heartbeat state; gormes system presence lists system presence entries. Events are written to the audit ledger (JSONL) and surfaced in gormes status. | operator, system | `internal/tools/system_events_test.go` | Unblocks Operator observability, Gateway discover/probe diagnostics. |
| 5 / 5.N | Gateway Discover and Probe | Port OpenClaw's gateway network discovery: gormes gateway discover finds local gateways via Bonjour/mDNS; gormes gateway probe shows gateway reachability + discovery + health + status summary; gormes gateway usage-cost fetches usage cost summary from session logs. | operator | `internal/tools/gateway_discover_test.go` | Unblocks Multi-instance fleet management. |
| 5 / 5.N | Channels Capabilities Introspection | Port OpenClaw's channels capabilities: gormes channels capabilities shows provider capabilities (intents/scopes + supported features) for each configured channel. Enables operators to understand what each channel adapter supports before configuring it. | operator | `internal/channels/capabilities_test.go` | Unblocks Channel configuration UX. |
<!-- PROGRESS:END -->
