---
title: "Hermes/Honcho To Gormes Go Runtime Plan"
weight: 20
---

# Hermes/Honcho To Gormes Go Runtime Plan

This is the canonical implementation plan for making Gormes a Go-native
runtime for the useful Hermes-Agent and Honcho surfaces.

It reconciles the feature map, upstream coverage ledger, swarm audit, and
`progress.json`. It is not a side backlog. Any implementation work named here
must be represented by a progress row before a builder agent starts coding.

This page answers the mapping question, not the implementation-complete
question. After the 2026-04-28 two-wave swarm pass, every broad Hermes/Honcho
surface is classified as `mapped-by-symbol`, `mapped-by-contract`, `owned`,
`excluded`, or `still row-backed`. No broad upstream Hermes/Honcho
subsystem-level `unknown/gap` remains; feature-level residuals remain
row-backed, and Gormes-native pending decisions are tracked separately.

## Evidence Corpus

| Source | Role |
|---|---|
| `../hermes-agent/run_agent.py`, `../hermes-agent/agent/**`, `../hermes-agent/tools/**`, `../hermes-agent/gateway/**`, `../hermes-agent/hermes_cli/**` | Hermes runtime, providers, tools, gateway, CLI, config, packaging, and operational behavior. |
| `../hermes-agent/acp_adapter/**`, `../hermes-agent/mcp_serve.py`, `../hermes-agent/plugins/**`, `../hermes-agent/skills/**`, `../hermes-agent/optional-skills/**` | Hermes extension, ACP, MCP, plugin, and skill behavior. |
| `../honcho/src/**`, `../honcho/migrations/**`, `../honcho/docs/v3/openapi.json`, `../honcho/tests/**` | Honcho API, persistence, queue, dialectic, dream, telemetry, webhook, and behavior fixtures. |
| `../honcho/sdks/python/src/**`, `../honcho/sdks/typescript/src/**`, `../honcho/sdks/typescript/__tests__/**`, `../honcho/mcp/src/tools/**`, `../honcho/honcho-cli/src/honcho_cli/**` | Honcho client, SDK, MCP, and CLI compatibility contracts. |
| `cmd/**`, `internal/**`, `docs/content/building-gormes/architecture_plan/progress.json` | Current Gormes implementation evidence and the only executable backlog. |
| `docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md`, `upstream-coverage-ledger.md`, `swarm-feature-parity-audit.md`, `docs/upstream_coverage_test.go` | Current planning, coverage, and test gates. |

## Classification Vocabulary

| Class | Meaning |
|---|---|
| `owned` | Gormes intentionally diverges with a Go-native contract and a stated replacement proof. |
| `mapped-by-symbol` | Exact upstream symbols/files have a Gormes target and row/test evidence. |
| `mapped-by-contract` | A large upstream surface is mapped by public behavior, fixtures, and progress rows rather than one symbol at a time. |
| `excluded` | The upstream surface is repo hygiene, generated output, hosted-service deployment detail, or intentionally unsupported behavior. |
| `still row-backed` | The feature is mapped and has a progress row, but implementation is not complete. |
| `unknown/gap` | No acceptable mapping exists. This class must not survive a planner pass; convert it to one of the classes above with rationale. |

Current status: no broad upstream Hermes/Honcho surface should remain
`unknown/gap`. Large surfaces are either `mapped-by-contract`,
`still row-backed`, `owned`, or `excluded`. Feature-level residuals are
represented by progress rows or row-split candidates; the remaining work is
implementation and row splitting, not rediscovery.

## Hermes Feature Map

| Hermes surface | Upstream evidence | Go implementation target | Classification | Progress/register owner |
|---|---|---|---|---|
| Normal agent loop, turn lifecycle, interrupts, trajectory | `run_agent.py`, `environments/agent_loop.py`, `hermes_state.py`, `trajectory_compressor.py`, `tests/run_agent/**` | `internal/kernel`, `internal/hermes`, `internal/transcript`, `internal/audit`, future `internal/e2e` | still row-backed | Phase 4.I, 4.E, 4.H |
| Provider transport layer | `agent/transports/base.py`, `agent/transports/types.py`, `agent/transports/chat_completions.py`, `agent/transports/codex.py`, `agent/transports/anthropic.py`, `agent/transports/bedrock.py`, `tests/agent/transports/**` | `internal/hermes.ProviderTransport`, provider request builders, and fake transcript fixtures | mapped-by-symbol for the shared transport/fixture contract; remaining live provider auth/routing/error work stays row-backed under adapter rows | Phase 4.A |
| Provider adapters and model families | `agent/anthropic_adapter.py`, `agent/bedrock_adapter.py`, `agent/gemini_native_adapter.py`, `agent/gemini_cloudcode_adapter.py`, `agent/codex_responses_adapter.py`, `agent/moonshot_schema.py`, `agent/model_metadata.py`, `agent/models_dev.py`, `model_tools.py` | `internal/hermes`, `internal/config`, `internal/cli` | still row-backed | Phase 4.A, 4.D |
| Credentials, auth, OAuth, rate, cache, errors | `agent/credential_pool.py`, `agent/credential_sources.py`, `agent/google_oauth.py`, `agent/retry_utils.py`, `agent/rate_limit_tracker.py`, `agent/prompt_caching.py`, `agent/error_classifier.py`, `agent/account_usage.py`, `hermes_cli/auth*.py` | `internal/config`, `internal/cli`, `internal/hermes`, `internal/telemetry`, `internal/gateway` | still row-backed | Phase 4.G, 4.H, 5.O |
| Provider account usage reporting | `agent/account_usage.py:render_account_usage_lines`, `agent/account_usage.py:fetch_account_usage`, `cli.py` `/usage`, `gateway/run.py:_handle_usage_command`, `tests/test_account_usage.py`, `tests/gateway/test_usage_command.py` | `internal/hermes`, `internal/telemetry`, `internal/gateway`, `cmd/gormes` | still row-backed | Phase 4.H rows `Provider account usage read model + renderer` and `Gateway /usage command binding over provider account usage` |
| Prompt/context/compression | `agent/prompt_builder.py`, `agent/context_engine.py`, `agent/context_compressor.py`, `agent/context_references.py`, `agent/manual_compression_feedback.py`, `agent/subdirectory_hints.py` | `internal/hermes`, `internal/kernel`, `internal/transcript`, `internal/skills` | still row-backed | Phase 4.B, 4.C |
| Tool registry and toolsets | `tools/registry.py`, `model_tools.py`, `toolsets.py`, `toolset_distributions.py`, `tests/agent/test_*tool*.py` | `internal/tools`, `internal/doctor`, `internal/cli` | mapped-by-symbol for current manifest; still row-backed for remaining handlers | Phase 2.A, 5.A |
| Tool execution, files, checkpoints, terminals | `tools/code_execution_tool.py`, `tools/file_operations.py`, `tools/file_tools.py`, `tools/checkpoint_manager.py`, `tools/process_registry.py`, `tools/terminal_tool.py` | `internal/tools`, `internal/cmdrunner`, `internal/transcript`, `internal/audit` | still row-backed | Phase 5.K, 5.L |
| Sandbox/environments | `tools/environments/{base,local,docker,ssh,managed_modal,modal,daytona,singularity,file_sync}.py`, `environments/hermes_base_env.py`, `environments/agentic_opd_env.py`, `environments/web_research_env.py`, `environments/tool_call_parsers/{hermes_parser,deepseek_v3_1_parser}.py` plus the remaining parser fixture matrix row | `internal/tools`, `internal/cmdrunner`, `internal/config`, `internal/hermes` parser fixtures | mapped-by-contract; parser fixtures are still row-backed by `Raw tool-call parser fixture matrix` | Phase 5.B, 5.M |
| Browser, web, media, voice, image | `tools/browser_*.py`, `tools/browser_providers/**`, `tools/web_tools.py`, `tools/vision_tools.py`, `tools/image_generation_tool.py`, `tools/tts_tool.py`, `tools/transcription_tools.py`, `tools/voice_mode.py` | `internal/tools`, future browser/media packages, `internal/hermes` content parts | mapped-by-contract | Phase 5.C, 5.D, 5.E |
| Security and approvals | `tools/approval.py`, `tools/path_security.py`, `tools/url_safety.py`, `tools/tirith_security.py`, `tools/website_policy.py`, `tools/osv_check.py`, `agent/redact.py` | `internal/tools`, `internal/doctor`, `internal/config`, `internal/audit` | still row-backed | Phase 5.J, 4.E |
| Operator tools | `tools/todo_tool.py`, `tools/clarify_tool.py`, `tools/send_message_tool.py`, `tools/session_search_tool.py`, `tools/debug_helpers.py`, `tools/interrupt.py`, `tools/memory_tool.py` | `internal/tools`, `internal/sessionsearchtool`, `internal/gateway`, `internal/subagent` | still row-backed | Phase 5.N |
| Subagents/delegation | `tools/delegate_tool.py`, `agent/test_subagent_*.py` | `internal/subagent`, `internal/tools`, `internal/audit` | mapped-by-symbol for deterministic runtime; still row-backed for policy polish | Phase 2.E, 5.J |
| MCP, managed tool gateway, ACP | `mcp_serve.py`, `tools/mcp_tool.py`, `tools/managed_tool_gateway.py`, `tools/mcp_oauth*.py`, `acp_adapter/auth.py:detect_provider`, `acp_adapter/entry.py:main`, `acp_adapter/{events,permissions,server,session,tools}.py`, `acp_registry/agent.json`, `tests/acp/**` | `internal/plugins`, future `internal/acp`, `internal/tools`, `internal/apiserver` | mapped-by-contract | Phase 5.G, 5.H |
| Plugins and memory plugins | `plugins/**`, especially `plugins/memory/honcho/**`, `plugins/google_meet/**`, `plugins/image_gen/**`, `plugins/disk-cleanup/**` | `internal/plugins`, `internal/tools`, `internal/goncho`, `internal/gonchotools` | mapped-by-contract; Goncho memory is owned | Phase 3.G, 5.I |
| Skills and optional skills | `skills/**`, `optional-skills/**`, `agent/skill_*.py`, `tools/skill*_tool.py` | `internal/skills`, `docs/development-skills`, `cmd/gormes`, `internal/hermes` | mapped-by-contract | Phase 5.F, 6.C |
| Gateway runtime and channels | `gateway/run.py`, `gateway/config.py`, `gateway/session*.py`, `gateway/delivery.py`, `gateway/stream_consumer.py`, `gateway/platforms/**`, `tests/gateway/**` | `internal/gateway`, `internal/channels/*`, `internal/store`, `internal/tuigateway` | still row-backed | Phase 2.B-2.F, 7 |
| Cron/scheduling | `cron/scheduler.py`, `cron/jobs.py`, `tools/cronjob_tools.py`, `tests/cron/**` | `internal/cron`, `internal/gateway`, `internal/tools` | mapped-by-symbol for current scheduler core; still row-backed for API/tool parity | Phase 2.D, 5.N |
| API, TUI, dashboard | `gateway/platforms/api_server.py`, `cli.py`, `tui_gateway/**`, `ui-tui/**`, `web/**`, `website/**`, `hermes_cli/curses_ui.py`, `hermes_cli/skin_engine.py`, `hermes_cli/commands.py` | `internal/apiserver`, `internal/tui`, `internal/tuigateway`, `internal/cli`, `www.gormes.ai`, `cmd/gormes` | still row-backed | Phase 5.Q, Phase 1 |
| CLI/config/status/doctor/backup | `cli.py`, `hermes`, `hermes_cli/**`, `tests/cli/**` | `cmd/gormes`, `internal/cli`, `internal/config`, `internal/doctor` | still row-backed | Phase 5.O |
| Packaging/release/install | `Dockerfile`, `docker/entrypoint.sh`, `docker-compose.yml`, `packaging/homebrew/hermes-agent.rb`, `packaging/**`, `flake.*`, `scripts/release.py`, `setup-hermes.sh`, release notes | installers, service units, OCI image, `Makefile`, `cmd/gormes`, `www.gormes.ai` | still row-backed; hosted Python install pieces excluded from final runtime | Phase 5.P |
| Research/batch/RL/datagen modes | `batch_runner.py`, `mini_swe_runner.py`, `rl_cli.py`, `datagen-config-examples/**` | future research packages, `internal/subagent`, `internal/tools` | mapped-by-contract; not core production gate | Phase 5.M, 5.O |

## Honcho Feature Map

| Honcho surface | Upstream evidence | Go implementation target | Classification | Progress/register owner |
|---|---|---|---|---|
| Data model, migrations, CRUD invariants | `src/models.py:MessageEmbedding,Collection,Document,QueueItem,ActiveQueueSession,WebhookEndpoint,SessionPeer`, `src/schemas/**`, `src/crud/{workspace,peer,peer_card,session,message,representation,document,webhook,deriver}.py`, `src/crud/collection.py:collection_cache_key,get_collection,get_or_create_collection,update_collection_internal_metadata`, `migrations/**`, `tests/crud/**` | `internal/goncho`, `internal/store`, `internal/memory`, `internal/session` | still row-backed | Phase 3.F, 3.G |
| Workspaces and API keys | `src/routers/workspaces.py`, `src/routers/keys.py`, `src/security.py`, `docs/v3/openapi.json`, `tests/routes/test_workspaces.py`, `tests/routes/test_keys.py` | `internal/goncho`, `internal/config`, optional `internal/apiserver` adapter | still row-backed; hosted auth is owned local replacement | Phase 3.G, 5.Q |
| Peers, sessions, messages, files | `src/routers/peers.py`, `src/routers/sessions.py`, `src/routers/messages.py`, `src/utils/files.py`, `tests/routes/test_{peers,sessions,messages,files}.py` | `internal/goncho`, `internal/gonchotools`, `internal/session`, `internal/memory` | still row-backed | Phase 3.F, 3.G |
| Conclusions, observations, representations | `src/routers/conclusions.py`, `src/crud/representation.py`, `src/crud/deriver.py`, `tests/routes/test_conclusions.py`, `tests/deriver/**` | `internal/goncho`, `internal/memory`, `internal/gonchotools` | still row-backed | Phase 3.F, 6 |
| Filter/search/context/summaries | `src/utils/filter.py`, `src/utils/search.py`, `src/utils/summarizer.py`, `src/utils/representation.py`, `tests/test_search.py`, `tests/test_advanced_filters.py` | `internal/goncho`, `internal/memory`, `internal/hermes` | still row-backed | Phase 3.F, 4.C |
| Queue/deriver/reconciler lifecycle | `src/deriver/**`, `src/crud/deriver.py`, `src/reconciler/**`, `tests/deriver/**`, `tests/reconciler/**`, `tests/routes/test_queue_status.py` | `internal/goncho`, `internal/cron`, `internal/doctor`, `internal/memory` | still row-backed | Phase 3.F, 6 |
| Dialectic chat and tool loop | `src/dialectic/chat.py`, `src/dialectic/core.py`, `src/dialectic/prompts.py`, `src/llm/tool_loop.py`, `tests/dialectic/**`, `tests/llm/**` | `internal/goncho`, `internal/kernel`, `internal/hermes` | mapped-by-contract | Phase 3.F, 3.G |
| Dreaming | `src/dreamer/dream_scheduler.py`, `src/dreamer/orchestrator.py`, `src/dreamer/specialists.py`, `src/dreamer/surprisal.py`, `src/dreamer/trees/{base,covertree,graph,lsh,prototype,rptree,sklearn_wrapper}.py`, `tests/dreamer/**`, `tests/unified/test_cases/dream_*.json` | `internal/goncho`, `internal/cron`, `internal/memory` | still row-backed | Phase 3.F, 6 |
| Vector stores and embeddings | `src/vector_store/{lancedb,turbopuffer}.py`, `src/reconciler/sync_vectors.py`, `src/embedding_client.py`, `tests/integration/test_message_embeddings.py` | `internal/memory.VectorStoreDivergenceRows`, `internal/memory.semanticSeeds`, SQLite `entity_embeddings` | owned with divergence contract; external LanceDB/Turbopuffer adapters and sync-vector reconciler are explicit excluded rows in Phase 3.F `Vector store + reconciler divergence proof` | Phase 3.D, 3.G |
| Telemetry, reasoning, metrics | `src/telemetry/emitter.py`, `src/telemetry/reasoning_traces.py`, `src/telemetry/metrics_collector.py`, `src/telemetry/sentry.py`, `src/telemetry/events/deletion.py`, `src/telemetry/events/**`, `tests/telemetry/**`, `tests/integration/test_telemetry.py` | `internal/telemetry`, `internal/audit`, `internal/goncho` | still row-backed by `Self-monitoring telemetry`; hosted metrics/exporters are owned local divergence unless later required | Phase 3.F, 4.E |
| Webhooks and keys | `src/routers/{webhooks,keys}.py`, `src/security.py`, `src/crud/webhook.py`, `src/webhooks/events.py`, `src/webhooks/webhook_delivery.py`, `tests/routes/test_{webhooks,keys}.py`, `tests/webhooks/test_webhook_delivery.py` | `internal/goncho.{CreateScopedKey,VerifyScopedKey,GetOrCreateWebhookEndpoint,ListWebhookEndpoints,DeleteWebhookEndpoint,NewTestWebhookEvent,SignWebhookPayload}`, future delivery worker | complete for local key/webhook CRUD/test/signature surface; outbound delivery/retry worker and OpenAPI/SDK route binding remain row-backed | Phase 3.G rows `Goncho keys + webhooks compatibility surface` and `Goncho webhook delivery retry worker contract`; 5.Q |
| Python SDK | `sdks/python/src/honcho/{client,aio,api_types,mixins,base,peer,session,message,conclusions,session_context,pagination,types}.py`, `tests/sdk/**` | Go SDK-style compatibility harness under `internal/goncho`/`internal/gonchotools` | mapped-by-contract | Phase 3.G |
| TypeScript SDK | `sdks/typescript/src/client.ts`, `sdks/typescript/src/validation.ts`, `sdks/typescript/src/http/streaming.ts`, `sdks/typescript/src/**`, `sdks/typescript/__tests__/**`, `tests/sdk_typescript/**` | Go fixture harness for SDK request/response behavior | mapped-by-contract | Phase 3.G |
| Honcho MCP | `mcp/src/tools/{workspace,peers,sessions,conclusions,system}.ts`, `mcp/src/server.ts`, `mcp/instructions.md` | `internal/goncho`, `internal/plugins`, `internal/tools` | mapped-by-contract | Phase 3.G, 5.G |
| Honcho CLI | `honcho-cli/src/honcho_cli/{main,config,output,validation,_help,branding}.py`, `honcho-cli/src/honcho_cli/commands/{workspace,peer,session,message,conclusion,config_cmd,setup}.py`, `honcho-cli/tests/**` | Phase 3.G complete row `Goncho CLI command-tree parity` in `cmd/gormes/goncho_cli_parity.go`; functional command execution, config preservation, exact JSON/table output, destructive confirms, validation, exit codes, and SDK/API binding remain row-backed | mapped-by-contract | Phase 3.G, 5.O |
| Hosted deploy/config | `.env.template`, `config.toml.example`, `Dockerfile`, `docker-compose.yml.example`, `docker/entrypoint.sh`, `docker/prometheus.yml`, `docker/grafana-datasource.yml`, `fly.toml`, `database/init.sql`, `alembic.ini` | local embedded Goncho config, doctor, and release docs | owned/excluded by surface | Phase 3.G, 5.P |
| Docs/examples/unified tests | `docs/v1/**`, `docs/v2/**`, `docs/v3/**`, `examples/**`, `tests/unified/**`, `tests/bench/**`, `tests/live_llm/**` | compatibility fixture donors and docs references | mapped-by-contract; benchmarks are evidence-only unless a row promotes them; live-provider execution under `tests/live_llm/**` is excluded except as a source for fake/local LLM contract rows | Phase 3.G, 6 |

## Go Implementation Order

| Order | Gate | Implementation package target | Why it comes here |
|---|---|---|---|
| 1 | Normal Go turn harness | `internal/kernel`, `internal/hermes`, `internal/tools`, `internal/session`, `internal/memory` | Proves Gormes is replacing Hermes runtime behavior rather than routing around it. |
| 2 | Provider transport contract | `internal/hermes`, `internal/config` | Gives every provider row one stream/event/error vocabulary. |
| 3 | Goncho SDK-style harness | `internal/goncho`, `internal/gonchotools`, `internal/memory` | Proves Honcho compatibility against local storage before normal turns depend on it. |
| 4 | Prompt/context/compression spine | `internal/hermes`, `internal/kernel`, `internal/transcript`, `internal/skills` | Prevents provider/tool rows from inventing their own prompt assembly. |
| 5 | Descriptor-first tool surface | `internal/tools`, `internal/doctor`, `internal/audit` | Lets every handler expose schema, trust, availability, and audit evidence uniformly. |
| 6 | Gateway/API/TUI consumers | `internal/gateway`, `internal/apiserver`, `internal/tui`, `internal/tuigateway`, `internal/channels/*` | UI and channels consume typed events instead of owning agent logic. |
| 7 | CLI/release/packaging | `cmd/gormes`, `internal/cli`, installers, `www.gormes.ai` | Public install and operations promises should follow proven runtime behavior. |
| 8 | Learning/research modes | `internal/skills`, `internal/subagent`, future research packages | Higher-level automation depends on stable turns, tools, memory, and release flow. |

## Nested Feature-Level Coverage Matrix

`TestNestedUpstreamFeatureCoverage` protects these representative nested
feature paths. It must skip cleanly when sibling upstream repos are absent and
fail with `nested_feature_unmapped` when an upstream path exists but no plan,
map, audit, ledger, or progress evidence mentions it.

| Lane | Upstream nested paths | Required evidence |
|---|---|---|
| Hermes runtime/providers | `agent/transports/base.py`, `agent/transports/chat_completions.py`, `agent/transports/codex.py`, `agent/transports/bedrock.py`, `agent/anthropic_adapter.py`, `agent/gemini_native_adapter.py`, `agent/gemini_cloudcode_adapter.py`, `agent/codex_responses_adapter.py`, `agent/moonshot_schema.py`, `agent/model_metadata.py`, `agent/models_dev.py`, `agent/account_usage.py`, `agent/context_compressor.py`, `agent/prompt_builder.py`, `model_tools.py`, `trajectory_compressor.py`, `environments/hermes_base_env.py`, `environments/agentic_opd_env.py`, `environments/web_research_env.py`, `environments/tool_call_parsers/hermes_parser.py`, `environments/tool_call_parsers/deepseek_v3_1_parser.py`, `tests/agent/transports/test_transport.py`, `tests/test_account_usage.py`, `tests/gateway/test_usage_command.py` | Feature map provider/model/context/environment rows, this plan, swarm audit, Phase 4/5 rows. |
| Hermes tools/security/plugins | `tools/registry.py`, `tools/environments/local.py`, `tools/environments/ssh.py`, `tools/yuanbao_tools.py`, `tools/mcp_tool.py`, `tools/managed_tool_gateway.py`, `tools/approval.py`, `tools/url_safety.py`, `tools/browser_providers/firecrawl.py`, `tools/browser_providers/browserbase.py`, `tools/browser_providers/browser_use.py`, `plugins/memory/honcho/client.py`, `plugins/google_meet/tools.py`, `skills/yuanbao/SKILL.md`, `optional-skills/mcp/DESCRIPTION.md` | Tool/plugin/skill map rows, swarm audit gaps, Phase 5 rows. |
| Hermes gateway/CLI/release | `gateway/run.py`, `gateway/channel_directory.py`, `gateway/stream_consumer.py`, `gateway/platforms/api_server.py`, `gateway/platforms/telegram.py`, `gateway/platforms/discord.py`, `cron/scheduler.py`, `hermes_cli/main.py`, `hermes_cli/auth.py`, `hermes_cli/plugins.py`, `hermes_cli/plugins_cmd.py`, `hermes_cli/web_server.py`, `scripts/release.py`, `Dockerfile`, `docker/entrypoint.sh`, `packaging/homebrew/hermes-agent.rb`, `flake.nix` | Gateway/CLI/release map rows, Phase 2/5/7 rows, release divergence notes. |
| Hermes ACP/MCP extensions | `acp_adapter/auth.py`, `acp_adapter/entry.py`, `acp_adapter/server.py`, `acp_adapter/session.py`, `acp_adapter/tools.py`, `acp_adapter/permissions.py`, `acp_registry/agent.json`, `mcp_serve.py` | ACP/MCP map rows, swarm audit rows, Phase 5.G/5.H rows. |
| Honcho core/Goncho | `src/models.py:MessageEmbedding`, `src/models.py:Collection`, `src/models.py:Document`, `src/models.py:QueueItem`, `src/models.py:ActiveQueueSession`, `src/models.py:WebhookEndpoint`, `src/models.py:SessionPeer`, `src/routers/workspaces.py`, `src/routers/peers.py`, `src/routers/sessions.py`, `src/routers/messages.py`, `src/routers/conclusions.py`, `src/routers/keys.py`, `src/routers/webhooks.py`, `src/crud/workspace.py`, `src/crud/peer.py`, `src/crud/peer_card.py`, `src/crud/session.py`, `src/crud/message.py`, `src/crud/document.py`, `src/crud/webhook.py`, `src/crud/deriver.py`, `src/crud/collection.py:collection_cache_key,get_collection,get_or_create_collection,update_collection_internal_metadata`, `src/crud/representation.py`, `src/deriver/queue_manager.py`, `src/dialectic/chat.py`, `src/dialectic/core.py`, `src/dialectic/prompts.py`, `src/dreamer/dream_scheduler.py`, `src/dreamer/surprisal.py`, `src/dreamer/specialists.py`, `src/dreamer/orchestrator.py`, `src/dreamer/trees/base.py`, `src/dreamer/trees/covertree.py`, `src/dreamer/trees/graph.py`, `src/dreamer/trees/lsh.py`, `src/dreamer/trees/prototype.py`, `src/dreamer/trees/rptree.py`, `src/dreamer/trees/sklearn_wrapper.py`, `src/webhooks/events.py`, `src/webhooks/webhook_delivery.py`, `src/telemetry/reasoning_traces.py`, `src/telemetry/metrics_collector.py`, `src/telemetry/sentry.py`, `src/telemetry/events/deletion.py`, `src/vector_store/lancedb.py`, `database/init.sql` | Honcho feature rows, Goncho plan rows, owned divergence rows. |
| Honcho SDK/MCP/CLI/deploy/tests | `docs/v3/openapi.json`, `sdks/python/src/honcho/client.py`, `sdks/python/src/honcho/aio.py`, `sdks/python/src/honcho/api_types.py`, `sdks/python/src/honcho/mixins.py`, `sdks/typescript/src/client.ts`, `sdks/typescript/src/validation.ts`, `sdks/typescript/src/http/streaming.ts`, `sdks/typescript/__tests__/streaming.test.ts`, `mcp/src/tools/sessions.ts`, `mcp/src/tools/conclusions.ts`, `honcho-cli/src/honcho_cli/main.py`, `honcho-cli/src/honcho_cli/config.py`, `honcho-cli/src/honcho_cli/output.py`, `honcho-cli/src/honcho_cli/validation.py`, `honcho-cli/src/honcho_cli/_help.py`, `honcho-cli/src/honcho_cli/branding.py`, `honcho-cli/src/honcho_cli/commands/workspace.py`, `honcho-cli/src/honcho_cli/commands/peer.py`, `honcho-cli/src/honcho_cli/commands/session.py`, `honcho-cli/src/honcho_cli/commands/message.py`, `honcho-cli/src/honcho_cli/commands/conclusion.py`, `honcho-cli/src/honcho_cli/commands/config_cmd.py`, `honcho-cli/src/honcho_cli/commands/setup.py`, `honcho-cli/tests/test_commands.py`, `tests/sdk/test_client.py`, `tests/routes/test_webhooks.py`, `tests/webhooks/test_webhook_delivery.py`, `tests/live_llm/README.md`, `.env.template`, `config.toml.example`, `Dockerfile`, `docker-compose.yml.example`, `docker/entrypoint.sh`, `docker/prometheus.yml`, `docker/grafana-datasource.yml`, `fly.toml` | SDK/MCP/CLI/deploy rows, compatibility harness rows, owned hosted-service divergence, and explicit `tests/live_llm` exclusion for live-provider-only execution. |

## Swarm Gap Matrix

| Gap class | Classification | Required implementation plan |
|---|---|---|
| Provider transport, provider-specific schemas, OAuth, credential/rate/cache/account-usage/error helpers | still row-backed | Split Phase 4.A/4.G/4.H into transcript-backed transport, auth, account-usage read-model, rate/cache, and error rows before broad provider ports. |
| Context compression, prompt builder, skill prompt injection | still row-backed | Split Phase 4.B/4.C rows into budget, protected context, references, manual feedback, project-instruction, and skill-snapshot fixtures. |
| Tool handlers, sandbox adapters, browser/media/security/operator tools | mapped-by-contract, still row-backed | Keep descriptor/trust/audit registry first, then implement category handlers with fake backends and path/URL/approval tests. |
| ACP/MCP/plugin/skill catalogs | mapped-by-contract | Add explicit descriptor/catalog rows before live adapters; Goncho memory plugin is an owned in-process replacement. |
| Gateway/channel/bootstrap/cron/API/TUI | still row-backed | Keep kernel event stream authoritative; port channel bootstrap and CLI/API management as consumers with fake adapters. |
| CLI/config/doctor/install/release | still row-backed | Split command groups by side effect and preserve noninteractive safety, profile/auth/config, status/log/backup, service/release proofs, and explicit Hermes/OpenClaw migration commands. |
| Honcho CRUD/API/OpenAPI/SDK/MCP/CLI | mapped-by-contract, still row-backed | Use SDK/OpenAPI/route fixtures to prove local Goncho compatibility; unsupported hosted fields require explicit evidence. |
| Honcho vector/deploy divergence | owned/excluded | SQLite/FTS/graph local memory replaces hosted vector stores unless a row proves adapter need; FastAPI/Postgres/Redis/Fly/Alembic deployment is excluded from the local runtime. |

## Row Split Candidates From Swarm

These candidates are not a parallel queue. They are the row-splitting checklist
for existing phase anchors when a planner or builder pass needs to make a broad
row executable.

| Candidate row | Source refs | Go target | Parent anchor |
|---|---|---|---|
| Provider transport abstraction | `../hermes-agent/agent/transports/{base,types,chat_completions,anthropic,bedrock,codex}.py`, `../hermes-agent/tests/agent/transports/**` | `internal/hermes.ProviderTransport`, normalized stream/event fixtures for Chat-Completions, Anthropic Messages, Bedrock Converse, and Codex Responses | Phase 4.A |
| Moonshot/Kimi schema sanitizer | `../hermes-agent/agent/moonshot_schema.py`, `../hermes-agent/tests/agent/test_moonshot_schema.py`, `../hermes-agent/tests/agent/transports/test_chat_completions.py` | `internal/hermes.{IsMoonshotModel,SanitizeToolSchemaForModel,SanitizeMoonshotToolParameters,SanitizeMoonshotToolDescriptors}` plus OpenAI-compatible request-builder wiring | Phase 4.A complete row `Moonshot/Kimi tool-schema sanitizer` |
| Provider fallback chain contract | `../hermes-agent/agent/retry_utils.py`, `agent/error_classifier.py`, `agent/rate_limit_tracker.py`, `tests/agent/test_unsupported_parameter_retry.py` | `internal/kernel`, `internal/hermes` | Phase 4.H |
| Token vault and credential-source precedence | `../hermes-agent/agent/credential_pool.py`, `agent/credential_sources.py`, `hermes_cli/auth.py` | `internal/config`, `internal/hermes`, `internal/cli` | Phase 4.G, 5.O |
| Prompt cache and rate guard evidence | `../hermes-agent/agent/prompt_caching.py`, `agent/nous_rate_guard.py`, `tests/agent/test_prompt_caching.py` | `internal/hermes`, `internal/telemetry` | Phase 4.H |
| Provider account usage reporting | `../hermes-agent/agent/account_usage.py:render_account_usage_lines`, `../hermes-agent/agent/account_usage.py:fetch_account_usage`, `../hermes-agent/cli.py` `/usage`, `../hermes-agent/gateway/run.py:_handle_usage_command`, `../hermes-agent/tests/test_account_usage.py`, `../hermes-agent/tests/gateway/test_usage_command.py` | `internal/hermes`, `internal/telemetry`, `internal/gateway`, `cmd/gormes` | Phase 4.H rows `Provider account usage read model + renderer` and `Gateway /usage command binding over provider account usage` |
| Context references stable-handle store | `../hermes-agent/agent/context_references.py`, `../hermes-agent/tests/agent/test_context_references.py` | `internal/hermes.ParseContextReferences`, `internal/hermes.AttachContextReferenceHandles`, `internal/contextrefs.Store`, `internal/transcript` aliases | Phase 4.B complete row `Context references stable-handle store`; live file/folder/git/url expansion remains row-backed |
| Context compression internals | `../hermes-agent/agent/context_compressor.py`, `manual_compression_feedback.py`, `context_references.py` | `internal/hermes`, `internal/kernel`, `internal/transcript` | Phase 4.B still row-backed for file/folder/git/url expansion, injection budget warnings, manual feedback, and remaining compression behavior |
| Prompt builder and skill prompt snapshots | `../hermes-agent/agent/prompt_builder.py`, `skill_preprocessing.py`, `skill_commands.py` | `internal/hermes`, `internal/skills`, `internal/cli` | Phase 4.C, 5.F |
| Telegram fallback network transport parity | `../hermes-agent/gateway/platforms/telegram_network.py` | `internal/channels/telegram`, `internal/config` | Phase 2.B/7 |
| Yuanbao tool parity manifest refresh | `../hermes-agent/tools/yuanbao_tools.py`, `../hermes-agent/toolsets.py`, `../hermes-agent/gateway/platforms/yuanbao*.py` | `internal/tools/testdata/upstream_tool_parity_manifest.json`, `internal/channels/yuanbao` | Phase 5.A, 7.E |
| Raw tool-call parser fixture matrix | `../hermes-agent/environments/tool_call_parsers/*.py` | `internal/hermes` parser/repair fixtures or a dedicated parser package | Phase 4.A/5.M |
| ACP server/session/tools/permissions matrix | `../hermes-agent/acp_adapter/{auth,entry,events,permissions,server,session,tools}.py`, `../hermes-agent/tests/acp/**` | future `internal/acp`, `internal/plugins`, `internal/apiserver` | Phase 5.H |
| MCP bridge/runtime matrix | `../hermes-agent/mcp_serve.py`, `tools/mcp_tool.py`, `tools/managed_tool_gateway.py`, `tools/mcp_oauth*.py` | `internal/tools`, `internal/plugins`, `internal/apiserver` | Phase 5.G |
| Hermes prompt_toolkit TUI parity | `../hermes-agent/cli.py:HermesCLI.run,_get_tui_prompt_fragments,_get_status_bar_fragments,_render_spinner_text,_get_approval_display_fragments`, `../hermes-agent/hermes_cli/skin_engine.py`, `../hermes-agent/hermes_cli/commands.py`, `../hermes-agent/tests/cli/test_cli_status_bar.py`, `../hermes-agent/tests/cli/test_tool_progress_scrollback.py`, `../hermes-agent/tests/cli/test_cli_approval_ui.py` | `internal/tui`, `internal/cli`, `cmd/gormes` | Phase 5.Q rows `Native TUI Hermes skin token renderer`, `Native TUI Hermes status bar renderer`, `Native TUI Hermes bottom-pinned chrome layout`, `Native TUI Hermes input keybinding semantics`, `Native TUI Hermes slash completion helpers`, and `Native TUI Hermes tool progress + modal panel renderers` |
| Hermes CLI command-tree parity manifest | `../hermes-agent/hermes_cli/main.py`, `../hermes-agent/hermes_cli/commands.py:GATEWAY_KNOWN_COMMANDS`, `../hermes-agent/gateway/run.py:_handle_status_command,_handle_restart_command,_handle_reset_command,_handle_help_command,_handle_model_command,_handle_profile_command,_handle_update_command,_handle_approve_command,_handle_deny_command,_handle_voice_command,_handle_usage_command`, `../hermes-agent/plugins/memory/__init__.py:discover_plugin_cli_commands`, `../hermes-agent/plugins/disk-cleanup/__init__.py:PluginContext.register_command`, `cmd/gormes/main.go`, `internal/cli/command_registry.go` | `cmd/gormes/hermes_cli_parity.go`, `cmd/gormes/hermes_cli_parity_test.go` | Phase 5.O row `Hermes CLI command-tree parity manifest`; classifies every top-level command, nested command, static slash command, dynamic plugin command, alias, gateway handler, implemented Hermes `-z/--oneshot` flag, and Gormes-owned addition before handler ports |
| Provider endpoint/API-key root flags + runtime resolution | `cmd/gormes/main.go`, `internal/config/config.go`, `../hermes-agent/hermes_cli/config.py`, `../hermes-agent/tests/hermes_cli/test_set_config_value.py` | `cmd/gormes`, `internal/config` | Phase 5.O row `Provider endpoint/API-key root flags + runtime resolution` |
| Gormes config command surface | `../hermes-agent/hermes_cli/config.py`, `../hermes-agent/tests/hermes_cli/test_set_config_value.py`, `../hermes-agent/tests/hermes_cli/test_config_env_expansion.py`, `internal/config/config.go`, `internal/config/dotenv.go` | `cmd/gormes config`, `internal/config` writer helpers | Phase 5.O row `Gormes config command surface` |
| Gormes config edit/check/native schema-migrate closeout | `../hermes-agent/hermes_cli/config.py:edit_config,check_config_version,validate_config_structure,migrate_config`, `../hermes-agent/tests/hermes_cli/test_config_validation.py`, `internal/config/config.go:CurrentConfigVersion,migrateConfig` | `cmd/gormes config`, `internal/config` check/migrate helpers | Phase 5.O row `Gormes config edit/check/native schema-migrate closeout`; distinct from cross-product `gormes migrate hermes/openclaw` imports |
| Hermes state/config migration command | `../hermes-agent/hermes_constants.py`, `hermes_state.py`, `hermes_cli/config.py`, `tests/hermes_cli/test_config*.py` | `cmd/gormes migrate hermes`, `internal/migrate/hermes`, `internal/config`, `internal/session`, `internal/memory` | Phase 5.O rows `Hermes config migration dry-run manifest` and `Hermes config migration writer` |
| OpenClaw migration command | `../hermes-agent/hermes_cli/claw.py`, `../hermes-agent/optional-skills/migration/openclaw-migration/scripts/openclaw_to_hermes.py`, `../hermes-agent/website/docs/guides/migrate-from-openclaw.md` | `cmd/gormes migrate openclaw`, `internal/migrate/openclaw`, `internal/config`, `internal/skills`, `internal/memory` | Phase 5.O rows `OpenClaw migration dry-run manifest` and `OpenClaw migration writer and cleanup command`; `gormes migrate ooenclaw` is a typo-suggestion path, not an import alias |
| Product gateway service lifecycle | `../hermes-agent/gateway/status.py`, `gateway/restart.py`, `hermes_cli/service`-style command behavior | `cmd/gormes`, `internal/gateway`, `internal/cli` | Phase 2.F, 5.O, 5.P |
| Gormes logs command and failure envelope | `../hermes-agent/hermes_logging.py`, `hermes_cli/logs.py`, CLI tests | `cmd/gormes`, `internal/cli`, `internal/audit` | Phase 5.O |
| Trajectory compressor core and compressed evidence lineage | `../hermes-agent/trajectory_compressor.py`, `../hermes-agent/tests/test_trajectory_compressor.py` | `internal/transcript`, `internal/audit` | Phase 4.E complete row `Trajectory compressor + compressed-evidence lineage` |
| Release packaging divergence matrix | `../hermes-agent/scripts/release.py`, `Dockerfile`, `docker/**`, `packaging/**`, `flake.*`, `.github/workflows/**` | installers, service units, OCI/Homebrew/Nix docs, `cmd/repoctl` | Phase 5.P |
| Supply-chain and skills-index workflow parity | `../hermes-agent/.github/workflows/supply-chain-audit.yml`, `skills-index.yml` | docs/CI contract or `cmd/repoctl` | Phase 5.P, 6 |
| Goncho OpenAPI v3 route manifest and unsupported-route contract | `../honcho/docs/v3/openapi.json`, `../honcho/src/routers/*.py`, `../honcho/tests/routes/**` | Phase 3.G complete row `Goncho HTTP route parity over OpenAPI v3` in `internal/goncho/http`; handler binding, auth, validation, pagination, response envelopes, status codes, and SDK route e2e remain row-backed | Phase 3.G, 5.Q |
| Goncho Honcho CRUD lifecycle invariant matrix | `../honcho/src/crud/{workspace,peer,session,message,document}.py`, `../honcho/tests/crud/**`, `../honcho/tests/routes/**` | `internal/goncho.Service.{CreateMessages,DeleteSession,DeleteWorkspace}` plus local SQLite lifecycle metadata; remaining document/API/SDK fixtures stay row-backed | Phase 3.F complete row `Goncho CRUD lifecycle invariants`, Phase 3.G for route/SDK compatibility |
| Honcho API validation limits and metadata sanitizer | `../honcho/src/schemas/api.py`, `../honcho/tests/test_schema_validations.py` | `internal/goncho/compat`, `internal/config` | Phase 3.G |
| Goncho effective config hierarchy fixtures | `../honcho/src/schemas/configuration.py`, `../honcho/tests/unified/test_cases/config_*.json` | `internal/config.GonchoCfg`, `internal/goncho.Config` | Phase 3.F, 3.G |
| Goncho deriver queue lifecycle and ownership contract | `../honcho/src/deriver/{queue_manager,enqueue,consumer,deriver}.py`, `../honcho/tests/deriver/**` | `internal/goncho.WorkQueue`, `internal/doctor` | Phase 3.F, 6 |
| Goncho observation document tree semantics | `../honcho/src/crud/document.py`, `../honcho/src/dreamer/trees/*.py`, `../honcho/tests/unified/test_cases/observation_*.json` | `internal/goncho.ConclusionStore`, `internal/memory` | Phase 3.F |
| Goncho model config and provider bridge contract | `../honcho/src/llm/**`, `../honcho/tests/llm/**` | `internal/goncho`, `internal/hermes` bridge or owned deterministic local engine | Phase 3.G, 4.A |
| Goncho dialectic and dream fake-provider execution | `../honcho/src/dialectic/**`, `../honcho/src/dreamer/**`, `../honcho/tests/dialectic/**`, `../honcho/tests/dreamer/**` | `internal/goncho.ChatEngine`, `internal/goncho.DreamRunner`, `internal/cron` | Phase 3.F, 6 |
| Goncho MCP catalog parity matrix | `../honcho/mcp/src/tools/{workspace,peers,sessions,conclusions,system}.ts`, `../honcho/mcp/README.md` | `internal/gonchotools.HonchoMCPToolCatalog` plus existing Goncho descriptors and field-level partial compatibility evidence | Phase 5.G complete row `Goncho MCP tool catalog` |
| Python and TypeScript SDK compatibility fixture matrices | `../honcho/sdks/python/src/honcho/**`, `../honcho/sdks/typescript/src/**`, `../honcho/tests/sdk/**`, `../honcho/tests/sdk_typescript/**`, `../honcho/sdks/typescript/__tests__/**` | `internal/goncho/compat/testdata` | Phase 3.G |
| Goncho Honcho CLI compatibility matrix | `../honcho/honcho-cli/src/honcho_cli/**`, `../honcho/honcho-cli/tests/**` | Phase 3.G complete row `Goncho CLI command-tree parity` covers command paths, global flags, and shared scope flags; execution rows remain for JSON/table output, config persistence, confirms, validation, exit codes, and SDK/API actions | Phase 3.G, 5.O |
| Honcho webhooks, keys, telemetry, and deploy divergence matrices | `../honcho/src/routers/{webhooks,keys}.py`, `../honcho/src/webhooks/webhook_delivery.py`, `../honcho/src/telemetry/reasoning_traces.py`, `../honcho/docker-compose.yml.example`, `../honcho/migrations/**`, `../honcho/fly.toml` | Phase 3.G complete row `Goncho keys + webhooks compatibility surface` covers scoped JWTs, webhook endpoint CRUD, test events, and HMAC signatures; `Goncho webhook delivery retry worker contract` covers outbound delivery/retry planning; `Self-monitoring telemetry` covers reasoning traces; route binding, SDK compatibility, and deploy divergence stay row-backed | Phase 3.G, 4.E, 5.Q, 5.P |
| Honcho integration guide and example fixture import | `../honcho/docs/v3/guides/**`, `../honcho/examples/**`, `../honcho/tests/unified/**` | `internal/goncho/compat/testdata`, docs examples | Phase 3.G, 6 |
| GBrain progress-json and abort contract | `../gbrain/docs/progress-events.md`, `../gbrain/src/core/backoff.ts`, `../gbrain/src/core/minions/queue.ts`, `../gbrain/src/core/errors.ts` | `cmd/gormes`, `internal/audit`, `internal/doctor` | Phase 2.E, 5.O, 6 |

## Progress Register Actions

| Action | Status |
|---|---|
| Keep `Hermes and Honcho feature parity map to Go implementation plan` as the parent planning row, with this document added as the canonical fixture. | complete |
| Promote `Nested feature-level coverage test matrix for swarm gaps` from draft to validated once `TestNestedUpstreamFeatureCoverage` lands. | complete |
| Keep feature implementation gaps in existing Phase 2/3/4/5/6/7 rows; split only when a builder selects the surface or a row lacks source refs, write scope, tests, acceptance, or done signal. | ongoing |
| Add no private TODO files. Any newly discovered runtime gap must update `progress.json`, the feature map, the swarm audit, and the nested coverage test in the same planner pass. | standing rule |
| `Provider account usage read model + renderer` and `Gateway /usage command binding over provider account usage` split the `agent/account_usage.py`, `cli.py`, and `gateway/run.py` usage feature gap into read-model/rendering and command-binding builder slices. | complete |
| `Goncho webhook delivery retry worker contract` and enriched `Self-monitoring telemetry` rows close the Honcho `src/webhooks/webhook_delivery.py` and `src/telemetry/reasoning_traces.py` row-backed classification gaps found by the 2026-04-28 Honcho mapper pass. | complete |

## Gormes-Native Owned Surfaces

The 2026-04-27 runtime auditor pass produced a small register of Go packages
that have no upstream parallel. They are owned-by-design and must not appear
as `unknown/gap` again. Each entry names the replacement evidence.

| Gormes path | Replacement evidence | Why owned |
|---|---|---|
| `internal/builderloop/**` | `internal/builderloop/{audit,backend,claims,companions,config,failures,ledger,lifecycle,promote,run_health,runner,ship_ratio,speculative,watchdog,worktree}_test.go` | Skill-driven delivery infrastructure replaces the deleted autonomous loop binary; AGENTS.md ratifies it. |
| `internal/plannerloop/**` | `internal/plannerloop/{lifecycle,service,health_preservation,divergence_lifecycle}_test.go` | Same; planner orchestration library used by skills. |
| `internal/plannertriggers/**` | `internal/plannertriggers/{triggers,triggers_concurrent}_test.go` | Trigger event journal feeding the planner library. |
| `internal/progress/**`, `internal/progressctl/**`, `cmd/progress` | `internal/progress/{progress,validate,health,markers,derive}_test.go` | `progress.json` schema, validators, derived stats; `cmd/progress write/validate` is the supported entrypoint. |
| `internal/repoctl/**`, `cmd/repoctl` | `internal/repoctl/{bench,makefile,readme}_test.go` | Repo bench/README hygiene CLI. |
| `internal/store/**` | `internal/store/{store,recording}_test.go` | Command/ack replay store backing kernel + gateway durability. |
| `internal/audit/jsonl.go` | `internal/audit/jsonl_test.go` | JSONL audit recorder; broader prompt-visible redaction (per `agent/redact.py`) is a still row-backed split target captured in this plan and the swarm audit. |
| `internal/pybridge/**` | `internal/pybridge/pybridge.go:NoRuntime,ErrNotImplemented`; no runtime caller should depend on it for Hermes parity | Excluded transition seam. Gormes' completion definition is Hermes-in-Go, so Python interop is not a runtime parity target; a later cleanup row may delete the dormant package. |

## Gormes-Native Pending Decisions

The same auditor pass surfaced Gormes-native package-shape decisions. These
are not upstream Hermes/Honcho `unknown/gap` items; they are cleanup decisions
inside the Go port.

| Surface | Symptom | Required decision |
|---|---|---|
| `internal/discord` and `internal/channels/discord` | Both packages target `gateway/platforms/discord.py` (legacy bot vs chassis). The feature map only references one. | Pick one canonical Discord package and either delete the other or rename it as an owned divergence (legacy adapter retained for transition). |
| `internal/pybridge` | Dormant Python-interop seam with `NoRuntime` placeholder; not invoked anywhere. AGENTS.md says Gormes is complete only when it is Hermes in Go. | Excluded as a runtime parity target in the owned/excluded table above; future builder cleanup may delete it. |

These are recorded so they cannot quietly survive future audits as
`unknown/gap`.

## Hermes Hidden-Risk Register (2026-04-27)

The Hermes mapper pass listed eleven hidden risks worth pinning into the
plan so subsequent planner passes preserve them:

| Hidden risk | Why it matters | Where it lives |
|---|---|---|
| `tinker-atropos/` empty submodule placeholder | Wires future RL training upstream via `.gitmodules`; skipping it skips RL parity. | Phase 5.M owned/excluded; ledger row records it. |
| `environments/` (repo root) RL/eval framework with 9 tool_call_parsers, 3 benchmarks, 2 env subdirs | Distinct from `tools/environments/` (sandboxes). Easy name collision; both are feature-bearing. | Phase 5.M; runtime plan order #8 (research modes). |
| `gateway/builtin_hooks/boot_md.py` | Always-on agent-spawning hook keyed on `~/.hermes/BOOT.md`; not just config glue. | Phase 2.F hooks/boot rows. |
| `gateway/restart.py` exit code 75 convention | systemd, NixOS modules rely on exit 75 for graceful restart. | Phase 2.F restart rows; `internal/gateway/restart.go` already encodes this. |
| `gateway/whatsapp_identity.py` JID collapse | Collapses `@lid` and `@s.whatsapp.net`; required for WhatsApp parity. | Phase 2.B.4 WhatsApp rows; `internal/channels/whatsapp/identity.go` already encodes this. |
| `cli-config.yaml.example` (51 KB) canonical config schema | Spec of record for `hermes_cli/config.py`; ledger now records it. | Phase 5.O; ledger row records it. |
| `RELEASE_v0.11.0.md` (43 KB recent feature deltas) | Most recent upgrade notes; should be diffed against any partial Gormes coverage. | Phase 5.P; existing `RELEASE_*` ignored by source-class test (intentional release-note exclusion). |
| `acp_registry/agent.json` public ACP discovery manifest | Without porting it Gormes cannot register itself in editor ACP catalogs. | Phase 5.H ACP rows. |
| `docker/entrypoint.sh` PID-1 reaping | Tested upstream; container parity depends on it. | Phase 5.P packaging rows. |
| `plugins/example-dashboard/` and `plugins/strike-freedom-cockpit/` pre-built dashboard JS | `web/src/plugins/registry.ts` mounts them; non-trivial plugin SPA contract. | Phase 5.I plugin rows. |
| `skills/index-cache/*.json` (anthropics_skills, claude_marketplace, lobehub, openai_skills) | Skill discovery requires either these or live equivalents. | Phase 5.F skills hub rows. |
