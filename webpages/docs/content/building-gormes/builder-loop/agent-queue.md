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

## 2. DingTalk real SDK binding

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

## 3. Agentic-porting-kit repo scaffold

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

## 4. Sandbox provider interface and virtual path security

- Phase: 9 / 9.B
- Owner: `orchestrator`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Gormes gains a sandbox provider abstraction layer over the existing tool execution environment (internal/tools/, internal/tools/environment_*.go, internal/tools/filesystem_scope.go), inspired by DeerFlow's SandboxProvider/Sandbox interface pair with virtual path security. The SandboxProvider interface defines acquire/get/release/shutdown lifecycle. A LocalSandboxProvider implementation provides filesystem-level isolation with per-thread virtual path mapping. The virtual path system uses a /mnt/user-data/{workspace,uploads,outputs} prefix that resolves to host paths with path-traversal rejection, path-family enforcement (read-only vs read-write), and output masking (host paths never leak into agent return values). Existing internal/tools/IsolationLevel (process/container/vm) and FilesystemScope continue to work — the sandbox provider wraps them rather than replacing them.
- Trust class: operator, system
- Ready when: internal/tools/ has existing IsolationLevel, FilesystemScope, and EnvironmentContract — the sandbox provider wraps these rather than replacing them., A VirtualPathResolver or equivalent exists for /mnt/user-data/ prefix resolution., LocalSandboxProvider implements the SandboxProvider interface at minimum — Docker/VM providers are follow-ups., Path traversal detection and output masking have test coverage.
- Not ready when: The sandbox provider duplicates or replaces internal/tools/ environment contracts without a migration plan., The row attempts to port the full Docker AioSandboxProvider in one slice — local filesystem provider first., The virtual path system hard-codes /mnt/ paths without configurable base directory support., Output masking regex patterns cause catastrophic backtracking (RE2-safe patterns required).
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/sandbox/provider.go`, `internal/sandbox/provider_test.go`, `internal/sandbox/sandbox.go`, `internal/sandbox/sandbox_test.go`, `internal/sandbox/local/provider.go`, `internal/sandbox/local/provider_test.go`, `internal/sandbox/paths.go`, `internal/sandbox/paths_test.go`, `internal/tools/`, `internal/config/`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/sandbox/... -count=1`, `go test ./internal/tools -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: SandboxProvider interface is defined; LocalSandboxProvider is implemented with virtual path resolution, traversal protection, and output masking; existing tools tests remain green.
- Acceptance: SandboxProvider interface exists with Acquire/Get/Release/Shutdown lifecycle., LocalSandboxProvider creates per-thread workspace/uploads/outputs directories and maps virtual paths., Path traversal (..) in virtual paths is rejected with a PermissionError-equivalent., Output masking replaces host paths with virtual paths in tool return values., Read-only path families (e.g. skills directory) cannot be written through the sandbox., Existing go test ./internal/tools -count=1 stays green after sandbox provider addition.
- Source refs: ./deer-flow/backend/packages/harness/deerflow/sandbox/sandbox_provider.py, ./deer-flow/backend/packages/harness/deerflow/sandbox/sandbox.py, ./deer-flow/backend/packages/harness/deerflow/sandbox/tools.py, ./deer-flow/backend/packages/harness/deerflow/sandbox/local/, docs/content/building-gormes/development-skills/deerflow-pattern-theft.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
