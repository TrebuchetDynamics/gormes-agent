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
## 1. Hermes Kanban slash/gateway/dashboard surfaces

- Phase: 5 / 5.M
- Owner: `gateway`
- Size: `large`
- Status: `in_progress`
- Priority: `P2`
- Contract: Gormes ports the operator surfaces around Kanban: /kanban routes in TUI/gateway use the same parser/output as gormes kanban, gateway status exposes dispatcher state and nudge capability, and the dashboard shows live Kanban tasks, lanes, filters, worker runs, and dispatcher nudges over authenticated Gormes dashboard routes.
- Trust class: operator, gateway, system
- Ready when: Hermes Kanban durable board core is complete., Dispatcher and worker-tool rows define the status/read-model events the dashboard should render., Dashboard authentication and WebSocket/event-stream patterns are validated for existing Gormes dashboard routes.
- Not ready when: Slash commands duplicate CLI parsing and diverge from gormes kanban output., Dashboard endpoints expose Kanban data without the active Gormes dashboard session token., The dashboard implies live dispatcher/worker features before the dispatcher row is complete.
- Degraded mode: When the dispatcher or dashboard stream is unavailable, operators see kanban_dispatcher_unavailable or kanban_dashboard_unavailable evidence; /kanban remains recognized and unavailable rather than leaking to the model.
- Fixture: `internal/gateway/kanban_command_test.go; internal/tui/kanban_slash_test.go; internal/dashboard/kanban_dashboard_test.go`
- Write scope: `internal/cli/`, `internal/gateway/`, `internal/tui/`, `internal/dashboard/`, `web/`, `cmd/gormes/kanban.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/gateway ./internal/cli -run TestKanbanSlash -count=1`, `go test ./internal/dashboard -run TestKanbanDashboard -count=1`, `go run ./cmd/progress validate`
- Done signal: /kanban, gateway status/nudge, and dashboard Kanban lanes are authenticated, share CLI output semantics, and surface unavailable dispatcher evidence clearly.
- Acceptance: Slash fixtures prove /kanban create/list/show/complete uses the same command implementation as gormes kanban and no slash text reaches the model., Gateway fixtures prove /kanban output is formatted for platform messages and respects active-turn policy., Dashboard route fixtures prove tasks, lanes, filters, runs, and dispatcher nudge endpoints require authentication and stream bounded updates., Status fixtures prove dispatcher state is operator-visible without reading live Hermes config.
- Source refs: ../hermes-agent/hermes_cli/commands.py@54e78cadb:CommandDef('kanban'), ../hermes-agent/hermes_cli/kanban.py@54e78cadb:run_slash, ../hermes-agent/gateway/run.py@54e78cadb:_handle_kanban_command, ../hermes-agent/plugins/kanban/dashboard@54e78cadb, ../hermes-agent/hermes_cli/kanban_specify.py@24d48ffb8:specify triage fleshing, ../hermes-agent/hermes_cli/kanban_diagnostics.py@7d66d30d7:tooltips + docs link
- Why now: Already active; contract metadata keeps execution bounded.

## 2. Nous OAuth device code + refresh token + agent key provisioning

- Phase: 5 / 5.O
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: Gormes ports Hermes' Nous OAuth device code login, refresh token rotation, and agent key minting pipeline. The credential_pool.go already holds the persistence schema (NousOAuthCredentials struct, SaveNousOAuthCredentials), but no actual OAuth handshake or token refresh logic exists. This row adds: (1) browser-based device code login flow matching Hermes' _nous_device_code_login, (2) refresh token rotation via X-Nous-Refresh-Token header matching _refresh_access_token, (3) short-lived (24h) agent API key minting from portal matching _mint_agent_key, and (4) runtime credential resolution orchestrating refresh→mint with retry on stale access tokens matching resolve_nous_runtime_credentials.
- Trust class: system
- Ready when: Credential pool persistence schema (NousOAuthCredentials + SaveNousOAuthCredentials) is present and validated., Hermes auth.py upstream contract is read and device-code/refresh/mint flow is understood.
- Not ready when: The row is skipped before the credential_pool.go schema and the Hermes upstream auth.py contract are reconciled., OAuth device code flow is implemented without redacted fixture coverage matching test_auth_nous_provider.py patterns.
- Degraded mode: When the Nous OAuth endpoint is unreachable or the refresh token is invalid, operators see a classified OAuth error (device_code_expired, refresh_token_revoked, agent_key_minting_failed) and are guided to re-run auth add nous.
- Fixture: `internal/hermes/nous_oauth_test.go`
- Write scope: `internal/hermes/nous_oauth.go`, `internal/hermes/nous_oauth_test.go`, `internal/config/credential_pool.go`, `cmd/gormes/auth.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/hermes -run TestNousOAuth -count=1`, `go test ./internal/config -run TestNousOAuthCredentialPool -count=1`, `go run ./cmd/progress validate`
- Done signal: Nous OAuth device code login, refresh token rotation, and agent key minting are fixture-backed with redacted test data matching Hermes test_auth_nous_provider.py patterns, and the credential pool round-trips persisted credentials.
- Acceptance: Device code login fixture proves the browser-based OAuth flow produces a valid access token and refresh token stored via SaveNousOAuthCredentials., Refresh token rotation fixture proves expired access tokens are refreshed via X-Nous-Refresh-Token header and the new token pair is persisted., Agent key minting fixture proves a short-lived agent API key is obtained from the portal after successful refresh., Runtime credential resolution fixture proves resolve flow orchestrates refresh→mint→retry for stale access tokens., Error classification fixture proves device_code_expired, refresh_token_revoked, and agent_key_minting_failed states produce actionable operator guidance.
- Source refs: ../hermes-agent/hermes_cli/auth.py:_nous_device_code_login, ../hermes-agent/hermes_cli/auth.py:_refresh_access_token, ../hermes-agent/hermes_cli/auth.py:_mint_agent_key, ../hermes-agent/hermes_cli/auth.py:resolve_nous_runtime_credentials, ../hermes-agent/hermes_cli/auth.py:persist_nous_credentials, ../hermes-agent/hermes_cli/auth.py:_shared_nous_store, ../hermes-agent/tests/hermes_cli/test_auth_nous_provider.py, internal/config/credential_pool.go:NousOAuthCredentials, internal/config/credential_pool.go:SaveNousOAuthCredentials, internal/hermes/provider_registry_manifest.go:nous ProviderOwned entry
- Unblocks: gormes auth add nous CLI command, Nous provider runtime credential resolution, Cross-profile Nous token sync (shared auth store)
- Why now: Unblocks gormes auth add nous CLI command, Nous provider runtime credential resolution, Cross-profile Nous token sync (shared auth store).

## 3. TD engineering blog scaffolded and live

- Phase: 8 / 8.A
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: TrebuchetDynamics has a publicly reachable engineering blog with a working Atom/RSS feed, an /about page that names the org and the methodology, and a deploy pipeline so a markdown commit becomes a published post without manual intervention. Hosting choice is owner's call (Astro/Hugo/Eleventy + Cloudflare/Vercel/GitHub Pages); the row is done when a stranger can subscribe to a feed and read one published post.
- Trust class: operator
- Ready when: Hosting choice and blog framework are decided (operator decision; not loop-driven)., A subdomain or path on an existing TD-controlled domain is available.
- Not ready when: The blog is private, password-protected, or behind authentication., There is no Atom/RSS feed at a stable URL., The first post is empty or placeholder text rather than the writeup #1 draft or a real introduction.
- Degraded mode: Without a publication outlet, every loop commit is invisible in the reputation market; the strategy described in success-plan.md cannot start.
- Fixture: `webpages/blog/ (or chosen blog repo path)`
- Write scope: `webpages/blog/ (or external blog repo path)`, `DNS / Cloudflare / hosting config (operator-only)`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: -
- No test required: Documentation/infrastructure row; success is the URL being live and the feed being reachable, validated by the acceptance checklist.
- Done signal: Public blog URL + feed URL recorded in success-plan.md and README.md.
- Acceptance: Blog is reachable at a public URL with at least one real (non-placeholder) post., An Atom or RSS feed exists at a stable, discoverable URL., Publishing a new post is a markdown-commit-and-merge operation; no console click-through required., An /about page exists that names TrebuchetDynamics and points at gormes-agent + agentic-porting-kit.
- Source refs: docs/content/building-gormes/strategy/success-plan.md, webpages/landing/
- Unblocks: Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline
- Why now: Unblocks Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline.

## 4. Hermes Kanban multi-board, workspace, and run-history parity

- Phase: 5 / 5.M
- Owner: `orchestrator`
- Size: `large`
- Status: `planned`
- Priority: `P2`
- Contract: Gormes extends the default Kanban core to Hermes multi-board and workspace semantics: board registry/list/create/switch/rename/remove, per-board database roots, scratch/worktree/dir workspace allocation, comments/events/runs/log retention, notification subscriptions, stats/watch/tail views, and garbage collection policies.
- Trust class: operator, child-agent, gateway, system
- Ready when: Hermes Kanban durable board core is complete., Dispatcher and worker-tool rows define run/log/heartbeat data that board views should expose., A Gormes-owned board namespace policy exists and rejects Hermes env aliases outside migrate commands.
- Not ready when: Board switching reads HERMES_KANBAN_BOARD or ~/.hermes board files as live config., Workspace allocation creates git worktrees in tests or deletes real operator directories., GC can delete active-task logs, workspaces, or board databases without explicit archived/retention checks.
- Degraded mode: Unsupported board names, workspace collisions, missing log files, invalid notification subscriptions, and GC denial return typed evidence without deleting active task state or crossing into Hermes home directories.
- Fixture: `internal/kanban/boards_test.go; internal/kanban/workspaces_test.go; cmd/gormes/kanban_command_test.go`
- Write scope: `internal/kanban/`, `cmd/gormes/kanban.go`, `internal/gateway/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/kanban -run 'TestKanban(Board\|Workspace\|Run\|Notification\|GC)' -count=1`, `go test ./cmd/gormes -run TestKanbanCommand -count=1`, `go run ./cmd/progress validate`
- Done signal: Gormes Kanban supports Gormes-owned board namespaces, workspaces, comments/events/runs/logs, notifications, stats/watch/tail, and safe GC without live Hermes config.
- Acceptance: Board fixtures prove list/create/switch/rename/remove behavior under temp GORMES_KANBAN_HOME roots with no Hermes state reads., Workspace fixtures prove scratch, worktree, and dir metadata resolution without creating real git worktrees in unit tests., Runs/log/event fixtures prove show/context/tail/watch/stats expose deterministic read models., Notification fixtures prove subscribe/list/unsubscribe records are scoped to Gormes gateway sources., GC fixtures prove archived-task workspaces, old events, and old logs are pruned only under explicit retention rules.
- Source refs: ../hermes-agent/hermes_cli/kanban.py@54e78cadb:boards/workspace/log/runs/gc commands, ../hermes-agent/hermes_cli/kanban_db.py@54e78cadb:board registry and workspace helpers, ../hermes-agent/tests/hermes_cli/test_kanban_db.py@54e78cadb:tenant/board/workspace tests, ../hermes-agent/tests/hermes_cli/test_kanban_cli.py@54e78cadb:context/tenant/json tests
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. Behavioral pattern extraction from session logs

- Phase: 6 / 6.K
- Owner: `orchestrator`
- Size: `large`
- Status: `planned`
- Priority: `P3`
- Contract: Mine session logs and tool execution audits for behavioral patterns: which tool sequences succeed vs fail, which reasoning patterns precede good outcomes, which response styles correlate with user satisfaction. Patterns feed into the self-evolution loop as candidate mutations.
- Trust class: operator
- Ready when: Session logs are structured and queryable, Tool execution audit log exists (Phase 3.E.2)
- Not ready when: No structured session data available, Tool audit log not yet implemented
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/hermes/pattern_extractor.go`, `internal/hermes/pattern_extractor_test.go`
- Test commands: `go test ./internal/hermes -run TestPatternExtractor -count=1`
- Done signal: Pattern extractor tests prove successful and failed patterns are correctly identified from log data
- Acceptance: Pattern extractor identifies tool sequences with >80% success rate, Identifies tool sequences with <30% success rate (anti-patterns), Extracts reasoning patterns preceding successful tool calls, Patterns stored in Goncho as structured behavioral knowledge, Pattern extraction is offline (does not run during agent turns)
- Source refs: docs/content/papers/agentic-os-design.md, Hermes Agent GEPA engine, Generative Agents reflection mechanism (Park et al. 2023), internal/goncho/extractor.go, internal/hermes/turn.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Agentic-porting-kit repo scaffold

- Phase: 8 / 8.E
- Owner: `skills`
- Size: `large`
- Status: `planned`
- Priority: `P2`
- Contract: The gormes-* skill set (gormes-planner, gormes-builder, gormes-tdd-slice, gormes-parity-auditor, gormes-references, gormes-skill-manager) is extracted into a separate public TrebuchetDynamics repo (`agentic-porting-kit` or equivalent), with a README that frames the kit as a generic Python→Go porting toolkit, a worked example using a small non-Hermes target, and a clear license. The kit must work standalone — its rows must be loadable by Codex or Claude Code in any repo, not just Gormes.
- Trust class: operator
- Ready when: All listed skills have a README of their own that does not assume the Gormes repo layout., Skills' references that hard-code Gormes paths have been parameterized or generalized.
- Not ready when: Skills still hard-code paths under docs/content/building-gormes/., The extracted kit cannot be tested without cloning Gormes.
- Degraded mode: Without extraction, the methodology is invisible to other teams; "the loop is the product" cannot be substantiated externally.
- Fixture: `(separate repo: TrebuchetDynamics/agentic-porting-kit)`
- Write scope: `(separate repo)`, `webpages/docs/development-skills/ (de-Gormes-fy paths)`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: -
- No test required: Cross-repo extraction; success is measured by the kit working standalone in a fresh checkout, not unit tests inside Gormes.
- Done signal: Repo URL recorded in success-plan.md and README.md; star count tracked monthly.
- Acceptance: Public repo TrebuchetDynamics/agentic-porting-kit exists with the listed skills., Repo README explains the kit independent of Gormes/Hermes., A worked example demonstrates the kit on a non-Hermes target (any small Python project being ported to Go)., Skills can be loaded into a fresh Codex or Claude Code session and successfully plan-and-execute one row in the example target.
- Source refs: docs/content/building-gormes/strategy/success-plan.md, webpages/docs/development-skills/gormes-planner/SKILL.md, webpages/docs/development-skills/gormes-builder/SKILL.md, webpages/docs/development-skills/gormes-tdd-slice/SKILL.md, webpages/docs/development-skills/gormes-parity-auditor/SKILL.md, webpages/docs/development-skills/gormes-references/SKILL.md, webpages/docs/development-skills/gormes-skill-manager/SKILL.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 7. Built-with-Gormes page scaffold

- Phase: 8 / 8.G
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P3`
- Contract: A page at gormes.ai/built-with (or equivalent path on the docs site) lists real production deployments of Gormes, even if there is initially only one entry (the operator's own). The page has a documented submission process (PR-based) and a template entry shape. The point is to make the slot exist so it can be filled, not to fake usage.
- Trust class: operator
- Ready when: Landing page exists., An entry template (yaml or md) is decided.
- Not ready when: Entries are fabricated., The submission process is unwritten.
- Degraded mode: Without the page, even genuine outside users have no place to land their name; reputation compounds through visibility.
- Fixture: `webpages/landing/src/pages/built-with.astro (or equivalent)`
- Write scope: `webpages/landing/src/`, `CONTRIBUTING.md`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `(cd webpages/landing && npm run test:e2e)`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Public page live with at least one truthful entry; submission process documented.
- Acceptance: /built-with (or chosen path) is reachable on the public landing site., The page renders at least one real entry (operator's own deployment, with truthful description)., A submission template + PR-based process is documented either inline on the page or in CONTRIBUTING.md.
- Source refs: docs/content/building-gormes/strategy/success-plan.md, webpages/landing/
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
