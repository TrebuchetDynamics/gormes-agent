---
title: "Hermes Behavior Atom Inventory"
date: 2026-05-26
description: "Evidence-first classification of every Hermes behavior atom against Gormes source. Trusts Hermes source + Gormes code/tests. Does not reference progress.json."
---

# Hermes Behavior Atom Inventory

**Purpose:** Source-backed inventory of every observable Hermes behavior atom
and its Gormes status. This is audit evidence only — not a backlog, not a
planning document.

**Method:** Every atom has an upstream Hermes file+line ref, a Gormes
file+line ref or explicit `missing`, and a classification.

**Classification vocabulary:**

| Term | Meaning |
|---|---|
| `covered` | Gormes preserves the observable behavior with source/test evidence. |
| `partial` | Gormes has some coverage but a named gap exists. |
| `missing` | No useful Gormes code or test exists. |
| `owned` | Intentional Gormes divergence with rationale and test. |
| `unknown` | Behavior exists in Hermes but has not been mapped. |

## How to read

- `HERMES` columns: file, line, function or class
- `GORMES` columns: file, line, function or test
- Every claim is source-backed. Read the refs before classifying.

---

## 1. Agent Runtime

### 1.1 Normal agent turn

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Normal turn loop: submit → provider stream → tool continuation → final | `run_agent.py` | `internal/kernel/kernel.go` `internal/hermes/client.go` | partial | Kernel loop exists; Hermes has richer context/prefill assembly. |
| Tool continuation multi-round | `run_agent.py` `_process_tool_call` | `internal/kernel/kernel.go` `handleToolCall` | covered | Loop drives multi-round tool calls. |
| Default 90-turn iteration budget | `run_agent.py` `max_iterations=90` | `internal/kernel/kernel.go` | covered | Default 90, toolless summary on exhaustion. |
| Cancel active turn | `run_agent.py` `cancel()` | `internal/kernel/kernel.go` `cancelCmd` | covered | Context cancellation. |
| Interrupt and replace draft | `run_agent.py` `interrupt` | `internal/kernel/kernel.go` + `internal/tui/update.go` `HermesActionInterrupt` | covered | TUI interrupt path tested. |
| Prefill messages injection | `cli.py` `_load_prefill_messages` | → `missing` | missing | Hermes loads prefill messages from env/file. |

### 1.2 Trajectory

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Trajectory compressor | `trajectory_compressor.py` | → `missing` | missing | Not ported. |

### 1.3 Context

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Context engine status | `agent/context_engine.py` | `internal/hermes/context_engine.go` | covered | Context status with token pressure. |
| Context compression | `agent/context_compressor.py` | → `missing` | missing | Not ported. |
| Manual compression feedback | `agent/manual_compression_feedback.py` | → `missing` | missing | Not ported. |
| Token budget | `agent/context_engine.py` | `internal/kernel/` | covered | Token budget tracking. |
| Protected head/tail | `agent/context_engine.py` | → `missing` | missing | Not ported. |
| Multimodal length | `agent/context_engine.py` | → `missing` | missing | Not ported. |
| Image charge | `agent/context_engine.py` | → `missing` | missing | Not ported. |
| Tool-result pruning | `agent/context_engine.py` | → `missing` | missing | Not ported. |
| Summary lineage | `agent/context_compressor.py` | → `missing` | missing | Not ported. |

### 1.4 Prompt builder

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| SOUL.md identity prompt | `agent/prompt_builder.py` | `internal/hermes/context_files.go` | covered | Context files scanned and injected. |
| AGENTS.md / project context | `agent/prompt_builder.py` | `internal/hermes/context_files.go` | covered | File discovery and injection. |
| USER.md / MEMORY.md durable context | `tools/memory_tool.py` | `internal/hermes/durable_user_context.go` | covered | Durable context built. |
| Skill guidance injection | `skill_preprocessing.py` | `internal/kernel/kernel.go` `SkillsPrompt` | partial | Hermes ordering differs. |
| Timestamp/model/provider metadata | `run_agent.py` `:3770-3779` | `internal/hermes/turn_metadata.go` | covered | Block assembly exists. |
| Platform/session context | `gateway/run.py` `BuildSessionContextPrompt` | `internal/gateway/` | covered | Session context built. |
| Developer role swap (GPT-5/Codex) | upstream provider tests | `internal/hermes/model_guidance.go` | partial | Helper exists; no API-boundary test. |
| Tool-use enforcement guidance | `agent/system_prompt.py` `build_system_prompt_parts` | `internal/gateway/live_turn_prompt.go` `buildToolUseEnforcementBlock` | covered | Wired into assembleLiveTurnPrompt; injects when model substring matches ToolUseEnforcementModels (gpt, codex, gemini, gemma, grok, glm). |
| Memory guidance | `agent/prompt_builder.py` `MEMORY_GUIDANCE` | `internal/hermes/guidance_constants.go` | covered | Byte-equivalent constant ported. |
| Skills guidance constant | `agent/prompt_builder.py` `SKILLS_GUIDANCE` | `internal/hermes/guidance_constants.go` | covered | Byte-equivalent constant ported. |

### 1.5 Redaction

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Audit log redaction | `agent/redact.py` | `internal/audit/` | covered | Redaction before audit write. |
| Prompt-visible redaction | `agent/redact.py` | `internal/redaction/` | covered | Path/key/secret redaction. |

---

## 2. Provider, Models, And Credentials

### 2.1 Provider transport

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Chat Completions transport | `agent/transports/chat_completions.py` | `internal/hermes/http_client.go` | covered | Transport request building and fixture replay. |
| Anthropic Messages transport | `agent/anthropic_adapter.py` | `internal/hermes/` | covered | Adapter shipped. |
| Bedrock Converse transport | `agent/bedrock_adapter.py` | `internal/hermes/` | partial | Stream events; SigV4 pending. |
| Codex Responses transport | `agent/codex_responses_adapter.py` | `internal/hermes/` | covered | Responses conversion shipped. |
| Gemini transport | `agent/gemini_native_adapter.py` | → `missing` | missing | Not ported. |
| Google Code Assist | `agent/gemini_cloudcode_adapter.py` | → `missing` | missing | Not ported. |
| OpenRouter | `tools/openrouter_client.py` | `internal/hermes/` | partial | Uses OpenAI-compatible path; attribution headers not proven. |

### 2.2 Provider registry

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Provider ID and alias manifest | `hermes_cli/providers.py` `HERMES_OVERLAYS` | `internal/hermes/provider_registry_manifest.go` | covered | Manifest with all Hermes provider IDs and aliases. |
| Model metadata and pricing | `agent/model_metadata.py` | → `missing` | missing | Not ported. |

### 2.3 Auth

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Credential pool | `agent/credential_pool.py` | `internal/config/` + `internal/cli/` | partial | Gormes has credential command surface; pool semantics differ. |
| OAuth device code | `hermes_cli/auth.py` `_login_openai_codex` | `cmd/gormes/auth.go` | partial | Codex OAuth path exists but paused/Hermes drift unclear. |
| Token vault | `agent/credential_sources.py` | → `missing` | missing | Not ported. |
| Auth commands (add/list/remove/reset/status/logout/spotify) | `hermes_cli/auth_commands.py` | `cmd/gormes/auth.go` | partial | Most commands exist; Spotify and top-level logout planned. |
| Secret ref validation | `hermes_cli/config.py` | `internal/provider/profile_provider_config.go` | covered | SecretRef env resolution and missing-ref evidence. |

### 2.4 Retry and rate limits

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Retry budget | `agent/retry_utils.py` | `internal/kernel/` `NewRetryBudget` | covered | Retry budget with backoff. |
| Rate limit tracker | `agent/rate_limit_tracker.py` | → `missing` | missing | Not ported. |
| Prompt cache | `agent/prompt_caching.py` | → `missing` | missing | Not ported. |
| Account usage reporting | `agent/account_usage.py` | `internal/hermes/` | partial | Provider account usage read model exists; renderer for Codex/Anthropic/OpenRouter. |

### 2.5 Error classification

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Error classifier | `agent/error_classifier.py` | `internal/hermes/` | partial | Basic error mapping; Hermes has richer provider-specific classes. |

---

## 3. CLI Command Tree

### 3.1 Parser structure

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Top-level parser | `hermes_cli/_parser.py` `hermes_cli/main.py` | `cmd/gormes/main.go` | partial | Cobra tree covers ~40 commands; Hermes covers ~50+ subcommands plus plugin dynamic commands. |
| Global flags | `hermes_cli/_parser.py` | `cmd/gormes/main.go` | partial | Most Hermes global flags covered; `--offline` is Gormes-owned. |
| `--model` / `-m` | `hermes_cli/_parser.py` | `cmd/gormes/main.go` | covered | Model override. |
| `--provider` / `-p` | `hermes_cli/_parser.py` | `cmd/gormes/main.go` | covered | Provider override. |
| `--endpoint` | `hermes_cli/_parser.py` | `cmd/gormes/main.go` | covered | Endpoint override. |
| `--api-key` | `hermes_cli/_parser.py` | `cmd/gormes/main.go` | covered | API key override. |
| `--oneshot` / `-z` | `hermes_cli/_parser.py` | → removed | owned | Gormes uses `-q` / `chat -q` for oneshot. |
| `--no-prompt-templates` | N/A (Gormes-owned) | `cmd/gormes/main.go` | owned | Gormes-owned flag. |
| `--offline` | N/A (Gormes-owned) | `cmd/gormes/main.go` | owned | Gormes-owned flag. |

### 3.2 Subcommand groups

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| `chat` subcommand | `cli.py` | `cmd/gormes/chat.go` | covered | TUI entry point. |
| `telegram` subcommand | `hermes_cli/main.py` `telegram_parser` | `cmd/gormes/telegram.go` | covered | Standalone Telegram bot. |
| `gateway` subcommand | `hermes_cli/gateway.py` | `cmd/gormes/gateway.go` | covered | Gateway lifecycle (run/stop/status). |
| `setup` subcommand | `hermes_cli/setup.py` | `cmd/gormes/setup*.go` | covered | Sectioned wizard shipped. |
| `auth` subcommand | `hermes_cli/auth_commands.py` | `cmd/gormes/auth.go` | partial | See 2.3. |
| `config` subcommand | `hermes_cli/config.py` | `cmd/gormes/config.go` | partial | Config show/check exists; edit/migrate partial. |
| `doctor` subcommand | `hermes_cli/doctor.py` | `cmd/gormes/doctor.go` | covered | Doctor with offline, JSON output. |
| `model` subcommand | `hermes_cli/main.py` `model_parser` | `cmd/gormes/model.go` | covered | Interactive model picker. |
| `profile` subcommand | `hermes_cli/profiles.py` | `cmd/gormes/profile*.go` | covered | Profile management. |
| `skills` subcommand | `hermes_cli/main.py` `skills_parser` | `cmd/gormes/skills.go` | covered | Skill install/list/inspect. |
| `mcp` subcommand | `hermes_cli/main.py` `mcp_parser` | `cmd/gormes/mcp.go` | covered | MCP login. |
| `kanban` subcommand | `hermes_cli/kanban.py` | `cmd/gormes/kanban.go` | covered | Task board. |
| `memory` subcommand | `plugins/memory/` | `cmd/gormes/goncho*.go` | covered | Memory status/doctor. |
| `browser` subcommand | `hermes_cli/browser_connect.py` | `cmd/gormes/browser*.go` | covered | Browser connect. |
| `uninstall` subcommand | `hermes_cli/uninstall.py` | → `missing` | missing | Not ported. |
| `migrate hermes` | `N/A` | `cmd/gormes/migrate.go` | covered | Hermes config/session migration. |
| `migrate openclaw` | `hermes_cli/claw.py` | `internal/migrate/openclaw/` | covered | OpenClaw migration shipped. |

### 3.3 Slash commands (gateway + TUI)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| `/help` | `hermes_cli/commands.py` | `internal/tui/slash_help.go` + `internal/gateway/commands.go` | covered | Both TUI and gateway. |
| `/new` / `/reset` | `hermes_cli/commands.py` | `internal/tui/slash_new.go` + `internal/gateway/` | covered | Session reset. |
| `/stop` | `hermes_cli/commands.py` | `internal/tui/slash_stop.go` + `internal/gateway/` | covered | Cancel active turn. |
| `/status` | `gateway/run.py` `_handle_status_command` | `internal/gateway/status_command.go` | partial | Field set same; formatting differs (no bold, no `⚡` on agent running). |
| `/title` | `gateway/run.py` `_handle_title_command` | `internal/gateway/title_command.go` + `title_command_test.go` | covered | Set/show session title with sanitization and persistence. |
| `/model` | `hermes_cli/main.py` `model_parser` | `internal/tui/slash_model.go` + `internal/gateway/model_picker.go` | covered | Interactive model picker. |
| `/skin` | `hermes_cli/commands.py` | `internal/tui/slash_skin.go` | covered | Skin switching. |
| `/compact` | `hermes_cli/commands.py` | `internal/tui/slash_compact.go` | covered | Compact transcript toggle. |
| `/details` | `hermes_cli/commands.py` | `internal/tui/slash_details.go` | covered | Detail section visibility. |
| `/history` | `hermes_cli/commands.py` | `internal/tui/slash_history.go` | covered | Transcript history page. |
| `/save` | `hermes_cli/commands.py` | `internal/tui/slash_save.go` | covered | Save conversation. |
| `/branch` | `hermes_cli/commands.py` | `internal/tui/slash_branch.go` | covered | Fork session. |
| `/copy` | `hermes_cli/commands.py` | `internal/tui/slash_copy.go` | covered | Copy to clipboard. |
| `/browser` | `hermes_cli/commands.py` | `internal/tui/slash_browser.go` | covered | Browser connect/status. |
| `/mouse` | `hermes_cli/commands.py` | `internal/tui/slash_mouse.go` | covered | Mouse tracking toggle. |
| `/indicator` | `hermes_cli/commands.py` | `internal/tui/slash_indicator.go` | covered | Busy indicator style. |
| `/busy` | `hermes_cli/commands.py` | `internal/tui/slash_busy.go` + `internal/gateway/` | covered | Busy input mode. |
| `/kanban` | `hermes_cli/kanban.py` | `internal/tui/slash_kanban.go` | covered | Task board. |
| `/logs` | `hermes_cli/logs.py` | `internal/tui/slash_logs.go` | covered | Gateway log tail. |
| `/session` | `hermes_cli/main.py` | `internal/tui/slash_sessions.go` | covered | Session browser. |
| `/reasoning` | `gateway/run.py` `_handle_reasoning_command` | `internal/gateway/reasoning_command.go` + `manager.go` | covered | Reasoning effort management with --global support. |
| `/voice` | `hermes_cli/voice.py` | → advertised unavailable | missing | Recognized, no handler. |
| `/tools` | `hermes_cli/commands.py` | → advertised unavailable | missing | Recognized, no handler. |
| `/skills` | `hermes_cli/commands.py` | `internal/tui/slash_skills.go` | covered | Skill install/inspect. |
| `/goal` | `hermes_cli/goals.py` | `internal/tui/slash_goal.go` | covered | Standing goal. |
| `/profile` | `hermes_cli/profiles.py` | `internal/tui/slash_profile.go` | covered | Profile info. |
| `/usage` | `hermes_cli/commands.py` | `internal/tui/slash_usage.go` + `internal/gateway/usage_command.go` | covered | Usage page/command. |

### 3.4 Active-turn slash policy

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Bypass commands during active turn | `gateway/run.py` `:2950-3225` | `internal/gateway/commands.go` + `internal/gateway/active_turn_command_bypass_test.go` | covered | Help/stop/status bypass. |
| Queue command during active turn | `gateway/run.py` | `internal/kernel/` + `internal/tui/` | covered | Queue/steer/interrupt modes. |
| Unavailable command evidence | `hermes_cli/commands.py` | `internal/cli/command_registry.go` | covered | `ActiveTurnPolicyUnavailable` with evidence. |
| Unknown command guidance | `gateway/run.py` `:3435-3452` | `internal/gateway/manager.go` `:704-727` | covered | Unknown slash intercepted. |

---

## 4. TUI / Terminal UX

### 4.1 Full-screen chat chrome

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Bottom-pinned layout | `ui-tui/src/components/appLayout.tsx` | `internal/tui/hermes_chrome.go` + `internal/tui/view.go` | covered | Transcript → hint → status → prompt. |
| Alt-screen rendering | `ui-tui/src/components/appLayout.tsx` | `internal/tui/` tea.AltScreen() | covered | Alternate screen used. |
| Role glyphs (❯ user, ┊ assistant) | `ui-tui/src/components/messageLine.tsx` | `internal/tui/view.go` `transcriptRowWithSkin` | covered | Skin-driven glyphs. |
| Prompt symbol | `ui-tui/src/components/composer.tsx` | `internal/tui/view.go` `renderComposerPrompt` | covered | Skin prompt symbol on input line. |
| Status bar | `ui-tui/src/components/appChrome.tsx` | `internal/tui/status_bar*.go` | covered | Status bar with mode/context/session. |
| Queued messages widget | `ui-tui/src/components/queuedMessages.tsx` | `internal/tui/queued_messages*.go` | covered | Queued message display. |
| Slash completion menu | `ui-tui/src/app/slash/registry.ts` `prompt_toolkit` | `internal/tui/slash_completion.go` | covered | Interactive completion with Up/Down/Tab/Enter/Escape. |
| Input history (Up/Down) | `ui-tui/src/hooks/useInputHistory.ts` | `internal/tui/history.go` | covered | In-memory history with draft restore. |
| Draft restore on Up/Down | `ui-tui/src/hooks/useInputHistory.ts` | `internal/tui/history.go` `PrevFrom` | covered | Partial draft preserved on Down-past-newest. |
| Consecutive duplicate dedupe | `lib/history.ts` | `internal/tui/history.go` `Append` | covered | Consecutive duplicates ignored. |
| Spinner/hint row | `ui-tui/src/components/thinking.tsx` | `internal/tui/thinking*.go` | covered | Conditional hint row when active. |
| Modal panels (approval/clarify/secret) | `ui-tui/src/components/` | `internal/tui/hermes_panels.go` | covered | Approval/clarify/secret panels. |
| Welcome/intro banner | `ui-tui/src/components/` | `internal/tui/banner.go` | covered | Welcome panel with tips. |
| Mouse tracking | `hermes_cli/pt_input_extras.py` | `internal/tui/mouse_tracking_test.go` | covered | Mouse toggle. |

### 4.2 Tool progress

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Tool progress rendering | `agent/display.py` `get_tool_emoji` | `internal/tooltrace/` | covered | Shared renderer with emoji icons. |
| `new/all/off` modes | `gateway/display_config.py` | `internal/config/` + `internal/tooltrace/` | covered | Tool progress display modes. |
| Duplicate collapse | `agent/display.py` | `internal/tooltrace/` | partial | Collapse for consecutive same-symbol tool traces. |
| Tool preview truncation | `agent/display.py` `build_tool_preview` | `internal/tooltrace/` | covered | Truncation of long tool args. |
| `(×N)` collapse | `agent/display.py` | `internal/tooltrace/` | covered | Identical consecutive tool calls collapsed. |
| `todo merge=true` wording | `agent/display.py` | `internal/tooltrace/` | covered | Special todo merge display. |
| Unknown-tool degradation | `agent/display.py` | `internal/tooltrace/` | covered | Unknown tools display as generic ⚡. |
| Tool result error display | `gateway/run.py` `:14716` | `internal/gateway/render.go` | covered | Error tool results rendered distinctly. |
| Tool progress override per-platform | `gateway/run.py` `display.platforms.<name>.tool_progress` | `internal/config/` | partial | Config exists; per-platform override not proven. |
| Tool progress env var override | `gateway/run.py` `HERMES_TOOL_PROGRESS` | → `missing` | missing | Hermes env var for tool progress mode. |

### 4.3 Composer behavior

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Enter submits | `ui-tui/src/app/useInputHandlers.ts` | `internal/tui/update.go` | covered | Enter dispatches. |
| Alt+Enter inserts newline | `ui-tui/src/app/useInputHandlers.ts` | `internal/tui/update.go` | covered | Alt+Enter handled. |
| Ctrl+C cancels/force-quits | `ui-tui/src/app/useInputHandlers.ts` | `internal/tui/update.go` | covered | Modal cancel → turn cancel → force quit. |
| Ctrl+D deletes char / exits | `ui-tui/src/app/useInputHandlers.ts` | `internal/tui/update.go` | covered | Delete char when draft non-empty. |
| Ctrl+L repaints | `ui-tui/src/app/useInputHandlers.ts` | `internal/tui/update.go` | covered | Force redraw. |
| Paste/image handling | `ui-tui/src/app/useInputHandlers.ts` | `internal/tui/composer_ingress.go` | covered | Clipboard paste and image attachment. |
| Voice recording key | `ui-tui/src/app/useInputHandlers.ts` | `internal/tui/` | partial | Voice key configurable, TTS not wired. |

---

## 5. Gateway And Channels

### 5.1 Gateway runtime

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Gateway lifecycle (run/stop/restart/reload/status) | `gateway/run.py` | `internal/gateway/` + `cmd/gormes/gateway.go` | covered | Manager lifecycle shipped. |
| Platform registry | `gateway/config.py` `Platform` | `internal/gateway/` | partial | Platform registry manifest; not every Hermes platform has Gormes adapter. |
| Channel adapter lifecycle | `gateway/platforms/*.py` | `internal/channels/*` | partial | Telegram, Discord, Slack, WhatsApp, Signal, Navivox, more. |
| Stream consumer | `gateway/stream_consumer.py` | `internal/gateway/render.go` | covered | Streaming event rendering. |
| Delivery | `gateway/delivery.py` | `internal/gateway/` | covered | Outbound message delivery. |
| Session mapping | `gateway/session.py` | `internal/gateway/` + `internal/session/` | covered | Session ID mapping. |
| Active-turn policy | `gateway/run.py` `:2950-3225` | `internal/gateway/manager.go` `:704-727` | covered | Channel-neutral policy. |
| Restart/PID | `gateway/restart.py` | `internal/gateway/` | covered | PID file, restart markers. |
| Platform pairing | `gateway/pairing.py` | `internal/gateway/` | covered | Approval pairing. |
| Hook loading | `gateway/hooks.py` | `internal/gateway/` | covered | Boot hooks. |

### 5.2 Channel adapters

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Telegram | `gateway/platforms/telegram.py` | `internal/channels/telegram/` | covered | Bot adapter with MarkdownV2, reply quoting, menus. |
| Discord | `gateway/platforms/discord.py` | `internal/channels/discord/` | covered | Discord adapter. |
| Slack | `gateway/platforms/slack.py` | `internal/channels/slack/` | covered | Slack Socket Mode adapter. |
| WhatsApp | `gateway/platforms/whatsapp.py` | `internal/channels/whatsapp/` | covered | WhatsApp bridge/adapter. |
| Signal | `gateway/platforms/signal.py` | `internal/channels/signal/` | covered | Signal adapter. |
| Matrix | `gateway/platforms/matrix.py` | `internal/channels/matrix/` | covered | Matrix adapter. |
| Mattermost | no Gormes channel | → `missing` | missing | Not ported. |
| Google Chat | no Gormes channel | → `missing` | missing | Not ported. |
| BlueBubbles | no Gormes channel | `internal/channels/bluebubbles/` | covered | BlueBubbles iMessage bridge. |
| Feishu | `gateway/platforms/feishu.py` | `internal/channels/feishu/` | covered | Feishu adapter. |
| DingTalk | `gateway/platforms/dingtalk.py` | `internal/channels/dingtalk/` | covered | DingTalk adapter. |
| QQ Bot | `gateway/platforms/qqbot.py` | `internal/channels/qqbot/` | covered | QQ Bot adapter. |
| WeCom | no Gormes channel | `internal/channels/wecom/` | covered | WeCom adapter. |
| SimpleX | `gateway/platforms/simplex.py` | `internal/channels/simplex/` | covered | SimpleX adapter. |

### 5.3 Telegram-specific

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| MarkdownV2 parse mode | `gateway/platforms/telegram.py` | `internal/channels/telegram/bot.go` | covered | Bold/italic/code/headers/spoilers. |
| Reply quoting | `gateway/platforms/telegram.py` `ReplyToMessageID` | `internal/channels/telegram/bot.go` | partial | Outbound reply quoting exists; reply modes not parity. |
| Placeholder lifecycle | `gateway/platforms/base.py` `:1718-1724` | `internal/gateway/coalesce.go` | partial | Editable ⏳ placeholder; typing action not proven. |
| setMyCommands | `gateway/platforms/telegram.py` `:822-837` | `internal/channels/telegram/bot.go` | partial | Static menu on startup; no dynamic refresh. |
| Silent notification defaults | `gateway/platforms/telegram.py` | `internal/channels/telegram/thread_send_test.go` | covered | Placeholder sends silent; finals notify. |

---

## 6. Tools

### 6.1 Core file tools

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| `read_file` | `tools/file_operations.py` | `internal/tools/` | covered | Read file with workspace guard. |
| `write_file` | `tools/file_operations.py` | `internal/tools/` | covered | Write file with workspace guard. |
| `search_files` | `tools/file_operations.py` | `internal/tools/` | covered | File search. |
| `patch` | `tools/file_operations.py` | `internal/tools/` | covered | Diff patch application. |
| `terminal` | `tools/file_operations.py` | `internal/tools/` | covered | Guarded command execution. |

### 6.2 Memory/search tools

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| `memory` tool (Hermes-compatible) | `tools/memory_tool.py` | → `missing` | missing | Goncho tools exist but no Hermes-compatible `memory` tool in default registry. |
| `session_search` tool | `tools/session_search_tool.py` | `internal/tools/sessionsearch/` | covered | Session search. |

### 6.3 Browser tools

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Browser action contract | `tools/browser_tool.py` | `internal/tools/browser_contract.go` | covered | Action/result schema. |
| Browser snapshots | `tools/browser_tool.py` | `internal/tools/browser_harness_tools.go` | partial | Backend planned. |
| Screenshot artifacts | `tools/browser_tool.py` | `internal/tools/browser_contract.go` | covered | Envelope fields. |
| SSRF guard | `tools/browser_tool.py` | `internal/tools/browser_ssrf_guard.go` | covered | Private URL guard. |

### 6.4 TTS / voice

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| TTS tool | `tools/tts_tool.py` | → `missing` | missing | Not ported. |
| Transcription tool | `tools/transcription_tools.py` | `internal/tools/whisper/` | covered | WASI Whisper STT. |

### 6.5 Kanban

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Kanban CRUD | `tools/kanban_tool.py` | `internal/tools/kanban/` | covered | Task board create/list/update. |
| Kanban dispatch | `tools/kanban_tool.py` | `internal/tools/kanban/` | covered | Dispatch to subagent. |

---

## 7. Cron And Scheduled Jobs

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Cron scheduler | `cron/scheduler.py` | `internal/cron/` | covered | Schedule parser and execution. |
| Cron job definitions | `cron/jobs.py` | `internal/cron/` | covered | Job store with CRUD. |
| Cron tool | `tools/cronjob_tools.py` | `internal/tools/` | covered | Cron management tool. |
| Schedule parser | `cron/jobs.py` `parse_schedule` | `internal/cron/` | covered | Natural language schedule parsing. |
| Compute next run | `cron/jobs.py` `compute_next_run` | `internal/cron/` | covered | Next execution time calculation. |
| Grace seconds | `cron/jobs.py` `_compute_grace_seconds` | → `missing` | missing | Grace window after missed schedule. |
| Delivery target resolution | `cron/scheduler.py` `_resolve_delivery_targets` | → `missing` | missing | Multi-platform delivery routing. |
| Multi-target delivery | `cron/scheduler.py` `_deliver_result` | → `missing` | missing | Send results to multiple channels. |
| Script execution | `cron/scheduler.py` `_run_job_script` | `internal/cron/` | covered | Run shell scripts as job actions. |
| Context_from chaining | `cron/jobs.py` | → `missing` | missing | Chain prompts from previous job output. |
| Resource release | `cron/jobs.py` | → `missing` | missing | Cleanup after job completion. |
| Job lock files | `cron/scheduler.py` `_get_lock_paths` | → `missing` | missing | PID-based job locking. |
| Cron prompt guard | `cron/scheduler.py` `CronPromptInjectionBlocked` | → `missing` | missing | Prevent cron prompt injection. |
| Recovery from missed schedule | `cron/jobs.py` `_recoverable_oneshot_run_at` | → `missing` | missing | Recover missed one-shot jobs. |

---

## 8. Skills

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Skill metadata (SKILL.md) | `skills/` | `internal/skills/` + `docs/development-skills/` | covered | Portable SKILL.md metadata. |
| Skill install/list/inspect | `hermes_cli/skills_hub.py` | `cmd/gormes/skills.go` | covered | Skill CLI commands. |
| Skill slash commands | `agent/skill_commands.py` | `internal/tui/slash_skills.go` | covered | Dynamic slash command registration. |
| Skill prompt snapshot | `skill_preprocessing.py` | `internal/kernel/kernel.go` | partial | Skill blocks injected; ordering differs. |
| Skill sync | `tools/skills_sync.py` | → `missing` | missing | Not ported. |

---

## 9. MCP / Plugins

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| MCP server | `mcp_serve.py` | `internal/mcpserver/` | covered | MCP stdio/HTTP server. |
| MCP tool | `tools/mcp_tool.py` | `internal/tools/` | partial | Tool registration; sessions not proven. |
| Plugin registry | `plugins/registry.py` | `internal/plugins/` | covered | Plugin manifest loader. |
| ACP adapter | `acp_adapter/` | → `missing` | missing | Not ported. |

---

## 10. APIServer (OpenAI-compatible)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| `/health` | `api_server.py` `:3400` | `internal/apiserver/` | covered | Health check. |
| `/health/detailed` | `api_server.py` `:3401` | → `missing` | missing | Detailed health (not ported). |
| `/v1/health` | `api_server.py` `:3402` | `internal/apiserver/` | covered | Versioned health. |
| `/v1/models` | `api_server.py` `:3403` | → `missing` | missing | Model listing. |
| `/v1/capabilities` | `api_server.py` `:3404` | → `missing` | missing | Server capabilities document. |
| `/v1/chat/completions` | `api_server.py` `:3405` | `internal/apiserver/` | covered | OpenAI-compatible streaming. |
| `/v1/responses` (POST) | `api_server.py` `:3406` | → `missing` | missing | New Responses API endpoint. |
| `/v1/responses/{id}` (GET) | `api_server.py` `:3407` | → `missing` | missing | Fetch stored response. |
| `/v1/responses/{id}` (DELETE) | `api_server.py` `:3408` | → `missing` | missing | Delete stored response. |
| `/api/jobs` (GET) | `api_server.py` `:3410` | → `missing` | missing | List scheduled jobs. |
| `/api/jobs` (POST) | `api_server.py` `:3411` | → `missing` | missing | Create scheduled job. |
| `/api/jobs/{id}` (GET) | `api_server.py` `:3412` | → `missing` | missing | Get job by ID. |
| `/api/jobs/{id}` (PATCH) | `api_server.py` `:3413` | → `missing` | missing | Update job. |
| `/api/jobs/{id}` (DELETE) | `api_server.py` `:3414` | → `missing` | missing | Delete job. |
| `/api/jobs/{id}/pause` | `api_server.py` `:3415` | → `missing` | missing | Pause job. |
| `/api/jobs/{id}/resume` | `api_server.py` `:3416` | → `missing` | missing | Resume job. |
| `/api/jobs/{id}/run` | `api_server.py` `:3417` | → `missing` | missing | Run job immediately. |
| `/v1/runs` (POST) | `api_server.py` `:3419` | → `missing` | missing | Create assistant run (threads API). |
| `/v1/runs/{id}` (GET) | `api_server.py` `:3420` | → `missing` | missing | Get run status. |
| `/v1/runs/{id}/events` (GET) | `api_server.py` `:3421` | → `missing` | missing | Stream run events (SSE). |
| `/v1/runs/{id}/approval` (POST) | `api_server.py` `:3422` | → `missing` | missing | Approve pending action. |
| `/v1/runs/{id}/stop` (POST) | `api_server.py` `:3423` | → `missing` | missing | Stop running run. |
| Dashboard | `hermes_cli/web_server.py` | → `missing` | missing | Dashboard not ported. |

---

## 11. Cron, Background, Learning Loop

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Learning-loop curator state machine | `agent/curator.py` | → `missing` | missing | Not ported. |
| Curator entity discovery | `agent/curator.py` | → `missing` | missing | Discover skills/tools for review. |
| Curator candidate extraction | `agent/curator.py` | → `missing` | missing | Extract candidates from turn output. |
| Curator review/promotion | `agent/curator.py` | → `missing` | missing | Queue review → promote to skill. |
| Background review fork | `run_agent.py` background review | → `missing` | missing | Fork background review agent. |
| Auxiliary curator model routing | `hermes_cli/config.py` auxiliary.curator | → `missing` | missing | Separate model for curator calls. |
| Curator CLI commands | `hermes_cli/curator.py` | → `missing` | missing | /curator status, run, pause, pin. |
| Curator skill state transitions | `tools/skill_manager_tool.py` | → `missing` | missing | Support-file write/patch/remove, absorbed_into, agent-created provenance. |

---

## 12. Release And Packaging

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Install script | `setup-hermes.sh` | `install.sh` + `install.ps1` | covered | Cross-platform installer. |
| OCI image | `Dockerfile` + `docker/entrypoint.sh` | `Makefile` | planned | Docker build exists but not published. |
| Homebrew formula | `packaging/homebrew/hermes-agent.rb` | → `missing` | missing | Not ported. |
| Nix build | `nix/` `flake.*` | → `missing` | missing | Not ported. |
| Version surface | `scripts/release.py` | `cmd/gormes/main.go` | covered | `--version` flag. |
| Release script | `scripts/release.py` | → `missing` | missing | Not ported. |
| Docker entrypoint | `docker/entrypoint.sh` | → `missing` | missing | Not ported. |
| Docker compose | `docker-compose.yml` | → `missing` | missing | Not ported. |

---

## 13. Honcho / Goncho Memory

| Atom | HERMES / HONCHO | GORMES | Status | Notes |
|---|---|---|---|---|
| Workspace identity | `src/models.py` `Collection` | `internal/goncho/` | covered | SQLite workspace with config defaults. |
| Peer identity | `src/routers/peers.py` | `internal/goncho/` | covered | Peer cards and representation scopes. |
| Session lifecycle | `src/routers/sessions.py` | `internal/goncho/` + `internal/session/` | covered | Local session crud. |
| Message CRUD | `src/routers/messages.py` | `internal/goncho/` + `internal/memory/` | covered | Workspace/session/peer sequence metadata. |
| File-backed messages | `src/crud/document.py` | → `missing` | missing | Not ported. |
| Conclusions / facts | `src/routers/conclusions.py` | → `missing` | missing | Not ported. |
| Representations by scope | `src/crud/representation.py` | → `missing` | missing | Not ported. |
| Search and filters | `src/utils/filter.py` `src/utils/search.py` | `internal/goncho/` | partial | FTS5 search exists; Honcho filter grammar not parsed. |
| Context retrieval | docs `get-context.mdx` | → `missing` | missing | Not ported. |
| Dialectic chat | `src/dialectic/chat.py` | → `missing` | missing | Not ported. |
| Streaming persistence | docs `streaming-response.mdx` | `internal/goncho/` | covered | Only final assistant message persisted. |
| Summaries | docs `summarizer.mdx` | → `missing` | missing | Not ported. |
| Dreaming scheduler | `src/dreamer/dream_scheduler.py` | → `missing` | missing | Not ported. |
| Webhook CRUD | `src/routers/webhooks.py` | → `missing` | missing | Not ported. |
| Webhook delivery | `src/webhooks/webhook_delivery.py` | → `missing` | missing | Not ported. |
| Queue status | docs `queue-status.mdx` | `internal/goncho/` | partial | Queue depth visible; derivation not proven. |
| Honcho SDK compatibility | `sdks/python/` `sdks/typescript/` | → `missing` | missing | No SDK e2e harness. |
| Honcho CLI compatibility | `honcho-cli/` | `cmd/gormes/goncho*.go` | partial | Goncho status/doctor; full CLI parity not proven. |

---

## 14. Agent Runtime — Deep (Titles, Context Refs, Curator)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Auto title generation | `agent/title_generator.py` | `internal/session/auto_title.go` | partial | Helper exists; gateway wiring not proven. |
| Session naming from user prompt | `run_agent.py` `auto_title` | → `missing` | missing | Not wired in production. |
| @ context reference parser | `agent/context_references.py` | `internal/hermes/` + `internal/contextrefs/` | covered | Stable parser shipped; file/folder/URL injection row-backed. |
| Subdirectory/project hints | `agent/subdirectory_hints.py` | → `missing` | missing | Not ported. |
| Background review fork | `run_agent.py` background review | → `missing` | missing | Not ported. |
| Curator state machine | `agent/curator.py` | → `missing` | missing | Not ported. |
| Curator CLI | `hermes_cli/curator.py` | → `missing` | missing | Not ported. |
| Memory prefetch/sync | `agent/memory_manager.py` | → `missing` | missing | Not ported. |
| Pre-compress hook | `agent/memory_manager.py` | → `missing` | missing | Not ported. |
| Ephemeral prefill messages | `cli.py` `_load_prefill_messages` | → `missing` | missing | Not ported. |
| Moonshot/Kimi schema sanitizer | `agent/moonshot_schema.py` | `internal/hermes/moonshot_schema.go` | covered | Tool-parameter sanitizer shipped. |

---

## 15. Provider — Deep (Bedrock, Gemini, OAuth)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Google Gemini native | `agent/gemini_native_adapter.py` | → `missing` | missing | Not ported. |
| Google Cloud Code adapter | `agent/gemini_cloudcode_adapter.py` | → `missing` | missing | Not ported. |
| Google Gemini OAuth | `agent/google_oauth.py` | → `missing` | missing | Not ported. |
| Google Code Assist | `agent/google_code_assist.py` | → `missing` | missing | Not ported. |
| Bedrock stream events | `agent/bedrock_adapter.py` | `internal/hermes/` | partial | Stream events partial; SigV4 pending. |
| Bedrock SigV4 credentials | `agent/bedrock_adapter.py` | → `missing` | missing | Not ported. |
| Bedrock stale-client eviction | `agent/bedrock_adapter.py` | → `missing` | missing | Not ported. |
| Codex OAuth / device-code | `hermes_cli/auth.py` `_login_openai_codex` | `cmd/gormes/auth.go` | partial | Codex auth path exists; paused pending upstream drift investigation. |
| Codex stale-token relogin | `agent/auxiliary_client.py` | → `missing` | missing | Not ported. |
| Codex model enumeration | `agent/models_dev.py` | → `missing` | missing | Not ported. |
| OpenRouter attribution headers | `tools/openrouter_client.py` | → `missing` | missing | Not ported. |
| Provider model metadata | `agent/model_metadata.py` | → `missing` | missing | Not ported. |
| Provider usage pricing | `agent/usage_pricing.py` | → `missing` | missing | Not ported. |
| Copilot ACP client | `agent/copilot_acp_client.py` | → `missing` | missing | Not ported. |
| Credential pool multi-source | `agent/credential_pool.py` | → `missing` | missing | Not ported. |
| Credential sources (env/dotenv/config) | `agent/credential_sources.py` | `internal/config/` | partial | Env and config loading exist; Hermes has richer fallback chain. |

---

## 16. Gateway — Deep (Hooks, Pairing, Restart, Webhooks)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Boot hooks (boot_md agent-spawning) | `gateway/builtin_hooks/boot_md.py` | → `missing` | missing | Not ported. |
| Hook loading infrastructure | `gateway/hooks.py` | `internal/gateway/` | partial | Gateway hooks exist; boot hooks not proven. |
| Platform pairing approval | `gateway/pairing.py` | `internal/gateway/` | covered | Pairing approval flow. |
| Gateway restart (exit code 75) | `gateway/restart.py` | `internal/gateway/` | covered | Restart marker and PID validation. |
| Gateway status JSON | `gateway/status.py` | `internal/gateway/status.go` + `cmd/gormes/gateway_status.go` | covered | Status CLI with JSON output. |
| Gateway config reload (SIGHUP) | `gateway/run.py` | `internal/gateway/` | covered | Config reload via SIGHUP. |
| Gateway webhook command | `hermes_cli/webhook.py` | → `missing` | missing | Not ported. |
| Gateway logs CLI | `hermes_cli/logs.py` | `internal/tui/slash_logs.go` + `cmd/gormes/` | covered | Log tail in TUI and CLI. |
| Gateway backup CLI | `hermes_cli/backup.py` | → `missing` | missing | Not ported. |
| Gateway failure/restart policy | `gateway/run.py` | `internal/gateway/` | covered | Restart on unexpected signal. |
| Multi-account channels | `gateway/run.py` | `internal/gateway/` | covered | Account-level registration. |
| Channel bootstrap sequence | `gateway/platforms/base.py` | `internal/channels/*` | partial | Individual channels; framework bootstrap not abstract. |

---

## 17. Tools — Deep (Delegate, Security, Operator, Sandbox)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Subagent delegate tool | `tools/delegate_tool.py` | `internal/subagent/` + `internal/tools/` | covered | Deterministic delegate runtime. |
| Approval tool | `tools/approval.py` | `internal/tools/approval/` | covered | Approval mode guards. |
| Path security tool | `tools/path_security.py` | `internal/tools/safety/` | covered | Workspace path guard. |
| URL safety tool | `tools/url_safety.py` | → `missing` | missing | Not ported. |
| Website policy tool | `tools/website_policy.py` | → `missing` | missing | Not ported. |
| OSV supply-chain check | `tools/osv_check.py` | → `missing` | missing | Not ported. |
| Todo tool | `tools/todo_tool.py` | `internal/tools/` | covered | Todo state management. |
| Clarify tool | `tools/clarify_tool.py` | `internal/tools/` | covered | Clarify prompts. |
| Send-message tool | `tools/send_message_tool.py` | → `missing` | missing | Not ported. |
| Debug helpers tool | `tools/debug_helpers.py` | → `missing` | missing | Not ported. |
| Interrupt tool | `tools/interrupt.py` | `internal/kernel/` | covered | Turn cancellation via context. |
| Code execution tool | `tools/code_execution_tool.py` | `internal/tools/` + `internal/cmdrunner/` | partial | Guarded local execution; process registry not proven. |
| Background process tool | `tools/code_execution_tool.py` background | `internal/tools/` | partial | Background mode not proven. |
| File operations checkpoint | `tools/checkpoint_manager.py` | → `missing` | missing | Not ported. |
| Image generation tool | `tools/image_generation_tool.py` | → `missing` | missing | Not ported. |
| Image routing by model | `agent/image_routing.py` | → `missing` | missing | Not ported. |
| Sandbox: Docker | `tools/environments/docker.py` | → `missing` | missing | Not ported. |
| Sandbox: Modal | `tools/environments/modal.py` | → `missing` | missing | Not ported. |
| Sandbox: SSH | `tools/environments/ssh.py` | → `missing` | missing | Not ported. |
| Sandbox: Singularity | `tools/environments/singularity.py` | → `missing` | missing | Not ported. |
| Sandbox: local | `tools/environments/local.py` | `internal/cmdrunner/` | partial | Guarded local execution. |
| Raw tool-call parser: DeepSeek | `environments/tool_call_parsers/deepseek_parser.py` | → `missing` | missing | Not ported. |
| Raw tool-call parser: Qwen | `environments/tool_call_parsers/qwen_parser.py` | → `missing` | missing | Not ported. |
| Raw tool-call parser: Mistral | `environments/tool_call_parsers/mistral_parser.py` | → `missing` | missing | Not ported. |
| Raw tool-call parser: GLM | `environments/tool_call_parsers/glm_parser.py` | → `missing` | missing | Not ported. |
| Raw tool-call parser: Hermes XML | `environments/tool_call_parsers/hermes_xml_parser.py` | → `missing` | missing | Not ported. |
| Raw tool-call parser: Llama | `environments/tool_call_parsers/llama_parser.py` | → `missing` | missing | Not ported. |

---

## 18. Toolsets, Tool Registry, Model Tools

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Tool registry | `tools/registry.py` | `internal/tools/` | covered | Descriptor-driven registry. |
| Toolsets | `toolsets.py` | `cmd/gormes/registry.go` | covered | Toolset enable/disable. |
| Model tools (model selectors) | `model_tools.py` | `internal/cli/` + `internal/hermes/` | partial | Model routing; Hermes model_tools abstractions not fully ported. |
| Toolset distributions | `toolset_distributions.py` | → `missing` | missing | Not ported. |

---

## 19. Config System

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Config YAML reading | `hermes_cli/config.py` `load_config` | `internal/config/` `Load` | covered | Hermes-compatible config.yaml as bridge. |
| Config env expansion | `hermes_cli/config.py` | `internal/config/` | covered | Env var expansion in config values. |
| Config profile resolution | `hermes_cli/config.py` | `internal/config/` + `internal/cli/profileseed` | covered | Profile loading and merging. |
| Config show command | `cli.py` | `cmd/gormes/config.go` | covered | `gormes config show`. |
| Config edit command | `cli.py` | `cmd/gormes/config.go` `newConfigEditCommand` + `config_closeout_test.go` | covered | Opens system editor for config file; creates file before opening; fallback editor chain EDITOR > VISUAL > common binaries. |
| Config check command | `cli.py` | `cmd/gormes/config.go` `newConfigCheckCommand` + `config_closeout_test.go` | covered | Validates config syntax, reports version, dotenv availability, missing provider fields; redacts secrets; future version fails. |
| Config migrate | `cli.py` | `internal/migrate/hermes/` | covered | Hermes → Gormes config migration. |
| Config env-path | `cli.py` | `cmd/gormes/config.go` | covered | `gormes config env-path`. |
| cli-config.yaml.example (51KB schema) | `cli-config.yaml.example` | → `missing` | missing | Not mirrored as canonical schema. |
| Secrets (.env) loading | `hermes_cli/env_loader.py` | `internal/config/` | covered | Dotenv loading with secret-ref validation. |
| Config validation | `hermes_cli/config.py` validation | `internal/config/` | partial | Schema validation not proven. |

---

## 20. Session Management

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Session ID generation (Hermes-style) | `gateway/session.py` | `internal/gateway/` | covered | Session ID with Hermes-style format. |
| Session title auto-generation | `agent/title_generator.py` | `internal/session/auto_title.go` | partial | Helper unwired in gateway. |
| Session title persistence | `gateway/session.py` | `internal/session/` | partial | Metadata persisted. |
| Created timestamp | `gateway/run.py` `:4672` | `internal/gateway/status_command.go` | covered | CreatedAt on fresh sessions. |
| Last activity timestamp | `gateway/run.py` `:4673` | `internal/gateway/status_command.go` | partial | UpdatedAt written; full refresh not proven. |
| Token accounting | `gateway/run.py` `:4674` | `internal/gateway/usage_command.go` | partial | Per-frame totals; session-wide not proven. |
| Session reset/new/retry/undo | `gateway/run.py` | `internal/gateway/` | partial | `/new` via EventReset; `/retry`, `/undo` missing. |
| Session resume | `gateway/run.py` | `internal/gateway/` | covered | Durable pause/resume. |
| Session context prompt (BuildSessionContextPrompt) | `gateway/run.py` | `internal/gateway/` | covered | Platform/session context block. |
| Compression boundary callbacks | `gateway/run.py` | → `missing` | missing | Not ported. |
| Auto-reset (idle/daily/suspended) | `gateway/run.py` | `internal/gateway/session_auto_reset_notify_test.go` | covered | Reason strings, notification policy. |
| Slash-confirm session-boundary cleanup | `gateway/run.py` | `internal/gateway/slash_confirm_test.go` | covered | Pending confirmations cleared on reset. |
| Session-boundary hooks | `gateway/run.py` | `internal/gateway/session_boundary_hooks_test.go` | covered | Finalize → reset hook ordering. |

---

## 21. Plugins

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Plugin discovery | `plugins/` | `internal/plugins/` | covered | Manifest and capability loader. |
| Plugin registry | `plugins/registry.py` | `internal/plugins/` | covered | Plugin registration. |
| Memory/Honcho plugin | `plugins/memory/` | `internal/goncho/` + `internal/plugins/` | covered | Goncho as memory plugin. |
| Disk cleanup plugin | `plugins/disk-cleanup/` | → `missing` | missing | Not ported. |
| Spotify plugin | `plugins/spotify/` | → `missing` | missing | Not ported. |
| Google Meet plugin | `plugins/google_meet/` | → `missing` | missing | Not ported. |
| Dynamic plugin CLI commands | `plugins/*/__init__.py` | → `missing` | missing | Not ported. |
| Plugin commands CLI | `hermes_cli/plugins_cmd.py` | `cmd/gormes/` | partial | Plugin commands not proven in CLI. |

---

## 22. Dashboard / Web Server

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Web dashboard | `web/` | → `missing` | missing | Not ported. |
| Web server (config/env/logs/OAuth) | `hermes_cli/web_server.py` | → `missing` | missing | Not ported. |
| Public website | `website/docs/` | `www.gormes.ai` + `webpages/docs/` | covered | Gormes public site and docs. |
| Plugin SPA contract | `web/` plugin assets | → `missing` | missing | Not ported. |

---

## 23. Batch / Mini-SWE / RL / Datagen

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Batch runner | `batch_runner.py` | → `missing` | missing | Not ported. |
| Mini-SWE runner | `mini_swe_runner.py` | → `missing` | missing | Not ported. |
| RL CLI | `rl_cli.py` | → `missing` | missing | Not ported. |
| Datagen config | `datagen-config-examples/` | → `missing` | missing | Not ported. |
| Tinker/Atropos environment | `tinker-atropos/` | → `missing` | missing | Not ported (empty placeholder). |

---

## 24. Hermes CLI Utilities

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Tips/tips CLI | `hermes_cli/tips.py` | → `missing` | missing | Not ported. |
| Inventory CLI | `hermes_cli/inventory.py` | → `missing` | missing | Not ported. |
| Slack CLI | `hermes_cli/slack_cli.py` | → `missing` | missing | Not ported (Slack in gateway only). |
| Dump CLI | `hermes_cli/dump.py` | → `missing` | missing | Not ported. |
| Platform CLI | `hermes_cli/platforms.py` | `cmd/gormes/gateway.go` | covered | Gateway status lists platforms. |
| Timeouts CLI | `hermes_cli/timeouts.py` | → `missing` | missing | Not ported. |
| Callbacks CLI | `hermes_cli/callbacks.py` | → `missing` | missing | Not ported. |
| Profile describer | `hermes_cli/profile_describer.py` | `internal/cli/profileseed/` | covered | Profile generation. |
| Profile distribution | `hermes_cli/profile_distribution.py` | → `missing` | missing | Not ported. |
| Relaunch (Termux) | `hermes_cli/relaunch.py` | → `missing` | missing | Not ported. |
| Azrure detect | `hermes_cli/azure_detect.py` | → `missing` | missing | Not ported. |

---

## 25. Voice, Recording, PTY

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Voice mode (TTS/STT toggle) | `tools/voice_mode.py` | → `missing` | missing | Voice toggle not proven. |
| Voice recording | `hermes_cli/voice.py` | → `missing` | missing | Not ported. |
| PTY bridge (terminal emulation) | `hermes_cli/pty_bridge.py` | → `missing` | missing | Not ported. |
| Push-to-talk keybinding | `cli.py` voice.record_key | `internal/tui/` `voiceRecordKey` | covered | Configurable voice key in TUI. |
| TTS result envelope | `tools/tts_tool.py` | → `missing` | missing | Hermes TTS tool result format. |
| WASI Whisper STT | `tools/transcription_tools.py` | `internal/tools/whisper/` | covered | Local STT via WASM. |
| Piper TTS backend | N/A (Gormes-owned) | → `missing` | owned | Gormes-owned TTS backend; not Hermes parity. |
| Voice mode state machine | `tools/voice_mode.py` | → `missing` | missing | Idle/recording/processing states. |
| TTS provider abstraction | `tools/tts_tool.py` | → `missing` | missing | Cloud/command/local TTS seam. |

---

## 26. Skin / Display / Indicator

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Skin engine | `hermes_cli/skin_engine.py` | `internal/tui/skin*.go` | covered | Built-in skins (default, ares, poseidon, mono, r1, diamond). |
| Busy indicator style | `cli.py` | `internal/tui/indicator*.go` | covered | Kaomoji, emoji, unicode, ascii styles. |
| Status bar toggle | `cli.py` | `internal/tui/status_bar*.go` | covered | Top/bottom/off modes. |
| Compact transcript | `cli.py` | `internal/tui/slash_compact.go` | covered | Compact toggle. |
| Details section visibility | `cli.py` | `internal/tui/slash_details.go` | covered | Thinking/tools/subagent visibility. |

---

## 27. Kanban (Deeper)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Kanban create/list/show/claim/complete/block/unblock | `tools/kanban_tool.py` + `hermes_cli/kanban.py` | `internal/tools/kanban/` + `internal/tui/slash_kanban.go` + `internal/gateway/` | covered | CRUD operations in tools, TUI, and gateway. |
| Kanban link/unlink | `tools/kanban_tool.py` | → `missing` | missing | Not ported. |
| Kanban comment | `tools/kanban_tool.py` | → `missing` | missing | Not ported. |
| Kanban heartbeat/reclaim/zombie | `tools/kanban_tool.py` | → `missing` | missing | Not ported. |
| Kanban init | `tools/kanban_tool.py` | `internal/tools/kanban/` | covered | Board initialization. |
| Kanban dispatch to subagent | `tools/kanban_tool.py` | `internal/tools/kanban/` | covered | Dispatch route. |
| Kanban archiving | `tools/kanban_tool.py` | → `missing` | missing | Not ported. |
| Kanban link/tail diagnostics | `hermes_cli/kanban_diagnostics.py` | → `missing` | missing | Not ported. |
| Kanban decompose | `hermes_cli/kanban_decompose.py` | → `missing` | missing | Not ported. |

---

## 28. Goal / Standing Objective

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Goal set/pause/resume/clear/status | `hermes_cli/goals.py` | `internal/tui/slash_goal.go` + `internal/tools/` | covered | Goal management in TUI and tools. |
| Goal subgoal add/remove | `hermes_cli/goals.py` | → `missing` | missing | Not ported. |
| Goal budget enforcement | `hermes_cli/goals.py` | → `missing` | missing | Not ported. |

---

## 29. Coding Agent Delegation

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Codex binary delegation | N/A (Hermes gateway delegates) | `internal/codingagents/` | covered | Codex/claude-code/opencode delegation scaffold. |
| Claude Code binary delegation | N/A | `internal/codingagents/` | covered | Shared CodingAgent interface. |
| OpenCode binary delegation | N/A | `internal/codingagents/` | covered | Shared CodingAgent interface. |

---

## 30. Explain / Format Helpers

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Hermes state constants | `hermes_constants.py` | `internal/hermes/` | partial | Not fully ported. |
| Logging (redacted) | `hermes_logging.py` | `internal/audit/` + `internal/telemetry/` | covered | Audit and telemetry logging. |
| Timezone resolution | `hermes_time.py` `_resolve_timezone_name` `get_timezone` | `internal/hermes/time_helpers.go` `GetTimezone` | covered | Reads GORMES_TIMEZONE then HERMES_TIMEZONE; returns *time.Location or nil. |
| `now()` helper | `hermes_time.py` `now` | `internal/hermes/time_helpers.go` `Now` | covered | Returns time.Now() in configured timezone or local. |
| `is_truthy_value` | `utils.py` `is_truthy_value` | `internal/hermes/helpers.go` `IsTruthyValue` | covered | Boolean coercion for nil/bool/string values. |
| `env_var_enabled` | `utils.py` `env_var_enabled` | `internal/hermes/helpers.go` `EnvVarEnabled` | covered | Check os.Getenv against truthy string set. |
| `atomic_replace` | `utils.py` `atomic_replace` | `internal/tools/atomic_replace.go` `AtomicReplace` | covered | Atomic file replacement preserving symlinks. |
| `atomic_json_write` | `utils.py` `atomic_json_write` | `internal/tools/atomic_replace.go` `AtomicWrite` | covered | Atomic file write using temp + rename. |

---

## 31. ACP Adapter

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| ACP auth/detect provider | `acp_adapter/auth.py` | → `missing` | missing | Not ported. |
| ACP server | `acp_adapter/server.py` | → `missing` | missing | Not ported. |
| ACP events | `acp_adapter/events.py` | → `missing` | missing | Not ported. |
| ACP permissions | `acp_adapter/permissions.py` | → `missing` | missing | Not ported. |
| ACP session | `acp_adapter/session.py` | → `missing` | missing | Not ported. |
| ACP tools | `acp_adapter/tools.py` | → `missing` | missing | Not ported. |

---

## 32. MCP (Deep)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| MCP server stdio | `mcp_serve.py` | `internal/mcpserver/` | covered | MCP stdio server. |
| MCP server HTTP | `mcp_serve.py` | `internal/mcpserver/` | covered | MCP HTTP transport. |
| MCP tool (client-side) | `tools/mcp_tool.py` | `internal/tools/` | partial | Tool registration exists; sessions not proven. |
| MCP OAuth flow | `tools/mcp_oauth*.py` | → `missing` | missing | Not ported. |
| MCP managed gateway | `tools/managed_tool_gateway.py` | → `missing` | missing | Not ported. |
| MCP config helpers | `hermes_cli/mcp_config.py` | `cmd/gormes/mcp.go` | covered | MCP config and login. |

---

## 33. Codex Runtime

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Codex runtime switch | `hermes_cli/codex_runtime_switch.py` | → `missing` | missing | Not ported. |
| Codex runtime plugin migration | `hermes_cli/codex_runtime_plugin_migration.py` | → `missing` | missing | Not ported. |
| Codex runtime provider | `hermes_cli/runtime_provider.py` | → `missing` | missing | Not ported. |
| Codex auth (device code) | `hermes_cli/auth.py` `_login_openai_codex` | `cmd/gormes/auth.go` | partial | Codex auth exists; paused. |
| Copilot auth | `hermes_cli/copilot_auth.py` | → `missing` | missing | Not ported. |
| Vercel auth | `hermes_cli/vercel_auth.py` | → `missing` | missing | Not ported. |
| DingTalk auth | `hermes_cli/dingtalk_auth.py` | → `missing` | missing | Not ported. |

---

## 34. Root Environments (Research)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Agentic OPD environment | `environments/agentic_opd_env.py` | → `missing` | missing | Not ported. |
| Web research environment | `environments/web_research_env.py` | → `missing` | missing | Not ported. |
| Hermes base environment | `environments/hermes_base_env.py` | → `missing` | missing | Not ported. |
| DeepSeek tool-call parser | `environments/tool_call_parsers/deepseek_parser.py` | → `missing` | missing | Not ported. |
| Qwen tool-call parser | `environments/tool_call_parsers/qwen_parser.py` | → `missing` | missing | Not ported. (duplicate of section 17 — noted in both for completeness) |

---

## 35. Gateway Message Rendering (Channel-neutral)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Tool progress emoji per channel | `agent/display.py` `get_tool_emoji` | `internal/tooltrace/` | covered | Shared renderer with channel-specific output. |
| Tool progress modes (new/all/off) | `gateway/display_config.py` | `internal/config/` + `internal/tooltrace/` | covered | Display.config.tool_progress mode. |
| Stream consumer: cursor handling | `gateway/stream_consumer.py` | `internal/gateway/render.go` | covered | Cursor management. |
| Stream consumer: fresh-final separation | `gateway/stream_consumer.py` | `internal/gateway/coalesce.go` | covered | Fresh final answer replace placeholder. |
| Stream consumer: tool progress | `gateway/stream_consumer.py` | `internal/gateway/render.go` | covered | Tool progress rendered inline. |
| Error formatting (redacted, no HTML) | `gateway/run.py` `:4396-4439` | `internal/gateway/render.go` | covered | Secret and HTML sanitization. |
| Typing action (Telegram) | `gateway/platforms/base.py` | `internal/channels/telegram/bot.go` | partial | Placeholder sent; typing action not proven. |
| Stale placeholder cleanup | `gateway/platforms/base.py` `:1718-1724` | `internal/gateway/coalesce.go` | covered | Fresh-final deletes and replaces. |
| Duplicate message suppression | `gateway/run.py` | `internal/gateway/` | partial | Restart duplicate suppressed; chat text dedup missing. |
| Silent notification defaults | `gateway/platforms/telegram.py` | `internal/channels/telegram/thread_send_test.go` | covered | Placeholder sends silent; finals notify. |

---

## 36. Hermes CLI Command Alias And Suggestion

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Alias canonicalization | `hermes_cli/commands.py` `resolve_command` | `internal/cli/command_registry.go` | covered | Command aliases canonicalize. |
| Unique prefix dispatch | `hermes_cli/_parser.py` | `internal/cli/` | partial | Unique prefix dispatch not proven. |
| Ambiguous prefix guidance | `hermes_cli/_parser.py` | `internal/tui/slash_dispatch.go` | covered | Ambiguous command guidance in TUI. |
| Quick-command aliases (preserve args) | `hermes_cli/fallback_cmd.py` | → `missing` | missing | Not ported. |

---

## 37. Security Advisories

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Security advisory CLI | `hermes_cli/security_advisories.py` | → `missing` | missing | Not ported. |
| Supply-chain audit CI | `.github/workflows/supply-chain-audit.yml` | → `missing` | missing | Not ported. |
| Advisory class/detection | `security_advisories.py` `Advisory` `detect_compromised` | → `missing` | missing | Advisory data model and compromise detection. |
| Advisory ack/ignore | `security_advisories.py` `ack_advisory` `get_acked_ids` | → `missing` | missing | Persist acknowledged advisories. |
| Startup banner | `security_advisories.py` `startup_banner` | → `missing` | missing | Show unacknowledged advisories at CLI start. |
| Doctor section | `security_advisories.py` `render_doctor_section` | → `missing` | missing | Report advisory status in doctor. |
| Gateway log message | `security_advisories.py` `gateway_log_message` | → `missing` | missing | Log advisory hits on gateway startup. |

---

## 38. Skills Index / Catalog

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Skills index CI | `.github/workflows/skills-index.yml` | → `missing` | missing | Not ported. |
| Skills hub | `hermes_cli/skills_hub.py` | `cmd/gormes/skills.go` + `internal/skills/` | covered | Skill install/search/list/inspect. |
| Skills guard | `tools/skills_guard.py` | → `missing` | missing | Not ported. |
| Skills sync | `tools/skills_sync.py` | → `missing` | missing | Not ported. |
| Skills index cache | `skills/index-cache/*.json` | → `missing` | missing | Not ported. |

---

## 39. Cron (Deep)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Cron schedule parser | `cron/scheduler.py` | `internal/cron/` | covered | Schedule parsing. |
| Cron job store | `cron/jobs.py` | `internal/cron/` | covered | Job CRUD. |
| Cron job execution | `cron/scheduler.py` | `internal/cron/` | covered | Scheduled execution. |
| Cron context_from chaining | `cron/jobs.py` | → `missing` | missing | Not ported. |
| Cron multi-target delivery | `cron/scheduler.py` | → `missing` | missing | Not ported. |
| Cron resource release | `cron/scheduler.py` | → `missing` | missing | Not ported. |
| Cron tool (gateway) | `tools/cronjob_tools.py` | `internal/tools/` | covered | Cron management tool. |

---

## 40. Backup / Restore

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Full backup CLI | `hermes_cli/backup.py` `run_backup` | → `missing` | missing | Full backup of config/sessions/memory. |
| Import CLI | `hermes_cli/backup.py` `run_import` | → `missing` | missing | Import from backup zip. |
| Backup validation | `hermes_cli/backup.py` `_validate_backup_zip` | → `missing` | missing | Validate backup integrity. |
| Quick snapshot create | `hermes_cli/checkpoints.py` `create_quick_snapshot` | → `missing` | missing | Pre-operation snapshot. |
| Quick snapshot list | `hermes_cli/checkpoints.py` `list_quick_snapshots` | → `missing` | missing | List available snapshots. |
| Checkpoint TUI | `hermes_cli/checkpoints.py` | `internal/tui/slash_checkpoint.go` | partial | Checkpoint via TUI slash command. |
| Rollback TUI | `hermes_cli/checkpoints.py` | `internal/tui/slash_rollback.go` | partial | Rollback via TUI slash command. |
| Snapshot prune | `hermes_cli/checkpoints.py` `cmd_prune` | → `missing` | missing | Remove old snapshots. |
| Snapshot clear (legacy) | `hermes_cli/checkpoints.py` `cmd_clear_legacy` | → `missing` | missing | Clear legacy checkpoint format. |

---

## 41. Model Switch

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Model switch CLI (interactive) | `hermes_cli/model_switch.py` | `cmd/gormes/model.go` + `internal/gateway/model_picker.go` | covered | Interactive provider/model picker. |
| Model catalog suggestions | `hermes_cli/model_catalog.py` | `internal/hermes/provider_registry_manifest.go` | covered | Provider-specific model catalog. |
| Model normalize | `hermes_cli/model_normalize.py` | → `missing` | missing | Not ported. |
| Direct alias resolution | `model_switch.py` `_ensure_direct_aliases` | `internal/hermes/model_switch.go` `DirectAlias` | covered | DirectAlias type with Model/Provider/BaseURL fields. |
| ModelIdentity parsing | `model_switch.py` `ModelIdentity` | `internal/hermes/model_switch.go` `ModelIdentity` + `ModelAliases` | covered | 22 built-in model aliases with vendor/family. |
| ModelSwitchResult | `model_switch.py` `ModelSwitchResult` | `internal/hermes/model_switch.go` `ModelSwitchResult` | covered | Structured result with success/newModel/provider/isGlobal/error. |
| `--global` flag support | `model_switch.py` `parse_model_flags` | `internal/hermes/model_switch.go` `ParseModelFlags` | covered | Parses --provider, --global, unicode dash normalization. |
| Model sort key | `model_switch.py` `_model_sort_key` | `internal/hermes/model_switch.go` `ModelSortKey` | covered | Deterministic sort key + SortedModelAliases. |

---

## 42. Send Command (CLI)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Send text message CLI | `hermes_cli/send_cmd.py` `cmd_send` | → `missing` | missing | Send message to a channel. |
| Oneshot chat (-q) | `hermes_cli/oneshot.py` | `cmd/gormes/chat.go` `-q` | covered | One-shot message to default provider. |
| Target resolution | `send_cmd.py` `_resolve_target` | → `missing` | missing | Resolve `--to platform:chat` targets. |
| Platform-aware target listing | `send_cmd.py` `_list_targets` | → `missing` | missing | List available send targets. |
| Message body reading | `send_cmd.py` `_read_message_body` | → `missing` | missing | Read message body from stdin/file/stdin. |
| Result formatting | `send_cmd.py` `_emit_result` | → `missing` | missing | Display send result. |
| Send subcommand registration | `send_cmd.py` `register_send_subparser` | → `missing` | missing | Register `gormes send` subcommand. |

---

## 43. Webhook (Gateway)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Webhook subscriptions load | `hermes_cli/webhook.py` `_load_subscriptions` | → `missing` | missing | Load webhook subscriptions. |
| Webhook subscriptions save | `hermes_cli/webhook.py` `_save_subscriptions` | → `missing` | missing | Persist webhook subscriptions. |
| Webhook config detection | `hermes_cli/webhook.py` `_get_webhook_config` | → `missing` | missing | Detect webhook config. |
| Webhook enabled check | `hermes_cli/webhook.py` `_is_webhook_enabled` | → `missing` | missing | Check if webhook is enabled in config. |
| Webhook base URL | `hermes_cli/webhook.py` `_get_webhook_base_url` | → `missing` | missing | Derive webhook base URL from config. |
| Webhook setup hint | `hermes_cli/webhook.py` `_setup_hint` | → `missing` | missing | Setup guidance for webhook. |
| Webhook command dispatch | `hermes_cli/webhook.py` `webhook_command` | → `missing` | missing | CLI entry point. |
| Webhook channel delivery | `gateway/platforms/webhook.py` | `internal/channels/webhook/` | covered | Webhook channel for outbound. |

---

## 44. PTY / Terminal Mid-session

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| PTY bridge spawn | `hermes_cli/pty_bridge.py` `PtyBridge.spawn` | → `missing` | missing | Spawn PTY for interactive terminal. |
| PTY read | `pty_bridge.py` `read` | → `missing` | missing | Read PTY output with timeout. |
| PTY write | `pty_bridge.py` `write` | → `missing` | missing | Write to PTY stdin. |
| PTY resize | `pty_bridge.py` `resize` | → `missing` | missing | Resize PTY dimensions. |
| PTY close/cleanup | `pty_bridge.py` `close` | → `missing` | missing | Close PTY and release resources. |
| PTY availability check | `pty_bridge.py` `is_available` | → `missing` | missing | Check if PTY is supported on platform. |
| PTY process PID | `pty_bridge.py` `pid` | → `missing` | missing | Get PTY child PID. |
| PTY aliveness check | `pty_bridge.py` `is_alive` | → `missing` | missing | Check if child process is still running. |
| Terminal process registry | `tools/code_execution_tool.py` | → `missing` | missing | Track active terminal sessions. |
| Terminal size/colour passthrough | `tools/code_execution_tool.py` | → `missing` | missing | Forward terminal dimensions to PTY. |
| PTY error handling | `pty_bridge.py` `PtyUnavailableError` | → `missing` | missing | Typed error when PTY not available. |

---

## 45. Environment Passthrough

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Credential files registration | `tools/credential_files.py` `register_credential_file` | → `missing` | missing | Register files for credential passthrough. |
| Credential file mounts | `tools/credential_files.py` `get_credential_file_mounts` | → `missing` | missing | Resolve credential mounts for sandbox. |
| Skills directory mount | `tools/credential_files.py` `get_skills_directory_mount` | → `missing` | missing | Mount skills dir read-only. |
| Skills files iterator | `tools/credential_files.py` `iter_skills_files` | → `missing` | missing | Iterate skill files for sandbox. |
| Env passthrough registration | `tools/env_passthrough.py` `register_env_passthrough` | → `missing` | missing | Allowlist env vars for sandbox passthrough. |
| Env passthrough check | `tools/env_passthrough.py` `is_env_passthrough` | → `missing` | missing | Check if a var is allowed through. |
| Config passthrough load | `tools/env_passthrough.py` `_load_config_passthrough` | → `missing` | missing | Load passthrough allowlist from config. |
| Env passthrough clear | `tools/env_passthrough.py` `clear_env_passthrough` | → `missing` | missing | Reset passthrough state. |

---

## 46. AIM / Character / Personality

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Personality list command | `hermes_cli/commands.py` `/personality` | `internal/gateway/personality_command.go` `handlePersonalityCommand` | covered | Lists available personalities with descriptions. |
| Personality switch command | `hermes_cli/commands.py` | `internal/gateway/personality_command.go` | covered | Switches active personality by name. |
| Personality prompt injection | `run_agent.py` | `internal/hermes/prompt_assembly.go` `PromptAssemblyOptions.Personality` | covered | Injects personality block when ActivePersonality is set. |\n| Personality source file loading | `hermes_cli/config.py` personalities config | `internal/config/config.go` `AgentCfg.Personalities` | covered | Loads personalities from config agent.personalities map. |\n| Personality `none` clear | `hermes_cli/commands.py` `personality none` | `internal/gateway/personality_command.go` | covered | Clears active personality via /personality none. |]

---

## 47. TUI Gateway (tuigateway)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| TUI as gateway adapter | `tui_gateway/` | `internal/tuigateway/` | covered | TUI gateway adapter. |
| TUI extension widgets | N/A (Gormes-owned) | `internal/tui/extension*.go` | owned | Gormes-owned in-process extension UI. |
| TUI welcome panel | `ui-tui/src/` | `internal/tui/banner.go` | covered | Welcome with version/tips. |
| TUI page panels (history, usage, skin) | `ui-tui/src/` | `internal/tui/slash*.go` | covered | Transient panels for slash commands. |

---

## 48. Default Soul / Prompt Identity

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Default SOUL.md identity | `hermes_cli/default_soul.py` | `internal/hermes/context_files.go` | covered | Context file loading from GORMES_HOME. |
| SOUL.md file discovery | `run_agent.py` | `internal/hermes/context_files.go` | covered | Workspace ancestor, Gormes home, Hermes home chain. |
| SOUL.md frontmatter stripping | `agent/prompt_builder.py` | `internal/hermes/context_files.go` | covered | Frontmatter removed before injection. |
| Truncation marker | `agent/prompt_builder.py` | `internal/hermes/context_files.go` | covered | `[...truncated ... kept H+T of N chars ...]` |
| Threat pattern scan | `agent/prompt_builder.py` | `internal/hermes/context_files.go` | covered | `[BLOCKED:` marker for prompt injection. |

---

## 49. Self-help / Quick Commands

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Self-help guidance ("what can you do") | `run_agent.py` | `internal/hermes/guidance_constants.go` | covered | Byte-equivalent self-help guidance. |
| Quick commands (keyboard shortcuts) | `hermes_cli/__init__.py` | → `missing` | missing | Not ported. |

---

## 50. STDIO / Pipe Mode

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| STDIO mode | `hermes_cli/stdio.py` | → `missing` | missing | Not ported. |
