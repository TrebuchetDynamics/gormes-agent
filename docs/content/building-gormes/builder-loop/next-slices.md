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
| 5 / 5.A | Stateful tool migration queue | Gormes defines the migration queue and execution guard for stateful Hermes tools before exposing write-capable tools to the native loop: file, session, checkpoint, and process tools declare state domains, XDG roots, rollback/audit behavior, concurrency policy, and degraded evidence; the first implementation is a registry/read-model contract that lets builders add one stateful tool at a time without bypassing path isolation. | operator | `internal/tools/stateful_migration_queue_test.go` | Unblocks File write/patch tool port, Checkpoint restore tool port, Terminal process execution port. |
| 5 / 5.C | Browser action contract + event transcript | Gormes freezes the native browser tool contract before binding Chromedp or Rod: action schema, page-state transcript events, screenshot/result envelope, console/content-none guards, private-URL safety handoff, oversized artifact pointer behavior, and unavailable-backend errors are represented as pure Go types and fixtures. | operator, child-agent, system | `internal/tools/browser_contract_test.go` | Unblocks Chromedp, Rod, Browser provider bridge + Firecrawl fallback. |
| 5 / 5.N | Todo | Gormes ports Hermes todo_tool as a small stateful native tool: items validate id/content/status, duplicate IDs keep the last entry while preserving list order, replace vs merge semantics match Hermes, only pending/in_progress items are injected into prompts, summary counts include pending/in_progress/completed/cancelled, and storage is deterministic under an injected per-session root. | operator | `internal/tools/todo_tool_test.go` | Unblocks TUI todo panel. |
| 5 / 5.N | Debug helpers | Gormes ports Hermes DebugSession as shared tool debug infrastructure: tool-specific env vars enable a per-tool session ID, log entries remain in memory until explicit save, save writes deterministic JSON under an injected debug log directory, disabled debug mode is a no-op, get_session_info returns enabled/session/path/count evidence, and sensitive arguments are redacted before persistence. | operator, system | `internal/tools/debug_helpers_test.go` | Unblocks Multi-model coordination, Debug share paste sweep scheduler contract, Web/search tool debug logging. |
| 7 / 7.E | Feishu transport/bootstrap layer | Gormes adds a fakeable Feishu/Lark transport bootstrap boundary before live SDK binding: config resolves app credentials, connection_mode selects webhook vs websocket, webhook URL verification and signature checks are pure helpers, websocket event handlers register message/reaction/card/customized processors, inbound events queue until the adapter loop is ready, and rich-text/card send failures return typed SendResult evidence with redacted tokens. | gateway, operator, system | `internal/channels/feishu/bootstrap_test.go` | Unblocks Feishu drive-comment rule + pairing seam, Feishu drive-comment reply workflow, Feishu live SDK binding. |
| 4 / 4.B | Tool-result pruning + protected head/tail summary | Gormes freezes the pure context-compression pruning pass before kernel mutation: protect system and first-turn head messages, choose the recent tail by token budget with at least three messages, keep assistant tool_calls paired with their tool results, prune old oversized tool result content without cutting tool-call arguments or JSON payloads, and emit summary-prefix-compatible replacement messages. | operator, system | `internal/hermes/context_compressor_pruning_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.H | Prompt-cache capability guard | Gormes applies Hermes prompt-cache markers only when provider, endpoint, API mode, and model policy allow them: native Anthropic uses native layout, OpenRouter Claude uses envelope layout, third-party Anthropic Claude gateways cache conservatively, Qwen on opencode/opencode-go/Alibaba gets envelope markers, and OpenAI-wire custom providers without an allow rule strip cache_control visibly. | operator, system | `internal/hermes/prompt_cache_policy_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.N | Clarify | Gormes ports Hermes clarify as a schema-validated, interruptible user-reply tool: required question text, up to four trimmed choices, platform-added Other behavior, callback/resume routing for gateway and TUI, deterministic unavailable output in non-interactive cron/oneshot contexts, and one-shot resume-token cleanup after the next user reply. | operator, gateway, child-agent, system | `internal/tools/clarify_tool_test.go; internal/gateway/clarify_resume_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
