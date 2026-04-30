---
title: "Hermes <-> Gormes parity matrix"
description: "Durable parity inventory for Hermes-in-Go (Gormes) delivery. One row per parity area with upstream + Gormes refs, status classification, lane assignment, and progress.json row pointer."
date: 2026-04-29
draft: false
---

# Hermes <-> Gormes parity matrix

Audit pass: 2026-04-29 (lane A parity-auditor).

This document is the durable parity inventory that replaces reactive
screenshot-driven fixes. Every row names one Hermes/Sidon parity area, cites
upstream + Gormes evidence, classifies the gap, and points at the
progress.json row that future implementation lanes (B-H) execute.

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
| 1 | `/status` heading + fields + reply quoting + gateway-side execution | partial | B/C | Telegram /status Hermes-format closeout (planned) |
| 2 | `/help` and Telegram setMyCommands menu | partial | B | Telegram dynamic BotCommand menu wiring (planned) |
| 3 | Slash command registry (Hermes vs Gormes command names + aliases) | partial | B/H | Hermes CLI command-tree parity manifest (complete inventory); Telegram slash registry handler-coverage gap (planner refinement) |
| 4 | Unknown command behavior | parity | B | Active-turn slash bypass tests cover unknown |
| 5 | Unavailable command behavior | parity | B | active_turn_command_bypass_test (covered) |
| 6 | Active-turn slash bypass behavior | parity | B | Gateway active-turn policy manifest closeout (planned for closeout edge cases) |
| 7 | Busy/admission behavior | parity | B | Active-turn follow-up queue + late-arrival drain policy (complete) |
| 8 | Telegram reply quoting (every bot response) | partial | B | Telegram reply_to_mode and reply-context parity (planned) |
| 9 | Telegram message threading (forum topic threads) | partial | B | Telegram reply_to_mode and reply-context parity (planned) |
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
| 20 | Tool trace formatting (memory/search_files/read_file/patch/terminal icons) | partial | B | Gateway stream/tool trace formatting fixture matrix (planned) |
| 21 | Duplicate collapse | partial | B | Restart duplicate-suppression covers operator path; chat-text dedup not implemented (new low-priority gap) |
| 22 | Mobile-readable formatting | parity for MarkdownV2 parse-mode; partial for broader style polish | B | Telegram MarkdownV2 parse-mode rendering closeout (complete); Gateway stream/tool trace formatting fixture matrix (planned) |
| 23 | Identity / persona ("My name is Gormes" not "ChatGPT") | parity | D | Live-turn SOUL.md and project context wiring (channel-neutral) (complete) |
| 24 | Final provider request includes Gormes identity (integration test) | parity at gateway, planned at production cmd | D | Telegram production live-turn provider payload golden (planned, P0) |
| 25 | Final provider request includes USER.md / MEMORY.md | parity at gateway | D | Live-turn USER.md and MEMORY.md durable user context block (channel-neutral) (complete) |
| 26 | Final provider request includes AGENTS.md / project context | parity at gateway | D | Live-turn SOUL.md and project context wiring (complete) |
| 27 | Final provider request includes skill guidance | partial | D | Toolset-aware skills prompt snapshot (planned, blocking parent umbrella) |
| 28 | Final provider request includes platform context | parity | D | Session context BuildSessionContextPrompt (covered by gateway test fixture) |
| 29 | Final provider request includes session metadata | parity | D | BuildSessionContextPrompt covers this |
| 30 | Final provider request includes timestamp/timezone/model/provider | parity at gateway | D | Live-turn timestamp + model/provider/session metadata block + self-help guidance (complete) |
| 31 | Final provider request includes tool guidance constants | partial | D | Live-turn model/tool guidance wiring (planned) |
| 32 | Context-file production discovery (~/.gormes/SOUL.md, workspace fallback, Hermes profile fallback) | parity | D | Live-turn SOUL/AGENTS wiring uses GORMES_HOME → memory/ → HERMES_HOME → CWD ancestor chain |
| 33 | Session ID generation (Hermes-style format) | partial | C | Gateway conversational session metadata refresh (planned) |
| 34 | Session title auto-generation | missing wiring | C | Gateway conversational session metadata refresh (planned); hermes.GenerateTitle exists but is unwired |
| 35 | Session title persistence (across restarts) | partial | C | Gateway conversational session metadata refresh (planned) |
| 36 | Manual title preservation (don't overwrite) | parity (in /status pull-through) | C | status_command.go preserves existing meta.Title |
| 37 | Created timestamp (no `(unknown)`) | partial | C | status_command.go uses session-ID prefix, returns `(unknown)` if not a known format. Gateway conversational session metadata refresh (planned) |
| 38 | Last activity timestamp | partial | C | Gateway conversational session metadata refresh (planned) |
| 39 | Token accounting accuracy | partial | C | Gateway session token accounting parity (planned) |
| 40 | Agent Running status accuracy | parity | C | hasActiveTurn() drives status |
| 41 | Connected Platforms accuracy | parity | C | connectedPlatforms() drives status |
| 42 | Session resume | parity | C | Durable pause/resume intent contract (complete) |
| 43 | Session reset/new/retry/undo | partial | C | /new shipped via EventReset; /retry, /undo unwired (49-file CLI tree port umbrella) |
| 44 | Memory/Goncho durable user memory | partial | E | Hermes memory tool over Goncho/local durable store (planned) |
| 45 | Memory prompt insertion | parity at gateway | D/E | Live-turn USER.md and MEMORY.md durable user context block (complete) |
| 46 | Memory write/read lifecycle | partial | E | Goncho memory provider lifecycle adapter (planned) |
| 47 | Memory redaction | parity | E | Existing redaction fixtures in internal/memory and internal/audit |
| 48 | Session summaries / compression boundary | partial | E | Long session management, Context compression, Manual compression feedback + context references, Kernel compression-boundary callback binding (all planned) |
| 49 | GONCHO branding (not Kancho, not Honcho-renamed) | parity | E | internal/goncho/ ships and tests assert workspace=gormes peer=gormes |
| 50 | Provider registry parity | partial | F | Bedrock / Gemini Cloud Code / OpenRouter / Google Code Assist / Codex provider runtime rows planned |
| 51 | Auth status command | parity | F/H | cmd/gormes/auth_status_command_test.go covers it |
| 52 | Auth add/list/remove/logout commands | parity | F/H | cmd/gormes/auth_command_test.go covers per-provider lifecycle |
| 53 | Codex device-code/OAuth path | partial | F/H | Hermes auth OAuth provider adapters (planned) |
| 54 | Credential pool | partial | F | Gormes auth bare interactive credential-pool readout (planned) |
| 55 | Provider request shape | parity | F | hermes.ChatRequest used end-to-end |
| 56 | Provider stream handling | parity | F | OpenStream + retry budget shipped in kernel |
| 57 | Retry behavior | parity | F | NewRetryBudget + retryStatus shipped |
| 58 | Health checks | parity | F | RuntimeStatusStore + ReadValidatedRuntimeStatusSnapshot |
| 59 | Rate-limit evidence | partial | F | Provider account-usage rows planned |
| 60 | Redacted diagnostics (no token leakage) | parity | F | render_test sanitize tests, audit ledger redaction |
| 61 | CLI command tree (Hermes vs Gormes surface) | partial | H | 49-file CLI tree port; Hermes CLI command-tree parity manifest (complete inventory) |
| 62 | CLI help text | parity | H | gatewayHelpText() + GatewayHelpLines() drive consistent help |
| 63 | Active-turn CLI policy | parity | H | CLI command registry parity + active-turn busy policy (complete) |
| 64 | Provider/config resolution | partial | H | Config, profile, auth, and setup command surfaces (planned) |
| 65 | Config path discovery | parity | H | config.GormesHome() + ConfigPath() shipped |
| 66 | Config show/check/edit/migrate | partial | H | Gormes config edit/check/native schema-migrate closeout (planned) |
| 67 | Diagnostics | partial | H | Diagnostics, backup, logs, and status CLI (planned) |
| 68 | Browser tool contract | parity | G | tools/browser_contract.go + browser_harness_tools.go shipped |
| 69 | Browser snapshots | partial | G | go-browser-harness Chromedp action backend (planned) |
| 70 | DOM text extraction | partial | G | go-browser-harness Chromedp action backend (planned) |
| 71 | Screenshot artifacts | partial | G | Browser artifact and console render contract (planned) |
| 72 | Console logs | partial | G | Browser artifact and console render contract (planned) |
| 73 | Browser navigation | partial | G | go-browser-harness Chromedp action backend (planned) |
| 74 | Browser click/type/scroll | partial | G | go-browser-harness Chromedp action backend (planned) |
| 75 | Browser session lifecycle | partial | G | go-browser-harness Chromedp action backend (planned) |
| 76 | Artifact budgets | parity | G | tools.ToolResultBudgetConfig shipped + browser_harness_tools_test |
| 77 | Private URL / SSRF safety | parity | G | tools/browser_ssrf_guard.go + browser_ssrf_guard_test.go |
| 78 | Browser tool result formatting compatibility with Hermes | partial | G | Browser artifact and console render contract (planned) |
| 79 | Browser channel rendering for Telegram | partial | G | Telegram browser artifact rendering (planned) |
| 80 | Go browser harness integration lane (placeholder for future repo) | missing | G | go-browser-harness binary repo lane (new umbrella) |

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
- **Gormes source**: `internal/gateway/status_command.go:18-69`
  (`formatGatewayStatus`). Renders the same field set in the same order
  but with plain `Session ID:` / `Title:` labels (no bold), Created/Last
  Activity using local 2006-01-02 15:04 format vs Hermes's strftime, and
  no `Yes ⚡` decoration on the Agent Running row.
- **Reply quoting**: `internal/gateway/status_command.go:15` calls
  `m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, ...)` so `/status`
  replies do thread to the triggering message via Telegram's
  `ReplyToMessageID`. Verified by
  `internal/gateway/status_command_test.go::statusReplyChannel`.
- **Gateway-side execution (no model)**: confirmed by
  `manager.go:670+776` dispatch on `EventStatus` short-circuiting
  before any kernel.Submit.
- **Status**: `partial`. Field set + reply quoting + bypass are all
  parity. Field labels and timestamp formatting differ (no bold;
  timezone-local instead of UTC strftime), and Title is omitted when
  not set instead of generating one. The auto-title gap is the single
  biggest visible difference.
- **Test coverage**:
  `internal/gateway/status_command_test.go`,
  `internal/gateway/statusview_test.go`,
  `internal/gateway/active_turn_command_bypass_test.go::TestManager_ActiveTurnSlashCommandBypass_ChannelNeutral`.
- **Lane**: B (Telegram UX), C (session metadata).
- **Progress row**: `Telegram /status Hermes-format closeout` (planned, P0).
- **Evidence command**:
  `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway -run Status -count=1`
- **Live Telegram evidence**: when the user sends `/status`, Gormes
  replies (threaded to the request) with the field block. MarkdownV2
  parse-mode is now set on Telegram sends, replies, and edits, so the
  bold field labels render instead of appearing as literal escaped
  markup. The remaining visible delta from Hermes is Title quality:
  live status can still show a synthetic "Telegram conversation with
  <user_id>" title until the session/title rows finish.

### 2. /help and slash menu (setMyCommands / getMyCommands)

- **Hermes source**: `../hermes-agent/gateway/platforms/telegram.py`
  bot startup builds the menu from `COMMANDS_BY_CATEGORY` and calls
  `setMyCommands`; `../hermes-agent/gateway/run.py:4937` formats
  `📖 **Hermes Commands**`.
- **Gormes source**: `internal/channels/telegram/bot.go:49-65`
  (`registerCommands`) calls `setMyCommands` once per startup using
  `gateway.TelegramBotCommands()` from `internal/gateway/commands.go:183`.
  Help output is built by `gatewayHelpText()` at
  `internal/gateway/commands.go:177-179`.
- **Status**: `partial`. setMyCommands is called once at startup with the
  static registry. Hermes also registers per-platform variants and
  refreshes when plugins/skills register dynamic commands. Gormes has
  the helper (`TelegramBotCommandsWith`) but doesn't drive it from the
  live runtime yet.
- **Test coverage**: `internal/channels/telegram/bot_test.go::TestBot_RegistersTelegramCommands`.
- **Lane**: B.
- **Progress row**: `Telegram dynamic BotCommand menu wiring` (planned).
- **Evidence**:
  `grep -n "TelegramBotCommands\|TelegramBotCommandsWith" internal/gateway/commands.go internal/channels/telegram/bot.go`.

### 3. Slash command registry (Hermes vs Gormes command names + aliases)

- **Hermes source**: `../hermes-agent/hermes_cli/commands.py:59-175`
  (`COMMAND_REGISTRY`) defines ~50 commands with categories, aliases,
  args_hint, subcommands, and cli_only / gateway_only flags.
- **Gormes source**: `internal/gateway/commands.go:38-90`
  (`CommandRegistry`) defines 20 commands. Of those, 13 carry
  `ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable`, meaning the
  registry recognizes them so an `/retry` request gets the friendly
  "/retry is recognized but unavailable" reply instead of "unknown
  command", but no handler is wired.
- **Missing entirely** (not even in the unavailable set): `clear`,
  `history`, `save`, `profile`, `sethome`, `set-home`, `resume`,
  `config`, `model`, `provider`, `gquota`, `personality`, `statusbar`,
  `verbose`, `yolo`, `reasoning`, `fast`, `skin`, `voice`, `tools`,
  `toolsets`, `skills`, `cron`, `reload`, `reload-mcp`, `reload_mcp`,
  `browser`, `plugins`, `commands`, `insights`, `platforms`, `gateway`
  (alias of platforms), `copy`, `paste`, `image`, `update`, `debug`,
  `quit`, `exit`. Aliases `reset` (for /new) and `start` (for /help)
  are present on Gormes; aliases `bg`, `q`, `tasks`, `snap`, `fork`
  are present.
- **Status**: `partial`. The `Hermes CLI command-tree parity manifest`
  row at `cmd/gormes/hermes_cli_parity_test.go::TestHermesCLIParityManifest`
  is `complete` and inventories every Hermes command with a
  classification, but the runtime registry (`CommandRegistry`) does not
  yet expose every classified-as-row-backed entry as a recognized name,
  so unknown-command responses are louder than they should be.
- **Test coverage**: `cmd/gormes/hermes_cli_parity_test.go`,
  `internal/gateway/commands_test.go`,
  `internal/gateway/active_turn_command_bypass_test.go`.
- **Lane**: B (gateway slash visibility), H (CLI handlers).
- **Progress row**: existing `Hermes CLI command-tree parity manifest`
  (complete) + `49-file CLI tree port` umbrella + new
  `Gateway slash registry parity sweep (recognized-name expansion)`
  (created in this audit).
- **Evidence**:
  `diff <(grep -oE '\"name\":\\s*\"[a-z_-]+\"' internal/gateway/commands.go | sort) <(grep -oE 'CommandDef\\(\"[a-z_-]+\"' /home/xel/.hermes/hermes-agent/hermes_cli/commands.py | sort)`.

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
- **Gormes source**: `internal/channels/telegram/bot.go:140-157`
  implements `SendReply`; `internal/gateway/manager.go:932-946`
  (`sendNoEdit`) routes terminal frames through
  `sendWithHooksReply(..., replyToMsgID, ...)`. However, the
  `ReplyMode` configuration (first/all/off) is not yet implemented:
  Gormes always replies on whatever the caller gives it.
- **Status**: `partial`. Quoting works; mode toggle missing; deleted-target
  fallback (Hermes retries without reply_to_id when "message to be
  replied not found") not implemented.
- **Test coverage**: `internal/channels/telegram/bot_test.go::TestBot_SendReplySetsTelegramReplyToMessageID`.
- **Lane**: B.
- **Progress row**: `Telegram reply_to_mode and reply-context parity`
  (planned).

### 9. Telegram message threading

- **Hermes source**: forum topic threads via `message_thread_id` on
  Telegram bot API. `../hermes-agent/gateway/platforms/telegram.py`
  forum-topic helpers + `_should_thread_reply`.
- **Gormes source**: `internal/channels/telegram/bot.go` does not yet
  pass `message_thread_id` on outbound messages; only `ReplyToMessageID`.
- **Status**: `partial`.
- **Lane**: B.
- **Progress row**: `Telegram reply_to_mode and reply-context parity`
  (planned, scope expansion).

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
- **Gormes source**: `internal/gateway/render.go:98-114`
  (`formatToolTracePlain`) hardcodes the same icons:
  - `🧠` memory
  - `🔎` search_files / browser_navigate / browser_snapshot / web_search
  - `📖` read_file
  - `🔧` patch / fallback / unknown
  - `🖥` terminal
- **Gap**: Gormes hardcodes the icon mapping. Hermes lets per-tool
  registrations declare their emoji, and the skin engine
  (`hermes_cli/skin_engine.py`) supports `tool_emojis` overrides.
  Gormes has no skin engine.
- **Status**: `partial`. Icon set matches; declarative override surface
  missing.
- **Test coverage**: `internal/gateway/render_test.go::TestFormatStreamPlain_IncludesHermesStyleToolTrace`.
- **Lane**: B.
- **Progress row**: `Gateway stream/tool trace formatting fixture matrix`
  (planned).

### 21. Duplicate collapse

- **Hermes source**: gateway suppresses duplicate restart-takeover
  markers; CLI deduplicates rapid repeated submits.
- **Gormes source**: `manager.go::restartDuplicate` + RuntimeStatusUpdate
  duplicate evidence; chat-level user-message deduplication is not
  implemented in Gormes (rapid `/help /help` will produce two replies).
- **Status**: `partial`. Operator-side covered; chat-side not.
- **Lane**: B.
- **Progress row**: no dedicated row exists. Low-priority gap; tracked
  here as awareness only — does NOT need a new row unless Juan's smoke
  testing surfaces it as user-visible noise.

### 22. Mobile-readable formatting

- **Hermes source**: MarkdownV2 enables bold/italic/code on mobile.
- **Gormes source**: #10 is now complete for Telegram ParseMode and
  parse-error fallback, so mobile clients receive the same MarkdownV2
  transport contract. Remaining mobile readability work is broader
  renderer/content polish (tool traces, long final layout, status title
  quality), not parse-mode wiring.
- **Status**: `partial` overall; `parity` for MarkdownV2 parse-mode.
- **Lane**: B.
- **Progress row**: `Telegram MarkdownV2 parse-mode rendering closeout`
  (complete) plus `Gateway stream/tool trace formatting fixture matrix`
  for remaining presentation polish.

### 23. Identity / persona ("My name is Gormes" not "ChatGPT")

- **Hermes source**: `../hermes-agent/run_agent.py` + `agent/prompt_builder.py`
  loads SOUL.md and project-context files into the system prompt.
- **Gormes source**: `internal/hermes/context_files.go::BuildContextFilesPrompt`
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
- **Gap**: this exercises the gateway path with a synthetic Manager
  config. The production binary entrypoint `cmd/gormes/telegram.go` has
  no equivalent capture-the-final-request golden test.
- **Status**: `parity` at gateway, `partial` at the production binary.
- **Lane**: D (P0).
- **Progress row**: `Telegram production live-turn provider payload golden`
  (planned, P0).
- **Evidence**:
  `GOCACHE=/tmp/gormes-go-cache go test ./internal/gateway -run TestLiveTurn_TelegramFinalProviderRequestIncludesOperatorContext -count=1`.

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
  (complete) — `internal/hermes/context_files.go` resolves
  HERMES.md → AGENTS.md → CLAUDE.md → .cursorrules from CWD ancestors.
- **Status**: `parity`.

### 27. Final provider request includes skill guidance

- **Gormes source**: `internal/kernel/kernel.go:319-331` calls
  `k.cfg.Skills.BuildSkillBlock` and prepends a `system` message when
  the block is non-empty.
- **Gap**: the `k.cfg.Skills` runtime is wired in unit tests but the
  skills toolset registry that decides which skills appear is not yet
  fully ported — `Toolset-aware skills prompt snapshot` is the
  blocking row for this slice.
- **Status**: `partial`.
- **Lane**: D.
- **Progress row**: `Toolset-aware skills prompt snapshot` (planned, P0,
  blocking the umbrella `Hermes live-turn prompt assembly parity`).

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
  `internal/hermes/live_turn_metadata.go` and assembled in
  `internal/gateway/live_turn_prompt.go::assembleLiveTurnPrompt`.
- **Status**: `parity`.

### 31. Final provider request includes tool guidance (memory / session_search / skills constants)

- **Hermes source**: `../hermes-agent/agent/prompt_builder.py` injects
  per-tool guidance constants when a toolset is enabled.
- **Gormes source**: `internal/hermes/prompt_guidance.go` and
  `internal/hermes/prompt_builder_guidance.go` expose the constants but
  the live wiring into the assembled live-turn prompt is the planned
  row `Live-turn model/tool guidance wiring`.
- **Status**: `partial`.
- **Progress row**: `Live-turn model/tool guidance wiring` (planned).

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
  `session_id = f"{now.strftime('%Y%m%d_%H%M%S')}_{uuid.uuid4().hex[:8]}"` —
  random suffix, new ID per session.
- **Gormes source**: `internal/gateway/status_command.go:101-108`
  (`generateStatusSessionID`) uses `YYYYMMDD_HHMMSS_<fnv32(chat_key+user_id)>`
  — deterministic suffix based on chat identity. This is a fallback
  path — when no SessionMap mapping exists. The persisted mapping route
  is the primary one, so the format only matters when `/status` is
  called before a turn has run.
- **Gap**: Hermes can produce many session IDs for one chat (one per
  reset); Gormes' fallback always produces the same ID per
  (chat,user). This is acceptable for the fallback path because
  `/status` doesn't reset the session; but it's a documented
  divergence.
- **Status**: `partial`.
- **Lane**: C.
- **Progress row**: `Gateway conversational session metadata refresh`
  (planned).

### 34. Session title auto-generation

- **Hermes source**: `../hermes-agent/run_agent.py:7616`
  `self._session_db.set_session_title(self.session_id, new_title)`
  after the LLM produces a candidate title from the first user prompt.
- **Gormes source**: `internal/hermes/title_generator.go::GenerateTitle`
  ships and is unit-tested by `title_generator_test.go`. The TUI helper
  `internal/tui/auto_title.go::BuildAutoTitleRequest` decides
  eligibility. Neither is wired into the gateway live turn — the
  gateway never calls `GenerateTitle`.
- **Status**: `missing wiring`.
- **Test coverage** for what exists: `internal/hermes/title_generator_test.go`,
  `internal/tui/auto_title_test.go` (presumed; not enumerated here),
  `internal/gateway/title_failure_test.go`.
- **Lane**: C.
- **Progress row**: existing
  `Gateway conversational session metadata refresh` (planned). The
  title-generation wiring should be a child slice of that row.
- **Evidence**:
  `grep -rn "GenerateTitle" internal/gateway/manager*.go internal/gateway/turn*.go` returns no matches.

### 35. Session title persistence (across restarts)

- **Hermes source**: `hermes_state.py::set_session_title` /
  `get_session_title` — SQLite state.
- **Gormes source**: `internal/session/session.go::Metadata` carries
  Title; `internal/session/bbolt.go` persists metadata (presumed; same
  surface as `sessionMetadataWriter`/`sessionMetadataReader`). The
  `/status` command reads it back via `m.lookupSessionMetadata`.
- **Status**: `partial`. Persistence works; the fallback synthetic
  title ("Telegram conversation with <user_id>") gets written and is
  hard to distinguish from a real title.
- **Lane**: C.
- **Progress row**: `Gateway conversational session metadata refresh`
  (planned).

### 36. Manual title preservation (don't overwrite)

- **Gormes source**: `status_command.go:146-148`
  `if existing, ok := m.lookupSessionMetadata(...); ok && strings.TrimSpace(existing.Title) != "" { title = existing.Title }`
  — preserves manual title.
- **Status**: `parity`.

### 37. Created timestamp (no `(unknown)`)

- **Gormes source**: `status_command.go:30-35` reads `meta.CreatedAt`
  when set; otherwise `statusCreatedAt(sessionID)` parses the
  `YYYYMMDD_HHMMSS` prefix; otherwise returns "(unknown)".
- **Gap**: imported sessions or sessions whose IDs don't start with the
  date prefix render `(unknown)`.
- **Status**: `partial`.
- **Lane**: C.
- **Progress row**: `Gateway conversational session metadata refresh`
  (planned).

### 38. Last activity timestamp

- **Gormes source**: `status_command.go:33-35` reads
  `meta.UpdatedAt`. If session metadata is not persisted on every
  turn, the timestamp lags.
- **Status**: `partial`.
- **Lane**: C.
- **Progress row**: `Gateway conversational session metadata refresh`.

### 39. Token accounting accuracy

- **Gormes source**: `kernel.RenderFrame.Telemetry.TokensInTotal +
  TokensOutTotal` is read in `formatGatewayStatus` from the *last
  usage frame snapshot*. The snapshot only reflects the most recent
  turn, not session totals.
- **Status**: `partial`. Per-turn data shown; session-aggregate not.
- **Lane**: C.
- **Progress row**: `Gateway session token accounting parity` (planned).

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
- **Gormes source**: `/new` is wired (EventReset). `/reset` alias is
  not present in the registry — Hermes treats `reset` as alias for
  `new`; Gormes does not. `/retry` and `/undo` are recognized as
  unavailable. The 49-file CLI tree port umbrella covers their
  handlers.
- **Status**: `partial`.
- **Lane**: B/H.
- **Progress row**: `49-file CLI tree port` (planned umbrella) +
  `Gateway slash registry parity sweep` (planner refinement).

### 44-48. Memory/Goncho areas

These are tracked under existing planned rows:
`Hermes memory tool over Goncho/local durable store`,
`Goncho memory provider lifecycle adapter`,
`Long session management`, `Context compression`,
`Manual compression feedback + context references`,
`Kernel compression-boundary callback binding`. **Status: partial.**
**Lane: E.**

### 49. GONCHO branding

- **Gormes source**: `internal/goncho/` ships; default workspace and
  observer peer are both `"gormes"`. There's no Kancho or Honcho-rename
  residue in the runtime.
- **Status**: `parity`.

### 50-60. Provider / auth / runtime

These map to existing planned/complete rows:
- 50 Provider registry parity → `Bedrock provider runtime binding`,
  `Gemini Cloud Code request/stream mapper`,
  `OpenRouter compatible-provider routing`,
  `Google Code Assist project/quota resolver`,
  `Codex` (planned) and other complete provider rows.
- 51 Auth status → `cmd/gormes/auth_status_command_test.go` (complete).
- 52 Auth add/list/remove/logout → `cmd/gormes/auth_command_test.go`
  (complete).
- 53 Codex device-code/OAuth → `Hermes auth OAuth provider adapters`
  (planned).
- 54 Credential pool → `Gormes auth bare interactive credential-pool readout`
  (planned).
- 55 Provider request shape → kernel.go uses `hermes.ChatRequest`. parity.
- 56 Provider stream handling → kernel.go OpenStream + retry. parity.
- 57 Retry behavior → `kernel.NewRetryBudget` + `RetryStatus`. parity.
- 58 Health checks → `RuntimeStatusStore.ReadValidatedRuntimeStatusSnapshot`
  + procRuntimeProcessTable. parity.
- 59 Rate-limit evidence → existing planned provider account-usage rows.
- 60 Redacted diagnostics → `render_test`, `internal/audit/`. parity.

### 61-67. CLI / config

- 61 CLI command tree → `49-file CLI tree port`,
  `Hermes CLI command-tree parity manifest` (complete inventory).
- 62 CLI help text → `gatewayHelpText`. parity.
- 63 Active-turn CLI policy → `CLI command registry parity + active-turn busy policy` (complete).
- 64 Provider/config resolution → `Config, profile, auth, and setup command surfaces` (planned).
- 65 Config path discovery → `config.GormesHome()` + `config.ConfigPath()`. parity.
- 66 Config show/check/edit/migrate → `Gormes config edit/check/native schema-migrate closeout` (planned).
- 67 Diagnostics → `Diagnostics, backup, logs, and status CLI` (planned).

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

All depend on the planned `go-browser-harness Chromedp action backend`
row, which itself depends on the future Go browser harness binary
repository. Today the `BrowserHarnessTool::Execute` path shells out to
a configured `Command`; if the binary doesn't exist, the tool returns
a deterministic `ErrBrowserHarnessUnavailable` (verified by
`internal/tools/browser_harness_tools_test.go`).

**Status: partial across the board.** **Lane: G.**

### 76. Artifact budgets

- **Gormes source**: `tools.ToolResultBudgetConfig` + tests. parity.

### 77. Private URL / SSRF safety

- **Gormes source**: `internal/tools/browser_ssrf_guard.go` + tests.
  parity.

### 78. Browser tool result formatting compatibility with Hermes

- **Hermes source**: `gateway/run.py::_render_browser_tool_result` +
  channel-specific renderers.
- **Gormes source**: `Browser artifact and console render contract`
  (planned).
- **Status**: `partial`.

### 79. Browser channel rendering for Telegram

- **Gormes source**: `Telegram browser artifact rendering` (planned).
- **Status**: `partial`.

### 80. Go browser harness integration lane (placeholder for future repo)

- **Future repo**: not present in the workspace today. Planned as a
  sibling Go repo that produces a `go-browser-harness` binary
  consuming the JSON action contract at
  `internal/tools/browser_harness_tools.go::BrowserHarnessActionRequest`.
  The Gormes side is already the consumer; the harness repo is the
  producer.
- **Status**: `missing` — repo not present.
- **Lane**: G.
- **Progress row**: new umbrella —
  `Go browser harness binary repo + integration lane (placeholder)`.

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
