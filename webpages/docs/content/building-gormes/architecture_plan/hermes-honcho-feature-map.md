---
title: "Hermes And Honcho Feature Map"
weight: 18
---

# Hermes And Honcho Feature Map

This is the planner map for finishing Gormes as Hermes in Go, with Goncho as
the Honcho-compatible Go memory port inside the same runtime.

Hard rule: assume Hermes parity is required for almost every operator-visible
surface. Config files, env precedence, commands, slash commands, gateway
handlers, provider routing/auth/usage, retries, TUI behavior, API shape, tools,
plugins, skills, cron, release, and recovery paths remain parity targets unless
this map or `progress.json` marks a narrow Go-native divergence with tests.
Telegram self-improvement is a proof milestone for that runtime, not the limit
of the product.

Use this page before splitting or building rows. It maps upstream feature
families to the Go packages, implementation strategy, proof gates, and
`progress.json` anchors that should own the work. It is not a side backlog:
missing implementation work still becomes or refines rows in
`architecture_plan/progress.json`.

Use [Upstream Coverage Ledger](../upstream-coverage-ledger/) to verify map
completeness. The map is complete only when every feature-bearing Hermes and
Honcho source class is represented there with a feature-map anchor, Go target,
and progress or owned/excluded decision.

Use [Hermes/Honcho To Gormes Go Runtime Plan](../hermes-honcho-go-runtime-plan/)
when a mapped feature needs concrete Go package ownership, subsystem
classification, dependency order, nested upstream evidence, and row-splitting
guidance for builders. That plan is the reconciled answer to whether a surface
is `mapped-by-symbol`, `mapped-by-contract`, `owned`, `excluded`,
`still row-backed`, or still an `unknown/gap`.

Use [Swarm Feature Parity Audit](../swarm-feature-parity-audit/) for the
feature-level gap register. The source-class ledger can pass while individual
provider, tool, gateway, SDK, MCP, CLI, webhook, queue, or release behavior is
still too vague for builder work.

## Reading Rules

- **Covered** means Gormes has repository evidence and tests.
- **Planned** means there is a specific progress row with source refs, write
  scope, tests, and done signal.
- **Vague** means the feature is represented only by an umbrella row or prose
  and must be split before a builder skill takes it.
- **Missing** means no useful Go surface or progress row was found.
- **Owned** means Gormes intentionally diverges with a Go-native contract.
- **Owned is rare.** Do not mark config, command, provider, or operator
  experience drift as owned just because the first Go implementation is
  simpler; owned divergence needs source-backed rationale, tests, and
  operator-visible docs.

Planner passes should convert vague or missing P0/P1 behavior into one
builder-sized row. Builder passes should not rediscover this map.

## Go Architecture Target

| Runtime concern | Go package target | Proof gate |
|---|---|---|
| Normal agent turn | `internal/kernel`, `internal/hermes`, future `internal/e2e` | `Python-free normal agent turn e2e harness` |
| Provider adapters and routing | `internal/hermes`, `internal/config`, `internal/cli` | Provider transcript and error-classification fixtures |
| Prompt, context, compression | `internal/hermes`, `internal/kernel`, `internal/transcript` | Context status, compression, prompt snapshot fixtures |
| Tool registry and execution | `internal/tools`, `internal/doctor`, `internal/plugins` | Descriptor, trust-class, audit, and handler fixtures |
| Goncho/Honcho memory | `internal/goncho`, `internal/gonchotools`, `internal/memory`, `internal/session` | Goncho SDK-style harness and normal-turn memory fixtures |
| Gateway and channels | `internal/gateway`, `internal/channels/*`, `internal/cron` | Fake-adapter channel, delivery, pairing, cron fixtures |
| API and TUI | `internal/apiserver`, `internal/tui`, `internal/tuigateway`, `cmd/gormes` | OpenAI-compatible API, SSE, health, and TUI fixture gates |
| CLI, config, secrets | `cmd/gormes`, `internal/cli`, `internal/config`, `internal/doctor` | Command parser, profile, auth, doctor, backup, and status tests |
| Skills and plugins | `internal/skills`, `internal/plugins`, `docs/development-skills` | Skill metadata, source, guard, plugin manifest, and docs gates |
| Observability and release | `internal/audit`, `internal/telemetry`, `www.gormes.ai`, installers | Audit/status, docs, site, installer, and package tests |

Namespace note: `internal/hermes` is a staging namespace for
Hermes-compatible runtime contracts while Gormes is still proving parity. The
long-term architecture moves Gormes-owned provider, model registry, tool-call,
prompt, context, and compression code into Gormes-owned package names, leaving
only upstream import/drift/deprecation compatibility under
`internal/compat/hermes`. The Phase 4.I row `Hermes compatibility namespace
retirement boundary` owns that split after registry and normal-turn fixtures
are stable.

## Hermes Feature Map

### Agent Runtime

| Hermes feature | Upstream refs | Gormes target | Implementation plan | Progress anchor |
|---|---|---|---|---|
| `AIAgent` normal loop | `../hermes-agent/run_agent.py`, `../hermes-agent/environments/agent_loop.py` | `internal/kernel`, `internal/hermes`, future `internal/e2e` | Build a Go turn runner that owns prompt assembly, provider stream, tool-call continuation, memory hooks, final response, and audit evidence behind small interfaces. Prove it with fake providers and temp state, not live credentials. | `4.I / Python-free normal agent turn e2e harness` |
| Iteration budget, cancellation, interrupt recovery | `run_agent.py`, `tools/interrupt.py`, `tests/run_agent/test_stream_interrupt_retry.py` | `internal/kernel`, `internal/gateway`, `internal/tools` | Represent each turn as a cancellable context with explicit retry budget and interrupt status. Interrupts must stop retry before provider replay. | Phase 4.H retry rows, Phase 2.F.5 steering rows |
| Transcript and session continuity | `hermes_state.py`, `gateway/session.py`, `run_agent.py` | `internal/session`, `internal/memory`, `internal/transcript`, `internal/store` | Store canonical turns in SQLite/FTS and keep gateway-to-session mapping separate. Preserve parent lineage for compression splits and source-filtered search. | Phase 3.E.8, Phase 4.I |
| Auto title and session naming | `agent/title_generator.py`, `hermes_cli/main.py` | `internal/hermes`, `cmd/gormes`, `internal/cli` | Make title generation an auxiliary provider call with truncation, failure visibility, and no dependency on the main turn completing. | Phase 4.F |
| Trajectory and local edit evidence | `agent/trajectory.py`, local edit snapshot references | `internal/transcript`, `internal/audit`, `internal/tools` | Append turn trajectory and redacted local-change evidence as structured records; never require Python checkpoint code. | Phase 4.E, Phase 5.L |

### Providers, Models, And Credentials

| Hermes feature | Upstream refs | Gormes target | Implementation plan | Progress anchor |
|---|---|---|---|---|
| Provider transport abstraction | `agent/transports/base.py`, `agent/transports/types.py`, `agent/transports/{chat_completions,anthropic,bedrock,codex}.py`, `tests/agent/transports/test_transport.py` | `internal/hermes.ProviderTransport` | Shipped as a Go-native provider request and fixture replay contract for Chat-Completions, Anthropic Messages, Bedrock Converse, and Codex Responses. Keep live credentials, retries, and provider-specific routing outside this contract. | Phase 4.A covered |
| Provider registry, aliases, and auth modes | `hermes_cli/providers.py:HERMES_OVERLAYS,ALIASES`, `agent/models_dev.py:PROVIDER_TO_MODELS_DEV`, `hermes_cli/main.py:--provider choices`, `hermes_cli/models.py` | `internal/hermes`, `internal/config`, `cmd/gormes` | Shipped a source-backed Go manifest in `internal/hermes/provider_registry_manifest.go`. It classifies canonical provider IDs, aliases, transport family, auth type, env/base-url rules, and models.dev mappings as implemented, row-backed, excluded, or owned, with `TestHermesProviderRegistryManifestCoversUpstream` as the drift gate. Unsupported providers remain visible to `gormes model`, status, and docs instead of disappearing because no runtime adapter exists yet. | Phase 4.A `Hermes provider registry and alias manifest` |
| Anthropic adapter | `agent/anthropic_adapter.py` | `internal/hermes` | Keep the completed adapter as the reference provider and add transcript fixtures for future normal-turn replay. | Phase 4.A covered |
| Bedrock adapter | `agent/bedrock_adapter.py` | `internal/hermes` | Finish in narrow slices: stream events, SigV4/credentials, stale-client eviction, retry/error taxonomy. | Phase 4.A |
| Gemini native and Cloud Code | `agent/gemini_native_adapter.py`, `agent/gemini_cloudcode_adapter.py`, `agent/gemini_schema.py` | `internal/hermes` | Add request/stream transcript fixtures, then model capability and tool-call translation. Keep OAuth separate. | Phase 4.A, 4.G |
| OpenRouter/OpenAI-compatible models | `tools/openrouter_client.py`, `agent/model_metadata.py`, `agent/usage_pricing.py` | `internal/hermes`, `internal/config`, `internal/cli` | Use one OpenAI-compatible request path with attribution headers, model family detection, usage/cost metadata, and provider-specific error mapping. | Phase 4.A, 4.D, 4.H |
| Codex Responses and ChatGPT/Codex OAuth | `agent/codex_responses_adapter.py`, `hermes_cli/auth.py`, `agent/auxiliary_client.py`, tests under `tests/run_agent/` | `internal/hermes`, `internal/config`, `internal/cli` | Pure Responses conversion, role-correct content parts, stream repair, and the Go HTTP client `/v1/responses` binding are shipped. Keep OAuth/token vault, account selection, stale-token relogin, model enumeration/fallbacks, and live streaming behavior as their own rows. The Phase 5.O `gormes auth add openai-codex strict isolation contract` and `gormes model interactive provider/model picker` rows are CLI bindings only and must delegate to the existing Codex OAuth state row (Phase 4.G) rather than duplicating token-vault behavior; `auth add openai-codex` is strict (device-code only, refuses `~/.codex/auth.json` import) while `gormes model` retains the legacy import path. | Phase 4.A, 4.G |
| Google/Gemini OAuth, Google Code Assist, and Copilot ACP | `agent/google_oauth.py`, `agent/google_code_assist.py`, `agent/gemini_cloudcode_adapter.py`, `agent/copilot_acp_client.py` | `internal/hermes`, `internal/config`, `internal/plugins` | Treat Google OAuth as Gemini/Google provider auth, not Codex auth. Treat Code Assist and Copilot ACP as provider/backchannel adapters with fake transport fixtures before any live auth. The Phase 5.O `gormes auth status per-provider aggregator` and `Hermes auth OAuth provider adapters` rows are CLI bindings over the existing Google OAuth flow + refresh seam; they must not re-derive Google OAuth state. | Phase 4.A, 4.G, 5.H |
| Auxiliary clients | `agent/auxiliary_client.py`, `tools/xai_http.py` | `internal/hermes` | Define a small auxiliary-completion interface for title, compression, and helper calls. Bind provider-specific clients behind it. | Phase 4.A, 4.F |
| Model metadata, pricing, capabilities | `agent/model_metadata.py`, `agent/models_dev.py`, `agent/usage_pricing.py`, `model_tools.py` | `internal/hermes`, `internal/cli` | One registry feeds routing, context limits, vision support, pricing, and CLI model pickers. Unknown models degrade visibly. | Phase 4.D, 5.O |
| Credential pool, token vault, and auth commands | `agent/credential_pool.py`, `agent/credential_sources.py`, `hermes_cli/auth.py`, `hermes_cli/auth_commands.py`, `hermes_cli/main.py:auth_parser`, `tools/credential_files.py` | `internal/config`, `internal/cli`, `internal/hermes`, `cmd/gormes` | Store credential source metadata without writing resolved plaintext. Provider rows consume a token-vault interface. CLI parity must follow the current non-deprecated Hermes auth surface: `auth add/list/remove/reset/status/logout/spotify`; top-level `login` is removed guidance, not the provider-login implementation target. Keep the detailed operator contract in [Hermes Auth CLI Parity Manifest](../hermes-auth-cli-parity/). | Phase 4.G, 5.O |
| Retry, rate limits, prompt cache, errors | `agent/retry_utils.py`, `agent/rate_limit_tracker.py`, `agent/nous_rate_guard.py`, `agent/prompt_caching.py`, `agent/error_classifier.py` | `internal/kernel`, `internal/hermes`, `internal/telemetry` | Classify errors before retry, preserve provider hints, guard retries on cancellation, and surface budget/rate/prompt-cache evidence. | Phase 4.H |
| Provider account usage reporting | `agent/account_usage.py:render_account_usage_lines`, `agent/account_usage.py:fetch_account_usage`, `cli.py` `/usage`, `gateway/run.py:_handle_usage_command`, `tests/test_account_usage.py`, `tests/gateway/test_usage_command.py` | `internal/hermes`, `internal/telemetry`, `internal/gateway`, `cmd/gormes` | Port the read-only account-usage snapshot and renderer as fixture-backed provider normalization for Codex, Anthropic, and OpenRouter. Bind CLI/gateway `/usage` in a dependent row so running-agent/cached-agent/history fallback and account-limit output are proven without bloating the read-model slice. Unsupported/failing providers preserve Hermes' nonfatal result while Gormes owns typed degraded evidence. | Phase 4.H rows `Provider account usage read model + renderer` and `Gateway /usage command binding over provider account usage` |
| Multimodal input/output and image generation | `agent/image_routing.py`, `agent/image_gen_provider.py`, `agent/image_gen_registry.py`, `tools/image_generation_tool.py`, `tools/vision_tools.py` | `internal/hermes`, `internal/tools` | Route image inputs by model capability, build native content parts, classify too-large errors, then add image generation result envelopes. | Phase 5.D |

### Prompt, Context, Compression, And Skills-In-Prompt

| Hermes feature | Upstream refs | Gormes target | Implementation plan | Progress anchor |
|---|---|---|---|---|
| Prompt builder | `agent/prompt_builder.py`, `run_agent.py`, `cli.py:_load_prefill_messages`, `gateway/run.py:_load_prefill_messages` | `internal/hermes`, `internal/kernel`, `internal/gateway` | Split into system/developer layers, project instructions, tool schemas, memory guidance, skill snapshots, model-specific role guidance, and API-call-time prefill messages. `Ephemeral prefill messages file injection` owns the visible `prefill_messages_file` / env behavior: load safely, insert after the system prompt before history, and never persist or display prefill contents. | Phase 4.C |
| Context engine and status | `agent/context_engine.py` | `internal/hermes` | Keep status and tool boundary typed; add callback vocabulary, kernel binding, pressure evidence, and context reference handles. | Phase 4.B |
| Context compression | `agent/context_compressor.py`, `agent/manual_compression_feedback.py` | `internal/hermes`, `internal/kernel` | Build helper rows first: token budget, protected head/tail, multimodal length, image charge, tool-result pruning, summary lineage, manual feedback. Bind to kernel only after fixtures pass. | Phase 4.B |
| Context references | `agent/context_references.py` | `internal/hermes`, `internal/contextrefs`, `internal/transcript` | Stable parser and pending-handle store are shipped; remaining rows attach or block file/folder/git/url content, budget warnings, and status/audit exposure. | Phase 4.B |
| Subdirectory/project hints | `agent/subdirectory_hints.py`, `hermes_cli/config.py` | `internal/hermes`, `internal/cli` | Resolve project hints before prompt assembly with path safety and deterministic test fixtures. | Phase 4.B/4.C |
| Skill preprocessing and slash skills | `agent/skill_preprocessing.py`, `agent/skill_commands.py`, `agent/skill_utils.py` | `internal/skills`, `internal/hermes`, `internal/cli` | Reuse the Gormes skill store; produce prompt snapshots and slash-command exposure from one reviewed skill registry. | Phase 5.F, 6.C |
| Redaction and evidence filtering | `agent/redact.py`, `hermes_logging.py` | `internal/audit`, `internal/telemetry`, `internal/hermes` | Apply redaction before audit, debug bundles, logs, and prompt-visible references. | Phase 4.E, 5.O |

### Tools, Sandboxes, And Security

| Hermes feature | Upstream refs | Gormes target | Implementation plan | Progress anchor |
|---|---|---|---|---|
| Tool registry and toolsets | `tools/registry.py`, `model_tools.py`, `toolsets.py`, `toolset_distributions.py` | `internal/tools`, `internal/doctor`, `internal/cli` | One descriptor registry must drive schemas, availability, trust classes, doctor checks, gateway/API exposure, and prompt-visible toolsets. | Phase 2.A, 5.A |
| Code execution | `tools/code_execution_tool.py`, `tools/process_registry.py` | `internal/tools`, `internal/cmdrunner` | Keep shipped guarded local execution; add process/session registry and backend sandbox rows separately. | Phase 5.K, 5.B |
| File operations and checkpoints | `tools/file_operations.py`, `tools/file_tools.py`, `tools/checkpoint_manager.py`, `tools/patch_parser.py` | `internal/tools`, `internal/transcript` | First-pass model-visible `read_file`, `search_files`, `write_file`, `patch`, and guarded foreground `terminal` handlers are shipped in the default registry with workspace-root and dangerous-command guardrails. Remaining rows own Hermes' fuzzy/V4A patching, syntax/lint hooks, checkpoint restore/rollback linkage, backend cwd tracking, artifact storage, and terminal process-registry/PTY behavior. | Phase 5.L `Native file task tool surface` plus follow-up file/terminal rows |
| Sandboxes and environments | `tools/environments/*.py`, `environments/hermes_base_env.py`, `environments/agentic_opd_env.py`, `environments/web_research_env.py`, `environments/tool_call_parsers/*.py` | `internal/tools`, `internal/cmdrunner`, `internal/hermes` parser fixtures | Define an environment interface, then add local, Docker, Modal, Daytona, SSH, Singularity, and file-sync adapters behind capability checks. Raw root-environment parsers are fixture-matrix rows, not proof from broad `environments/**` coverage; current parser inventory includes Hermes, DeepSeek, GLM, Kimi, Llama, LongCat, Mistral, Qwen, and Qwen3-Coder families. | Phase 5.B, 5.M |
| Browser and web research | `tools/browser_*.py`, `tools/browser_providers/*.py`, `hermes_cli/browser_connect.py`, `tools/web_tools.py`, sibling `../go-browser-harness/pkg/harness/action.go`, legacy reference `../browser-harness/SKILL.md` | `internal/tools`, `../go-browser-harness`, future browser package, `docs/development-skills/gormes-browser-harness` | First freeze action/event transcripts; then wire Browser Use/go-browser-harness action JSON, Browserbase/Firecrawl, CDP/browser-connect, and chromedp/rod runtime bridges behind the Hermes `browser_*` tool surface. Python browser-harness is reference/explicit legacy compatibility only. | Phase 5.C |
| Voice, TTS, transcription | `tools/tts_tool.py`, `tools/voice_mode.py`, `tools/transcription_tools.py`, `tools/neutts_synth.py` | `internal/tools`, `internal/cli` | Port result envelopes and state machines before live audio backends. | Phase 5.E |
| Operator tools | `tools/todo_tool.py`, `tools/clarify_tool.py`, `tools/session_search_tool.py`, `tools/send_message_tool.py`, `tools/debug_helpers.py`, `tools/interrupt.py` | `internal/tools`, `internal/sessionsearchtool`, `internal/gateway` | Keep each operator-visible contract separate: todo state, clarify prompts, session search, send-message routing, debug share cleanup, interrupt handling. `todo` and `session_search` have native handlers, but default live-turn exposure still needs session-aware binding; do not register them like stateless tools until the current session/store contract is proven. | Phase 5.N |
| Delegate/subagents | `tools/delegate_tool.py` | `internal/subagent`, `internal/tools` | Preserve shipped deterministic runtime and add policy/security rows around long-running workers and approval behavior. | Phase 2.E, 5.J |
| MCP tools and managed gateway | `tools/mcp_tool.py`, `tools/mcp_oauth*.py`, `tools/managed_tool_gateway.py`, `mcp_serve.py` | `internal/plugins`, `internal/tools`, `internal/apiserver` | Start with manifest/config and OAuth state machines; then add stdio/HTTP tool bridge, sampling, and managed provider clients. | Phase 5.G |
| ACP adapter | `acp_adapter/auth.py:detect_provider`, `acp_adapter/entry.py:main`, `acp_adapter/{events,permissions,server,session,tools}.py`, `acp_registry/agent.json` | `internal/plugins`, `internal/apiserver`, future `internal/acp` | Port auth/session/events/tool permission contracts with fake client fixtures before IDE-specific integration. | Phase 5.H |
| Approval and security policy | `tools/approval.py`, `tools/path_security.py`, `tools/url_safety.py`, `tools/tirith_security.py`, `tools/website_policy.py`, `tools/osv_check.py` | `internal/tools`, `internal/doctor`, `internal/config` | Normalize approval modes, classify dangerous commands, enforce path/URL/site policies, and make non-interactive denial explicit. | Phase 5.J |

### Gateway, Channels, Cron, API, TUI, And CLI

| Hermes feature | Upstream refs | Gormes target | Implementation plan | Progress anchor |
|---|---|---|---|---|
| Gateway runtime | `gateway/run.py`, `gateway/config.py` | `internal/gateway`, `cmd/gormes` | One manager owns channel lifecycle, command dispatch, active-turn policy, session mapping, delivery, status, and restart. | Phase 2.B, 2.F |
| Session context and delivery | `gateway/session_context.py`, `gateway/delivery.py`, `gateway/session.py` | `internal/gateway`, `internal/session` | Keep session context prompt injection typed; add home-channel ownership, contact directory, cross-channel delivery resolution. | Phase 2.F.4 |
| Stream consumer and UI fanout | `gateway/stream_consumer.py`, `gateway/display_config.py` | `internal/gateway`, `internal/tuigateway`, `internal/tui` | Emit normalized frames once; let channels/TUI render them without owning kernel behavior. | Phase 2.F, 5.Q |
| Hooks, boot, pairing, restart, status | `gateway/hooks.py`, `gateway/builtin_hooks/boot_md.py`, `gateway/pairing.py`, `gateway/restart.py`, `gateway/status.py` | `internal/gateway`, `cmd/gormes` | Keep hook loading, pairing approval, restart markers, PID validation, and status JSON as separate fixtures. | Phase 2.F |
| Channels | `gateway/config.py:Platform`, `gateway/platforms/*.py` | `internal/channels/*`, `internal/slack`, `internal/discord`, `internal/gateway` | First lock a source-backed platform registry manifest so every Hermes platform enum value and connector file is classified. Then port each channel as fake-adapter slices: inbound normalization, command passthrough, outbound delivery, identity/self-filter, bootstrap, then optional live smoke docs. | Phase 2.B, 7; `Hermes gateway platform registry manifest` |
| Cron and scheduled jobs | `cron/scheduler.py`, `cron/jobs.py`, `tools/cronjob_tools.py` | `internal/cron`, `internal/gateway`, `internal/tools` | Store durable jobs/runs in Go, support schedule parser, context_from chaining, prompt/script safety, resource release, and multi-target delivery. | Phase 2.D, 5.N |
| OpenAI-compatible API | `gateway/platforms/api_server.py`, `.plans/openai-api-server.md` | `internal/apiserver` | Implement `/v1/chat/completions`, Responses, Runs, detailed health, cron admin, proxy mode, disconnect snapshots, and dashboard contracts as independent rows. | Phase 5.Q |
| TUI gateway and interactive UI | `ui-tui/src/components/appLayout.tsx`, `ui-tui/src/components/appChrome.tsx`, `ui-tui/src/components/messageLine.tsx`, `ui-tui/src/components/thinking.tsx`, `ui-tui/src/components/queuedMessages.tsx`, `ui-tui/src/app/createSlashHandler.ts`, `ui-tui/src/app/slash/registry.ts`, `tui_gateway/*`, legacy `cli.py` only for behavior not represented in the current Ink UI | `internal/tui`, `internal/tuigateway`, `internal/cli`, `cmd/gormes` | Keep Bubble Tea native but match the active Hermes Ink operator contract: alt-screen full-screen chat chrome, bottom-pinned transcript/status/composer stack, role glyphs, queued messages, sticky prompt, slash completion and slash dispatch, busy-input routing, modal panels, paste/image handling, tool progress, and save/branch/interrupt/edit helpers. Prompt-toolkit refs are legacy evidence only when the current Ink source has no equivalent behavior. | Phase 1, 5.Q |
| CLI command tree | `hermes_cli/main.py`, `cli.py`, `hermes_cli/commands.py`, `hermes_cli/auth.py`, `hermes_cli/auth_commands.py`, `gateway/run.py`, `plugins/memory/__init__.py`, `plugins/disk-cleanup/__init__.py` | `cmd/gormes`, `internal/cli`, `internal/config`, `internal/doctor`, `internal/plugins` | First lock a source-backed manifest for every top-level Hermes parser command, nested subcommand, root/global flag, slash command, alias, gateway handler, and dynamic plugin command; then port command groups by public behavior: setup/auth/profile/model/provider, gateway/cron/webhook, backup/logs/status/doctor, output and completion helpers. Runtime-visible command alias/prefix/suggestion behavior is tracked by `Hermes CLI alias and suggestion fidelity matrix` so aliases canonicalize before dispatch/hooks, unique prefixes and ambiguity match Hermes, quick-command aliases preserve args, and unknown commands stay guidance instead of model prompt text. The auth subgroup must be kept current with `auth add/list/remove/reset/status/logout/spotify`, including removed top-level `login` guidance and secret/destructive flags. Gormes-owned additions such as `goncho`, `--offline`, `--remote`, and XDG/TOML config paths must be classified as owned, not Hermes gaps; Hermes-owned `-z/--oneshot` is parity/covered, not owned. The non-deprecated provider auth/lifecycle commands are the explicit set `gormes auth add/list/remove/reset/status/logout/spotify`, plus the parallel top-level `gormes logout --provider {nous|openai-codex|spotify}` shortcut, `gormes model` (interactive provider/model picker), `gormes setup` (completed minimal sectioned wizard plus planned chooser, full-wizard summary, tools checklist, gateway checklist, terminal/TTS, and agent-settings rows; model/provider delegates to the model picker and auth rows), and `gormes mcp login <name>` (OAuth-only re-auth bridge); `gormes login` is registered as a deprecated parser shim that accepts every legacy flag without argparse error and prints a deterministic three-line redirect to `gormes auth`, `gormes model`, and `gormes setup` (typo-suggestion parity with `gormes migrate ooenclaw -> gormes migrate openclaw`, never silent). The detailed operator contract for these surfaces lives in [Hermes Auth CLI Parity Manifest](../hermes-auth-cli-parity/). | Phase 5.O |
| Config command and provider overrides | `hermes_cli/config.py`, `tests/hermes_cli/test_set_config_value.py`, `tests/hermes_cli/test_config_env_expansion.py`, `tests/hermes_cli/test_config_validation.py` | `cmd/gormes`, `internal/config`, `internal/cli` | Preserve Hermes `config.yaml` compatibility as the default bridge: runtime reads Hermes profile `config.yaml` for mapped settings such as Telegram bot config, `model.default`, and `model.provider` before higher-precedence Gormes overrides. Add `gormes config show/path/env-path/set`, then close `edit/check/migrate` so persistent writes converge toward Hermes semantics where possible. Invocation-only endpoint/model/api-key flags stay separate from persistent config, secret redaction is mandatory, explicit-empty handling follows Hermes, and broad cross-product imports remain explicit migration rows. | Phase 5.O |
| Hermes state/config migration | `hermes_constants.py`, `hermes_state.py`, `hermes_cli/config.py`, `tests/hermes_cli/test_config*.py` | `cmd/gormes`, `internal/migrate/hermes`, `internal/config`, `internal/session`, `internal/memory` | `gormes migrate hermes` is the only path that reads `HERMES_HOME` or `~/.hermes`. Start with a dry-run manifest, then apply config/dotenv with backups; session and memory import remain explicit follow-up rows. | Phase 5.O |
| OpenClaw migration | `hermes_cli/claw.py`, `optional-skills/migration/openclaw-migration/scripts/openclaw_to_hermes.py`, `website/docs/guides/migrate-from-openclaw.md` | `cmd/gormes`, `internal/migrate/openclaw`, `internal/config`, `internal/skills`, `internal/memory` | Port Hermes' OpenClaw migration contract into Go as `gormes migrate openclaw`: dry-run manifest first, field-level mappings for config/secrets/memory/skills/MCP/channels/providers, explicit secret opt-in, reports, backups, and a separate cleanup command. Misspelled requests such as `gormes migrate ooenclaw` must suggest `gormes migrate openclaw` without becoming silent aliases. | Phase 5.O |
| Packaging and release | `scripts/release.py`, `Dockerfile`, `docker/entrypoint.sh`, `docker-compose.yml`, `packaging/homebrew/hermes-agent.rb`, installer docs | `Makefile`, `cmd/gormes`, `www.gormes.ai`, package scripts | Ship one Go binary with installers, service units, OCI image, Homebrew/Windows/Unix flows, version surfaces, and docs matching runtime behavior. | Phase 5.P |

### Plugins, Skills, Learning, And Specialized Modes

| Hermes feature | Upstream refs | Gormes target | Implementation plan | Progress anchor |
|---|---|---|---|---|
| Skill manager/hub/sync/guard | `tools/skill_manager_tool.py`, `tools/skills_hub.py`, `tools/skills_sync.py`, `tools/skills_guard.py`, `skills/`, `optional-skills/` | `internal/skills`, `docs/development-skills`, `internal/cli` | Preserve portable `SKILL.md` metadata, source lockfiles, readiness/guard state, review status, retrieval, and prompt snapshots. | Phase 5.F, 6 |
| Plugin SDK and first-party plugins | `plugins/*`, `plugins/spotify`, `plugins/google_meet`, image-gen plugins | `internal/plugins`, `internal/tools`, `internal/apiserver` | Start with manifest/capability loader, then add one first-party plugin fixture, dashboard assets, and tool registration. | Phase 5.I |
| Memory plugins | `plugins/memory/*`, especially `plugins/memory/honcho` | `internal/goncho`, `internal/gonchotools`, `internal/plugins` | External compatibility names stay `honcho_*`; internal memory implementation remains Goncho. Other memory plugins become adapter rows after generic plugin contract. | Phase 3.G, 5.I |
| Batch, mini-SWE, RL, datagen | `batch_runner.py`, `mini_swe_runner.py`, `rl_cli.py`, `datagen-config-examples/` | `internal/tools`, `internal/subagent`, future research packages | Defer research modes until normal turn, tools, and release lanes are stable. Port only hermetic runner contracts first. | Phase 5.M/5.O |
| Learning loop | `run_agent.py` background review, `agent/curator.py`, `hermes_cli/curator.py`, `hermes_cli/config.py` auxiliary defaults, `hermes_cli/main.py` auxiliary picker, `hermes_cli/web_server.py` auxiliary model API, skill and memory rows plus Gormes Phase 6 docs | `internal/hermes`, `internal/skills`, `internal/memory`, `internal/config`, `internal/cli`, `internal/kernel` | Port Hermes background-review forks, `auxiliary.curator` model routing, and curator state/CLI first, then add Gormes-native detectors, candidate extraction, review/promotion, hybrid retrieval, and outcome scoring. | Phase 6 |

## Honcho Feature Map For Goncho

Goncho is not a hosted Honcho clone. It is the in-process Go memory system that
must preserve Honcho-compatible public contracts where Gormes tools, SDK-style
flows, or users depend on them.

### Honcho Concepts And APIs

| Honcho feature | Upstream refs | Gormes target | Implementation plan | Progress anchor |
|---|---|---|---|---|
| Workspaces/apps | `src/models.py:Collection,Document,QueueItem,ActiveQueueSession,WebhookEndpoint`, `src/routers/workspaces.py`, `src/crud/workspace.py`, `docs/v3/api-reference/endpoint/workspaces/*` | `internal/goncho`, `internal/config` | Represent workspace identity in SQLite with config defaults and explicit missing-workspace diagnostics. Do not require hosted app keys for local use. | Phase 3.F/3.G |
| Peers and peer config | `src/routers/peers.py`, `src/crud/peer.py`, `src/crud/peer_card.py` | `internal/goncho`, `internal/gonchotools` | Preserve peer identity, cards, representation scopes, and empty-card hints behind `honcho_profile` compatibility tools. | Phase 3.F, 3.G |
| Sessions and session peers | `src/models.py:SessionPeer`, `src/routers/sessions.py`, `src/crud/session.py`, docs `sessions/*` | `internal/goncho`, `internal/session`, `internal/memory` | Local session delete cascades and workspace isolation are shipped; clone/update/session-peer HTTP compatibility remains row-backed. | Phase 3.F complete row `Goncho CRUD lifecycle invariants`; Phase 3.G |
| Messages and file-backed messages | `src/models.py:MessageEmbedding`, `src/routers/messages.py`, `src/crud/message.py`, `src/crud/document.py`, `docs/v3/api-reference/endpoint/messages/*` | `internal/goncho`, `internal/memory` | Local message creation now stores workspace/session/peer sequence metadata; file-backed message upload/import and route pagination/filter/update/delete compatibility remain row-backed. | Phase 3.F complete row `Goncho CRUD lifecycle invariants`; Phase 3.G |
| Conclusions and facts | `src/routers/conclusions.py`, `src/crud/deriver.py`, `src/deriver/*` | `internal/goncho`, `internal/memory` | Map conclusions to Gormes facts/entities/relationships with provenance and queue evidence. Keep derivation async and auditable. | Phase 3.F, 6 |
| Representations | `src/crud/representation.py`, `src/crud/collection.py:collection_cache_key,get_collection,get_or_create_collection,update_collection_internal_metadata`, docs `representation.mdx`, `representation-scopes.mdx` | `internal/goncho`, `internal/memory` | Store representation cards by scope and peer/session context; expose deterministic context options and diagnostics. Collection cache/key behavior is tracked as a source-backed CRUD invariant rather than hidden under `src/crud/**`. | Phase 3.F |
| Search and filters | `src/utils/filter.py`, `src/utils/search.py`, docs `search.mdx`, `using-filters.mdx` | `internal/goncho`, `internal/memory` | Parse Honcho-compatible filter grammar into safe SQL/FTS/graph queries, rejecting unsupported filters visibly. | Phase 3.F |
| Context retrieval | docs `get-context.mdx`, `src/utils/summarizer.py`, `src/utils/representation.py` | `internal/goncho`, `internal/gonchotools`, `internal/hermes` | Build scoped context from session, user, peer cards, summaries, and observations within a bounded prompt budget. | Phase 3.F, 4.C |
| Dialectic chat | `src/dialectic/chat.py`, `src/dialectic/core.py`, `src/dialectic/prompts.py`, docs `chat.mdx` | `internal/goncho`, `internal/kernel` | Provide a local dialectic chat contract first, then integrate with normal agent turn using fake provider fixtures. | Phase 3.F/3.G |
| Streaming chat persistence | docs `streaming-response.mdx`, dialectic chat tests | `internal/goncho`, `internal/memory` | Buffer stream chunks and persist only the final assistant message; interruptions produce evidence without memory pollution. | Phase 3.F covered |
| Summaries | docs `summarizer.mdx`, `src/utils/summarizer.py` | `internal/goncho`, `internal/memory`, `internal/hermes` | Store summary context separately from raw messages, with prompt-budget accounting and stale summary evidence. | Phase 3.F |
| Dreaming | `src/dreamer/dream_scheduler.py`, `src/dreamer/orchestrator.py`, `src/dreamer/specialists.py`, `src/dreamer/surprisal.py`, `src/dreamer/trees/{base,covertree,graph,lsh,prototype,rptree,sklearn_wrapper}.py`, docs `dreaming.mdx` | `internal/goncho`, `internal/memory`, `internal/cron` | Keep scheduler and queue state local; add LLM-backed dream execution only after provider and queue fixtures. Tree/surprisal algorithms stay source-backed by exact rows before any broad dream parity claim. | Phase 3.F, 6 |
| Queue status | docs `queue-status.mdx`, `src/deriver/queue_manager.py`, `src/reconciler/*` | `internal/goncho`, `internal/doctor`, `cmd/gormes` | Expose queue depth, failures, stale work, and degraded derivation state in `gormes goncho doctor` and JSON status. | Phase 3.F |
| File uploads | docs `file-uploads.mdx`, `src/routers/messages.py`, `src/utils/files.py` | `internal/goncho`, `internal/memory` | Import files into local observations/documents with content-type, provenance, and safe size limits. | Phase 3.F |
| Keys/auth | `src/routers/keys.py`, `src/security.py`, docs key endpoints | `internal/config`, `internal/apiserver`, `internal/goncho` | Local Goncho should not need hosted Honcho auth; any compatibility HTTP adapter must validate local operator/API keys. | Phase 3.G/5.Q |
| Webhooks | `src/routers/webhooks.py`, `src/webhooks/events.py`, `src/webhooks/webhook_delivery.py`, `tests/webhooks/test_webhook_delivery.py` | `internal/apiserver`, `internal/gateway`, `internal/goncho` | Keep endpoint CRUD/test/signing complete in Goncho, then port outbound delivery as a fake-HTTP worker contract with queue-empty/test events, HMAC headers, retry/backoff/failure evidence, and disabled-endpoint behavior before API route or SDK claims. | Phase 3.G rows `Goncho keys + webhooks compatibility surface` and `Goncho webhook delivery retry worker contract`; 5.Q |
| SDKs and MCP | `sdks/python/src/honcho/{client,aio,api_types,mixins}.py`, `sdks/typescript/src/client.ts`, `sdks/typescript/src/validation.ts`, `sdks/typescript/src/http/streaming.ts`, `mcp/src/*` | `internal/goncho`, future local compatibility adapter | Build a hermetic SDK-style harness in Go that covers request/response shape before claiming drop-in compatibility. | `3.G / Goncho Honcho SDK compatibility e2e harness` |
| CLI/self-hosting docs | `honcho-cli/src/honcho_cli/{main,config,output,validation,_help,branding}.py`, `honcho-cli/src/honcho_cli/commands/{workspace,peer,session,message,conclusion,config_cmd,setup}.py`, `.env.template`, `config.toml.example`, `docker-compose.yml.example`, `docker/entrypoint.sh`, `docker/prometheus.yml`, `docker/grafana-datasource.yml`, docs `self-hosting.mdx`, `configuration.mdx` | `cmd/gormes`, `internal/config`, `internal/doctor` | Map required config knobs into `[goncho]`; expose local doctor/status instead of copying Honcho service deployment. Hosted FastAPI/Postgres/Redis/Fly/Alembic and Prometheus/Grafana adjuncts are owned/excluded until a local exporter/deploy row requires them. | Phase 3.F/3.G |
| Vector stores and reconciler | `src/vector_store/*`, `src/reconciler/*`, migrations | `internal/memory`, `internal/store` | Prefer SQLite/FTS/graph first; only add vector-store adapters if a compatibility row proves local embeddings are insufficient. | Phase 3.D/3.G |
| Telemetry and reasoning traces | `src/telemetry/emitter.py`, `src/telemetry/reasoning_traces.py`, `src/telemetry/metrics_collector.py`, `src/telemetry/sentry.py`, `src/telemetry/events/deletion.py`, docs reasoning | `internal/telemetry`, `internal/audit`, `internal/goncho` | Emit local reasoning/queue/context evidence with trace IDs, tree levels, event names, and redaction; classify hosted Prometheus/Sentry/exporter-only behavior as owned/excluded unless a later row adds exporters. | Phase 3.F, 4.E row `Self-monitoring telemetry` |

## Implementation Order

The feature map collapses into this order. The order prioritizes proof of the
normal turn, but it does not demote CLI/config/provider/operator experience
parity to nice-to-have work:

1. **Prove the normal Go turn.** Build `Python-free normal agent turn e2e
   harness` before expanding broad provider or tool ports. It is the first
   product-level proof that Gormes is replacing Hermes, not wrapping it.
2. **Lock operator-control parity early.** Keep command-tree, config
   lifecycle, provider account/usage, status, slash-command, gateway-command,
   and TUI interaction rows near the front whenever they affect how long coding
   turns are configured, steered, inspected, or recovered.
3. **Prove Goncho drop-in behavior.** Build `Goncho Honcho SDK compatibility
   e2e harness`, then bind Goncho into the normal turn. That is the point at
   which Goncho can be evaluated as Honcho-in-Go.
4. **Split Phase 4 broad rows.** Provider adapters, context compression,
   prompt assembly, model metadata, credentials, retry, and prompt cache should
   be small transcript-backed rows.
5. **Descriptor-first tools.** Do not port 61 tools by hand. Port the registry,
   toolset, trust, availability, audit, and doctor contract first; then port
   handlers by category.
6. **Expose through API/TUI/gateway from typed events.** Channels and UI should
   consume typed turn events, not recreate agent logic, while still matching
   Hermes' operator-facing commands, slash completion, progress, approvals,
   interrupt, status, and usage behavior.
7. **Release packaging and learning loop last.** Public install promises and
   automatic skill generation must wait for normal-turn, memory, tool, command,
   config, and provider-control gates.

## Rows That Must Stay Near The Top

| Row | Why |
|---|---|
| `Python-free normal agent turn e2e harness` | Product-level proof for Hermes-in-Go. |
| `Goncho Honcho SDK compatibility e2e harness` | Product-level proof for Honcho-in-Go memory compatibility. |
| `Goncho memory integration into normal agent turn` | Binds memory to the real agent spine. |
| `Provider-tool-memory golden transcript suite` | Keeps future provider/tool/memory changes from drifting. |
| `Hermes CLI command-tree parity manifest` | Prevents command, slash, gateway-handler, flag, alias, and plugin-command drift from becoming invisible. |
| `Hermes auth command-tree manifest refresh` | Prevents Gormes from implementing deprecated/stale auth paths such as top-level `login` or `auth login` instead of current `auth add/list/remove/reset/status/logout/spotify`. |
| `Hermes auth credential-pool command surface` | Gives operators a native way to add, list, remove, reset, status-check, and logout provider credentials before Go-native provider turns depend on multiple accounts. |
| `Hermes auth OAuth provider adapters` | Binds Codex, Anthropic, Nous, Google Gemini CLI, and Qwen OAuth login into the native credential store with fake-client fixtures before live provider use. |
| `Provider endpoint/API-key root flags + runtime resolution` | Makes provider/model/account selection deterministic before dogfood coding turns rely on it. |
| `Gormes config command surface` | Closes the config lifecycle operators use to inspect and repair the running runtime. |
| `Native TUI Hermes slash completion helpers` | Keeps the local operator experience aligned with Hermes while Gormes stays Go-native. |
| `Skill-manager selection matrix hardening` | Keeps future agents on the skill-driven control plane. |
| `ContextEngine compression-boundary callback vocabulary` | Unblocks compression/kernel binding with a narrow fixture. |

## Builder Packet Contract

Every feature-map-derived progress row should give the builder skill a complete
packet before runtime edits start:

- upstream feature or Honcho concept from this map;
- target Go package and public interface under test;
- fixture or fake transport to avoid live providers and hosted services;
- degraded mode for unsupported, missing, or credential-gated behavior;
- row-local acceptance and done signal;
- narrow write scope;
- focused test commands.

Builder passes should read this map, the selected `progress.json` row, and the
row's upstream refs before editing. If the row and the map disagree, the next
move is a planner update, not an implementation guess.

## Skill Consumption Contract

The Gormes skills consume this map in a fixed order:

- `gormes-skill-manager` routes the task by feature-map area, progress row, and
  whether the work is planning, auditing, interface design, building, TDD, or
  skill creation.
- `gormes-parity-auditor` classifies map entries as covered, planned, vague,
  missing, or owned against current code and `progress.json`.
- `gormes-planner` updates this map and `progress.json` when upstream behavior
  is unmapped, too broad, or out of order.
- `gormes-interface-designer` shapes the Go package boundary when the map names
  a target package but the row lacks a stable public contract.
- `gormes-builder` selects one builder-ready row, builds a packet from this map
  and the row fields, then implements only that slice.
- `gormes-tdd-slice` turns the packet into red-green-refactor tests for one
  observable behavior at a time.

## Planner Duties After This Map

- When adding a row, cite exact upstream paths from the tables above.
- When a row is broad, split it before builder selection.
- When a feature is intentionally Go-native, mark it `owned` and state why.
- When live credentials would be required, write a fake-transport fixture row
  first.
- When the docs map and `progress.json` disagree, fix `progress.json` and
  regenerate generated docs with `go run ./cmd/progress write`.
