---
title: "Must-Have Features — Long-Term Plan"
description: "Comprehensive feature catalogue derived from 12+ open-source AI agent projects. Every feature a production Go agent runtime must ship."
weight: 18
---

# Must-Have Features — Long-Term Plan

> **Source scope:** 12+ open-source projects studied April 2026.
> **Projects:** hermes-agent (Python upstream), honcho (Python memory), gormes (TypeScript memory/runtime), browser-harness / go-browser-harness, mercury-agent (TypeScript), space-agent (JS browser-first), picoclaw (Go lightweight agent), MLT-OSS/hermes-agent-go (Go port), trpc-agent-go, AgenticGoKit, claw-prime, langchaingo, 8 go-agent-os reference repos.
> **Audience:** gormes-agent planner, builder, and reviewer skills. Not end-user marketing.
> **Relationship to progress.json:** This document names the features gormes must eventually have. The canonical execution queue is `progress.json`. This document is the _why_ and the _what_. `progress.json` is the _when_ and the _how_.

---

## Feature Architecture

Every feature is classified by three axes:

| Axis | Values | Meaning |
|------|--------|---------|
| **Criticality** | `foundational` / `production` / `differentiator` / `commodity` | How essential the feature is to gormes's existence |
| **Maturity** | `shipped` / `in-flight` / `planned` / `gap` | Current implementation status |
| **Complexity** | `trivial` / `moderate` / `large` / `subsystem` | Implementation effort required |

A feature is **must-have** if it scores `foundational` on criticality OR `production` on criticality with `gap` maturity. Differentiators are must-have when they define gormes's unique value. Commodity features are must-have only when they block user adoption.

---

## 1. Provider Adapters — The Model Surface

> **Donor projects:** hermes-agent, picoclaw, mercury-agent, gormes, trpc-agent-go, langchaingo
> **Current parity:** ~70% (7/10 major providers)
> **Criticality:** foundational

Every provider gormes cannot speak is a provider the operator cannot use without running Python Hermes alongside. This is not a nice-to-have. It is the single biggest block to a Python-free agent turn.

### 1.1 Must-Have Provider Completion

| Provider | Status | Rationale |
|----------|--------|-----------|
| Anthropic Messages API | Shipped | Primary provider for Gormes development |
| OpenAI Chat Completions | Shipped | Largest model ecosystem |
| Codex Responses API | Partial | Required for Codex CLI parity |
| AWS Bedrock (Converse + Anthropic) | In-flight | Enterprise deployment vector |
| Gemini / Google Cloud Code | Partial | Google ecosystem |
| Google Code Assist | Partial | Google developer tooling |
| OpenRouter | Partial | Multi-model routing; 250+ models |
| DeepSeek / Kimi reasoning | Partial | Reasoning-model isolation needed |
| Moonshot / Kimi tool schema | Partial | Chinese model ecosystem |
| xAI / Grok | Gap | Growing provider, no Go adapter |
| LM Studio / local inference | Gap | Offline-capable agent runtime |
| Azure Foundry (OpenAI + Anthropic) | In-flight | Enterprise Azure deployment |

### 1.2 Provider Infrastructure (Must-Have)

| Feature | Source | Status |
|---------|--------|--------|
| Provider-neutral stream contract | hermes-agent, gormes | Shipped |
| Tool-call normalization + continuation | hermes-agent | Shipped |
| Provider registry + alias manifest | hermes-agent, picoclaw | Shipped |
| Retry-After header parsing + backoff | plandex (go-agent-os) | Shipped |
| Provider error taxonomy (rate-limit vs auth vs capacity) | plandex, hermes-agent | Shipped |
| Prompt-cache capability guard | hermes-agent | Shipped |
| Provider rate guard + degraded-state evidence | hermes-agent | Shipped |
| Provider timeout config fail-closed | hermes-agent | Shipped |
| Cross-provider reasoning-tag sanitization | hermes-agent | Shipped |
| Tool-call argument repair + schema sanitizer | hermes-agent | Shipped |
| Model metadata registry (context limits, pricing, capabilities) | hermes-agent, gormes | In-flight |
| Smart model routing with fallback chain | mercury-agent (DeepSeek→OpenAI→Anthropic→Grok→Ollama) | In-flight |
| Per-turn model selection + reasoning effort propagation | hermes-agent | Shipped |
| Provider account usage read model (`/usage`) | hermes-agent | Shipped |

---

## 2. Native Prompt Assembly — The Agent's Brain

> **Donor projects:** hermes-agent (prompt_builder.py), mercury-agent (SOUL/persona system), space-agent (prompt-items.js)
> **Current parity:** ~40%
> **Criticality:** foundational

Without native prompt assembly, the Go runtime cannot construct a correct Hermes-compatible system prompt. Every turn depends on Python's prompt builder. This is the critical path for Python-free operation.

### 2.1 System Prompt Block Assembly

| Block | Source | Status |
|-------|--------|--------|
| SOUL.md identity + AGENTS.md project context | hermes-agent, mercury-agent (soul.md) | Shipped |
| USER.md + MEMORY.md durable user context | hermes-agent, space-agent (plain markdown includes) | Shipped |
| Tool/memory guidance constants (byte-equivalent port) | hermes-agent (prompt_builder.py) | Shipped |
| Skills guidance block (toolset-aware) | hermes-agent, space-agent (metadata-driven placement) | In-flight |
| Model-specific role + tool-use enforcement | hermes-agent | Shipped |
| Timestamp + model/provider/session metadata | hermes-agent (run_agent.py:3770-3779) | Shipped |
| Platform/session context block | hermes-agent | Shipped |
| Ephemeral memory/plugin recall (injected into user turn) | hermes-agent | Gap |
| WSL environment hint | hermes-agent | Shipped |
| Developer-role model routing | hermes-agent | Shipped |

### 2.2 Prompt Budgeting (Must-Have from space-agent design)

| Feature | Source | Status |
|---------|--------|--------|
| Prompt contributors as keyed, budgeted items | space-agent (prompt-items.js) | Gap |
| Cached token counts per contributor | space-agent | Gap |
| Trim policy with runtime read-back seam | space-agent | Gap |
| Long-context trimming without silent text drop | space-agent | Gap |

---

## 3. Context Compression — The Long Session Engine

> **Donor projects:** hermes-agent (ContextEngine), gormes (tiered enrichment)
> **Current parity:** ~60%
> **Criticality:** production

Real agent sessions run long. Without compression, the context window fills and the agent degrades. Hermes's ContextEngine is a well-defined contract. Gormes must port it completely.

### 3.1 Compression Engine

| Feature | Source | Status |
|---------|--------|--------|
| ContextEngine interface + status contract | hermes-agent | Shipped |
| Token-budget trigger + summary sizing | hermes-agent | Shipped |
| Protected head/tail message invariants | hermes-agent | Shipped |
| Tool call/result pairs kept together | hermes-agent | Shipped |
| Compression lineage record | hermes-agent | Shipped |
| Auxiliary compression headroom for system + tool schemas | hermes-agent | Shipped |
| Provider-aware context cap resolution | hermes-agent | Shipped |
| Tool-result pruning | hermes-agent | Shipped |
| Image-token budget charge | hermes-agent | Shipped |
| Compression-boundary callback (notify kernel) | hermes-agent | In-flight |
| Manual compression feedback + context references | hermes-agent | Gap |
| Kernel compression-boundary callback binding | hermes-agent | Gap |
| **Recalculate budgets on model-switch** | hermes-agent (2026-04-25 sync) | Gap |

---

## 4. Gateway + Channels — The Operator Surface

> **Donor projects:** hermes-agent (gateway/run.py), picoclaw (18+ channels), mercury-agent (permission-hardened Telegram)
> **Current parity:** ~60% (5/19 channels shipped; shared chassis in place)

### 4.1 Shared Gateway Infrastructure (Must-Have)

| Feature | Source | Status |
|---------|--------|--------|
| Shared gateway chassis (multi-channel) | hermes-agent, gormes | Shipped |
| Slash command registry + active-turn policy | hermes-agent | Shipped |
| Session store + SessionSource parity | hermes-agent | Shipped |
| Session metadata refresh (created, activity, platform, title) | hermes-agent | Shipped |
| Gateway auto-title generation | hermes-agent | Shipped |
| Gateway `/status` Hermes-format closeout | hermes-agent | Shipped |
| Gateway `/title` manual title command | hermes-agent | Shipped |
| Gateway session token accounting | hermes-agent | Shipped |
| Gateway stream/tool trace formatting | hermes-agent | Shipped |
| Gateway typing-action wiring during stream | hermes-agent | Shipped |
| Placeholder edit-failure fallback hardening | hermes-agent | Shipped |
| Gateway fresh-final send/delete fallback | hermes-agent | Shipped |
| Gateway message deduplicator | hermes-agent | Shipped |
| Gateway inbound dedup manager binding | hermes-agent | Shipped |
| **Notify-to delivery routing** | hermes-agent | Gap |
| **Channel directory refresh + stale-target invalidation** | hermes-agent | Gap |
| **Mirror + sticker cache surfaces** | hermes-agent | Gap |

### 4.2 Channel Adapters (Must-Have by Priority)

| Channel | Status | Priority Rationale |
|---------|--------|--------------------|
| Telegram | Shipped | Primary dogfood channel |
| Discord | Shipped | Developer community channel |
| Slack | Shipped | Enterprise channel |
| WhatsApp | Shipped | Global messaging |
| WeChat (WeCom + WeiXin) | Shipped | Chinese enterprise |
| BlueBubbles (iMessage) | Shipped | Apple ecosystem |
| Home Assistant | Shipped | Smart home integration |
| Signal | Partial | Privacy-focused users |
| Email + SMS | Shipped | Fallback delivery |
| Matrix | Gap | Open federated protocol |
| Mattermost | Gap | Self-hosted enterprise |
| Feishu / Lark | Partial | Chinese enterprise |
| DingTalk | Partial | Chinese enterprise |
| QQ | Partial | Chinese consumer |
| LINE | Gap | Japanese market |
| VK | Gap | Russian market |
| IRC | Gap | Legacy community |
| Yuanbao | Partial | Chinese AI platform |
| **Webhook (generic)** | Shipped | Universal integration |

### 4.3 Gateway Active-Turn Policy (Must-Have from Hermes Design)

| Feature | Source | Status |
|---------|--------|--------|
| `/stop` interrupt | hermes-agent | Shipped |
| `/queue` wait | hermes-agent | Shipped |
| `/steer` mid-turn injection | hermes-agent | Shipped |
| Busy command guard for compression + long CLI actions | hermes-agent | Shipped |
| Gateway-handled slash commands bypass active-session guard | hermes-agent | Shipped |
| Mid-run steer injection between tool calls | hermes-agent | Shipped |
| Hardline command blocks (yolo/approval-off/cron) | hermes-agent (2026-04-26 sync) | Partial |
| **Approvals bypass interrupt semantics** | hermes-agent | Gap |

---

## 5. Tool Registry — The Agent's Hands

> **Donor projects:** hermes-agent (40+ tools), mercury-agent (31 tools, permission-hardened), picoclaw (Go native tools), gormes (operation catalog)
> **Current parity:** ~25% (15/61 tools)
> **Criticality:** foundational

### 5.1 Tool Infrastructure (Must-Have Before Tool Breadth)

| Feature | Source | Status |
|---------|--------|--------|
| Typed Tool interface + in-process registry | hermes-agent, gormes | Shipped |
| **Tool descriptor layer** (toolset, availability, mutating flag, trust class, timeout, result budget, audit kind, prompt-visible flag) | gormes (OperationSpec), hermes-agent | Gap |
| Streamed tool_calls accumulation | hermes-agent | Shipped |
| Kernel tool loop (90-turn budget, summary grace turn) | hermes-agent | Shipped |
| Doctor verification (`--offline`) | hermes-agent, gormes | Shipped |
| Tool registry inventory + schema parity harness | hermes-agent | Shipped |
| **Executor-enforced trust class + timeout before handler runs** | gormes (trust-class enforcement) | Gap |

### 5.2 Core Task Tools (Must-Have for Agent Utility)

| Tool | Source | Status |
|------|--------|--------|
| **read_file** | hermes-agent | Shipped |
| **search_files** | hermes-agent | Shipped |
| **write_file** | hermes-agent | Shipped |
| **patch** | hermes-agent | Shipped |
| **terminal (foreground)** | hermes-agent | Shipped |
| Terminal (background, PTY) | hermes-agent | Partial |
| **todo** (session-bound) | hermes-agent | Shipped |
| **session_search** (session-bound) | hermes-agent | Shipped |
| **clarify** | hermes-agent | Shipped |
| **Checkpoint manager** (atomic snapshot, shadow-repo) | hermes-agent | In-flight |
| Code execution (sandboxed) | hermes-agent | Shipped |
| Process registry | hermes-agent | Gap |
| **Fuzzy patch/lint/checkpoint restore** | hermes-agent | Gap |

### 5.3 Web + Browser Tools

| Tool | Source | Status |
|------|--------|--------|
| Browser navigate/snapshot/click/type/scroll | hermes-agent, go-browser-harness | Shipped |
| Browser console/vision/CDP | hermes-agent, go-browser-harness | Shipped |
| Browser dialog/get-images | hermes-agent, go-browser-harness | Shipped |
| Web search (DuckDuckGo, Tavily, Brave, Perplexity, SearXNG) | hermes-agent, picoclaw | Shipped |
| Web extract + crawl | hermes-agent | Shipped |
| **Browser daemon lifecycle** (start/stop/ensure) | hermes-agent | Gap |
| **Browser profile management** | hermes-agent | Gap |
| **Browser tab enumeration/switch** | hermes-agent | Gap |
| **Browser doctor/diagnostics** | hermes-agent | Gap |
| **Browser event drain** | hermes-agent | Gap |
| Browser Use cloud + Go harness bridge | hermes-agent, go-browser-harness | Shipped |
| **HTTP GET (no-browser)** | hermes-agent | Gap |
| **Iframe targeting** | hermes-agent | Gap |

### 5.4 Media + Vision Tools

| Tool | Source | Status |
|------|--------|--------|
| Image generation | hermes-agent | Shipped |
| Image input mode router + native content parts | hermes-agent | Shipped |
| **Vision analysis** | hermes-agent | Gap |
| **TTS synthesis + voice-mode state** | hermes-agent | Gap |
| Transcription tool | hermes-agent | Shipped |

### 5.5 Delegation + Subagents

| Feature | Source | Status |
|---------|--------|--------|
| Deterministic subagent runtime (goroutines) | hermes-agent, gormes | Shipped |
| Max-depth guard + bounded batch execution | hermes-agent, gormes | Shipped |
| Timeout + cancellation scopes (context.Context) | hermes-agent, gormes | Shipped |
| Typed result envelope + append-only run log | hermes-agent, gormes | Shipped |
| Runner-enforced tool allowlists + blocked-tool policy | hermes-agent, gormes | Shipped |
| Interrupt propagation to concurrent-tool workers | hermes-agent, gormes | Shipped |
| **Child-agent message/tool-call replay from ledger** | gormes (subagent_messages) | Shipped |
| **Durable subagent/job ledger** (SQLite) | gormes (Minions), gormes | Shipped |
| **Background review toolset restriction** | hermes-agent | Shipped |

### 5.6 Scheduling + Background Tasks

| Feature | Source | Status |
|---------|--------|--------|
| robfig/cron scheduler + bbolt/SQLite job store | hermes-agent | Shipped |
| Heartbeat [SYSTEM:] + [SILENT] delivery contract | hermes-agent | Shipped |
| **Durable job queue** (claim, renew, complete, fail, retry, cancel) | gormes (Minions), gormes | Shipped |
| Durable job backpressure + timeout audit | gormes, gormes | Shipped |
| Durable worker supervisor status seam | gormes, gormes | Shipped |
| Durable pause/resume intent contract | gormes, gormes | Shipped |
| Durable replay and inbox message contract | gormes, gormes | Shipped |
| **Cronjob tool API + schedule parser parity** | hermes-agent | Gap |
| **Parent-child DAGs with completion policies** | gormes (Minions) | Gap |
| **Per-job token accounting** | gormes | Gap |
| **Cron multi-target delivery** | hermes-agent | Shipped |
| **Cron context_from output chaining** | hermes-agent | Shipped |

### 5.7 MCP (Model Context Protocol)

| Feature | Source | Status |
|---------|--------|--------|
| MCP client (stdio + HTTP transport) | hermes-agent, gormes | Shipped |
| MCP tool discovery + schema normalization | hermes-agent, gormes | Shipped |
| MCP OAuth state store + token refresh | hermes-agent, gormes | Shipped |
| Goncho MCP tool catalog | gormes | Shipped |
| Gormes-native MCP host runtime boundary | gormes | Shipped |

---

## 6. Memory System — Goncho / Honcho Parity

> **Donor projects:** honcho (3-agent memory, peer paradigm), gormes (knowledge graph, brain-first lookup), mercury-agent (Second Brain, 10 typed memories), hermes-agent (MemoryProvider interface)
> **Current parity:** Shipped (Goncho Phase 3 complete)
> **Criticality:** production (core is shipped; enhancements are differentiators)

### 6.1 Shipped Memory Foundation

| Feature | Status |
|---------|--------|
| SQLite + FTS5 lattice + schema migrations | Shipped |
| Ontological graph + LLM extractor + dead-letter queue | Shipped |
| Neural recall (2-layer seed selection, CTE traversal) | Shipped |
| Semantic fusion (Ollama embeddings, cosine similarity, hybrid) | Shipped |
| Memory mirror (USER.md async export) | Shipped |
| Session index mirror (bbolt→index.yaml) | Shipped |
| Tool execution audit log (JSONL) | Shipped |
| Transcript export (gormes session export) | Shipped |
| Extraction state visibility (gormes memory status) | Shipped |
| Insights audit log (daily usage.jsonl) | Shipped |
| Memory decay (relationships.last_seen + weight attenuation) | Shipped |
| Cross-chat synthesis (user_id concept, opt-in cross-chat recall) | Shipped |
| Session lineage + cross-source search | Shipped |
| Honcho-compatible scope/source tool schemas | Shipped |
| Honcho host + SillyTavern integration fixtures | Shipped |
| Goncho configuration namespace + dreaming scheduler | Shipped |
| Goncho streaming chat persistence | Shipped |
| Hermes memory tool over Goncho/local store | Shipped |
| Goncho drop-in compatibility (keys, webhooks, HTTP, CLI, SDK) | Shipped |

### 6.2 Memory Enhancements (Must-Have Differentiators)

| Feature | Source | Status |
|---------|--------|--------|
| **Typed memory categories** (identity, preference, goal, habit, episode, reflection) | mercury-agent (10 types), honcho | Gap |
| **Confidence + durability scoring** per observation | mercury-agent, honcho | Gap |
| **Conflict resolution** (confidence wins; equal→newer) | mercury-agent, honcho | Gap |
| **Auto-pruning** (21-day stale memories) | mercury-agent | Gap |
| **Zero-LLM knowledge graph wiring** (regex-based auto-links) | gormes | Gap |
| **Brain-first lookup** (5-step before external API) | gormes | Gap |
| **Retrieval eval harness** (precision@k, recall@k, cross-chat negative tests) | gormes | Gap |
| **Source-aware retrieval ranking** (curated/reviewed boost, bulk-source damping) | gormes | Gap |
| **Compiled truth + timeline pattern** (per-entity page) | gormes | Gap |
| **Directional peer cards** (users AND agents as peers) | honcho | Gap |
| **Multi-memory backends** (Turbopuffer, LanceDB, Redis cache layer) | honcho | Gap |
| **Three-agent memory loop** (Deriver/Dialectic/Dreamer) | honcho | Gap |

---

## 7. CLI / Operator Surface — The Control Panel

> **Donor projects:** hermes-agent (49-file CLI tree), mercury-agent (token budget, loop detection)
> **Current parity:** ~50%
> **Criticality:** production

### 7.1 CLI Commands (Must-Have by Function)

| Command | Source | Status |
|---------|--------|--------|
| `gormes` (TUI, scripted chat, gateway modes) | hermes-agent | Shipped |
| `gormes doctor` | hermes-agent | Shipped |
| `gormes gateway status/start/stop` | hermes-agent | Shipped |
| `gormes session export` | hermes-agent | Shipped |
| `gormes memory status` | gormes | Shipped |
| `gormes config` (check, edit, migrate) | hermes-agent | Shipped |
| `gormes auth` (login, logout, status, add) | hermes-agent | In-flight |
| `gormes model` (interactive picker) | hermes-agent | Shipped |
| `gormes setup` (wizard) | hermes-agent | Shipped |
| `gormes profile` | hermes-agent | Shipped |
| `gormes agent reset` | hermes-agent | Shipped |
| `gormes mcp login` | hermes-agent | Shipped |
| **`gormes backup` / `gormes restore`** | hermes-agent | Gap |
| **`gormes logs`** | hermes-agent | Gap |
| **`gormes web-server`** | hermes-agent | Gap |
| **`gormes uninstall`** | hermes-agent | Gap |
| **`gormes cron` management CLI** | hermes-agent | Gap |
| **`gormes plugins` management** | hermes-agent | Gap |
| **Diagnostics, backup, and status CLI closeout** | hermes-agent | Gap |

### 7.2 API Server (Must-Have)

| Feature | Source | Status |
|---------|--------|--------|
| OpenAI-compatible chat-completions API | hermes-agent | Shipped |
| Responses API store + run event stream | hermes-agent | Shipped |
| API server disconnect snapshot persistence | hermes-agent | Shipped |
| Gateway proxy mode forwarding | hermes-agent | Shipped |
| Dashboard API client contract | hermes-agent | Shipped |
| API server detailed health endpoint | hermes-agent | Shipped |
| API server cron admin endpoints | hermes-agent | Shipped |
| **SSE streaming to Bubble Tea TUI** | gormes | Shipped |

### 7.3 TUI (Terminal UI)

| Feature | Source | Status |
|---------|--------|--------|
| Bubble Tea shell + 16ms coalescing mailbox | gormes | Shipped |
| SSE reconnect | gormes | Shipped |
| Hermes skin token renderer | hermes-agent, gormes | Shipped |
| Hermes status bar + bottom-pinned chrome layout | hermes-agent, gormes | Shipped |
| Native TUI slash-command dispatch | gormes | Shipped |
| Native TUI `/save`, `/branch`, conversation viewport | gormes | Shipped |
| **Web dashboard** (TypeScript/React parity with Hermes UI) | hermes-agent | Gap |

---

## 8. Security + Safety — The Trust Boundary

> **Donor projects:** mercury-agent (permission system), hermes-agent (approval, URL safety, Tirith), gormes (trust-class enforcement)
> **Current parity:** ~30%
> **Criticality:** foundational

**Gormes has zero permission hardening.** This is the most critical gap in the entire project. An agent that can execute arbitrary shell commands, write to any filesystem path, and make arbitrary network requests is not safe to operate.

### 8.1 Must-Have Security Features

| Feature | Source | Status |
|---------|--------|--------|
| **Shell blocklist** (36+ dangerous patterns) | mercury-agent | **Gap** |
| **Filesystem scoping** (folder-level read/write restrictions) | mercury-agent | **Gap** |
| **Permission approval UX** (inline y/n/always) | mercury-agent, hermes-agent | **Gap** |
| Hardline command pattern detection (non-recoverable host destruction) | hermes-agent, gormes | In-flight |
| Recoverable dangerous patterns + blocked-result schema | hermes-agent, gormes | In-flight |
| Approval mode config normalization | hermes-agent, gormes | In-flight |
| Gateway hook auto-accept strict parser | hermes-agent, gormes | Shipped |
| Subagent dangerous-command non-interactive policy | gormes | Shipped |
| **URL safety / SSRF guard** | hermes-agent | Partial |
| **Path security** (traversal prevention, workspace confinement) | hermes-agent | Gap |
| **Credential redaction** in logs and telemetry | hermes-agent | Partial |
| **Website policy** (per-domain allow/block) | hermes-agent | Gap |
| **Cron dangerous-command approval mode** | hermes-agent | Gap |
| **Tirith security integration** | hermes-agent | Gap |
| **OSV vulnerability check** | hermes-agent | Gap |
| **Trust-class enforcement in shared tool executor** | gormes, mercury-agent | **Gap** |
| **WhatsApp identity safety** (ASCII-only, anti-traversal) | hermes-agent (2026-04-27 sync) | Shipped |

### 8.2 Operator Safety (Must-Have from Mercury)

| Feature | Source | Status |
|---------|--------|--------|
| **Loop detection** (5 types: hard/failing/text/no-action/same-tool) | mercury-agent | **Gap** |
| **Token budget system** (daily + per-session, auto-concise at 70%) | mercury-agent | **Gap** |
| **Permission manifest** (YAML-based, per-agent allow/deny) | mercury-agent | Gap |
| Filesystem operation preflight (batch validation before writes) | space-agent | Gap |

---

## 9. Skills System — The Agent's Knowledge

> **Donor projects:** hermes-agent (skills_hub.py, 118K lines), gormes (fat skills, thin harness), space-agent (metadata-driven skills), mercury-agent (skill preprocessing)
> **Current parity:** ~40%
> **Criticality:** differentiator

### 9.1 Skills Infrastructure

| Feature | Source | Status |
|---------|--------|--------|
| SKILL.md parsing + active store | hermes-agent, gormes | Shipped |
| Deterministic selection + prompt injection block | hermes-agent, gormes | Shipped |
| Kernel injection + usage log | hermes-agent, gormes | Shipped |
| Inactive candidate drafting + explicit promotion | hermes-agent, gormes | Shipped |
| SKILL.md frontmatter validation guard | gormes, gormes | Shipped |
| Hermes creative skill metadata compatibility | hermes-agent, gormes | Shipped |
| Skill preprocessing + dynamic slash commands | hermes-agent, gormes | Shipped |
| Skills hub search (in-memory registry) | hermes-agent, gormes | Shipped |
| Skills list (enabled/disabled status + filter) | gormes | Shipped |
| **Metadata-driven skill placement** (when, loaded, placement, context tags) | space-agent | Gap |
| **Skill conformance checks + routing evals** | gormes | Gap |
| **Skill conflict detection** (overlapping triggers) | gormes | Gap |
| **Bundled productivity skills** (Airtable, Spotify, TouchDesigner, Google Meet) | hermes-agent | In-flight |
| **Skill registries** (external repo fetching) | hermes-agent | Gap |
| **Skill confidence/feedback scoring** | hermes-agent | Gap |
| **Skill versioning + review history** | gormes | Gap |

---

## 10. Sandboxing + Execution Environments

> **Donor projects:** hermes-agent (Docker, Modal, Daytona, Singularity), picoclaw (7 terminal backends)
> **Current parity:** ~30%
> **Criticality:** production (Docker is must-have; others are differentiators)

| Backend | Source | Status | Priority |
|---------|--------|--------|-----------|
| **Docker** | hermes-agent | In-flight | Must-have |
| Environment interface + file sync contract | hermes-agent, gormes | Shipped | Must-have |
| Modal | hermes-agent | Gap | Differentiator |
| Daytona | hermes-agent | Gap | Differentiator |
| Singularity | hermes-agent | Gap | Differentiator |
| Local (SSH, direct exec) | hermes-agent, picoclaw | Shipped | Shipped |
| **Code execution mode policy** (strict/project/default resolver) | hermes-agent | Gap | Must-have |

---

## 11. Packaging + Distribution

> **Donor projects:** hermes-agent, picoclaw (Desktop/Headless/Android/Docker/Termux), claw-prime
> **Current parity:** ~60%
> **Criticality:** production

| Artifact | Status | Priority |
|----------|--------|----------|
| Single Go binary (Linux, macOS, Windows) | Shipped | Foundational |
| OCI / Docker image | Shipped | Must-have |
| Homebrew formula | Shipped | Must-have |
| Unix installer (install.sh) source-backed update flow | Shipped | Must-have |
| Windows installer (install.ps1 + install.cmd) | Shipped | Must-have |
| **Installer site asset/route coverage** | Gap | Must-have |
| Debian/RPM packages | Gap | Commodity |
| Android APK / Termux | Gap | Commodity |

---

## 12. Soul / Personality System — Phase 6

> **Donor projects:** mercury-agent (soul.md, persona.md, taste.md, heartbeat.md), space-agent (personality.system.include.md)
> **Current parity:** Not started (Phase 6)
> **Criticality:** differentiator

| Feature | Source | Status |
|---------|--------|--------|
| SOUL.md identity (heart) | mercury-agent | Planned |
| persona.md behavior (face) | mercury-agent | Planned |
| taste.md preferences (palate) | mercury-agent | Planned |
| heartbeat.md lifecycle (breathing) | mercury-agent | Planned |
| User-owned markdown files (operator-editable) | mercury-agent, space-agent | Planned |
| **Plain-text memory includes as first-class** | space-agent | Planned |
| **Git-backed personality time travel** (rollback/revert) | space-agent | Planned |

---

## 13. Learning Loop — Phase 6

> **Donor projects:** gormes (skill extraction, maintenance cycles), hermes-agent (skill generation)
> **Current parity:** Not started (Phase 6)
> **Criticality:** differentiator

| Feature | Source | Status |
|---------|--------|--------|
| Complexity detector (heuristic/LLM signal) | hermes-agent, gormes | Gap |
| Skill extractor (LLM-assisted pattern distillation) | hermes-agent, gormes | Gap |
| Hybrid lexical + semantic skill lookup | gormes, gormes | Gap |
| Skill effectiveness scoring + feedback loop | gormes, gormes | Gap |
| TUI + Telegram skill browsing | gormes | In-flight |
| **Delta-bounded skill/memory maintenance** (changed-source IDs only) | gormes (e2961c0) | In-flight |
| **Source-aware retrieval damping fixtures** | gormes | Gap |
| **Code Cathedral II** (call-graph edges, two-pass retrieval) | gormes | Gap |
| **Incremental extract + embed-stale filters** | gormes (v0.22.1) | Gap |
| **Per-source sync anchor** (source-scoped last-commit) | gormes (v0.22.5) | Gap |
| **Fail-improve loop** (logs regex failures, generates better patterns) | gormes | Gap |

---

## 14. Observability + Telemetry

> **Donor projects:** hermes-agent (logs, status, tips), gormes (doctor, degraded mode), mercury-agent (heartbeat)
> **Current parity:** ~40%
> **Criticality:** production

| Feature | Source | Status |
|---------|--------|--------|
| `gormes doctor --offline` | hermes-agent | Shipped |
| `gormes gateway status` (read-only) | hermes-agent, gormes | Shipped |
| `gormes memory status` (extractor queue, dead-letter) | gormes | Shipped |
| Tool execution audit log (JSONL) | gormes | Shipped |
| Insights audit log (daily usage.jsonl) | gormes | Shipped |
| CLI log redactor (secret shapes) | hermes-agent | Shipped |
| CLI log snapshot reader | hermes-agent | Shipped |
| CLI dump support-summary helper | hermes-agent | Shipped |
| **Visible degraded mode** (every fallback reported in status/doctor/audit) | gormes, mercury-agent | Gap |
| **Semantic recall disabled warning** (no embedding model) | gormes | Gap |
| **Extractor queue depth + dead-letter count** | gormes, gormes | Shipped |
| **Graph extraction age** | gormes | Gap |
| **Skill resolver warnings** | gormes | Gap |
| **Durable job stalled count** | gormes, gormes | Shipped |
| **Child-agent replay not available warning** | gormes | Gap |
| **Tips system** (deterministic, seen-state) | hermes-agent, gormes | Shipped |
| **Web dashboard** (session, skills, chat, config, logs) | hermes-agent | Gap |

---

## 15. Plugin Architecture

> **Donor projects:** hermes-agent (plugin SDK, hooks, tools, slash commands), gormes (isolated MCP plugins)
> **Current parity:** ~40%
> **Criticality:** commodity (must-have infrastructure; plugins themselves are differentiators)

| Feature | Source | Status |
|---------|--------|--------|
| Plugin SDK | hermes-agent | Shipped |
| Dashboard theme/plugin slot inventory | gormes | Shipped |
| First-party Spotify plugin fixture | hermes-agent, gormes | Shipped |
| First-party Google Meet plugin fixture | hermes-agent, gormes | Shipped |
| **Third-party extension system** | hermes-agent | Gap |
| **Plugin isolation** (subprocess/WASM for untrusted) | gormes, hermes-agent | Gap |
| **Plugin manifest + capability declaration** | gormes | Gap |
| **Operator enablement required before activation** | gormes | Gap |
| **Hindsight memory setup** | hermes-agent, gormes | Gap |

---

## 16. Mixture of Agents

> **Donor projects:** hermes-agent (multi-model coordination), gormes (subagent orchestration)
> **Current parity:** Gap
> **Criticality:** differentiator

| Feature | Source | Status |
|---------|--------|--------|
| Multi-model coordination | hermes-agent | Gap |
| Agent ensemble with voting/consensus | hermes-agent | Gap |
| Model-specific task routing | hermes-agent | Gap |

---

## Cross-Cutting Must-Have Patterns (from All Projects)

These are not features but architectural patterns that every subsystem must adopt:

| Pattern | Source | Applied To |
|---------|--------|------------|
| **Contract-first operations** (one descriptor → CLI, MCP, tool schemas, doctor) | gormes | Tools, commands, gateways |
| **Trust-class enforcement in shared executor** (operator/gateway/child/system) | gormes, hermes-agent | All tool execution |
| **Visible degraded mode** (every fallback reported) | gormes, mercury-agent | All subsystems |
| **Provider-neutral events** (adapters own quirks; kernel sees one contract) | hermes-agent | Provider layer |
| **Thin harness, fat skills** (intelligence in skills, not runtime) | gormes | Skills system |
| **Hermetic fixtures over live tests** (replayable, no live credentials) | gormes, hermes-agent | All tests |
| **Metadata-driven discovery** (not hardcoded branches) | space-agent | Skills, tools, commands |
| **Prompt contributors as keyed, budgeted items** | space-agent | Prompt assembly |
| **Goroutines + channels** (not asyncio threads) | All Go projects | Concurrency |
| **Context.Context propagation** (cancellation, deadlines) | All Go projects | All I/O |
| **SQLite + FTS5 first** (Postgres optional behind interface) | gormes, gormes | Memory, jobs, state |
| **Single binary, zero runtime deps** | claw-prime, gormes | Distribution |
| **Durable job ledger** (restartable work, not ephemeral goroutines) | gormes | Cron, subagents, long work |
| **Provenance-rich memory** (source, extractor, confidence, freshness) | gormes, honcho | Goncho relationships |
| **Skills as reviewed code** (frontmatter, conformance, routing evals) | gormes, hermes-agent | Skills store |

---

## Feature Gap Summary by Criticality

### Foundational Gaps (Block Python-Free Agent Turn)

| # | Feature | Source Project | Estimated Effort |
|---|---------|---------------|-------------------|
| F1 | Provider completion (xAI/Grok, LM Studio, DeepSeek, Moonshot) | hermes-agent, picoclaw | 4-6 weeks |
| F2 | Native prompt builder assembly (skills snapshot, memory/search guidance) | hermes-agent | 2-3 weeks |
| F3 | Tool descriptor layer (toolset, trust class, timeout, audit, budget) | gormes | 2-3 weeks |
| F4 | Shell blocklist + filesystem scoping | mercury-agent | 2-3 weeks |
| F5 | Permission approval UX (inline y/n/always) | mercury-agent, hermes-agent | 2-3 weeks |
| F6 | Trust-class enforcement in shared executor | gormes | 1-2 weeks |

### Production Gaps (Block Real-World Operation)

| # | Feature | Source Project | Estimated Effort |
|---|---------|---------------|-------------------|
| P1 | Context compression complete (manual feedback, model-switch recalculation) | hermes-agent | 2-3 weeks |
| P2 | Loop detection (5 types) | mercury-agent | 1-2 weeks |
| P3 | Token budget system + auto-concise | mercury-agent | 2-3 weeks |
| P4 | URL safety / SSRF guard complete | hermes-agent | 1-2 weeks |
| P5 | Browser daemon lifecycle + doctor | hermes-agent, go-browser-harness | 2-3 weeks |
| P6 | Docker sandbox backend | hermes-agent | 1-2 weeks |
| P7 | Code execution mode policy (strict/project/default) | hermes-agent | 2-3 weeks |
| P8 | CLI closeout (backup, logs, web-server, uninstall, diagnostics) | hermes-agent | 3-4 weeks |
| P9 | Packaging closeout (install.sh, install.ps1, install.cmd) | hermes-agent | 2-3 weeks |
| P10 | Visible degraded mode in all subsystems | gormes, mercury-agent | Ongoing |
| P11 | Channel adapters (Matrix, Mattermost, LINE, IRC, VK, Feishu/DingTalk closeout) | hermes-agent, picoclaw | 4-6 weeks |

### Differentiator Gaps (Define Gormes's Unique Value)

| # | Feature | Source Project | Estimated Effort |
|---|---------|---------------|-------------------|
| D1 | Typed memory categories + confidence scoring | mercury-agent, honcho | 3-4 weeks |
| D2 | Zero-LLM knowledge graph wiring (regex auto-links) | gormes | 2-3 weeks |
| D3 | Brain-first lookup (5-step before external API) | gormes | 2-3 weeks |
| D4 | Retrieval eval harness (precision@k, recall@k) | gormes | 2-3 weeks |
| D5 | Metadata-driven skill placement (when, loaded, context tags) | space-agent | 2-3 weeks |
| D6 | Minions DAG job dependencies (parent-child with policies) | gormes | 3-4 weeks |
| D7 | Soul/personality system (soul.md, persona.md, taste.md, heartbeat.md) | mercury-agent | 2-3 weeks |
| D8 | Learning loop (complexity detector, skill extractor, feedback) | hermes-agent, gormes | 6-8 weeks |
| D9 | Code Cathedral II (call-graph edges, two-pass retrieval) | gormes | 4-6 weeks |
| D10 | Web dashboard (TypeScript/React parity with Hermes UI) | hermes-agent | 6-8 weeks |
| D11 | Multi-memory backends (Turbopuffer, LanceDB, Redis) | honcho | 3-4 weeks |
| D12 | Three-agent memory loop (Deriver/Dialectic/Dreamer) | honcho | 4-6 weeks |
| D13 | Mixture of agents (multi-model coordination) | hermes-agent | 6-8 weeks |

---

## Implementation Sequence (Recommended)

### Immediate (Next 30 Days) — Close the Safety Gap + Provider Gap

```
Week 1-2: Provider completion (xAI/Grok, LM Studio, DeepSeek/Kimi)
Week 1-2: Tool descriptor layer (OperationSpec with trust classes)
Week 2-3: Prompt builder assembly final closeout
Week 3-4: Shell blocklist + filesystem scoping + permission approval
Week 3-4: Trust-class enforcement in tool executor
```

### Short-Term (Next 90 Days) — Production Hardening

```
Week 5-6: Context compression complete (manual feedback, recalculation)
Week 5-6: Loop detection (5 types)
Week 7-8: Token budget system + auto-concise
Week 7-8: Docker sandbox backend
Week 9-10: Browser daemon lifecycle + doctor
Week 9-10: Code execution mode policy
Week 11-12: CLI closeout (backup, logs, diagnostics)
Week 11-12: Packaging closeout (installers)
```

### Medium-Term (Next 6 Months) — Differentiators

```
Month 3: Typed memory categories + confidence scoring
Month 3: Zero-LLM knowledge graph wiring
Month 4: Brain-first lookup + retrieval eval harness
Month 4: Minions DAG job dependencies
Month 5: Metadata-driven skill placement
Month 5: Soul/personality system
Month 6: Channel adapters (Matrix, Mattermost, LINE, IRC)
```

### Long-Term (Next 12 Months) — Capstone Features

```
Month 7-8: Learning loop (skill extraction, feedback)
Month 8-9: Web dashboard
Month 9-10: Code Cathedral II
Month 10-11: Multi-memory backends
Month 11-12: Three-agent memory loop + Mixture of agents
```

---

## Not Recommended for Gormes

Features studied but explicitly excluded from the must-have list:

| Feature | Source | Reason |
|---------|--------|--------|
| Browser-first runtime (agent lives in browser) | space-agent | Fundamentally different runtime model |
| WebLLM / HuggingFace browser-side inference | space-agent | Requires browser runtime, not Go |
| Mercury's Telegram org model (admin/member roles) | mercury-agent | Gormes gateway channels use different abstraction |
| Picoclaw's ultra-lightweight constraints ($10 hardware) | picoclaw | Different deployment target |
| Full Kubernetes ACP controllers | go-agent-os references | Use local state machine instead |
| TypeScript indexer / tree-sitter WASM in runtime | gormes | Keep runtime small; optional code-context evidence only |
| Hermes's dynamic Python import-time tool registration | hermes-agent | Explicit registration is safer for Go |
| Hermes's sync/async/thread complexity | hermes-agent | Go goroutines + context.Context are sufficient |
| Hermes's Postgres-backed cron | hermes-agent | SQLite-backed is correct for single-binary promise |

---

## References

- [Cross-Project Feature Map](../cross-project-feature-map/) — Detailed matrix of all 12 projects
- [Upstream Hermes Study](../../upstream-hermes/) — Full Hermes source analysis
- [Upstream Lessons](../upstream-lessons/) — Durable contracts to absorb
- [What Hermes Gets Wrong](../what-hermes-gets-wrong/) — Why Gormes exists
- [Contract Readiness](../contract-readiness/) — Row-level handoff fields
- [Gormes Completion Plan](../architecture_plan/completion-plan/) — Execution doctrine
- [Go Donor Reference Map](../architecture_plan/go-donor-reference-map/) — Go implementation patterns
- [GO-HERMES-PORTS-FORKS.md](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/docs/GO-HERMES-PORTS-FORKS.md) — All known Go Hermes ports
- [progress.json](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/webpages/docs/content/building-gormes/architecture_plan/progress.json) — Canonical execution queue

---

*Generated: April 30, 2026*
*Source: 12+ open-source projects, 7-phase gormes roadmap, cross-project feature map, upstream lessons*
