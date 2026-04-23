---
title: "Phase 4 — The Brain Transplant"
weight: 50
---

# Phase 4 — The Brain Transplant (Powertrain)

**Status:** ⏳ planned

**Deliverable:** Native Go agent orchestrator + prompt builder.

Phase 4 is when Hermes becomes optional. Each sub-phase is a separable spec.

## Phase 4 Sub-phase Outline

| Subphase | Status | Deliverable |
|---|---|---|
| 4.A — Provider Adapters | ⏳ planned | Start with a provider-neutral stream fixture harness and tool-call continuation contract, then port Anthropic, Bedrock, Gemini, OpenRouter, Google Code Assist, and Codex as adapters over that shared seam |
| 4.B — Context Engine + Compression | ⏳ planned | Port `agent/{context_engine,context_compressor,context_references}.py`; execute as smaller slices: interface/status contract, token-budget trigger, tool-result pruning with protected head/tail summary, then manual feedback/context references |
| 4.C — Native Prompt Builder | ⏳ planned | Port `agent/prompt_builder.py`; execute as smaller slices: context-file discovery and injection scan, model-specific role/tool guidance, toolset-aware skills prompt snapshots, and memory/session-search guidance assembly |
| 4.D — Smart Model Routing | ⏳ planned | Port model metadata first, then a pure routing/fallback selector, then per-turn model selection once provider capabilities are stable |
| 4.E — Trajectory + Insights | ⏳ planned | Port the opt-in trajectory writer with redaction gates, then bridge native runtime metrics into the existing Phase 3 insights rollup |
| 4.F — Title Generation | ⏳ planned | Freeze title prompt/truncation behavior before wiring auto-naming into session persistence |
| 4.G — Credentials + OAuth | ⏳ planned | Port XDG-scoped token storage, credential-pool selection, and Google OAuth refresh/device-browser flows before provider adapters consume secrets |
| 4.H — Rate / Retry / Caching | ⏳ planned | Port classified provider errors, Retry-After/jittered backoff, prompt-cache capability guards, and provider rate/budget telemetry as independent slices |

Once 4.A–4.D are shipped Gormes can call LLMs directly. The `:8642` health check becomes optional.

## Build Priority Context

Phase 4 is **optimization**, not **differentiation**. The Python bridge works. Replace it only after the OS-AI spine and the wider gateway surface prove the architecture is correct. The current dependency chain is:

> 2.E0 deterministic subagent runtime → 2.G static skills + reviewed candidate flow → runner-enforced delegation policy + wider gateway surface → native agent loop

**The rule:** stabilize the runtime substrate first, then add explicit skills and the reviewed skill flow, then harden delegation policy, then widen adapters, and only then replace the Python bridge.

## TDD Handoff Notes

Phase 4 should not start with "port `run_agent.py`." The next execution agents should first freeze provider-independent contracts that can be tested without live model credentials:

1. **4.A provider interface + stream fixture harness** — request shape, stream events, tool-call deltas, usage, finish reasons, and continuation payloads.
2. **4.B context-engine status contract** — `get_status`, context-window updates, token-budget trigger, and compression cooldown behavior.
3. **4.C prompt-builder context-file discovery** — SOUL/HERMES/AGENTS/CLAUDE ordering, truncation, frontmatter stripping, and prompt-injection scans.
4. **4.H provider error taxonomy** — retryable/rate-limit/auth/context-overflow/non-retryable classification with table-driven fixtures.
5. **4.D model metadata registry** — context limits, pricing, capability flags, and fallback selector behavior before any per-turn model switch mutates live conversations.

Only after those are green should provider-specific adapters land. This keeps prompt-cache, compression, role mapping, and tool-call continuation bugs visible instead of hiding them behind a large native-agent-loop rewrite.
