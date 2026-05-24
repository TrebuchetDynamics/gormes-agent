---
title: "Architecture Plan"
weight: 20
---

# Gormes — Executive Roadmap

**Single source of truth:** [`progress.json`](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/docs/content/building-gormes/architecture_plan/progress.json) — machine-readable, validated + regenerated on build.

**Public site:** https://gormes.ai

**Linked surfaces:**
- [README.md](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/README.md) — Quick start + rollup phase table
- [Landing page](https://gormes.ai) — Marketing + roadmap section
- [docs.gormes.ai](https://docs.gormes.ai/building-gormes/architecture_plan/) — This page
- [Source code](https://github.com/TrebuchetDynamics/gormes-agent) — Implementation

**Execution control plane:** repo-local Gormes skills consume this
`progress.json` and the generated `docs/content/building-gormes/` pages to
select and execute eligible work. The old loop command binaries are gone; the
roadmap is still the machine-readable queue for developing the full
`gormes-agent`.

**Completion doctrine:** [Gormes Completion Plan](./completion-plan/) defines
the non-negotiable finish line: Gormes is complete only when it is Hermes in Go,
with Goncho as the Honcho-compatible Go port inside Gormes.

**Operating model:** [Completion Lane Roadmap](./lane-roadmap/) maps phases to
finish lanes and gates; [Agent Operating Model](./agent-operating-model/) tells
agents how to run bounded parity, planner, builder, TDD, and interface-design
passes.

**Feature map:** [Hermes And Honcho Feature Map](./hermes-honcho-feature-map/)
maps upstream Hermes and Honcho feature families to Go packages, implementation
strategy, proof gates, and `progress.json` anchors.

**Contract pairings:** [Hermes/Gormes Contract Pairings](./hermes-gormes-contract-pairings/)
defines the shared vocabulary for pairing upstream Hermes symbols with their
Go-native Gormes adapters before rows are renamed or split.

**Messaging setup contract:** [Messaging Platform Setup Fidelity](./messaging-platform-setup-fidelity/)
records the Hermes-fidelity rules for `gormes setup gateway`, channel config,
Telegram-first setup, env compatibility, and migration boundaries.

**CLI parity matrix:** [Hermes Command Surface Parity Matrix](./hermes-command-surface-parity/)
records the operator-visible Hermes command tree, current Gormes state, and the
`progress.json` rows that own remaining command/auth gaps.

**Runtime plan:** [Hermes/Honcho To Gormes Go Runtime Plan](./hermes-honcho-go-runtime-plan/)
reconciles the feature map, source-class ledger, swarm audit, nested coverage
matrix, and progress rows into one implementation-ready subsystem plan.

**Completeness audit:** [Upstream Coverage Ledger](./upstream-coverage-ledger/)
lists the upstream source classes that must be represented in the feature map,
so a planner pass can tell whether Hermes/Honcho mapping is complete or has
drifted.

**Feature-level swarm audit:** [Swarm Feature Parity Audit](./swarm-feature-parity-audit/)
records the raw sub-agent parity findings that feed the runtime plan's
classification and row-backed implementation queue.

## How To Read This Roadmap

- The generated checklist below is rebuilt from `progress.json`; do not hand-edit
  content between `PROGRESS` markers.
- Start with the [Completion Plan](./completion-plan/) when deciding what to
  build next; then use the [Completion Lane Roadmap](./lane-roadmap/) for lane
  gates and the [Agent Operating Model](./agent-operating-model/) for the
  exact pass workflow.
- Use [Hermes And Honcho Feature Map](./hermes-honcho-feature-map/) when
  mapping upstream capabilities or deciding where a feature belongs in Go.
- Use [Hermes/Gormes Contract Pairings](./hermes-gormes-contract-pairings/)
  when a Gormes file name hides the upstream Hermes contract it adapts.
- Use [Hermes/Honcho To Gormes Go Runtime Plan](./hermes-honcho-go-runtime-plan/)
  when turning mapped upstream capabilities into Go package targets,
  classifications, dependency order, and builder-ready row splits.
- Use [Upstream Coverage Ledger](./upstream-coverage-ledger/) to verify that no
  feature-bearing Hermes/Honcho source class is unmapped.
- Use [Swarm Feature Parity Audit](./swarm-feature-parity-audit/) to find broad
  source classes that still hide missing or vague feature-level rows.
- Use the phase pages for design intent and boundaries, then use
  [Contract Readiness](../contract-readiness/) and [Agent Queue](../builder-loop/agent-queue/)
  for assignable work.
- When a row is too broad for one agent, split it in `progress.json` first and
  let [Umbrella Cleanup](../builder-loop/umbrella-cleanup/) show the remaining inventory.
- When a row is blocked, keep the unblock condition explicit so
  [Blocked Slices](../builder-loop/blocked-slices/) stays useful to operators and agents.

---

## Progress

<!-- PROGRESS:START kind=docs-full-checklist -->
**Overall:** 105/111 subphases shipped · 6 in progress · 0 planned

| Phase | Status | Shipped |
|-------|--------|---------|
| Phase 1 — The Dashboard | ✅ | 6/6 subphases |
| Phase 2 — The Gateway | ✅ | 22/22 subphases |
| Phase 3 — The Black Box (Memory) | ✅ | 16/16 subphases |
| Phase 4 — The Brain Transplant | ✅ | 13/13 subphases |
| Phase 5 — The Final Purge | 🔨 | 21/23 subphases |
| Phase 6 — The Learning Loop (Soul) | ✅ | 12/12 subphases |
| Phase 7 — Paused Channel Backlog | ✅ | 5/5 subphases |
| Phase 8 — Reputation & Publication | 🔨 | 4/7 subphases |
| Phase 9 — Design & Security Hardening | 🔨 | 6/7 subphases |

---

## Phase 1 — The Dashboard ✅

*Tactical bridge: Go TUI over Python's api_server HTTP+SSE boundary*

### 1.A — Core TUI ✅

- [x] `tui` Bubble Tea shell
- [x] `tui` 16ms coalescing mailbox
- [x] `gateway` SSE reconnect

### 1.B — Wire Doctor ✅

- [x] `doctor` Offline tool validation

### 1.C — Automation Reliability ✅

- [x] `fleet` Orchestrator failure-row stabilization for 4-8 workers
- [x] `fleet` Soft-success-nonzero bats coverage
- [x] `planner` Planner wrapper/test consistency closeout
- [x] `fleet` Autoloop row health and quarantine contract
- [x] `planner` Planner self-healing verdict loop
- [x] `planner` Planner divergence and provenance awareness
- [x] `fleet` Watchdog checkpoint coalescing
- [x] `fleet` PR-intake idle backoff
- [x] `fleet` Watchdog dead-process vs slow-progress separation
- [x] `progress` Builder-loop self-improvement vs user-feature ratio metric

### 1.D — Skill-Driven Control Plane ✅

- [x] `skills` Skill control-plane docs and Hugo navigation closeout
- [x] `skills` Skill-manager selection matrix hardening
- [x] `skills` Skill-pack coverage audit for Hermes-in-Go completion
- [x] `skills` Canonical development-skills directory and loader symlinks
- [x] `planner` External review feedback ingestion for planner rows

### 1.E — Gormes-owned: Unified Bubble Tea admin TUI ✅

- [x] `tui` Shared Bubble Tea wizard step chassis under internal/tui/wizard
- [x] `tui` Unified admin TUI shell with tab navigation
- [x] `tui` Admin TUI: Setup health screen with missing-config callouts
- [x] `tui` Admin TUI: Chat tab with keybinding to jump in from any screen
- [x] `tui` Admin TUI: Agents screen wired to the 2.H dynamic registry
- [x] `tui` Admin TUI: Commands catalog over the root CLI tree
- [x] `tui` Admin TUI: Safe command execution from the Commands tab
- [x] `tui` Admin TUI: Searchable Commands palette

### 5.X — Termux Runtime Compatibility ✅

- [x] `install` Gormes Termux Runtime Compatibility
- [x] `doctor` Termux runtime doctor check
- [x] `install` Termux install and release smoke guide
- [x] `install` Termux storage and path safety audit
- [x] `gateway` Termux gateway foreground tmux lifecycle
- [x] `install` Termux notification bridge via termux-api
- [x] `install` Termux real-device smoke evidence
- [x] `install` Termux remote execution guidance

## Phase 2 — The Gateway ✅

*Go-native operator wiring harness: tools, Telegram, shared gateway chassis, shipped cron, and the first OS-AI spine slices before focused channel closeout*

### 2.A — Tool Registry ✅

- [x] `gateway` In-process Go tool registry
- [x] `gateway` Streamed tool_calls accumulation
- [x] `gateway` Kernel tool loop
- [x] `doctor` Doctor verification
- [x] `gateway` Coding-agent delegation tooling (codex/claude-code/opencode)
- [x] `gateway` Coding-agent delegation: Phase 1 scaffold (internal/codingagents)

### 2.B.1 — Telegram Scout ✅

- [x] `channels` Telegram adapter
- [x] `channels` Long-poll ingress
- [x] `channels` Edit coalescing
- [x] `channels` Telegram important notification default

### 2.B.2 — Gateway Chassis + Discord ✅

- [x] `channels` Reusable gateway chassis
- [x] `channels` Telegram on shared chassis
- [x] `channels` gormes gateway multi-channel entrypoint
- [x] `channels` Discord

### 2.B.3 — Slack on Shared Chassis ✅

- [x] `channels` Slack Socket Mode adapter
- [x] `channels` Thread routing + coalesced reply flow
- [x] `channels` Slack CommandRegistry parser wiring
- [x] `channels` Slack gateway.Channel adapter shim
- [x] `channels` Slack config + cmd/gormes gateway registration
- [x] `channels` Slack env-token enabled-state preservation
- [x] `channels` Slack app manifest App Home and private-channel scopes

### 2.B.4 — WhatsApp Adapter ✅

- [x] `channels` Bridge-vs-native runtime decision
- [x] `channels` WhatsApp identity resolution + self-chat guard
- [x] `channels` Inbound normalization + command passthrough
- [x] `channels` Pairing, reconnect, and send contract
- [x] `channels` WhatsApp outbound pairing gate + raw peer mapping
- [x] `channels` WhatsApp reconnect backoff + send retry policy

### 2.B.5 — Session Context + Delivery Routing ✅

- [x] `gateway` Gateway session store + SessionSource parity
- [x] `gateway` Gateway manual reset session-boundary hooks
- [x] `gateway` Gateway session reset notification parity
- [x] `gateway` Gateway slash-confirm session-boundary cleanup
- [x] `gateway` SessionContext prompt injection
- [x] `gateway` Hermes live-turn prompt assembly parity (channel-neutral)
- [x] `gateway` Live-turn SOUL.md and project context wiring (channel-neutral)
- [x] `gateway` Live-turn USER.md and MEMORY.md durable user context block (channel-neutral)
- [x] `gateway` Live-turn timestamp + model/provider/session metadata block + self-help guidance (channel-neutral)
- [x] `builder` Hermes prompt-builder guidance constants port (data-only, byte-equivalent)
- [x] `gateway` Hermes MEMORY_GUIDANCE stale-artifact exclusion refresh
- [x] `gateway` Live-turn metadata production wiring (cmd/gormes -> Manager seams)
- [x] `channels` BlueBubbles iMessage session-context prompt guidance
- [x] `channels` Telegram production live-turn provider payload golden
- [x] `channels` Telegram /status Hermes-format closeout
- [x] `gateway` Gateway /title manual session title command
- [x] `gateway` Session metadata manual-title protection flag
- [x] `gateway` Gateway auto-title generation wiring
- [x] `channels` Telegram reply_to_mode and reply-context parity
- [x] `channels` Telegram sendChatAction typing API
- [x] `gateway` Gateway typing-action wiring during stream
- [x] `gateway` Placeholder edit-failure fallback hardening
- [x] `gateway` Gateway stream/tool trace formatting fixture matrix
- [x] `channels` Telegram dynamic BotCommand menu wiring
- [x] `profiles` Active Hermes/Sidon profile context root resolver for live turns
- [x] `gateway` Durable context ordering and frozen snapshot decision fixture
- [x] `gateway` Live-turn model/tool guidance wiring
- [x] `gateway` Gateway active-turn policy manifest closeout
- [x] `gateway` Gateway conversational session metadata refresh
- [x] `gateway` Gateway session token accounting parity
- [x] `gateway` Gateway startup allowlist + weak credential guard
- [x] `stt` Telegram voice/audio inbound attachment markers
- [x] `gateway` DeliveryRouter + --deliver target parsing
- [x] `gateway` Gateway stream consumer for agent-event fan-out
- [x] `gateway` Non-editable gateway progress/commentary send fallback
- [x] `channels` WhatsApp identifier safety predicate
- [x] `channels` WhatsApp unsafe sender/chat inbound evidence
- [x] `channels` WhatsApp unsafe alias endpoint inbound evidence
- [x] `gateway` Gateway fresh-final eligibility helper
- [x] `gateway` Gateway fresh-final send/delete fallback
- [x] `channels` Telegram fresh-final delete and config exposure
- [x] `channels` Telegram group bot-command mention gate helper
- [x] `channels` Telegram require-mention config fields
- [x] `channels` Telegram group require-mention bot binding
- [x] `channels` Slack rich-text quotes/lists + link-unfurl ingress
- [x] `channels` Slack thread-parent context + team-scoped cache key
- [x] `gateway` Gateway message deduplicator bounded helper
- [x] `gateway` Gateway inbound dedup key helper
- [x] `gateway` Gateway inbound dedup manager binding
- [x] `channels` Email outbound Date header contract
- [x] `channels` Telegram MarkdownV2 parse-mode rendering closeout
- [x] `channels` Telegram topic mode off/help/auth/debounce closeout
- [x] `channels` Telegram document/photo cache + batch attachment parity
- [x] `channels` Discord authenticated attachment download safety
- [x] `channels` Slack Block Kit approval buttons + action callback
- [x] `channels` Discord thread participation persistence
- [x] `gateway` Cross-platform image/document MEDIA delivery routing
- [x] `channels` Telegram inline approval buttons + callback auth
- [x] `channels` Telegram polling conflict + webhook secret startup guard
- [x] `channels` Slack mention/free-response gating + strict thread-memory guard
- [x] `channels` Discord interaction authorization + mention safety guards
- [x] `channels` Gateway processing lifecycle reactions for Telegram and Discord
- [x] `channels` Telegram text batching + caption merge parity
- [x] `gateway` Cross-platform multi-image native batching
- [x] `channels` Discord message admission + reply-mode policy
- [x] `channels` Webhook dynamic route reload + signed rate-limit order
- [x] `channels` Slack/Discord channel-scoped skills, prompts, and reload resync
- [x] `channels` Telegram fallback transport + polling reconnect recovery
- [x] `channels` Telegram sticker vision adapter binding
- [x] `channels` Discord native slash/thread command registration parity
- [x] `channels` Telegram entity-only mention boundary closeout
- [x] `channels` Telegram thread-aware outbound text + typing seam
- [x] `channels` Telegram forum thread fallback + send retry safety
- [x] `channels` Telegram DM topic reply-fallback routing
- [x] `channels` Telegram semantic MarkdownV2 formatter + table rewrite
- [x] `channels` Telegram Markdown table row-label bullet suppression
- [x] `channels` Telegram streaming edit Markdown safety
- [x] `channels` Telegram guest mention allowlist bypass
- [x] `gateway` Gateway platform reconnect isolation + channel health limits
- [x] `gateway` Gateway per-platform circuit breaker + /platform pause/resume/list command
- [x] `gateway` Gateway /model interactive provider/model picker
- [x] `gateway` Gateway memory monitor pressure policy

### 2.B.10 — WeChat Adapter ✅

- [x] `channels` WeCom + WeiXin shared-chassis bot seam
- [x] `channels` WeCom + WeiXin transport/bootstrap layer

### 2.B.11 — Discord Forum Channels ✅

- [x] `channels` Discord forum channel ingress + thread lifecycle
- [x] `channels` Discord SessionSource guild/parent/message evidence
- [x] `channels` Discord forum media + polish parity

### 2.B.12 — Channel-Neutral Native Runtime Adapter ✅

- [x] `gateway` Channel-neutral native runtime turn adapter
- [x] `gateway` Hermes gateway platform registry manifest
- [x] `channels` MSGraph webhook platform manifest drift closeout
- [x] `gateway` Bundled platform plugin manifest drift guard
- [x] `navivox` Navivox stdio protocol control-plane tracer
- [x] `navivox` Navivox QR pairing descriptor CLI
- [x] `navivox` Navivox Flutter voice morph surface
- [x] `gateway` Multimodal photo attachment passthrough
- [x] `gateway` Hermes-style default prompt and image-path hints for inbound photos
- [x] `channels` Hermes gateway platform strict-fidelity source-pair expansion

### 2.C — Thin Mapping Persistence ✅

- [x] `gateway` bbolt session resume
- [x] `gateway` (platform, chat_id) -> session_id

### 2.D — Cron / Scheduled Automations ✅

- [x] `fleet` robfig/cron scheduler + bbolt job store
- [x] `fleet` SQLite cron_runs audit + CRON.md mirror
- [x] `fleet` Heartbeat [SYSTEM:] + [SILENT] delivery contract
- [x] `planner` Architecture planner tasks manager script
- [x] `gateway` Cron no-agent script-only short-circuit
- [x] `fleet` Durable operator run report for unattended jobs
- [x] `fleet` Scheduled briefing job emits operator run report

### 2.E.0 — OS-AI Spine: Deterministic Subagent Runtime ✅

- [x] `gateway` Deterministic subagent runtime
- [x] `gateway` Max-depth guard + bounded batch execution
- [x] `gateway` Timeout + cancellation scopes
- [x] `gateway` Typed result envelope
- [x] `gateway` Append-only run log

### 2.E.1 — OS-AI Spine: Delegation Policy + Child Execution ✅

- [x] `gateway` Runner-enforced tool allowlists + blocked-tool policy
- [x] `gateway` Tool-call audit in typed child results
- [x] `gateway` Real child Hermes stream loop
- [x] `fleet` Durable job routing policy
- [x] `gateway` Durable subagent/job ledger

### 2.E.2 — OS-AI Spine: Concurrent-Tool Cancellation ✅

- [x] `gateway` Interrupt propagation to concurrent-tool workers

### 2.E.3 — OS-AI Spine: Durable Job Resilience ✅

- [x] `fleet` Durable job backpressure + timeout audit
- [x] `fleet` Durable worker supervisor status seam
- [x] `fleet` Durable pause/resume intent contract
- [x] `fleet` Durable replay and inbox message contract
- [x] `fleet` Durable worker execution loop
- [x] `fleet` Durable worker abort-slot recovery safety net
- [x] `fleet` Durable worker RSS watchdog policy helper
- [x] `fleet` Durable worker RSS drain integration
- [x] `doctor` Durable worker RSS doctor/status evidence

### 2.F.1 — Slash Command Registry + Gateway Dispatch ✅

- [x] `gateway` Canonical CommandDef registry
- [x] `gateway` Gateway slash dispatch + per-platform exposure
- [x] `gateway` Gateway slash registry parity sweep (recognized-name expansion)

### 2.F.2 — Hook Registry + BOOT.md ✅

- [x] `gateway` Gateway per-event hook registry
- [x] `gateway` Hook manifest discovery + handler loading
- [x] `gateway` Built-in BOOT.md startup hook

### 2.F.3 — Restart / Pairing / Status ✅

- [x] `gateway` Graceful restart drain + managed shutdown
- [x] `gateway` Adapter startup failure cleanup contract
- [x] `gateway` Gateway channel disconnect timeout on failed startup
- [x] `gateway` Gateway shutdown capped adapter disconnect
- [x] `gateway` Active-turn follow-up queue + late-arrival drain policy
- [x] `gateway` Drain-timeout resume_pending recovery
- [x] `gateway` Pairing read-model schema + atomic persistence
- [x] `gateway` Pairing approval + rate-limit semantics
- [x] `gateway` Unauthorized DM pairing response contract
- [x] `gateway` `gormes gateway status` read-only command
- [x] `gateway` Runtime status JSON + PID/process validation
- [x] `gateway` Token-scoped gateway locks
- [x] `gateway` Gateway /restart command + takeover markers
- [x] `gateway` Gateway restart notification opt-out
- [x] `gateway` Session expiry finalized-flag migration
- [x] `gateway` Session expiry hook cleanup retry evidence
- [x] `gateway` Channel lifecycle writers into status model

### 2.F.4 — Home Channel + Operator Surfaces ✅

- [x] `gateway` Home channel ownership resolver fixtures
- [x] `gateway` Notify-to delivery routing
- [x] `gateway` Channel directory atomic persistence + lookup
- [x] `gateway` Channel directory refresh + stale-target invalidation
- [x] `gateway` Manager remember-source hook
- [x] `gateway` Mirror + sticker cache surfaces
- [x] `gateway` Gateway delivery evidence in operator run report

### 2.F.5 — Gateway Mid-Run Steering + Active-Turn Policy ✅

- [x] `gateway` Steer slash command parser + preview helper
- [x] `gateway` Steer slash command registry + queue fallback
- [x] `gateway` Mid-run steer injection between tool calls
- [x] `gateway` Gateway-handled slash commands bypass active-session guard
- [x] `gateway` Gateway persistent goal loop + continuation judge

### 2.G — OS-AI Spine: Skills Runtime ✅

- [x] `gateway` SKILL.md parsing + active store
- [x] `gateway` Deterministic selection + prompt block
- [x] `gateway` Kernel injection + usage log
- [x] `gateway` Inactive candidate drafting
- [x] `gateway` Explicit promotion flow

### 2.H — Gormes-owned: Dynamic agents and per-thread spawn UX ✅

- [x] `goncho` Goncho-backed dynamic agent registry
- [x] `gateway` gormes agent spawn/list/inspect/bind/unbind CLI
- [x] `channels` Telegram /spawn opens forum topic bound to spawned agent
- [x] `channels` Discord /spawn opens thread bound to spawned agent

## Phase 3 — The Black Box (Memory) ✅

*SQLite + FTS5 + ontological graph + semantic fusion in Go; 3.E closes session visibility, audit trails, decay, and cross-chat/session boundaries*

### 3.A — SQLite + FTS5 Lattice ✅

- [x] `memory` SqliteStore
- [x] `memory` FTS5 triggers
- [x] `config` Schema migrations v3a->v3d

### 3.B — Ontological Graph + LLM Extractor ✅

- [x] `memory` Extractor
- [x] `memory` Entity/relationship upsert
- [x] `memory` Dead-letter queue

### 3.C — Neural Recall + Context Injection ✅

- [x] `memory` RecallProvider
- [x] `memory` 2-layer seed selection
- [x] `memory` CTE traversal
- [x] `memory` <memory-context> fence

### 3.D — Semantic Fusion + Local Embeddings ✅

- [x] `providers` Ollama embeddings
- [x] `memory` Vector cache
- [x] `memory` Cosine similarity recall
- [x] `memory` Hybrid fusion

### 3.D.5 — Memory Mirror (USER.md sync) ✅

- [x] `memory` Async background export
- [x] `memory` SQLite as source of truth

### 3.E.1 — Session Index Mirror ✅

- [x] `config` Read-only bbolt sessions.db -> index.yaml mirror
- [x] `sessions` Deterministic mirror refresh without mutating session state

### 3.E.2 — Tool Execution Audit Log ✅

- [x] `tools` Append-only JSONL writer + schema
- [x] `tools` Kernel + delegate_task audit hooks
- [x] `tools` Outcome, duration, and error capture

### 3.E.3 — Transcript Export Command ✅

- [x] `sessions` gormes session export <id> --format=markdown
- [x] `tools` Render turns, tool calls, and timestamps from SQLite

### 3.E.4 — Extraction State Visibility ✅

- [x] `memory` gormes memory status command
- [x] `memory` Extractor queue depth + dead-letter summary

### 3.E.5 — Insights Audit Log ✅

- [x] `memory` Append-only daily usage.jsonl writer
- [x] `sessions` Session, token, and cost rollups from local runtime

### 3.E.6 — Memory Decay ✅

- [x] `memory` relationships.last_seen schema + backfill
- [x] `memory` Relationship writer freshness updates
- [x] `memory` Deterministic weight attenuation at recall time

### 3.E.7 — Cross-Chat Synthesis ✅

- [x] `sessions` user_id concept above chat_id
- [x] `memory` Same-chat default recall fence
- [x] `memory` Opt-in user-scope recall + source filters
- [x] `memory` Interrupted-turn memory sync suppression
- [x] `goncho` Honcho-compatible scope/source tool schema
- [x] `goncho` Honcho host integration compatibility fixtures
- [x] `memory` SillyTavern persona and group-chat mapping fixtures
- [x] `memory` Cross-chat deny-path fixtures
- [x] `memory` Cross-chat operator evidence

### 3.E.8 — Session Lineage + Cross-Source Search ✅

- [x] `sessions` parent_session_id lineage for compression splits
- [x] `gateway` Gateway resume follows compression continuation
- [x] `sessions` Source-filtered session/message search core
- [x] `goncho` GONCHO user-scope search/context parameters
- [x] `sessions` Lineage-aware source-filtered search hits
- [x] `sessions` Operator-auditable search evidence

### 3.F — Goncho Honcho Memory Parity ✅

- [x] `goncho` Goncho context representation options
- [x] `goncho` Goncho search filter grammar
- [x] `goncho` Vector store + reconciler divergence proof
- [x] `goncho` Directional peer cards and representation scopes
- [x] `goncho` Goncho queue status read model
- [x] `goncho` Goncho summary context budget
- [x] `goncho` Goncho dialectic chat contract
- [x] `goncho` Goncho file upload import ingestion
- [x] `goncho` Goncho topology design fixtures
- [x] `goncho` Goncho operator diagnostics contract
- [x] `goncho` Goncho streaming chat persistence contract
- [x] `goncho` Goncho configuration namespace
- [x] `goncho` Goncho dreaming scheduler contract
- [x] `goncho` Goncho CRUD lifecycle invariants
- [x] `goncho` Goncho empty peer-card hint contract
- [x] `goncho` Hermes memory tool over Goncho/local durable store
- [x] `goncho` Goncho memory provider lifecycle adapter
- [x] `goncho` Goncho Memory V1 compatibility contract and migration harness
- [x] `goncho` GONCHO local-first markdown MCP memory requirement

### 3.G — Goncho Drop-In Compatibility Closure ✅

- [x] `goncho` Goncho keys + webhooks compatibility surface
- [x] `goncho` Goncho webhook delivery retry worker contract
- [x] `goncho` Goncho HTTP route parity over OpenAPI v3
- [x] `goncho` Goncho CLI command-tree parity
- [x] `goncho` Goncho Honcho SDK compatibility e2e harness
- [x] `goncho` Goncho memory integration into normal agent turn

### 3.H — Goncho Memory Quality & UX Improvements ✅

- [x] `goncho` Goncho session-end structured summary capture
- [x] `goncho` Goncho BM25 + RRF parallel retrieval fusion
- [x] `goncho` Goncho /memory and /continue CLI commands
- [x] `goncho` Goncho dream fact extraction and memory compression
- [x] `goncho` Goncho skill-outcome tracking as conclusions
- [x] `goncho` Goncho workspace isolation with explicit global scope

## Phase 4 — The Brain Transplant ✅

*Native Go agent orchestrator + prompt builder*

### 4.A — Provider Adapters ✅

- [x] `providers` Provider interface + stream fixture harness
- [x] `providers` Hermes provider registry and alias manifest
- [x] `providers` OpenRouter Pareto router request plugin
- [x] `providers` Tool-call normalization + continuation contract
- [x] `providers` DeepSeek/Kimi reasoning_content echo for tool-call replay
- [x] `providers` DeepSeek/Kimi cross-provider reasoning isolation
- [x] `providers` DeepSeek/Kimi all-assistant reasoning_content replay
- [x] `providers` Moonshot/Kimi tool-schema sanitizer
- [x] `providers` Anthropic
- [x] `providers` Azure OpenAI query/default_query transport contract
- [x] `providers` Azure Anthropic Messages endpoint contract
- [x] `providers` Azure Foundry transport probe read model
- [x] `providers` Azure Foundry probe — path sniffing
- [x] `providers` Azure Foundry probe — /models classification + Anthropic fallback
- [x] `providers` Azure Foundry runtime env/config read model
- [x] `providers` Azure Foundry CLI setup/status manual fallback
- [x] `providers` Azure Foundry Responses-only model-family API mode
- [x] `providers` Bedrock provider runtime binding
- [x] `providers` Bedrock Converse payload mapping (no AWS SDK)
- [x] `gateway` Bedrock stream event decoding (SSE fixtures)
- [x] `providers` Bedrock SigV4 + credential seam
- [x] `providers` Bedrock stale-client eviction + retry classification
- [x] `providers` Gemini Cloud Code request/stream mapper
- [x] `providers` OpenRouter compatible-provider routing
- [x] `providers` OpenRouter Grok prompt-cache affinity header
- [x] `providers` Google Code Assist project/quota resolver
- [x] `providers` Codex
- [x] `providers` Codex Responses pure conversion harness
- [x] `providers` Codex Responses assistant content role types
- [x] `providers` Codex Responses HTTP client binding
- [x] `providers` Codex OAuth state + stale-token relogin
- [x] `providers` Codex stream repair + tool-call leak sanitizer
- [x] `providers` Cross-provider reasoning-tag sanitization
- [x] `providers` Tool-call argument repair + schema sanitizer
- [x] `providers` OpenAI-compatible developer-role API-boundary swap
- [x] `providers` xAI Grok provider adapter
- [x] `providers` LM Studio provider adapter
- [x] `providers` Vision-unsupported provider retry (strip-images-and-resend)

### 4.B — Context Engine + Compression ✅

- [x] `sessions` Long session management
- [x] `sessions` Context compression
- [x] `tools` ContextEngine interface + status tool contract
- [x] `sessions` Compression token-budget trigger + summary sizing
- [x] `tools` Aux compression headroom for system and tool schemas
- [x] `providers` Aux compression provider-aware context cap
- [x] `tools` Tool-result pruning + protected head/tail summary
- [x] `sessions` Aux compression single-prompt threshold reconciliation
- [x] `sessions` Compression protected-tail multimodal length estimator
- [x] `sessions` Context compressor image-token budget charge
- [x] `sessions` Context references stable-handle store
- [x] `sessions` Manual compression feedback + context references
- [x] `sessions` Manual compression feedback renderer + focus parser
- [x] `sessions` ContextEngine compression-boundary callback vocabulary
- [x] `sessions` Kernel compression-boundary callback binding
- [x] `sessions` ContextEngine session-end hook on reset

### 4.C — Native Prompt Builder ✅

- [x] `builder` Default agent identity / SOUL.md loader
- [x] `builder` Context-file discovery + injection scan
- [x] `builder` Progressive subdirectory hint tracker
- [x] `builder` Model-specific role and tool-use guidance
- [x] `builder` Toolset-aware skills prompt snapshot
- [x] `builder` Memory guidance constant + injection
- [x] `builder` Session search guidance constant + injection
- [x] `builder` Gormes self-help skill/docs prompt guidance
- [x] `builder` [SYSTEM:→[IMPORTANT: meta-instruction prefix rename for Azure content filter compatibility
- [x] `builder` Native full prompt assembly
- [x] `builder` Ephemeral prefill messages file injection

### 4.D — Smart Model Routing ✅

- [x] `providers` Model metadata registry + context limits
- [x] `providers` Provider-enforced context-length resolver
- [x] `providers` Model pricing/capability registry fixtures
- [x] `providers` Ollama Cloud models.dev suffix normalization
- [x] `providers` Model catalog cache + preferred-provider live merge
- [x] `providers` Routing policy and fallback selector
- [x] `providers` Per-turn model selection
- [x] `providers` Per-turn reasoning effort propagation
- [x] `providers` Provider-default model resolution at config load
- [x] `providers` OpenAI Codex Spark catalog and context parity
- [x] `providers` Image input mode resolver + vision_analyze text fallback

### 4.E — Trajectory + Insights ✅

- [x] `providers` Trajectory writer + redaction gates
- [x] `gateway` Trajectory compressor + compressed-evidence lineage
- [x] `providers` Self-monitoring telemetry

### 4.F — Title Generation ✅

- [x] `sessions` Title prompt and truncation contract
- [x] `sessions` Title auxiliary failure visibility
- [x] `sessions` Auto-naming sessions

### 4.G — Credentials + OAuth ✅

- [x] `providers` Token vault
- [x] `providers` Anthropic OAuth/keychain credential discovery
- [x] `providers` Multi-account auth
- [x] `providers` Credential non-ASCII sanitizer + one-shot warning
- [x] `providers` Google OAuth flow + refresh seam
- [x] `providers` MiniMax OAuth provider registry and default auth routing
- [x] `providers` GitHub Copilot token exchange + Responses mode selector

### 4.H — Rate / Retry / Caching ✅

- [x] `providers` Provider-side resilience
- [x] `providers` Classified provider-error taxonomy
- [x] `providers` Generic provider timeout message classifier
- [x] `providers` Provider image-too-large error classification
- [x] `providers` Unsupported temperature retry + Codex no-temperature guard
- [x] `providers` Codex Responses temperature guard after flush removal
- [x] `providers` Generic unsupported-parameter retry + max_tokens guard
- [x] `providers` Jittered reconnect backoff schedule
- [x] `providers` Retry-After header parsing + HTTPError hint
- [x] `providers` Kernel retry honors Retry-After hint
- [x] `providers` Streaming interrupt retry suppression
- [x] `providers` Provider stream-drop retry diagnostics
- [x] `providers` Provider stream-drop timing and upstream diagnostics
- [x] `providers` Provider timeout config fail-closed helper
- [x] `providers` Prompt-cache capability guard
- [x] `providers` Provider account usage read model + renderer
- [x] `gateway` Gateway /usage command binding over provider account usage
- [x] `providers` Provider rate guard + budget telemetry
- [x] `providers` Provider rate guard — x-ratelimit header classification
- [x] `providers` Provider rate guard — degraded-state + last-known-good evidence
- [x] `providers` Hermes fast-mode request override serializer

### 4.I — Native Agent Turn Closure ✅

- [x] `runtime` Python-free normal agent turn e2e harness
- [x] `providers` Provider-tool-memory golden transcript suite
- [x] `planner` Hermes and Honcho feature parity map to Go implementation plan
- [x] `planner` Upstream source coverage ledger for Hermes/Honcho mapping completeness
- [x] `goncho` Swarm feature-level parity audit for Hermes/Honcho map
- [x] `stt` Hermes/Honcho Go runtime plan second-wave reconciliation
- [x] `runtime` Nested feature-level coverage test matrix for swarm gaps
- [x] `runtime` Hermes website docs mirror coverage gate
- [x] `providers` Gormes setup/channel/provider docs webpage parity gate
- [x] `gateway` Native runtime provider gateway binding
- [x] `runtime` Hermes compatibility namespace retirement boundary
- [x] `runtime` Hermes agent runtime strict-fidelity source-pair expansion

### 4.J — Permission-Hardened Tool Execution ✅

- [x] `tools` Shell blocklist + filesystem scoping + permission approval

### 4.K — Provider Fallback Chain ✅

- [x] `providers` Resilient provider chain dispatch
- [x] `providers` Hermes fallback activation + classifier carve-outs
- [x] `providers` Fallback entry api_key_env credential alias

### 4.L — Safety-Anchored Turn Loop (MOSAIC) ✅

- [x] `runtime` Plan gate hook in agent turn loop
- [x] `tools` Tool gate pre-execution validation
- [x] `runtime` Refusal-as-action in ReAct cycle
- [x] `runtime` Safety loop end-to-end integration

### 4.M — Advanced Provider Routing ✅

- [x] `providers` Circuit breaker per provider and API key
- [x] `providers` P95 latency-aware failover
- [x] `providers` Capability-based model tier routing

## Phase 5 — The Final Purge 🔨

*Python tool scripts ported to Go or WASM*

### 5.A — Tool Surface Port ✅

- [x] `tools` 61-tool registry port
- [x] `tools` Tool registry inventory + schema parity harness
- [x] `tools` Tool-call JSON-string array/object coercion parity
- [x] `tools` Tool parity manifest refresh for Hermes b35d692f
- [x] `tools` Tool parity manifest refresh for Hermes ea86714 computer_use
- [x] `tools` Tool parity manifest refresh for Hermes 524cbabd patch schema
- [x] `tools` Microsoft Graph auth/client helper parity
- [x] `channels` Discord tool split + platform-scoped toolsets
- [x] `channels` Discord tool limit coercion helper
- [x] `tools` Home Assistant HASS_TOKEN platform-toolset carveout
- [x] `tools` Home Assistant tool handlers + service safety validation
- [x] `tools` Pure core tools first
- [x] `config` Stateful tool migration queue
- [x] `tools` Terminal process watch notification throttle contract
- [x] `tools` Tool output budget persisted artifact pointer
- [x] `tools` Tool descriptor layer (OperationSpec)
- [x] `tools` Hermes tool tail strict-fidelity source-pair expansion

### 5.B — Sandboxing Backends ✅

- [x] `tools` Environment interface + file sync contract
- [x] `tools` Terminal snapshot source stdout suppression guard
- [x] `tools` Terminal deleted-cwd recovery guard
- [x] `tools` Raw tool-call parser fixture matrix
- [x] `install` Docker execution backend (container lifecycle + mount policy)
- [x] `install` Docker backend top-level container reuse semantics
- [x] `tools` Modal
- [x] `tools` Daytona
- [x] `tools` Singularity command/preflight contract
- [x] `tools` Sandbox Policy Explain

### 5.C — Browser Automation 🔨

- [x] `browser` Browser action contract + event transcript
- [x] `browser` go-browser-harness Chromedp action backend
- [x] `browser` Rod
- [x] `browser` Browser provider bridge + Firecrawl fallback
- [x] `browser` Camofox REST browser mode and managed identity bridge
- [x] `browser` Browser Use cloud + Go browser harness bridge
- [x] `browser` Go browser-harness Hermes browser_* tool wrappers
- [x] `browser` Go-native Hermes web_search/web_extract tool wrappers
- [x] `browser` Go-native Hermes web backend matrix and config resolver
- [x] `browser` Go-native Hermes web extract safety policy and summarizer
- [x] `browser` Goscrapling local extraction for web_extract
- [x] `browser` Go-native Hermes web_crawl tool adapter
- [x] `browser` Go-native Hermes web managed gateway status and live smoke closure
- [x] `browser` Brave Search + DDGS web search provider parity
- [x] `browser` Browser artifact and console render contract
- [x] `browser` Browser console expression CDP result shaping
- [x] `browser` Telegram browser artifact rendering
- [x] `browser` Browser hybrid private-URL local sidecar routing
- [x] `browser` Browser SSRF quoted-false guard
- [x] `browser` Go browser harness binary repo + integration lane (placeholder)
- [x] `browser` Browser session inactivity cleanup thread
- [x] `browser` Goscrapling browser-backed extraction gate for web_extract
- [ ] `browser` Goscrapling local crawler adapter gate for web_crawl

### 5.D — Vision + Image Generation ✅

- [x] `tools` Multimodal in/out
- [x] `tools` Image input mode router + native content parts
- [x] `tools` vision_analyze native multimodal tool-result path
- [x] `tools` Image-too-large shrink retry helper
- [x] `tools` Image generation result contract
- [x] `providers` Image generation provider registry + plugin dispatch
- [x] `tools` FAL image generation queue REST binding
- [x] `tools` Native video_analyze tool contract

### 5.E — TTS / Voice / Transcription 🔨

- [x] `tts` Voice mode port
- [x] `tts` Voice mode environment detector + audio provider seam
- [x] `tts` Transcription tool contract
- [x] `tts` Telegram voice/audio STT ingress hook
- [x] `tts` TTS tool contract + media delivery seam
- [x] `tts` MiniMax TTS v1 text_to_speech raw-audio compatibility
- [x] `tts` TTS provider matrix + dotenv/command-provider resolution
- [x] `tts` TTS synthesis + voice-mode state
- [x] `tts` Voice record-key config binding for native TUI
- [x] `tts` Telegram voice STT HTTP-provider fallback
- [x] `tts` Pure-Go STT exploration
- [x] `tts` wazero WASI smoke harness
- [x] `tts` whisper.cpp WASI module discovery
- [x] `tts` Pure-Go Whisper transcribe one WAV
- [x] `tts` Whisper tiny.en model cache fetcher
- [x] `tts` Wire Pure-Go Whisper into Telegram resolver
- [x] `tts` WASI Whisper ffmpeg preprocess + fixed-window chunker
- [x] `tts` Audio preprocessing and chunking pipeline
- [x] `tts` Whisper benchmark harness + perf budget
- [x] `tts` Go-native OGG/Opus decoder decision
- [x] `tts` Go-native OGG/Opus decoder implementation
- [x] `tts` Pure-Go TTS decision research
- [x] `tts` Shared speech artifact cache for Go-owned TTS
- [ ] `tts` Go-owned WASM TTS backend

### 5.F — Skills System (Remaining) ✅

- [x] `providers` Skills hub search result types + in-memory registry provider
- [x] `providers` Skills hub search read-model function over registry providers
- [x] `skills` Skill registries
- [x] `skills` Skills hub direct URL candidate parser
- [x] `skills` Skills hub direct URL install name/category guard
- [x] `skills` Skill preprocessing + dynamic slash commands
- [x] `skills` [IMPORTANT:] prompt prefix for cron and skill commands
- [x] `skills` Skills list — enabled/disabled status column + --enabled-only filter
- [x] `profiles` Update bundled skills across active and named profiles
- [x] `skills` Bundled Airtable productivity skill contract
- [x] `skills` Bundled TouchDesigner MCP skill catalog contract

### 5.G — MCP Integration ✅

- [x] `tools` MCP client
- [x] `goncho` Goncho MCP tool catalog
- [x] `config` MCP server config/env resolver
- [x] `tools` MCP stdio transport + tool/list discovery
- [x] `tools` MCP HTTP transport + tool/list discovery
- [x] `tools` MCP schema normalization + structured-content adapter
- [x] `providers` MCP OAuth state store + noninteractive auth errors
- [x] `providers` MCP OAuth refresh + 401 session-expired recovery
- [x] `gateway` Managed tool gateway bridge
- [x] `tools` MCP circuit breaker cooldown + reconnect reset
- [x] `tools` MCP stdio orphan cleanup after cron ticks
- [x] `tools` Gormes-native MCP host runtime boundary
- [x] `channels` MCP channels_list tool

### 5.H — ACP Integration ✅

- [x] `tools` ACP server side
- [x] `tools` ACP Client Bridge Mode
- [x] `tools` ACP JSON-RPC stdio session/prompt closeout
- [x] `tools` ACP stdio benign ping/probe suppression
- [x] `tools` ACP session CWD propagation into prompt runners
- [x] `tools` ACP setup-browser bootstrap parity

### 5.I — Plugins Architecture ✅

- [x] `skills` Plugin SDK
- [x] `skills` Dashboard theme/plugin extension status contract
- [x] `skills` Dashboard page-scoped plugin slot inventory
- [x] `skills` Third-party extensions
- [x] `skills` Hermes plugin CLI lifecycle parity
- [x] `skills` Teams pipeline plugin CLI metadata + disabled runtime inventory
- [x] `goncho` Goncho Honcho plugin session config + async write compatibility
- [x] `skills` First-party Spotify plugin fixture
- [x] `skills` First-party Google Meet plugin metadata fixture
- [x] `skills` Hindsight memory setup blank-input preservation
- [x] `skills` Agent Hooks Registry
- [x] `doctor` Plugin Marketplace + Doctor
- [x] `skills` Extension Lifecycle Hook System
- [x] `skills` Plugin lifecycle hook: transform_llm_output
- [x] `skills` Hermes plugin catalog strict-fidelity classifier

### 5.J — Approval / Security Guards ✅

- [x] `tools` Dangerous action gating
- [x] `gateway` Gateway approval FIFO queue resolver
- [x] `tools` Hardline command pattern table + DetectHardline function
- [x] `tools` Recoverable dangerous patterns + blocked-result schema
- [x] `config` Approval mode config normalization
- [x] `gateway` Gateway hook auto-accept strict parser
- [x] `tools` delegate_task batch JSON-string task recovery
- [x] `tools` Subagent dangerous-command non-interactive approval policy
- [x] `tools` Concurrent tool approval callback propagation
- [x] `tools` Background review toolset restriction
- [x] `tools` Cron dangerous-command approval mode
- [x] `config` Cron approval mode config normalizer
- [x] `tools` Tirith external security finding ingestion
- [x] `tools` Unified security guard decision composer
- [x] `tools` Shell blocklist (36+ dangerous patterns)
- [x] `tools` Filesystem scoping (folder-level read/write restrictions)
- [x] `tools` Permission approval UX (inline y/n/always)
- [x] `tools` Trust-class enforcement in shared tool executor
- [x] `runtime` Secrets Runtime Controls
- [x] `tools` Security Audit Command
- [x] `channels` Email allowlist pre-dispatch loop guard
- [x] `tools` Auth state TOCTOU close + redaction default-on parity
- [x] `gateway` Gateway allowed_chats/channels/rooms whitelist parity

### 5.K — Code Execution ✅

- [x] `tools` Sandboxed exec

### 5.L — File Ops + Patches ✅

- [x] `tools` Atomic file write helper with temp+rename pattern
- [x] `tools` File tool atomic checkpoint integration
- [x] `tools` Checkpoints CLI (status/list/prune/clear/clear-legacy)
- [x] `tools` Checkpoint shadow-repo GC policy
- [x] `tools` File read dedup cache invalidation and wrapper guard
- [x] `tools` File read repeated-stub BLOCKED escalation
- [x] `tools` Native file task tool surface
- [x] `tools` V4A patch mode for native patch tool
- [x] `tools` V4A move operation for native patch tool
- [x] `tools` Symlink-preserving atomic writer helper
- [x] `tools` File write/patch staleness registry + cwd tracking
- [x] `config` Terminal cwd config bridge
- [x] `tools` Terminal deleted-cwd recovery
- [x] `tools` search_files hidden-root and context-line parsing drift
- [x] `tools` Structured lint delta for native write/patch tools
- [x] `tools` Python syntax lint delta for native write/patch tools
- [x] `tools` Shell lint delta for native write/patch tools
- [x] `tools` Patch replace no-match did-you-mean hint
- [x] `tools` Core fuzzy replace strategies for native patch tool
- [x] `tools` Unicode-normalized fuzzy replace for native patch tool
- [x] `tools` Block-anchor fuzzy replace for native patch tool
- [x] `tools` V4A fuzzy hunk matching for native patch tool
- [x] `tools` Context-aware fuzzy replace for native patch tool
- [x] `tools` V4A patch apply rollback for native patch tool
- [x] `tools` Patch replace post-write verification
- [x] `tools` Hermes LSP write-time semantic diagnostics

### 5.M — Mixture of Agents ✅

- [x] `providers` Multi-model coordination
- [x] `kanban` Hermes Kanban durable board core
- [x] `kanban` Hermes Kanban dispatcher and worker spawn loop
- [x] `kanban` Hermes Kanban production worker process binding
- [x] `kanban` Hermes Kanban worker tools and prompt gating
- [x] `kanban` Kanban orchestrator board-routing tools
- [x] `kanban` Kanban comment author hardening and cross-task handoff policy
- [x] `kanban` Hermes Kanban slash/gateway/dashboard surfaces
- [x] `kanban` Native TUI /kanban slash command binding over gormes kanban
- [x] `kanban` Gateway /kanban shared command-runner binding
- [x] `kanban` Kanban slash help and usage-error UX
- [x] `kanban` Kanban dashboard dispatch quick path
- [x] `kanban` Kanban dashboard task run history endpoint
- [x] `kanban` Kanban dispatcher status in gateway /status
- [x] `kanban` Kanban multi-board isolation
- [x] `kanban` Kanban workspace context injection
- [x] `kanban` Kanban run history persistence
- [x] `kanban` Kanban notification delivery parity
- [x] `kanban` Kanban chat board DB pin
- [x] `kanban` Kanban schema migration duplicate-column race guard
- [x] `kanban` Kanban notify subscription store and CLI
- [x] `kanban` Kanban notify delivery engine blocked retention
- [x] `kanban` Kanban stats command and board summary
- [x] `kanban` Kanban corrupt timestamp age hardening
- [x] `kanban` Kanban named-board workspace and log roots
- [x] `kanban` Kanban current-board task command routing
- [x] `kanban` Kanban task run history command
- [x] `kanban` Kanban boards list/show task-count read model
- [x] `kanban` Kanban global --board task command override
- [x] `kanban` Kanban GC terminal event and worker-log retention
- [x] `kanban` Kanban worker log read command
- [x] `kanban` Kanban task event tail command
- [x] `kanban` Kanban worker heartbeat, reclaim, and zombie detection
- [x] `kanban` Hermes Kanban specify triage parity

### 5.N — Misc Operator Tools ✅

- [x] `tools` Todo
- [x] `tools` Clarify
- [x] `tools` Session search tool schema and argument validation
- [x] `tools` Session search tool execution wrapper
- [x] `tools` Session shutdown memory transcript handoff
- [x] `tools` Debug helpers
- [x] `tools` Debug share paste sweep scheduler contract
- [x] `doctor` Doctor GitHub CLI auth fallback
- [x] `planner` Planner audit blank-subphase control-plane bucket
- [x] `fleet` Autoloop recent-failure detail excerpts
- [x] `tools` Backend usage-limit stdin health bypass
- [x] `tools` Cronjob tool API + schedule parser parity
- [x] `tools` Cron schedule parser + repeat state fixtures
- [x] `tools` Cron recurring next-run failure preservation
- [x] `tools` Cron prompt/script safety + pre-run script contract
- [x] `tools` Cron GitHub auth-header scanner parity
- [x] `tools` Cronjob tool action envelope over native store
- [x] `tools` Cron run resource release contract
- [x] `tools` Cron run resource release executor binding
- [x] `tools` Cron context_from output chaining
- [x] `tools` Cron prompt/script safety + pre-run script contract (deprecated umbrella)
- [x] `tools` Cron multi-target delivery + media/live-adapter fallback
- [x] `tools` Cron deliver=all routing intent expansion
- [x] `skills` Plugin standalone sender cron delivery fallback
- [x] `goncho` Goncho serialized write queue + relation candidates
- [x] `tools` Blocker Policy Integration
- [x] `tools` OpenClaw SecretRef core resolver
- [x] `config` Cross-agent config isolation
- [x] `tools` SecretRef runtime snapshot activation
- [x] `tools` OpenClaw security audit --deep --fix
- [x] `doctor` ACP bridge doctor/status evidence
- [x] `gateway` Gateway probe auth/capability HTTP closeout
- [x] `tools` Safety-critical panic and swallowed-error closeout
- [x] `tools` Session Health Monitoring
- [x] `tools` Evidence-Before-Claims Quality Gate
- [x] `tools` Git Delivery Contract Enforcement
- [x] `tools` QMD Hybrid Search
- [x] `tools` Session Rollover Automation
- [x] `fleet` System Events, Heartbeat, and Presence
- [x] `gateway` Gateway Discover and Probe
- [x] `tools` Channels Capabilities Introspection
- [x] `channels` Teams configured-state in channel capabilities
- [x] `tools` Prompt Fragment Include System
- [x] `gateway` Multi-agent gateway runtime activation
- [x] `tools` Multi-agent auth and tool-policy runtime isolation
- [x] `channels` Per-agent channel bot tokens (Telegram/Discord/Slack)
- [x] `tools` Cron env-ref expansion + parallel run state serialization
- [x] `tools` Cron origin delivery isolation from session identity
- [x] `tools` Cron script/workdir/inactivity execution binding
- [x] `fleet` Cron no-agent script-only watchdog mode
- [x] `providers` Cron partial legacy job read-model normalization
- [x] `tools` Cron dashboard partial-record page
- [x] `navivox` Navivox host setup apply with transient sudo
- [x] `gateway` Gateway auto-resume on restart
- [x] `tools` Hermes x_search tool and auth surface
- [x] `goncho` Goncho durable recall trace IR + fused ranking pipeline
- [x] `goncho` Goncho recall diagnostics CLI over RecallTrace
- [x] `goncho` Goncho replayable retrieval traces
- [x] `goncho` Goncho proof matrix and fixture harness
- [x] `runtime` Morning degraded-status summary over latest run report
- [x] `providers` Provider/auth readiness preflight for unattended jobs
- [x] `goncho` Goncho golden transcript e2e harness
- [x] `goncho` Goncho retrieval benchmark corpus

### 5.O — Hermes CLI Parity ✅

- [x] `cli` 49-file CLI tree port
- [x] `cli` Hermes CLI command-tree parity manifest
- [x] `cli` Hermes CLI nested parser inventory refresh
- [x] `cli` Hermes auth command-tree manifest refresh
- [x] `providers` Hermes auth credential-pool command surface
- [x] `providers` Hermes auth OAuth provider adapters
- [x] `providers` Hermes auth Spotify service-provider subcommand
- [x] `channels` Deterministic helper-file ports (banner/output/tips/webhook/dump)
- [x] `cli` CLI banner/output formatting helpers
- [x] `cli` CLI deterministic tip selector
- [x] `cli` CLI OpenClaw residue detection and hint text
- [x] `config` CLI onboarding seen-state map helpers
- [x] `config` CLI contextual first-touch onboarding hint renderers
- [x] `cli` CLI bracketed-paste wrapper sanitizer
- [x] `cli` CLI slow bracketed-paste diagnostic threshold
- [x] `tools` CLI terminal control-response sanitizer
- [x] `cli` CLI submitted user-message preview formatter
- [x] `channels` CLI webhook URL normalizer
- [x] `cli` CLI dump support-summary helper
- [x] `cli` PTY bridge protocol adapter
- [x] `cli` CLI command registry parity + active-turn busy policy
- [x] `gateway` Gateway /reasoning command parser
- [x] `gateway` Gateway /reasoning apply + dispatch
- [x] `sessions` Busy command guard for compression and long CLI actions
- [x] `profiles` Config, profile, auth, and setup command surfaces
- [x] `cli` Gormes agent template reset command
- [x] `cli` Hermes py2many parity mapping report
- [x] `cli` Hermes source-pair manifest and Phase 0 refresh mode
- [x] `providers` Gormes auth bare interactive credential-pool readout
- [x] `providers` Gormes auth status per-provider aggregator
- [x] `providers` Gormes auth add openai-codex strict isolation contract
- [x] `planner` Gormes auth add bedrock open-question planning note
- [x] `profiles` Gormes profile command binding
- [x] `profiles` Gormes profile distribution metadata readout
- [x] `profiles` Gormes profile create clone-all infrastructure exclusion
- [x] `profiles` Model and profile selector seam (Cobra + gateway)
- [x] `providers` Gormes top-level logout provider shortcut
- [x] `providers` Top-level logout configured-provider fallback
- [x] `cli` Gormes removed top-level login guidance
- [x] `providers` Gormes model interactive provider/model picker
- [x] `config` Gormes setup minimal sectioned wizard slice
- [x] `config` Gormes setup top-level chooser menu
- [x] `config` Gormes setup full-wizard shell and branded summary
- [x] `providers` Gormes setup model step uses the dynamic provider-tracked model picker
- [x] `config` Hermes setup entry-mode and reset semantics
- [x] `config` Gormes setup tools checklist command binding
- [x] `gateway` Gormes setup gateway platform checklist command binding
- [x] `tui` Bubble Tea Messaging Platforms setup: Telegram-first Hermes fidelity
- [x] `tts` Gormes setup terminal TTS and agent-settings section bindings
- [x] `install` Gormes uninstall dry-run command contract
- [x] `tools` Gormes mcp login interface seam + noninteractive default
- [x] `browser` Gormes mcp login browser callback flow
- [x] `providers` Hermes fallback provider chain CLI commands
- [x] `providers` Provider endpoint/API-key root flags + runtime resolution
- [x] `profiles` Gormes profile skills chat invocation shim
- [x] `channels` Hermes config.yaml Telegram compatibility bridge
- [x] `config` Gormes config command surface
- [x] `config` Gormes config set comment-preserving TOML writes
- [x] `config` Gormes config edit/check/native schema-migrate closeout
- [x] `config` Hermes config migration dry-run manifest
- [x] `config` Hermes config migration writer
- [x] `config` OpenClaw migration dry-run manifest
- [x] `config` OpenClaw migration writer and cleanup command
- [x] `profiles` CLI profile name validator
- [x] `profiles` CLI profile root resolver
- [x] `profiles` CLI active-profile store
- [x] `profiles` CLI profile path and active-profile store (deprecated umbrella)
- [x] `providers` Scripted chat query model/provider resolver
- [x] `cli` Oneshot final-output writer boundary
- [x] `tools` Oneshot noninteractive safety and clarify policy
- [x] `config` Platform toolset config persistence + MCP sentinel
- [x] `tools` Platform toolset mixed composite runtime expansion
- [x] `skills` Effective toolset picker dedupes bundled plugin keys
- [x] `channels` Gateway, platform, webhook, and cron management CLI
- [x] `channels` WhatsApp top-level pairing wizard shell
- [x] `channels` WhatsApp live Baileys QR pairing wizard
- [x] `gateway` Gateway management CLI read-model closeout
- [x] `gateway` Gateway mutating-subcommand unavailability stub
- [x] `gateway` Windows gateway Scheduled Task lifecycle commands
- [x] `gateway` Windows detached gateway Ctrl+C boundary
- [x] `cli` Service RestartSec parser helper
- [x] `cli` Service restart active-status poller
- [x] `cli` Diagnostics, backup, logs, and status CLI
- [x] `sessions` Hermes sessions CLI MRU browse/delete ergonomics
- [x] `cli` Backup/update opt-in and exclusion policy
- [x] `cli` Self-update command lifecycle safety
- [x] `planner` Gormes update release planner and dry-run contract
- [x] `install` Gormes update verified binary swap and rollback
- [x] `gateway` Gormes update bundled assets and skills sync
- [x] `cli` Gormes update managed service drain and restart
- [x] `doctor` doctorCustomEndpointReadiness check function
- [x] `doctor` gormes doctor actionable issues summary and --fix auto-remediation
- [x] `doctor` gormes doctor ◆ Section grouping + upstream section ordering (UX parity)
- [x] `doctor` gormes doctor section-content parity (Security Advisories / Directory Structure / Skills Hub / Auth Providers / Profiles)
- [x] `doctor` gormes doctor ◆ Directory Structure section content
- [x] `doctor` gormes doctor ◆ Skills Hub section content
- [x] `doctor` gormes doctor ◆ Auth Providers section content
- [x] `doctor` gormes doctor ◆ Profiles section content
- [x] `doctor` gormes doctor ◆ Security Advisories section content
- [x] `config` gormes setup <section> boxed header + completion footer (UX parity)
- [x] `profiles` Profile Control Center v2 umbrella — single root config and active services
- [x] `profiles` gormes setup profiles — section scaffold + per-profile workspace list
- [x] `profiles` gormes setup profiles — per-profile channels (telegram/whatsapp/discord/slack)
- [x] `navivox` Navivox multi-server profile routing config model
- [x] `providers` Custom provider model-switch credential preservation
- [x] `providers` Custom provider model-switch key_env write guard
- [x] `cli` CLI log redactor for known secret shapes
- [x] `cli` CLI log snapshot reader using shared redactor
- [x] `providers` Hermes config.yaml model/provider runtime bridge
- [x] `config` Interactive Onboarding
- [x] `config` Internal onboarding interactive action runner
- [x] `config` CLI setup/onboard/help text fidelity matrix
- [x] `cli` Hermes CLI alias and suggestion fidelity matrix
- [x] `cli` Logs Command
- [x] `gateway` Gateway planned stop marker + WSL systemd PATH parity
- [x] `gateway` Gateway stale-code self-check uses git HEAD SHA
- [x] `runtime` Agent lifecycle hooks (agent:start, agent:step, agent:end)
- [x] `providers` Nous OAuth device code + refresh token + agent key provisioning
- [x] `cli` Hermes send command stdin/file payload parity
- [x] `sessions` Hermes session recap command surface
- [x] `profiles` Profile workspace allow-list enforcement policy
- [x] `profiles` Profile-local subprocess HOME parity
- [x] `profiles` Long-term plan: profile fleet supervisor and single control-plane gateway
- [x] `cli` CLI module contract registry and manifest gate
- [x] `cli` cmd/gormes profile command package extraction
- [x] `cli` cmd/gormes setup section registry extraction
- [x] `providers` cmd/gormes provider usage command package extraction
- [x] `providers` cmd/gormes provider command surface package extraction
- [x] `gateway` cmd/gormes gateway row-backed command package extraction
- [x] `channels` cmd/gormes channels capabilities command package extraction
- [x] `gateway` cmd/gormes live gateway command package extraction
- [x] `channels` cmd/gormes channel service command package extraction
- [x] `cli` cmd/gormes root command assembly extraction
- [x] `config` Root config.toml v2 profile service schema
- [x] `config` Legacy profile config v2 migration planner
- [x] `profiles` Profile Control Center read model
- [x] `profiles` Profile Control Center TUI shell and draft apply flow
- [x] `providers` Per-profile provider credential readiness
- [x] `channels` Per-profile channel credential readiness and allow-lists
- [x] `providers` Gormes setup providers plural alias

### 5.P — Docker / Packaging ✅

- [x] `install` OCI image
- [x] `install` Homebrew
- [x] `install` Nix flake package and NixOS module contract
- [x] `install` Unix installer (install.sh) source-backed update flow
- [x] `install` Unix installer root/FHS layout policy
- [x] `install` Windows installer (install.ps1 + install.cmd) parity
- [x] `install` Installer script serving and MIME validation
- [x] `install` Install isolation: GORMES_BIN_DIR is an authoritative sandbox boundary
- [x] `install` Install isolation: skip shell-rc PATH write when bin dir is under /tmp
- [x] `install` Install isolation: skip system service install when sandbox bin dir is set
- [x] `install` Install: prefer pre-built release binary over source build by default
- [x] `install` Install: Termux publishes a real $PREFIX/bin binary, not an $HOME-targeting symlink
- [x] `install` Termux exec argv path-alias sanitizer
- [x] `install` Termux binary-fetch publish verification source fallback

### 5.Q — API Server + TUI Gateway Streaming ✅

- [x] `profiles` Deterministic helper-file ports (tool-progress/image/completion-path/personality/platform-event)
- [x] `gateway` TUI gateway tool-progress mode normalizer
- [x] `gateway` TUI gateway completion path normalizer
- [x] `gateway` TUI gateway tool summary formatter
- [x] `profiles` TUI gateway image/personality/platform-event helpers
- [x] `gateway` TUI gateway config health null-section probe
- [x] `tui` TUI mouse tracking config + slash toggle
- [x] `tui` Native TUI bundle independence check
- [x] `gateway` TUI launch model override + static alias resolver
- [x] `gateway` TUI prompt-submit auto-title eligibility helper
- [x] `gateway` TUI TerminalNativeSelectionHelp constant + help-string fixture
- [x] `tui` Native TUI slash-command dispatch table
- [x] `tui` Native TUI /save canonical session export
- [x] `tui` Native TUI /save XDG export helper
- [x] `tui` Native TUI /save local runtime binding
- [x] `tui` Native TUI /branch session fork + transcript target switch
- [x] `tui` Native TUI /branch local runtime resident-session binding
- [x] `tui` Native TUI resident session-switch replay helper
- [x] `gateway` TUI running-agent placeholder surfaces interrupt + queued slash actions
- [x] `tui` Native TUI conversation viewport tail helper
- [x] `tui` Native TUI queued-message edit helper
- [x] `tui` Native TUI renderConv viewport budget binding
- [x] `tui` Native TUI Hermes skin token renderer
- [x] `tui` Native TUI Hermes status bar renderer
- [x] `tui` Native TUI Hermes bottom-pinned chrome layout
- [x] `tui` Native TUI Hermes input keybinding semantics
- [x] `tui` Native TUI Shift+Enter newline CSI-u parity
- [x] `tui` Native TUI clipboard, OSC52, and terminal setup parity
- [x] `tui` Native TUI image/file drop + paste collapse ingress
- [x] `tui` Native TUI Hermes slash completion helpers
- [x] `tui` Native TUI absolute path completion routing
- [x] `tui` Native TUI Hermes slash dispatch behavioral matrix
- [x] `tui` Native TUI /quit local exit binding
- [x] `tui` Native TUI Hermes tool progress + modal panel renderers
- [x] `tui` Native TUI Ink behavioral transcript golden matrix
- [x] `tui` Native TUI markdown soft-wrap boundary trim
- [x] `gateway` Channel/TUI iteration-limit finalization transcript fixture
- [x] `tui` SSE streaming to Bubble Tea TUI
- [x] `gateway` TUI websocket attach transport
- [x] `gateway` OpenAI-compatible chat-completions API server
- [x] `gateway` API server multimodal content preservation
- [x] `gateway` Responses API store + run event stream
- [x] `gateway` API server disconnect snapshot persistence
- [x] `gateway` Gateway proxy mode forwarding contract
- [x] `gateway` Gateway proxy replay assistant metadata preservation
- [x] `gateway` Dashboard API client contract
- [x] `gateway` Dashboard PTY chat sidecar contract
- [x] `gateway` API server detailed health snapshot contract
- [x] `gateway` API server detailed health endpoint
- [x] `gateway` API server cron admin read-only endpoints
- [x] `gateway` API server cron admin mutating endpoints
- [x] `gateway` API server legacy jobs routes + default toolset
- [x] `gateway` Provider client lazy-init for TUI cold-start budget
- [x] `tui` Native TUI /model slash command binding over the existing model picker
- [x] `tui` Kernel in-session model-switch seam for the native TUI
- [x] `gateway` Kernel cross-provider client swap for in-session model switch
- [x] `tui` Native TUI slash handler-port coverage
- [x] `tui` Native TUI shipped slash command registry availability metadata
- [x] `tui` Native TUI Terminal.app truecolor and ANSI sanitizer parity
- [x] `tui` Hermes ui-tui strict-fidelity action matrix
- [x] `gateway` Hermes web dashboard strict-fidelity contract map
- [x] `tui` Native TUI /help slash command binding
- [x] `tui` Native TUI /redraw local repaint binding
- [x] `tui` Native TUI /statusbar chrome mode binding
- [x] `tui` Native TUI /details detail-section visibility binding
- [x] `tui` Native TUI /indicator busy-indicator style binding
- [x] `tui` Native TUI /history current transcript page binding
- [x] `tui` Native TUI /status current frame page binding
- [x] `tui` Native TUI /logs gateway tail page binding
- [x] `tui` Native TUI /title session-title binding
- [x] `tui` Native TUI /sessions and /resume picker page binding
- [x] `tui` Native TUI /resume session switch binding
- [x] `tui` Native TUI /usage local frame usage page binding
- [x] `tui` Native TUI /usage provider account usage adapter binding
- [x] `tui` Native TUI /clear and /new reset-session binding
- [x] `tui` Native TUI /compact transcript toggle binding
- [x] `tui` Native TUI /skills read-only hub binding
- [x] `tui` Native TUI /tools enable-disable binding
- [x] `tui` Native TUI /voice status and toggle binding
- [x] `tui` Native TUI /skin get-set binding

### 5.R — Code Execution Mode Policy ✅

- [x] `config` Execution-mode resolver + config precedence
- [x] `tools` Strict-mode CWD + interpreter parity
- [x] `tools` Project-mode CWD + active venv detection
- [x] `config` Default mode selection + config cut-over

### 5.S — Loop Detection ✅

- [x] `runtime` 5-type loop detector

### 5.T — Browser Harness Doctor ✅

- [x] `browser` go-browser-harness doctor subcommand

### 5.U — Fault-Tolerant Sandbox Execution ✅

- [x] `tools` Pre-execution command classification
- [x] `tools` Transactional tool execution with snapshot/rollback
- [x] `tools` Sandbox isolation depth selection

### 5.V — Unified Event Bus ✅

- [x] `runtime` Event bus core: pub/sub interface + in-process implementation
- [x] `channels` Gateway channel adapters publish to event bus
- [x] `gateway` Gateway outbound sends publish message-sent events
- [x] `channels` Weixin gateway event-bus adapter
- [x] `channels` WeCom gateway event-bus adapter
- [x] `channels` Telegram gateway event-bus adapter
- [x] `channels` Discord gateway event-bus adapter
- [x] `channels` Slack gateway event-bus adapter
- [x] `channels` WhatsApp gateway event-bus adapter
- [x] `tools` Agent turn and tool execution events on bus
- [x] `gateway` Event bus integration test: full message flow

### 5.W — i18n Internationalization ✅

- [x] `runtime` Hermes i18n static-message port
- [x] `runtime` Hermes i18n expanded locale catalog parity

## Phase 6 — The Learning Loop (Soul) ✅

*Hermes-compatible background review and skill curation, plus Gormes-native evidence gates for safe compounding intelligence.*

### 6.A — Complexity Detector ✅

- [x] `learning-loop` Hermes background review fork lifecycle
- [x] `learning-loop` Deterministic learning-loop trigger signals

### 6.B — Skill Extractor ✅

- [x] `skills` LLM-assisted pattern distillation

### 6.C — Skill Storage Format ✅

- [x] `skills` SKILL.md frontmatter validation guard
- [x] `skills` Hermes creative skill metadata compatibility
- [x] `skills` Portable SKILL.md format
- [x] `skills` Hermes v0.14 optional skill catalog refresh
- [x] `skills` Hermes skill catalog strict-fidelity classifier

### 6.D — Skill Retrieval + Matching ✅

- [x] `skills` Hybrid lexical + semantic lookup
- [x] `skills` Source-aware retrieval damping fixtures
- [x] `gateway` Delta-bounded skill and memory maintenance passes
- [x] `skills` Code Cathedral II code-context retrieval fixtures

### 6.E — Feedback Loop ✅

- [x] `providers` Hermes curator auxiliary model routing slot
- [x] `skills` Hermes curator state transitions and run reports
- [x] `skills` Hermes curator rename summary notice
- [x] `learning-loop` Hermes review prompt transient-environment guard
- [x] `skills` Skill effectiveness scoring

### 6.F — Skill Surface ✅

- [x] `skills` Hermes skill_manage support-file and curator intent actions
- [x] `skills` Hermes curator command surface
- [x] `skills` Hermes curator archive/list/prune CLI catch-up
- [x] `channels` TUI + Telegram browsing
- [x] `skills` Native skills list/view tool surface

### 6.G — Structured Memory Types ✅

- [x] `memory` 6 typed memory categories with confidence scoring

### 6.H — Skill Metadata Placement ✅

- [x] `skills` SKILL.md metadata.when/loaded/placement schema

### 6.I — Zero-LLM Knowledge Graph ✅

- [x] `memory` Regex-based auto-link extraction + brain-first lookup

### 6.J — Agentic Memory Lifecycle (AgeMem) ✅

- [x] `tools` Memory operations as agent-callable tools
- [x] `memory` Agent-controlled memory retention with importance scoring
- [x] `sessions` Cross-session memory continuity

### 6.K — Self-Evolution Engine (GEPA) ✅

- [x] `learning-loop` Prompt evaluation harness
- [x] `learning-loop` Iterative prompt mutation and scoring loop
- [x] `sessions` Behavioral pattern extraction from session logs

### 6.L — Composable Skill Execution (Voyager) ✅

- [x] `skills` Skill code execution runtime
- [x] `skills` Skill dependency resolution and composition
- [x] `profiles` Agent personalities + enhanced display config
- [x] `stt` Session auto-reset + STT config parity

## Phase 7 — Paused Channel Backlog ✅

*Deferred non-priority channel adapters after Telegram, Discord, Slack, WhatsApp, and WeChat stabilize*

### 7.A — Signal Adapter ✅

- [x] `channels` Inbound event normalization + session identity
- [x] `channels` Reply/send contract on shared chassis
- [x] `channels` Signal transport/bootstrap layer
- [x] `channels` Signal markdown bodyRanges + attachment rate scheduler

### 7.B — Email + SMS Adapters ✅

- [x] `channels` Email ingress + outbound delivery contract
- [x] `channels` SMS ingress + outbound delivery contract

### 7.C — Matrix + Mattermost Adapters ✅

- [x] `channels` Threaded text adapter contract suite
- [x] `channels` Matrix shared-chassis bot seam
- [x] `channels` Matrix self/bridge sender drop helper
- [x] `channels` Mattermost shared-chassis bot seam
- [x] `channels` Matrix real client/bootstrap layer
- [x] `channels` Matrix E2EE device-id crypto-store binding
- [x] `channels` Mattermost REST/WS bootstrap layer

### 7.D — Webhook + Trigger Ingress ✅

- [x] `channels` Signed event parsing + auth gates
- [x] `channels` Prompt-to-delivery routing bridge

### 7.E — Regional + Device Adapter Backlog ✅

- [x] `channels` BlueBubbles + HomeAssistant adapters
- [x] `channels` BlueBubbles iMessage bubble formatting parity
- [x] `channels` Feishu shared-chassis bot seam
- [x] `channels` DingTalk shared-chassis bot seam
- [x] `channels` QQ Bot shared-chassis bot seam
- [x] `channels` Feishu transport/bootstrap layer
- [x] `channels` Feishu native update prompt cards
- [x] `channels` Feishu drive-comment rule + pairing seam
- [x] `channels` Feishu drive-comment reply workflow
- [x] `channels` DingTalk transport/bootstrap layer
- [x] `channels` DingTalk real SDK binding
- [x] `channels` DingTalk AI Cards streaming-update contract
- [x] `channels` DingTalk emoji reaction send/receive parity
- [x] `channels` DingTalk media (image/file) attachment routing
- [x] `channels` Yuanbao protocol envelope + markdown fixtures
- [x] `channels` Yuanbao media/sticker attachment normalization
- [x] `gateway` Yuanbao gateway runtime + toolset registration
- [x] `channels` Microsoft Teams adapter plugin seam
- [x] `channels` QQ Bot transport/bootstrap layer
- [x] `channels` Google Chat shared-chassis platform adapter seam
- [x] `channels` Google Chat relay sender-type self-filter
- [x] `channels` Google Chat standalone cron sender
- [x] `channels` Google Chat install dependency hint refresh
- [x] `channels` SimpleX Chat platform plugin parity

## Phase 8 — Reputation & Publication 🔨

*TrebuchetDynamics has a credible public face (blog, writeups, talks) that documents Gormes's autonomous-porting methodology and one or two sharp differentiators. Reputation is built through publication cadence, not parity scope.*

### 8.A — Publication Infrastructure 🔨

- [x] `docs` TD engineering blog scaffolded and live
- [ ] `docs` TD social presence connected to blog feed

### 8.B — Repository Messaging ✅

- [x] `docs` README rewrite to methodology-first positioning
- [x] `docs` README release and benchmark metadata sync
- [x] `landing` gormes.ai landing page positioning audit
- [x] `docs` Gormes market comparison positioning brief
- [x] `docs` Public comparison matrix: Gormes vs Hermes, OpenClaw, hosted agents
- [x] `channels` Channel capability matrix with stable/fixture/planned labels
- [x] `learning-loop` Learning-loop proof demo for skills, memory, and curator
- [x] `install` No-stack first-run proof path from install to offline doctor
- [x] `docs` Canonical config.toml v2 profile schema docs

### 8.C — Engineering Writeups 🔨

- [ ] `docs` Engineering writeup #1: autonomous Hermes-porting loop
- [x] `docs` Engineering writeup #1 cost telemetry evidence packet
- [x] `docs` Engineering writeup #1 local publication review packet
- [x] `docs` Hermes v0.14 release feature-to-module pairing ledger
- [x] `docs` Hermes contract inventory gate
- [x] `docs` Strict-fidelity upstream test-suite classifier

### 8.D — Sharp v1.0 ✅

- [x] `release` Sharp v1.0 differentiator decision
- [x] `release` Single-binary cross-platform release pipeline
- [x] `release` Release binary version/provenance smoke guard
- [x] `install` CI and installer Go toolchain floor sync
- [x] `release` Release prep guide target matrix sync
- [x] `install` Windows install.ps1 release binary fetch selector
- [x] `install` OCI image PR build and arm64 smoke workflow
- [x] `release` Release build-date provenance injection
- [x] `landing` Landing release metadata date-alias sync
- [x] `release` GitHub release title date-alias binding
- [x] `release` Release notes artifact size table
- [x] `release` Release SBOM attestation binding
- [x] `release` Release build provenance attest action contract
- [x] `release` Release notes SBOM attestation wording
- [x] `release` Release archive 30 MB size gate
- [x] `install` Termux android/arm64 release artifact and installer selector
- [x] `tui` Gormes-owned chat TUI divergence ratification
- [x] `tui` Gormes-owned session-aware welcome panel
- [x] `tui` Gormes-owned semantic chat style system
- [x] `tui` Gormes-owned streaming feedback uplift
- [x] `tui` Gormes streaming tool-trail status + spinner cadence wiring
- [x] `tui` Gormes welcome panel version/tool-count wiring
- [x] `release` Termux latest-installer follow-up release publication
- [x] `release` Removal of public v0.2.20 Termux latest-install caveats from README, landing, install docs, and troubleshooting docs

### 8.E — Toolkit Extraction 🔨

- [x] `docs` Agentic-porting-kit extraction spec
- [x] `docs` Agentic-porting-kit local standalone fixture
- [x] `docs` Agentic-porting-kit local porting skill skeletons
- [x] `docs` Agentic-porting-kit local README and license fixture
- [x] `docs` Agentic-porting-kit local public layout assembly gate
- [ ] `docs` Agentic-porting-kit public repo scaffold

### 8.F — Cost Discipline & Loop Economics ✅

- [x] `progress` Loop $/iteration cost metric in status file
- [x] `landing` Stop git-tracking duplicate landing progress mirrors (build-time generate)
- [x] `progress` Compact completed-row shipped-evidence notes to a one-line pointer
- [x] `progress` Module-split the progress backlog (per-subsystem files, parity-aligned)
- [x] `progress` Backlog split C1: lossless multi-file loader/writer behind the single-file API
- [x] `landing` Backlog split C2: docs/landing generators read the split layout
- [x] `progress` Backlog split C3: migrate remaining backlog consumers and the write path to the split layout
- [x] `progress` Backlog split C5a: optional per-row module key + deterministic derivation + backfill
- [x] `progress` Backlog split C4: AGENTS.md + gormes-* skills source-order updated to the split layout
- [x] `progress` Backlog split C5b: module-keyed split layout behind the existing API
- [x] `progress` Backlog split C5c: migrate webpages/docs raw progress.json readers to internal/progress.Load
- [x] `progress` Backlog split C5d: migrate gormes-* skill discovery commands off raw jq of the canonical progress.json
- [x] `progress` Backlog split C5e: make non-Go raw progress.json consumers (fleet scripts + CI path globs) split-directory-safe
- [x] `progress` Backlog split C5f: replace coarse module buckets with the approved feature taxonomy
- [x] `progress` Backlog split C5g: explicitly classify every row into a valid feature module
- [x] `progress` Backlog split C5h: add module-scoped progress commands for planner and builder selection
- [x] `progress` Backlog split C5i: render per-module roadmap pages before the physical split
- [x] `progress` Backlog split C5: single atomic operator-gated flip to the module-keyed split directory
- [x] `progress` OpenCode part-cost telemetry adapter for builder loop
- [x] `progress` Progress next-work read-only selector
- [x] `progress` Progress next-work repo-scope filter
- [x] `progress` Internal topology guard for package consolidation
- [x] `cli` Internal CLI surface package rehome
- [x] `tools` Internal tool compact helper package rehome
- [x] `tools` Internal tool trace helper package rehome
- [x] `tools` Internal session search tool package rehome

### 8.G — Community & External Contributions ✅

- [x] `docs` Built-with-Gormes page scaffold
- [x] `docs` Upstream Hermes user-stories static mirror

## Phase 9 — Design & Security Hardening 🔨

*Owned architecture improvements from DeerFlow patterns: declarative middleware chain for the agent runtime, and sandbox provider abstraction with virtual path security layer.*

### 9.A — Declarative Agent Middleware Chain ✅

- [x] `runtime` Agent middleware chain framework

### 9.B — Sandbox Provider Abstraction + Virtual Path System ✅

- [x] `providers` Sandbox provider interface and virtual path security

### 9.C — Hermes Config Parity — Personalities & Display ✅

- [x] `profiles` Agent personalities + enhanced display config

### 9.D — Speech-to-Text Tool Wiring ✅

- [x] `stt` Transcribe audio tool registration + local whisper provider

### 9.E — Navivox HTTP-only Hardening ✅

- [x] `navivox` Remove SSH Navivox stdio path
- [x] `navivox` Remove Flutter Navivox fake-server mode and wire protocol
- [x] `navivox` Remove Flutter SSH keys feature
- [x] `navivox` Navivox VPN host enumeration helper
- [x] `navivox` Navivox HTTP gateway mandatory-VPN bind
- [x] `navivox` Navivox HTTP gateway connect command
- [x] `navivox` Navivox setup QR image pairing handoff

### 9.F — Navivox Operator Activation 🔨

- [x] `navivox` Navivox HTTP/WS documentation refresh
- [x] `navivox` Navivox connect-and-talk first screen
- [x] `navivox` Navivox profile contact summary API
- [x] `navivox` Navivox continuous voice command mode
- [ ] `navivox` Navivox Telegram-inspired chat polish
- [x] `navivox` Navivox natural-language profile seed backend API
- [ ] `navivox` Navivox natural-language profile seed Flutter UI
- [x] `navivox` Navivox structured tool event cards backend API
- [ ] `navivox` Navivox structured tool event cards Flutter UI
- [x] `navivox` Navivox safe config admin backend API
- [ ] `navivox` Navivox safe config admin Flutter UI
- [x] `navivox` Navivox voice run records backend API
- [ ] `navivox` Navivox voice run records Flutter inspection UI
- [x] `navivox` Navivox per-profile BYO voice profiles backend API
- [ ] `navivox` Navivox per-profile BYO voice profiles Flutter UI

### 9.G — External Issue Radar Regression Guards ✅

- [x] `runtime` PicoClaw-derived channel media and identity regression matrix
- [x] `providers` PicoClaw-derived session ledger read-model regression matrix
- [x] `providers` PicoClaw-derived provider stream and auth regression matrix
- [x] `install` PicoClaw-derived tool path safety regression pack
- [x] `tools` MCP Streamable HTTP session lifecycle compatibility
- [x] `runtime` Dynamic agent identity inheritance regression matrix

<!-- PROGRESS:END -->

---

## Phase 3 Deep Dive

`3.E.7` and `3.E.8` now have a frozen architecture target in `docs/superpowers/plans/2026-04-22-gormes-phase3-identity-lineage-plan.md`. The contract is `user_id > chat_id > session_id`, recall remains same-chat default, cross-chat recall is opt-in, and `parent_session_id` is reserved for compression/fork descendants instead of becoming a generic session rewrite mechanism.

Execution is now sequenced in `docs/superpowers/plans/2026-04-22-gormes-phase3-identity-lineage-execution-plan.md`, with the closeout order fixed as `3.E.6.1 -> 3.E.7.2 -> 3.E.8.1 -> 3.E.8.2` so freshness, fence safety, lineage metadata, and search/observability land in that order.

---

## Phase 4 Entry Gate

Before any Phase 4 coding starts, the [Pre-Phase-4 E2E Gate](./phase-3-memory/) must be green. Freeze the Hermes-backed hybrid baseline for delivery envelopes, `<memory-context>` fences, and transcript/export artifacts first, then follow the entry rule in [Phase 4 — The Brain Transplant](./phase-4-brain-transplant/).

---

## Data Format

[`progress.json`](https://github.com/TrebuchetDynamics/gormes-agent/blob/main/docs/content/building-gormes/architecture_plan/progress.json) is the machine-readable source of truth. Top-level structure:

- `meta` — schema version, last-updated timestamp, canonical URLs
- `phases` — six phases keyed `"1"`..`"6"`, each containing `subphases`
- each subphase carries either `items` (the normal case) or an explicit `status`

Stats (complete/in-progress/planned counts) are **not stored** — they are computed on render. Updated automatically on `make build`.
