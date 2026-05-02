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
| 5 / 5.J | Secrets Runtime Controls | Port OpenClaw's secrets runtime control surface: secrets apply for deploying previously generated plans, secrets audit to detect plaintext secrets/unresolved refs/precedence drift, secrets configure for interactive provider setup with SecretRef mapping and preflight validation, and secrets reload to re-resolve secret references and atomically swap the runtime snapshot. | operator | `internal/tools/secrets_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.J | Security Audit Command | Port OpenClaw's security audit: gormes security audit --deep --fix --json. Deep mode includes live gateway probe checks. Fix mode applies safe remediations and file-permission fixes. JSON mode produces machine-readable output. Audit categories: gateway auth status, state integrity, channel security warnings, shell blocklist coverage, filesystem scoping, credential redaction. | operator | `internal/tools/security_audit_test.go` | P0 handoff; needs contract proof before closeout. |
| 5 / 5.O | Interactive Onboarding | Promote gormes onboard from setup alias into a truthful first-run command now, then complete the full interactive flow: model/provider selection -> auth setup -> gateway channel configuration -> browser/CDP checks -> skill discovery -> dashboard launch. Match OpenClaw's onboarding depth without pretending partial onboarding is complete. | operator | `cmd/gormes/skills_onboard_test.go::TestOnboardExplainsRuntimeSkillsAndLearningState,TestOnboardShowsConfiguredProviderDetails; future full wizard: internal/cli/onboard_test.go` | Already active; contract metadata keeps execution bounded. |
| 5 / 5.H | ACP Client Bridge Mode | Complete the ACP integration with client bridge mode: gormes acp client connects to the Go-native ACP server (5.H server side is validated) with session key/label resolution, reset-session capability, require-existing guard, provenance modes (off/meta/meta+receipt), and --no-prefix-cwd flag. Match OpenClaw's ACP bridge surface. | operator, system | `internal/acp/client_test.go` | Unblocks Multi-agent interoperability, Editor integrations. |
| 5 / 5.I | Extension Lifecycle Hook System | Port agent-zero extension lifecycle hook system: register extensions at 8+ lifecycle points (agent_init, monologue_start/end, message_loop_start/end, before_main_llm_call, prompt_before/after, stream_chunk, tool_before/after, context_deleted). Extension chain executes in registration order with per-extension timeout and panic isolation. | operator, system | `internal/kernel/extensions_test.go` | Unblocks Plugin ecosystem, Skill injection pipeline. |
| 5 / 5.N | System Events, Heartbeat, and Presence | Port OpenClaw's system event surface: gormes system event enqueues a system event and optionally triggers a heartbeat; gormes system heartbeat shows and controls heartbeat state; gormes system presence lists system presence entries. Events are written to the audit ledger (JSONL) and surfaced in gormes status. | operator, system | `internal/tools/system_events_test.go` | Unblocks Operator observability, Gateway discover/probe diagnostics. |
| 5 / 5.N | Gateway Discover and Probe | Port OpenClaw's gateway network discovery: gormes gateway discover finds local gateways via Bonjour/mDNS; gormes gateway probe shows gateway reachability + discovery + health + status summary; gormes gateway usage-cost fetches usage cost summary from session logs. | operator | `internal/tools/gateway_discover_test.go` | Unblocks Multi-instance fleet management. |
| 5 / 5.N | Channels Capabilities Introspection | Port OpenClaw's channels capabilities: gormes channels capabilities shows provider capabilities (intents/scopes + supported features) for each configured channel. Enables operators to understand what each channel adapter supports before configuring it. | operator | `internal/channels/capabilities_test.go` | Unblocks Channel configuration UX. |
| 5 / 5.N | Prompt Fragment Include System | Port agent-zero prompt fragment system: prompts stored as fragments with {{include filename.md}} directives, priority search order (agent profile > user > plugin > default), {{include original}} chains through hierarchy, variables substituted at render time. | operator, system | `internal/hermes/prompt_fragments_test.go` | Unblocks Agent profile customization, Plugin prompt injection. |
| 4 / 4.L | Plan gate hook in agent turn loop | Before tool execution, the agent loop invokes a plan-gate safety check. Unsafe plans are refused with explanation. Safe plans proceed. This mirrors MOSAIC (2025) plan->check->act/refuse pattern. | operator, system | `-` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
