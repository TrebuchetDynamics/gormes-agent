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
| Normal turn loop: submit → provider stream → tool continuation → final | `run_agent.py` | `internal/kernel/kernel.go` `internal/llm/client.go` | partial | Kernel loop exists; Hermes has richer context/prefill assembly. |
| Tool continuation multi-round | `run_agent.py` `_process_tool_call` | `internal/kernel/kernel.go` `handleToolCall` | covered | Loop drives multi-round tool calls. |
| Default 90-turn iteration budget | `run_agent.py` `max_iterations=90` | `internal/kernel/kernel.go` | covered | Default 90, toolless summary on exhaustion. |
| Cancel active turn | `run_agent.py` `cancel()` | `internal/kernel/kernel.go` `cancelCmd` | covered | Context cancellation. |
| Interrupt and replace draft | `run_agent.py` `interrupt` | `internal/kernel/kernel.go` + `internal/tui/update.go` `HermesActionInterrupt` | covered | TUI interrupt path tested. |
| Prefill messages injection | `cli.py` `_load_prefill_messages`; `gateway/run.py` `_load_prefill_messages` | `internal/config/config.go` `LoadPrefillMessages`; `internal/kernel/kernel.go` `PrefillMessages`; `cmd/gormes/prefill.go` | covered | Loads JSON prefill messages from `agent.prefill_messages_file` or `HERMES_PREFILL_MESSAGES_FILE` / `GORMES_PREFILL_MESSAGES_FILE`, injects them after system/context messages before the current user turn, and keeps them out of visible history; covered by `internal/config/prefill_messages_test.go` and `internal/kernel/prefill_test.go`. |

### 1.2 Trajectory

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Trajectory compressor | `trajectory_compressor.py` | → `missing` | missing | Not ported. |

### 1.3 Context

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Context engine status | `agent/context_engine.py` | `internal/llm/context_engine.go` | covered | Context status with token pressure. |
| Context compression | `agent/context_compressor.py` | `internal/llm/context_compressor_engine.go`; `internal/llm/context_compressor_pruning.go`; `internal/llm/context_compressor_*_test.go` | partial | Provider-backed explicit `ContextEngine.Compress` path now summarizes middle turns through the shared LLM client boundary, preserves protected head/tail, prunes with the existing pure helper, and reports enabled status/error evidence. Remaining gap: bind the engine into operator-facing `/compress`/runtime compression surfaces without hidden normal-turn compression. |
| Manual compression feedback | `agent/manual_compression_feedback.py`; `tests/test_cli_manual_compress.py` | `internal/llm/compression/manual_feedback.go`; `internal/llm/compression/manual_feedback_test.go`; `internal/llm/manual_compression_feedback.go`; `internal/llm/manual_compression_feedback_test.go` | covered | Pure Go manual compression feedback now matches Hermes' user-facing noop/compressed headlines, comma-formatted approximate-token lines with `→`, denser-summary note, `/compress <focus>` parsing, and safe session-split evidence. |
| Token budget | `agent/context_engine.py` | `internal/kernel/` | covered | Token budget tracking. |
| Protected head/tail | `agent/context_engine.py`; `agent/context_compressor.py` `_protect_head_size` `_find_tail_cut_by_tokens` `_align_boundary_backward` | `internal/llm/context_compression_boundary.go`; `internal/llm/context_compression_boundary_test.go` | covered | Pure Go boundary planner preserves leading system prompt plus first N non-system head messages, selects a token-budget tail, keeps the latest user message in protected tail, and avoids splitting assistant tool-call/result groups before summarization. |
| Multimodal length | `agent/context_compressor.py` `_content_length_for_budget`; `_find_tail_cut_by_tokens`; `_prune_old_tool_results` | `internal/llm/context_compressor_content.go`; `internal/llm/context_compressor_pruning.go`; `internal/llm/context_compression_boundary.go`; `internal/llm/context_compressor_image_budget_test.go`; `internal/llm/context_compressor_pruning_test.go` | covered | Multimodal content-list budgeting now sums text parts once, treats `ContentParts` as the provider-visible list when present, and feeds the same helper into pruning and protected-tail boundary planning. |
| Image charge | `agent/context_compressor.py` `_IMAGE_TOKEN_ESTIMATE`; `_IMAGE_CHAR_EQUIVALENT`; `_content_length_for_budget` | `internal/llm/context_compressor_budget.go`; `internal/llm/context_compressor_content.go`; `internal/llm/context_compressor_image_budget_test.go`; `internal/llm/context_compressor_pruning_test.go` | covered | Gormes ports Hermes' flat 1600-token / 6400-char budget per `image_url`, `input_image`, or `image` part and ignores raw base64 transport length during context-compression token accounting. |
| Tool-result pruning | `agent/context_compressor.py` `_prune_old_tool_results`; `_sanitize_tool_pairs`; `_align_boundary_backward` | `internal/llm/context_compressor_pruning.go`; `internal/llm/context_compressor_pruning_test.go` | covered | Pure Go pruning pass summarizes oversized historical tool results before summarization, preserves recent protected tail results, aligns tool-call/result groups, records typed pruning/degraded evidence, and fixture-proves invalid tool-call arguments do not mutate. |
| Summary lineage | `agent/context_compressor.py` `SUMMARY_PREFIX`; `_strip_summary_prefix`; `_with_summary_prefix`; `_find_latest_context_summary`; `compress`; `tests/agent/test_context_compressor_summary_continuity.py` | `internal/llm/context_compressor_pruning.go`; `internal/llm/context_compressor_engine.go`; `internal/llm/context_compressor_pruning_test.go`; `internal/llm/context_compressor_engine_test.go` | covered | Current Hermes compaction handoff prefix is ported; legacy `[CONTEXT SUMMARY]:` and historical `resume exactly` prefixes normalize away; pure lineage planning rehydrates persisted handoffs as previous-summary state; provider-backed compression now sends exactly that previous summary plus only new resumed turns to the summarizer and replaces the old handoff with a normalized updated summary. |

### 1.4 Prompt builder

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| SOUL.md identity prompt | `agent/prompt_builder.py` | `internal/llm/context_files.go` | covered | Context files scanned and injected. |
| AGENTS.md / project context | `agent/prompt_builder.py` | `internal/llm/context_files.go` | covered | File discovery and injection. |
| USER.md / MEMORY.md durable context | `tools/memory_tool.py` | `internal/llm/durable_user_context.go` | covered | Durable context built. |
| Skill guidance injection | `skill_preprocessing.py` | `internal/kernel/kernel.go` `SkillsPrompt` | partial | Hermes ordering differs. |
| Timestamp/model/provider metadata | `run_agent.py` `:3770-3779` | `internal/llm/turn_metadata.go` | covered | Block assembly exists. |
| Platform/session context | `gateway/run.py` `BuildSessionContextPrompt` | `internal/gateway/` | covered | Session context built. |
| Developer role swap (GPT-5/Codex) | `tests/run_agent/test_provider_parity.py:TestDeveloperRoleSwap`; `tests/agent/transports/test_chat_completions.py:test_developer_role_swap` | `internal/llm/model_guidance.go`; `internal/llm/guidance/modelpolicy/model.go`; `internal/llm/openai_compatible_role_test.go`; `internal/llm/http_client.go` | covered | API-boundary tests prove OpenAI-compatible chat requests serialize the first system message as `developer` for GPT-5/Codex models, keep nonmatching models as `system`, avoid mutating internal messages, and keep Codex Responses instructions separate. |
| Tool-use enforcement guidance | `agent/prompt_builder.py` `TOOL_USE_ENFORCEMENT_MODELS`; `agent/system_prompt.py` `build_system_prompt_parts` | `internal/llm/guidance/text/constants.go`; `internal/gateway/liveprompt/liveprompt.go` `buildToolUseEnforcementBlock` | covered | Constants match upstream and live prompt injection uses substring matches for ToolUseEnforcementModels (gpt, codex, gemini, gemma, grok, glm, qwen, deepseek). |
| Memory guidance | `agent/prompt_builder.py` `MEMORY_GUIDANCE` | `internal/llm/guidance_constants.go` | covered | Byte-equivalent constant ported. |
| Skills guidance constant | `agent/prompt_builder.py` `SKILLS_GUIDANCE` | `internal/llm/guidance_constants.go` | covered | Byte-equivalent constant ported. |

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
| Chat Completions transport | `agent/transports/chat_completions.py` | `internal/llm/http_client.go` | covered | Transport request building and fixture replay. |
| Anthropic Messages transport | `agent/anthropic_adapter.py` | `internal/llm/` | covered | Adapter shipped. |
| Bedrock Converse transport | `agent/bedrock_adapter.py` | `internal/llm/` | partial | Stream events; SigV4 pending. |
| Codex Responses transport | `agent/codex_responses_adapter.py`; `agent/transports/codex.py` | `internal/llm/` | covered | Responses conversion shipped, including stable session `prompt_cache_key` body routing and ChatGPT Codex backend cache-scope headers. |
| Gemini transport | `agent/gemini_native_adapter.py` | → `missing` | missing | Not ported. |
| Google Code Assist | `agent/gemini_cloudcode_adapter.py`; `agent/google_code_assist.py` | `internal/llm/gemini_cloudcode.go`; `internal/llm/gemini_cloudcode_test.go`; `internal/llm/google_code_assist.go`; `internal/llm/google_code_assist_test.go`; `internal/llm/googlecodeassist/` | covered | Gemini Cloud Code request/stream mapper plus Google Code Assist project/quota resolver are complete with fake token/HTTP fixtures for headers, project precedence, onboarding, quota parsing, stream/tool normalization, and safe Google error classification. Browser OAuth/live Google credentials remain outside these pure provider fixtures. |
| OpenRouter | `tools/openrouter_client.py` | `internal/llm/openrouter_compatible.go`; `internal/llm/http_client.go`; `internal/llm/openrouter_compatible_test.go` | covered | OpenRouter runtime resolution, OpenAI-compatible transport, attribution headers (`HTTP-Referer`, `X-OpenRouter-Title`, categories), OpenRouter-base custom routes, Grok prompt-cache affinity, model metadata/pricing, Pareto extra body, and safe error classification are fixture-covered without live credentials. |

### 2.2 Provider registry

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Provider ID and alias manifest | `hermes_cli/providers.py` `HERMES_OVERLAYS` | `internal/llm/provider_registry_manifest.go` | covered | Manifest with all Hermes provider IDs and aliases. |
| Model metadata and pricing | `agent/model_metadata.py`; `agent/models_dev.py`; `agent/usage_pricing.py` | `internal/llm/model_registry.go`; `internal/llm/model_registry_test.go`; `internal/llm/model_context_resolver_test.go`; `internal/llm/routing/modelcatalog/`; `internal/llm/modelcatalog/`; `internal/llm/openrouter_compatible_test.go` | covered | Static registry/context resolver fixtures expose provider family, context windows, max output, pricing, capabilities, explicit unknown states, provider-enforced caps, OpenRouter pricing, and models.dev/catalog cache/merge behavior without live network. |

### 2.3 Auth

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Credential pool | `agent/credential_pool.py` | `internal/config/` + `internal/cli/` | partial | Gormes has credential command surface; pool semantics differ. |
| OAuth device code | `hermes_cli/auth.py` `_login_openai_codex` | `cmd/gormes/auth.go` | partial | Codex OAuth path exists but paused/Hermes drift unclear. |
| Credential file token vault | `tools/credential_files.py`; `tools/path_security.py`; `agent/credential_sources.py` source-removal registry | `internal/config/token_vault.go` | covered | Gormes row 4.G covers safe relative credential-file resolution, unsafe-path rejection, dedupe, clear semantics, and redacted evidence. |
| Bitwarden Secrets Manager source | `agent/secret_sources/bitwarden.py`; `hermes_cli/env_loader.py` `_apply_external_secret_sources`; `hermes_cli/secrets_cli.py` | `internal/config/externalsecrets/bitwarden.go`; `internal/config/config.go`; `internal/app/secrets/`; `internal/platform/cli/gormescli/secrets.go` | partial | Gormes loads `[secrets.bitwarden]` during config startup, invokes `bws secret list`, injects env vars before env config resolution, labels applied keys as Bitwarden, preserves the bootstrap token, degrades without blocking startup, exposes CLI `secrets bitwarden` status/sync/disable/install/setup, ships managed pinned `bws` install/checksum verification, and has setup token/env/config/project-selection/test-fetch redaction tests. Remaining Hermes parity: disk cache (planned row `Bitwarden disk cache parity`) and credential-pool borrowed-source persistence. |
| Auth commands (add/list/remove/reset/status/logout/spotify) | `hermes_cli/auth_commands.py` | `cmd/gormes/auth.go` | partial | Most commands exist; Spotify and top-level logout planned. |
| Secret ref validation | `hermes_cli/config.py` | `internal/provider/profile_provider_config.go` | covered | SecretRef env resolution and missing-ref evidence. |

### 2.4 Retry and rate limits

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Retry budget | `agent/retry_utils.py` | `internal/kernel/` `NewRetryBudget` | covered | Retry budget with backoff. |
| Rate limit tracker | `agent/rate_limit_tracker.py` | `internal/llm/rate_limit_tracker.go`; `internal/llm/rate_limit_tracker_test.go`; `internal/llm/http_client.go`; `internal/llm/status.go` | covered | Nous/OpenRouter/OpenAI-compatible `x-ratelimit-*` headers parse into request/token minute/hour buckets with captured-at freshness, elapsed reset estimates, Hermes-shaped full/compact display helpers, malformed/no-header degraded behavior, and HTTP provider status capture without live credentials. |
| Prompt cache | `agent/prompt_caching.py`; `tests/agent/test_prompt_caching.py` | `internal/llm/prompt_cache_policy.go`; `internal/llm/prompt_cache_policy_test.go`; `internal/llm/http_client.go`; `internal/llm/status.go`; `internal/llm/anthropic_client.go`; `internal/llm/provider_status_test.go` | covered | Hermes `system_and_3` prompt-cache policy is ported with native Anthropic, OpenRouter Claude, third-party Anthropic-compatible, MiniMax, Qwen/opencode/Alibaba envelope, unsupported-provider stripping, deep-copy/no-mutation, 1h TTL, four-breakpoint, request serialization, and visible provider-status evidence fixtures. |
| Account usage reporting | `agent/account_usage.py` | `internal/llm/` | partial | Provider account usage read model exists; renderer for Codex/Anthropic/OpenRouter. |

### 2.5 Error classification

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Error classifier | `agent/error_classifier.py` | `internal/llm/` | partial | Basic error mapping; Hermes has richer provider-specific classes. |

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
| `uninstall` subcommand | `hermes_cli/main.py:15469` + `hermes_cli/uninstall.py` | `internal/app/gormescmd/main.go`; `internal/app/uninstall/service.go`; `internal/platform/cli/gormescli/uninstall_dryrun_test.go`; `internal/platform/cli/gormescli/uninstall_legacy_xdg_test.go` | covered | Top-level `gormes uninstall` is wired and fixture-covered for dry-run-by-default previews, `--yes` destructive execution, JSON reports, keep-config/credential filters, managed-home artifacts, legacy XDG cleanup, and install.sh-published symlink cleanup. Gormes intentionally diverges from Hermes GUI-only flags because Gormes has no Electron desktop GUI surface. |
| `migrate hermes` | `N/A` | `cmd/gormes/migrate.go` | covered | Hermes config/session migration. |
| `migrate openclaw` | `hermes_cli/claw.py` | `internal/platform/migrate/openclaw/` | covered | OpenClaw migration shipped. |

### 3.3 Slash commands (gateway + TUI)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| `/help` | `hermes_cli/commands.py` | `internal/tui/slash_help.go` + `internal/gateway/commands.go` | covered | Both TUI and gateway. |
| `/new` / `/reset` | `hermes_cli/commands.py` | `internal/tui/slash_new.go` + `internal/gateway/` | covered | Session reset. |
| `/stop` | `hermes_cli/commands.py` | `internal/tui/slash_stop.go` + `internal/gateway/` | covered | Cancel active turn. |
| `/status` | `gateway/run.py` `_handle_status_command`; `locales/en.yaml` `gateway.status.agent_running`, `gateway.status.tokens`, `gateway.status.queued` | `internal/gateway/status_command.go`; `internal/gateway/status_command_test.go` | covered | Gateway `/status` renders the Hermes field order and labels with Markdown bold, active-agent marker `Yes ⚡`, `Cumulative API tokens (re-sent each call)` with comma-formatted totals, queued follow-up depth, connected platforms, title/session metadata, and reply-thread/provider-bypass behavior; Gormes-owned route/Kanban status sections remain additive. |
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
| `/tools` | `hermes_cli/commands.py` | `internal/tui/slash_tools.go`; `cmd/gormes/hermes_rowbacked_commands.go`; `cmd/gormes/tools_command_test.go` | partial | TUI `/tools enable|disable` is config-backed and the root `gormes tools list|enable|disable` command now persists CLI `platform_toolsets` instead of returning row-backed unavailable evidence; gateway/channel command registry still marks `/tools` unavailable. |
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
| Duplicate collapse | `agent/display.py` | `internal/tui/panels/panels.go`; `internal/tui/panels/hermes_compat_test.go` | covered | `new` mode suppresses consecutive duplicate tool scrollback entries while preserving distinct tools and `all` mode history. |
| Tool preview truncation | `agent/display.py` `build_tool_preview` | `internal/tooltrace/` | covered | Truncation of long tool args. |
| `(×N)` collapse | `agent/display.py` | `internal/tooltrace/` | covered | Identical consecutive tool calls collapsed. |
| `todo merge=true` wording | `agent/display.py` | `internal/tooltrace/` | covered | Special todo merge display. |
| Unknown-tool degradation | `agent/display.py` | `internal/tooltrace/` | covered | Unknown tools display as generic ⚡. |
| Tool result error display | `gateway/run.py` `:14716` | `internal/gateway/render.go` | covered | Error tool results rendered distinctly. |
| Tool progress override per-platform | `gateway/run.py` `display.platforms.<name>.tool_progress` | `internal/gateway/manager.go` `toolProgressMode`; `internal/config/hermes_display_writer.go`; `internal/config/integration/load/config_test.go`; `internal/gateway/manager_test.go` | covered | Named-platform and base-platform overrides take precedence over global mode; `/verbose` persists native `display.platforms.<platform>.tool_progress` with config round-trip tests. |
| Tool progress env var fallback | `hermes_cli/config.py` `HERMES_TOOL_PROGRESS`, `HERMES_TOOL_PROGRESS_MODE`; `gateway/run.py` env fallback when config absent | `internal/config/config.go`; `internal/config/tool_progress_env_test.go` | covered | Deprecated Hermes env vars are honored only when `display.tool_progress` is not configured: `HERMES_TOOL_PROGRESS=false` maps to `off`, `HERMES_TOOL_PROGRESS=true` maps to `all`, and `HERMES_TOOL_PROGRESS_MODE` maps to normalized mode. Config file values win. |

### 4.3 Composer behavior

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Enter submits | `ui-tui/src/app/useInputHandlers.ts` | `internal/tui/update.go` | covered | Enter dispatches. |
| Alt+Enter inserts newline | `ui-tui/src/app/useInputHandlers.ts` | `internal/tui/update.go` | covered | Alt+Enter handled. |
| Ctrl+C cancels/force-quits | `ui-tui/src/app/useInputHandlers.ts` | `internal/tui/update.go` | covered | Modal cancel → turn cancel → force quit. |
| Ctrl+D deletes char / exits | `ui-tui/src/app/useInputHandlers.ts` | `internal/tui/update.go` | covered | Delete char when draft non-empty. |
| Ctrl+L repaints | `ui-tui/src/app/useInputHandlers.ts` | `internal/tui/update.go` | covered | Force redraw. |
| Paste/image handling | `ui-tui/src/app/useInputHandlers.ts` | `internal/tui/composer_ingress.go` | covered | Clipboard paste and image attachment. |
| Voice recording key | `ui-tui/src/app/useInputHandlers.ts`; `hermes_cli/voice.py` | `internal/tui/update.go`; `internal/tui/model.go`; `internal/tui/hermes_keybindings_test.go`; `internal/tui/hermes_voice_runtime_test.go`; `cmd/gormes/tui_voice_slash.go` | covered | Configurable `voice.record_key` preserves the status-only `/voice` adapter when no runtime is injected and, with fake recorder/STT/TTS seams, starts capture without submit, stops into transcript composer insertion, surfaces recorder/STT/TTS evidence, and invokes assistant playback once per idle frame without live audio hardware. |

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
| Session mapping | `gateway/session.py` | `internal/gateway/` + `internal/persistence/session/` | covered | Session ID mapping. |
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
| Mattermost | `gateway/platforms/mattermost.py`; plugin config bridge | `internal/adapters/channels/mattermost/{seam,bootstrap}.go`; `internal/adapters/channels/threadtext` | partial | Implemented fakeable Mattermost seam: posted-event normalization, self/system/duplicate drops, mention/allowed-channel gating, thread reply modes, processing hooks, config/auth bootstrap evidence, sanitized REST helpers, reconnect policy, upload/edit/send request shaping. Remaining gaps: live gateway registration/operation and full plugin lifecycle parity. |
| Google Chat | `plugins/platforms/google_chat/`; `gateway/config.py` Google Chat bridge | `internal/adapters/channels/googlechat/{runtime,standalone}.go` | partial | Implemented fakeable Google Chat channel seam: platform metadata, Pub/Sub event normalization, send/send-thread adapter contract, markdown/platform hint, no-transport degraded errors, and standalone text delivery request shaping with token/poster interfaces. Remaining gaps: live Pub/Sub/OAuth setup and attachment/rich-card delivery. |
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
| `memory` tool (Hermes-compatible) | `tools/memory_tool.py` | `internal/tools/memory/tool.go`; `internal/tools/memory/tool_test.go`; `cmd/gormes/registry.go`; `cmd/gormes/registry_test.go` | covered | Default registry includes a Hermes-compatible `memory` tool with add/replace/remove/read actions, `user` and `memory` targets, Hermes delimiter parsing/writing, file locking, char-limit evidence, and prompt-injection/secret rejection tests. |
| `session_search` tool | `tools/session_search_tool.py` | `internal/tools/sessionsearch/` | covered | Session search. |

### 6.3 Web tools

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| `web_search` / `web_extract` / `web_crawl` | `tools/web_tools.py`; `tools/x_search_tool.py` degraded citation handling | `internal/tools/web_tools.go`; `internal/tools/web_tools_test.go` | partial | Native web tools cover multiple backends, mark Perplexity search answers without citations as degraded with backend/source provenance, return typed unavailable evidence when `goscrapling_crawler` is selected without an injected adapter, and now fixture-prove an explicit goscrapling crawler seam with normalized result/evidence output, policy/private/secret URL gates, duplicate/offsite accounting, max-page stats, and degraded runtime errors. Remaining gaps: dedicated root web research environment, full dependency-gated goscrapling crawler runtime binding to public robots/cache/checkpoint/session-adapter APIs, and provider-specific source/citation contracts across every backend. |

### 6.4 Browser tools

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Browser action contract | `tools/browser_tool.py` | `internal/tools/browser_contract.go` | covered | Action/result schema. |
| Browser snapshots | `tools/browser_tool.py` | `internal/tools/browser_harness_tools.go` | partial | Backend planned. |
| Screenshot artifacts | `tools/browser_tool.py` | `internal/tools/browser_contract.go` | covered | Envelope fields. |
| SSRF guard | `tools/browser_tool.py` | `internal/tools/browser_ssrf_guard.go` | covered | Private URL guard. |

### 6.5 TTS / voice

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| TTS tool | `tools/tts_tool.py` | `internal/tools/tts/tool.go`; `internal/tools/tts/go_native_provider.go`; `internal/speech/tts/fixture.go` | partial | Tool descriptor, result envelope, command/cloud provider seam, and Go-owned local fixture WAV provider are ported; remaining parity gaps are provider-specific Hermes built-ins beyond shipped adapters. |
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
| Cron scheduler | `cron/scheduler.py` | `internal/automation/cron/` | covered | Schedule parser and execution. |
| Cron job definitions | `cron/jobs.py` | `internal/automation/cron/` | covered | Job store with CRUD. |
| Cron tool | `tools/cronjob_tools.py` | `internal/tools/` | covered | Cron management tool. |
| Schedule parser | `cron/jobs.py` `parse_schedule` | `internal/automation/cron/` | covered | Natural language schedule parsing. |
| Compute next run | `cron/jobs.py` `compute_next_run` | `internal/automation/cron/` | covered | Next execution time calculation. |
| Grace seconds | `cron/jobs.py` `_compute_grace_seconds` | `internal/automation/cron/schedule_parser.go`; `internal/automation/cron/schedule_parser_test.go` | covered | `CronNextRunDecision` covers late one-shot grace windows, finite repeat exhaustion, and recurring fast-forward behavior. |
| Delivery target resolution | `cron/scheduler.py` `_resolve_delivery_targets` | `internal/automation/cron/delivery_plan.go`; `internal/gateway/delivery.go` | covered | Explicit platform/chat/thread targets, origin delivery, `all` routing intent, home-channel expansion, dedupe, invalid-target evidence, and directory-missing evidence are fixture-covered. |
| Multi-target delivery | `cron/scheduler.py` `_deliver_result` | `internal/automation/cron/delivery_plan.go`; `internal/automation/cron/delivery_plan_test.go` | covered | Delivery plans fan out to multiple targets, prefer live adapters, fall back to standalone senders and the legacy delivery sink with per-target evidence. |
| Script execution | `cron/scheduler.py` `_run_job_script` | `internal/automation/cron/` | covered | Run shell scripts as job actions. |
| Context_from chaining | `cron/jobs.py` | `internal/automation/cron/context_from.go`; `internal/automation/cron/context_from_test.go` | covered | Previous completed cron outputs are injected before the prompt, capped per source, and missing/invalid/unreadable sources are skipped with evidence. |
| Resource release | `cron/jobs.py`; `cron/scheduler.py` cleanup path | `internal/automation/cron/run_release.go`; `internal/automation/cron/release_binding_test.go` | covered | Per-run release ledger closes session DBs, HTTP idle connections, and subprocess resources at run end, including kernel-error/cancel paths and idempotent no-resource evidence. |
| Job lock files | `cron/scheduler.py` `_get_lock_paths`; `cron/scheduler.py` `tick` file-lock path | `internal/automation/cron/scheduler.go`; `internal/automation/cron/scheduler_lock_unix.go`; `internal/automation/cron/scheduler_lock_windows.go`; `internal/automation/cron/scheduler_test.go` | covered | Scheduler `Start` resolves `<GORMES_HOME>/cron/.tick.lock`, acquires a nonblocking cross-process tick lock before running due jobs, skips overlapping schedulers, serializes same-scheduler schedule groups, releases after job fanout and MCP orphan cleanup, and fixture-proves overlapping schedulers do not double-run the same tick. |
| Cron prompt guard | `cron/scheduler.py` `CronPromptInjectionBlocked`; `tools/cronjob_tools.py` `_scan_cron_prompt` | `internal/automation/cron/prompt_script_safety.go`; `internal/automation/cron/executor.go`; `internal/automation/cron/*safety*_test.go`; `internal/automation/cron/executor_test.go` | covered | Cron create/update preflight scans prompts/scripts for critical injection and exfiltration patterns before persistence, and runtime assembled prompts are rescanned before kernel submit so unsafe prompt/context/script content records a blocked run, delivers a clean non-leaking failure notice, and proves the agent runner is not invoked. |
| Recovery from missed schedule | `cron/jobs.py` `_recoverable_oneshot_run_at` | `internal/automation/cron/schedule_parser.go`; `internal/automation/cron/run_completion.go` | covered | Missed one-shot recovery and terminal one-shot completion are fixture-covered by schedule decision and run-completion tests. |

---

## 8. Skills

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Skill metadata (SKILL.md) | `skills/` | `internal/skills/` + `development-skills/` | covered | Portable SKILL.md metadata. |
| Skill install/list/inspect | `hermes_cli/skills_hub.py` | `cmd/gormes/skills.go` | covered | Skill CLI commands. |
| Skill slash commands | `agent/skill_commands.py` | `internal/tui/slash_skills.go` | covered | Dynamic slash command registration. |
| Skill prompt snapshot | `skill_preprocessing.py` | `internal/kernel/kernel.go` | partial | Skill blocks injected; ordering differs. |
| Claude Design HTML artifact skill | `skills/creative/claude-design/SKILL.md` | → `missing` | missing | Hermes bundles a CLI/API artifact workflow that requires a complete local HTML file, exact on-disk path, file-existence/static verification, and no hosted `/projects/<id>` or `window.claude.complete()` publish assumptions. Gormes has no equivalent built-in skill/guard, so generated HTML artifacts can drift between requested repo paths, local artifact paths, and publish claims. |
| Skill sync | `tools/skills_sync.py` | → `missing` | missing | Not ported. |

---

## 9. MCP / Plugins

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| MCP server | `mcp_serve.py` | `internal/mcpserver/` | covered | MCP stdio/HTTP server. |
| MCP tool | `tools/mcp_tool.py` | `internal/tools/mcp/boundary` | partial | Tool registration/audit has redacted `mcp_host_unavailable` evidence for list/discovery failures, invoke transport errors, and unavailable tool results, generic host auth-required outcomes now carry a typed `AuthRequired` audit flag with redacted `mcp_auth_required` evidence, and managed MCP gateway discovery/tool calls now preflight `initialize`, capture `Mcp-Session-Id`, map missing-session negotiation failures to typed `auth_required` evidence, and reinitialize/retry once when a streamable HTTP transport session expires during `tools/list` or `tools/call` (`internal/tools/mcp/boundary/host.go`, `internal/tools/mcp/boundary/host_test.go`, `internal/tools/managed_tool_gateway_test.go`). Remaining gaps: full OAuth/session lifecycle parity outside the managed gateway bridge. |
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
| Session lifecycle | `src/routers/sessions.py` | `internal/goncho/` + `internal/persistence/session/` | covered | Local session crud. |
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
| Auto title generation | `agent/title_generator.py` | `internal/persistence/session/auto_title.go` | partial | Helper exists; gateway wiring not proven. |
| Session naming from user prompt | `run_agent.py` `auto_title` | → `missing` | missing | Not wired in production. |
| @ context reference parser | `agent/context_references.py` | `internal/llm/` + `internal/contextrefs/` | covered | Stable parser shipped; file/folder/URL injection row-backed. |
| Subdirectory/project hints | `agent/subdirectory_hints.py` `agent/tool_executor.py` | `internal/llm/contextfiles/subdirhints/` `internal/kernel/toolexec.go` | covered | Tracker is active for tool-capable kernels; tool-call path args lazily append discovered AGENTS.md/CLAUDE.md/.cursorrules hints to text and multimodal tool results with duplicate suppression. |
| Background review fork | `run_agent.py` background review | → `missing` | missing | Not ported. |
| Curator state machine | `agent/curator.py` | → `missing` | missing | Not ported. |
| Curator CLI | `hermes_cli/curator.py` | → `missing` | missing | Not ported. |
| Memory prefetch/sync | `agent/memory_manager.py` | → `missing` | missing | Not ported. |
| Pre-compress hook | `agent/memory_manager.py` | → `missing` | missing | Not ported. |
| Ephemeral prefill messages | `cli.py` `_load_prefill_messages` | → `missing` | missing | Not ported. |
| Moonshot/Kimi schema sanitizer | `agent/moonshot_schema.py` | `internal/llm/moonshot_schema.go` | covered | Tool-parameter sanitizer shipped. |

---

## 15. Provider — Deep (Bedrock, Gemini, OAuth)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Google Gemini native | `agent/gemini_native_adapter.py` | → `missing` | missing | Not ported. |
| Google Cloud Code adapter | `agent/gemini_cloudcode_adapter.py` | → `missing` | missing | Not ported. |
| Google Gemini OAuth | `agent/google_oauth.py` | → `missing` | missing | Not ported. |
| Google Code Assist | `agent/google_code_assist.py` | → `missing` | missing | Not ported. |
| Bedrock stream events | `agent/bedrock_adapter.py` | `internal/llm/` | partial | Stream events partial; SigV4 pending. |
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
| Subagent delegate tool | `tools/delegate_tool.py` | `internal/core/subagent/` + `internal/tools/` | covered | Deterministic delegate runtime. |
| Approval tool | `tools/approval.py` | `internal/tools/approval/` | covered | Approval mode guards. |
| Path security tool | `tools/path_security.py` | `internal/tools/safety/` | covered | Workspace path guard. |
| URL safety tool | `tools/url_safety.py` | `internal/tools/url_safety.go`; `internal/tools/safety/urlsafety/safety.go`; `internal/tools/web_tools.go` | covered | Implemented: default URL safety policy, static checker, SSRF/private/cloud-metadata blocking, allow/block rules, cache invalidation, env-controlled private-URL policy, and web-tool prefetch blocking are covered by `internal/tools/safety/urlsafety` and `internal/tools` tests. |
| Website policy tool | `tools/website_policy.py` | `internal/tools/web_tools.go` `WebWebsitePolicy`; `internal/tools/web_tools_test.go` | covered | Implemented for web tools: Hermes-style domain blocklist policy is applied before web_extract/web_crawl fetches, returns `website_policy_blocked` evidence, and is fixture-covered for Firecrawl/Tavily/goscrapling paths. |
| OSV supply-chain check | `tools/osv_check.py` | `internal/tools/mcp/osv_malware_check.go`; `internal/tools/mcp/osv_malware_check_test.go`; `internal/tools/mcp_client.go`; `internal/tools/mcp_osv_test.go`; `internal/tools/mcp_stdio.go`; `internal/tools/mcp_stdio_test.go` | covered | Ported Hermes MCP package malware check: npx/uvx/pipx package launches infer npm/PyPI package/version payloads, block only OSV `MAL-*` advisories, ignore ordinary CVEs, redact advisory summaries, and fail open on unknown commands, missing args, and OSV/network errors through an injectable client with no live network in tests. |
| Todo tool | `tools/todo_tool.py` | `internal/tools/` | covered | Todo state management. |
| Clarify tool | `tools/clarify_tool.py` | `internal/tools/` | covered | Clarify prompts. |
| Send-message tool | `tools/send_message_tool.py` | `internal/tools/sendmessage/send_message.go`; completed row `Hermes send_message tool list and target contract` | partial | Implemented the hermetic Hermes list/send contract: schema exposes optional `action` enum (`send`, `list`), injected directory listing/resolution returns deterministic targets or typed unavailable evidence, `send` validates target/message, parses `platform[:chat[:thread]]` through shared gateway routing, and fails closed when no sender is configured. Remaining gaps: live platform adapters, home-channel config binding, media delivery, and gateway session mirroring. |
| Debug helpers tool | `tools/debug_helpers.py` | `internal/tools/debuglog/session.go`; `internal/tools/debuglog/session_test.go`; `internal/tools/debug_helpers.go` | covered | Existing Go debug session helper matches Hermes' optional per-tool debug session seam with env-gated no-op behavior, session info, JSON log persistence, total-call accounting, and stronger secret/content redaction; no builder row needed. |
| Interrupt tool | `tools/interrupt.py` | `internal/kernel/` | covered | Turn cancellation via context. |
| Code execution tool | `tools/code_execution_tool.py` | `internal/tools/` + `internal/cmdrunner/` | partial | Guarded local execution; process registry not proven. |
| Background process tool | `tools/code_execution_tool.py` background | `internal/tools/` | partial | Background mode not proven. |
| File operations checkpoint | `tools/checkpoint_manager.py` | `internal/tools/checkpoint_manager.go`; `internal/tools/checkpoint/manager.go`; `internal/tools/checkpoint/manager_test.go` | partial | Checkpoint store status, startup GC, orphan/stale shadow cleanup, prune/dry-run, clear, legacy clear, and redacted evidence are implemented; pre-operation snapshot creation/restore integration remains separate checkpoint CLI/TUI parity. |
| Image generation tool | `tools/image_generation_tool.py` | `internal/tools/imagegen/generation.go`; `internal/tools/imagegen/provider.go`; `internal/tools/imagegen/managed_gateway_provider.go`; `internal/tools/imagegen/*_test.go`; `internal/tools/image_generation_provider.go` | partial | Go image generation tool exists with schema, runner/provider registry, plugin discovery refresh, configured-provider routing, managed-gateway provider binding, artifact envelope writing, aspect-ratio/model normalization, disabled/unavailable/provider-error evidence, timeout handling, and prompt/secret redaction tests; full Hermes provider matrix/live gateway lifecycle parity remains unproven. |
| Image routing by model | `agent/image_routing.py` | `internal/llm/image_routing.go`; `internal/llm/image_routing_test.go`; `internal/gateway/turn_adapter.go`; `internal/gateway/manager_image_mode_test.go` | covered | Gormes ports Hermes auto/native/text image routing by model vision capability and auxiliary vision override, native content-part assembly for local files and remote image URLs, image-reference extraction that skips code spans, local-file validation/deduplication, URL deduplication, and byte-sniffed image MIME data URLs. |
| Sandbox: Docker | `tools/environments/docker.py` | `internal/tools/environment_docker.go`; `internal/tools/docker/exec.go`; `internal/tools/docker/*_test.go`; `internal/tools/environment_test.go` | partial | Docker environment and exec seams cover config parsing, image selection, mount policy, env passthrough, stdout/stderr capture, timeout cleanup, reusable container keys, and unavailable evidence; full Hermes sandbox lifecycle parity and live integration remain unproven. |
| Sandbox: Modal | `tools/environments/modal.py` | → `missing` | missing | Not ported. |
| Sandbox: SSH | `tools/environments/ssh.py` | `internal/tools/environment_ssh.go`; `internal/tools/environment_test.go` | partial | SSH environment seam covers config parsing, path mapping, upload/download/execute/cleanup evidence, control socket shape, remote home detection, and bulk tar upload; live credential/session parity and full Hermes lifecycle remain unproven. |
| Sandbox: Singularity | `tools/environments/singularity.py` | `internal/tools/singularity_env.go`; `internal/tools/singularity_env_test.go` | partial | Apptainer/Singularity executable resolution, version preflight, hardened instance start planning, overlay binding, exec, login shell, cleanup planning, timeout/error evidence, and redacted bounded output are fixture-covered; live runtime execution lifecycle remains unproven. |
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
| Model tools (model selectors) | `model_tools.py` | `internal/cli/` + `internal/llm/` | partial | Model routing; Hermes model_tools abstractions not fully ported. |
| Toolset distributions | `toolset_distributions.py` | `internal/tools/parity/toolset_distributions.go`; `internal/platform/cli/toolsets/distribution.go` | covered | Ported the 17-name Hermes distribution manifest plus deterministic sampler with injected RNG, existing toolset-catalog validation, unknown-distribution error evidence, invalid-toolset skip evidence, and highest-probability valid fallback; batch/datagen runner integration remains out of scope. |

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
| Config migrate | `cli.py` | `internal/platform/migrate/hermes/` | covered | Hermes → Gormes config migration. |
| Config env-path | `cli.py` | `cmd/gormes/config.go` | covered | `gormes config env-path`. |
| cli-config.yaml.example (51KB schema) | `cli-config.yaml.example` | → `missing` | missing | Not mirrored as canonical schema. |
| Secrets (.env) loading | `hermes_cli/env_loader.py` | `internal/config/` | covered | Dotenv loading with secret-ref validation. |
| Config validation | `hermes_cli/config.py` `validate_config_structure` | `internal/config/configcheck/check.go`; `internal/config/configcheck/check_test.go` | partial | Read-only `gormes config check` now ports Hermes-style structural diagnostics for misplaced provider-like root keys and unknown top-level sections without echoing secret values or mutating config files. Remaining gaps: fuller Hermes config.yaml schema compatibility for custom_providers/fallback_model shapes and startup warning rendering. |

---

## 20. Session Management

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Session ID generation (Hermes-style) | `gateway/session.py` | `internal/gateway/` | covered | Session ID with Hermes-style format. |
| Session title auto-generation | `agent/title_generator.py` | `internal/persistence/session/auto_title.go` | partial | Helper unwired in gateway. |
| Session title persistence | `gateway/session.py` | `internal/persistence/session/` | partial | Metadata persisted. |
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
| Voice mode (TTS/STT toggle) | `tools/voice_mode.py` | `internal/tools/voice_mode.go`; `internal/tools/voice_mode_test.go`; `cmd/gormes/tui_voice_slash.go`; `internal/tui/hermes_voice_runtime_test.go` | partial | Tool/runner seam persists per-chat `off`/`voice_only`/`all`, exposes requirement checks, gates playback by mode, and native TUI orchestration now exercises fake recorder/STT/TTS providers. Remaining gap is real-device/provider binding and Hermes-equivalent cancellation/silence lifecycle outside fakeable TUI orchestration. |
| Voice recording | `hermes_cli/voice.py` | `internal/tools/voice_mode.go`; `internal/tools/voice_mode_test.go`; `internal/tui/update.go`; `internal/tui/model.go`; `internal/tui/hermes_keybindings_test.go`; `internal/tui/hermes_voice_runtime_test.go` | partial | Recording/transcription seams and native TUI push-to-talk orchestration are fixture-covered with typed unavailable evidence, transcript insertion, and TTS playback hooks. Remaining gap is real microphone/silence-detection/device lifecycle parity, not the native TUI record-key orchestration. |
| PTY bridge (terminal emulation) | `hermes_cli/pty_bridge.py` | `internal/platform/cli/pty/`; `internal/platform/cli/pty/bridge/`; `internal/platform/cli/pty/pty_bridge_test.go` | partial | Go PTY adapter now covers platform availability, spawn normalization, byte-safe read/write, resize, close cleanup, PID/aliveness, typed unavailable/invalid-message errors, and dashboard sidecar event isolation. Remaining gaps: actual dashboard/websocket `/api/pty` binding and terminal process registry integration. |
| Push-to-talk keybinding | `cli.py` voice.record_key | `internal/tui/` `voiceRecordKey` | covered | Configurable voice key in TUI. |
| TTS result envelope | `tools/tts_tool.py` | `internal/tools/tts/tool.go`; `internal/tools/tts/tool_test.go`; `internal/tools/tts/go_native_provider_test.go` | covered | Returns success/file_path/MEDIA evidence, voice-compatible audio tags, and typed failure evidence. |
| WASI Whisper STT | `tools/transcription_tools.py` | `internal/tools/whisper/` | covered | Local STT via WASM. |
| Go-owned local TTS backend | N/A (Gormes-owned) | `internal/speech/tts/fixture.go`; `internal/tools/tts/go_native_provider.go` | owned | Pure-Go local fixture/formant WAV provider behind TTSProvider seam; deliberately not neural Piper parity. |
| Voice mode state machine | `tools/voice_mode.py` | `internal/tools/voice_mode.go`; `internal/tools/voice_mode_test.go`; `internal/tui/hermes_voice_runtime_test.go` | partial | Chat-level mode state, requirement checks, record/transcribe, playback, provider failure transitions, and native TUI recording/processing flag resets are fixture-covered. Remaining gap is full real-device cancellation/silence lifecycle parity. |
| TTS provider abstraction | `tools/tts_tool.py`; `agent/tts_provider.py` | `internal/tools/tts/tool.go`; `internal/tools/tts/command_provider.go`; `internal/tools/tts/go_native_provider.go` | covered | Cloud/command/local provider seam with explicit provider selection, no built-in shadowing, and Go-owned local runtime adapter. |

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
| Kanban link/unlink | `tools/kanban_tool.py`; `hermes_cli/kanban.py` | `internal/tools/kanban/kanban_tools.go`; `internal/tools/kanban/kanban_tools_test.go`; `cmd/gormes/kanban.go`; `cmd/gormes/hermes_cli_parity.go` | partial | `kanban_link` and CLI `kanban link`/`unlink` surfaces exist with dependency-aware readiness recomputation; exact Hermes model-tool unlink coverage is not proven. |
| Kanban comment | `tools/kanban_tool.py` | `internal/tools/kanban/kanban_tools.go`; `internal/tools/kanban/kanban_tools_test.go` | covered | `kanban_comment` stores durable task comments, rejects caller-supplied author overrides, supports scoped and cross-task handoff comments, and exposes comments through `kanban_show`. |
| Kanban heartbeat/reclaim/zombie | `tools/kanban_tool.py`; `hermes_cli/kanban.py` | `internal/tools/kanban/kanban_tools.go`; `internal/tools/kanban/kanban_tools_test.go`; `cmd/gormes/kanban.go`; `cmd/gormes/kanban_command_test.go` | partial | Worker `kanban_heartbeat` and CLI heartbeat/GC evidence exist; Hermes-style reclaim/zombie worker takeover semantics are not proven. |
| Kanban init | `tools/kanban_tool.py` | `internal/tools/kanban/` | covered | Board initialization. |
| Kanban dispatch to subagent | `tools/kanban_tool.py` | `internal/tools/kanban/` | covered | Dispatch route. |
| Kanban archiving | `tools/kanban_tool.py`; `hermes_cli/kanban.py` | `cmd/gormes/kanban.go`; `cmd/gormes/kanban_command_test.go`; `internal/tools/kanban/kanban_tools_test.go` | partial | CLI archive and archived-task list filtering are implemented with tests; model-tool archive parity is not exposed as a normal Kanban tool. |
| Kanban link/tail diagnostics | `hermes_cli/kanban_diagnostics.py` | `cmd/gormes/kanban_tail_test.go`; `cmd/gormes/kanban_log_test.go`; `cmd/gormes/kanban.go` | partial | Tail/log diagnostics stream task events and print bounded logs; full Hermes diagnostic command parity remains unproven. |
| Kanban decompose | `hermes_cli/kanban_decompose.py` | → `missing` | missing | Not ported. |

---

## 28. Goal / Standing Objective

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Goal set/pause/resume/clear/status | `hermes_cli/goals.py` | `internal/tui/slash_goal.go` + `internal/tools/` | covered | Goal management in TUI and tools. |
| Goal subgoal add/remove/list | `hermes_cli/goals.py` | `internal/gateway/goal_loop.go` `handleSubgoalCommand` | covered | Add/remove/clear subgoals; saves in GoalState.Subgoals. |
| Goal budget enforcement | `hermes_cli/goals.py` | `internal/persistence/session/goal_state.go` `GoalState.MaxTurns` + `goal_loop.go` | covered | Turn budget tracked via MaxTurns; pauses goal when exhausted. |

---

## 29. Coding Agent Delegation

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Codex binary delegation | N/A (Hermes gateway delegates) | `internal/core/codingagents/` | covered | Codex/claude-code/opencode delegation scaffold. |
| Claude Code binary delegation | N/A | `internal/core/codingagents/` | covered | Shared CodingAgent interface. |
| OpenCode binary delegation | N/A | `internal/core/codingagents/` | covered | Shared CodingAgent interface. |

---

## 30. Explain / Format Helpers

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Hermes state constants | `hermes_constants.py` | `internal/llm/` | partial | Not fully ported. |
| Logging (redacted) | `hermes_logging.py` | `internal/audit/` + `internal/telemetry/` | covered | Audit and telemetry logging. |
| Timezone resolution | `hermes_time.py` `_resolve_timezone_name` `get_timezone` | `internal/llm/time_helpers.go` `GetTimezone` | covered | Reads GORMES_TIMEZONE then HERMES_TIMEZONE; returns *time.Location or nil. |
| `now()` helper | `hermes_time.py` `now` | `internal/llm/time_helpers.go` `Now` | covered | Returns time.Now() in configured timezone or local. |
| `is_truthy_value` | `utils.py` `is_truthy_value` | `internal/llm/helpers.go` `IsTruthyValue` | covered | Boolean coercion for nil/bool/string values. |
| `env_var_enabled` | `utils.py` `env_var_enabled` | `internal/llm/helpers.go` `EnvVarEnabled` | covered | Check os.Getenv against truthy string set. |
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
| MCP tool (client-side) | `tools/mcp_tool.py` | `internal/tools/mcp/boundary` | partial | Tool registration and unavailable evidence exist; generic host list/discovery failures and invoke transport errors now audit as unavailable with redacted `mcp_host_unavailable` evidence, generic host OAuth noninteractive failures and host-returned auth-required results now audit as `auth_required` with redacted `mcp_auth_required` evidence plus a typed `AuthRequired` audit flag, managed gateway client discovery/calls initialize before `tools/list`/`tools/call`, reuse `Mcp-Session-Id`, classify initialize/auth/session-required failures before tool budget, and recover expired streamable HTTP transport sessions by reinitializing and retrying once, while OAuth status/refresh refuses to treat missing access tokens as valid sessions or persist unusable refresh responses (`internal/tools/mcp/boundary/host.go`, `internal/tools/mcp/boundary/host_test.go`, `internal/tools/managed_tool_gateway.go`, `internal/tools/managed_tool_gateway_test.go`, `internal/tools/mcp/oauth/store_test.go`, `internal/tools/mcp/oauth/refresh_test.go`). Remaining gaps: broader OAuth/session lifecycle parity. |
| MCP OAuth flow | `tools/mcp_oauth*.py`; `hermes_cli/mcp_config.py` `_oauth_tokens_present` | `internal/tools/mcp/login`; `internal/tools/mcp/oauth` | partial | Browser callback/token-exchange flow exists with redacted persistence tests, OAuth status/refresh now treats incomplete sessions without access tokens as refresh-needed instead of valid, reports expired noninteractive sessions with no usable refresh token as `noninteractive_required`, rejects refresh responses that omit a usable access token without overwriting the previous credential, whitespace-only refresh tokens degrade to noninteractive auth required without calling the refresher, refreshed whitespace-only refresh tokens are treated as omitted so the previous usable refresh token is preserved, and generic MCP host plus managed gateway OAuth noninteractive/auth-required outcomes surface as redacted `auth_required` instead of generic unavailable or raw host reason text (`internal/tools/mcp/oauth/store_test.go`, `internal/tools/mcp/oauth/refresh_test.go`, `internal/tools/mcp/boundary/host_test.go`, `internal/tools/managed_tool_gateway_test.go`). Remaining gaps: authenticated-session preflight/status wiring across all generic MCP hosts and fuller OAuth session lifecycle parity. |
| MCP managed gateway | `tools/managed_tool_gateway.py` | `internal/tools/managed_tool_gateway.go`; `internal/tools/managed_tool_gateway_test.go`; `internal/tools/mcp_circuit_breaker_test.go`; `internal/tools/web_tools.go`; `internal/tools/image_generation_provider.go` | partial | Managed gateway bridge initializes HTTP MCP sessions, applies bearer/override headers, discovers tools through shared normalization, rejects bad schemas, passes through tool calls/arguments/cancellation, classifies auth/session/OAuth-noninteractive failures as `auth_required`, classifies unavailable/tool-call/circuit-breaker evidence, and is consumed by managed web/tool/image-generation routes; full Hermes managed gateway lifecycle/config surface remains unproven. |
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
| Web research environment | `environments/web_research_env.py`; `tools/web_tools.py`; `tools/x_search_tool.py` degraded citation handling | `internal/tools/web_tools.go` | partial | Root environment not ported, but native web_search now emits backend/source provenance and marks Perplexity no-citation answers as degraded so research flows can distinguish source-backed results from model-synthesized answers. |
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
| Advisory class/detection | `security_advisories.py` `Advisory` `detect_compromised` | `internal/platform/security/advisories.go`; `internal/platform/security/advisories_test.go` | covered | Gormes carries the upstream advisory data shape/catalog and fixture-proves compromised-package detection through an injectable package-version seam while defaulting to no hits in the pure-Go runtime. |
| Advisory ack/ignore | `security_advisories.py` `ack_advisory` `get_acked_ids` | `internal/platform/security/advisories.go`; `internal/platform/security/advisories_test.go` | covered | Gormes persists acknowledged advisory IDs under `$GORMES_HOME/security/acked_advisories.json`, treats missing/corrupt ack state as empty, keeps ack idempotent, and filters acked hits. |
| Startup banner | `security_advisories.py` `short_banner_lines` `hits_due_for_banner` `startup_banner` | `internal/platform/security/advisories.go`; `internal/platform/security/advisories_test.go` | partial | Pure startup banner helper now renders compact active-advisory guidance with `gormes doctor`, filters acked hits, caches recently displayed advisory IDs under `$GORMES_HOME/cache/advisory_banner_seen`, and re-shows unacked hits after 24h. Remaining gap: wire this helper into actual CLI startup. |
| Doctor section | `security_advisories.py` `render_doctor_section` | `internal/platform/security/advisories.go`; `internal/platform/security/advisories_test.go` | partial | Pure doctor-section renderer now filters acked advisory hits, returns a clean `No active security advisories.  ✓` section when none remain, expands active hits to the full remediation block, and separates multiple advisories with blank lines. Remaining gap: wire this renderer into actual `gormes doctor` output. |
| Gateway log message | `security_advisories.py` `gateway_log_message` | `internal/platform/security/advisories.go`; `internal/platform/security/advisories_test.go` | partial | Pure gateway log formatter now filters acked advisory hits, renders single-hit package/version/title/URL messages, summarizes multiple active advisory IDs, and points operators at `gormes doctor` instead of Hermes. Remaining gap: wire it into actual gateway startup logging. |

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
| Cron schedule parser | `cron/scheduler.py` | `internal/automation/cron/` | covered | Schedule parsing. |
| Cron job store | `cron/jobs.py` | `internal/automation/cron/` | covered | Job CRUD. |
| Cron job execution | `cron/scheduler.py` | `internal/automation/cron/` | covered | Scheduled execution. |
| Cron context_from chaining | `cron/jobs.py` | `internal/automation/cron/context_from.go`; `internal/automation/cron/context_from_test.go` | covered | Injects prior completed outputs, truncates each source, and skips unavailable sources with evidence. |
| Cron multi-target delivery | `cron/scheduler.py` | `internal/automation/cron/delivery_plan.go`; `internal/automation/cron/delivery_plan_test.go` | covered | Multi-target delivery routing, `all` expansion, live/standalone/fallback paths, and per-target reports are implemented. |
| Cron resource release | `cron/scheduler.py` | `internal/automation/cron/run_release.go`; `internal/automation/cron/release_binding_test.go` | covered | Registered cron resources are released at run end across success, kernel-error, cancel, and release-error paths. |
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
| Snapshot prune | `hermes_cli/checkpoints.py` `cmd_prune` | `cmd/gormes/checkpoints.go`; `cmd/gormes/checkpoints_test.go`; `internal/tools/checkpoint/manager.go` | covered | `gormes checkpoints prune` supports retention, max-size, keep-orphans, dry-run, JSON outcome, and deletion of orphan/stale checkpoint stores. |
| Snapshot clear (legacy) | `hermes_cli/checkpoints.py` `cmd_clear_legacy` | `cmd/gormes/checkpoints.go`; `cmd/gormes/checkpoints_test.go`; `internal/tools/checkpoint/manager.go` | covered | `gormes checkpoints clear-legacy` reports legacy archives, requires force for destructive clear, emits JSON/noop shapes, and removes legacy checkpoint archives. |

---

## 41. Model Switch

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Model switch CLI (interactive) | `hermes_cli/model_switch.py` | `cmd/gormes/model.go` + `internal/gateway/model_picker.go` | covered | Interactive provider/model picker. |
| Model catalog suggestions | `hermes_cli/model_catalog.py` | `internal/llm/provider_registry_manifest.go` | covered | Provider-specific model catalog. |
| Model normalize | `hermes_cli/model_normalize.py` | `internal/llm/routing/modelcatalog/model_normalize.go`; `internal/llm/model_registry.go`; `internal/llm/model_normalize_test.go` | covered | Provider-aware normalizer ports aggregator vendor prefixes, Anthropic dot-to-hyphen, OpenCode flat namespaces, Copilot alias repair, DeepSeek V-series handling, native prefix/casing rules, and `NormalizeProviderModelID` integration while preserving Gormes Ollama Cloud suffix cleanup. Validation: `go test ./internal/llm/routing/... ./internal/llm -count=1`. |
| Direct alias resolution | `model_switch.py` `_ensure_direct_aliases` | `internal/llm/model_switch.go` `DirectAlias` | covered | DirectAlias type with Model/Provider/BaseURL fields. |
| ModelIdentity parsing | `model_switch.py` `ModelIdentity` | `internal/llm/model_switch.go` `ModelIdentity` + `ModelAliases` | covered | 22 built-in model aliases with vendor/family. |
| ModelSwitchResult | `model_switch.py` `ModelSwitchResult` | `internal/llm/model_switch.go` `ModelSwitchResult` | covered | Structured result with success/newModel/provider/isGlobal/error. |
| `--global` flag support | `model_switch.py` `parse_model_flags` | `internal/llm/model_switch.go` `ParseModelFlags` | covered | Parses --provider, --global, unicode dash normalization. |
| Model sort key | `model_switch.py` `_model_sort_key` | `internal/llm/model_switch.go` `ModelSortKey` | covered | Deterministic sort key + SortedModelAliases. |

---

## 42. Send Command (CLI)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Send text message CLI | `hermes_cli/send_cmd.py` `cmd_send` | `cmd/gormes/send.go`; `cmd/gormes/send_command_test.go` | partial | Top-level `gormes send` is registered and covers positional/file/stdin payloads, `--to`, `--subject`, `--list`, `--quiet`, `--json`, `--dry-run`, no-TUI startup, sanitized errors, and typed `send_backend_unavailable`; live standalone platform delivery remains a backend gap. |
| Oneshot chat (-q) | `hermes_cli/oneshot.py` | `cmd/gormes/chat.go` `-q` | covered | One-shot message to default provider. |
| Target resolution | `send_cmd.py` `_resolve_target` | `cmd/gormes/send.go`; `internal/gateway/delivery.go`; `internal/automation/cron/delivery_plan.go` | partial | CLI accepts and sanitizes `--to PLATFORM[:channel[:thread]]` and shares gateway/cron target parsing, but root `gormes send` still delegates final friendly-name resolution to future standalone delivery backends. |
| Platform-aware target listing | `send_cmd.py` `_list_targets` | `cmd/gormes/send.go` `runSendListCommand`; `internal/gateway/channel_directory*` | covered | `gormes send --list [platform]` reads the channel directory, filters by platform, emits text/JSON, and reports missing targets with configured-platform evidence. |
| Message body reading | `send_cmd.py` `_read_message_body` | `internal/platform/cli/input/messagebody/send_message.go`; `cmd/gormes/send_command_test.go` | covered | Positional, file, `--file -`, and piped stdin bodies preserve newlines, reject invalid UTF-8/NUL bytes, strip terminal response leaks, and report missing body errors. |
| Result formatting | `send_cmd.py` `_emit_result` | `cmd/gormes/send.go` `emitSendCommandResult`; `cmd/gormes/send_command_test.go` | covered | Human, quiet, JSON, dry-run, backend-unavailable, and backend-error outputs are deterministic and sanitized, with nonzero exit codes on errors. |
| Send subcommand registration | `send_cmd.py` `register_send_subparser` | `cmd/gormes/main.go`; `cmd/gormes/send.go`; `cmd/gormes/hermes_cli_parity.go` | covered | `gormes send` is part of the Cobra root and Hermes CLI parity manifest. |

---

## 43. Webhook (Gateway)

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Webhook subscriptions load | `hermes_cli/webhook.py` `_load_subscriptions` | `internal/platform/cli/webhooks/webhook.go`; `internal/platform/cli/webhooks/webhook_test.go` | covered | Hermes-compatible dynamic subscription load returns an empty map for missing, malformed, or non-object JSON files. Validation: `go test ./internal/platform/cli/webhooks -count=1`. |
| Webhook subscriptions save | `hermes_cli/webhook.py` `_save_subscriptions` | `internal/platform/cli/webhooks/webhook.go`; `internal/platform/cli/webhooks/webhook_test.go` | covered | Dynamic subscriptions persist via same-directory atomic temp file, parent creation, JSON object round-trip, and forced `0600` mode before and after rename for HMAC-secret safety. Validation: `go test ./internal/platform/cli/webhooks -count=1`. |
| Webhook config detection | `hermes_cli/webhook.py` `_get_webhook_config` | `internal/platform/cli/webhooks/webhook.go`; `internal/platform/cli/webhooks/webhook_test.go` | covered | Pure helper extracts `platforms.webhook` and degrades missing or malformed config shapes to an empty map like Hermes' guarded config load. Validation: `go test ./internal/platform/cli/webhooks -count=1`. |
| Webhook enabled check | `hermes_cli/webhook.py` `_is_webhook_enabled` | `internal/platform/cli/webhooks/webhook.go`; `internal/platform/cli/webhooks/webhook_test.go` | covered | Enabled check evaluates the extracted webhook config's `enabled` value with Hermes-style truthiness while missing config remains false. Validation: `go test ./internal/platform/cli/webhooks -count=1`. |
| Webhook base URL | `hermes_cli/webhook.py` `_get_webhook_base_url` | `internal/platform/cli/webhooks/webhook.go`; `internal/platform/cli/webhooks/webhook_test.go` | covered | Base URL helper uses Hermes defaults (`0.0.0.0`, `8644`), displays wildcard host as `localhost`, and honors configured host/port. Validation: `go test ./internal/platform/cli/webhooks -count=1`. |
| Webhook setup hint | `hermes_cli/webhook.py` `_setup_hint` | `internal/platform/cli/webhooks/webhook.go`; `internal/platform/cli/webhooks/webhook_test.go` | covered | Setup guidance preserves Hermes' disabled-webhook structure (wizard, manual config, env vars, gateway start) while using Gormes commands and TOML config path instead of stale Hermes labels. Validation: `go test ./internal/platform/cli/webhooks -count=1`. |
| Webhook command dispatch | `hermes_cli/webhook.py` `webhook_command` | `internal/platform/cli/webhooks/webhook.go`; `internal/platform/cli/webhooks/webhook_test.go`; `internal/platform/cli/gormescli/modules/gateway/rowbacked/rowbacked.go` | partial | Pure dispatcher ports Hermes usage-before-enable behavior, disabled-platform setup hint, and `subscribe/add`, `list/ls`, `remove/rm`, `test` action routing with injected handlers; the dynamic `list` side effect renders empty-list guidance, route count, URLs, events, deliver/direct mode metadata, and omits HMAC secrets; `remove` deletes persisted subscriptions, preserves `0600` mode on save, and explains static config routes with Gormes TOML wording; `test` now sends signed JSON POSTs with `X-Hub-Signature-256`, `X-GitHub-Event=test`, response output, and gateway-run failure guidance without leaking secrets. Cobra command currently preserves the same subcommand/alias surface as row-backed CLI wiring, while concrete subscribe side effects and command wiring remain. Validation: `go test ./internal/platform/cli/webhooks -count=1`. |
| Webhook channel delivery | `gateway/platforms/webhook.py` | `internal/channels/webhook/` | covered | Webhook channel for outbound. |

---

## 44. PTY / Terminal Mid-session

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| PTY bridge spawn | `hermes_cli/pty_bridge.py` `PtyBridge.spawn` | `internal/platform/cli/pty/bridge/pty_bridge.go`; `internal/platform/cli/pty/bridge/pty_bridge_linux.go`; `internal/platform/cli/pty/pty_bridge_test.go` | covered | `NewPtyAdapter` rejects unavailable platforms before spawn, validates argv/size, opens a Linux PTY pair, starts the child with cwd/env/winsize, and backfills blank/missing `TERM` to `xterm-256color` without mutating caller env. |
| PTY read | `pty_bridge.py` `read` | `internal/platform/cli/pty/bridge/pty_bridge_linux.go`; `internal/platform/cli/pty/pty_bridge_test.go` | covered | Byte reads poll with bounded timeout, cap chunk size, return empty chunks on timeout, and surface EOF on closed/exited PTYs. |
| PTY write | `pty_bridge.py` `write` | `internal/platform/cli/pty/bridge/pty_bridge.go`; `internal/platform/cli/pty/bridge/pty_bridge_linux.go`; `internal/platform/cli/pty/pty_bridge_test.go` | covered | Client bytes write to the PTY master, oversized/empty payloads fail before reaching the session, and short writes are drained. |
| PTY resize | `pty_bridge.py` `resize` | `internal/platform/cli/pty/bridge/pty_bridge.go`; `internal/platform/cli/pty/bridge/pty_bridge_linux.go`; `internal/platform/cli/pty/pty_bridge_test.go` | covered | Escape-coded resize messages validate dimensions and forward TIOCSWINSZ updates to the child; invalid resize messages never write to PTY stdin. |
| PTY close/cleanup | `pty_bridge.py` `close` | `internal/platform/cli/pty/bridge/pty_bridge_linux.go`; `internal/platform/cli/pty/pty_bridge_test.go` | covered | Close is idempotent, closes the master, signals the process group with HUP/TERM/KILL escalation, waits/reaps the child, and reports not alive after close. |
| PTY availability check | `pty_bridge.py` `is_available` | `internal/platform/cli/pty/bridge/pty_bridge.go`; `internal/platform/cli/pty/pty_bridge_test.go` | covered | `PtyAvailable`/platform guard allow Linux and return typed unavailable errors for Windows/unsupported GOOS before invoking spawn. |
| PTY process PID | `pty_bridge.py` `pid` | `internal/platform/cli/pty/bridge/pty_bridge.go`; `internal/platform/cli/pty/bridge/pty_bridge_linux.go`; `internal/platform/cli/pty/pty_bridge_test.go` | covered | Adapter exposes child PID through the session and fixture-proves a positive child PID before close. |
| PTY aliveness check | `pty_bridge.py` `is_alive` | `internal/platform/cli/pty/bridge/pty_bridge.go`; `internal/platform/cli/pty/bridge/pty_bridge_linux.go`; `internal/platform/cli/pty/pty_bridge_test.go` | covered | Adapter reports session aliveness and returns false once closed/exited. |
| Terminal process registry | `tools/code_execution_tool.py` | → `missing` | missing | Track active terminal sessions. |
| Terminal size/colour passthrough | `tools/code_execution_tool.py`; `hermes_cli/pty_bridge.py` `PtyBridge.spawn` | `internal/platform/cli/pty/bridge/pty_bridge.go`; `internal/platform/cli/pty/bridge/pty_bridge_linux.go`; `internal/platform/cli/pty/pty_bridge_test.go` | partial | PTY spawn and resize now pass terminal rows/cols to the child and backfill `TERM=xterm-256color` for ANSI/color-aware programs. Remaining gap: thread this through the model-facing terminal process registry/tool session surface. |
| PTY error handling | `pty_bridge.py` `PtyUnavailableError` | `internal/platform/cli/pty/bridge/pty_bridge.go`; `internal/platform/cli/pty/pty_bridge_test.go` | covered | Typed `PtyUnavailableError` and `PtyInvalidMessageError` wrap sentinel errors, preserve unavailable reasons such as native Windows/WSL guidance, and are fixture-proven with `errors.Is`. |

---

## 45. Environment Passthrough

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Credential files registration | `tools/credential_files.py` `register_credential_file` | → `missing` | missing | Register files for credential passthrough. |
| Credential file mounts | `tools/credential_files.py` `get_credential_file_mounts` | `internal/tools/docker/mount_policy.go`; `internal/tools/docker/exec.go`; `internal/tools/docker/exec_test.go` | partial | Docker exec resolves configured host-path mounts, blocks dangerous system paths and Docker socket access, maps workspace read-write, and mounts other allowlisted paths read-only; Hermes-style credential-file registration and cross-sandbox mount registry remain missing. |
| Skills directory mount | `tools/credential_files.py` `get_skills_directory_mount` | → `missing` | missing | Mount skills dir read-only. |
| Skills files iterator | `tools/credential_files.py` `iter_skills_files` | → `missing` | missing | Iterate skill files for sandbox. |
| Env passthrough registration | `tools/env_passthrough.py` `register_env_passthrough` | `internal/tools/environment/env_passthrough.go`; `internal/tools/environment/env_passthrough_test.go`; `internal/tools/docker/exec.go`; `internal/tools/docker/exec_test.go` | partial | Pure session-scoped registry now registers skill-declared environment variables, rejects Hermes/Gormes provider credentials such as `OPENAI_API_KEY` and `ANTHROPIC_TOKEN`, preserves non-provider API keys such as `TENOR_API_KEY`, and Docker exec still honors injected allowlists. Remaining gap: wire skill declarations/runtime setup to the registry-backed sandbox allowlist. |
| Env passthrough check | `tools/env_passthrough.py` `is_env_passthrough` `get_all_passthrough` | `internal/tools/environment/env_passthrough.go`; `internal/tools/environment/env_passthrough_test.go`; `internal/tools/docker/exec.go`; `internal/tools/docker/exec_test.go` | partial | Standalone registry now exposes deterministic `IsAllowed` and sorted `All` union checks across session-registered and injected config allowlists, while Docker exec proves allowlist pass/block behavior. Remaining gap: consume the registry from all terminal/code-execution backends. |
| Config passthrough load | `tools/env_passthrough.py` `_load_config_passthrough` | `internal/tools/environment/env_passthrough.go`; `internal/tools/docker/exec.go` | partial | Sandbox execution consumes configured allowlist values when injected by callers, and the pure registry accepts an injected config allowlist while filtering provider credentials. Hermes-compatible config-file loading into that registry is not yet wired. |
| Env passthrough clear | `tools/env_passthrough.py` `clear_env_passthrough` | `internal/tools/environment/env_passthrough.go`; `internal/tools/environment/env_passthrough_test.go` | covered | `ClearRegistered` resets only the session/skill-scoped allowlist while preserving configured allowlist entries, matching Hermes' reset semantics without cross-session bleed. |

---

## 46. AIM / Character / Personality

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Personality list command | `hermes_cli/commands.py` `/personality` | `internal/gateway/personality_command.go` `handlePersonalityCommand` | covered | Lists available personalities with descriptions. |
| Personality switch command | `hermes_cli/commands.py` | `internal/gateway/personality_command.go` | covered | Switches active personality by name. |
| Personality prompt injection | `run_agent.py` | `internal/llm/prompt_assembly.go` `PromptAssemblyOptions.Personality` | covered | Injects personality block when ActivePersonality is set. |\n| Personality source file loading | `hermes_cli/config.py` personalities config | `internal/config/config.go` `AgentCfg.Personalities` | covered | Loads personalities from config agent.personalities map. |\n| Personality `none` clear | `hermes_cli/commands.py` `personality none` | `internal/gateway/personality_command.go` | covered | Clears active personality via /personality none. |]

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
| Default SOUL.md identity | `hermes_cli/default_soul.py` | `internal/llm/context_files.go` | covered | Context file loading from GORMES_HOME. |
| SOUL.md file discovery | `run_agent.py` | `internal/llm/context_files.go` | covered | Workspace ancestor, Gormes home, Hermes home chain. |
| SOUL.md frontmatter stripping | `agent/prompt_builder.py` | `internal/llm/context_files.go` | covered | Frontmatter removed before injection. |
| Truncation marker | `agent/prompt_builder.py` | `internal/llm/context_files.go` | covered | `[...truncated ... kept H+T of N chars ...]` |
| Threat pattern scan | `agent/prompt_builder.py` | `internal/llm/context_files.go` | covered | `[BLOCKED:` marker for prompt injection. |

---

## 49. Self-help / Quick Commands

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Self-help guidance ("what can you do") | `run_agent.py` | `internal/llm/guidance_constants.go` | covered | Byte-equivalent self-help guidance. |
| Quick commands (keyboard shortcuts) | `hermes_cli/__init__.py` | → `missing` | missing | Not ported. |

---

## 50. STDIO / Pipe Mode

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| STDIO mode | `hermes_cli/stdio.py` | → `missing` | missing | Not ported. |

---

## 51. Security & Workspace Permissions (Gormes-owned enhancements)

> These atoms are Gormes-owned enhancements motivated by user pain points (Reddit
> "containerization is a huge hassle") rather than Hermes parity. Rationale:
> Hermes leans on Docker and Python venv isolation; Gormes as a single Go binary
> must provide understandable, container-free file-access safety. Status is
> `owned` with source-backed evidence.

| Atom | HERMES | GORMES | Status | Notes |
|---|---|---|---|---|
| Security Guard (Tirith + path allowlist + URL safety compose) | `agent/file_safety.py` (Hermes write deny list only) | `internal/security/guard.go` `internal/security/tirith.go` | **covered** | Guard composer with deny-wins-over-allow priority. Tests in `internal/security/guard_test.go`. |
| Profile workspace scope | N/A (Hermes uses `HERMES_WRITE_SAFE_ROOT` env var) | `internal/tools/profile_workspace_scope.go` | **covered** | Project roots, profile roots, access levels (read/write/execute/delegate). Tests in `internal/tools/profile_workspace_scope_test.go`. |
| Filesystem scope audit | `hermes_cli/doctor.py` (Vercel sandbox scope check) | `internal/tools/security_audit.go` `SecurityAuditCategoryFilesystemScoping` | **covered** | `gormes security audit --json` checks `FilesystemScoping` with read/write probes. Tests in `internal/tools/security_audit_test.go`. |
| Tool execution audit log (JSONL) | `agent/redact.py` (Hermes redaction only) | `internal/audit/jsonl.go` `internal/kernel/toolexec.go` | **covered** | Per-call audit records with redaction, duration, status. Wired in kernel tool execution. |
| Security advisory check | `hermes_cli/security_advisories.py` | `internal/doctor/doctor_security_advisories.go` | **covered** | Supply-chain advisory check with ack support. |
| Shell command blocklist | `tools/path_security.py` (Hermes path guard) | `internal/tools/shell_blocklist.go` `SecurityAuditCategoryShellBlocklist` | **covered** | Blocklist enforcement with destructive/network/privilege/crypto/data-exfil categories. |
| Doctor command with security fix | `hermes_cli/doctor.py` | `cmd/gormes/doctor.go` | **covered** | `gormes doctor --fix --ack --target --json --offline`. |
| Workspace guard (deny list + allowed roots) | N/A (Hermes has `is_write_denied()`) | `internal/core/codingagents/workspace.go` | **covered** | WorkspaceGuard with default deny of .ssh/.aws/.gcloud/.kube/.gormes. |
| URL safety checker | `tools/url_safety.py` | `internal/tools/url_safety.go` | **covered** | Default allowlist/blocklist policy with URL safety checker. |
| Browser SSRF guard | N/A (Hermes browser tool) | `internal/tools/browser_ssrf_guard.go` | **covered** | Blocks private/loopback/link-local URLs. |
| **Configurable workspace read-only mode** | N/A (Gormes-owned) | `internal/config/config.go` `internal/tools/filesystem_scope.go` `internal/tools/security_audit.go` | **covered** | `[workspace] mode = "readonly"` config option. `FilesystemScope.ReadOnly` denies all writes. Security audit reports it. Tests in `filesystem_scope_test.go`. |
| **`gormes doctor permissions` command** | N/A (Gormes-owned) | → `missing` | **owned** | Show exactly which directories Gormes can read/write, which commands are blocked, and what profiles have access. |
| **User-configurable path allowlist in config** | N/A (Gormes-owned) | → `missing` | **owned** | TOML `[[paths.allow]]` syntax with `path` and `access` (read/readwrite). Default deny outside configured workspace. |
| **Pre-tool dry-run explain access** | N/A (Gormes-owned) | → `missing` | **owned** | Before a tool reads/writes outside the active workspace, Gormes explains the access and asks for confirmation. |
| **macOS-specific permission docs** | N/A (Gormes-owned) | → `missing` | **owned** | Document TCC, Full Disk Access, iCloud folders, Desktop/Documents protections, separate-user setup, and common permission failures. |
| **Air-gapped-ish profile mode** | N/A (Gormes-owned) | → `missing` | **owned** | `[security] network = "off"`, `workspace_access = "readonly"`, `shell = "restricted"` in config. |
| **Threat model documentation** | N/A (Gormes-owned) | → `missing` | **owned** | Documented trust modes: convenience, workspace-restricted, read-only review, advanced container. Explain what each protects against. |
| **Audit log user-facing command** | N/A (Gormes-owned) | → `missing` | **owned** | `gormes audit` or `gormes security audit --recent` to show recent filesystem/tool actions with allow/deny evidence. JSONL already exists. |
| **Default least-privilege workspace** | N/A (Gormes-owned) | → `missing` | **owned** | Default: read/write only inside active workspace. No credential reads unless explicitly requested. No destructive commands without confirmation.
