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
| 4 / 4.A | DeepSeek/Kimi reasoning_content echo for tool-call replay | Thinking-mode providers that require reasoning_content on assistant tool-call turns receive an echoed value during persistence and API replay | system | `internal/hermes/reasoning_content_echo_test.go` | Unblocks Cross-provider reasoning-tag sanitization, OpenRouter, Codex stream repair + tool-call leak sanitizer. |
| 4 / 4.A | Codex Responses pure conversion harness | OpenAI Responses request/response conversion for Codex-compatible providers without live OAuth | system | `internal/hermes/codex_responses_adapter_test.go` | Unblocks Codex stream repair + tool-call leak sanitizer, Codex OAuth state + stale-token relogin. |
| 4 / 4.A | Tool-call argument repair + schema sanitizer | Provider tool-call arguments are repaired or rejected against available tool schemas before execution | system, child-agent | `internal/hermes/tool_call_argument_repair_test.go` | Unblocks Codex stream repair + tool-call leak sanitizer, OpenRouter, Bedrock stream event decoding (SSE fixtures). |
| 7 / 7.E | BlueBubbles iMessage bubble formatting parity | BlueBubbles outbound iMessage sends are non-editable, markdown-stripped, paragraph-split bubbles without pagination suffixes | gateway, system | `internal/channels/bluebubbles/bot_test.go` | Unblocks BlueBubbles iMessage session-context prompt guidance. |
<!-- PROGRESS:END -->
