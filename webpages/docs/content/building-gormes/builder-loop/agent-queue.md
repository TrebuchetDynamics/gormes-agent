---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
skill-driven implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared skill handoff facts live in [Skill Builder Handoff](../builder-loop-handoff/):
the main skill entrypoint, plan, candidate source, generated docs, tests, and
candidate policy. Keep those control-plane facts in `meta.builder_loop`, and
keep row-specific execution facts in `progress.json`.

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. Navivox natural-language profile seed Flutter UI

- Phase: 9 / 9.F
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Add the sibling Navivox Flutter profile seed UI that calls the Gormes backend profile-seed API, offers Create from seed in the chat/profile flow, renders the returned editable draft fields, requires explicit workspace path entry or confirmation, applies only through the backend, and then shows the new profile as a contact. The Flutter app must not write TOML or infer/grant workspace roots on its own.
- Trust class: operator, gateway, system
- Ready when: `Navivox natural-language profile seed backend API` has landed so the app can call a real server-authoritative draft/apply API., Navivox app setup/connect flow can reach the configured Gormes HTTP gateway., The UI preserves the server-authoritative config model; the Flutter app requests creation and renders/edit drafts, it does not write TOML.
- Not ready when: The backend profile seed API is missing or unvalidated., The app writes profile config files directly instead of calling Gormes., The UI grants workspace paths inferred from the seed without explicit operator confirmation., The slice adds campaigns, bulk outbound calls, telephony transfer, or call-center scheduling.
- Degraded mode: Without a model/provider, seed creation uses a deterministic local template and marks generation_source=template; invalid or risky seeds are rejected with redacted typed evidence and do not mutate profile config or workspace roots.
- Fixture: `../navivox-app/test/features/profiles/profile_seed_flow_test.dart`
- Write scope: `../navivox-app/lib/features/chat/`, `../navivox-app/lib/features/profiles/`, `../navivox-app/test/features/chat/`, `../navivox-app/test/features/profiles/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `sh -c 'cd ../navivox-app && flutter test test/features/profiles test/features/chat/profile_contact_list_test.dart'`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Navivox Flutter can request a profile seed draft from Gormes, edit/confirm it safely, apply it through the backend, and show the new contact without direct TOML writes or unconfirmed workspace access.
- Acceptance: Navivox profile creation exposes Create from seed, calls the backend draft API, and renders editable profile_id, display_name, instructions, provider/model, tool policy, voice metadata, and workspace suggestions., The app requires explicit workspace path entry/confirmation before apply and never writes TOML directly., After apply, the newly seeded profile appears in the profile/contact list and can be selected for a first chat turn., Provider-unconfigured backend responses render generation_source=template and redacted evidence without blocking first-run use.
- Source refs: ../navivox-app/navivox-ui-design.md:2.8, ../navivox-app/navivox-chat-ui-research.md:10, ../navivox-app/lib/features/chat/, ../navivox-app/lib/features/profiles/, internal/channels/navivox/channel.go, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md:Navivox HTTP/WS Flutter channel
- Unblocks: Navivox per-profile BYO voice profiles
- Why now: Unblocks Navivox per-profile BYO voice profiles.

## 2. Navivox per-profile BYO voice profiles Flutter UI

- Phase: 9 / 9.F
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Add the sibling Navivox Flutter profile/config controls that consume the backend voice-profile contract without writing config files or storing raw secrets.
- Trust class: -
- Ready when: `Navivox per-profile BYO voice profiles backend API` is complete and exposes authenticated read/validate fixtures., The app has authenticated GatewayNavivoxChannel wiring for config/profile backend calls.
- Not ready when: The Flutter app writes profile config TOML directly., Any widget, fixture, log, or snapshot stores raw provider credentials., The UI adds telephony, scheduling, campaigns, or human handoff.
- Degraded mode: -
- Fixture: `../navivox-app/test/features/profiles/; ../navivox-app/test/features/config/`
- Write scope: `../navivox-app/lib/features/profiles/`, `../navivox-app/lib/features/config/`, `../navivox-app/test/features/profiles/`, `../navivox-app/test/features/config/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `sh -c 'cd ../navivox-app && flutter test test/features/profiles test/features/config test/features/voice'`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Navivox Flutter lets operators inspect and edit per-profile voice settings through the backend contract; secrets stay write-only and degraded voice states remain explicit.
- Acceptance: Profile create/edit UI can set and display voice profile fields from the backend schema/read model., Credential controls show status/source refs only and route changes through backend safe config/admin flows., Voice provider fallback evidence from run records is visible after a fake or real voice turn., Text chat remains usable when voice providers are unavailable.
- Source refs: ../navivox-app/navivox-ui-design.md:2.8, ../navivox-app/lib/features/profiles/, ../navivox-app/lib/features/config/, docs/content/building-gormes/architecture_plan/progress.json:Navivox per-profile BYO voice profiles backend API
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Navivox safe config admin Flutter UI

- Phase: 9 / 9.F
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Render the Navivox config admin backend contract in the sibling Flutter app: schema-driven controls, redacted current values, diff/validate/apply confirmation, secret set/rotate/delete/test actions, and reload-or-pending-restart status. Flutter consumes backend schema and actions only; it never edits config.toml, .env, or raw secret values directly.
- Trust class: operator, gateway, system
- Ready when: `Navivox safe config admin backend API` has landed and exposes schema/get/diff/validate/apply/reload-or-pending-restart fixtures., The app has an authenticated GatewayNavivoxChannel connection and can route config-admin requests through the backend contract., The UI renders only schema-provided fields and action metadata; no free-form TOML editor.
- Not ready when: The backend API child row is still planned or failing validation., Flutter attempts to edit config.toml or .env directly., Any widget, log, fixture, or snapshot stores raw secret values.
- Degraded mode: When config validation fails, Navivox shows typed field errors and keeps the last-good server config active; secret values are never returned, logged, or echoed, only status/source/redacted evidence.
- Fixture: `../navivox-app/test/features/config/config_screen_test.dart`
- Write scope: `../navivox-app/lib/core/channel/navivox_channel.dart`, `../navivox-app/lib/core/channel/gateway_navivox_channel.dart`, `../navivox-app/lib/features/config/`, `../navivox-app/test/features/config/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `sh -c 'cd ../navivox-app && flutter test test/features/config'`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Navivox Flutter can inspect, validate, and apply safe config changes through backend schema/actions only; secret values stay write-only and pending_restart/reload evidence is visible.
- Acceptance: Schema-driven controls render supported safe config fields and current redacted values from the backend., Diff and validate responses render exact non-secret before/after confirmation and field-scoped errors., Secret controls render set/rotate/delete/test actions and status/source evidence without reading or storing raw secret values., Apply success renders reload_applied or pending_restart evidence from the backend.
- Source refs: ../navivox-app/navivox-decision-record.md:141, ../navivox-app/lib/core/channel/navivox_channel.dart, ../navivox-app/lib/core/channel/gateway_navivox_channel.dart, ../navivox-app/lib/features/config/screens/config_screen.dart, ../navivox-app/test/features/config/config_screen_test.dart, docs/content/building-gormes/architecture_plan/progress.json:Navivox safe config admin backend API
- Unblocks: Navivox per-profile BYO voice profiles
- Why now: Unblocks Navivox per-profile BYO voice profiles.

## 4. Navivox structured tool event cards Flutter UI

- Phase: 9 / 9.F
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: The sibling Navivox Flutter app consumes the Gormes backend structured tool-progress contract, upserts one durable ToolCallCard per tool_call_id for started/updated/finished states, renders redacted artifact rows, and never converts tool events into assistant prose.
- Trust class: operator, gateway, system
- Ready when: `Navivox structured tool event cards backend API` is complete and exposes started/updated/finished backend fixtures., Flutter already has a basic NavivoxToolCall model and ToolCall body renderer to extend.
- Not ready when: The UI writes backend progress as assistant text instead of structured ToolCallCard messages., The UI displays raw tool arguments, stdout, credentials, or full logs., The UI changes the Gormes backend contract or requires non-repo credentials to validate.
- Degraded mode: If a channel lacks structured tool progress, gateway falls back to existing bounded text progress; if Navivox receives malformed tool metadata, it renders redacted error evidence inside a tool card instead of assistant text.
- Fixture: `../navivox-app/test/core/channel/gateway_navivox_channel_test.dart; ../navivox-app/test/features/chat/`
- Write scope: `../navivox-app/lib/core/gateway/navivox_gateway_protocol.dart`, `../navivox-app/lib/core/channel/gateway_navivox_channel.dart`, `../navivox-app/lib/core/protocol/navivox_event.dart`, `../navivox-app/lib/features/chat/widgets/simple_chat_adapter.dart`, `../navivox-app/test/core/channel/gateway_navivox_channel_test.dart`, `../navivox-app/test/features/chat/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `sh -c 'cd ../navivox-app && flutter test test/core/channel/gateway_navivox_channel_test.dart test/features/chat'`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Navivox Flutter renders backend tool-progress lifecycle events as durable cards with redacted status/artifact evidence and no assistant-prose leakage.
- Acceptance: GatewayNavivoxChannel upserts a single ToolCall message per tool_call_id for started, updated, and finished backend events., ToolCallCard renders status, summary, approval state when present, and bounded artifact rows with id/kind/title/summary/ref., Malformed or oversized event metadata is truncated/redacted in UI fixtures and never becomes assistant prose., Existing chat fixtures remain green for normal assistant streaming and final messages.
- Source refs: docs/content/building-gormes/architecture_plan/progress.json:Navivox structured tool event cards backend API, ../navivox-app/lib/core/protocol/navivox_event.dart:NavivoxToolCall, ../navivox-app/lib/core/channel/gateway_navivox_channel.dart:_upsertToolCall, ../navivox-app/lib/features/chat/widgets/simple_chat_adapter.dart:_ToolCallBody
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. Navivox voice run records Flutter inspection UI

- Phase: 9 / 9.F
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Render the sibling Flutter Navivox inspection surface for backend run records after the Gormes run-record API lands. The app should fetch or receive redacted run records, show text and voice transcript evidence, STT/TTS metadata, tool timeline cards, attachment/artifact refs, terminal status, and provider usage/cost with explicit unknown states. This row is intentionally cross-root and must not be selected during repo-only Gormes iterations.
- Trust class: operator, gateway, system
- Ready when: `Navivox voice run records backend API` has landed and exposes a redacted JSON read model for text and voice turns., The sibling Navivox app already has the connect-and-talk chat fixture and can identify the active session/run after a turn ends., The UI keeps raw audio, provider secrets, and direct Gormes store access out of Flutter state.
- Not ready when: The backend run-record API is still missing or returns only mock data., The slice stores raw audio by default or hides retention status., The slice reports fake token cost when provider usage is absent., The slice edits Gormes backend persistence instead of consuming the backend contract from the app side.
- Degraded mode: If audio bytes, STT, TTS, or usage data are unavailable, the run record stores explicit unavailable evidence and preserves the text transcript/tool timeline instead of dropping the run or faking costs.
- Fixture: `../navivox-app/test/features/chat/navivox_run_records_test.dart`
- Write scope: `../navivox-app/lib/core/protocol/navivox_event.dart`, `../navivox-app/lib/core/channel/gateway_navivox_channel.dart`, `../navivox-app/lib/features/chat/`, `../navivox-app/test/core/channel/gateway_navivox_channel_test.dart`, `../navivox-app/test/features/chat/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `sh -c 'cd ../navivox-app && flutter test test/core/channel/gateway_navivox_channel_test.dart test/features/chat'`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Navivox Flutter can inspect backend-produced text and voice run records with transcript, tool timeline, STT/TTS metadata, artifacts, terminal status, and explicit usage/cost unknown states.
- Acceptance: After a text turn ends, the app can open a run detail or transcript panel showing redacted transcript, status, timestamps, session/run ids, and provider usage when available., After a voice turn ends, the app renders device transcript, optional server STT evidence, audio duration/codec metadata, TTS metadata, and explicit raw-audio retention status., Typed tool timeline entries render as tool cards or bounded timeline rows instead of assistant prose., Provider usage/cost fields render as `unknown` or unavailable when absent; the UI never fabricates zero cost.
- Source refs: internal/apiserver/runs.go:runRecord, internal/channels/navivox/channel.go:sessionState, ../navivox-app/lib/core/protocol/navivox_event.dart:NavivoxVoiceMessage, ../navivox-app/lib/core/channel/gateway_navivox_channel.dart, ../navivox-app/lib/features/chat/, https://docs.dograh.com/core-concepts/how-dograh-works
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Hermes integrations claim audit + source-backed plugin/skill parity map

- Phase: 8 / 8.C
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Turn the sanitized Reddit/WebAfterAI Hermes integrations post into a source-backed parity map without accepting marketing shorthand as fact: classify each named integration as first-party bundled skill, bundled plugin/backend, gateway/platform/tool, optional skill, indirect web/browser/MCP/scraping workflow, Gormes-owned candidate, or unsupported/excluded claim. The audit must explicitly handle cases where a workflow is achievable through generic web scraping, browser automation, MCP, or Firecrawl-style extraction without being a direct Hermes plugin or tool, and it must not create implementation rows for Reddit, Stripe, InsForge, Graphiti/Zep, or Fireflies unless exact current Hermes source appears.
- Trust class: operator, system
- Ready when: The audit uses only sanitized transcript text plus checked-in Hermes/Gormes source refs; no live private ~/.hermes, credentials, browser sessions, or external API accounts are read., Each of the 12 post items is classified with source refs or explicit unsupported/excluded evidence., Indirect capabilities are allowed as a separate class: generic web scraping, browser automation, Firecrawl extraction, MCP, or skill workflows may satisfy a use case without proving a direct Hermes plugin/tool exists.
- Not ready when: The row is used to implement all integrations in one pass instead of producing a bounded source-backed audit/map., Unsupported claims are copied into README/docs as if they were Hermes-native integrations., The audit treats `hermes plugins install reddit\|stripe\|insforge\|graphiti\|fireflies` as valid without exact current Hermes source or an external plugin repository URL., The audit reads live user config, token stores, memory databases, or private home directories as evidence.
- Degraded mode: Until the claims are source-classified, public roadmap and parity work can overstate Hermes/Gormes integration breadth by treating scraped workflows, optional skills, and unsupported social-post claims as native plugins.
- Fixture: `webpages/docs/content/building-gormes/architecture_plan/hermes-integrations-claim-audit.md`
- Write scope: `webpages/docs/content/building-gormes/architecture_plan/hermes-integrations-claim-audit.md`, `webpages/docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md`, `webpages/docs/content/building-gormes/architecture_plan/upstream-coverage-ledger.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`, `README.md`
- Test commands: `go run ./cmd/progress validate`, `go test ./webpages/docs -run 'TestUpstreamCoverageLedgerMatchesSourceClasses\|TestProgressCanonical' -count=1`, `git diff --check`
- Done signal: Report the 12-row classification table, exact Hermes source refs, unsupported/excluded claims, indirect scraping/browser/MCP classifications, and any newly-created follow-up progress row names.
- Acceptance: A checked-in audit document or architecture-plan section lists all 12 post items and classifies each as direct first-party skill/plugin/tool/gateway, optional skill, indirect scraping/browser/MCP workflow, Gormes-owned candidate, or unsupported/excluded., The audit explicitly notes that some user-visible workflows are not direct tools: e.g. competitor/site/reddit-style research may be covered by generic web search/extract/crawl, browser automation, or future MCP/web-scraping rows rather than a named Hermes Reddit plugin., Unsupported/excluded claims for Reddit, Stripe, InsForge, Graphiti/Zep, and Fireflies remain excluded or row-backed as discovery-only until source refs are found., Any follow-up implementation intent is routed into separate small progress rows by source class; this audit row does not broaden into a 12-integration implementation batch., Public messaging/docs are updated only with evidence-backed wording and avoid inflated integration counts.
- Source refs: sanitized user-provided Reddit/WebAfterAI transcript 2026-05-24: '12 Hermes Integrations That Actually Matter', hermes-agent/hermes_cli/plugins_cmd.py@43e566f77: `hermes plugins install` clones Git plugins into ~/.hermes/plugins and does not imply a built-in short-name registry for every social-post claim, hermes-agent/hermes_cli/plugins.py@43e566f77: bundled/user/project/pip plugin discovery and opt-in semantics, hermes-agent/skills/productivity/google-workspace/SKILL.md@43e566f77: first-party Gmail/Calendar/Drive/Docs/Sheets skill, hermes-agent/skills/note-taking/obsidian/SKILL.md@43e566f77: filesystem-first Obsidian vault skill, hermes-agent/plugins/web/firecrawl/plugin.yaml@43e566f77 and provider.py: bundled Firecrawl web backend with direct/gateway/self-hosted config, hermes-agent/tools/web_tools.py@43e566f77: generic web_search/web_extract/web_crawl dispatch; supports web-scraping/extraction workflows without naming them as native integrations, hermes-agent/skills/github/DESCRIPTION.md@43e566f77 and skills/github/*/SKILL.md: GitHub auth/repo/issues/PR/code-review skills, hermes-agent/skills/media/youtube-content/SKILL.md@43e566f77: YouTube transcript helper skill, hermes-agent/gateway/platforms/discord.py@43e566f77 and hermes-agent/tools/discord_tool.py@43e566f77: Discord gateway and Discord admin/core tools, hermes-agent/optional-skills/productivity/telephony/SKILL.md@43e566f77 and scripts/telephony.py: Twilio, Bland.ai, and Vapi optional telephony skill, hermes-agent/gateway/platforms/sms.py@43e566f77: Twilio-backed SMS gateway contract, repository search 2026-05-24: no first-party Hermes refs found for reddit, stripe API plugin, insforge, graphiti/zep, or fireflies beyond incidental text
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 7. CLIProxyAPI-compatible upstream route adapter

- Phase: 4 / 4.A
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Priority: `P3`
- Contract: After the Gormes Router MVP exists, optionally allow a CLIProxyAPI server to be configured as a normal OpenAI-compatible upstream base URL. This must not import CLIProxyAPI runtime code, management APIs, OAuth automation, or multi-account pooling; it only treats CLIProxyAPI as a user-configured upstream endpoint.
- Trust class: operator, system
- Ready when: The builder uses fake providers/httptest and checked-in fixtures, not live credentials or locally installed Ollama/LM Studio., The implementation preserves the user-owned credential boundary from the router plan.
- Not ready when: The slice claims free/unlimited LLM access, requires Ollama/LM Studio, automates OAuth/browser token capture, or copies CLIProxyAPI runtime code., Secrets appear in logs, status JSON, docs, progress evidence, or tests.
- Degraded mode: -
- Fixture: `internal/provider/router/cliproxy_upstream_test.go`
- Write scope: `internal/provider/router/`, `internal/config/`, `cmd/gormes/`, `webpages/docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/provider/router -run 'TestRouterCLIProxyAPIUpstream' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Report focused fake-provider test output, redaction evidence, and progress validation.
- Acceptance: A CLIProxyAPI-style upstream can be represented as a custom OpenAI-compatible route with base_url and api_key_env., The adapter only relies on /v1/models and /v1/chat/completions-compatible behavior., No OAuth, management API, WebSocket, token scraping, or account-pool semantics are added., Tests use httptest fake CLIProxyAPI-compatible responses and redaction assertions.
- Source refs: docs/content/building-gormes/architecture_plan/gormes-router-plan.md:Config schema, CLIProxyAPI@50d19e2 README.md: /v1/chat/completions and provider route aliases, CLIProxyAPI@50d19e2 docs/sdk-advanced.md: /v1/models exposure, internal/hermes/provider_transport.go:chat_completions transport
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
