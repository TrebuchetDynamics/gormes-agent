---
title: "Next Slices"
weight: 35
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
| 2 / 2.B.10 | BlueBubbles iMessage bubble formatting parity | BlueBubbles outbound iMessage sends are non-editable, markdown-stripped, paragraph-split bubbles without pagination suffixes | gateway, system | `internal/channels/bluebubbles/bot_test.go` | Unblocks BlueBubbles iMessage session-context prompt guidance. |
| 2 / 2.E.1 | GBrain minion-orchestrator routing policy | Durable-job routing separates deterministic restart-survivable work from live LLM subagents, following GBrain's unified minion-orchestrator skill while keeping Gormes Go-native subagent APIs | operator, child-agent, system | `internal/subagent/minion_policy_test.go` | Unblocks Durable subagent/job ledger. |
| 3 / 3.E.7 | Interrupted-turn memory sync suppression | Interrupted or cancelled turns cannot flush partial observations into GONCHO or external Honcho-compatible memory | system | `internal/memory/interrupted_sync_test.go` | Unblocks Honcho host integration compatibility fixtures, Cross-chat operator evidence. |
| 3 / 3.E.7 | Honcho-compatible scope/source tool schema | Honcho-compatible tool schemas expose GONCHO scope and source allowlist controls without renaming public tools | operator, system | `internal/tools/honcho_tools_test.go` | Unblocks Cross-chat deny-path fixtures, Honcho host integration compatibility fixtures. |
| 4 / 4.A | Bedrock Converse payload mapping (no AWS SDK) | Pure Bedrock Converse request mapping over the shared provider message/tool contract | system | `internal/hermes/bedrock_converse_mapping_test.go` | Unblocks Bedrock stream event decoding (SSE fixtures). |
| 4 / 4.A | Codex Responses pure conversion harness | OpenAI Responses request/response conversion for Codex-compatible providers without live OAuth | system | `internal/hermes/codex_responses_adapter_test.go` | Unblocks Codex stream repair + tool-call leak sanitizer, Codex OAuth state + stale-token relogin. |
| 4 / 4.A | Tool-call argument repair + schema sanitizer | Provider tool-call arguments are repaired or rejected against available tool schemas before execution | system, child-agent | `internal/hermes/tool_call_argument_repair_test.go` | Unblocks Codex stream repair + tool-call leak sanitizer, OpenRouter, Bedrock stream event decoding (SSE fixtures). |
| 5 / 5.F | Skill preprocessing + dynamic slash commands | Skill content preprocessing and skill-backed slash commands are deterministic, disabled-skill aware, and prompt-safe | operator, gateway, system | `internal/skills/preprocessing_commands_test.go` | Unblocks Toolset-aware skills prompt snapshot, TUI + Telegram browsing. |
| 5 / 5.O | PTY bridge protocol adapter | Dashboard/TUI PTY sessions expose bounded read, write, resize, close, and unavailable-state behavior through a testable adapter | operator | `internal/cli/pty_bridge_test.go` | Unblocks SSE streaming to Bubble Tea TUI, Dashboard PTY chat sidecar contract. |
<!-- PROGRESS:END -->
