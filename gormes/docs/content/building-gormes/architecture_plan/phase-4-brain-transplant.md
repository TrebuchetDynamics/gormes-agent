---
title: "Phase 4 — The Brain Transplant"
weight: 50
---

# Phase 4 — The Brain Transplant (Powertrain)

**Status:** ◌ in_progress

**Deliverable:** Native Go agent orchestrator + prompt builder.

Phase 4 is when Hermes becomes optional. Each sub-phase is a separable spec.

## Phase 4 Sub-phase Outline

| Subphase | Status | Deliverable |
|---|---|---|
| 4.A — Provider Adapters | ◌ in_progress | Anthropic is landed on the shared provider seam; Bedrock is fully planned (none of the five codexu/* worker attempts on 2026-04-23 merged to main) and now tracked as three dependency-ordered TDD slices in the ledger — payload mapping, stream decoding, SigV4/credential seam — and Gemini, OpenRouter, Google Code Assist, and Codex are still planned over the same contract |
| 4.B — Context Engine + Compression | ⏳ planned | Port `agent/{context_engine,context_compressor,context_references}.py`; execute as smaller slices: interface/status contract, token-budget trigger, tool-result pruning with protected head/tail summary, then manual feedback/context references |
| 4.C — Native Prompt Builder | ⏳ planned | Port `agent/prompt_builder.py`; execute as smaller slices: context-file discovery and injection scan, model-specific role/tool guidance, toolset-aware skills prompt snapshots, and memory/session-search guidance assembly |
| 4.D — Smart Model Routing | ◌ in_progress | Model metadata and the pure routing/fallback selector remain planned; the kernel now supports turn-scoped model overrides that persist across tool iterations without mutating the default model |
| 4.E — Trajectory + Insights | ⏳ planned | Port the opt-in trajectory writer with redaction gates, then bridge native runtime metrics into the existing Phase 3 insights rollup |
| 4.F — Title Generation | ⏳ planned | Freeze title prompt/truncation behavior before wiring auto-naming into session persistence |
| 4.G — Credentials + OAuth | ⏳ planned | Port XDG-scoped token storage, credential-pool selection, and Google OAuth refresh/device-browser flows before provider adapters consume secrets |
| 4.H — Rate / Retry / Caching | ◌ in_progress | Jittered reconnect backoff (1s/2s/4s/8s/16s +/-20%) is landed and wired into the kernel; Retry-After header/body parsing and capped provider-hint plumbing on `HTTPError` are not, so the retry half is still partial. Remaining slices: Retry-After parsing + capped hint, richer structured error taxonomy, prompt-cache capability guards, and provider rate/budget telemetry |

Once 4.A–4.D are shipped Gormes can call LLMs directly. The `:8642` health check becomes optional.

Current state: the first provider-native adapter now lives in `internal/hermes`, and the shared provider seam is no longer hypothetical. `internal/hermes/client.go` freezes the common ChatRequest/Message/Event/ToolCall contracts, and Anthropic (`anthropic_client.go` + `anthropic_stream.go` + `anthropic_client_test.go`) ships direct Messages API request shaping for cache-control metadata, streamed tool-use delta accumulation, stop-reason mapping, and 429 handling. Bedrock is **not yet landed on main** — five codexu/* worker attempts on 2026-04-23 produced candidate adapters (`feat(phase4): add bedrock provider adapter` x2, `feat(provider): add bedrock client`, `feat(provider): add native Bedrock adapter`, `feat(gormes): add native bedrock provider adapter`) but none merged, and no `bedrock_*.go` file is tracked in `internal/hermes/`. The remaining 4.A closeout is: extract a reusable transcript fixture harness and lock cross-provider tool-continuation behavior over the Anthropic adapter first, then land Bedrock as three small TDD slices (payload mapping, stream event decoding, minimal SigV4/credential seam) before Gemini/OpenRouter/Codex. Phase 4.H is partial: provider HTTP failures classify retryable vs fatal auth/context/rate-limit signals via `internal/hermes/errors.go` and the kernel reconnect budget applies 1s/2s/4s/8s/16s +/-20% jitter; Retry-After header/body parsing is **not** yet landed — HTTPError currently exposes only `{Status, Body}`, so provider hints cannot cap kernel retry delays. Smart Model Routing has also started: `internal/kernel` now accepts a turn-scoped model override, carries it across tool iterations, and surfaces the active model in render frames and telemetry without mutating the kernel default. The remaining 4.D work is the metadata registry and pure selector.

## Build Priority Context

Phase 4 is **optimization**, not **differentiation**. The Python bridge works. Replace it only after the OS-AI spine and the wider gateway surface prove the architecture is correct. The current dependency chain is:

> 2.E0 deterministic subagent runtime → 2.G static skills + reviewed candidate flow → runner-enforced delegation policy + wider gateway surface → native agent loop

**The rule:** stabilize the runtime substrate first, then add explicit skills and the reviewed skill flow, then harden delegation policy, then widen adapters, and only then replace the Python bridge.

## TDD Handoff Notes

Phase 4 should not start with "port `run_agent.py`." The next execution agents should close the partially landed contract work with small, test-first slices that do not require live model credentials:

1. **4.A shared transcript fixture harness closeout** — harvest Anthropic request/stream transcripts into reusable fixtures that assert usage, finish reasons, and shared event decoding, so Bedrock/Gemini/OpenRouter/Codex can be added against one replayable contract.
2. **4.A cross-provider tool-continuation fixtures** — pin assistant tool-call messages, streamed tool-call deltas, and tool-result continuation payloads against the Anthropic adapter before adding a second provider to the harness.
3. **4.H Retry-After parsing + capped provider-hint plumbing** — add Retry-After header/body parsing to `internal/hermes/errors.go`, surface a capped hint on HTTPError, and make the kernel open-stream retry prefer the provider hint over the fixed schedule; do **not** widen the jittered backoff path that is already green.
4. **4.A Bedrock payload mapping (no AWS SDK)** — port only the Converse request shaping and canonical Message→Bedrock tool-aware mapping with pure fixtures, proving the 5 prior codexu/* attempts can be replaced by a tightly scoped slice.
5. **4.A Bedrock stream event decoding (SSE fixtures)** — decode reasoning/text/tool-use deltas into the shared `hermes.Event` model from recorded SSE fixtures, without binding the real AWS SDK yet.
6. **4.A Bedrock minimal SigV4 + credential seam** — land credential discovery and signing behind a small dep-isolated helper so regional/error handling can follow without dragging the full AWS SDK into kernel paths.
7. **4.H structured error-taxonomy closeout** — extend the current retryable/fatal classification into explicit rate-limit/auth/context/non-retryable envelopes without regressing the shipped retry path.
8. **4.D model metadata registry** — context limits, pricing, capability flags, and provider-family facts in a read-only registry.
9. **4.D routing selector** — a pure fallback/override selector over the metadata registry before any automatic model choice is allowed.
10. **4.B context-engine status contract** — `get_status`, context-window updates, token-budget trigger, and compression cooldown behavior once the provider seam is frozen.
11. **4.C prompt-builder context-file discovery** — SOUL/HERMES/AGENTS/CLAUDE ordering, truncation, frontmatter stripping, and prompt-injection scans after the context-engine contract is pinned.

Only after the Bedrock slices (1–6) are green should more provider-specific adapters land. This keeps retry, role mapping, and tool-call continuation bugs visible instead of hiding them behind a large native-agent-loop rewrite.
