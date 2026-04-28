---
title: "Swarm Feature Parity Audit"
weight: 21
---

# Swarm Feature Parity Audit

This page records the 2026-04-28 sub-agent swarm audit for the question:
**how do we know all Hermes and Honcho features are mapped into Gormes?**

The answer has three layers:

1. **Source-class coverage is executable.** `Upstream Coverage Ledger` plus
   `go test ./docs -run TestUpstreamCoverageLedgerMatchesSourceClasses -count=1`
   proves every feature-like top-level upstream source class is represented or
   explicitly ignored as repo hygiene.
2. **Raw feature-level gaps are recorded here.** Five read-only parity agents
   audited separate Hermes/Honcho surfaces and found broad map coverage plus
   missing or vague feature-level rows. Those gaps are mapped below to Go
   targets and progress actions.
3. **Second-wave reconciliation is canonical.**
   [Hermes/Honcho To Gormes Go Runtime Plan](../hermes-honcho-go-runtime-plan/)
   classifies every broad subsystem as `mapped-by-symbol`,
   `mapped-by-contract`, `owned`, `excluded`, or `still row-backed`. No
   subsystem-level `unknown/gap` remains after that pass.

Do not claim implementation parity from this page. You may claim feature
mapping completeness only when the source-class test passes, the runtime plan
has no `unknown/gap` subsystem, and `TestNestedUpstreamFeatureCoverage` passes
against available sibling upstream checkouts.

## 2026-04-27 Seven-Agent Pass

A second overnight reconciliation re-ran the audit as seven independent
specialists (Hermes mapper, Honcho mapper, Go architecture planner,
runtime/contracts auditor, test matrix planner, docs/register maintainer,
gap classifier). Each specialist worked independently and reported back; the
findings landed verbatim into the surfaces below. The five-lane register
beneath this section is the lane-level view of the same investigation.

| Specialist | Scope | Outputs landed in | Verdict |
|---|---|---|---|
| Hermes mapper (#1) | Full Hermes feature inventory of `run_agent.py`, `agent/**`, `agent/transports/**`, `tools/**`, `environments/**`, `gateway/**`, `cron/**`, `hermes_cli/**`, `mcp_serve.py`, `acp_adapter/**`, `plugins/**`, `skills/**`, `optional-skills/**`, `tui_gateway/**`, `web/**`, `website/**`, packaging | feature map provider/tool/gateway rows, this register, runtime plan nested matrix | 200-feature taxonomy with 11 hidden risks (notably `environments/` vs `tools/environments/` name collision, `gateway/restart.py` exit-code-75 contract, `gateway/whatsapp_identity.py` JID collapse, `cli-config.yaml.example` 51 KB schema, dashboard plugin SPA contract, `skills/index-cache/*.json` four third-party indexes, `gateway/builtin_hooks/boot_md.py` agent-spawning startup hook). |
| Honcho mapper (#2) | Full Honcho feature inventory of `src/**`, `migrations/**`, `database/**`, `docs/v1..v3/**`, `sdks/**`, `mcp/**`, `honcho-cli/**`, `examples/**`, `tests/**`, deploy/config files | feature map Goncho rows, this register, runtime plan nested matrix | 169-feature taxonomy plus a 30-tool MCP catalog, full v3 endpoint matrix (44 endpoints), 25-migration timeline, dreamer surprisal-tree taxonomy (rptree/kdtree/balltree/covertree/lsh/graph/prototype), and explicit `database/init.sql` (`CREATE EXTENSION vector`) deploy divergence. |
| Go architecture planner (#3) | Go-native package boundaries and deep-module interfaces for 30 subsystems | feature map Go architecture target table, runtime plan classifications | Deep-module score per subsystem; flagged eight risk-of-shallow areas (provider transport, prompt builder, operator tools, channel adapters, API server, plugins SDK, observability, packaging) with widening recommendations. |
| Runtime/contracts auditor (#4) | Six-bucket classification of every `internal/**`, `cmd/**`, `www.gormes.ai/**` package | runtime plan classification column, this register's classification column | 50+ packages classified; flagged duplicate `internal/discord` vs `internal/channels/discord` (both target `gateway/platforms/discord.py`), dormant `internal/pybridge`, thin `internal/telemetry` vs Honcho's multi-file `src/telemetry/**`, and missing operator-tool symbols in Phase 5.N. |
| Test matrix planner (#5) | Nested feature-level coverage test matrix and fixture donor map | `TestNestedUpstreamFeatureCoverage` matrix, fixture donor recommendations | Identified 18 nested-glob registries needed; ~150 missing Goncho coverage rows (router endpoints × success/empty/auth/pagination, CRUD invariants, MCP tools, SDK matrices, honcho-cli commands); 3 missing Go channel packages (matrix, mattermost, yuanbao); top-10 broad-glob risk register. |
| Docs/register maintainer (#6) | Edit plan for the six target planning docs (this page, feature map, ledger, parity-map, progress-row contract, upstream coverage test) | this section, ledger rows for `cli-config.yaml.example`/`constraints-termux.txt`/`tinker-atropos`/`database/init.sql`, test extensions for `.sql`/`.txt`/`.example` source detection, `assets/` ignored entries | Seven new ledger entries; test source-extension expansion landed; six runtime-plan classifications stay accurate after detection extends. |
| Gap classifier (#7) | Seven-bucket classification of every Hermes/Honcho subsystem and every progress.json subphase, plus phase counts | this register's classification, runtime plan splits, `Row Split Candidates From Swarm` table | 337 complete · 146 planned · 31 porting · 2 owned · 2 converged across `progress.json`; 33 enumerated unknown/gap items each with sub-agent task brief; 10 builder-ready row proposals; ~16 umbrella rows still need splitting (3 in Phase 4, 13 in Phase 5). |

The seven-agent pass re-confirmed the 2026-04-28 verdict at higher
resolution: source-class coverage stays executable, no subsystem-level
`unknown/gap` remains, and the feature-level gap register below carries the
remaining row-split work into existing phase anchors.

## 2026-04-28 Swarm Lanes (Five Lanes)

| Lane | Upstream scope | Verdict |
|---|---|---|
| Hermes runtime/providers | `run_agent.py`, `agent/**`, `environments/agent_loop.py`, `trajectory_compressor.py` | Source-class covered; feature-level transport, trajectory, Moonshot, OAuth, compression, prompt, and credential rows need sharper anchors. |
| Hermes tools/security/plugins | `tools/**`, `plugins/**`, `skills/**`, `optional-skills/**`, `acp_adapter/**`, `mcp_serve.py` | Source-class covered; ACP/MCP/plugins/skills/sandbox/browser/security/operator-tool coverage is too broad. |
| Hermes gateway/CLI/release | `gateway/**`, `cron/**`, `hermes_cli/**`, `tui_gateway/**`, `web/**`, `website/**`, packaging/deploy files | Source-class covered; gateway operator, dashboard management, CLI umbrella, channel bootstrap, and release rows need splits. |
| Honcho core/Goncho | `src/**`, migrations, routers, CRUD, dialectic, deriver, dreamer, vector, telemetry, webhooks | Source-class covered; API, CRUD invariants, queue, document tree, dialectic/dream execution, webhooks, telemetry, and divergence rows are missing or vague. |
| Honcho SDK/docs/MCP/CLI/deploy | `docs/**`, `sdks/**`, `mcp/**`, `honcho-cli/**`, `examples/**`, `tests/**`, deploy/config files | Source-class covered; SDK refs are stale and OpenAPI/MCP/CLI/deploy/example/test fixture rows are missing. |

## Canonical Surfaces And Ownership

| Surface | Canonical file | Classification |
|---|---|---|
| Source-class inventory | [Upstream Coverage Ledger](../upstream-coverage-ledger/) | mapped-by-symbol at top-level source-class granularity |
| Feature-family map | [Hermes And Honcho Feature Map](../hermes-honcho-feature-map/) | mapped-by-contract |
| Reconciled implementation plan | [Hermes/Honcho To Gormes Go Runtime Plan](../hermes-honcho-go-runtime-plan/) | owned planning surface for Go package targets, classifications, dependency order, and row split candidates |
| Executable nested coverage gate | `docs/upstream_coverage_test.go::TestNestedUpstreamFeatureCoverage` | mapped-by-symbol for representative nested upstream paths |
| Implementation backlog | `docs/content/building-gormes/architecture_plan/progress.json` | owned canonical register; no side backlog |

## Feature-Level Gap Register

Every gap below must resolve to one of three states before implementation:

1. an exact builder-ready `progress.json` row;
2. an `owned` or `excluded` decision in the runtime plan with replacement proof;
3. a nested coverage matrix entry that keeps the gap visible until split.

## Hermes Feature-Level Gaps

| Gap | Upstream refs | Go target | Classification / progress action |
|---|---|---|---|
| Provider transport abstraction | `../hermes-agent/agent/transports/{base,types,chat_completions,codex,anthropic,bedrock}.py` | `internal/hermes` transport interfaces and stream/event fixtures | complete in Phase 4.A row `Provider interface + stream fixture harness`; `internal/hermes/provider_transport_contract_test.go` proves `ProviderTransport` request building and fixture replay for `chat_completions`, `anthropic_messages`, `bedrock_converse`, and `codex_responses`. Remaining provider-specific live auth/routing/error features stay row-backed under their adapter rows. |
| Trajectory compressor | `../hermes-agent/trajectory_compressor.py` | `internal/transcript`, `internal/audit`, `internal/telemetry`, `internal/hermes` | Phase 4.E row `Trajectory compressor + compressed-evidence lineage` now ports the pure protected-region compression metrics and redacted audit lineage; provider selection, async directory processing, and CLI/batch IO remain future row-backed work. |
| Moonshot/Kimi schema sanitizer | `../hermes-agent/agent/moonshot_schema.py`, `../hermes-agent/tests/agent/test_moonshot_schema.py`, `../hermes-agent/tests/agent/transports/test_chat_completions.py` | `internal/hermes` schema sanitizer | complete in Phase 4.A row `Moonshot/Kimi tool-schema sanitizer`; `internal/hermes/moonshot_schema_test.go` covers `sanitize_moonshot_tool_parameters`, `sanitize_moonshot_tools`, `is_moonshot_model`, and request-boundary aggregator routing. |
| Google/Gemini OAuth correction | `../hermes-agent/agent/google_oauth.py`, Gemini auth callers | `internal/config`, `internal/cli`, `internal/hermes` | still row-backed by Phase 4.G; fix feature-map ownership so Google OAuth belongs to Gemini/Google provider auth, not Codex OAuth. |
| Codex OAuth refs | `../hermes-agent/hermes_cli/auth.py`, `../hermes-agent/agent/auxiliary_client.py`, `../hermes-agent/run_agent.py` | `internal/config`, `internal/cli`, `internal/hermes` | Update Phase 4.A/4.G refs so Codex auth uses Codex/ChatGPT credentials and stale-token refresh evidence. |
| Context compression internals | `context_compressor.py`, `manual_compression_feedback.py`, `context_references.py` | `internal/hermes`, `internal/kernel`, `internal/transcript` | Phase 4.B row `Context references stable-handle parser` now ports typed @ reference parsing and pending handle storage; expansion/injection, sensitive-path filtering, URL/git execution, manual feedback, and remaining compression internals stay row-backed. |
| Prompt builder internals | `prompt_builder.py`, `skill_preprocessing.py`, `skill_commands.py` | `internal/hermes`, `internal/skills`, `internal/cli` | Add source refs and acceptance to Phase 4.C rows for context-file scan, HERMES.md discovery, frontmatter stripping, skill manifest, and skill system prompt. |
| Credential/rate/cache helpers | `credential_pool.py`, `credential_sources.py`, `prompt_caching.py`, `rate_limit_tracker.py`, `nous_rate_guard.py` | `internal/config`, `internal/hermes`, `internal/telemetry` | still row-backed by Phase 4.G/4.H; convert note rows into builder-ready credential, rate, and prompt-cache slices. |
| Raw tool-call parser matrix | `../hermes-agent/environments/tool_call_parsers/*.py` | `internal/hermes` parser/repair fixtures or dedicated parser package | still row-backed by Phase 4.A/5.M; add parser fixture matrix before claiming raw model-output parity. |
| Telegram fallback transport | `../hermes-agent/gateway/platforms/telegram_network.py` | `internal/channels/telegram`, `internal/config` | still row-backed by Phase 2.B/7; add fallback network transport parity row if Telegram runtime needs upstream-compatible proxy behavior. |
| Hermes state/config migration | `../hermes-agent/hermes_constants.py`, `hermes_state.py`, `hermes_cli/config.py` | `cmd/gormes`, `internal/config`, `internal/session`, `internal/memory` | owned config layout plus row-backed migration path; add explicit migration command before claiming legacy state import parity. |

## Hermes Tooling And Operations Gaps

| Gap | Upstream refs | Go target | Classification / progress action |
|---|---|---|---|
| Yuanbao tool drift | `../hermes-agent/tools/yuanbao_tools.py`, `toolsets.py`, `model_tools.py` | `internal/tools/testdata/upstream_tool_parity_manifest.json` | still row-backed by Phase 5.A/7.E; refresh manifest for Hermes audit commit and include `yb_query_group_info`, `yb_query_group_members`, `yb_send_dm`, `yb_search_sticker`, `yb_send_sticker`. |
| ACP adapter slices | `../hermes-agent/acp_adapter/{auth,entry,events,permissions,server,session,tools}.py` | future `internal/acp` or `internal/plugins` + `internal/apiserver` | Split Phase 5.H into registry/stdio launch, session lifecycle, event rendering, tool rendering, permission bridge, and model/slash-command state rows. |
| MCP server/runtime | `../hermes-agent/mcp_serve.py`, `tools/mcp_tool.py`, `tools/managed_tool_gateway.py` | `internal/plugins`, `internal/tools`, `internal/apiserver` | Split Phase 5.G rows for conversation bridge, resource/prompt wrappers, sampling runtime, prompt-injection scan, and OSV checks. |
| Plugin inventory | `../hermes-agent/plugins/**` | `internal/plugins`, `internal/tools`, `internal/goncho` | Add Phase 5.I rows for disk cleanup, image generation plugins, and memory plugins beyond Honcho. |
| Skill catalog parity | `../hermes-agent/skills/**`, `../hermes-agent/optional-skills/**` | `internal/skills`, `docs/development-skills`, `cmd/gormes` | Add Phase 5.F/6.C inventory rows for bundled and optional skill catalogs, provenance, enablement, and prompt snapshots. |
| Sandbox environment details | `tools/environments/**`, `environments/**`, `tools/credential_files.py`, `tools/env_passthrough.py` | `internal/tools`, `internal/cmdrunner`, `internal/config` | Split Phase 5.B into local env scrubbing, SSH, managed Modal, file sync checksum/delete, credential mounts, and env passthrough. |
| Browser/media/security surfaces | `browser_*`, `browser_providers/**`, `tts_tool.py`, `voice_mode.py`, `transcription_tools.py`, `tirith_security.py`, `website_policy.py`, `url_safety.py` | `internal/tools`, `internal/doctor`, future browser/media packages | Split Phase 5.C/5.E/5.J rows by CDP/dialog/snapshot/provider routing, voice state, TTS/transcription, URL/site/tool-output security. |
| Operator tools | `todo_tool.py`, `clarify_tool.py`, `debug_helpers.py`, `send_message_tool.py`, `interrupt.py`, `memory_tool.py`, `rl_training_tool.py`, `mixture_of_agents_tool.py` | `internal/tools`, `internal/gateway`, `internal/sessionsearchtool`, `internal/subagent` | Add Phase 5.N rows for each operator-visible contract before handler ports. |
| Supply-chain and skills-index workflow parity | `.github/workflows/supply-chain-audit.yml`, `.github/workflows/skills-index.yml` | docs/CI contract or `cmd/repoctl` | still row-backed by Phase 5.P/6; add a CI contract row if these workflows remain part of public operational parity. |

## Hermes Gateway, CLI, And Release Gaps

| Gap | Upstream refs | Go target | Classification / progress action |
|---|---|---|---|
| Channel directory/mirror/sticker cache | `gateway/channel_directory.py`, `gateway/mirror.py`, `gateway/sticker_cache.py` | `internal/gateway`, `internal/channels/*`, `internal/store` | Convert Phase 2.F.4 note rows into builder-ready slices for persistence, refresh/stale invalidation, mirror writes, and sticker cache injection. |
| Dashboard management routes | `hermes_cli/web_server.py`, `web/**`, `website/**` | `internal/apiserver`, `www.gormes.ai`, `internal/config` | Add Phase 5.Q rows for config/env/logs/OAuth mutation/theme/plugin assets, or mark unsupported routes as owned exclusions. |
| CLI umbrellas | `hermes_cli/**`, `cli.py` | `cmd/gormes`, `internal/cli`, `internal/config`, `internal/doctor` | Split Phase 5.O umbrella rows into command-tree, auth/setup/config, management CLI, and diagnostics/log/status contracts. |
| Channel bootstrap | `gateway/platforms/**` | `internal/channels/*`, `internal/gateway` | Add builder-ready metadata for Signal, Matrix, Mattermost, Feishu, DingTalk SDK, QQ, and Yuanbao transport/bootstrap rows. |
| Release and packaging | `scripts/release.py`, `Dockerfile`, `docker/**`, `docker-compose.yml`, `packaging/**`, `nix/**`, `flake.*` | installers, service units, OCI image, `Makefile`, `cmd/gormes`, `www.gormes.ai` | Split Phase 5.P into OCI/Docker, Homebrew, Nix/flake/NixOS, release artifact/versioning, service install, and public installer/site contracts. |

## Honcho/Goncho Feature-Level Gaps

| Gap | Upstream refs | Go target | Classification / progress action |
|---|---|---|---|
| SDK refs and fixture matrix | `../honcho/tests/sdk/**`, `../honcho/sdks/typescript/__tests__/**`, `../honcho/sdks/python/src/**`, `../honcho/sdks/typescript/src/**` | `internal/goncho`, `internal/gonchotools`, future compatibility adapter | Fix stale `../honcho/sdks/*/tests` refs and split 3.G into Python SDK matrix, TypeScript SDK matrix, and unsupported-field evidence rows. |
| OpenAPI v3 route manifest | `../honcho/docs/v3/openapi.json`, `../honcho/src/routers/**`, `../honcho/tests/routes/**` | future `internal/goncho/http`, `internal/apiserver` | Add 3.G/5.Q row for route shape, aliases, validation, status codes, pagination, errors, and unsupported-route evidence. |
| Honcho MCP catalog | `../honcho/mcp/src/tools/**`, MCP README/instructions | `internal/gonchotools`, `internal/plugins`, `internal/tools` | Complete in Phase 5.G row `Goncho MCP tool catalog`; `internal/gonchotools/honcho_mcp_catalog_test.go` proves all upstream MCP tool names are mapped, partially mapped with field-level degradation, or explicit unsupported rows. |
| Honcho CLI compatibility | `../honcho/honcho-cli/src/honcho_cli/**`, tests | `cmd/gormes goncho`, `internal/config`, `internal/doctor` | Add row for command groups, JSON/table output, config preservation, ID validation, destructive confirms, and exit codes. |
| Deploy/config divergence | `.env.template`, `config.toml.example`, `Dockerfile`, compose, `fly.toml`, `docker/entrypoint.sh`, `database/init.sql`, `mcp/.dev.vars.example` | `cmd/gormes`, `internal/config`, `internal/doctor`, release docs | Add owned/excluded divergence matrix for hosted FastAPI/Postgres/Redis/Fly/Alembic surfaces versus local Goncho. |
| CRUD invariants | `src/models.py`, `src/schemas/**`, `src/crud/**`, migrations | `internal/goncho`, `internal/store`, `internal/memory`, `internal/session` | Add row for workspace/peer/session/message/session-peer lifecycle, config merge/replace, metadata validation, sequence assignment, clone/delete cascades. |
| Deriver queue lifecycle | `src/deriver/queue_manager.py`, `src/deriver/**`, `src/reconciler/**` | `internal/goncho`, `internal/cron`, `internal/doctor` | Add row for queue item state, ownership, stale cleanup, errors, and `queue.empty` event evidence before LLM derivation. |
| Observation document tree | `src/crud/representation.py`, `src/deriver/**`, migrations | `internal/goncho`, `internal/memory` | Add row for explicit/deductive/inductive/contradiction observations, source IDs, duplicate replacement, soft delete, and derived-count semantics. |
| Dialectic and dream execution | `src/dialectic/**`, `src/dreamer/**`, tests | `internal/goncho`, `internal/kernel`, `internal/cron` | Add fake-provider rows for tool-loop chat, reasoning levels, streaming finalization, scheduled dream execution, observations, and peer-card side effects. |
| Webhooks, keys, telemetry | `src/routers/{keys,webhooks}.py`, `src/webhooks/**`, `src/telemetry/**` | `internal/apiserver`, `internal/gateway`, `internal/telemetry`, `internal/audit` | Add rows for key creation, webhook CRUD/test/HMAC/delivery failures, queue-empty integration, local event names, and reasoning evidence. |
| Vector/reconciler divergence | `src/vector_store/**`, `src/reconciler/**` | `internal/memory`, `internal/store` | Complete in Phase 3.F row `Vector store + reconciler divergence proof`; `internal/memory/divergence_test.go` proves LanceDB/Turbopuffer/reconciler surfaces are explicit excluded divergence and SQLite semantic recall has no external sync-state dependency. |
| Goncho tool schema drift | `internal/gonchotools/honcho_tools.go`, `internal/goncho/types.go` | `internal/gonchotools` | Add small row to expose `honcho_search.filters`, `honcho_search.limit`, and `honcho_context.include_dream_status` if kept as public compatibility knobs. |
| Honcho API validation limits | `../honcho/src/schemas/api.py`, `../honcho/tests/test_schema_validations.py` | `internal/goncho/compat`, `internal/config` | still row-backed by Phase 3.G; add metadata, ID regex, token limit, and message limit sanitizer fixtures. |
| Honcho effective config hierarchy | `../honcho/src/schemas/configuration.py`, `../honcho/tests/unified/test_cases/config_*.json` | `internal/config.GonchoCfg`, `internal/goncho.Config` | mapped-by-contract but needs row split for workspace/peer/session/message override fixtures. |
| Honcho model config/provider bridge | `../honcho/src/llm/**`, `../honcho/tests/llm/**` | `internal/goncho`, `internal/hermes` bridge or owned deterministic engine | still row-backed by Phase 3.G/4.A; fake-provider fixtures decide which hosted LLM behavior is compatible versus owned. |
| Honcho integration guides and examples | `../honcho/docs/v3/guides/**`, `../honcho/examples/**`, `../honcho/tests/unified/**` | `internal/goncho/compat/testdata`, docs examples | mapped-by-contract; import representative fixture examples before claiming guide-level compatibility. |
| Honcho live LLM tests | `../honcho/tests/live_llm/**` | fake/local LLM fixtures only | excluded for live-provider execution; any behavior needed by Gormes must move through hermetic `../honcho/tests/llm/**` fixtures. |

## Required Next Planning Gates

The map can only be called feature-complete after these gates pass:

1. Source-class ledger test passes.
2. Swarm gap register contains no `missing` or `vague` entry without a
   builder-ready row.
3. All feature-derived rows have `source_refs`, `write_scope`, `test_commands`
   or `no_test_required`, acceptance, and done signal.
4. Rows with intentionally different Go behavior state `owned` or `excluded`
   and name the replacement test.
5. A focused docs test checks selected nested source classes, not only
   top-level directories.

## Update Rules

- Do not add a side TODO list. Runtime work belongs in `progress.json`.
- When a broad gap gets split, update the runtime plan, this register, and the
  nested coverage matrix in the same planner pass.
- When Gormes intentionally diverges, record `owned` or `excluded` with a
  replacement proof or explicit live-only exclusion.
- When a nested upstream path appears in a sibling checkout and is not in the
  matrix, add it or explain why it is repo hygiene.

## Per-Pass Audit Index

| Pass date | Form | Notes |
|---|---|---|
| 2026-04-27 | Seven-agent | First pass to apply seven-bucket classification end-to-end and to add coverage detection for `cli-config.yaml.example`, `constraints-termux.txt`, `tinker-atropos`, and `database/init.sql`. |
| 2026-04-28 | Five-lane | Lane-level summary of the same overnight investigation with the gap-register form below. |

Each new swarm pass appends a row here and adds a `## YYYY-MM-DD ... Pass`
section above with the same specialist table template.

## 2026-04-27 Builder-Ready Row Candidates

The gap classifier proposed ten builder-ready rows that the next planner
passes should write into `progress.json`. They are recorded here as the
authoritative split list; each one names the parent phase anchor.

| # | Row name | Phase anchor | Go target | Test command (focused) | Status |
|---|---|---|---|---|---|
| 1 | Provider transport interface + stream fixture harness | 4.A | `internal/hermes.ProviderTransport` + transcript fixtures across Anthropic/Bedrock/Codex/Chat-Completions | `go test ./internal/hermes -run ProviderTransport -count=1` | shipped 2026-04-28 (Phase 4.A row complete) |
| 2 | Moonshot/Kimi tool-schema sanitizer | 4.A | `internal/hermes/moonshot_schema*.go` | `go test ./internal/hermes -run Moonshot -count=1` | shipped 2026-04-28 (Phase 4.A row complete) |
| 3 | Trajectory compressor + compressed-evidence lineage | 4.E | `internal/transcript/trajectory_compressor*.go`, `internal/audit/trajectory_link.go` | `go test ./internal/transcript -run TrajectoryCompressor -count=1` | shipped 2026-04-28 (Phase 4.E row complete) |
| 4 | Context references stable-handle store | 4.B | `internal/hermes/context_references*.go`, `internal/contextrefs/refs.go`, `internal/transcript/refs.go` | `go test ./internal/hermes -run ContextReferences -count=1` | shipped 2026-04-28 (Phase 4.B row complete; parser + pending handle store only, live expansion remains row-backed) |
| 5 | Goncho keys + webhooks compatibility surface | 3.G | `internal/goncho/{keys,webhooks}*.go`; HTTP route binding remains row-backed under the OpenAPI route parity slice | `go test ./internal/goncho -run Keys -count=1`, `go test ./internal/goncho -run Webhooks -count=1` | shipped 2026-04-28 (Phase 3.G row complete; internal key/webhook surface only, HTTP route binding and delivery workers remain row-backed) |
| 6 | Goncho MCP tool catalog | 5.G | `internal/gonchotools/honcho_mcp_catalog*.go` | `go test ./internal/gonchotools -run HonchoMCPCatalog -count=1` | shipped 2026-04-28 (Phase 5.G row complete) |
| 7 | Goncho HTTP route parity over OpenAPI v3 | 3.G/5.Q | `internal/goncho/http/*` | `go test ./internal/goncho/http -count=1` | shipped 2026-04-28 (Phase 3.G row complete; OpenAPI route manifest and unsupported-route contract only, handler binding remains row-backed) |
| 8 | Goncho CLI command-tree parity | 3.G/5.O | `cmd/gormes/goncho*.go`, `cmd/gormes/goncho_cli_parity_test.go` | `go test ./cmd/gormes -run GonchoCLIParity -count=1` | shipped 2026-04-28 (Phase 3.G row complete; command-tree manifest only, functional command execution remains row-backed) |
| 9 | Goncho CRUD lifecycle invariants | 3.F | `internal/goncho/crud_invariants_test.go`, `internal/goncho/sql.go` | `go test ./internal/goncho -run CRUDInvariants -count=1` | shipped 2026-04-28 (Phase 3.F row complete; local SQLite lifecycle invariants only) |
| 10 | Vector store + reconciler divergence proof | 3.D/3.G | `internal/memory/divergence_test.go`, `internal/memory/vector_divergence.go` | `go test ./internal/memory -run Divergence -count=1` | shipped 2026-04-28 (Phase 3.F row complete) |

Plus 23 additional unknown/gap items the gap classifier surfaced (sandbox
env split, voice/TTS/transcription split, browser provider rows, ACP 7-row
split, MCP server-side runtime split, operator-tool rows, CLI umbrella
splits, dashboard mutation rows, SDK matrix split, deploy divergence
matrix, prompt/audit/debug-bundle redaction, Google OAuth ownership
re-anchor, deriver queue lifecycle, observation document tree, dialectic
and dream fake-provider execution, Goncho telemetry, Honcho deploy
divergence matrix, Yuanbao manifest refresh, Codex umbrella deletion,
Yuanbao gateway/toolset cross-link, Goncho tool schema knobs, plugin
inventory beyond Spotify/Google Meet, deprecated-cron-umbrella cleanup).
Each one has a one-sentence sub-agent brief in the gap classifier transcript
and either a row-split target or an owned/excluded decision required.
