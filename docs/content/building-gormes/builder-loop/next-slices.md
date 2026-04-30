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
| 2 / 2.B.5 | Gateway /title manual session title command | Implement Hermes-compatible `/title` handling in the gateway: `/title` shows the current title, `/title <name>` sanitizes and stores a manual title, manual titles are not overwritten by auto-title, invalid titles return operator guidance, and the command never reaches the provider. | gateway, operator | `internal/gateway/title_command_test.go` | P0 handoff; needs contract proof before closeout. |
| 3 / 3.F | Hermes memory tool over Goncho/local durable store | Expose the Hermes-visible `memory` tool with add, replace, and remove actions over memory/user targets, backed by Goncho or local durable USER.md/MEMORY.md storage, while preserving safe write responses, redaction, locks, and prompt-insertion semantics. | system, operator | `internal/tools/memory_tool_test.go + internal/memory` | P0 handoff; needs contract proof before closeout. |
| 2 / 2.F.1 | Gateway slash registry parity sweep (recognized-name expansion) | internal/gateway/commands.go::CommandRegistry recognizes every Hermes/Sidon command from cmd/gormes/hermes_cli_parity_test.go's manifest, even when the handler is not yet implemented, so unknown-command replies only fire on actual non-Hermes inputs. Each newly-recognized command lands with ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable and a friendly description, mirroring the existing /retry, /undo, /title, /branch, /compress pattern. Aliases (reset for new, set-home for sethome, gateway for platforms, etc.) resolve to the canonical command. Handler ports remain owned by the 49-file CLI tree port umbrella; this row only changes recognition. | gateway, operator | `internal/gateway/commands_test.go + cmd/gormes/hermes_cli_parity_test.go` | Unblocks 49-file CLI tree port. |
| 5 / 5.A | Stateful tool migration queue | Gormes defines the migration queue and execution guard for stateful Hermes tools before exposing write-capable tools to the native loop: file, session, checkpoint, and process tools declare state domains, XDG roots, rollback/audit behavior, concurrency policy, and degraded evidence; the first implementation is a registry/read-model contract that lets builders add one stateful tool at a time without bypassing path isolation. | operator | `internal/tools/stateful_migration_queue_test.go` | Unblocks File write/patch tool port, Checkpoint restore tool port, Terminal process execution port. |
| 5 / 5.E | Transcription tool contract | Native STT/transcription tool helper validates local audio input and provider selection before gateway media hooks call it: files must exist, be regular files, use supported audio suffixes, and stay under configured max bytes; explicit provider selection among local, local_command, groq, openai, mistral, and xai never silently falls back; auto mode chooses Hermes order local, groq, openai, mistral, xai from injected availability; model defaults and overrides are normalized per provider; tool results return transcript/provider/model/language on success or typed redacted error evidence on failure. | operator, gateway, system | `internal/tools/transcription_tool_test.go` | Unblocks TTS synthesis + voice-mode state, Gateway media transcription hooks, Voice attachment handling for Signal and QQ Bot. |
| 5 / 5.N | Debug helpers | Gormes ports Hermes DebugSession as shared tool debug infrastructure: tool-specific env vars enable a per-tool session ID, log entries remain in memory until explicit save, save writes deterministic JSON under an injected debug log directory, disabled debug mode is a no-op, get_session_info returns enabled/session/path/count evidence, and sensitive arguments are redacted before persistence. | operator, system | `internal/tools/debug_helpers_test.go` | Unblocks Multi-model coordination, Debug share paste sweep scheduler contract, Web/search tool debug logging. |
| 7 / 7.E | Feishu transport/bootstrap layer | Gormes adds a fakeable Feishu/Lark transport bootstrap boundary before live SDK binding: config resolves app credentials, connection_mode selects webhook vs websocket, webhook URL verification and signature checks are pure helpers, websocket event handlers register message/reaction/card/customized processors, inbound events queue until the adapter loop is ready, and rich-text/card send failures return typed SendResult evidence with redacted tokens. | gateway, operator, system | `internal/channels/feishu/bootstrap_test.go` | Unblocks Feishu drive-comment rule + pairing seam, Feishu drive-comment reply workflow, Feishu live SDK binding. |
| 2 / 2.B.5 | Telegram reply_to_mode and reply-context parity | Telegram replies honor Hermes-style reply mode configuration, fall back cleanly if a target message was deleted, and inbound Telegram reply text can be attached to session context without leaking raw slash commands to the model. | gateway, operator | `internal/channels/telegram/reply_mode_test.go + internal/gateway/manager_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 2 / 2.B.5 | Telegram typing action + placeholder lifecycle parity | Telegram turn progress matches Hermes/Sidon lifecycle: typing action or placeholder appears while work runs, stale hourglass messages are deleted or finalized, duplicate ghost replies collapse, and final answers remain readable. | gateway, operator | `internal/channels/telegram/placeholder_lifecycle_test.go + internal/gateway/coalesce_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 2 / 2.B.5 | Gateway stream/tool trace formatting fixture matrix | Channel-neutral stream rendering has source-backed fixtures for Hermes/Sidon text deltas, tool progress, errors, and final answer separation, with Telegram MarkdownV2 escaping and compact labels for memory/search/read/patch/terminal/browser actions. | gateway, operator | `internal/gateway/render_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
