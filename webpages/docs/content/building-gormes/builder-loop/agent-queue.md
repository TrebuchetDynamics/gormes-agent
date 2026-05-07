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
## 1. Sharp v1.0 differentiator decision

- Phase: 8 / 8.D
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P0`
- Contract: A short decision document records the sharp v1.0 differentiator: a single paragraph stating what Gormes 1.0 will be (recommended: "runs the 30 most-used Hermes skills unchanged, in a single 30 MB Go binary, on Termux + Windows-without-Python + locked-down corp Linux"), the curated 30-skill list, the explicit exclusion list of what 1.0 will NOT do, and the date the decision was ratified. The decision unblocks every downstream messaging row.
- Trust class: operator
- Ready when: Operator has read success-plan.md and considered alternative differentiators., An evidence-backed shortlist of the 30 most-used Hermes skills exists (telemetry, repo signal, or operator judgement — but recorded).
- Not ready when: The doc is empty placeholder text., The differentiator is still phrased as "Hermes in Go" or "feature parity"., The exclusion list is missing — readers cannot self-select.
- Degraded mode: Without a written, dated differentiator, the README and landing page cannot be rewritten coherently and the parity backlog has no Pareto filter.
- Fixture: `docs/content/building-gormes/strategy/v1-differentiator.md`
- Write scope: `docs/content/building-gormes/strategy/v1-differentiator.md`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: -
- No test required: Strategic decision row; the artifact is a written-down decision. Validation is the existence of the doc with the required sections.
- Done signal: v1-differentiator.md exists, fits the schema above, and is referenced from README.md and landing-page hero.
- Acceptance: docs/content/building-gormes/strategy/v1-differentiator.md exists with: differentiator paragraph, 30-skill list, exclusion list, ratified-on date., The differentiator paragraph fits in <50 words., The exclusion list explicitly mentions areas Gormes 1.0 will not chase (TUI parity beyond core, dashboard, web app, full i18n, etc.).
- Source refs: docs/content/building-gormes/strategy/success-plan.md, hermes-agent/skills/, internal/skills/
- Unblocks: README rewrite to methodology-first positioning, gormes.ai landing page positioning audit, Single-binary cross-platform release pipeline, Benchmarks page at gormes.ai/benchmarks
- Why now: P0 handoff; needs contract proof before closeout.

## 2. Docker execution backend (container lifecycle + mount policy)

- Phase: 5 / 5.B
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: Gormes executes agent tools inside Docker containers through the existing DockerContainerKey helper and Environment interface. The backend handles: image selection/resolution from config or Hermes-compatible defaults, mount policy (allowlisted host paths mapped read-only, workspace mapped read-write, blocked dangerous mounts), env passthrough from config with allowlist filtering, container lifecycle with timeout cleanup, and stdout/stderr capture for tool output.
- Trust class: operator, system
- Ready when: Docker backend top-level container reuse semantics (DockerContainerKey) is complete., Environment interface + file sync contract is complete., Tests use fake Docker client or Docker socket stub; no live Docker daemon required in CI.
- Not ready when: The slice implements Modal, Daytona, or Singularity backends — those remain separate rows., The slice changes the DockerContainerKey helper or Environment interface shape., Tests require a running Docker daemon on the CI agent.
- Degraded mode: Missing Docker socket, image pull failure, or container timeout produce structured errors with mount_policy_blocked, image_pull_failed, or container_timeout reasons. Gateway status reports Docker backend availability without exposing raw Docker socket paths.
- Fixture: `internal/tools/docker_exec_test.go`
- Write scope: `internal/tools/docker_exec.go`, `internal/tools/docker_exec_test.go`, `internal/tools/docker_mount_policy.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run TestDockerExec -count=1`, `go test ./internal/tools -count=1`, `go run ./cmd/progress validate`
- Done signal: Docker execution fixtures prove image resolution, mount allowlist/blocking, env passthrough, timeout cleanup, and stdout/stderr capture without a live Docker daemon.
- Acceptance: TestDockerExec_ImageResolution uses config- or default-specified image and verifies container start with correct image tag., TestDockerExec_MountPolicyAllowlist allows configured host paths (read-only) and workspace path (read-write) while blocking /etc, /proc, /sys, and Docker socket mounts., TestDockerExec_EnvPassthrough passes only allowlisted env vars from config to the container., TestDockerExec_TimeoutCleanup verifies container is stopped and removed after timeout or tool completion., TestDockerExec_StdoutStderrCapture captures container stdout/stderr as tool output and returns structured errors on non-zero exit.
- Source refs: ../hermes-agent/tools/docker.py, ../hermes-agent/tools/sandboxing.py, internal/tools/docker_container_key.go (DockerContainerKey helper), internal/tools/env_interface.go (Environment interface), internal/tools/file_sync.go (file sync contract), internal/tools/timeout.go (timeout cleanup)
- Unblocks: Modal, Daytona, Singularity
- Why now: Unblocks Modal, Daytona, Singularity.

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

## 4. Sandbox isolation depth selection

- Phase: 5 / 5.U
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P3`
- Contract: Operator can select sandbox isolation depth: process-level (fast, weaker isolation), container-level (Docker/gVisor, balanced), or VM-level (Firecracker, strongest isolation). Default is process-level with transactional rollback.
- Trust class: operator
- Ready when: Transactional executor exists (5.U row 2)
- Not ready when: No sandbox backend available
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/tools/isolation_depth.go`, `internal/tools/isolation_depth_test.go`
- Test commands: `go test ./internal/tools -run TestIsolationDepth -count=1`
- Done signal: Isolation depth tests prove all three levels selectable and process-level works without Docker
- Acceptance: Process-level isolation is the default and requires zero setup, Docker/gVisor isolation selectable via config, Firecracker VM isolation selectable via config, Isolation depth is per-session configurable, Deeper isolation correctly fails if backend not available
- Source refs: docs/content/papers/safety-and-deployment.md, OpenSandbox (github.com/alibaba/OpenSandbox), internal/tools/sandbox.go
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
