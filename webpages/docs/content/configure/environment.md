---
title: "Environment"
description: "Every environment variable Gormes reads, grouped by family with verified defaults."
weight: 32
aliases:
  - /reference/environment/
---

# Environment

Gormes resolves settings in this order:

1. CLI flags
2. Environment variables (including values loaded from `$GORMES_HOME/.env`)
3. `config.toml`
4. Built-in defaults

The dotenv file at `$GORMES_HOME/.env` is read before `config.toml` and
populates `os.Setenv` for any key the shell did not already set — so shell env
always wins. There is no project-local dotenv search; only the file at
`$GORMES_HOME/.env` is loaded automatically.

The variables below are grouped by what they affect at runtime. Every name and
default in this page is verified against the binary source. Variables not
listed here are either test-only fixtures or aliases of variables shown here.

## Core runtime

These variables affect process-wide behavior — paths, language, voice, and the
inference identity Gormes uses when starting a turn.

| Name | Default | Purpose |
|---|---|---|
| `GORMES_HOME` | `~/.gormes` | Native Gormes state/config root. Drives every other path in this table. |
| `GORMES_CONTEXT_HOME` | `$GORMES_HOME` (with a `memory/` migration fallback, then ancestor walk for `SOUL.md`) | Override the live-turn profile directory. |
| `GORMES_CONTEXT_MEMORY_DIR` | resolved relative to context home | Override the live-turn memory directory. |
| `GORMES_SOURCE_ROOT` / `GORMES_SOURCE_DIR` | unset | Override the managed source checkout location used by `gormes update`. |
| `GORMES_BUNDLED_SKILLS_ROOT` | binary-bundled skills tree | Override the bundled-skill root used by skill discovery. |
| `GORMES_LANGUAGE` | `""` (falls back to `LANG`) | UI language code (e.g. `en`, `zh`). |
| `GORMES_LOCALES_DIR` | binary-bundled `locales/` | Override the locale file directory. |
| `GORMES_NO_CLEAR_SCREEN` | unset | When truthy, suppress wizard screen clears for sandboxed terminals. |
| `GORMES_TUI_MOUSE_TRACKING` | `true` (from `[tui]`) | Enable or disable TUI mouse capture. |
| `GORMES_VOICE_RECORD_KEY` | `ctrl+b` | Voice-record hot-key. |
| `GORMES_INFERENCE_PROVIDER` | unset | Provider override for TUI/scripted chat startup. Must be set together with `GORMES_INFERENCE_MODEL`; setting only one is a hard error. |
| `GORMES_INFERENCE_MODEL` | unset | Model override for TUI/scripted chat startup. Must be set together with `GORMES_INFERENCE_PROVIDER`. |
| `GORMES_ENDPOINT` | from `[hermes].endpoint` (default `""`) | Provider base URL override. |
| `GORMES_MODEL` | from `[hermes].model` (default `"hermes-agent"`) | Persistent default model override. |
| `GORMES_API_KEY` | from `[hermes].api_key` (default `""`) | Provider API key. Stored in `.env`. |
| `GORMES_ALLOW_PRIVATE_URLS` | unset | When truthy, web tools may target private IPs. |
| `GORMES_ENABLE_PROJECT_PLUGINS` (or legacy `HERMES_ENABLE_PROJECT_PLUGINS`) | unset | Opt in to per-project plugin discovery. |
| `GORMES_EXTRACTOR_API_KEY` | unset | API key used by the memory extractor's HTTP client during integration tests. |
| `GORMES_SEMANTIC_MODEL` | `nomic-embed-text` | Embedding model name for semantic memory paths. |

## Kanban

Native Kanban storage lives under `$GORMES_HOME` by default. Override only
when running side-by-side fleets or migrating from Hermes.

| Name | Default | Purpose |
|---|---|---|
| `GORMES_KANBAN_DB` | `$GORMES_HOME/kanban.db` | Absolute path to the Kanban SQLite database. |
| `GORMES_KANBAN_HOME` | `$GORMES_HOME` | Root for the Kanban board registry. Named boards live at `<home>/kanban/boards/<slug>/kanban.db`. |
| `HERMES_KANBAN_BOARD` | unset | Hermes-compatible alias for selecting a named board. |

## Web, browser, and gateway

Routing for web search, browser tools, and the outbound gateway proxy.

| Name | Default | Purpose |
|---|---|---|
| `GORMES_WEB_BACKEND` | from `[web].backend` (default `""`) | Web search backend name. Lowercased on read. |
| `GORMES_WEB_USE_GATEWAY` | from `[web].use_gateway` (default `false`) | Route web requests through the gateway. |
| `GORMES_BROWSER_CDP_URL`, `BROWSER_CDP_URL`, `CHROME_REMOTE_DEBUGGING_URL` | from `[browser].cdp_url` (default `""`) | CDP endpoint. The three are evaluated in this order; the first non-empty value wins. |
| `GORMES_BROWSER_HARNESS_STATE_DIR` | OS temp dir | Override the browser harness state directory. |
| `GORMES_DASHBOARD_API_KEY` | `gormes-dashboard-dev` when starting the dashboard; required for cross-process discovery | Bearer token for the local dashboard and gateway discovery clients. |
| `GATEWAY_PROXY_URL` | from `[gateway].proxy_url` (default `""`) | Outbound HTTP proxy URL. |
| `GATEWAY_PROXY_KEY` | from `[gateway].proxy_key` (default `""`) | Outbound proxy bearer. Auto-stored in `.env`. |

## Goncho

Honcho-compatible memory facade. Every variable below maps 1:1 onto a
`[goncho]` config field. Defaults come from the `goncho` package, mirrored
into config defaults at startup.

| Name | Default | Purpose |
|---|---|---|
| `GORMES_GONCHO_ENABLED` | `true` | Master toggle. |
| `GORMES_GONCHO_WORKSPACE` | `goncho.DefaultWorkspaceID` | Workspace identifier. |
| `GORMES_GONCHO_OBSERVER_PEER` | `goncho.DefaultObserverPeerID` | Observer peer identifier. |
| `GORMES_GONCHO_RECENT_MESSAGES` | `goncho.DefaultRecentMessages` | Recent-window message count. |
| `GORMES_GONCHO_MAX_MESSAGE_SIZE` | `goncho.DefaultMaxMessageSize` | Per-message byte ceiling. |
| `GORMES_GONCHO_MAX_FILE_SIZE` | `goncho.DefaultMaxFileSize` | Per-file byte ceiling. |
| `GORMES_GONCHO_GET_CONTEXT_MAX_TOKENS` | `goncho.DefaultGetContextMaxTokens` | Token budget for `get_context`. |
| `GORMES_GONCHO_REASONING_ENABLED` | `true` | Persist reasoning blocks. |
| `GORMES_GONCHO_PEER_CARD_ENABLED` | `true` | Persist peer cards. |
| `GORMES_GONCHO_SUMMARY_ENABLED` | `true` | Persist summaries. |
| `GORMES_GONCHO_DREAM_ENABLED` | `false` | Enable idle/dream consolidation. |
| `GORMES_GONCHO_DREAM_IDLE_TIMEOUT_MINUTES` | derived from `goncho.DefaultDreamIdleTimeout` | Idle minutes before dream runs. |
| `GORMES_GONCHO_DERIVER_WORKERS` | `goncho.DefaultDeriverWorkers` | Deriver worker pool size. |
| `GORMES_GONCHO_REPRESENTATION_BATCH_MAX_TOKENS` | `goncho.DefaultRepresentationBatchMaxTokens` | Representation batch token cap. |
| `GORMES_GONCHO_DIALECTIC_DEFAULT_LEVEL` | `"low"` | Default dialectic level. |

## Telegram

Gormes prefers `GORMES_TELEGRAM_*` and accepts the Hermes-compatible
`TELEGRAM_*` names so a copied `.env` keeps working. Native names win when
both are set. See [Telegram](../telegram/) for the intended setup
path.

| Name | Default | Purpose |
|---|---|---|
| `GORMES_TELEGRAM_BOT_TOKEN` / `GORMES_TELEGRAM_TOKEN` / `TELEGRAM_BOT_TOKEN` / `TELEGRAM_TOKEN` | from `[telegram].bot_token` (default `""`) | Bot token. First non-empty wins. |
| `GORMES_TELEGRAM_CHAT_ID` / `TELEGRAM_HOME_CHANNEL` / `TELEGRAM_CHAT_ID` | from `[telegram].allowed_chat_id` (default `0`) | Single allowed chat ID for cron delivery and gateway allow-listing. |
| `GORMES_TELEGRAM_ALLOWED_USERS` / `TELEGRAM_ALLOWED_USERS` | from `[telegram].allowed_user_ids` | Comma-separated user IDs allowed to message the bot. |
| `GORMES_TELEGRAM_ALLOWED_CHATS` / `TELEGRAM_ALLOWED_CHATS` | from `[telegram].allowed_chats` | Comma-separated additional allow-list chat IDs. |
| `GORMES_TELEGRAM_GUEST_MODE` / `TELEGRAM_GUEST_MODE` | from `[telegram].guest_mode` (default `false`) | Allow unknown users in read-only mode. |
| `GORMES_TELEGRAM_NOTIFICATIONS` / `HERMES_TELEGRAM_NOTIFICATIONS` / `TELEGRAM_NOTIFICATIONS` | from `[telegram].notifications` (default `"important"`) | Notification level. |

## Discord

| Name | Default | Purpose |
|---|---|---|
| `GORMES_DISCORD_TOKEN` | from `[discord].token` (default `""`) | Bot token. |
| `GORMES_DISCORD_CHANNEL_ID` | from `[discord].allowed_channel_id` (default `""`) | Primary allow-list channel ID. |
| `GORMES_DISCORD_ALLOWED_CHANNELS` / `DISCORD_ALLOWED_CHANNELS` | from `[discord].allowed_channels` | CSV of channel IDs. |
| `GORMES_DISCORD_IGNORED_CHANNELS` / `DISCORD_IGNORED_CHANNELS` | from `[discord].ignored_channels` | CSV of ignored channel IDs. |
| `GORMES_DISCORD_FREE_RESPONSE_CHANNELS` / `DISCORD_FREE_RESPONSE_CHANNELS` | from `[discord].free_response_channels` | Channels where mentions are not required. |
| `GORMES_DISCORD_NO_THREAD_CHANNELS` / `DISCORD_NO_THREAD_CHANNELS` | from `[discord].no_thread_channels` | Channels where threads must not be created. |
| `GORMES_DISCORD_REQUIRE_MENTION` / `DISCORD_REQUIRE_MENTION` | from `[discord].require_mention` (default `false`) | Require @bot mention. |
| `GORMES_DISCORD_AUTO_THREAD` / `DISCORD_AUTO_THREAD` | from `[discord].auto_thread` | Auto-thread incoming requests. |
| `GORMES_DISCORD_REPLY_TO_MODE` / `DISCORD_REPLY_TO_MODE` | from `[discord].reply_to_mode` (default `"first"`) | One of `first`, `all`, `off`. |
| `GORMES_DISCORD_ALLOW_BOTS` / `DISCORD_ALLOW_BOTS` | from `[discord].allow_bots` (default `"none"`) | One of `none`, `mentions`, `all`. |
| `GORMES_DISCORD_SERVER_ACTIONS` | from `[discord].server_actions` | CSV of allowed server actions. |

## Slack

| Name | Default | Purpose |
|---|---|---|
| `GORMES_SLACK_ENABLED` | from `[slack].enabled` (default `false`) | Master toggle. |
| `GORMES_SLACK_BOT_TOKEN` | from `[slack].bot_token` (default `""`) | xoxb token. |
| `GORMES_SLACK_APP_TOKEN` | from `[slack].app_token` (default `""`) | xapp token. |
| `GORMES_SLACK_CHANNEL_ID` | from `[slack].allowed_channel_id` (default `""`) | Primary allow-list channel ID. Slack has no `default_channel_id`. |
| `GORMES_SLACK_ALLOWED_CHANNELS` / `SLACK_ALLOWED_CHANNELS` | from `[slack].allowed_channels` | CSV of channel IDs. |
| `GORMES_SLACK_COALESCE_MS` | from `[slack].coalesce_ms` (default `1000`) | Edit coalescer interval in milliseconds. |
| `GORMES_SLACK_FIRST_RUN_DISCOVERY` | from `[slack].first_run_discovery` (default `false`) | Discover channels on first run. |
| `GORMES_SLACK_REQUIRE_MENTION` | from `[slack].require_mention` (default `true`) | Require @mention. |
| `GORMES_SLACK_STRICT_MENTION` | from `[slack].strict_mention` | Reject indirect mentions. |
| `GORMES_SLACK_FREE_RESPONSE_CHANNELS` | from `[slack].free_response_channels` | Channels where mentions are not required. |
| `GORMES_SLACK_REPLY_IN_THREAD` | from `[slack].reply_in_thread` (default `true`) | Reply in thread. |

## Teams

Microsoft Teams uses bare `TEAMS_*` names rather than `GORMES_TEAMS_*`. Only
the master toggle is namespaced.

| Name | Default | Purpose |
|---|---|---|
| `GORMES_TEAMS_ENABLED` | from `[teams].enabled` (default `false`) | Master toggle. |
| `TEAMS_CLIENT_ID` | from `[teams].client_id` (default `""`) | Azure app client ID. |
| `TEAMS_CLIENT_SECRET` | from `[teams].client_secret` (default `""`) | Azure app secret. |
| `TEAMS_TENANT_ID` | from `[teams].tenant_id` (default `""`) | Azure tenant ID. |
| `TEAMS_PORT` | from `[teams].port` (default `3978`) | Local webhook port. |
| `TEAMS_ALLOWED_USERS` | from `[teams].allowed_users` (default `[]`) | CSV of allowed user IDs. |
| `TEAMS_ALLOW_ALL_USERS` | from `[teams].allow_all_users` (default `false`) | Skip the user allow-list. |

## TTS and STT

Voice synthesis and transcription. TTS routes through `[runtime].tts_provider`
(default `"edge"`), then per-provider env vars apply.

| Name | Default | Purpose |
|---|---|---|
| `GORMES_TTS_COMMAND` | unset | External command for the `command` TTS provider. |
| `GORMES_TTS_VOICE` | unset | Voice name passed to the active TTS provider. |
| `GORMES_TTS_OPENAI_KEY` (or `OPENAI_API_KEY`) | unset | API key for OpenAI TTS. |
| `GORMES_TTS_EDGE_KEY` / `GORMES_AZURE_TTS_KEY` | unset | API key for Edge TTS. |
| `GORMES_TTS_EDGE_REGION` | unset | Azure region for Edge TTS. |
| `GORMES_TTS_EDGE_BASE_URL` | unset | Endpoint override for Edge TTS. |
| `GORMES_STT_OPENAI_KEY` (or `OPENAI_API_KEY`, `VOICE_TOOLS_OPENAI_KEY`) | unset | API key for OpenAI Whisper STT. |
| `GORMES_STT_OPENAI_BASE_URL` | unset | Endpoint override for OpenAI Whisper STT. |
| `GORMES_STT_CACHE_DIR` | OS temp | Local cache directory for STT artifacts. |
| `GORMES_WASI_WHISPER_MODEL` | bundled default | Model identifier for the WASI Whisper transcriber. |
| `GORMES_WHISPER_COMMAND` | unset | External whisper command for Telegram audio transcription. |
| `GORMES_WHISPER_MODEL` | unset | Model passed to `GORMES_WHISPER_COMMAND`. |

## Installer

These variables are read by `install.sh` and `scripts/install.ps1` only — the
running binary does not inspect them. See [Paths & logs](../paths/)
for the resolved install layout.

| Name | Default | Purpose |
|---|---|---|
| `GORMES_PREFIX` | unset | Compatibility prefix: installer publishes into `$GORMES_PREFIX/bin`. |
| `GORMES_BIN_DIR` | platform-default `bin/` | Published binary directory override. |
| `GORMES_RESTART_GATEWAY` | `auto` | Restart policy for the gateway after install: `auto`, `always`, or `never`. |
| `GORMES_UNINSTALL_FORCE_PURGE` | unset | When truthy, `gormes uninstall` permanently deletes instead of moving artifacts to a quarantine directory. |
