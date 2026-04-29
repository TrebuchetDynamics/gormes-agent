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
| 5 / 5.C | Browser action contract + event transcript | Gormes freezes the native browser tool contract before binding Chromedp or Rod: action schema, page-state transcript events, screenshot/result envelope, console/content-none guards, private-URL safety handoff, oversized artifact pointer behavior, and unavailable-backend errors are represented as pure Go types and fixtures. | operator, child-agent, system | `internal/tools/browser_contract_test.go` | Unblocks Chromedp, Rod, Browser provider bridge + Firecrawl fallback. |
| 4 / 4.B | Tool-result pruning + protected head/tail summary | Gormes freezes the pure context-compression pruning pass before kernel mutation: protect system and first-turn head messages, choose the recent tail by token budget with at least three messages, keep assistant tool_calls paired with their tool results, prune old oversized tool result content without cutting tool-call arguments or JSON payloads, and emit summary-prefix-compatible replacement messages. | operator, system | `internal/hermes/context_compressor_pruning_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.H | Prompt-cache capability guard | Gormes applies Hermes prompt-cache markers only when provider, endpoint, API mode, and model policy allow them: native Anthropic uses native layout, OpenRouter Claude uses envelope layout, third-party Anthropic Claude gateways cache conservatively, Qwen on opencode/opencode-go/Alibaba gets envelope markers, and OpenAI-wire custom providers without an allow rule strip cache_control visibly. | operator, system | `internal/hermes/prompt_cache_policy_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.N | Clarify | Gormes ports Hermes clarify as a schema-validated, interruptible user-reply tool: required question text, up to four trimmed choices, platform-added Other behavior, callback/resume routing for gateway and TUI, deterministic unavailable output in non-interactive cron/oneshot contexts, and one-shot resume-token cleanup after the next user reply. | operator, gateway, child-agent, system | `internal/tools/clarify_tool_test.go; internal/gateway/clarify_resume_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
