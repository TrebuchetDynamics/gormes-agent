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
## 1. Native video_analyze tool contract

- Phase: 5 / 5.D
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: Gormes exposes a `video_analyze` tool that accepts a local video path or URL and a prompt, routes the call to a vision-capable multimodal provider (Gemini today, others as the registry grows), and returns analysis text plus optional structured metadata. Routing is gated by the existing Image input mode router/provider-vision-capability check; non-vision providers return a typed unsupported_video error rather than a fake reply. Mirrors Hermes v0.13.0 PR #19301.
- Trust class: operator, child-agent
- Ready when: Image input mode router (5.D `Image input mode router + native content parts`) is complete and reusable for video parts., Provider registry exposes a vision-capability check seam reachable from tests., A fake multimodal provider exists in tests and can echo back attached media metadata.
- Not ready when: The slice ships a real Gemini transport in this row (use the existing provider abstraction; transport-specific work is its own row)., Video uploads to providers without vision capability are attempted with a silent fallback., Local file path validation collapses into the same code path as URL fetching without explicit fixture coverage.
- Degraded mode: When no vision-capable provider is configured, the tool returns unsupported_video evidence with the configured providers listed; it never uploads the video file or attempts a non-vision text fallback.
- Fixture: `internal/tools/video_analyze_test.go`
- Write scope: `internal/tools/video_analyze.go`, `internal/tools/video_analyze_test.go`, `internal/tools/registry.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/tools -run VideoAnalyze -count=1`, `go test ./internal/provider -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: video_analyze tool descriptor is registered, fixture-proven against a fake vision provider, and rejects unsupported providers and unsafe inputs with typed evidence.
- Acceptance: TestVideoAnalyze_RoutesToVisionCapableProvider routes a fake mp4 path to the fake vision provider and returns the echoed prompt+metadata., TestVideoAnalyze_NonVisionProviderUnsupportedError proves a non-vision provider returns the typed unsupported_video error with configured-providers context., TestVideoAnalyze_LocalPathSanitization proves directory-traversal and absolute-path inputs are bounded by the workspace root and rejected with workspace_root_violation evidence., TestVideoAnalyze_URLSchemeAllowlist proves only http/https URL schemes pass to a fake fetcher; file/ftp/data are rejected.
- Source refs: hermes-agent/RELEASE_v0.13.0.md, hermes-agent/tools/video_analyze.py, hermes-agent/tools/image_input.py, internal/tools/, internal/provider/
- Unblocks: Gemini video transport adapter, video_analyze gateway delivery (multipart preview)
- Why now: Unblocks Gemini video transport adapter, video_analyze gateway delivery (multipart preview).

## 2. Provider client lazy-init for TUI cold-start budget

- Phase: 5 / 5.Q
- Owner: `provider`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: Provider HTTP client construction (OpenAI, Anthropic, Bedrock helpers, Firecrawl-equivalent web client) is lazy: clients are only instantiated when a code path actually selects that provider, not at package init or config load. A checked-in benchmark exercises the gormes binary cold-start path (process exec → first interactive frame, in `gormes -z`) and asserts a documented budget; the budget rationale cites Hermes v0.12.0 PR #17046 (lazy OpenAI/Anthropic/Firecrawl) within the broader ~57% cold-start cut.
- Trust class: operator
- Ready when: Provider registry exposes per-provider constructor seams reachable from a benchmark without launching the full TUI., A `gormes -z <prompt>` non-interactive entry point exists and can be exec'd from a Go benchmark with a fake provider.
- Not ready when: The slice rewrites unrelated TUI rendering, gateway dispatch, or provider transport., Cold start is asserted only by manual stopwatch timing rather than a checked-in `go test -bench` fixture., Lazy-init introduces a global mutable client cache without a clear lifetime/reset path.
- Degraded mode: Without lazy provider construction, cold-start cost grows with installed-provider count and runs OAuth/network probes for unselected providers; the benchmark surfaces the regression but does not block startup.
- Fixture: `internal/runtime/coldstart_bench_test.go`
- Write scope: `internal/provider/lazy_client.go`, `internal/provider/lazy_client_test.go`, `internal/runtime/coldstart_bench_test.go`, `cmd/gormes/oneshot.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/provider -run LazyInit -count=1`, `go test -bench BenchmarkGormesColdStart -benchtime=5x -run ^$ ./internal/runtime`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Cold-start benchmark is checked in with a documented budget and fails when an unselected provider is constructed on the cold path.
- Acceptance: BenchmarkGormesColdStart exercises a representative cold path and fails if cold-start exceeds the documented budget on the developer baseline machine., TestProviderClientLazyInit_DoesNotConstructUnselectedProvider proves selecting Anthropic does not construct OpenAI/Bedrock/Firecrawl clients on the cold path., TestProviderClientLazyInit_ConstructedOnce proves the selected provider client is constructed once per process and reused., TestProviderClientLazyInit_ResetForTesting proves a test-only reset path exists so subsequent benchmarks/tests start from a clean state.
- Source refs: hermes-agent/RELEASE_v0.12.0.md, hermes-agent/agent/agent.py, hermes-agent/hermes_cli/main.py, internal/provider/, cmd/gormes/
- Unblocks: Config loader mtime cache for cold start, Tool definitions memoization for cold start, Dangerous-pattern precompilation for cold start
- Why now: Unblocks Config loader mtime cache for cold start, Tool definitions memoization for cold start, Dangerous-pattern precompilation for cold start.

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

## 4. Behavioral pattern extraction from session logs

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

## 5. Agentic-porting-kit repo scaffold

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

## 6. Built-with-Gormes page scaffold

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
