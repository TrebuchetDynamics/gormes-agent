---
title: "Config file"
description: "The canonical config.toml schema for Gormes, with verified defaults from the binary."
weight: 31
aliases:
  - /reference/config/
---

# Config file

Gormes loads exactly one config file:

```
$GORMES_HOME/config.toml
```

The default `$GORMES_HOME` is `~/.gormes`. There is no project-local override,
no XDG path, and no implicit YAML location — the only YAML fallback is
`$GORMES_HOME/config.yaml`, which exists solely so a Hermes user can paste
their `config.yaml` while migrating. Secrets are never written to
`config.toml`; they live in `$GORMES_HOME/.env`.

Precedence: CLI flag > environment variable > `config.toml` > built-in default.

Use the binary to inspect and edit the file:

```bash
gormes config path          # absolute path to the active config
gormes config show          # resolved config with secrets redacted
gormes config get <key>     # read one value
gormes config set <k> <v>   # write to config.toml or .env (auto-routed)
gormes config check         # schema + dotenv presence
gormes config edit          # open in $EDITOR / $VISUAL
gormes config migrate       # apply Gormes-native schema migrations
```

## Schema

Every section below is read by `internal/config`. Defaults are the values the
binary applies when the key is missing. Boolean values default to `false`
unless noted.

### `[hermes]`

Provider and model identity for the default agent.

| Field | Default | Purpose |
|---|---|---|
| `endpoint` | `""` (empty — provider client picks the URL) | Override the provider's base URL. |
| `provider` | `""` | Provider name (e.g. `openai`, `anthropic`, `openrouter`). |
| `model` | `"hermes-agent"` | Default model. The literal `hermes-agent` is a sentinel that triggers per-provider default-model resolution. |
| `api_key` | `""` | API key. Always written to `.env`, never to `config.toml`. |
| `api_key_ref` | unset | Optional `SecretRef` pointing at an external secrets provider. |

### `[agent]`

Per-turn runtime knobs for the default agent.

| Field | Default | Purpose |
|---|---|---|
| `max_turns` | `60` | Hard cap on agent turns per session. |
| `reasoning_effort` | `"medium"` | Reasoning effort hint for compatible providers. |
| `gateway_timeout` | `1800` (seconds) | Hermes-compatible gateway request timeout. |
| `gateway_timeout_warning` | `0` | Warning threshold for slow turns. |
| `api_max_retries` | `3` | Retry budget for provider calls. |
| `image_input_mode` | `""` | Image input handling for vision-capable models. |
| `verbose` | `false` | Verbose turn diagnostics. |
| `active_personality` | `""` | Name of the personality preset to apply. |
| `personalities` | built-in map (`helpful`, `concise`, `technical`, `creative`, `teacher`, `kawaii`, `catgirl`, `noir`, `pirate`, `philosopher`, `hype`, `shakespeare`) | Personality preset text. Operators may add or override entries. |

### `[runtime]`

Cross-agent runtime policy.

| Field | Default | Purpose |
|---|---|---|
| `max_tool_iterations` | `90` | Per-turn tool call ceiling. |
| `terminal_backend` | `"local"` | Default terminal backend label. |
| `tts_provider` | `"edge"` | Default TTS provider. |
| `compression_threshold` | `0.5` | History compression trigger. |
| `session_reset_policy` | `"inactivity"` | One of `inactivity`, `daily`, `off`. |
| `session_reset_after_minutes` | `1440` | Inactivity window (24 hours). |
| `session_reset_daily_hour` | `4` | Hour of the day for daily reset. |
| `session_reset_memory_summary` | `true` | Capture a memory summary on reset. |

### `[tts]`

Free-form provider-keyed TTS settings (`map[string]any`). Provider-specific
keys live under `[tts.<provider>]`. Defaults to `{}`.

### `[image_gen]`

Free-form provider-keyed image generation settings. Defaults to `{}`.

### `[gateway]`

Hermes-compatible gateway transport.

| Field | Default | Purpose |
|---|---|---|
| `proxy_url` | `""` | Outbound proxy URL. Mirror of `GATEWAY_PROXY_URL`. |
| `proxy_key` | `""` | Proxy bearer token. Auto-routed to `.env` when set via `gormes config set`. |
| `platforms.<name>.gateway_restart_notification` | unset | Per-platform restart-notification toggle. |

### `[terminal]`

| Field | Default | Purpose |
|---|---|---|
| `backend` | `""` (uses `runtime.terminal_backend`) | Override terminal backend per-section. |
| `cwd` | `"."` | Default working directory for terminal sessions. |

### `[code_execution]`

| Field | Default | Purpose |
|---|---|---|
| `mode` | `"strict"` | `execute_code` policy. `strict` blocks anything outside the shell sandbox. |

### `[display]`

| Field | Default | Purpose |
|---|---|---|
| `language` | `""` | UI language. Overridden by `GORMES_LANGUAGE`. |
| `personality` | `""` | Personality preset name shown in UI. |
| `tool_progress` | `""` | Tool progress display mode. |
| `tool_progress_command` | `false` | Show the command for each tool step. |
| `show_reasoning` | `false` | Render reasoning blocks in the TUI/transcripts. |
| `streaming` | `false` | Stream provider output. |
| `bell_on_complete` | `false` | Terminal bell at end of turn. |
| `compact` | `false` | Compact transcript layout. |
| `cleanup_progress` | `false` | Erase progress lines after completion. |
| `interim_assistant_messages` | `false` | Show interim assistant turns. |
| `background_process_notifications` | `""` | Background-task notification mode. |
| `busy_input_mode` | `""` | TUI behavior while a turn is in flight. |
| `platforms.<name>.tool_progress` | unset | Per-platform tool progress override. |

### `[tui]`

| Field | Default | Purpose |
|---|---|---|
| `theme` | `"dark"` | TUI theme. |
| `mouse_tracking` | `true` | Enable mouse capture. Override with `GORMES_TUI_MOUSE_TRACKING`. |

### `[input]`

| Field | Default | Purpose |
|---|---|---|
| `max_bytes` | `200000` | Maximum bytes the input box accepts in one submission. |
| `max_lines` | `10000` | Maximum lines per submission. |

### `[approvals]`

| Field | Default | Purpose |
|---|---|---|
| `cron_mode` | `"deny"` | Approval policy for cron-scheduled actions. |

### `[voice]`

| Field | Default | Purpose |
|---|---|---|
| `record_key` | `"ctrl+b"` | TUI hot-key for voice recording. Override with `GORMES_VOICE_RECORD_KEY`. |

### `[stt]`

| Field | Default | Purpose |
|---|---|---|
| `enabled` | `false` | Enable STT for voice inputs. |
| `provider` | `""` | STT provider name (e.g. `openai`, `local`). |
| `local.model` | `""` | Local model identifier. |
| `local.language` | `""` | Local model language hint. |
| `openai.model` | `""` | OpenAI Whisper model name. |

### `[auxiliary]`

Auxiliary inference tasks (curator, vision). Each subkey accepts the same
shape.

| Field | Default | Purpose |
|---|---|---|
| `auxiliary.curator.provider` | `"auto"` | Provider override (`auto`, `main`, or any provider name). |
| `auxiliary.curator.model` | `""` | Model override. |
| `auxiliary.curator.base_url` | `""` | Endpoint override. |
| `auxiliary.curator.api_key` | `""` | API key override (routed to `.env`). |
| `auxiliary.curator.timeout` | `600` | Timeout seconds. |
| `auxiliary.curator.extra_body` | `{}` | Free-form provider request extras. |
| `auxiliary.vision.*` | empty | Same shape, vision provider. |

### `[curator]`

Legacy alias for `[auxiliary.curator]`. `curator.auxiliary` accepts the same
fields as `auxiliary.curator` for Hermes parity.

### `[telegram]`

The Telegram bot adapter. See [Telegram](../telegram/) for the
intended setup path. Selected defaults:

| Field | Default | Purpose |
|---|---|---|
| `bot_token` | `""` | Bot token (routed to `.env`). |
| `allowed_chat_id` | `0` | Single allow-list chat ID. |
| `allowed_chats` | unset | List of allow-list chat IDs. |
| `allowed_user_ids` | `[]` | Allow-list user IDs. |
| `require_mention` | `false` | Require @bot mention in groups. |
| `guest_mode` | `false` | Allow unknown users (read-only). |
| `coalesce_ms` | `1000` | Edit coalescer interval (milliseconds). |
| `fresh_final_after_seconds` | `60.0` | Mark a message as final after this idle window. |
| `notifications` | `"important"` | Notification level. |
| `first_run_discovery` | `true` | Discover the home chat on first run. |
| `memory_queue_cap` | `1024` | Async memory queue capacity. |
| `extractor_batch_size` | `5` | Memory extractor batch size. |
| `extractor_poll_interval` | `10s` | Memory extractor poll interval. |
| `recall_enabled` | `true` | Enable graph recall. |
| `recall_weight_threshold` | `1.0` | Recall weight cutoff. |
| `recall_max_facts` | `10` | Max recalled facts per turn. |
| `recall_depth` | `2` | Graph traversal depth. |
| `recall_decay_horizon_days` | `180` | Linear weight decay horizon. |
| `mirror_enabled` | `true` | Mirror memory to USER.md. |
| `mirror_path` | `$GORMES_HOME/memory/USER.md` | Mirror destination. |
| `mirror_interval` | `30s` | Mirror flush interval. |
| `semantic_enabled` | `false` | Opt-in semantic fusion. |
| `semantic_endpoint` / `semantic_model` / `semantic_top_k` / `semantic_min_similarity` | `""` / `""` / `3` / `0.35` | Semantic recall tuning. |
| `embedder_poll_interval` | `30s` | Embedder poll interval. |
| `embedder_batch_size` | `10` | Embedder batch size. |
| `embedder_call_timeout` | `10s` | Embedder request timeout. |
| `query_embed_timeout` | `60ms` | Query-side embedding timeout. |

### `[discord]`

| Field | Default | Purpose |
|---|---|---|
| `token` | `""` | Bot token (routed to `.env`). |
| `allowed_channel_id` | `""` | Primary allowed channel ID. |
| `allowed_channels` | unset | List of allowed channel IDs. |
| `ignored_channels` | unset | List of channels to ignore. |
| `free_response_channels` | unset | Channels where mentions are not required. |
| `no_thread_channels` | unset | Channels where threads must not be created. |
| `require_mention` | unset | Require @bot mention (defaults to `false` when unset). |
| `auto_thread` | unset | Auto-create threads. |
| `reply_to_mode` | `"first"` | One of `first`, `all`, `off`. |
| `allow_bots` | `"none"` | One of `none`, `mentions`, `all`. |
| `server_actions` | `[]` | Server action allow-list. |
| `coalesce_ms` | `1000` | Edit coalescer interval. |
| `first_run_discovery` | `false` | Discover channels on first run. |

### `[slack]`

| Field | Default | Purpose |
|---|---|---|
| `enabled` | `false` | Master toggle for the Slack adapter. |
| `bot_token` | `""` | xoxb token (routed to `.env`). |
| `app_token` | `""` | xapp token (routed to `.env`). |
| `allowed_channel_id` | `""` | Primary allowed channel ID. There is no `default_channel_id` field. |
| `allowed_channels` | unset | List of allowed channel IDs. |
| `coalesce_ms` | `1000` | Edit coalescer interval. |
| `first_run_discovery` | `false` | Channel discovery on first run. |
| `require_mention` | `true` | Require @mention in shared channels. |
| `strict_mention` | unset | Reject indirect mentions. |
| `reply_in_thread` | `true` | Reply in thread by default. |
| `free_response_channels` | unset | Channels where mentions are not required. |

### `[teams]`

Microsoft Teams adapter. Uses `TEAMS_*` env vars rather than `GORMES_TEAMS_*`.

| Field | Default | Purpose |
|---|---|---|
| `enabled` | `false` | Master toggle. |
| `client_id` | `""` | Azure app client ID. |
| `client_secret` | `""` | Azure app secret (routed to `.env`). |
| `tenant_id` | `""` | Azure tenant ID. |
| `port` | `3978` | Local webhook port. |
| `allowed_users` | `[]` | Allow-list of user IDs. |
| `allow_all_users` | `false` | Skip the user allow-list. |

### `[yuanbao]`

Disabled-by-default adapter. `enabled=false`, `login_token=""`, `hy_source=""`,
`agent_id=""`, `allowed_conversation_id=""`, `coalesce_ms=0`,
`first_run_discovery=false`.

### `[web]`

| Field | Default | Purpose |
|---|---|---|
| `backend` | `""` | Web search backend (e.g. `tavily`, `serper`). Override with `GORMES_WEB_BACKEND`. |
| `use_gateway` | `false` | Route web requests through the gateway. Override with `GORMES_WEB_USE_GATEWAY`. |

### `[browser]`

| Field | Default | Purpose |
|---|---|---|
| `cdp_url` | `""` | CDP endpoint URL. Overridden by `GORMES_BROWSER_CDP_URL`, `BROWSER_CDP_URL`, or `CHROME_REMOTE_DEBUGGING_URL`. |

### `[security]`

| Field | Default | Purpose |
|---|---|---|
| `website_blocklist.enabled` | `false` | Enable the website blocklist. |
| `website_blocklist.domains` | `[]` | Hostname blocklist entries. |
| `website_blocklist.shared_files` | `[]` | Paths to shared blocklist files. |

### `[secrets]`

External secret-provider routing for `SecretRef` lookups.

| Field | Default | Purpose |
|---|---|---|
| `defaults.env` | `"default"` | Provider alias used by `SecretRef{source="env"}`. |
| `defaults.file` | `""` | Provider alias for file-backed refs. |
| `defaults.exec` | `""` | Provider alias for exec-backed refs. |
| `providers.<alias>.source` | unset | Provider source kind (`env`, `file`, `exec`). |
| `providers.<alias>.path` / `mode` / `allowlist` / `max_bytes` / `allow_insecure_path` | unset | Provider-specific knobs. |

### `[agents]`, `[[agents.list]]`, `[agents.defaults]`

Multi-agent registry. `agents.defaults` carries fallback workspace/model;
`[[agents.list]]` defines named agents with `id`, `name`, `workspace`,
`model`, and `default`.

| Field | Purpose |
|---|---|
| `agents.defaults.workspace` | Single default workspace path used for backward-compatible agent defaults. |
| `agents.defaults.workspaces` | Per-profile project workspace list persisted by `gormes setup profiles`. Current releases round-trip this list but do not yet enforce it as an access boundary. Planned policy: empty list means operator home; non-empty list is the project read/write allow-list, while the active profile root remains available for profile state. |
| `agents.defaults.channels` | Per-profile messaging-channel list persisted by `gormes setup profiles`; distinct from `[[bindings]]` routing and channel credentials. |
| `[[agents.list]].workspace` | Per-agent primary workspace path. This is different from Goncho's memory workspace id. |

### `[[bindings]]`

Channel-to-agent routing rules. Each entry: `agent_id` plus a `bindings.match`
table (`channel`, `account_id`, etc.).

### `[cron]`

| Field | Default | Purpose |
|---|---|---|
| `enabled` | `false` | Enable scheduled tasks. |
| `call_timeout` | `60s` | Per-cron-call timeout. |
| `mirror_interval` | `30s` | CRON.md mirror flush interval. |
| `mirror_path` | `""` (defaults to `$GORMES_HOME/cron/CRON.md` at runtime) | CRON.md mirror destination. |

### `[skills]`

| Field | Default | Purpose |
|---|---|---|
| `root` | `""` (defaults to `$GORMES_HOME/skills` at runtime) | Skills root directory. |
| `selection_cap` | `3` | Maximum skills considered per turn. |
| `max_document_bytes` | `65536` | Per-document byte ceiling. |
| `usage_log_path` | `""` (defaults to `<skills_root>/usage.jsonl`) | Append-only skill usage log. |

### `[delegation]`

| Field | Default | Purpose |
|---|---|---|
| `enabled` | `false` | Enable subagent delegation. |
| `max_depth` | `2` | Maximum recursion depth. |
| `max_concurrent_children` | `3` | Concurrent child cap. |
| `default_max_iterations` | `8` | Per-child iteration cap. |
| `default_timeout` | `45s` | Per-child timeout. |
| `run_log_path` | `""` (defaults to `$GORMES_HOME/subagents/runs.jsonl`) | Run log destination. |
| `max_waiting` | `128` | Maximum waiting children. |

### `[goncho]`

In-process Honcho-compatible memory facade.

| Field | Default | Purpose |
|---|---|---|
| `enabled` | `true` | Enable Goncho. |
| `workspace` | `goncho.DefaultWorkspaceID` | Workspace identifier. |
| `observer_peer` | `goncho.DefaultObserverPeerID` | Observer peer identifier. |
| `recent_messages` | `goncho.DefaultRecentMessages` | Recent-window message count. |
| `max_message_size` | `goncho.DefaultMaxMessageSize` | Per-message byte ceiling. |
| `max_file_size` | `goncho.DefaultMaxFileSize` | Per-file byte ceiling. |
| `get_context_max_tokens` | `goncho.DefaultGetContextMaxTokens` | Token budget for `get_context`. |
| `reasoning_enabled` | `true` | Persist reasoning blocks. |
| `peer_card_enabled` | `true` | Persist peer cards. |
| `summary_enabled` | `true` | Persist summaries. |
| `dream_enabled` | `false` | Enable dream/idle consolidation. |
| `dream_idle_timeout_minutes` | `goncho.DefaultDreamIdleTimeout` (in minutes) | Idle window before dream. |
| `deriver_workers` | `goncho.DefaultDeriverWorkers` | Deriver worker pool size. |
| `representation_batch_max_tokens` | `goncho.DefaultRepresentationBatchMaxTokens` | Representation batch budget. |
| `dialectic_default_level` | `"low"` | Default dialectic level. |

### `[updates]`

| Field | Default | Purpose |
|---|---|---|
| `pre_update_backup` | `false` | Snapshot before `gormes update`. |
| `backup_keep` | `5` | Backup retention budget. |

## Minimal example

```toml
_config_version = 1

[hermes]
provider = "openai"
model = "gpt-4o"

[input]
max_bytes = 200000
max_lines = 10000

[telegram]
allowed_user_ids = [123456789]

[goncho]
enabled = true
```

Secrets for the same example live in `~/.gormes/.env`:

```dotenv
GORMES_API_KEY=sk-...
GORMES_TELEGRAM_TOKEN=123:abc
```
