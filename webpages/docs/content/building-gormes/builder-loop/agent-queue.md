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
- No test required: Documentation/infrastructure row; success is the URL being live and the feed being reachable, validated by the acceptance checklist.
- Done signal: Public blog URL + feed URL recorded in success-plan.md and README.md.
- Acceptance: Blog is reachable at a public URL with at least one real (non-placeholder) post., An Atom or RSS feed exists at a stable, discoverable URL., Publishing a new post is a markdown-commit-and-merge operation; no console click-through required., An /about page exists that names TrebuchetDynamics and points at gormes-agent + agentic-porting-kit.
- Source refs: docs/content/building-gormes/strategy/success-plan.md, webpages/landing/
- Unblocks: Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline
- Why now: Unblocks Engineering writeup #1: autonomous Hermes-porting loop, Monthly digest pipeline.

## 2. Pure-Go STT exploration

- Phase: 5 / 5.E
- Owner: `provider`
- Size: `large`
- Status: `planned`
- Priority: `P3`
- Contract: Track the architectural question: how should Gormes ship local STT given the Go ecosystem currently has no production-quality pure-Go (CGO_ENABLED=0) STT library? Document the four mutually-exclusive choices (status quo / cgo build-tag / WASI productionization / wait+monitor) with their tradeoffs so future planners do not re-research the same dead ends. This row is exploratory and decision-tracking — not builder-selectable until a stakeholder commits to one of the four paths.
- Trust class: operator, system
- Ready when: A credible pure-Go STT library appears in the Go ecosystem (production-quality, actively maintained, ≥3 stars/non-trivial users), OR, A stakeholder explicitly accepts the cgo hit and approves a build-tag-gated STT path, OR, A stakeholder commits engineering time to productionize the WASI+wazero+whisper.cpp path despite the ~5x perf gap.
- Not ready when: Assigned as one combined builder slice (this is an exploration that resolves into one of four sub-rows, not an implementation slice itself)., A planner attempts to mark complete without naming which of the four architectural choices was selected and why.
- Degraded mode: Without resolution, Gormes' local STT story stays at: shell out to local whisper/whisper-cli (separate install) OR cloud HTTP fallback (Groq free tier wired 2026-05-09, OpenAI Whisper, etc.). The 'free, in-binary, no install, local STT' intersection remains empty.
- Fixture: `(no fixture — research/decision row)`
- Write scope: `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: -
- No test required: Research and architectural-decision row; no Go code is written by this row directly. Each of the four downstream paths spawns its own builder/test rows.
- Done signal: Row is closed with a note recording (a) which architectural path was selected, (b) the date, (c) the spawned implementation row IDs (if any), and (d) the rationale for choosing that path over the other three.
- Acceptance: When a stakeholder selects path #1 (status quo): this row is closed with a note pointing at the existing local-CLI + Groq HTTP fallback rows; no new code rows are spawned., When a stakeholder selects path #2 (cgo build-tag): this row spawns a builder-ready row 'Add stt_cgo build tag for sherpa-onnx integration' with concrete write_scope, tests, and acceptance., When a stakeholder selects path #3 (WASI): this row spawns a planner pass to scope the wazero+whisper.cpp.wasm productionization (model file distribution, perf budget, thread workaround tracking), then a builder row., When a stakeholder selects path #4 (wait+monitor): this row stays planned with a quarterly review cadence noted; no code rows are spawned.
- Source refs: Makefile (CGO_ENABLED=0 single-static-binary stance), internal/channels/telegram/transcriber.go (current local-CLI shim), internal/tools/transcription_providers.go (current HTTP provider clients), cmd/gormes/telegram_transcriber.go (resolver wiring local + Groq + OpenAI), github.com/ggerganov/whisper.cpp/bindings/go (cgo, MIT, active 2026-05-02), github.com/kardianos/whisper.cpp (cgo fork, MIT), github.com/mutablelogic/go-whisper (cgo HTTP server wrapper, Apache-2.0, active 2026-01-27), github.com/xPrimeTime/go-whisper-ct2 (cgo + CTranslate2 + libsndfile + libsamplerate + openblas from source, MIT, 0 stars 2026-01-10), github.com/k2-fsa/sherpa-onnx-go-linux (cgo + onnxruntime native lib, Apache-2.0, v1.13.1 2026-05-09 — most production-grade cgo option), github.com/agnivade/whisper-wasi (pure Go via wazero+WASI, MIT, 5 commits/2 stars — prototype, ~5x slower than native, no thread support in wazero), https://agniva.me/wasi/2023/12/09/whisper-wasi.html (author's writeup of the WASI approach + perf numbers), ../hermes-agent/tools/transcription_tools.py (Hermes' Python+faster-whisper local default — the path Gormes deliberately rejects)
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Pure-Go TTS path scoping

- Phase: 5 / 5.E
- Owner: `provider`
- Size: `large`
- Status: `planned`
- Priority: `P3`
- Contract: Track the parallel architectural question for outbound TTS: how should Gormes ship local text-to-speech given pure-Go TTS is even less mature than pure-Go STT? Document the three candidate paths (external TTS API / Piper sidecar process / experimental Piper or Kokoro WASM) with their tradeoffs so future planners and stakeholders have a clear decision surface.
- Trust class: operator, system
- Ready when: Row D 'Wire Pure-Go Whisper into Telegram resolver' is shipped (validates the wazero+ggml-style WASI pipeline pattern works for STT before betting it works for TTS), AND, Stakeholder explicitly requests local TTS work (today the cloud TTS clients in internal/tools satisfy operator demand).
- Not ready when: Assigned as one combined builder slice (this is an exploration that resolves into one of three sub-rows)., A planner attempts to mark complete without naming which architectural choice was selected.
- Degraded mode: Without resolution, Gormes' local TTS story stays at: existing TTS providers in internal/tools (edge-tts default, OpenAI/Google/etc. cloud) — same as today.
- Fixture: `(no fixture — research/decision row)`
- Write scope: `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: -
- No test required: Research and architectural-decision row; no Go code is written by this row directly. Each of the three downstream paths spawns its own builder/test rows.
- Done signal: Row is closed with a note recording (a) which architectural path was selected, (b) the date, (c) the spawned implementation row IDs (if any), and (d) the rationale.
- Acceptance: When a stakeholder selects path #1 (external TTS API): row closes with a note pointing at existing cloud TTS rows; no new code rows spawn., When a stakeholder selects path #2 (Piper sidecar): row spawns a builder-ready row 'Add Piper sidecar process orchestration' with concrete write_scope, lifecycle ownership, and tests., When a stakeholder selects path #3 (Piper/Kokoro WASM): row spawns a planner pass to scope the wazero+piper.wasm productionization, then a builder row mirroring the STT path Rows A-D shape.
- Source refs: internal/tools/tts_providers.go (current cloud TTS clients), https://github.com/rhasspy/piper (Piper TTS upstream — neural, ONNX-based), https://github.com/hexgrad/kokoro (Kokoro TTS — newer, also ONNX), Stakeholder plan Phase 5 + milestones 7-8 (recorded on Pure-Go STT exploration row note), docs/content/building-gormes/architecture_plan/progress.json (Pure-Go STT exploration row 5.E[10])
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. Agentic-porting-kit repo scaffold

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

<!-- PROGRESS:END -->
