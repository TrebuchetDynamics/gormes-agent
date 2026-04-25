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

<!-- PROGRESS:START kind=next-slices -->
| Phase | Slice | Contract | Trust class | Fixture | Why now |
|---|---|---|---|---|---|
| 2 / 2.B.5 | Non-editable gateway progress/commentary send fallback | Channels without placeholder/edit capabilities receive progress-safe interim or final assistant messages through the plain Send path without EditMessage calls | gateway, system | `internal/gateway/manager_test.go` | Unblocks BlueBubbles iMessage session-context prompt guidance. |
| 2 / 2.E.1 | GBrain minion-orchestrator routing policy | Durable-job routing separates deterministic restart-survivable work from live LLM subagents, following GBrain's unified minion-orchestrator skill while keeping Gormes Go-native subagent APIs | operator, child-agent, system | `internal/subagent/minion_policy_test.go` | Unblocks Durable subagent/job ledger. |
| 3 / 3.E.7 | Interrupted-turn memory sync suppression | Interrupted or cancelled turns cannot flush partial observations into GONCHO or external Honcho-compatible memory | system | `internal/memory/interrupted_sync_test.go` | Unblocks Honcho host integration compatibility fixtures, Cross-chat operator evidence. |
| 3 / 3.E.7 | Honcho-compatible scope/source tool schema | Honcho-compatible tool schemas expose GONCHO scope and source allowlist controls without renaming public tools | operator, system | `internal/tools/honcho_tools_test.go` | Unblocks Cross-chat deny-path fixtures, Honcho host integration compatibility fixtures. |
| 3 / 3.E.8 | parent_session_id lineage for compression splits | Session metadata records compression/fork lineage and can resolve the live descendant without rewriting ancestor history | operator, gateway, system | `internal/session/lineage_test.go` | Unblocks Gateway resume follows compression continuation, Lineage-aware source-filtered search hits. |
| 4 / 4.A | DeepSeek/Kimi reasoning_content echo for tool-call replay | Thinking-mode providers that require reasoning_content on assistant tool-call turns receive an echoed value during persistence and API replay | system | `internal/hermes/reasoning_content_echo_test.go` | Unblocks Cross-provider reasoning-tag sanitization, OpenRouter, Codex stream repair + tool-call leak sanitizer. |
| 4 / 4.A | Bedrock Converse payload mapping (no AWS SDK) | Pure Bedrock Converse request mapping over the shared provider message/tool contract | system | `internal/hermes/bedrock_converse_mapping_test.go` | Unblocks Bedrock stream event decoding (SSE fixtures). |
| 4 / 4.A | Codex Responses pure conversion harness | OpenAI Responses request/response conversion for Codex-compatible providers without live OAuth | system | `internal/hermes/codex_responses_adapter_test.go` | Unblocks Codex stream repair + tool-call leak sanitizer, Codex OAuth state + stale-token relogin. |
| 4 / 4.A | Tool-call argument repair + schema sanitizer | Provider tool-call arguments are repaired or rejected against available tool schemas before execution | system, child-agent | `internal/hermes/tool_call_argument_repair_test.go` | Unblocks Codex stream repair + tool-call leak sanitizer, OpenRouter, Bedrock stream event decoding (SSE fixtures). |
| 4 / 4.D | Provider-enforced context-length resolver | Displayed and budgeted context windows prefer provider-enforced limits over raw models.dev metadata | operator, system | `internal/hermes/model_context_resolver_test.go` | Unblocks Compression token-budget trigger + summary sizing, Routing policy and fallback selector. |
<!-- PROGRESS:END -->
