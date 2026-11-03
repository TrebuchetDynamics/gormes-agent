---
title: "Upstream Coverage Ledger"
weight: 19
aliases:
  - /building-gormes/architecture_plan/upstream-coverage-ledger/
---

# Upstream Coverage Ledger

The feature map says what Hermes and Honcho behavior means in Go. This ledger
answers the harder question: **how do we know the map is complete?**

Gormes does not claim completeness by vibes or by a long prose essay. A planner
pass can claim a mapped upstream only when every feature-bearing upstream source
class below has a feature-map anchor, a Go target, and either a concrete
`progress.json` row or an explicit owned/excluded decision.

## Completeness Standard

A Hermes or Honcho surface is mapped only when all five checks pass:

1. **Inventory:** the upstream file, directory, endpoint group, SDK surface, or
   public document belongs to a row in this ledger.
2. **Feature-map anchor:** the row links to a section in
   [Hermes And Honcho Feature Map](../hermes-honcho-feature-map/).
3. **Go target:** the row names a Gormes package, command, doc surface, or an
   intentional Go-native replacement.
4. **Backlog reachability:** incomplete behavior has a `progress.json` anchor;
   broad anchors stay non-builder-ready until split.
5. **Drift rule:** new upstream feature-bearing files must update this ledger,
   the feature map, and `progress.json` in the same planner pass.

The status vocabulary is the same as the feature map: `covered`, `planned`,
`vague`, `missing`, `owned`, and `excluded`. `Excluded` is allowed only for
upstream repo hygiene, release notes, generated assets, or examples that do not
change public runtime behavior.

## Mapping Is Not Behavioral Completion

This ledger proves that feature-bearing upstream source classes are represented
in the plan; it does **not** prove their behavior is implemented. A class marked
`covered`, a complete progress row, or a passing source-class inventory test
cannot by itself support a full Hermes-in-Go claim. Use the feature-level
`webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md` inventory and the
[runtime plan's full parity exit contract](../hermes-honcho-go-runtime-plan/#full-hermes-parity-exit-contract).

A strict completion result is scoped to an immutable Hermes SHA and requires no
unmapped behavior, no `partial`/`missing`/`unknown` atoms, passing observable
contract tests, and no behavior-changing `owned`/`excluded` divergences. A newer
upstream `main` invalidates the current completion result until its delta is
reconciled. The existing source-class and nested-path tests are necessary
inventory checks, not the strict completion gate.

## Audit Commands

Use these commands during a full parity pass. They intentionally inventory
source classes, not vendored caches or Git metadata:

```sh
find ../hermes-agent -type f \
  \( -name '*.py' -o -name '*.md' -o -name '*.json' -o -name '*.yaml' -o -name '*.yml' -o -name '*.toml' -o -name '*.lock' -o -name '*.nix' -o -name '*.sh' -o -name 'Dockerfile' -o -name 'docker-compose.yml' \) \
  ! -path '*/.git/*' ! -path '*/node_modules/*' ! -path '*/__pycache__/*' \
  | sed 's#^../hermes-agent/##' | awk -F/ '{print $1}' | sort | uniq -c

find ../honcho -type f \
  \( -name '*.py' -o -name '*.md' -o -name '*.mdx' -o -name '*.ts' -o -name '*.tsx' -o -name '*.json' -o -name '*.yaml' -o -name '*.yml' -o -name '*.toml' -o -name '*.lock' -o -name '*.ini' -o -name '*.sh' -o -name 'Dockerfile' \) \
  ! -path '*/.git/*' ! -path '*/node_modules/*' ! -path '*/__pycache__/*' \
  | sed 's#^../honcho/##' | awk -F/ '{print $1}' | sort | uniq -c
```

If either command shows a new feature-bearing class that is not represented
below, the map is no longer complete.

## Executable Check

`go test ./docs -run TestUpstreamCoverageLedgerMatchesSourceClasses -count=1`
compares this ledger against the local sibling `../hermes-agent` and
`../honcho` checkouts when they are present. The test fails when a feature-like
top-level source class is neither represented in this ledger nor explicitly
classified as repo hygiene. It skips a missing sibling checkout so normal docs
CI does not require cloning upstream repos.

This is necessary but not sufficient for feature parity. Use
[Hermes/Honcho To Gormes Go Runtime Plan](../hermes-honcho-go-runtime-plan/)
for the reconciled subsystem classification and Go implementation order. Use
[Swarm Feature Parity Audit](../swarm-feature-parity-audit/) for nested
feature-level gaps found inside broad source classes such as `agent/**`,
`tools/**`, `gateway/**`, `src/**`, `sdks/**`, `mcp/**`, and `tests/**`.
Use `TestNestedUpstreamFeatureCoverage` to make representative nested paths
executable when sibling upstream checkouts are present.

## Hermes Source Coverage

| Upstream source class | Feature-map anchor | Go target | Progress anchor | Coverage |
|---|---|---|---|---|
| `run_agent.py`, `environments/agent_loop.py`, `hermes_state.py`, `trajectory_compressor.py` | Agent Runtime | `internal/kernel`, `internal/llm`, `internal/persistence/transcript`, `internal/e2e` | Phase 4.I, 4.B, 4.E | planned |
| `agent/*.py`, `agent/transports/*` | Providers, Models, Credentials; Prompt, Context, Compression | `internal/llm`, `internal/kernel`, `internal/config`, `internal/telemetry` | Phase 4.A-4.H | planned |
| `providers/**`, `plugins/model-providers/**` | Declarative provider profiles, registry/discovery, auth and model-catalog metadata, transport request hooks | `internal/llm/providerregistry`, `internal/llm/modelcatalog`, `internal/config/credentials`, `internal/doctor` | Phase 4.A, 4.D, 4.H provider rows | planned |
| `agent/account_usage.py:render_account_usage_lines`, `agent/account_usage.py:fetch_account_usage`, `cli.py` `/usage`, `gateway/run.py:_handle_usage_command`, `tests/agent/test_account_usage.py`, `tests/gateway/test_usage_command.py` | Provider account-usage reporting and gateway usage command | `internal/llm`, `internal/telemetry`, `internal/gateway`, `cmd/gormes` | Phase 4.H rows `Provider account usage read model + renderer` and `Gateway /usage command binding over provider account usage` | planned |
| root helpers: `hermes_constants.py`, `hermes_logging.py`, `hermes_time.py`, `utils.py` | Redaction, evidence filtering, config/time helpers | `internal/config`, `internal/audit`, `internal/telemetry`, `internal/cli` | Phase 4.E, 5.O | planned |
| `tools/*.py`, `tools/environments/*`, `tools/browser_providers/*`, `tools/neutts_*` | Tools, Sandboxes, Security | `internal/tools`, `internal/cmdrunner`, `internal/plugins`, `internal/doctor` | Phase 2.A, 5.A-5.N | planned |
| root tool/model helpers: `model_tools.py`, `toolsets.py`, `toolset_distributions.py`; model registry files `agent/model_metadata.py`, `agent/models_dev.py` | Tool registry, toolsets, model metadata | `internal/tools`, `internal/llm`, `internal/doctor` | Phase 2.A, 4.D, 5.A | planned |
| `gateway/**/*.py` | Gateway, Channels, Cron, API, TUI, CLI | `internal/gateway`, `internal/channels/*`, `internal/slack`, `internal/discord` | Phase 2.B-2.F, 7 | planned |
| `cron/*.py` | Cron and scheduled jobs | `internal/automation/cron`, `internal/gateway`, `internal/tools` | Phase 2.D, 5.N | planned |
| `cli.py`, `hermes`, `hermes_cli/**/*.py`, exact auth files `hermes_cli/auth.py`, `hermes_cli/auth_commands.py`, `hermes_cli/main.py:auth_parser`, exact plugin manager files `hermes_cli/plugins.py`, `hermes_cli/plugins_cmd.py`, and plugin command providers such as `plugins/memory/__init__.py` and `plugins/disk-cleanup/__init__.py` | CLI command tree, slash commands, dynamic plugin commands, config, secrets, current auth command surface (`auth add/list/remove/reset/status/logout/spotify`), removed top-level login guidance, setup/profile/status, backup/import, logs, release/update commands | `cmd/gormes`, `cmd/gormes/auth.go`, `internal/cli`, `internal/config`, `internal/doctor`, `internal/plugins` | Phase 5.O rows `Hermes CLI command-tree parity manifest`, `Hermes auth command-tree manifest refresh`, `Hermes auth credential-pool command surface`, `Hermes auth OAuth provider adapters`, `Hermes auth Spotify service-provider subcommand`, the Phase 5.O auth/lifecycle splits added 2026-04-29 (`Gormes auth bare interactive credential-pool readout`, `Gormes auth status per-provider aggregator`, `Gormes auth add openai-codex strict isolation contract`, `Gormes auth add bedrock open-question planning note`, `Gormes top-level logout provider shortcut`, `Gormes removed top-level login guidance`, `Gormes model interactive provider/model picker`, `Gormes setup minimal sectioned wizard slice`, `Gormes uninstall dry-run command contract`, `Gormes mcp login OAuth re-auth bridge`), Phase 5.O config/migration rows, Phase 5.P | planned |
| `cli-config.yaml.example` | CLI/config canonical schema example (51 KB; spec of record for `hermes_cli/config.py`) | `cmd/gormes`, `internal/cli`, `internal/config`, `development-skills` | Phase 5.O | planned |
| `hermes_cli/claw.py`, `optional-skills/migration/openclaw-migration/**`, `website/docs/guides/migrate-from-openclaw.md` | OpenClaw migration command, mapping matrix, report/archive/cleanup behavior | `cmd/gormes`, `internal/platform/migrate/openclaw`, `internal/config`, `internal/skills`, `internal/memory` | Phase 5.O | planned |
| `constraints-termux.txt` | Packaging hygiene (Termux/Android pip constraints used by `setup-hermes.sh`) | installers, `Makefile`, `www.gormes.ai` | Phase 5.P | excluded as packaging hygiene |
| `acp_adapter/auth.py:detect_provider`, `acp_adapter/entry.py:main`, `acp_adapter/{events,permissions,server,session,tools}.py`, `acp_registry/**` | ACP adapter | `internal/plugins`, `internal/apiserver`, future `internal/protocols/acp` | Phase 5.H row `ACP server side` | planned |
| `mcp_serve.py`, MCP config helpers | MCP tools and managed gateway | `internal/plugins`, `internal/tools`, `internal/apiserver` | Phase 5.G | planned |
| `optional-mcps/**`, `hermes_cli/mcp_catalog.py`, `hermes_cli/mcp_picker.py`, `tests/hermes_cli/test_mcp_catalog.py` | Nous-approved optional MCP catalog manifests, diagnostics, picker/list, install/configure, and explicit update behavior | `internal/tools/mcp/catalog`, `internal/platform/cli/gormescli` | Phase 5.G rows `Hermes approved MCP catalog manifest + list command` and `Hermes MCP catalog install/configure lifecycle` | planned |
| `plugins/**` | Plugins, memory plugins, specialized modes | `internal/plugins`, `internal/tools`, `internal/goncho`, `internal/gonchotools` | Phase 3.G, 5.I; source-backed integration-claim classifications live in [Hermes Integrations Claim Audit](../hermes-integrations-claim-audit/) | planned |
| `skills/**`, `optional-skills/**`, `optional-skills/yuanbao/SKILL.md` | Skill manager, slash skills, learning loop | `internal/skills`, `development-skills`, `cmd/gormes`; source-backed integration-claim classifications live in [Hermes Integrations Claim Audit](../hermes-integrations-claim-audit/) | Phase 5.F, 6 | planned |
| `tui_gateway/**`, `ui-tui/**`, `web/**`, `website/**` | TUI, API, public web surfaces | `internal/tui`, `internal/tuigateway`, `internal/apiserver`, `www.gormes.ai` | Phase 5.Q, 5.P | planned |
| `apps/{desktop,bootstrap-installer,shared}/**` | Electron desktop shell, shared gateway RPC client, and Tauri bootstrap installer | Gormes-owned native TUI plus Go/HTMX operator dashboard and existing Go-native installers; no Electron/Tauri runtime dependency | Phase 1 dashboard/TUI rows and Phase 5.P installer rows | owned replacement |
| `locales/**`, `tests/agent/test_i18n.py` | Static operator-message localization catalogs and fallback rules | `internal/tui/i18n`, gateway message renderers | Phase 5.W row `Hermes i18n static-message port` | covered |
| `batch_runner.py`, `mini_swe_runner.py`, `rl_cli.py`, `datagen-config-examples/**` | Batch, mini-SWE, RL, datagen | future research packages, `internal/subagent`, `internal/tools` | Phase 5.M/5.O | planned |
| `tinker-atropos/**` | RL tinker submodule placeholder (currently empty; reserves slot for atropos environment integration) | future research packages, `internal/subagent` | Phase 5.M | owned (excluded until populated) |
| `Dockerfile`, `docker/entrypoint.sh`, `docker-compose.yml`, `docker-compose.windows.yml`, `nix/**`, `nix/hermes-agent.nix`, `flake.*`, `hermes_bootstrap.py`, `setup-hermes.sh`, `setup.py`, `package.json`, `package-lock.json`, `pyproject.toml`, `uv.lock`, `scripts/release.py`, `scripts/**` | Packaging, Windows UTF-8/import-path bootstrap, and release | installers, service units, Go-native UTF-8 process defaults, OCI image, Homebrew formula, `Makefile`, `cmd/gormes` | Phase 5.P rows `OCI image` and `Homebrew`; Windows installer/runtime fixtures | planned/owned divergence |
| `tests/**`, `docs/**`, `.plans/**`, `plans/**`, release notes, `AGENTS.md`, `CLAUDE.md`, translated and canonical README/SECURITY/CONTRIBUTING files, `hermes-already-has-routines.md` | Fixture and documentation donors | docs, fixtures, progress source refs | Rows that cite the fixture | covered as evidence |

At pinned Hermes `bb5fc723`, the former root `environments/**` eval classes and raw tool-call parser family are no longer active upstream source classes. Current process environments live under `tools/environments/**`; Gormes' completed raw-parser fixtures remain historical compatibility evidence rather than an active missing parity contract.

### Hermes root-class refresh (2026-09 decomposition)

The 2026-09 `d0288be5b3330d2442e3907185b8e9d0958297bb` decomposition split or
added these feature-bearing root classes. Each maps to an existing Gormes
subsystem owned by the rows named below; none is a new runtime contract beyond
those rows.

| Upstream source class | Feature-map anchor | Go target | Progress anchor | Coverage |
|---|---|---|---|---|
| `hermes_state.py` split modules: `hermes_state_common.py`, `hermes_state_compression.py`, `hermes_state_dbfile.py`, `hermes_state_errors.py`, `hermes_state_fts.py`, `hermes_state_gateway.py`, `hermes_state_guard.py`, `hermes_state_health.py`, `hermes_state_holders.py`, `hermes_state_ids.py`, `hermes_state_lockguard.py`, `hermes_state_lockowners.py`, `hermes_state_maintenance.py`, `hermes_state_messages.py`, `hermes_state_portability.py`, `hermes_state_profile_repair.py`, `hermes_state_readpool.py`, `hermes_state_registry.py`, `hermes_state_repair.py`, `hermes_state_rewind.py`, `hermes_state_schema.py`, `hermes_state_search.py`, `hermes_state_sessions.py`, `hermes_state_telegram.py`, `hermes_state_timeline.py`, `hermes_state_titles.py`, `hermes_state_usage.py`, `hermes_state_user_copy.py`, `hermes_state_wal.py`, `hermes_yaml.py` | Sessions/transcript store, WAL and lock ownership, FTS, usage counters, YAML policy | `internal/persistence/session`, `internal/persistence/transcript`, `internal/config` | Phase 3.E, 4.E | planned |
| `hermes_platform/**` | Platform declaration and resolver helpers | `internal/platform`, `internal/config` | Phase 5.O | planned |
| `hermes_startup_watchdog.py` | Gateway startup-liveness watchdog | `internal/gateway`, `internal/doctor` | Phase 2.F | planned |
| `hermes_constants_scratch.py` | Scratch-directory retention and reaping | `internal/tools`, `internal/doctor` | Phase 4.E | planned |
| `hermes_constants_scratch_*.py` 2026-11 detach/stuck variants | Scratch-directory retention, detach, and stuck-process reaping | `internal/tools`, `internal/doctor` | Phase 4.E, 4.I | planned |
| `hermes_dir_acl.py` | Directory ACL and permission policy | `internal/tools`, `internal/platform/security` | Phase 4.I, 5.J | planned |
| `agent/transports/hermes_tools_mcp_server.py` | Hermes MCP server transport after core `mcp_serve.py` moved to the official `mcp-serve` plugin | `internal/tools/mcp`, `internal/adapters/apiserver` | Phase 4.I, 5.G | planned |
| `plugin-catalog/mcp-serve.yaml`, `plugin-catalog/honcho.yaml`, `plugin-catalog/agent-importers.yaml` | Official plugin-catalog moves for MCP serve, Honcho memory, and agent importers | `internal/extensibility/plugins`, `internal/adapters/goncho`, `internal/app/migrate` | Phase 4.I, 5.G, 5.O | planned |
| `hermes_cli/migrate.py`, `hermes_cli/setup_migration.py` | OpenClaw and agent-import migration CLI after `hermes_cli/claw.py` left core | `internal/app/migrate`, `cmd/gormes` | Phase 4.I, 5.O | planned |
| `hermes_state_*.py` 2026-11 split modules: `api_content`, `compaction_display`, `coverage`, `detached_results`, `identity`, `keys`, `local`, `local_lineage`, `local_migration`, `media`, `mutation_*`, `mutations`, `pidns`, `prompt_refresh`, `prune_rewrite`, `reconcile`, `runtime`, `runtime_workers`, `session_list`, `sessions_model_config`, `terminal`, `todo`, `worker_*` | Session persistence, local lineage/migration, mutation guards, detached results, worker lifecycle, terminal/todo state, and media identity | `internal/persistence/session`, `internal/persistence/transcript`, `internal/gateway`, `internal/tools` | Phase 4.I, 4.E | planned |
| `registration_lifecycle.py` | Replaceable-registration ownership leases | `internal/plugins`, `internal/config` | Phase 5.I | planned |
| `native/**`, `native/fts5_cjk` | Native FTS5 CJK tokenizer extension | `internal/persistence/session` | Phase 3.C | planned |
| `plugin-catalog/**` | Plugin and MCP catalog manifests | `internal/tools/mcp/catalog`, `internal/plugins` | Phase 5.G | planned |
| `pm/**` | Build/package-manager adjunct | installers, `scripts/`, `cmd/gormes` | Phase 5.P | planned/owned divergence |
| `evals/**` | Deterministic eval probes | `internal/e2e`, `internal/kernel` | Phase 4.I | planned |
| `COMPAT_MANIFEST.md`, `compat_manifest.json` | Temporary plugin-compat pointers from the decomposition | `internal/plugins`, `internal/compat/hermes` | Phase 5.I | excluded as temporary compatibility pointers |
| `SOUL.md` | Default agent identity/personality seed | `internal/llm/default_soul.go`, `internal/core/agenttemplate` | Phase 4.I | planned |
| `tests-js/**` | Browser/desktop JS fixture donors | docs, fixtures, progress source refs | Rows that cite the fixture | covered as evidence |

## Honcho Source Coverage

| Upstream source class | Feature-map anchor | Go target | Progress anchor | Coverage |
|---|---|---|---|---|
| `src/models.py:MessageEmbedding,Collection,Document,QueueItem,ActiveQueueSession,WebhookEndpoint,SessionPeer`, `src/schemas/**`, `migrations/**` | Workspaces, peers, sessions, messages, conclusions, queues, webhooks, embeddings | `internal/goncho`, `internal/persistence/store`, `internal/config` | Phase 3.F/3.G rows including `Goncho CRUD lifecycle invariants` | planned |
| `src/routers/{workspaces,peers,sessions,messages,conclusions,keys,webhooks}.py` | Honcho concepts and APIs | `internal/goncho`, `internal/gonchotools`, `internal/apiserver` | Phase 3.G row `OpenAPI v3 route manifest`, Phase 5.Q | planned |
| `src/crud/{workspace,peer,message,document,webhook,peer_card,deriver,session,representation}.py`, `src/crud/collection.py:collection_cache_key,get_collection,get_or_create_collection,update_collection_internal_metadata` | Storage semantics, collection cache keys, peer cards, messages, documents, webhooks, conclusions | `internal/goncho`, `internal/memory`, `internal/persistence/session` | Phase 3.F/3.G rows including `Goncho CRUD lifecycle invariants` | planned |
| `src/dialectic/chat.py`, `src/dialectic/core.py`, `src/dialectic/prompts.py` | Dialectic chat | `internal/goncho`, `internal/kernel` | Phase 3.F/3.G | planned |
| `src/deriver/**`, `src/dreamer/dream_scheduler.py`, `src/dreamer/orchestrator.py`, `src/dreamer/specialists.py`, `src/dreamer/surprisal.py`, `src/dreamer/trees/{base,covertree,graph,lsh,prototype,rptree,sklearn_wrapper}.py`, `src/reconciler/**`, `src/llm/**` | Conclusions, dreaming, surprisal/tree algorithms, queue status | `internal/goncho`, `internal/memory`, `internal/automation/cron`, `internal/doctor` | Phase 3.F, 6 | planned |
| `src/utils/**`, `src/cache/**` | Filters, search, summaries, representation, files | `internal/goncho`, `internal/memory`, `internal/llm` | Phase 3.F, 4.C | planned |
| `src/vector_store/**` | Vector stores and reconciler | `internal/memory`, `internal/persistence/store` | Phase 3.D/3.G | owned |
| `src/telemetry/reasoning_traces.py`, `src/telemetry/metrics_collector.py`, `src/telemetry/sentry.py`, `src/telemetry/events/deletion.py`, `src/telemetry/events/**`, `src/webhooks/events.py`, `src/webhooks/webhook_delivery.py` | Telemetry, reasoning traces, hosted metrics/Sentry divergence, deletion events, webhooks | `internal/telemetry`, `internal/audit`, `internal/goncho`, `internal/apiserver`, `internal/gateway` | Phase 3.G row `Goncho webhook delivery retry worker contract`, Phase 4.E row `Self-monitoring telemetry`, Phase 5.Q | planned |
| `docs/v1/**`, `docs/v2/**`, `docs/v3/**` | Honcho public API and guide contracts | Goncho compatibility docs and fixtures | Phase 3.G | planned |
| `sdks/python/src/honcho/{client,aio,api_types,mixins}.py`, `sdks/typescript/src/client.ts`, `sdks/typescript/src/validation.ts`, `sdks/typescript/src/http/streaming.ts`, `sdks/typescript/**`, `mcp/**` | SDKs and MCP | `internal/goncho`, future local compatibility adapter | Phase 3.G row `Goncho Honcho SDK compatibility e2e harness` | planned |
| `honcho-cli/src/honcho_cli/{main,config,output,validation,_help,branding}.py`, `honcho-cli/src/honcho_cli/commands/{workspace,peer,session,message,conclusion,config_cmd,setup}.py`, `.env.template`, `config.toml.example`, `alembic.ini` | CLI/self-hosting docs | `cmd/gormes`, `internal/config`, `internal/doctor` | Phase 3.G row `Goncho CLI command-tree parity`, Phase 5.O | planned |
| `database/init.sql` | Hosted Postgres bootstrap (`CREATE EXTENSION IF NOT EXISTS vector;`) | `cmd/gormes`, `internal/config`, `internal/doctor`, release docs | Phase 3.G, 5.P | owned (hosted-only divergence; replacement is local SQLite goncho store) |
| `Dockerfile`, `docker-compose.yml.example`, `docker/entrypoint.sh`, `docker/prometheus.yml`, `docker/grafana-datasource.yml`, `docker/**`, `fly.toml`, `pyproject.toml`, `uv.lock`, `scripts/**` | Self-hosting, deploy, observability adjuncts, and maintenance tooling | `cmd/gormes`, `internal/doctor`, installer/release docs | Phase 3.G, 5.P | owned/planned divergence |
| `examples/**`, `tests/**`, `CHANGELOG.md`, README/AGENTS/CLAUDE/CONTRIBUTING | Compatibility fixture and documentation donors | Goncho testdata and progress refs | Phase 3.G, 6 | covered as evidence |
| `.claude/skills/**` | Development workflow donor | `development-skills` when useful | Phase 1.D/6 | owned |

## What Counts As Unmapped

An upstream feature is unmapped when any of these are true:

- its source class is absent from this ledger;
- its ledger row points to no feature-map section;
- the feature map names no Go target or intentional divergence;
- incomplete behavior has no `progress.json` row;
- a row exists but has no `source_refs`, `write_scope`, `test_commands` or
  `no_test_required`, acceptance, and done signal.

When that happens, the correct action is a `gormes-parity-auditor` pass followed
by a `gormes-planner` pass. Do not hand the behavior to `gormes-builder` until
the map and row are builder-ready.
