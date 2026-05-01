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
## 1. Goncho honcho_reasoning LLM-backed synthesis

- Phase: 5 / 5.V
- Owner: `goncho`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: HonchoReasoningTool must use Gormes' native goncho.DialecticCaller seam for LLM-backed synthesis, matching Hermes HonchoMemoryProvider's honcho_reasoning tool path through manager.dialectic_query. Existing Gormes code already has Service.SetDialecticCaller/DialecticCaller and a tool-level caller branch, but this row remains open until the caller path is fixture-locked and the production Goncho service wiring installs a native Go provider-backed caller. Keep deterministic fallback with degraded evidence when no caller is configured or the caller fails.
- Trust class: operator, system
- Ready when: Write RED tests for caller success and caller failure in internal/gonchotools before modifying production code, Add the smallest production wiring fixture for native provider-backed DialecticCaller installation, Keep existing deterministic no-caller fallback behavior and evidence stable
- Not ready when: The slice removes deterministic fallback entirely, The slice changes honcho_reasoning tool schema, The slice depends on launching Python hermes-agent or Honcho services instead of a native Go caller seam
- Degraded mode: When no DialecticCaller is configured, returns deterministic answer with reasoning_llm_unavailable evidence; when the caller errors, returns deterministic answer with reasoning_llm_failed evidence.
- Fixture: `internal/gonchotools/honcho_tools_test.go`
- Write scope: `internal/goncho/types.go`, `internal/goncho/service.go`, `internal/gonchotools/honcho_tools.go`, `internal/gonchotools/honcho_tools_test.go`, `production Goncho service construction site that owns provider/client wiring`
- Test commands: `go test ./internal/gonchotools -run 'TestHonchoReasoningTool_(UsesDialecticCaller\|DialecticCallerFailureFallsBack\|ReturnsDeterministicAnswer)' -count=1`, `go test ./internal/goncho ./internal/gonchotools -count=1`, `go run ./cmd/progress validate`
- Done signal: honcho_reasoning is covered by caller-success, caller-failure, and no-caller fallback tests, and production Goncho construction wires a native provider-backed DialecticCaller.
- Acceptance: TestHonchoReasoningTool_UsesDialecticCaller proves honcho_reasoning sends peer, query, and a context-backed system prompt to goncho.DialecticCaller and returns the LLM answer without reasoning_llm_unavailable evidence., TestHonchoReasoningTool_DialecticCallerFailureFallsBack proves caller errors keep the deterministic answer and emit reasoning_llm_failed evidence., A production wiring fixture proves the normal kernel/config Goncho service installs a DialecticCaller; no Python hermes-agent runtime process is required.
- Source refs: /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:plugins/memory/honcho/__init__.py:91-128, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:plugins/memory/honcho/__init__.py:1248-1261, /home/xel/git/sages-openclaw/workspace-gormes/gormes-agent@81d89846e:internal/goncho/types.go:316-318, /home/xel/git/sages-openclaw/workspace-gormes/gormes-agent@81d89846e:internal/gonchotools/honcho_tools.go:197-212
- Why now: P0 handoff; needs contract proof before closeout.

## 2. Web dashboard server shell + degraded inventory

- Phase: 5 / 5.V
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: Create the Go web dashboard server shell that matches Hermes' FastAPI dashboard startup, static SPA serving, localhost security guard, public API allowlist, and degraded inventory semantics before any React feature work lands.
- Trust class: operator, system
- Ready when: A small internal/apiserver package can be tested without launching live uvicorn/React tooling, The public API allowlist is fixture-locked from Hermes web_server.py
- Not ready when: The slice ports React pages or PTY chat before the server/security/degraded contract exists, The implementation allows non-loopback Host headers by default
- Degraded mode: When the dashboard dist is absent or disabled, API status exposes dashboard_available=false with missing_dist evidence and the server does not claim React parity.
- Fixture: `internal/apiserver/dashboard_server_test.go`
- Write scope: `internal/apiserver/dashboard_server.go`, `internal/apiserver/dashboard_server_test.go`, `cmd/gormes`
- Test commands: `go test ./internal/apiserver -run 'TestDashboard(Server\|API\|Unavailable)' -count=1`, `go run ./cmd/progress validate`
- Done signal: internal/apiserver exposes a tested dashboard server shell with static SPA serving, Host/session-token guards, public endpoint allowlist, and unavailable inventory.
- Acceptance: TestDashboardServerServesSPAAndDist proves a loopback-only server serves index.html and static assets from an embedded or configured dist without exposing arbitrary filesystem paths., TestDashboardAPIGuardsHostAndSessionToken proves localhost Host validation, CORS/session-token gating, and public endpoint allowlist behavior for status/defaults/model/theme/plugin discovery., TestDashboardUnavailableInventory proves missing React dist returns a deterministic JSON inventory with unavailable/degraded evidence instead of starting a broken server.
- Source refs: /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:hermes_cli/web_server.py:64-110, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:hermes_cli/web_server.py:194-240, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/index.html, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/vite.config.ts
- Why now: P0 handoff; needs contract proof before closeout.

## 3. Web dashboard React/Vite scaffold + 9-page route manifest

- Phase: 5 / 5.V
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P0`
- Contract: Port the minimal Vite/React scaffold and route manifest for Hermes' 9-page dashboard without filling each page's detailed behavior yet.
- Trust class: operator, system
- Ready when: Dashboard server shell child row exists, Node tooling is isolated under web/ and Go tests do not require network access
- Not ready when: The slice ports full page functionality instead of route/fallback scaffolding, The server shell row is not present as the embedding target
- Degraded mode: Each route can render a typed unavailable panel with the API endpoint it needs; no blank page or unhandled promise rejection is acceptable.
- Fixture: `web/src/App.test.tsx`
- Write scope: `web/package.json`, `web/src/App.tsx`, `web/src/main.tsx`, `web/src/pages/`, `internal/apiserver/dashboard_assets.go`
- Test commands: `npm --prefix web test -- --run DashboardFrontendScaffold`, `go test ./internal/apiserver -run TestDashboardDistManifest -count=1`, `go run ./cmd/progress validate`
- Done signal: web/ exists with React/Vite scaffold, route manifest, and fallback-rendering tests for all Hermes dashboard pages.
- Acceptance: TestDashboardFrontendScaffoldDefinesRoutes proves the SPA route manifest includes dashboard, chat, config, env, sessions, logs, cron, skills, docs, and analytics entries matching Hermes page files., TestDashboardFrontendBuildManifest proves the new web package has Vite/React build metadata and emits a deterministic dist manifest for the Go server shell., TestDashboardPageFallbacks prove pages render explicit unavailable panels when their backing API endpoints are not implemented yet.
- Source refs: /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/package.json, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/App.tsx, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/main.tsx, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/pages/AnalyticsPage.tsx, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/pages/ChatPage.tsx, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/pages/ConfigPage.tsx, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/pages/CronPage.tsx, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/pages/DocsPage.tsx, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/pages/EnvPage.tsx, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/pages/LogsPage.tsx, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/pages/SessionsPage.tsx, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/pages/SkillsPage.tsx
- Why now: P0 handoff; needs contract proof before closeout.

## 4. Skill registries

- Phase: 5 / 5.F
- Owner: `skills`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Native skills hub registry providers expose source-backed, read-only metadata for the current Hermes skills hub source adapters before any write-capable install flow: OptionalSkillSource, HermesIndexSource, SkillsShSource, WellKnownSkillSource, UrlSource, GitHubSource, ClawHubSource, ClaudeMarketplaceSource, and LobeHubSource are the active upstream contract at Hermes 69d4800d. This executable slice should add only the missing remote registry read-model providers over the existing HubRegistryProvider/Search seam, with source filtering, trust normalization, centralized-index preference, stale-cache fallback, and typed degraded evidence for unavailable, malformed, timeout, empty, and rate-limited upstreams. Url direct parsing and optional bundled-skill inventory stay in their existing rows; this slice must not install, activate, quarantine, guard-scan, or mutate skills.
- Trust class: operator, system
- Ready when: The existing internal/skills HubRegistryProvider/Search read model remains the public seam for registry metadata., Tests inject fake HTTP clients, response fixtures, or temp cache roots for Skills.sh/GitHub, WellKnown, HermesIndex, ClawHub, ClaudeMarketplace, and LobeHub; no live network, GitHub token, gh CLI, active skill store, or quarantine directory is required., UrlSource direct SKILL.md parsing is treated as already covered by the separate complete `Skills hub direct URL candidate parser` row rather than reimplemented here.
- Not ready when: The slice downloads arbitrary bundle files, writes active/candidate skills, performs guard scans, runs install commands, or changes skill prompt injection., The slice omits current Hermes 69d4800d source adapters (HermesIndexSource, ClaudeMarketplaceSource, or LobeHubSource) from the read-model contract, or treats the centralized index preference in parallel_search_sources as out of scope without a source-backed split row., ClawHub, Skills.sh, WellKnown, or GitHub results are treated as builtin/trusted without the upstream trust rules, or malformed remote payloads panic instead of returning typed degraded evidence.
- Degraded mode: Network failures, non-200 responses, malformed JSON, expired/missing cache, empty registries, and rate limits return typed evidence such as registry_unavailable, registry_rate_limited, registry_malformed, registry_cache_stale, or registry_empty without panics and without active-store mutation.
- Fixture: `internal/skills/hub_registry_sources_test.go`
- Write scope: `internal/skills/hub_registry_sources.go`, `internal/skills/hub_registry_sources_test.go`, `internal/skills/hub_search.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/skills -run 'TestClawHubProvider\|TestHermesIndexProvider\|TestClaudeMarketplaceProvider\|TestLobeHubProvider\|TestSkillsShProvider\|TestWellKnownRegistryProvider\|TestRegistryProvider' -count=1`, `go test ./internal/skills -count=1`, `go run ./cmd/progress validate`
- Done signal: Native registry providers expose current Hermes HermesIndex, Skills.sh/GitHub, WellKnown, ClawHub, ClaudeMarketplace, and LobeHub metadata through HubRegistryProvider/Search with fixture-backed cache/degraded evidence tests, centralized-index source filtering, and no install/store mutation.
- Acceptance: TestClawHubProviderCommunityTrustAndDegradedEvidence proves ClawHub search/inspect normalizes slug/name/tags, assigns community trust, reports registry_unavailable or registry_rate_limited for failures, and never mutates the active store., TestHermesIndexProviderPrefersCachedIndex proves centralized Hermes index fixtures return metadata with zero API calls and source-router search can skip duplicate remote API sources when the index is available and source_filter=all., TestClaudeMarketplaceProviderCommunityTrustAndCacheEvidence proves marketplace.json fixtures resolve source paths, normalize trust through TRUSTED_REPOS, and report typed malformed/unavailable evidence without store writes., TestLobeHubProviderAgentMetadataAndDegradedEvidence proves LobeHub agent index fixtures convert title/identifier/tags into community-trust metadata and return typed timeout/malformed evidence instead of panicking., TestSkillsShProviderDelegatesThroughGitHubMetadata proves Skills.sh result identifiers resolve to GitHub metadata/fetch IDs while preserving source=skills-sh and deterministic source-filter behavior., TestWellKnownRegistryProviderReadsIndexMetadata proves .well-known/skills index fixtures produce community-trust metadata without network or filesystem writes beyond the temp cache., TestRegistryProviderCacheFallback proves malformed or unavailable network responses reuse a valid stale cache when present and return typed evidence when no cache exists., TestRegistryProvidersDoNotInstall proves the registry-source package has no dependency on active store mutators, quarantine install paths, gateway adapters, or provider/model clients.
- Source refs: /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:GitHubSource.search,GitHubSource.fetch,GitHubSource.inspect, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:SkillsShSource.search,SkillsShSource.fetch,SkillsShSource.inspect,_discover_identifier, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:WellKnownSkillSource.search,WellKnownSkillSource.fetch,WellKnownSkillSource.inspect, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:ClawHubSource.search,ClawHubSource.fetch,ClawHubSource.inspect,_load_catalog_index, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:HermesIndexSource.search,fetch,inspect,is_available,_load_hermes_index, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:ClaudeMarketplaceSource.search,fetch,inspect,_fetch_marketplace_index, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:LobeHubSource.search,fetch,inspect,_fetch_index,_fetch_agent, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tools/skills_hub.py:create_source_router,parallel_search_sources,unified_search, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tests/tools/test_skills_hub.py:TestSkillsShSource,TestWellKnownSkillSource,TestUrlSource,TestSkillSourceRouter,TestUnifiedSearch, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d/tests/tools/test_skills_hub_clawhub.py:TestClawHubSource, internal/skills/hub_search.go, internal/skills/hub_search_test.go, internal/skills/hub_registry_sources.go:WellKnownRegistryProvider,ClawHubRegistryProvider, internal/skills/hub_registry_sources_test.go:TestWellKnownRegistryProviderReadsIndexMetadata,TestClawHubProviderCommunityTrustAndDegradedEvidence,TestClawHubProviderDegradedEvidence, internal/skills/url_candidate.go
- Unblocks: Skills hub install binding over registry metadata, Skills hub source filter CLI/RPC, Skill registries unavailable-network UX fixtures
- Why now: Unblocks Skills hub install binding over registry metadata, Skills hub source filter CLI/RPC, Skill registries unavailable-network UX fixtures.

## 5. Web dashboard core components + data-state fixtures

- Phase: 5 / 5.V
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Port Hermes dashboard's reusable component set as typed React fixtures so later page slices compose parity components instead of re-inventing UI behavior per page.
- Trust class: operator, system
- Ready when: React/Vite scaffold row is present, Fixture data contracts are stable enough for page slices to reuse
- Not ready when: The slice depends on real provider keys, gateway processes, or live cron/session databases, The slice changes API contracts instead of consuming typed fixtures
- Degraded mode: Components render explicit empty/error states with redacted diagnostic text when data is absent or APIs are unavailable.
- Fixture: `web/src/components/dashboard-components.test.tsx`
- Write scope: `web/src/components/`, `web/src/components/ui/`, `web/src/hooks/`, `web/src/contexts/`
- Test commands: `npm --prefix web test -- --run DashboardCoreComponents`, `go run ./cmd/progress validate`
- Done signal: web/src/components and hooks cover Hermes dashboard core UI primitives with data-state fixtures.
- Acceptance: TestDashboardCoreComponentsRenderDataStates proves core cards, markdown, model info, platform, slash popover, toast, and tool-call components render loading/empty/error/success states from typed fixtures., TestDashboardSidebarAndHeaderState proves sidebar status, page header, language/theme controls, and destructive-confirm dialogs preserve user-visible labels and keyboard affordances., Component fixtures avoid live gateway/provider calls and can run in CI without credentials.
- Source refs: /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/components/, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/components/ui/, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/hooks/, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/contexts/
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Web dashboard PTY chat + event websocket fixtures

- Phase: 5 / 5.V
- Owner: `gateway`
- Size: `large`
- Status: `planned`
- Priority: `P1`
- Contract: Port the dashboard embedded-chat/PTY websocket contract and React chat integration with fakes before binding to a live terminal or provider runtime.
- Trust class: operator, system
- Ready when: Dashboard server shell has session-token gating, React route scaffold includes ChatPage fallback
- Not ready when: The slice launches a real shell/provider in tests, The slice exposes chat before session-token and enablement gates are proven
- Degraded mode: When embedded chat is disabled, websocket attempts close with dashboard_chat_disabled evidence and ChatPage renders instructions rather than hanging.
- Fixture: `internal/apiserver/dashboard_pty_test.go`
- Write scope: `internal/apiserver/dashboard_pty.go`, `internal/apiserver/dashboard_pty_test.go`, `web/src/pages/ChatPage.tsx`, `web/src/lib/gatewayClient.ts`
- Test commands: `go test ./internal/apiserver -run 'TestDashboard(PTY\|Event\|Chat)' -count=1`, `npm --prefix web test -- --run DashboardChatPage`, `go run ./cmd/progress validate`
- Done signal: Go apiserver websocket tests and React ChatPage fixtures cover PTY/chat/event streaming without live credentials.
- Acceptance: TestDashboardPTYWebsocketNegotiates proves /api/pty rejects disabled or unauthenticated sessions, accepts enabled fake sessions, and redacts command/environment details in close reasons., TestDashboardChatPageStreamsToolAndAssistantEvents proves ChatPage consumes PTY/event fixtures for user text, assistant deltas, tool-call panels, completion, and error states., TestDashboardEventWebsocketsFanout proves /api/ws, /api/pub, and /api/events compatibility shims either stream typed events or return a documented unavailable status.
- Source refs: /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:hermes_cli/web_server.py:77-79, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:hermes_cli/web_server.py:2401-2588, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/pages/ChatPage.tsx, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/lib/gatewayClient.ts
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 7. Web dashboard theme catalog + switcher parity

- Phase: 5 / 5.V
- Owner: `gateway`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Port Hermes dashboard theme catalog and theme-selection API as a small independent slice, separate from TUI skin parity and page functionality.
- Trust class: operator, system
- Ready when: Dashboard server shell exposes public theme catalog endpoint, React scaffold can mount ThemeSwitcher fixtures
- Not ready when: The slice conflates dashboard themes with terminal skins without a mapping fixture, Theme persistence writes secrets or unrelated config keys
- Degraded mode: Unknown or unavailable themes fall back to the default dashboard theme with invalid_theme evidence; TUI skins are not used as a substitute unless explicitly mapped.
- Fixture: `internal/apiserver/dashboard_theme_test.go`
- Write scope: `internal/apiserver/dashboard_theme.go`, `internal/apiserver/dashboard_theme_test.go`, `web/src/themes/`, `web/src/components/ThemeSwitcher.tsx`
- Test commands: `go test ./internal/apiserver -run TestDashboardTheme -count=1`, `npm --prefix web test -- --run DashboardTheme`, `go run ./cmd/progress validate`
- Done signal: Dashboard theme API and React context/switcher are fixture-backed and list the same presets as Hermes web/src/themes.
- Acceptance: TestDashboardThemeCatalogMatchesHermes proves all Hermes dashboard theme presets, color tokens, and display names are available through the Go API and React theme context., TestDashboardThemeSelectionPersists proves PUT /api/dashboard/theme stores a valid selected theme and rejects unknown values with redacted errors., TestDashboardThemeSwitcherFixture proves the React ThemeSwitcher applies theme variables without reloading or losing current route state.
- Source refs: /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:hermes_cli/web_server.py:2919-2962, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/themes/, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/components/ThemeSwitcher.tsx
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 8. Web dashboard OAuth provider flows + EN/ZH i18n

- Phase: 5 / 5.V
- Owner: `gateway`
- Size: `large`
- Status: `planned`
- Priority: `P1`
- Contract: Port the dashboard OAuth-provider UI contract and EN/ZH i18n catalog as fixture-backed surfaces; do not implement real browser OAuth beyond fake-provider endpoint shapes in this slice.
- Trust class: operator, system
- Ready when: Dashboard server shell has session-token gating, React core components include OAuth modal/card fixtures
- Not ready when: The slice performs live OAuth browser/device flows, The slice leaves English-only labels in newly ported dashboard UI
- Degraded mode: OAuth endpoints return oauth_provider_unavailable or credentials_missing with redacted evidence when a provider is not configured; untranslated labels fail tests rather than falling back silently.
- Fixture: `internal/apiserver/dashboard_oauth_test.go`
- Write scope: `internal/apiserver/dashboard_oauth.go`, `internal/apiserver/dashboard_oauth_test.go`, `web/src/components/OAuthLoginModal.tsx`, `web/src/components/OAuthProvidersCard.tsx`, `web/src/i18n/`
- Test commands: `go test ./internal/apiserver -run 'TestDashboard(OAuth\|I18n)' -count=1`, `npm --prefix web test -- --run 'Dashboard(OAuth\|I18n)'`, `go run ./cmd/progress validate`
- Done signal: OAuth provider API fakes, React OAuth modal/card fixtures, and EN/ZH i18n catalogs are tested without real credentials.
- Acceptance: TestDashboardOAuthProvidersUsesFakes proves provider listing, start, submit, poll, delete, and redacted error paths match Hermes endpoint shapes without requiring real OAuth credentials., TestDashboardI18nCatalog proves English and Chinese dashboard catalogs cover every route, component label, OAuth state, and unavailable/degraded message used by the scaffold., TestDashboardLanguageSwitcherFixture proves the language switcher updates rendered copy and preserves current route/theme state.
- Source refs: /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:hermes_cli/web_server.py:1290-1325, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:hermes_cli/web_server.py:1867-1933, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/components/OAuthLoginModal.tsx, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/components/OAuthProvidersCard.tsx, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/i18n/en.ts, /home/xel/git/sages-openclaw/workspace-mineru/hermes-agent@69d4800d:web/src/i18n/zh.ts
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
