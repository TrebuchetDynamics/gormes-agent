---
title: "Hermes <-> Gormes parity matrix"
description: "Durable parity inventory for Hermes-in-Go (Gormes) delivery. One row per parity area with upstream + Gormes refs, status classification, lane assignment, and progress.json row pointer."
date: 2026-04-29
draft: false
aliases:
  - /building-gormes/architecture_plan/hermes-gormes-parity-matrix/
---

# Hermes <-> Gormes parity matrix

Audit pass: 2026-04-29 (lane A parity-auditor).

This document is the durable parity inventory that replaces reactive
screenshot-driven fixes. Every row names one Hermes/Sidon parity area, cites
upstream + Gormes evidence, classifies the gap, and points at the
progress.json row that future implementation lanes (B-H) execute.

For symbol-level pairings such as "which Hermes Python code does
`internal/gateway/render.go` adapt?", use
[Hermes/Gormes Contract Pairings](../hermes-gormes-contract-pairings/). The
matrix below stays area/status oriented; the pairings page owns the taxonomy
for `surface`, `contract_layer`, `upstream_contract`, and `gormes_adapter`.

Statuses use the parity-auditor convention: `parity` (implemented and tested),
`partial` (implemented in part; specific gap noted), `missing` (no Gormes
evidence), `regressed` (was working, now broken or contradictory),
`excluded` (intentional Gormes-owned divergence).

Lane key (per the parity operating model):

- B: Telegram UX and command parity
- C: Session/status/title parity
- D: Prompt/provider identity parity (final-payload integration test)
- E: Memory/Goncho parity
- F: Provider/auth/runtime parity
- G: Browser harness/tool parity (including future Go browser harness lane)
- H: CLI/config parity

## At-a-glance table

| # | Area | Status | Lane | Progress row |
|---|---|---|---|---|
| 1 | `/status` heading + fields + reply quoting + gateway-side execution | parity | B/C | Hermes-format /status field order, optional title, tokens, running marker, reply quoting, and provider bypass are tested |
| 2 | `/help` and Telegram setMyCommands menu | parity | B | Registry-driven /help plus Telegram setMyCommands with dynamic skill commands, sanitization, and platform limit coverage are tested |
| 3 | Slash command registry (Hermes vs Gormes command names + aliases) | parity | B/H | Gateway and CLI registries recognize shared/gateway command names and aliases, preserve unavailable evidence, and derive help/menu surfaces from the canonical registry |
| 4 | Unknown command behavior | parity | B | Active-turn slash bypass tests cover unknown |
| 5 | Unavailable command behavior | parity | B | active_turn_command_bypass_test (covered) |
| 6 | Active-turn slash bypass behavior | parity | B | Gateway active-turn policy manifest closeout (planned for closeout edge cases) |
| 7 | Busy/admission behavior | parity | B | Active-turn follow-up queue + late-arrival drain policy (complete) |
| 8 | Telegram reply quoting (every bot response) | parity | B | Reply modes all/first/off, SendReply/SendThreadReply reply_to_message_id, and deleted-target fallback are wired and tested |
| 9 | Telegram message threading (forum topic threads) | parity | B | Thread-aware sends include message_thread_id, General-topic behavior, chat actions, and stale-thread fallback tests |
| 10 | Telegram markdown rendering (bold/italic/code/headers/spoilers/strike) | parity | B | Telegram MarkdownV2 parse-mode rendering closeout (complete) |
| 11 | Telegram code blocks | parity | B | Telegram MarkdownV2 parse-mode rendering closeout (complete) |
| 12 | Telegram bullets/headings | parity | B | Telegram MarkdownV2 parse-mode rendering closeout (complete) |
| 13 | Error message formatting | parity | B | Existing render_test fixtures cover sanitization |
| 14 | Progress messages (interim) | parity | B | Coalescer + render_test cover stream cadence |
| 15 | Final answer separation | parity | B | Coalescer fresh-final + sendNoEdit cover phase split |
| 16 | Stale placeholder cleanup | parity | B | Coalescer fresh-final tests cover delete-and-resend |
| 17 | Hourglass lifecycle (no clutter) | parity | B | Telegram typing action + placeholder lifecycle parity (planned for typing-indicator polish) |
| 18 | Message edits vs new messages | parity | B | Coalescer placeholder-then-edit fixture covers it |
| 19 | Streaming cadence | parity | B | Coalescer window + freshFinalAfter govern cadence |
| 20 | Tool trace formatting (memory/search_files/read_file/patch/terminal icons) | parity | B | Gateway/TUI/shared tool trace icons, modes, duplicate collapse, previews, and per-platform mode overrides are validated; skin YAML emoji customization is tracked outside this renderer row |
| 21 | Duplicate collapse | parity | B | Message-ID dedup plus same-content active/follow-up collapse are scoped by platform/chat/thread/user and tested |
| 22 | Mobile-readable formatting | parity | B | Telegram MarkdownV2 parse-mode, fallback, shared tool-progress traces, and native TUI tool progress/modal renderers are covered |
| 23 | Identity / persona ("My name is Gormes" not "ChatGPT") | parity | D | Live-turn SOUL.md and project context wiring (channel-neutral) (complete) |
| 24 | Final provider request includes Gormes identity (integration test) | parity | D | Gateway and production Telegram provider-payload golden tests assert Gormes identity/system context reaches the final request |
| 25 | Final provider request includes USER.md / MEMORY.md | parity | D | Live-turn USER.md and MEMORY.md durable user context block (channel-neutral) (complete) |
| 26 | Final provider request includes AGENTS.md / project context | parity | D | Live-turn SOUL.md and project context wiring (complete) |
| 27 | Final provider request includes skill guidance | parity | D | Final provider requests inject SkillsGuidance before selected skill blocks and record selected skill usage when skills are active |
| 28 | Final provider request includes platform context | parity | D | Session context BuildSessionContextPrompt (covered by gateway test fixture) |
| 29 | Final provider request includes session metadata | parity | D | BuildSessionContextPrompt covers this |
| 30 | Final provider request includes timestamp/timezone/model/provider | parity | D | Live-turn timestamp + model/provider/session metadata block + self-help guidance (complete) |
| 31 | Final provider request includes tool guidance constants | parity | D | Final provider requests inject memory, session_search, skills, tool-use enforcement, model, and research guidance when matching capabilities are active |
| 32 | Context-file production discovery (~/.gormes/SOUL.md, workspace fallback, Hermes profile fallback) | parity | D | Live-turn SOUL/AGENTS wiring uses GORMES_HOME → memory/ → HERMES_HOME → CWD ancestor chain |
| 33 | Session ID generation (Hermes-style format) | parity | C | Conversational submits generate YYYYMMDD_HHMMSS_<suffix> IDs, persist mapping metadata, and reuse the mapped ID across turns |
| 34 | Session title auto-generation | parity | C | Gateway PhaseIdle auto-title calls the configured TitleModel/TitleStore once per eligible session and production config wires both seams |
| 35 | Session title persistence (across restarts) | parity | C | Manual and auto titles persist in session metadata and /status reads preserved titles |
| 36 | Manual title preservation (don't overwrite) | parity | C | status_command.go preserves existing meta.Title |
| 37 | Created timestamp (no `(unknown)`) | parity | C | Fresh conversational sessions write CreatedAt metadata and /status reads it before fallback parsing |
| 38 | Last activity timestamp | parity | C | Conversational session metadata refresh preserves CreatedAt and updates UpdatedAt on subsequent turns |
| 39 | Token accounting accuracy | parity | C | Render-frame token totals persist into session metadata and /status renders durable cumulative totals |
| 40 | Agent Running status accuracy | parity | C | hasActiveTurn() drives status |
| 41 | Connected Platforms accuracy | parity | C | connectedPlatforms() drives status |
| 42 | Session resume | parity | C | Durable pause/resume intent contract (complete) |
| 43 | Session reset/new/retry/undo | parity | C | /new + /reset alias, durable SQLite-backed /retry transcript rewrite, and /undo [N] rewind/resume are wired and tested |
| 44 | Memory/Goncho durable user memory | parity | E | Hermes memory tool over local durable markdown/Goncho Memory V1 store is covered by memory and Goncho tests |
| 45 | Memory prompt insertion | parity | D/E | Live-turn USER.md and MEMORY.md durable user context block is injected channel-neutrally and tested |
| 46 | Memory write/read lifecycle | parity | E | Memory add/read/replace/remove plus Goncho provider lifecycle/markdown reload/export are covered |
| 47 | Memory redaction | parity | E | Existing redaction fixtures in internal/memory and internal/audit |
| 48 | Session summaries / compression boundary | parity | E | Provider-backed compression, summary lineage, manual /compress binding, feedback, context references, and boundary callbacks are covered |
| 49 | GONCHO branding (not Kancho, not Honcho-renamed) | parity | E | internal/goncho/ ships and tests assert workspace=gormes peer=gormes |
| 50 | Provider registry parity | parity | F | Registry/aliases plus OpenRouter, Google Code Assist, Gemini Cloud Code, Codex Responses, Bedrock runtime/SigV4/stale-client, Gemini native transport/runtime, and google-gemini-cli OAuth login/runtime/refresh are covered |
| 51 | Auth status command | parity | F/H | cmd/gormes/auth_status_command_test.go covers it |
| 52 | Auth add/list/remove/logout commands | parity | F/H | cmd/gormes/auth_command_test.go covers per-provider lifecycle |
| 53 | Codex device-code/OAuth path | parity | F/H | gormes auth add openai-codex --type oauth runs Hermes-compatible device flow plus Codex CLI import/fallback |
| 54 | Credential pool | parity | F | Bare gormes auth lists credential pools with redacted provider/account evidence and Bedrock identity status |
| 55 | Provider request shape | parity | F | hermes.ChatRequest used end-to-end |
| 56 | Provider stream handling | parity | F | OpenStream + retry budget shipped in kernel |
| 57 | Retry behavior | parity | F | NewRetryBudget + retryStatus shipped |
| 58 | Health checks | parity | F | RuntimeStatusStore + ReadValidatedRuntimeStatusSnapshot |
| 59 | Rate-limit evidence | parity | F | Rate-limit tracker plus Codex/Anthropic/OpenRouter account-usage windows, degraded evidence, and /usage rendering are covered |
| 60 | Redacted diagnostics (no token leakage) | parity | F | render_test sanitize tests, audit ledger redaction |
| 61 | CLI command tree (Hermes vs Gormes surface) | parity | H | Live Cobra tree, slash registry, module ownership manifest, aliases, and row-backed unavailable surfaces are covered by contract tests |
| 62 | CLI help text | parity | H | gatewayHelpText() + GatewayHelpLines() drive consistent help |
| 63 | Active-turn CLI policy | parity | H | CLI command registry parity + active-turn busy policy (complete) |
| 64 | Provider/config resolution | parity | H | Config/profile/auth/setup resolution seams and fallback config are covered by integration tests |
| 65 | Config path discovery | parity | H | config.GormesHome() + ConfigPath() shipped |
| 66 | Config show/check/edit/migrate | parity | H | Config show/get/check/edit plus native profile-v2 migrate are covered by CLI tests |
| 67 | Diagnostics | parity | H | Doctor/status/logs plus backup/restore diagnostics are covered by CLI tests |
| 68 | Browser tool contract | parity | G | tools/browser_contract.go + browser_harness_tools.go shipped |
| 69 | Browser snapshots | parity | G | In-process CDP/Chromedp backend snapshot path is covered by fake transport tests |
| 70 | DOM text extraction | parity | G | Snapshot text/interactive extraction is covered by browser backend tests |
| 71 | Screenshot artifacts | parity | G | Browser artifact envelopes and bounded screenshots are covered |
| 72 | Console logs | parity | G | Console expression/log result shaping is covered |
| 73 | Browser navigation | parity | G | CDP target creation/navigation and Browser Use bridge routing are covered |
| 74 | Browser click/type/scroll | parity | G | CDP click/type/scroll/press actions are covered |
| 75 | Browser session lifecycle | parity | G | Target session persistence and restoration are covered |
| 76 | Artifact budgets | parity | G | tools.ToolResultBudgetConfig shipped + browser_harness_tools_test |
| 77 | Private URL / SSRF safety | parity | G | tools/browser_ssrf_guard.go + browser_ssrf_guard_test.go |
| 78 | Browser tool result formatting compatibility with Hermes | parity | G | Browser artifact/console result contracts are covered by backend and gateway renderer tests |
| 79 | Browser channel rendering for Telegram | parity | G | Telegram browser artifact rendering is covered by gateway renderer tests |
| 80 | Go browser harness integration lane (placeholder for future repo) | parity | G | Superseded by in-process CDP/Chromedp backend plus Browser Use bridge tests |

## Per-area detail (H3 sections)

The H3 sections below give the prose evidence for each row. Tables linearize
poorly for screen readers, so the matrix above is a cheap index; this
section is the audit's source-of-truth. Each section follows the parity
audit output template (Subsystem, Upstream Behavior, Gormes Evidence,
Classification, Progress row).

### 1. /status (heading, Session ID, Title, Created, Last Activity, Tokens, Agent Running, Connected Platforms, reply quoting, gateway-side execution)

- **Hermes source**: `../hermes-agent/gateway/run.py:4646-4680`
  (`_handle_status_command`). Returns
  `📊 **Hermes Gateway Status**` followed by Markdown bold field rows in
  the order Session ID, optional Title, Created, Last Activity, Tokens,
  Agent Running, Connected Platforms.
- **Gormes source**: `internal/gateway/status_command.go`
  (`formatGatewayStatus`). Renders the same field set and order with
  Markdown bold labels, optional Title when metadata has one, Created/Last
  Activity in `%Y-%m-%d %H:%M` shape, comma-formatted cumulative tokens,
  `Yes ⚡`/`No` Agent Running state, queued follow-up depth, and Connected
  Platforms. Gormes-owned agent-route and Kanban lines are additive when
  enabled.
- **Reply quoting**: `internal/gateway/status_command.go:15` calls
  `m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, ...)` so `/status`
  replies do thread to the triggering message via Telegram's
  `ReplyToMessageID`. Verified by
  `internal/gateway/status_command_test.go::statusReplyChannel`.
- **Gateway-side execution (no model)**: confirmed by
  `manager.go:670+776` dispatch on `EventStatus` short-circuiting
  before any kernel.Submit.
- **Status**: `parity`. Field set/order, optional title behavior, reply
  quoting, gateway-side provider bypass, active-turn marker, queued follow-up
  count, connected platforms, and durable token/title/session metadata are
  covered by tests.
- **Test coverage**:
  `internal/gateway/status_command_test.go`,
  `internal/gateway/statusview_test.go`,
  `internal/gateway/active_turn_command_bypass_test.go::TestManager_ActiveTurnSlashCommandBypass_ChannelNeutral`.
- **Lane**: B (Telegram UX), C (session metadata).
- **Progress row**: reconciled by current `/status` implementation and tests.
- **Evidence command**:
  `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway -run Status -count=1`
- **Live Telegram evidence**: when the user sends `/status`, Gormes
  replies (threaded to the request) with the field block. MarkdownV2
  parse-mode is set on Telegram sends, replies, and edits, so the bold
  field labels render instead of appearing as literal escaped markup.
  Empty titles are omitted, matching Hermes.

### 2. /help and slash menu (setMyCommands / getMyCommands)

- **Hermes source**: `../hermes-agent/gateway/platforms/telegram.py`
  bot startup builds the menu from `COMMANDS_BY_CATEGORY` and calls
  `setMyCommands`; `../hermes-agent/gateway/run.py:4937` formats
  `📖 **Hermes Commands**`.
- **Gormes source**: `internal/gateway/commandregistry/registry.go`
  derives `/help` text and Telegram menu entries from the canonical command
  registry. `internal/adapters/channels/telegram/bot.go::registerCommands`
  calls `setMyCommands` on startup using `TelegramBotCommandsWith`, and
  `internal/app/gateway/channels.go::TelegramDynamicCommands` loads enabled
  skill slash commands for live gateway startup.
- **Status**: `parity`.
- **Test coverage**: `internal/adapters/channels/telegram/bot_test.go`
  covers Hermes command registration, dynamic skill commands, and Telegram's
  platform limit; `internal/gateway/commandregistry/registry_parity_test.go`
  covers `/help` derivation and command sanitization; `internal/app/gormescmd/gateway_test.go`
  covers dynamic skill command collection.
- **Lane**: B.
- **Progress row**: Telegram dynamic BotCommand menu wiring is covered.
- **Evidence**:
  `go test ./internal/adapters/channels/telegram ./internal/gateway/commandregistry ./internal/app/gormescmd ./internal/app/telegram -run 'RunRegisters|DynamicCommands|GatewayHelp|CommandRegistry|Help|setMyCommands|Commands' -count=1`.

### 3. Slash command registry (Hermes vs Gormes command names + aliases)

- **Hermes source**: `../hermes-agent/hermes_cli/commands.py:59-175`
  (`COMMAND_REGISTRY`) defines commands with categories, aliases, args_hint,
  subcommands, and cli_only / gateway_only flags.
- **Gormes source**: `internal/gateway/commandregistry/registry.go` is the
  gateway source of truth and `internal/platform/cli/commands/registry/catalog/data.go`
  is the CLI policy manifest. Gateway tests prove command names, aliases,
  typed raw-command evidence, unavailable-command preservation, `/help`,
  Telegram menu, Slack mappings, and CLI/gateway policy parity.
- **Status**: `parity` for registry/name/alias recognition. Individual
  handler implementation gaps remain tracked by their specific rows (for
  example browser/session/config/diagnostics rows), not this registry row.
- **Test coverage**: `internal/gateway/commandregistry/registry_parity_test.go`
  and `internal/platform/cli/commands/registry` parity tests.
- **Lane**: B (gateway slash visibility), H (CLI handlers).
- **Progress row**: Gateway slash registry parity sweep is covered.
- **Evidence**:
  `go test ./internal/gateway/commandregistry ./internal/platform/cli/commands/registry -run 'CommandRegistry|Parity|Aliases|Hermes|Unavailable|TelegramBotCommands|SlackSubcommand' -count=1`.

### 4. Unknown command behavior

- **Hermes source**: `../hermes-agent/gateway/run.py` per-platform
  unknown-command handlers reply with help excerpt.
- **Gormes source**: `internal/gateway/manager.go` dispatches
  `EventUnknown` to a friendly "unknown command" reply; verified by
  `internal/gateway/active_turn_command_bypass_test.go:95` which asserts
  `/does-not-exist` produces "unknown command".
- **Status**: `parity`.
- **Test coverage**: `active_turn_command_bypass_test.go`.
- **Lane**: B.
- **Progress row**: covered by existing tests; no new row needed.

### 5. Unavailable command behavior

- **Hermes source**: `../hermes-agent/hermes_cli/commands.py` flags some
  commands `cli_only=True` so when issued via gateway they return a
  friendly fallback.
- **Gormes source**: `internal/gateway/commands.go:107-119`
  (`buildRecognizedUnavailableSlashCommands`) +
  `internal/gateway/active_turn_command_bypass_test.go:94` which asserts
  `/retry` returns "/retry is recognized but unavailable".
- **Status**: `parity`.
- **Lane**: B.
- **Progress row**: no new row.

### 6. Active-turn slash bypass behavior

- **Hermes source**: `../hermes-agent/gateway/run.py` allows `/help`,
  `/stop`, `/status`, `/usage` to flow through during an active turn.
- **Gormes source**: `CommandActiveTurnPolicyImmediate` /
  `CommandActiveTurnPolicyDrain` /
  `CommandActiveTurnPolicyReject` /
  `CommandActiveTurnPolicyUnavailable` enum at
  `internal/gateway/commands.go:21-28` plus
  `active_turn_command_bypass_test.go::TestManager_ActiveTurnSlashCommandBypass_ChannelNeutral`.
- **Status**: `parity` for the four commands tested; closeout edge
  cases tracked.
- **Lane**: B.
- **Progress row**: `Gateway active-turn policy manifest closeout`
  (planned).

### 7. Busy/admission behavior

- **Hermes source**: `../hermes-agent/gateway/run.py` admits / queues /
  rejects mid-turn submits.
- **Gormes source**: kernel ErrTurnInFlight + active-turn follow-up
  queue (`Active-turn follow-up queue + late-arrival drain policy`).
- **Status**: `parity`.
- **Lane**: B.
- **Progress row**: existing complete row.

### 8. Telegram reply quoting (every bot response quotes triggering message)

- **Hermes source**: `../hermes-agent/gateway/platforms/telegram.py:973-986,
  1023-1031` always passes `reply_to_message_id` and falls back when the
  target was deleted. Reply mode `first | all | off` driven by
  `_reply_to_mode` config.
- **Gormes source**: `internal/gateway/manager.go::replyTargetForTurn`
  implements reply modes `all` (default), `first`, and `off`; terminal
  frames route through `sendWithHooksReplyThread`. Telegram `SendReply` and
  `SendThreadReply` pass `reply_to_message_id`; deleted reply targets retry
  once without the reply id.
- **Status**: `parity`.
- **Test coverage**: `internal/gateway/status_command_test.go`,
  `internal/gateway/thread_delivery_test.go`, and
  `internal/adapters/channels/telegram/thread_fallback_test.go`.
- **Lane**: B.
- **Progress row**: covered by Telegram reply/thread parity work.

### 9. Telegram message threading

- **Hermes source**: forum topic threads via `message_thread_id` on
  Telegram bot API. `../hermes-agent/gateway/platforms/telegram.py`
  forum-topic helpers + `_should_thread_reply`.
- **Gormes source**: `internal/gateway/thread_delivery_test.go` proves
  gateway thread routing; `internal/adapters/channels/telegram/thread_send_test.go`
  proves `message_thread_id` on thread sends/replies/actions and General-topic
  text behavior; `thread_fallback_test.go` proves stale topic retry without
  losing reply behavior.
- **Status**: `parity`.
- **Lane**: B.
- **Progress row**: covered by Telegram reply/thread parity work.

### 10. Telegram markdown rendering (bold/italic/code/headers/spoilers/strike)

- **Hermes source**: `../hermes-agent/gateway/platforms/telegram.py:91-122`
  defines MarkdownV2 escape regex + `format_message` helpers; line 985
  posts with `parse_mode=ParseMode.MARKDOWN_V2` and falls back to plain
  on parse failure (line 998).
- **Gormes source**:
  - render side: `internal/gateway/render.go`
    (`FormatStreamTelegram`, `FormatFinalTelegram`, `FormatErrorTelegram`)
    uses `tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, ...)` on every
    Telegram-bound string.
  - send side: `internal/channels/telegram/bot.go::Send`,
    `SendReply`, and `EditMessage` set `ParseMode =
    tgbotapi.ModeMarkdownV2`; `SendPlaceholder` and
    `SendReplyPlaceholder` inherit this through Send/SendReply.
  - fallback side: `internal/channels/telegram/bot.go::sendWithParseFallback`
    retries once with `ParseMode` unset on parse/markdown errors while
    preserving the byte-identical message body.
- **Net effect on Telegram**: escaped MarkdownV2 from the shared render
  layer now reaches Telegram with the matching parse mode, so bold,
  italic, code, spoiler, strike, status labels, and code-like snippets
  render instead of showing literal backslashes.
- **Status**: `parity` for parse-mode wiring and parse-error fallback.
- **Test coverage**: `internal/channels/telegram/bot_parse_mode_test.go`
  covers Send, SendReply, EditMessage, parse-error fallback, and
  render-layer byte preservation. `internal/gateway/render_test.go`
  continues to cover escaping/sanitization.
- **Lane**: B.
- **Progress row**: `Telegram MarkdownV2 parse-mode rendering closeout`
  (complete, validated).
- **Evidence command**:
  `GOCACHE=/tmp/gormes-go-cache go test ./internal/channels/telegram -run 'MarkdownV2|ParseMode|Send|Reply|EditMessage' -count=1`.

### 11. Telegram code blocks

- **Hermes source**: same as #10. Triple-backtick blocks render via
  MarkdownV2 fenced code.
- **Gormes source**: same as #10 — rendering escapes MarkdownV2 and the
  Telegram channel now sets MarkdownV2 parse mode on sends/replies/edits.
- **Status**: `parity` for the parse-mode transport layer; content-level
  code-block styling remains owned by the shared renderer fixtures.
- **Lane**: B.
- **Progress row**: same as #10 (`Telegram MarkdownV2 parse-mode
  rendering closeout`, complete).

### 12. Telegram bullets / headings

- **Hermes source**: same as #10. Bullets and headings rely on
  client-side rendering of MarkdownV2.
- **Gormes source**: same as #10. Parse-mode wiring is complete; broader
  mobile-readable style polish remains under stream/tool trace and
  Telegram UX rows.
- **Status**: `parity` for parse-mode transport; `partial` only for
  broader style polish outside this row.
- **Lane**: B.
- **Progress row**: same as #10 for parse-mode, with remaining style
  polish row-backed by `Gateway stream/tool trace formatting fixture matrix`.

### 13. Error message formatting

- **Hermes source**: `../hermes-agent/gateway/run.py` error replies are
  short, prefixed, and never leak provider HTML.
- **Gormes source**: `internal/gateway/render.go::FormatErrorPlain` /
  `FormatErrorTelegram` prefix `❌` and sanitize provider HTML/secret
  markers (verified by `render_test.go::TestFormatErrorPlain_*` and
  `TestFormatErrorTelegram_SanitizesProviderHTMLBody`).
- **Status**: `parity`.
- **Lane**: B.
- **Progress row**: no new row.

### 14. Progress messages (interim)

- **Hermes source**: streaming partial frames sent as edits during a
  turn.
- **Gormes source**: `internal/gateway/coalesce.go` plus
  `Coalescer_PlaceholderThenEdit` test fixture covers the cadence.
- **Status**: `parity`.
- **Lane**: B.
- **Progress row**: no new row.

### 15. Final answer separation

- **Hermes source**: terminal phase is its own send.
- **Gormes source**: `manager.go::sendNoEdit` checks `kernel.PhaseIdle`
  and posts a final message.
- **Status**: `parity`.
- **Lane**: B.
- **Progress row**: no new row.

### 16. Stale placeholder cleanup

- **Hermes source**: deletes hourglass when a fresh final is needed.
- **Gormes source**: `coalesce_fresh_final_test.go::TestCoalescerFreshFinal_OldPreviewSendsFreshAndDeletesOld`.
- **Status**: `parity`.
- **Lane**: B.
- **Progress row**: no new row.

### 17. Hourglass lifecycle (no clutter)

- **Hermes source**: typing indicator + placeholder edits.
- **Gormes source**: `bot.go::SendPlaceholder/SendReplyPlaceholder` send
  ⏳; coalescer manages edits. Typing-action API not yet wired.
- **Status**: `partial`. Functional placeholder works; native typing
  action (`telegram.sendChatAction`) is not used.
- **Lane**: B.
- **Progress row**: `Telegram typing action + placeholder lifecycle parity`
  (planned).

### 18. Message edits vs new messages

- **Hermes source**: edits the streaming bubble; sends fresh on stale.
- **Gormes source**: same — coalescer covers both.
- **Status**: `parity`.
- **Lane**: B.
- **Progress row**: no new row.

### 19. Streaming cadence

- **Hermes source**: throttles edits to keep within Telegram rate
  limits.
- **Gormes source**: `coalescer` window driven by `CoalesceMs` and
  `freshFinalAfter`.
- **Status**: `parity`.
- **Lane**: B.
- **Progress row**: no new row.

### 20. Tool trace formatting

- **Hermes source**: `../hermes-agent/agent/display.py::get_tool_emoji`
  + per-tool registrations (file_tools `read_file = 📖`, `search_files
  = 🔎`, memory_tool `🧠`, web tools `🔎`).
- **Gormes source**: `internal/tooltrace/tooltrace.go` provides the
  shared Hermes-style formatter used by gateway, TUI, Slack, and
  Discord. It recognizes the same visible action families:
  - `🧠` memory
  - `📚` skill tools
  - `📋` todo
  - `🔎` search_files
  - `🔍` web_search
  - browser action icons
  - `📖` read_file
  - `🔧` patch / write_file / fallback
  - `💻` terminal / process
  - `💻` execute_code
- **Coverage**: Channel-visible formatting is validated, including
  duplicate collapse, `new/all/off` display modes, bounded previews,
  per-platform mode overrides, and suppression of `tool done:` completion
  noise. Hermes skin-engine `tool_emojis` customization is a theme/skin
  capability, not a renderer parity blocker for this row.
- **Status**: `parity`.
- **Test coverage**:
  `internal/tooltrace/tooltrace_test.go`,
  `internal/gateway/render_test.go`,
  `internal/slack/render_test.go`,
  `internal/discord/render_test.go`,
  `internal/tui/viewport_history_test.go`.
- **Lane**: B.
- **Progress row**: `Gateway stream/tool trace formatting fixture matrix`
  (complete/validated) plus `Native TUI Hermes tool progress + modal
  panel renderers` (complete/validated). Future skin YAML loading or
  per-tool emoji overrides belong under the TUI skin-engine lane, not
  this renderer row.

### 21. Duplicate collapse

- **Hermes source**: gateway suppresses duplicate restart-takeover
  markers; CLI deduplicates rapid repeated submits.
- **Gormes source**: `message_deduplicator.go` tracks platform message IDs;
  `manager.go::queueFollowUpIfActive` also collapses same-content active-turn
  and follow-up submissions scoped by platform/account/chat/thread/user.
  `message_deduplicator_manager_test.go` proves same text with different
  message IDs is dropped for the same user and allowed for different users.
- **Status**: `parity`.
- **Lane**: B.
- **Progress row**: covered by gateway duplicate-suppression parity work.

### 22. Mobile-readable formatting

- **Hermes source**: MarkdownV2 enables bold/italic/code on mobile.
- **Gormes source**: #10 is complete for Telegram ParseMode and parse-error
  fallback, so mobile clients receive the same MarkdownV2 transport contract.
  Shared tool-trace formatting and native TUI tool-progress/modal renderers are
  fixture-covered. Future subjective renderer/content polish is not a concrete
  Hermes parity blocker for this row.
- **Status**: `parity`.
- **Lane**: B.
- **Progress row**: `Telegram MarkdownV2 parse-mode rendering closeout`
  (complete) plus `Gateway stream/tool trace formatting fixture matrix`
  and `Native TUI Hermes tool progress + modal panel renderers`
  (complete/validated) for the now-shipped tool-progress surfaces.

### 23. Identity / persona ("My name is Gormes" not "ChatGPT")

- **Hermes source**: `../hermes-agent/run_agent.py` + `agent/prompt_builder.py`
  loads SOUL.md and project-context files into the system prompt.
- **Gormes source**: `internal/llm/context_files.go::BuildContextFilesPrompt`
  builds the same block, and `internal/gateway/live_turn_prompt.go`
  + `manager.go:1632-1648` wires it into the channel-neutral
  `kernel.PlatformEvent.SessionContext`. The kernel prepends it as a
  system role message at `internal/kernel/kernel.go:299-301`.
- **Status**: `parity`.
- **Test coverage**:
  `internal/gateway/live_turn_prompt_test.go::TestLiveTurn_SystemPrompt_IncludesSOUL`,
  `..._IncludesProjectContext`,
  `..._BlockOrder`,
  `..._ChannelNeutral`,
  `..._MissingProfileNoPanic`,
  `..._ThreatBlocked`,
  and the integration test
  `..._TelegramFinalProviderRequestIncludesOperatorContext`.
- **Lane**: D.
- **Progress row**: `Live-turn SOUL.md and project context wiring (channel-neutral)`
  (complete).

### 24. Final provider request includes Gormes identity (integration test, not unit)

- **Hermes source**: end-to-end behavior of run_agent.py.
- **Gormes source**: at the gateway level, the test
  `internal/gateway/live_turn_prompt_test.go::TestLiveTurn_TelegramFinalProviderRequestIncludesOperatorContext`
  (lines 583-676) drives a fake Telegram inbound through the real
  `gateway.Manager` and a `hermes.NewMockClient` provider, captures the
  outbound `ChatRequest`, and asserts the system prompt contains
  "You are Gormes, not ChatGPT.", "# User\nName: Juan",
  "# Memory\nGormes identity must persist.", and
  "## Current Session Context".
- **Test result**: `go test ./internal/gateway -run TestLiveTurn_TelegramFinalProviderRequestIncludesOperatorContext -count=1` → ok.
- **Production command evidence**:
  `internal/app/gormescmd/telegram_test.go::TestTelegramProductionProviderPayloadIncludesOperatorContext`
  runs the production Telegram manager-config path with a mock provider and
  asserts the final provider request contains SOUL.md identity, USER.md,
  MEMORY.md, model/provider metadata, session context, and the Telegram user
  message.
- **Status**: `parity`.
- **Lane**: D (P0).
- **Progress row**: reconciled by gateway and production Telegram provider
  payload golden tests.
- **Evidence**:
  `go test ./internal/gateway -run TestLiveTurn_TelegramFinalProviderRequestIncludesOperatorContext -count=1`
  and `go test ./internal/app/gormescmd -run TestTelegramProductionProviderPayloadIncludesOperatorContext -count=1`.

### 25. Final provider request includes USER.md / MEMORY.md

- **Gormes source**: `internal/gateway/live_turn_prompt.go:104-130`
  resolves the memory directory via `GORMES_CONTEXT_MEMORY_DIR` →
  `${GORMES_HOME}/memory` → `${GORMES_HOME}/memories` →
  `${HERMES_HOME}/memories` → `${HERMES_HOME}/memory` → `${HERMES_HOME}` →
  CWD ancestor → CWD ancestor `memory/` subdir. Wired into the same
  channel-neutral assembly site.
- **Status**: `parity`.
- **Test coverage**: same suite as #23, plus the integration test in
  #24.
- **Lane**: D.
- **Progress row**: `Live-turn USER.md and MEMORY.md durable user context block (channel-neutral)`
  (complete).

### 26. Final provider request includes AGENTS.md / project context

- Covered by `Live-turn SOUL.md and project context wiring (channel-neutral)`
  (complete) — `internal/llm/context_files.go` resolves
  HERMES.md → AGENTS.md → CLAUDE.md → .cursorrules from CWD ancestors.
- **Status**: `parity`.

### 27. Final provider request includes skill guidance

- **Gormes source**: `internal/kernel/turn_request_assembly.go` calls
  `k.cfg.Skills.BuildSkillBlock`, injects `llm.SkillsGuidance` followed
  by the selected skill block, and records selected skill usage when a usage
  recorder is configured.
- **Status**: `parity`.
- **Lane**: D.
- **Progress row**: covered by final provider request guidance tests.
- **Evidence**: `internal/kernel/turn_request_assembly_test.go` and
  `internal/kernel/guidance_test.go`.

### 28. Final provider request includes platform context

- **Gormes source**: `internal/gateway/session_context.go::BuildSessionContextPrompt`
  emits `## Current Session Context` with Source, User ID, Session Key,
  Session ID, Delivery Targets — covered by the integration test in #24
  (asserts substring `**Source:** telegram chat \`42\``).
- **Status**: `parity`.

### 29. Final provider request includes session metadata

- Same source as #28; `BuildSessionContextPrompt` includes session
  identity. **Status**: `parity`.

### 30. Final provider request includes timestamp / timezone / model / provider

- **Gormes source**: `Live-turn timestamp + model/provider/session
  metadata block + self-help guidance (channel-neutral)` is `complete`
  (per progress.json) and the metadata block is built by
  `internal/llm/live_turn_metadata.go` and assembled in
  `internal/gateway/live_turn_prompt.go::assembleLiveTurnPrompt`.
- **Status**: `parity`.

### 31. Final provider request includes tool guidance (memory / session_search / skills constants)

- **Hermes source**: `../hermes-agent/agent/prompt_builder.py` injects
  per-tool guidance constants when a toolset is enabled.
- **Gormes source**: `internal/kernel/turn_request_assembly.go` injects
  `MemoryGuidance` with recall context, `SessionSearchGuidance` when the
  `session_search` tool is present, `SkillsGuidance` with selected skills,
  and model/tool-use/research guidance from `liveTurnGuidanceBlocks`.
- **Status**: `parity`.
- **Progress row**: covered by final provider request guidance tests.
- **Evidence**: `go test ./internal/kernel ./internal/llm ./internal/llm/guidance/... -run 'Guidance|Skills|TurnRequest|PromptAssembly|ToolUse|MemoryGuidance|SessionSearch' -count=1`.

### 32. Context-file production discovery (paths)

- **Discovery order** in `internal/gateway/live_turn_prompt.go:80-130`:
  1. `${GORMES_HOME}/SOUL.md` (default `~/.gormes/SOUL.md`)
  2. `${GORMES_HOME}/memory/SOUL.md` (migrated layout)
  3. `${HERMES_HOME}/SOUL.md` if env is set (Hermes-profile fallback)
  4. CWD ancestor with `SOUL.md`
- **Memory directory**: `GORMES_CONTEXT_MEMORY_DIR` override →
  `${GORMES_HOME}/memory` → `${GORMES_HOME}/memories` →
  `${HERMES_HOME}/memories|memory|.` → CWD ancestor → CWD ancestor's
  `memory/` subdir.
- **Status**: `parity`.
- **Test coverage**: live_turn_prompt_test fixtures for each branch.

### 33. Session ID generation (Hermes-style)

- **Hermes source**: `../hermes-agent/gateway/session.py:735`
  `session_id = f"{now.strftime('%Y%m%d_%H%M%S')}_{uuid.uuid4().hex[:8]}"`.
- **Gormes source**: `internal/gateway/manager.go::refreshConversationalSessionMetadata`
  generates `YYYYMMDD_HHMMSS_<suffix>` conversational session IDs on first
  submit, persists the chat/session mapping, and reuses the mapped ID for
  subsequent turns. `/status` fallback IDs use the same timestamp-prefix
  shape when called before a turn.
- **Status**: `parity`.
- **Lane**: C.
- **Evidence**: `internal/gateway/manager_test.go::TestManager_Inbound_SubmitCreatesAndRefreshesConversationalSessionMetadata`.

### 34. Session title auto-generation

- **Hermes source**: `../hermes-agent/run_agent.py:7616`
  `self._session_db.set_session_title(self.session_id, new_title)` after the
  title model produces a candidate title.
- **Gormes source**: `internal/gateway/auto_title_wiring.go` runs auto-title
  after PhaseIdle frames using the configured `TitleModel` and `TitleStore`;
  `internal/app/gormescmd/gateway_test.go` proves production gateway config
  wires both seams.
- **Status**: `parity`.
- **Lane**: C.
- **Evidence**: `internal/gateway/auto_title_wiring_test.go` and
  `internal/app/gormescmd/gateway_test.go::TestGatewayManagerConfigWiresTitleModelAndStore`.

### 35. Session title persistence (across restarts)

- **Hermes source**: `hermes_state.py::set_session_title` /
  `get_session_title` persist session titles in state.
- **Gormes source**: `internal/persistence/session` metadata stores titles;
  gateway manual-title and auto-title paths write through the same metadata
  store, and `/status` reads existing titles without overwriting manual titles.
- **Status**: `parity`.
- **Lane**: C.
- **Evidence**: `internal/gateway/title_command_test.go`,
  `internal/gateway/auto_title_wiring_test.go`, and status tests.

### 36. Manual title preservation (don't overwrite)

- **Gormes source**: `status_command.go` preserves existing metadata titles
  and `session.PerformAutoTitle` skips sessions that already have a title.
- **Status**: `parity`.

### 37. Created timestamp (no `(unknown)`)

- **Gormes source**: fresh conversational sessions write `CreatedAt` metadata
  and `/status` reads that metadata before falling back to timestamp-prefix
  parsing for Hermes-shaped IDs.
- **Status**: `parity`.
- **Lane**: C.
- **Evidence**: `internal/gateway/manager_test.go::TestManager_Inbound_SubmitCreatesAndRefreshesConversationalSessionMetadata`.

### 38. Last activity timestamp

- **Gormes source**: conversational session metadata refresh preserves
  `CreatedAt` and updates `UpdatedAt` on subsequent turns; `/status` renders
  `UpdatedAt` as Last Activity.
- **Status**: `parity`.
- **Lane**: C.
- **Evidence**: `internal/gateway/manager_test.go::TestManager_Inbound_SubmitCreatesAndRefreshesConversationalSessionMetadata`.

### 39. Token accounting accuracy

- **Gormes source**: `internal/gateway/usage_command.go::rememberUsageFrame`
  persists positive render-frame token totals into session metadata, and
  `status_command.go::formatGatewayStatus` renders durable metadata totals
  when they exceed the current frame.
- **Status**: `parity`.
- **Lane**: C.
- **Evidence**: `internal/gateway/status_command_test.go::TestStatusCommand_PersistsAndRendersAccumulatedSessionTokenTotals`.

### 40. Agent Running status accuracy

- **Gormes source**: `m.hasActiveTurn()` covers it.
- **Status**: `parity`.

### 41. Connected Platforms accuracy

- **Gormes source**: `m.connectedPlatforms()` enumerates registered
  channels; falls back to the inbound platform if empty.
- **Status**: `parity`.

### 42. Session resume

- **Gormes source**: `Durable pause/resume intent contract` (complete);
  `manager.go::submitPinned` injects `resumePendingNote` when a session
  was interrupted.
- **Status**: `parity`.

### 43. Session reset / new / retry / undo

- **Hermes source**: `/new` (alias `/reset`), `/retry`, `/undo` are all
  handlers in `gateway/run.py` and `cli.py`.
- **Gormes source**: `/new` is wired through EventReset and `/reset` is a
  registry alias. `/retry` and `/undo [N]` are live gateway handlers in
  `internal/gateway/command_dispatch.go`; they use the runtime-wired SQLite
  `SessionHistoryStore` from `internal/gateway/session_history_store.go` to
  rewrite or rewind the durable `memory.db` transcript before resuming the
  kernel. Covered by `internal/gateway/session_history_store_test.go` and
  command-dispatch tests.
- **Status**: `parity`.
- **Lane**: B/H.
- **Progress row**: Covered by the gateway slash registry parity sweep.

### 44-48. Memory/Goncho areas

Rows 44 and 46 are covered: `internal/tools/memory/durable/tool_test.go`
validates Hermes-compatible add/read/replace/remove behavior over USER.md and
MEMORY.md, `internal/tools/goncho/...` validates Honcho/Goncho tool catalogs
and Memory V1 transcripts, and `internal/memory/goncho/...` plus
`internal/memory/lifecycle/...` validate local Goncho markdown reload/export,
conflict handling, migrations, lifecycle startup/shutdown, and recall/storage
surfaces. **Status: parity for rows 44 and 46.**

Row 48 is covered: `internal/llm/context_compressor_*_test.go` validates
summary lineage, protected head/tail planning, provider-backed summarization,
context references, and no-secret error evidence; `internal/llm/manual_compression_feedback_test.go`
validates Hermes-style manual compression feedback; `internal/kernel/manual_compress_test.go`
validates explicit history rewrite and compression-boundary callbacks; and
`internal/gateway/command_dispatch_test.go` binds operator-facing `/compress`
with focus parsing, disabled evidence, and error redaction. **Status: parity.**
**Lane: E.**

Row 45 is covered: `liveprompt.Assemble` calls
`llm.BuildDurableUserContextPrompt`, production seams resolve USER.md and
MEMORY.md from the Gormes memory directory/workspace, and
`internal/gateway/live_turn_prompt_test.go` verifies block order,
channel-neutral insertion, missing-file elision, and threat blocking. **Status:
parity.**

### 49. GONCHO branding

- **Gormes source**: `internal/goncho/` ships; default workspace and
  observer peer are both `"gormes"`. There's no Kancho or Honcho-rename
  residue in the runtime.
- **Status**: `parity`.

### 50-60. Provider / auth / runtime

These map to existing planned/complete rows:
- 50 Provider registry parity → registry/aliases plus OpenRouter,
  Google Code Assist, Gemini Cloud Code, Codex Responses, Bedrock
  runtime/SigV4/stale-client, Gemini native transport/runtime, and
  google-gemini-cli OAuth login/runtime/refresh are covered by provider,
  credential, and CLI tests.
- 51 Auth status → `cmd/gormes/auth_status_command_test.go` (complete).
- 52 Auth add/list/remove/logout → `cmd/gormes/auth_command_test.go`
  (complete).
- 53 Codex device-code/OAuth → `gormes auth add openai-codex --type oauth`
  covers Codex CLI import, expired-import fallback, device user-code,
  poll, token exchange, redacted output, and credential-pool persistence
  (`auth_oauth_runtime_test.go`). parity.
- 54 Credential pool → bare `gormes auth` lists provider credential pools,
  credential counts, redacted entries, current marker, and Bedrock identity
  status without leaking access or refresh tokens (`auth_runtime_test.go`).
  parity.
- 55 Provider request shape → kernel.go uses `hermes.ChatRequest`. parity.
- 56 Provider stream handling → kernel.go OpenStream + retry. parity.
- 57 Retry behavior → `kernel.NewRetryBudget` + `RetryStatus`. parity.
- 58 Health checks → `RuntimeStatusStore.ReadValidatedRuntimeStatusSnapshot`
  + procRuntimeProcessTable. parity.
- 59 Rate-limit evidence → `internal/llm/rate_limit_tracker*`,
  `internal/llm/account_usage_test.go`, `internal/gateway/usage_command_test.go`,
  and `internal/app/gormescmd/gateway_test.go` cover header-derived rate-limit
  status, Codex/Anthropic/OpenRouter account-usage windows, degraded evidence,
  redaction, and `/usage` rendering. parity.
- 60 Redacted diagnostics → `render_test`, `internal/audit/`. parity.

### 61-67. CLI / config

- 61 CLI command tree → `internal/platform/cli/gormescli/contractruntime`
  derives module ownership from the live Cobra tree, `internal/platform/cli/commands/registry`
  validates slash-command names, aliases, active-turn policies, and unavailable
  surfaces, and row-backed command tests cover Hermes-compatible placeholder
  commands. `go test ./internal/platform/cli/gormescli ./internal/platform/cli/commands/... -run 'Contract|CommandManifest|Registry|CommandTree|ParentSubcommands|RootExecute|HermesRowBacked|SetupRegistry' -count=1`
  passes. parity.
- 62 CLI help text → `gatewayHelpText`. parity.
- 63 Active-turn CLI policy → `CLI command registry parity + active-turn busy policy` (complete).
- 64 Provider/config resolution → `internal/config/auth/provider_credential_resolution_test.go`,
  `internal/config/integration/providers/provider_credential_resolution_test.go`,
  profile/setup app tests, doctor profile tests, setup profile tests, and
  provider fallback-config tests cover route/env, SecretRef, inline,
  manifest env, Codex OAuth, credential-pool/profile filtering, setup-first,
  setup-profile, and fallback chain resolution. parity.
- 65 Config path discovery → `config.GormesHome()` + `config.ConfigPath()`. parity.
- 66 Config show/check/edit/migrate → `config_closeout_test.go`,
  `config_command_test.go`, `config_get*_test.go`, and
  `config_profile_migration_test.go` cover native show/get, check, edit,
  schema-version evidence, dry-run/apply migration JSON, backup creation,
  and no-secret output. parity.
- 67 Diagnostics → `doctor_*_test.go`, `status_command_test.go`,
  `logs_test.go`, `restore_command_test.go`, backup/restore validation tests,
  and `hermes_rowbacked_commands_test.go` cover doctor/status/logs,
  restore inventory/extract JSON, backup creation/dry-run JSON, and zip
  validation. parity.

### 68. Browser tool contract

- **Gormes source**: `internal/tools/browser_contract.go::BrowserAction`
  + `ValidateBrowserAction` ships; `internal/tools/browser_harness_tools.go::NewBrowserHarnessTools`
  exposes 12 `browser_*` tools (back, cdp, click, console, dialog,
  get_images, navigate, press, scroll, snapshot, type, vision).
- **Status**: `parity` for the contract surface; the runtime backend is
  the planned row.
- **Lane**: G.
- **Progress row**: `go-browser-harness Chromedp action backend` (planned).

### 69-75. Browser features (snapshots, DOM extraction, screenshots, console, navigation, click/type/scroll, session lifecycle)

These are now covered by the in-process browser backend rather than a future
sibling binary. `internal/tools/browser_harness_backend.go` dispatches accepted
browser actions through a fakeable CDP transport; production wiring can use the
Chromedp remote transport in `browser_harness_chromedp_transport.go`. Tests in
`internal/tools/browser_use_harness_bridge_test.go`,
`internal/tools/browser_harness_tools_test.go`, and browser contract packages
cover navigate, snapshot/DOM text, screenshot artifacts, console shaping,
click/type/scroll/press, back, dialog/CDP/vision/get-images, unavailable
evidence, Browser Use bridge request shaping, and target-session persistence.

**Status: parity across rows 69-75.** **Lane: G.**

### 76. Artifact budgets

- **Gormes source**: `tools.ToolResultBudgetConfig` + tests. parity.

### 77. Private URL / SSRF safety

- **Gormes source**: `internal/tools/browser_ssrf_guard.go` + tests.
  parity.

### 78. Browser tool result formatting compatibility with Hermes

- **Hermes source**: `gateway/run.py::_render_browser_tool_result` +
  channel-specific renderers.
- **Gormes source**: `internal/tools/browser_harness_backend.go`,
  `internal/tools/browser_contract.go`, and `internal/gateway/rendering/browserartifacts`
  produce Hermes-shaped text/artifact/console evidence with bounded results.
- **Status**: `parity`.

### 79. Browser channel rendering for Telegram

- **Gormes source**: `internal/gateway/rendering/browserartifacts/telegram.go`
  and `internal/gateway/rendering/telegram_browser_render_test.go` cover
  Telegram browser artifact rendering.
- **Status**: `parity`.

### 80. Go browser harness integration lane (placeholder for future repo)

- **Gormes source**: the future sibling-binary lane has been superseded by
  the in-process CDP/Chromedp backend and Browser Use bridge. The same JSON
  action contract remains at `BrowserHarnessActionRequest`, but Gormes can now
  execute it directly through `BrowserHarnessActionBackend`.
- **Status**: `parity`.
- **Lane**: G.
- **Progress row**: covered by the in-process browser backend and Browser Use bridge tests.

## Additional row mapping

The audit identifies three additional gaps that should be reconciled through
existing progress rows or small planner refinements before implementation.

1. `Telegram MarkdownV2 parse-mode rendering closeout` — Lane B.
   Fold into `Gateway stream/tool trace formatting fixture matrix` unless a
   planner pass splits it into a narrower Telegram renderer row.
2. `Gateway slash registry parity sweep (recognized-name expansion)` —
   Lane B/H. Fold into `Gateway active-turn policy manifest closeout` and
   the existing Hermes CLI command-tree rows.
3. `Go browser harness binary repo + integration lane (placeholder)` —
   Lane G. Covered by `go-browser-harness Chromedp action backend` and the
   validated Browser Use / Go browser harness bridge rows.

## Coordinator brief

See [swarm-feature-parity-audit](../swarm-feature-parity-audit/) for the
dispatch order, per-lane backlog summary, cross-lane dependencies, and risks.
