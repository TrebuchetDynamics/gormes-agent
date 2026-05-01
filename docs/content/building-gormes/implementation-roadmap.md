---
title: "Implementation Roadmap"
description: "The single planning entry point for gormes-agent. Current state, decision trees, execution horizons, and document map."
weight: 12
---

# Implementation Roadmap

> **Purpose:** This is the single planning entry point for building gormes-agent. It tells you where we are, what is blocked, what comes next, and which document to read for the details. If you are a planner, builder, or reviewer, start here before touching `progress.json` or writing code.
> 
> **Relationship to other docs:**
> - `architecture_plan/progress.json` = the canonical execution queue (machine-readable)
> - [Completion Plan](../architecture_plan/completion-plan/) = the finish line definition
> - [Lane Roadmap](../architecture_plan/lane-roadmap/) = lane ownership and exit gates
> - [Must-Have Features](../must-have-features/) = feature catalogue from 12+ upstream projects
> - **This document** = the human-readable plan that ties them all together

---

## Current State at a Glance

**As of April 30, 2026**

| Phase | Status | Shipped | In Progress | Planned | Open Rows |
|-------|--------|---------|-------------|---------|-----------|
| Phase 1 — Dashboard | ✅ Complete | 4/4 | 0 | 0 | 0 |
| Phase 2 — Gateway | 🔨 Mostly Done | 20/21 | 1 | 0 | 3 |
| Phase 3 — Memory | ✅ Complete | 15/15 | 0 | 0 | 0 |
| Phase 4 — Brain Transplant | 🔨 In Progress | 3/9 | 6 | 0 | 15 |
| Phase 5 — Final Purge | 🔨 Large Backlog | 3/18 | 8 | 7 | 38 |
| Phase 6 — Learning Loop | 🔨 Not Started | 0/6 | 0 | 6 | 8 |
| Phase 7 — Paused Channels | 🔨 Backlog | 2/5 | 3 | 0 | 11 |

**Overall:** 47/78 subphases shipped · 26 in progress · 5 planned · 75 open rows

**First closure target:** Python-free normal agent turn with local Goncho memory and tested tool-call continuation. This is a dogfood gate, not a reduced finish line.

---

## Product Hardening Borrow List

These are Gormes-owned follow-up slices from the current PicoClaw comparison. They are not Hermes parity blockers, but they matter for distribution quality and constrained-machine adoption.

| Slice | Target |
|-------|--------|
| Pre-compiled binaries | Tag-driven GitHub Release workflow emits static Linux/macOS/Windows amd64+arm64 archives with SHA-256 checksums; signing and package-manager manifests remain follow-up release-hardening work. |
| Onboarding wizard | Promote `gormes onboard` from the current `setup` alias into a full interactive first-run flow covering model/provider, auth, gateway, browser/CDP, skills, and dashboard launch. |
| Hardware matrix | Maintain `using-gormes/hardware` as the tested-device matrix for x86_64, ARM64, Raspberry Pi-class boards, low-memory Linux hosts, and Android/Termux-style environments, with binary size and steady-state RSS recorded per release. |
| Lite build profiles | Keep the default parity build feature-complete, and keep `-tags gormes_lite` / `-tags slim` green as documented constrained-target builds that can exclude audio, dashboard extras, and optional channel adapters. |
| Browser launcher | Keep `gormes dashboard` as the local launcher path and extend it with first-run CDP checks, Chrome install guidance, and an explicit headless/no-open mode for servers. |
| Skill marketplace | Design a ClawHub-like community skill source for Gormes that keeps bundled system skills separate from third-party taps, trust metadata, credential prerequisites, and review state. |

---

## Document Map

The `building-gormes/` directory contains 13 entries. Here is how they relate:

```
implementation-roadmap.md (this file) ──► decision tree + state + horizons
    │
    ├── must-have-features.md ──► feature catalogue from 12+ upstream projects
    │   └── cross-project-feature-map.md ──► detailed per-project matrix
    │
    ├── architecture_plan/ ──► phases, completion plan, lanes, progress.json
    │   ├── completion-plan.md ──► finish line definition
    │   ├── lane-roadmap.md ──► 6 lanes with exit gates
    │   ├── progress.json ──► canonical machine-readable queue
    │   ├── phase-1-dashboard.md ──► phase intent and boundaries
    │   ├── phase-2-gateway.md
    │   ├── phase-3-memory.md
    │   ├── phase-4-brain-transplant.md
    │   ├── phase-5-final-purge.md
    │   ├── phase-6-learning-loop.md
    │   ├── phase-7-paused-channel-backlog.md
    │   ├── hermes-honcho-feature-map.md ──► upstream → Go package mapping
    │   ├── hermes-honcho-go-runtime-plan.md ──► reconciled implementation plan
    │   ├── upstream-coverage-ledger.md ──► source-class completeness audit
    │   ├── swarm-feature-parity-audit.md ──► sub-agent gap register
    │   └── ... (20+ more reference docs)
    │
    ├── builder-loop/ ──► execution mechanics
    │   ├── agent-queue.md ──► generated: builder-ready rows
    │   ├── next-slices.md ──► generated: ranked shortlist
    │   ├── blocked-slices.md ──► generated: blocked rows with unblock conditions
    │   ├── umbrella-cleanup.md ──► generated: rows needing split
    │   ├── builder-loop-handoff.md ──► skill entrypoint + candidate policy
    │   └── progress-schema.md ──► row schema reference
    │
    ├── core-systems/ ──► stable runtime model
    │   ├── gateway.md ──► platform adapters, command policy, session routing
    │   ├── memory.md ──► recall, graph, search, mirrors, Goncho
    │   ├── tool-execution.md ──► operation registry, schema, trust classes
    │   └── learning-loop.md ──► skill detection, distillation, feedback
    │
    ├── gateway-donor-map/ ──► per-channel adaptation patterns
    │   ├── shared-adapter-patterns.md
    │   └── 15 channel dossiers (telegram, discord, slack, whatsapp, ...)
    │
    ├── goncho_honcho_memory/ ──► memory subsystem deep-dive
    │   ├── 01-prompts.md
    │   ├── 02-tool-schemas.md
    │   ├── 03-honcho-docs-study.md
    │   ├── 04-agent-work-packets.md
    │   └── 05-operator-playbook.md
    │
    ├── upstream-lessons.md ──► durable contracts from Hermes + GBrain
    ├── what-hermes-gets-wrong.md ──► why Gormes exists
    ├── contract-readiness.md ──► row-level handoff contract
    ├── porting-a-subsystem.md ──► contribution path for upstream ports
    └── testing.md ──► test strategy and fixture classes
```

**Rule:** If a document contradicts `progress.json`, `progress.json` wins. If `progress.json` contradicts this roadmap, this roadmap wins until a planner updates `progress.json`.

---

## Decision Tree: What Should I Work On?

### Q1: Are you a planner or a builder?

**Planner** → Go to [Agent Queue](../builder-loop/agent-queue/). If empty, go to [Next Slices](../builder-loop/next-slices/). If also empty, sharpen a planned row using `gormes-planner` skill.

**Builder** → Pick one row from Agent Queue. Read its contract, write_scope, test_commands, acceptance, and done_signal. Use `gormes-builder` + `gormes-tdd-slice` skills. Do not invent work outside the row.

### Q2: Is the row blocked?

Check [Blocked Slices](../builder-loop/blocked-slices/). If your row is listed, read the unblock condition. If you can satisfy it, do so. If not, pick a different row.

### Q3: Is the row an umbrella?

Check [Umbrella Cleanup](../builder-loop/umbrella-cleanup/). Umbrella rows are inventory only. Split them into small/medium/large rows before building. Use `gormes-planner` for splitting.

### Q4: Do you know which lane the row belongs to?

Use the [Lane Roadmap](../architecture_plan/lane-roadmap/) lane crosswalk. Each lane has an exit gate. Know the gate before you start.

### Q5: Do you know the upstream contract?

Read [Upstream Lessons](../upstream-lessons/) for durable contracts. Read [Hermes And Honcho Feature Map](../architecture_plan/hermes-honcho-feature-map/) for the upstream → Go package mapping. Read [Porting a Subsystem](../porting-a-subsystem/) for the contribution path.

### Q6: Is the Go shape unclear?

Use `gormes-interface-designer` skill. Read [Go Donor Reference Map](../architecture_plan/go-donor-reference-map/) for donor file patterns.

### Q7: Are you doing memory work?

Read [Core Systems: Memory](../core-systems/memory/) first. Then read [Goncho Honcho Memory](../goncho_honcho_memory/) for the deep-dive.

### Q8: Are you doing gateway/channel work?

Read [Core Systems: Gateway](../core-systems/gateway/) first. Then read the relevant [Gateway Donor Map](../gateway-donor-map/) dossier.

### Q9: Are you doing tool/security work?

Read [Core Systems: Tool Execution](../core-systems/tool-execution/) first. Then read [Must-Have Features §8](../must-have-features/#8-security--safety--the-trust-boundary).

---

## Execution Horizons

These horizons are derived from the [Must-Have Features](../must-have-features/) gap analysis and mapped to `progress.json` phases/subphases.

### Horizon 1: Safety + Provider Completion (Next 30 Days)

**Goal:** Close the two biggest blockers to a safe, Python-free agent turn.

| Week | Target | progress.json Rows | Lane | Why |
|------|--------|-------------------|------|-----|
| 1-2 | Complete xAI/Grok, LM Studio, DeepSeek/Kimi providers | 4.A (Bedrock runtime binding, Gemini, OpenRouter, Google Code Assist, Codex) | Lane 1 | Python-free turn requires all major providers |
| 1-2 | Tool descriptor layer (OperationSpec with trust classes) | 5.A (tool descriptor, toolsets) | Lane 3 | Every tool must declare who can call it |
| 2-3 | Prompt builder assembly closeout (skills snapshot, memory guidance) | 4.C (system+memory+tools+history, toolset-aware skills) | Lane 1 | Complete the prompt assembly pipeline |
| 3-4 | Shell blocklist + filesystem scoping | 5.J (dangerous action gating, Tirith/path/URL policy) | Lane 3 | **Critical safety gap** |
| 3-4 | Permission approval UX (inline y/n/always) | 5.J (approval workflow) | Lane 3 | **Critical safety gap** |
| 3-4 | Trust-class enforcement in shared tool executor | 5.A (tool descriptor enforcement) | Lane 3 | **Critical safety gap** |

**Exit criterion:** `gormes doctor` reports zero trust-class violations in default registry. All major providers have Go adapters. First Hermes-compatible normal turn runs without Python.

### Horizon 2: Production Hardening (Next 90 Days)

**Goal:** Make Gormes safe and reliable for real-world operation.

| Week | Target | progress.json Rows | Lane | Why |
|------|--------|-------------------|------|-----|
| 5-6 | Context compression complete | 4.B (long session management, manual feedback, kernel callback binding) | Lane 1 | Long sessions degrade without compression |
| 5-6 | Loop detection (5 types) | 5.J (loop detection) | Lane 3 | Runaway loops are a real production problem |
| 7-8 | Token budget system + auto-concise | 5.N (token accounting) | Lane 5 | Cost control for production deployments |
| 7-8 | Docker sandbox backend | 5.B (Docker backend) | Lane 3 | Sandboxed execution for untrusted code |
| 9-10 | Browser daemon lifecycle + doctor | 5.C (browser daemon, profile, doctor) | Lane 3 | Browser tools need production lifecycle |
| 9-10 | Code execution mode policy | 5.R (execution-mode resolver) | Lane 3 | Safe defaults for code execution |
| 11-12 | CLI closeout (backup, logs, diagnostics) | 5.O (backup, logs, diagnostics CLI) | Lane 5 | Operator needs visibility and recovery |
| 11-12 | Packaging closeout (install.sh, install.ps1, install.cmd) | 5.P (Unix/Windows installers) | Lane 5 | Frictionless installation |

**Exit criterion:** A new user can `curl install.sh | bash`, run `gormes doctor`, see green checks for all configured providers, and start a safe normal turn. Loop detection fires on runaway sessions. Token budget prevents surprise bills.

### Horizon 3: Memory Differentiation (Next 6 Months)

**Goal:** Make Goncho meaningfully better than session-based memory.

| Month | Target | progress.json Rows | Lane | Why |
|-------|--------|-------------------|------|-----|
| 3 | Typed memory categories + confidence scoring | 6.C (skill storage), 6.D (retrieval) | Lane 2 | Structured memory is major UX improvement |
| 3 | Zero-LLM knowledge graph wiring | 6.D (source-aware retrieval) | Lane 2 | Reduces LLM calls for entity resolution |
| 4 | Brain-first lookup (5-step before external API) | 6.D (retrieval eval) | Lane 2 | Significantly reduces LLM calls |
| 4 | Retrieval eval harness (precision@k, recall@k) | 6.D (retrieval eval) | Lane 2 | Turns "memory feels better" into testable contract |
| 5 | Metadata-driven skill placement | 6.C (portable SKILL.md format) | Lane 6 | More granular skill activation control |
| 5 | Soul/personality system (soul.md, persona.md, taste.md, heartbeat.md) | 6.A (complexity detector), 6.B (skill extractor) | Lane 6 | Planned for Phase 6 |
| 6 | Channel adapters (Matrix, Mattermost, LINE, IRC) | 7.C, 7.E | Lane 4 | Complete the long-tail channel surface |

**Exit criterion:** Goncho recall quality is measurably better than Hermes's default memory. Retrieval eval harness runs on every memory change. Skills activate with context-aware metadata. Personality files are operator-editable markdown.

### Horizon 4: Capstone Features (Next 12 Months)

**Goal:** Features that make Gormes uniquely valuable beyond Hermes parity.

| Month | Target | progress.json Rows | Lane | Why |
|-------|--------|-------------------|------|-----|
| 7-8 | Learning loop (skill extraction, feedback, scoring) | 6.A-6.F | Lane 6 | The feature Hermes doesn't have |
| 8-9 | Web dashboard (TypeScript/React) | 5.Q (API server + TUI gateway) | Lane 5 | Hermes has 191K-line TUI gateway |
| 9-10 | Code Cathedral II (call-graph edges, two-pass retrieval) | 6.D (Code Cathedral II) | Lane 6 | Code-aware agent capabilities |
| 10-11 | Multi-memory backends (Turbopuffer, LanceDB, Redis) | 3.* (future memory work) | Lane 2 | Scale beyond single-node SQLite |
| 11-12 | Three-agent memory loop (Deriver/Dialectic/Dreamer) | 3.* (future memory work) | Lane 2 | Honcho's unique memory paradigm |
| 11-12 | Mixture of agents (multi-model coordination) | 5.M | Lane 3 | Agent ensemble capabilities |

**Exit criterion:** Gormes is not just "Hermes in Go" — it is a demonstrably better agent runtime with compounding skills, proven memory quality, and operator-visible intelligence.

---

## Risk Register

Features or dependencies that could derail the plan:

| Risk | Impact | Mitigation | Owner |
|------|--------|-----------|-------|
| Security gaps persist (no shell blocklist, no filesystem scoping) | **Critical** — unsafe to operate | Horizon 1 priority #1 | Lane 3 |
| Provider parity stalls (Bedrock, Gemini, OpenRouter gaps) | **High** — Python still required | Weekly provider-audit pass | Lane 1 |
| Context compression never completes | **High** — long sessions degrade | Tight scope: only manual feedback + model-switch recalc | Lane 1 |
| Loop detection missing | **High** — production runaway loops | Port Mercury's 200-line TypeScript detector | Lane 3 |
| progress.json drifts from reality | **Medium** — wrong work gets built | `go run ./cmd/progress validate` on every PR | Lane 0 |
| Channel expansion outruns core agent | **Medium** — shallow adapters | Phase 7 rule: build only fixture-ready slices | Lane 4 |
| Learning loop scope creep | **Medium** — never ships | Hard gate: only after skill storage + resolver evals are reliable | Lane 6 |
| Memory backend abstraction too early | **Low** — SQLite-first promise broken | Keep Postgres behind interface; default remains SQLite | Lane 2 |

---

## Weekly Cadence (Recommended)

**Monday:** Review [Agent Queue](../builder-loop/agent-queue/) and [Blocked Slices](../builder-loop/blocked-slices/). Pick 1-2 rows.

**Tuesday-Thursday:** Build rows using `gormes-builder` + `gormes-tdd-slice`. Run `go test ./... -count=1` and `go run ./cmd/progress validate` before claiming done.

**Friday:** Review done signals. Update `progress.json` evidence. If queue is empty, run `gormes-planner` pass to sharpen planned rows.

**End of Month:** Run `gormes-parity-auditor` pass against one Hermes/Honcho subsystem. Update [Cross-Project Feature Map](../cross-project-feature-map/) if gaps changed.

---

## Success Metrics

| Horizon | Date Target | Metric | Current |
|---------|-------------|--------|---------|
| H1: Safety + Providers | May 30, 2026 | Provider parity >90%, zero trust-class violations | ~70% parity, 0 security hardening |
| H2: Production Harding | Jul 30, 2026 | `gormes doctor` all-green, loop detection shipped, token budget active | doctor partial, no loop detection, no budget |
| H3: Memory Differentiation | Oct 30, 2026 | Retrieval eval harness running, typed memories shipped, 15+ channels | no eval harness, no typed memories, 10 channels |
| H4: Capstone Features | Apr 30, 2027 | Learning loop extracting skills, web dashboard live, multi-model coordination | none started |
| **Final: Hermes in Go** | Apr 30, 2027 | 80%+ feature parity, all foundational + production gaps closed, differentiators shipping | ~30-40% parity |

---

## Quick Reference: skill → Document Mapping

| Skill | Primary Document | Secondary Documents |
|-------|-----------------|---------------------|
| `gormes-skill-manager` | [Skill Builder Handoff](../builder-loop/builder-loop-handoff/) | [Contract Readiness](../contract-readiness/) |
| `gormes-planner` | [Completion Plan](../architecture_plan/completion-plan/) | [Lane Roadmap](../architecture_plan/lane-roadmap/), [Must-Have Features](../must-have-features/) |
| `gormes-builder` | [Agent Queue](../builder-loop/agent-queue/) | [Next Slices](../builder-loop/next-slices/), [Contract Readiness](../contract-readiness/) |
| `gormes-tdd-slice` | [Testing](../testing/) | [Porting a Subsystem](../porting-a-subsystem/) |
| `gormes-parity-auditor` | [Cross-Project Feature Map](../cross-project-feature-map/) | [Hermes And Honcho Feature Map](../architecture_plan/hermes-honcho-feature-map/), [Upstream Coverage Ledger](../architecture_plan/upstream-coverage-ledger/) |
| `gormes-interface-designer` | [Go Donor Reference Map](../architecture_plan/go-donor-reference-map/) | [Core Systems](../core-systems/) |
| `gormes-provider-parity` | [GO-HERMES-PORTS-FORKS.md](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/docs/GO-HERMES-PORTS-FORKS.md) | [Upstream Lessons](../upstream-lessons/) |
| `gormes-browser-harness` | [Gateway Donor Map](../gateway-donor-map/) | [Core Systems: Gateway](../core-systems/gateway/) |
| `gormes-dev-runtime` | [Using Gormes](../../using-gormes/) | [Why Gormes](../../why-gormes/) |
| `gormes-references` | [Go Donor Reference Map](../architecture_plan/go-donor-reference-map/) | `references/go-agent-os/` |
| `gormes-readme` | [README.md](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/README.md) | [Why Gormes](../../why-gormes/) |
| `gormes-landing-web` | `www.gormes.ai/` | [Why Gormes](../../why-gormes/) |

---

*Generated: April 30, 2026*
*Source: progress.json v2.0, must-have-features.md, lane-roadmap.md, completion-plan.md*
*Update rule: Refresh this document when progress.json major state changes or when must-have-features.md is updated.*
