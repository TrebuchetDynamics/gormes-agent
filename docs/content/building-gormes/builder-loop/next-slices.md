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
| 2 / 2.F.1 | Gateway slash registry parity sweep (recognized-name expansion) | internal/gateway/commands.go::CommandRegistry recognizes every Hermes/Sidon command from cmd/gormes/hermes_cli_parity_test.go's manifest, even when the handler is not yet implemented, so unknown-command replies only fire on actual non-Hermes inputs. Each newly-recognized command lands with ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable and a friendly description, mirroring the existing /retry, /undo, /title, /branch, /compress pattern. Aliases (reset for new, set-home for sethome, gateway for platforms, etc.) resolve to the canonical command. Handler ports remain owned by the 49-file CLI tree port umbrella; this row only changes recognition. | gateway, operator | `internal/gateway/commands_test.go + cmd/gormes/hermes_cli_parity_test.go` | Unblocks 49-file CLI tree port. |
| 5 / 5.A | Stateful tool migration queue | Gormes defines the migration queue and execution guard for stateful Hermes tools before exposing write-capable tools to the native loop: file, session, checkpoint, and process tools declare state domains, XDG roots, rollback/audit behavior, concurrency policy, and degraded evidence; the first implementation is a registry/read-model contract that lets builders add one stateful tool at a time without bypassing path isolation. | operator | `internal/tools/stateful_migration_queue_test.go` | Unblocks File write/patch tool port, Checkpoint restore tool port, Terminal process execution port. |
| 5 / 5.N | Debug helpers | Gormes ports Hermes DebugSession as shared tool debug infrastructure: tool-specific env vars enable a per-tool session ID, log entries remain in memory until explicit save, save writes deterministic JSON under an injected debug log directory, disabled debug mode is a no-op, get_session_info returns enabled/session/path/count evidence, and sensitive arguments are redacted before persistence. | operator, system | `internal/tools/debug_helpers_test.go` | Unblocks Multi-model coordination, Debug share paste sweep scheduler contract, Web/search tool debug logging. |
| 7 / 7.E | Feishu transport/bootstrap layer | Gormes adds a fakeable Feishu/Lark transport bootstrap boundary before live SDK binding: config resolves app credentials, connection_mode selects webhook vs websocket, webhook URL verification and signature checks are pure helpers, websocket event handlers register message/reaction/card/customized processors, inbound events queue until the adapter loop is ready, and rich-text/card send failures return typed SendResult evidence with redacted tokens. | gateway, operator, system | `internal/channels/feishu/bootstrap_test.go` | Unblocks Feishu drive-comment rule + pairing seam, Feishu drive-comment reply workflow, Feishu live SDK binding. |
| 2 / 2.B.5 | Gateway session token accounting parity | Accumulate per-session provider usage into session metadata so `/status` reports Hermes-compatible token totals rather than only the last usage frame. | gateway, system, operator | `internal/gateway/token_accounting_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 3 / 3.F | Goncho memory provider lifecycle adapter | Create a native Goncho memory-provider lifecycle adapter covering initialize, prefetch, sync turn, pre-compress contribution, memory-write mirror, delegation, and shutdown evidence so Gormes matches Hermes MemoryManager behavior without a hosted Honcho dependency. | system | `internal/memory/provider_lifecycle_test.go + internal/goncho` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 4 / 4.H | Prompt-cache capability guard | Gormes applies Hermes prompt-cache markers only when provider, endpoint, API mode, and model policy allow them: native Anthropic uses native layout, OpenRouter Claude uses envelope layout, third-party Anthropic Claude gateways cache conservatively, Qwen on opencode/opencode-go/Alibaba gets envelope markers, and OpenAI-wire custom providers without an allow rule strip cache_control visibly. | operator, system | `internal/hermes/prompt_cache_policy_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.C | Browser artifact and console render contract | Browser tool results expose bounded screenshot paths, DOM snapshots, console logs, page errors, and artifact metadata in a channel-neutral envelope; renderers show safe previews without raw bytes, base64 blobs, private URLs, CDP secrets, or unbounded provider output. | system, gateway, operator | `internal/tools/browser_artifact_test.go + internal/gateway/render_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.C | Telegram browser artifact rendering | Telegram rendering for browser results is mobile-readable and Hermes/Sidon-compatible: screenshot/artifact pointers, DOM excerpts, console errors, and browser progress traces are MarkdownV2-safe, bounded, reply-threaded, and separate from final answers. | gateway, system, operator | `internal/gateway/telegram_browser_render_test.go + internal/tools/browser_artifact_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 5 / 5.N | Clarify | Gormes ports Hermes clarify as a schema-validated, interruptible user-reply tool: required question text, up to four trimmed choices, platform-added Other behavior, callback/resume routing for gateway and TUI, deterministic unavailable output in non-interactive cron/oneshot contexts, and one-shot resume-token cleanup after the next user reply. | operator, gateway, child-agent, system | `internal/tools/clarify_tool_test.go; internal/gateway/clarify_resume_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
