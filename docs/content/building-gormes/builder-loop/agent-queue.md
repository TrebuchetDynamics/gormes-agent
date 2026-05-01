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
## 1. Web dashboard core components + data-state fixtures

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

## 2. Web dashboard PTY chat + event websocket fixtures

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

## 3. Web dashboard theme catalog + switcher parity

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

## 4. Web dashboard OAuth provider flows + EN/ZH i18n

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
