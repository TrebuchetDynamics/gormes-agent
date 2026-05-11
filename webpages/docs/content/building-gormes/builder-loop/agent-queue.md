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
## 1. TD engineering blog scaffolded and live

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
- No test required: Documentation/research/planning row — automated tests not applicable
- Done signal: Public blog URL + feed URL recorded in success-plan.md and README.md.
- Acceptance: Blog is reachable at a public URL with at least one real (non-placeholder) post., An Atom or RSS feed exists at a stable, discoverable URL., Publishing a new post is a markdown-commit-and-merge operation; no console click-through required., An /about page exists that names TrebuchetDynamics and points at gormes-agent + agentic-porting-kit.
- Source refs: docs/content/building-gormes/strategy/success-plan.md, webpages/landing/
- Unblocks: Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline
- Why now: Unblocks Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline.

## 2. Tirith external security finding ingestion

- Phase: 5 / 5.J
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P3`
- Contract: Port the Hermes Tirith external security finding ingestion path: load findings from a JSON file or env-sourced source, classify by severity, and expose a security guard decision that gateway/cron/CLI callers can query before executing dangerous commands.
- Trust class: operator, system
- Ready when: Hermes Tirith source files are mapped and the load/classify/allow/deny contract is understood.
- Not ready when: -
- Degraded mode: Missing or corrupt Tirith source returns tirith_unavailable evidence and falls back to the existing config-level allowlist rather than denying all commands.
- Fixture: `internal/security/tirith_test.go`
- Write scope: `internal/security/tirith.go`, `internal/security/tirith_test.go`
- Test commands: `go test ./internal/security -run TestTirith -count=1`
- Done signal: Tirith finding ingestion passes all acceptance tests.
- Acceptance: TestTirithLoadsFindings proves findings are parsed and classified by severity., TestTirithEmptySourceReturnsSafeEvidence proves empty/missing source returns safe (allow) with typed evidence., TestTirithCorruptSourceDegrades proves corrupt findings return deny with tirith_corrupt_evidence.
- Source refs: ../hermes-agent/agent/tirith.py finding ingestion, ../hermes-agent/tests/test_tirith.py
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Unified security guard decision composer

- Phase: 5 / 5.J
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P3`
- Contract: Compose Tirith findings, path-based allowlists, URL safety rules, and website policies into one security guard decision that gateway/cron/CLI can call before executing any tool. The composer resolves conflicts deterministically (deny wins over allow, policy overrides Tirith) and always returns typed evidence explaining the decision.
- Trust class: operator, system
- Ready when: Tirith external security finding ingestion row is complete., URL safety rules and website policy surfaces are mapped.
- Not ready when: -
- Degraded mode: If any policy source is unavailable, the composer continues with the remaining sources and reports guard_partial_evidence.
- Fixture: `internal/security/guard_test.go`
- Write scope: `internal/security/guard.go`, `internal/security/guard_test.go`
- Test commands: `go test ./internal/security -run TestUnifiedGuard -count=1`
- Done signal: Unified guard composer passes all acceptance tests and is callable from gateway/cron/CLI.
- Acceptance: TestGuardDenyOverridesAllow proves a Tirith deny blocks even when the path allowlist permits., TestGuardPolicyOverridesTirith proves a URL policy explicitly permits even when Tirith warns., TestGuardEmptyComposerAllows proves with no policies loaded, the guard allows and reports guard_no_policies evidence.
- Source refs: ../hermes-agent/agent/tirith.py decision composer, internal/tools/url_safety.go, internal/tools/website_policy.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. Kanban multi-board isolation

- Phase: 5 / 5.M
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Kanban dispatcher enforces board-scoped isolation: workers spawned for board A cannot see or mutate board B's tasks. The SQLite store uses per-board database files or namespaced tables, and the dispatcher validates the board name before spawning.
- Trust class: operator, system
- Ready when: Existing Kanban single-board surface is validated and the multi-board isolation contract is understood from Hermes upstream.
- Not ready when: -
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/kanban/multi_board.go`, `internal/kanban/multi_board_test.go`
- Test commands: `go test ./internal/kanban -run TestKanbanMultiBoard -count=1`
- Done signal: Multi-board isolation tests pass and the dispatcher prevents cross-board task access.
- Acceptance: TestKanbanMultiBoardIsolation proves worker A on board_alpha cannot query tasks from board_beta., TestKanbanMultiBoardDispatcherHonoursBoardBoundary proves the dispatcher rejects cross-board task assignments.
- Source refs: ../hermes-agent/hermes_cli/kanban.py multi-board isolation
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. Kanban workspace context injection

- Phase: 5 / 5.M
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Kanban worker spawning injects the board's workspace directory as the worker's working directory and loads the workspace's AGENTS.md/CLAUDE.md context, mirroring Hermes workspace-path isolation.
- Trust class: operator, system
- Ready when: Existing Kanban worker spawning path is understood and workspace/AGENTS.md loading is mapped.
- Not ready when: -
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/kanban/workspace_context.go`, `internal/kanban/workspace_context_test.go`
- Test commands: `go test ./internal/kanban -run TestKanbanWorkspace -count=1`
- Done signal: Workspace context injection tests pass.
- Acceptance: TestKanbanWorkspaceContextInjection proves a spawned worker's working directory matches the board's configured workspace., TestKanbanWorkspaceAGENTSLoad proves the worker loads AGENTS.md from the board workspace.
- Source refs: ../hermes-agent/hermes_cli/kanban.py workspace+agent dir injection
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 6. Kanban run history persistence

- Phase: 5 / 5.M
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Kanban run history records spawn attempts, successes, failures, and completion evidence per task so operators and the dispatcher can inspect past runs and detect spin-loop failures.
- Trust class: operator, system
- Ready when: Existing Kanban run lifecycle is validated and Hermes run history schema is mapped.
- Not ready when: -
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/kanban/run_history.go`, `internal/kanban/run_history_test.go`
- Test commands: `go test ./internal/kanban -run TestKanbanRunHistory -count=1`
- Done signal: Run history persistence passes all acceptance tests.
- Acceptance: TestKanbanRunHistoryRecordsSpawn proves a spawn creates a run record with started_at and status=running., TestKanbanRunHistoryRecordsCompletion proves a completed run updates the record with finished_at and status=done., TestKanbanRunHistoryAutoBlockAfterConsecutiveFailures proves 5+ consecutive failures auto-block the task.
- Source refs: ../hermes-agent/hermes_cli/kanban.py run_history + auto-block
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 7. Kanban notification delivery parity

- Phase: 5 / 5.M
- Owner: `orchestrator`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Kanban worker completion triggers notification delivery to the board owner's configured channel (Telegram/Discord/Slack) with task summary and run evidence.
- Trust class: operator, system
- Ready when: Existing Kanban completion path and gateway notification surface are validated.
- Not ready when: -
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/kanban/notifications.go`, `internal/kanban/notifications_test.go`
- Test commands: `go test ./internal/kanban -run TestKanbanNotification -count=1`
- Done signal: Notification delivery passes tests with platform-specific message formatting.
- Acceptance: TestKanbanNotificationOnComplete proves worker completion sends a notification message to the configured platform., TestKanbanNotificationOnFailure proves worker failure sends an error notification., TestKanbanNotificationThrottle proves rapid consecutive completions are throttled to one notification per board per minute.
- Source refs: ../hermes-agent/hermes_cli/kanban.py notify-* methods
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 8. Installer script serving and MIME validation

- Phase: 5 / 5.P
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P3`
- Contract: Wire install.sh, install.ps1, and install.cmd into the www.gormes.ai Go server with correct Content-Type headers (text/x-shellscript, text/plain, text/plain), cache-control, and static export. Tests verifies each script is embedded, served with the correct MIME, and static-exported.
- Trust class: operator, system
- Ready when: www.gormes.ai site server code is understood and the installer parity plan is reviewed.
- Not ready when: -
- Degraded mode: -
- Fixture: `www.gormes.ai/internal/site/assets_test.go`
- Write scope: `www.gormes.ai/internal/site/assets.go`, `www.gormes.ai/internal/site/assets_test.go`
- Test commands: `go test ./www.gormes.ai/... -run TestInstallerAssets -count=1`
- Done signal: All three installer scripts serve with correct MIME types and static export verification passes.
- Acceptance: TestInstallShEmbeddedAndServed proves /install.sh serves with text/x-shellscript MIME and correct content., TestInstallPs1EmbeddedAndServed proves /install.ps1 serves with text/plain MIME., TestInstallCmdEmbeddedAndServed proves /install.cmd serves with text/plain MIME., TestInstallerStaticExport proves all three scripts appear in static_export output.
- Source refs: www.gormes.ai/internal/site/assets.go, docs/superpowers/plans/2026-04-23-gormes-installer-parity.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 9. DingTalk real SDK binding

- Phase: 7 / 7.E
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Bind the Gormes DingTalk channel to a real DingTalk Stream Mode SDK (replacing the current stub/fake). Implement credential loading (AppKey/AppSecret from config.toml), receive loop via the SDK's callback, send lifecycle, and reconnection with the existing retry seam.
- Trust class: operator, system
- Ready when: A Go-compatible DingTalk Stream Mode SDK exists (or a REST fallback is decided) and the existing contract/adapter tests pass.
- Not ready when: -
- Degraded mode: If the SDK is unavailable or credentials are missing, the channel reports dingtalk_sdk_unavailable evidence and the gateway skips only this channel.
- Fixture: `internal/channels/dingtalk/client_test.go`
- Write scope: `internal/channels/dingtalk/bot.go`, `internal/channels/dingtalk/client.go`, `internal/channels/dingtalk/client_test.go`
- Test commands: `go test ./internal/channels/dingtalk -run TestDingTalk -count=1`
- Done signal: Real DingTalk SDK binding passes all acceptance tests with the existing integration test suite.
- Acceptance: TestDingTalkCredentialResolution proves AppKey/AppSecret are resolved from config and env., TestDingTalkReceiveLoop proves incoming messages from the SDK are forwarded as gateway.InboundEvent., TestDingTalkSendLifecycle proves outbound messages reach the SDK send method., TestDingTalkReconnect proves connection loss triggers the retry seam without data loss.
- Source refs: ../hermes-agent/gateway/platforms/dingtalk.py Stream Mode adapter, internal/channels/dingtalk/
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 10. Agentic-porting-kit repo scaffold

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
- No test required: Documentation/research/planning row — automated tests not applicable
- Done signal: Repo URL recorded in success-plan.md and README.md; star count tracked monthly.
- Acceptance: Public repo TrebuchetDynamics/agentic-porting-kit exists with the listed skills., Repo README explains the kit independent of Gormes/Hermes., A worked example demonstrates the kit on a non-Hermes target (any small Python project being ported to Go)., Skills can be loaded into a fresh Codex or Claude Code session and successfully plan-and-execute one row in the example target.
- Source refs: docs/content/building-gormes/strategy/success-plan.md, webpages/docs/development-skills/gormes-planner/SKILL.md, webpages/docs/development-skills/gormes-builder/SKILL.md, webpages/docs/development-skills/gormes-tdd-slice/SKILL.md, webpages/docs/development-skills/gormes-parity-auditor/SKILL.md, webpages/docs/development-skills/gormes-references/SKILL.md, webpages/docs/development-skills/gormes-skill-manager/SKILL.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
