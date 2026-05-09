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
**Overall:** 84/100 subphases shipped · 12 in progress · 4 planned

| Phase | Status | Shipped |
|-------|--------|---------|
| Phase 1 — The Dashboard | ✅ | 4/4 subphases |
| Phase 2 — The Gateway | ✅ | 21/21 subphases |
| Phase 3 — The Black Box (Memory) | ✅ | 15/15 subphases |
| Phase 4 — The Brain Transplant | ✅ | 13/13 subphases |
| Phase 5 — The Final Purge | 🔨 | 17/23 subphases |
| Phase 6 — The Learning Loop (Soul) | 🔨 | 8/12 subphases |
| Phase 7 — Paused Channel Backlog | 🔨 | 4/5 subphases |
| Phase 8 — Reputation & Publication | 🔨 | 2/7 subphases |

---

## Phase 1 — The Dashboard ✅

*Tactical bridge: Go TUI over Python's api_server HTTP+SSE boundary*

### 1.A — Core TUI ✅

- [x] Bubble Tea shell
- [x] 16ms coalescing mailbox
- [x] SSE reconnect

### 1.B — Wire Doctor ✅

- [x] Offline tool validation

### 1.C — Automation Reliability ✅

- [x] Orchestrator failure-row stabilization for 4-8 workers
- [x] Soft-success-nonzero bats coverage
- [x] Planner wrapper/test consistency closeout
- [x] Autoloop row health and quarantine contract
- [x] Planner self-healing verdict loop
- [x] Planner divergence and provenance awareness
- [x] Watchdog checkpoint coalescing
- [x] PR-intake idle backoff
- [x] Watchdog dead-process vs slow-progress separation
- [x] Builder-loop self-improvement vs user-feature ratio metric

### 1.D — Skill-Driven Control Plane ✅

- [x] Skill control-plane docs and Hugo navigation closeout
- [x] Skill-manager selection matrix hardening
- [x] Skill-pack coverage audit for Hermes-in-Go completion
- [x] Canonical development-skills directory and loader symlinks

## Phase 2 — The Gateway ✅

*Go-native operator wiring harness: tools, Telegram, shared gateway chassis, shipped cron, and the first OS-AI spine slices before focused channel closeout*

### 2.A — Tool Registry ✅

- [x] In-process Go tool registry
- [x] Streamed tool_calls accumulation
- [x] Kernel tool loop
- [x] Doctor verification

### 2.B.1 — Telegram Scout ✅

- [x] Telegram adapter
- [x] Long-poll ingress
- [x] Edit coalescing

### 2.B.2 — Gateway Chassis + Discord ✅

- [x] Reusable gateway chassis
- [x] Telegram on shared chassis
- [x] gormes gateway multi-channel entrypoint
- [x] Discord

### 2.B.3 — Slack on Shared Chassis ✅

- [x] Slack Socket Mode adapter
- [x] Thread routing + coalesced reply flow
- [x] Slack CommandRegistry parser wiring
- [x] Slack gateway.Channel adapter shim
- [x] Slack config + cmd/gormes gateway registration
- [x] Slack env-token enabled-state preservation
- [x] Slack app manifest App Home and private-channel scopes

### 2.B.4 — WhatsApp Adapter ✅

- [x] Bridge-vs-native runtime decision
- [x] WhatsApp identity resolution + self-chat guard
- [x] Inbound normalization + command passthrough
- [x] Pairing, reconnect, and send contract
- [x] WhatsApp outbound pairing gate + raw peer mapping
- [x] WhatsApp reconnect backoff + send retry policy

### 2.B.5 — Session Context + Delivery Routing ✅

- [x] Gateway session store + SessionSource parity
- [x] Gateway manual reset session-boundary hooks
- [x] Gateway session reset notification parity
- [x] SessionContext prompt injection
- [x] Hermes live-turn prompt assembly parity (channel-neutral)
- [x] Live-turn SOUL.md and project context wiring (channel-neutral)
- [x] Live-turn USER.md and MEMORY.md durable user context block (channel-neutral)
- [x] Live-turn timestamp + model/provider/session metadata block + self-help guidance (channel-neutral)
- [x] Hermes prompt-builder guidance constants port (data-only, byte-equivalent)
- [x] Live-turn metadata production wiring (cmd/gormes -> Manager seams)
- [x] BlueBubbles iMessage session-context prompt guidance
- [x] Telegram production live-turn provider payload golden
- [x] Telegram /status Hermes-format closeout
- [x] Gateway /title manual session title command
- [x] Session metadata manual-title protection flag
- [x] Gateway auto-title generation wiring
- [x] Telegram reply_to_mode and reply-context parity
- [x] Telegram sendChatAction typing API
- [x] Gateway typing-action wiring during stream
- [x] Placeholder edit-failure fallback hardening
- [x] Gateway stream/tool trace formatting fixture matrix
- [x] Telegram dynamic BotCommand menu wiring
- [x] Active Hermes/Sidon profile context root resolver for live turns
- [x] Durable context ordering and frozen snapshot decision fixture
- [x] Live-turn model/tool guidance wiring
- [x] Gateway active-turn policy manifest closeout
- [x] Gateway conversational session metadata refresh
- [x] Gateway session token accounting parity
- [x] Gateway startup allowlist + weak credential guard
- [x] Telegram voice/audio inbound attachment markers
- [x] DeliveryRouter + --deliver target parsing
- [x] Gateway stream consumer for agent-event fan-out
- [x] Non-editable gateway progress/commentary send fallback
- [x] WhatsApp identifier safety predicate
- [x] WhatsApp unsafe sender/chat inbound evidence
- [x] WhatsApp unsafe alias endpoint inbound evidence
- [x] Gateway fresh-final eligibility helper
- [x] Gateway fresh-final send/delete fallback
- [x] Telegram fresh-final delete and config exposure
- [x] Telegram group bot-command mention gate helper
- [x] Telegram require-mention config fields
- [x] Telegram group require-mention bot binding
- [x] Slack rich-text quotes/lists + link-unfurl ingress
- [x] Slack thread-parent context + team-scoped cache key
- [x] Gateway message deduplicator bounded helper
- [x] Gateway inbound dedup key helper
- [x] Gateway inbound dedup manager binding
- [x] Email outbound Date header contract
- [x] Telegram MarkdownV2 parse-mode rendering closeout
- [x] Telegram topic mode off/help/auth/debounce closeout
- [x] Telegram document/photo cache + batch attachment parity
- [x] Discord authenticated attachment download safety
- [x] Slack Block Kit approval buttons + action callback
- [x] Discord thread participation persistence
- [x] Cross-platform image/document MEDIA delivery routing
- [x] Telegram inline approval buttons + callback auth
- [x] Telegram polling conflict + webhook secret startup guard
- [x] Slack mention/free-response gating + strict thread-memory guard
- [x] Discord interaction authorization + mention safety guards
- [x] Gateway processing lifecycle reactions for Telegram and Discord
- [x] Telegram text batching + caption merge parity
- [x] Cross-platform multi-image native batching
- [x] Discord message admission + reply-mode policy
- [x] Webhook dynamic route reload + signed rate-limit order
- [x] Slack/Discord channel-scoped skills, prompts, and reload resync
- [x] Telegram fallback transport + polling reconnect recovery
- [x] Telegram sticker vision adapter binding
- [x] Discord native slash/thread command registration parity
- [x] Telegram entity-only mention boundary closeout
- [x] Telegram thread-aware outbound text + typing seam
- [x] Telegram forum thread fallback + send retry safety
- [x] Telegram DM topic reply-fallback routing
- [x] Telegram semantic MarkdownV2 formatter + table rewrite
- [x] Gateway platform reconnect isolation + channel health limits

### 2.B.10 — WeChat Adapter ✅

- [x] WeCom + WeiXin shared-chassis bot seam
- [x] WeCom + WeiXin transport/bootstrap layer

### 2.B.11 — Discord Forum Channels ✅

- [x] Discord forum channel ingress + thread lifecycle
- [x] Discord SessionSource guild/parent/message evidence
- [x] Discord forum media + polish parity

### 2.B.12 — Channel-Neutral Native Runtime Adapter ✅

- [x] Channel-neutral native runtime turn adapter
- [x] Hermes gateway platform registry manifest
- [x] MSGraph webhook platform manifest drift closeout
- [x] Navivox stdio protocol control-plane tracer
- [x] Navivox QR pairing descriptor CLI
- [x] Navivox Flutter voice morph surface

### 2.C — Thin Mapping Persistence ✅

- [x] bbolt session resume
- [x] (platform, chat_id) -> session_id

### 2.D — Cron / Scheduled Automations ✅

- [x] robfig/cron scheduler + bbolt job store
- [x] SQLite cron_runs audit + CRON.md mirror
- [x] Heartbeat [SYSTEM:] + [SILENT] delivery contract
- [x] Architecture planner tasks manager script
- [x] Cron no-agent script-only short-circuit

### 2.E.0 — OS-AI Spine: Deterministic Subagent Runtime ✅

- [x] Deterministic subagent runtime
- [x] Max-depth guard + bounded batch execution
- [x] Timeout + cancellation scopes
- [x] Typed result envelope
- [x] Append-only run log

### 2.E.1 — OS-AI Spine: Delegation Policy + Child Execution ✅

- [x] Runner-enforced tool allowlists + blocked-tool policy
- [x] Tool-call audit in typed child results
- [x] Real child Hermes stream loop
- [x] Durable job routing policy
- [x] Durable subagent/job ledger

### 2.E.2 — OS-AI Spine: Concurrent-Tool Cancellation ✅

- [x] Interrupt propagation to concurrent-tool workers

### 2.E.3 — OS-AI Spine: Durable Job Resilience ✅

- [x] Durable job backpressure + timeout audit
- [x] Durable worker supervisor status seam
- [x] Durable pause/resume intent contract
- [x] Durable replay and inbox message contract
- [x] Durable worker execution loop
- [x] Durable worker abort-slot recovery safety net
- [x] Durable worker RSS watchdog policy helper
- [x] Durable worker RSS drain integration
- [x] Durable worker RSS doctor/status evidence

### 2.F.1 — Slash Command Registry + Gateway Dispatch ✅

- [x] Canonical CommandDef registry
- [x] Gateway slash dispatch + per-platform exposure
- [x] Gateway slash registry parity sweep (recognized-name expansion)

### 2.F.2 — Hook Registry + BOOT.md ✅

- [x] Gateway per-event hook registry
- [x] Hook manifest discovery + handler loading
- [x] Built-in BOOT.md startup hook

### 2.F.3 — Restart / Pairing / Status ✅

- [x] Graceful restart drain + managed shutdown
- [x] Adapter startup failure cleanup contract
- [x] Gateway channel disconnect timeout on failed startup
- [x] Gateway shutdown capped adapter disconnect
- [x] Active-turn follow-up queue + late-arrival drain policy
- [x] Drain-timeout resume_pending recovery
- [x] Pairing read-model schema + atomic persistence
- [x] Pairing approval + rate-limit semantics
- [x] Unauthorized DM pairing response contract
- [x] `gormes gateway status` read-only command
- [x] Runtime status JSON + PID/process validation
- [x] Token-scoped gateway locks
- [x] Gateway /restart command + takeover markers
- [x] Session expiry finalized-flag migration
- [x] Session expiry hook cleanup retry evidence
- [x] Channel lifecycle writers into status model

### 2.F.4 — Home Channel + Operator Surfaces ✅

- [x] Home channel ownership resolver fixtures
- [x] Notify-to delivery routing
- [x] Channel directory atomic persistence + lookup
- [x] Channel directory refresh + stale-target invalidation
- [x] Manager remember-source hook
- [x] Mirror + sticker cache surfaces

### 2.F.5 — Gateway Mid-Run Steering + Active-Turn Policy ✅

- [x] Steer slash command parser + preview helper
- [x] Steer slash command registry + queue fallback
- [x] Mid-run steer injection between tool calls
- [x] Gateway-handled slash commands bypass active-session guard
- [x] Gateway persistent goal loop + continuation judge

### 2.G — OS-AI Spine: Skills Runtime ✅

- [x] SKILL.md parsing + active store
- [x] Deterministic selection + prompt block
- [x] Kernel injection + usage log
- [x] Inactive candidate drafting
- [x] Explicit promotion flow

## Phase 3 — The Black Box (Memory) ✅

*SQLite + FTS5 + ontological graph + semantic fusion in Go; 3.E closes session visibility, audit trails, decay, and cross-chat/session boundaries*

### 3.A — SQLite + FTS5 Lattice ✅

- [x] SqliteStore
- [x] FTS5 triggers
- [x] Schema migrations v3a->v3d

### 3.B — Ontological Graph + LLM Extractor ✅

- [x] Extractor
- [x] Entity/relationship upsert
- [x] Dead-letter queue

### 3.C — Neural Recall + Context Injection ✅

- [x] RecallProvider
- [x] 2-layer seed selection
- [x] CTE traversal
- [x] <memory-context> fence

### 3.D — Semantic Fusion + Local Embeddings ✅

- [x] Ollama embeddings
- [x] Vector cache
- [x] Cosine similarity recall
- [x] Hybrid fusion

### 3.D.5 — Memory Mirror (USER.md sync) ✅

- [x] Async background export
- [x] SQLite as source of truth

### 3.E.1 — Session Index Mirror ✅

- [x] Read-only bbolt sessions.db -> index.yaml mirror
- [x] Deterministic mirror refresh without mutating session state

### 3.E.2 — Tool Execution Audit Log ✅

- [x] Append-only JSONL writer + schema
- [x] Kernel + delegate_task audit hooks
- [x] Outcome, duration, and error capture

### 3.E.3 — Transcript Export Command ✅

- [x] gormes session export <id> --format=markdown
- [x] Render turns, tool calls, and timestamps from SQLite

### 3.E.4 — Extraction State Visibility ✅

- [x] gormes memory status command
- [x] Extractor queue depth + dead-letter summary

### 3.E.5 — Insights Audit Log ✅

- [x] Append-only daily usage.jsonl writer
- [x] Session, token, and cost rollups from local runtime

### 3.E.6 — Memory Decay ✅

- [x] relationships.last_seen schema + backfill
- [x] Relationship writer freshness updates
- [x] Deterministic weight attenuation at recall time

### 3.E.7 — Cross-Chat Synthesis ✅

- [x] user_id concept above chat_id
- [x] Same-chat default recall fence
- [x] Opt-in user-scope recall + source filters
- [x] Interrupted-turn memory sync suppression
- [x] Honcho-compatible scope/source tool schema
- [x] Honcho host integration compatibility fixtures
- [x] SillyTavern persona and group-chat mapping fixtures
- [x] Cross-chat deny-path fixtures
- [x] Cross-chat operator evidence

### 3.E.8 — Session Lineage + Cross-Source Search ✅

- [x] parent_session_id lineage for compression splits
- [x] Gateway resume follows compression continuation
- [x] Source-filtered session/message search core
- [x] GONCHO user-scope search/context parameters
- [x] Lineage-aware source-filtered search hits
- [x] Operator-auditable search evidence

### 3.F — Goncho Honcho Memory Parity ✅

- [x] Goncho context representation options
- [x] Goncho search filter grammar
- [x] Vector store + reconciler divergence proof
- [x] Directional peer cards and representation scopes
- [x] Goncho queue status read model
- [x] Goncho summary context budget
- [x] Goncho dialectic chat contract
- [x] Goncho file upload import ingestion
- [x] Goncho topology design fixtures
- [x] Goncho operator diagnostics contract
- [x] Goncho streaming chat persistence contract
- [x] Goncho configuration namespace
- [x] Goncho dreaming scheduler contract
- [x] Goncho CRUD lifecycle invariants
- [x] Goncho empty peer-card hint contract
- [x] Hermes memory tool over Goncho/local durable store
- [x] Goncho memory provider lifecycle adapter
- [x] Goncho Memory V1 compatibility contract and migration harness
- [x] GONCHO local-first markdown MCP memory requirement

### 3.G — Goncho Drop-In Compatibility Closure ✅

- [x] Goncho keys + webhooks compatibility surface
- [x] Goncho webhook delivery retry worker contract
- [x] Goncho HTTP route parity over OpenAPI v3
- [x] Goncho CLI command-tree parity
- [x] Goncho Honcho SDK compatibility e2e harness
- [x] Goncho memory integration into normal agent turn

## Phase 4 — The Brain Transplant ✅

*Native Go agent orchestrator + prompt builder*

### 4.A — Provider Adapters ✅

- [x] Provider interface + stream fixture harness
- [x] Hermes provider registry and alias manifest
- [x] Tool-call normalization + continuation contract
- [x] DeepSeek/Kimi reasoning_content echo for tool-call replay
- [x] DeepSeek/Kimi cross-provider reasoning isolation
- [x] DeepSeek/Kimi all-assistant reasoning_content replay
- [x] Moonshot/Kimi tool-schema sanitizer
- [x] Anthropic
- [x] Azure OpenAI query/default_query transport contract
- [x] Azure Anthropic Messages endpoint contract
- [x] Azure Foundry transport probe read model
- [x] Azure Foundry probe — path sniffing
- [x] Azure Foundry probe — /models classification + Anthropic fallback
- [x] Azure Foundry runtime env/config read model
- [x] Azure Foundry CLI setup/status manual fallback
- [x] Azure Foundry Responses-only model-family API mode
- [x] Bedrock provider runtime binding
- [x] Bedrock Converse payload mapping (no AWS SDK)
- [x] Bedrock stream event decoding (SSE fixtures)
- [x] Bedrock SigV4 + credential seam
- [x] Bedrock stale-client eviction + retry classification
- [x] Gemini Cloud Code request/stream mapper
- [x] OpenRouter compatible-provider routing
- [x] Google Code Assist project/quota resolver
- [x] Codex
- [x] Codex Responses pure conversion harness
- [x] Codex Responses assistant content role types
- [x] Codex Responses HTTP client binding
- [x] Codex OAuth state + stale-token relogin
- [x] Codex stream repair + tool-call leak sanitizer
- [x] Cross-provider reasoning-tag sanitization
- [x] Tool-call argument repair + schema sanitizer
- [x] OpenAI-compatible developer-role API-boundary swap
- [x] xAI Grok provider adapter
- [x] LM Studio provider adapter

### 4.B — Context Engine + Compression ✅

- [x] Long session management
- [x] Context compression
- [x] ContextEngine interface + status tool contract
- [x] Compression token-budget trigger + summary sizing
- [x] Aux compression headroom for system and tool schemas
- [x] Aux compression provider-aware context cap
- [x] Tool-result pruning + protected head/tail summary
- [x] Aux compression single-prompt threshold reconciliation
- [x] Compression protected-tail multimodal length estimator
- [x] Context compressor image-token budget charge
- [x] Context references stable-handle store
- [x] Manual compression feedback + context references
- [x] Manual compression feedback renderer + focus parser
- [x] ContextEngine compression-boundary callback vocabulary
- [x] Kernel compression-boundary callback binding

### 4.C — Native Prompt Builder ✅

- [x] Default agent identity / SOUL.md loader
- [x] Context-file discovery + injection scan
- [x] Progressive subdirectory hint tracker
- [x] Model-specific role and tool-use guidance
- [x] Toolset-aware skills prompt snapshot
- [x] Memory guidance constant + injection
- [x] Session search guidance constant + injection
- [x] Gormes self-help skill/docs prompt guidance
- [x] [SYSTEM:→[IMPORTANT: meta-instruction prefix rename for Azure content filter compatibility
- [x] Native full prompt assembly
- [x] Ephemeral prefill messages file injection

### 4.D — Smart Model Routing ✅

- [x] Model metadata registry + context limits
- [x] Provider-enforced context-length resolver
- [x] Model pricing/capability registry fixtures
- [x] Ollama Cloud models.dev suffix normalization
- [x] Model catalog cache + preferred-provider live merge
- [x] Routing policy and fallback selector
- [x] Per-turn model selection
- [x] Per-turn reasoning effort propagation

### 4.E — Trajectory + Insights ✅

- [x] Trajectory writer + redaction gates
- [x] Trajectory compressor + compressed-evidence lineage
- [x] Self-monitoring telemetry

### 4.F — Title Generation ✅

- [x] Title prompt and truncation contract
- [x] Title auxiliary failure visibility
- [x] Auto-naming sessions

### 4.G — Credentials + OAuth ✅

- [x] Token vault
- [x] Anthropic OAuth/keychain credential discovery
- [x] Multi-account auth
- [x] Credential non-ASCII sanitizer + one-shot warning
- [x] Google OAuth flow + refresh seam
- [x] GitHub Copilot token exchange + Responses mode selector

### 4.H — Rate / Retry / Caching ✅

- [x] Provider-side resilience
- [x] Classified provider-error taxonomy
- [x] Provider image-too-large error classification
- [x] Unsupported temperature retry + Codex no-temperature guard
- [x] Codex Responses temperature guard after flush removal
- [x] Generic unsupported-parameter retry + max_tokens guard
- [x] Jittered reconnect backoff schedule
- [x] Retry-After header parsing + HTTPError hint
- [x] Kernel retry honors Retry-After hint
- [x] Streaming interrupt retry suppression
- [x] Provider timeout config fail-closed helper
- [x] Prompt-cache capability guard
- [x] Provider account usage read model + renderer
- [x] Gateway /usage command binding over provider account usage
- [x] Provider rate guard + budget telemetry
- [x] Provider rate guard — x-ratelimit header classification
- [x] Provider rate guard — degraded-state + last-known-good evidence
- [x] Hermes fast-mode request override serializer

### 4.I — Native Agent Turn Closure ✅

- [x] Python-free normal agent turn e2e harness
- [x] Provider-tool-memory golden transcript suite
- [x] Hermes and Honcho feature parity map to Go implementation plan
- [x] Upstream source coverage ledger for Hermes/Honcho mapping completeness
- [x] Swarm feature-level parity audit for Hermes/Honcho map
- [x] Hermes/Honcho Go runtime plan second-wave reconciliation
- [x] Nested feature-level coverage test matrix for swarm gaps
- [x] Hermes website docs mirror coverage gate
- [x] Gormes setup/channel/provider docs webpage parity gate
- [x] Native runtime provider gateway binding
- [x] Hermes compatibility namespace retirement boundary

### 4.J — Permission-Hardened Tool Execution ✅

- [x] Shell blocklist + filesystem scoping + permission approval

### 4.K — Provider Fallback Chain ✅

- [x] Resilient provider chain dispatch
- [x] Hermes fallback activation + classifier carve-outs

### 4.L — Safety-Anchored Turn Loop (MOSAIC) ✅

- [x] Plan gate hook in agent turn loop
- [x] Tool gate pre-execution validation
- [x] Refusal-as-action in ReAct cycle
- [x] Safety loop end-to-end integration

### 4.M — Advanced Provider Routing ✅

- [x] Circuit breaker per provider and API key
- [x] P95 latency-aware failover
- [x] Capability-based model tier routing

## Phase 5 — The Final Purge 🔨

*Python tool scripts ported to Go or WASM*

### 5.A — Tool Surface Port ✅

- [x] 61-tool registry port
- [x] Tool registry inventory + schema parity harness
- [x] Tool parity manifest refresh for Hermes b35d692f
- [x] Tool parity manifest refresh for Hermes ea86714 computer_use
- [x] Tool parity manifest refresh for Hermes 524cbabd patch schema
- [x] Microsoft Graph auth/client helper parity
- [x] Discord tool split + platform-scoped toolsets
- [x] Discord tool limit coercion helper
- [x] Home Assistant HASS_TOKEN platform-toolset carveout
- [x] Home Assistant tool handlers + service safety validation
- [x] Pure core tools first
- [x] Stateful tool migration queue
- [x] Terminal process watch notification throttle contract
- [x] Tool output budget persisted artifact pointer
- [x] Tool descriptor layer (OperationSpec)

### 5.B — Sandboxing Backends ✅

- [x] Environment interface + file sync contract
- [x] Terminal snapshot source stdout suppression guard
- [x] Terminal deleted-cwd recovery guard
- [x] Raw tool-call parser fixture matrix
- [x] Docker execution backend (container lifecycle + mount policy)
- [x] Docker backend top-level container reuse semantics
- [x] Modal
- [x] Daytona
- [x] Singularity command/preflight contract
- [x] Sandbox Policy Explain

### 5.C — Browser Automation ✅

- [x] Browser action contract + event transcript
- [x] go-browser-harness Chromedp action backend
- [x] Rod
- [x] Browser provider bridge + Firecrawl fallback
- [x] Camofox REST browser mode and managed identity bridge
- [x] Browser Use cloud + Go browser harness bridge
- [x] Go browser-harness Hermes browser_* tool wrappers
- [x] Go-native Hermes web_search/web_extract tool wrappers
- [x] Go-native Hermes web backend matrix and config resolver
- [x] Go-native Hermes web extract safety policy and summarizer
- [x] Go-native Hermes web_crawl tool adapter
- [x] Go-native Hermes web managed gateway status and live smoke closure
- [x] Brave Search + DDGS web search provider parity
- [x] Browser artifact and console render contract
- [x] Telegram browser artifact rendering
- [x] Browser hybrid private-URL local sidecar routing
- [x] Browser SSRF quoted-false guard
- [x] Go browser harness binary repo + integration lane (placeholder)
- [x] Browser session inactivity cleanup thread

### 5.D — Vision + Image Generation ✅

- [x] Multimodal in/out
- [x] Image input mode router + native content parts
- [x] Image-too-large shrink retry helper
- [x] Image generation result contract
- [x] Image generation provider registry + plugin dispatch
- [x] Native video_analyze tool contract

### 5.E — TTS / Voice / Transcription ✅

- [x] Voice mode port
- [x] Voice mode environment detector + audio provider seam
- [x] Transcription tool contract
- [x] Telegram voice/audio STT ingress hook
- [x] TTS tool contract + media delivery seam
- [x] MiniMax TTS v1 text_to_speech raw-audio compatibility
- [x] TTS provider matrix + dotenv/command-provider resolution
- [x] TTS synthesis + voice-mode state
- [x] Voice record-key config binding for native TUI

### 5.F — Skills System (Remaining) ✅

- [x] Skills hub search result types + in-memory registry provider
- [x] Skills hub search read-model function over registry providers
- [x] Skill registries
- [x] Skills hub direct URL candidate parser
- [x] Skills hub direct URL install name/category guard
- [x] Skill preprocessing + dynamic slash commands
- [x] [IMPORTANT:] prompt prefix for cron and skill commands
- [x] Skills list — enabled/disabled status column + --enabled-only filter
- [x] Update bundled skills across active and named profiles
- [x] Bundled Airtable productivity skill contract
- [x] Bundled TouchDesigner MCP skill catalog contract

### 5.G — MCP Integration ✅

- [x] MCP client
- [x] Goncho MCP tool catalog
- [x] MCP server config/env resolver
- [x] MCP stdio transport + tool/list discovery
- [x] MCP HTTP transport + tool/list discovery
- [x] MCP schema normalization + structured-content adapter
- [x] MCP OAuth state store + noninteractive auth errors
- [x] MCP OAuth refresh + 401 session-expired recovery
- [x] Managed tool gateway bridge
- [x] MCP circuit breaker cooldown + reconnect reset
- [x] MCP stdio orphan cleanup after cron ticks
- [x] Gormes-native MCP host runtime boundary
- [x] MCP channels_list tool

### 5.H — ACP Integration ✅

- [x] ACP server side
- [x] ACP Client Bridge Mode
- [x] ACP JSON-RPC stdio session/prompt closeout
- [x] ACP stdio benign ping/probe suppression

### 5.I — Plugins Architecture 🔨

- [x] Plugin SDK
- [x] Dashboard theme/plugin extension status contract
- [x] Dashboard page-scoped plugin slot inventory
- [ ] Third-party extensions
- [x] Hermes plugin CLI lifecycle parity
- [x] Teams pipeline plugin CLI metadata + disabled runtime inventory
- [x] Goncho Honcho plugin session config + async write compatibility
- [x] First-party Spotify plugin fixture
- [x] First-party Google Meet plugin metadata fixture
- [x] Hindsight memory setup blank-input preservation
- [x] Agent Hooks Registry
- [x] Plugin Marketplace + Doctor
- [x] Extension Lifecycle Hook System
- [x] Plugin lifecycle hook: transform_llm_output

### 5.J — Approval / Security Guards 🔨

- [ ] Dangerous action gating
- [x] Hardline command pattern table + DetectHardline function
- [x] Recoverable dangerous patterns + blocked-result schema
- [x] Approval mode config normalization
- [x] Gateway hook auto-accept strict parser
- [x] delegate_task batch JSON-string task recovery
- [x] Subagent dangerous-command non-interactive approval policy
- [x] Concurrent tool approval callback propagation
- [x] Background review toolset restriction
- [x] Cron dangerous-command approval mode
- [x] Cron approval mode config normalizer
- [ ] Tirith, path, URL, and website policy integration
- [x] Shell blocklist (36+ dangerous patterns)
- [x] Filesystem scoping (folder-level read/write restrictions)
- [x] Permission approval UX (inline y/n/always)
- [x] Trust-class enforcement in shared tool executor
- [x] Secrets Runtime Controls
- [x] Security Audit Command
- [x] Email allowlist pre-dispatch loop guard
- [x] Auth state TOCTOU close + redaction default-on parity
- [x] Gateway allowed_chats/channels/rooms whitelist parity

### 5.K — Code Execution ✅

- [x] Sandboxed exec

### 5.L — File Ops + Patches 🔨

- [ ] Atomic checkpoints
- [x] Checkpoints CLI (status/list/prune/clear/clear-legacy)
- [x] Checkpoint shadow-repo GC policy
- [x] File read dedup cache invalidation and wrapper guard
- [x] File read repeated-stub BLOCKED escalation
- [x] Native file task tool surface
- [x] Symlink-preserving atomic writer helper
- [x] File write/patch staleness registry + cwd tracking
- [x] Terminal cwd config bridge
- [x] Terminal deleted-cwd recovery
- [x] search_files hidden-root and context-line parsing drift

### 5.M — Mixture of Agents 🔨

- [x] Multi-model coordination
- [x] Hermes Kanban durable board core
- [x] Hermes Kanban dispatcher and worker spawn loop
- [x] Hermes Kanban production worker process binding
- [x] Hermes Kanban worker tools and prompt gating
- [x] Kanban comment author hardening and cross-task handoff policy
- [ ] Hermes Kanban slash/gateway/dashboard surfaces
- [x] Native TUI /kanban slash command binding over gormes kanban
- [x] Gateway /kanban shared command-runner binding
- [x] Kanban dashboard dispatch quick path
- [x] Kanban dashboard task run history endpoint
- [x] Kanban dispatcher status in gateway /status
- [ ] Hermes Kanban multi-board, workspace, and run-history parity
- [x] Kanban named-board workspace and log roots
- [x] Kanban current-board task command routing
- [x] Kanban task run history command
- [x] Kanban boards list/show task-count read model
- [x] Kanban global --board task command override
- [x] Kanban GC terminal event and worker-log retention
- [x] Kanban worker heartbeat, reclaim, and zombie detection
- [x] Hermes Kanban specify triage parity

### 5.N — Misc Operator Tools ✅

- [x] Todo
- [x] Clarify
- [x] Session search tool schema and argument validation
- [x] Session search tool execution wrapper
- [x] Session shutdown memory transcript handoff
- [x] Debug helpers
- [x] Debug share paste sweep scheduler contract
- [x] Doctor GitHub CLI auth fallback
- [x] Planner audit blank-subphase control-plane bucket
- [x] Autoloop recent-failure detail excerpts
- [x] Backend usage-limit stdin health bypass
- [x] Cronjob tool API + schedule parser parity
- [x] Cron schedule parser + repeat state fixtures
- [x] Cron recurring next-run failure preservation
- [x] Cron prompt/script safety + pre-run script contract
- [x] Cronjob tool action envelope over native store
- [x] Cron run resource release contract
- [x] Cron run resource release executor binding
- [x] Cron context_from output chaining
- [x] Cron prompt/script safety + pre-run script contract (deprecated umbrella)
- [x] Cron multi-target delivery + media/live-adapter fallback
- [x] Cron deliver=all routing intent expansion
- [x] Goncho serialized write queue + relation candidates
- [x] Blocker Policy Integration
- [x] OpenClaw SecretRef core resolver
- [x] Cross-agent config isolation
- [x] SecretRef runtime snapshot activation
- [x] OpenClaw security audit --deep --fix
- [x] ACP bridge doctor/status evidence
- [x] Gateway probe auth/capability HTTP closeout
- [x] Safety-critical panic and swallowed-error closeout
- [x] Session Health Monitoring
- [x] Evidence-Before-Claims Quality Gate
- [x] Git Delivery Contract Enforcement
- [x] QMD Hybrid Search
- [x] Session Rollover Automation
- [x] System Events, Heartbeat, and Presence
- [x] Gateway Discover and Probe
- [x] Channels Capabilities Introspection
- [x] Prompt Fragment Include System
- [x] Multi-agent gateway runtime activation
- [x] Multi-agent auth and tool-policy runtime isolation
- [x] Cron env-ref expansion + parallel run state serialization
- [x] Cron origin delivery isolation from session identity
- [x] Cron script/workdir/inactivity execution binding
- [x] Cron no-agent script-only watchdog mode
- [x] Cron partial legacy job read-model normalization
- [x] Cron dashboard partial-record page
- [x] Navivox host setup apply with transient sudo
- [x] Gateway auto-resume on restart

### 5.O — Hermes CLI Parity ✅

- [x] 49-file CLI tree port
- [x] Hermes CLI command-tree parity manifest
- [x] Hermes CLI nested parser inventory refresh
- [x] Hermes auth command-tree manifest refresh
- [x] Hermes auth credential-pool command surface
- [x] Hermes auth OAuth provider adapters
- [x] Hermes auth Spotify service-provider subcommand
- [x] Deterministic helper-file ports (banner/output/tips/webhook/dump)
- [x] CLI banner/output formatting helpers
- [x] CLI deterministic tip selector
- [x] CLI OpenClaw residue detection and hint text
- [x] CLI onboarding seen-state map helpers
- [x] CLI contextual first-touch onboarding hint renderers
- [x] CLI bracketed-paste wrapper sanitizer
- [x] CLI slow bracketed-paste diagnostic threshold
- [x] CLI terminal control-response sanitizer
- [x] CLI submitted user-message preview formatter
- [x] CLI webhook URL normalizer
- [x] CLI dump support-summary helper
- [x] PTY bridge protocol adapter
- [x] CLI command registry parity + active-turn busy policy
- [x] Gateway /reasoning command parser
- [x] Gateway /reasoning apply + dispatch
- [x] Busy command guard for compression and long CLI actions
- [x] Config, profile, auth, and setup command surfaces
- [x] Gormes agent template reset command
- [x] Gormes auth bare interactive credential-pool readout
- [x] Gormes auth status per-provider aggregator
- [x] Gormes auth add openai-codex strict isolation contract
- [x] Gormes auth add bedrock open-question planning note
- [x] Gormes profile command binding
- [x] Gormes profile distribution metadata readout
- [x] Model and profile selector seam (Cobra + gateway)
- [x] Gormes top-level logout provider shortcut
- [x] Gormes login removed-command typo suggestion contract
- [x] Gormes model interactive provider/model picker
- [x] Gormes setup minimal sectioned wizard slice
- [x] Gormes setup top-level chooser menu
- [x] Gormes setup full-wizard shell and branded summary
- [x] Hermes setup entry-mode and reset semantics
- [x] Gormes setup tools checklist command binding
- [x] Gormes setup gateway platform checklist command binding
- [x] Gormes setup terminal TTS and agent-settings section bindings
- [x] Gormes uninstall dry-run command contract
- [x] Gormes mcp login interface seam + noninteractive default
- [x] Gormes mcp login browser callback flow
- [x] Hermes fallback provider chain CLI commands
- [x] Provider endpoint/API-key root flags + runtime resolution
- [x] Gormes profile skills chat invocation shim
- [x] Hermes config.yaml Telegram compatibility bridge
- [x] Gormes config command surface
- [x] Gormes config edit/check/native schema-migrate closeout
- [x] Hermes config migration dry-run manifest
- [x] Hermes config migration writer
- [x] OpenClaw migration dry-run manifest
- [x] OpenClaw migration writer and cleanup command
- [x] CLI profile name validator
- [x] CLI profile root resolver
- [x] CLI active-profile store
- [x] CLI profile path and active-profile store (deprecated umbrella)
- [x] Top-level oneshot flag and model/provider resolver
- [x] Oneshot final-output writer boundary
- [x] Oneshot noninteractive safety and clarify policy
- [x] Platform toolset config persistence + MCP sentinel
- [x] Effective toolset picker dedupes bundled plugin keys
- [x] Gateway, platform, webhook, and cron management CLI
- [x] WhatsApp top-level pairing wizard shell
- [x] WhatsApp live Baileys QR pairing wizard
- [x] Gateway management CLI read-model closeout
- [x] Gateway mutating-subcommand unavailability stub
- [x] Windows gateway Scheduled Task lifecycle commands
- [x] Service RestartSec parser helper
- [x] Service restart active-status poller
- [x] Diagnostics, backup, logs, and status CLI
- [x] Hermes sessions CLI MRU browse/delete ergonomics
- [x] Backup/update opt-in and exclusion policy
- [x] Self-update command lifecycle safety
- [x] doctorCustomEndpointReadiness check function
- [x] Custom provider model-switch credential preservation
- [x] Custom provider model-switch key_env write guard
- [x] CLI log redactor for known secret shapes
- [x] CLI log snapshot reader using shared redactor
- [x] Hermes config.yaml model/provider runtime bridge
- [x] Interactive Onboarding
- [x] Gormes onboard interactive action runner
- [x] CLI setup/onboard/help text fidelity matrix
- [x] Hermes CLI alias and suggestion fidelity matrix
- [x] Logs Command
- [x] Gateway planned stop marker + WSL systemd PATH parity
- [x] Gateway stale-code self-check uses git HEAD SHA
- [x] Agent lifecycle hooks (agent:start, agent:step, agent:end)
- [x] Nous OAuth device code + refresh token + agent key provisioning

### 5.P — Docker / Packaging 🔨

- [x] OCI image
- [x] Homebrew
- [x] Nix flake package and NixOS module contract
- [x] Unix installer (install.sh) source-backed update flow
- [x] Unix installer root/FHS layout policy
- [x] Windows installer (install.ps1 + install.cmd) parity
- [ ] Installer site asset/route coverage
- [x] Install isolation: GORMES_BIN_DIR is an authoritative sandbox boundary
- [x] Install isolation: skip shell-rc PATH write when bin dir is under /tmp
- [x] Install isolation: skip system service install when sandbox bin dir is set
- [x] Install: prefer pre-built release binary over source build by default

### 5.Q — API Server + TUI Gateway Streaming 🔨

- [ ] Deterministic helper-file ports (tool-progress/image/completion-path/personality/platform-event)
- [x] TUI gateway tool-progress mode normalizer
- [x] TUI gateway completion path normalizer
- [x] TUI gateway tool summary formatter
- [x] TUI gateway image/personality/platform-event helpers
- [x] TUI gateway config health null-section probe
- [x] TUI mouse tracking config + slash toggle
- [x] Native TUI bundle independence check
- [x] TUI launch model override + static alias resolver
- [x] TUI prompt-submit auto-title eligibility helper
- [x] TUI TerminalNativeSelectionHelp constant + help-string fixture
- [x] Native TUI slash-command dispatch table
- [x] Native TUI /save canonical session export
- [x] Native TUI /save XDG export helper
- [x] Native TUI /save local runtime binding
- [x] Native TUI /branch session fork + transcript target switch
- [x] TUI running-agent placeholder surfaces interrupt + queued slash actions
- [x] Native TUI conversation viewport tail helper
- [x] Native TUI queued-message edit helper
- [x] Native TUI renderConv viewport budget binding
- [x] Native TUI Hermes skin token renderer
- [x] Native TUI Hermes status bar renderer
- [x] Native TUI Hermes bottom-pinned chrome layout
- [x] Native TUI Hermes input keybinding semantics
- [x] Native TUI Shift+Enter newline CSI-u parity
- [x] Native TUI clipboard, OSC52, and terminal setup parity
- [x] Native TUI image/file drop + paste collapse ingress
- [x] Native TUI Hermes slash completion helpers
- [x] Native TUI absolute path completion routing
- [x] Native TUI Hermes slash dispatch behavioral matrix
- [x] Native TUI /quit local exit binding
- [x] Native TUI Hermes tool progress + modal panel renderers
- [x] Native TUI Ink behavioral transcript golden matrix
- [x] Native TUI markdown soft-wrap boundary trim
- [x] Channel/TUI iteration-limit finalization transcript fixture
- [x] SSE streaming to Bubble Tea TUI
- [x] TUI websocket attach transport
- [x] OpenAI-compatible chat-completions API server
- [x] API server multimodal content preservation
- [x] Responses API store + run event stream
- [x] API server disconnect snapshot persistence
- [x] Gateway proxy mode forwarding contract
- [x] Dashboard API client contract
- [x] Dashboard PTY chat sidecar contract
- [x] API server detailed health snapshot contract
- [x] API server detailed health endpoint
- [x] API server cron admin read-only endpoints
- [x] API server cron admin mutating endpoints
- [x] API server legacy jobs routes + default toolset
- [x] Provider client lazy-init for TUI cold-start budget

### 5.R — Code Execution Mode Policy ✅

- [x] Execution-mode resolver + config precedence
- [x] Strict-mode CWD + interpreter parity
- [x] Project-mode CWD + active venv detection
- [x] Default mode selection + config cut-over

### 5.S — Loop Detection ✅

- [x] 5-type loop detector

### 5.T — Browser Harness Doctor ✅

- [x] go-browser-harness doctor subcommand

### 5.U — Fault-Tolerant Sandbox Execution ✅

- [x] Pre-execution command classification
- [x] Transactional tool execution with snapshot/rollback
- [x] Sandbox isolation depth selection

### 5.V — Unified Event Bus ✅

- [x] Event bus core: pub/sub interface + in-process implementation
- [x] Gateway channel adapters publish to event bus
- [x] Gateway outbound sends publish message-sent events
- [x] Weixin gateway event-bus adapter
- [x] WeCom gateway event-bus adapter
- [x] Telegram gateway event-bus adapter
- [x] Discord gateway event-bus adapter
- [x] Slack gateway event-bus adapter
- [x] WhatsApp gateway event-bus adapter
- [x] Agent turn and tool execution events on bus
- [x] Event bus integration test: full message flow

### 5.W — i18n Internationalization ✅

- [x] Hermes i18n static-message port

## Phase 6 — The Learning Loop (Soul) 🔨

*Hermes-compatible background review and skill curation, plus Gormes-native evidence gates for safe compounding intelligence.*

### 6.A — Complexity Detector 🔨

- [x] Hermes background review fork lifecycle
- [ ] Heuristic or LLM-scored signal

### 6.B — Skill Extractor ✅

- [x] LLM-assisted pattern distillation

### 6.C — Skill Storage Format ✅

- [x] SKILL.md frontmatter validation guard
- [x] Hermes creative skill metadata compatibility
- [x] Portable SKILL.md format

### 6.D — Skill Retrieval + Matching ✅

- [x] Hybrid lexical + semantic lookup
- [x] Source-aware retrieval damping fixtures
- [x] Delta-bounded skill and memory maintenance passes
- [x] Code Cathedral II code-context retrieval fixtures

### 6.E — Feedback Loop 🔨

- [x] Hermes curator auxiliary model routing slot
- [x] Hermes curator state transitions and run reports
- [ ] Skill effectiveness scoring

### 6.F — Skill Surface 🔨

- [x] Hermes skill_manage support-file and curator intent actions
- [x] Hermes curator command surface
- [ ] TUI + Telegram browsing
- [x] Native skills list/view tool surface

### 6.G — Structured Memory Types ✅

- [x] 6 typed memory categories with confidence scoring

### 6.H — Skill Metadata Placement ✅

- [x] SKILL.md metadata.when/loaded/placement schema

### 6.I — Zero-LLM Knowledge Graph ✅

- [x] Regex-based auto-link extraction + brain-first lookup

### 6.J — Agentic Memory Lifecycle (AgeMem) ✅

- [x] Memory operations as agent-callable tools
- [x] Agent-controlled memory retention with importance scoring
- [x] Cross-session memory continuity

### 6.K — Self-Evolution Engine (GEPA) 🔨

- [x] Prompt evaluation harness
- [x] Iterative prompt mutation and scoring loop
- [ ] Behavioral pattern extraction from session logs

### 6.L — Composable Skill Execution (Voyager) ✅

- [x] Skill code execution runtime
- [x] Skill dependency resolution and composition
- [x] Skill validation on load with execution proof

## Phase 7 — Paused Channel Backlog 🔨

*Deferred non-priority channel adapters after Telegram, Discord, Slack, WhatsApp, and WeChat stabilize*

### 7.A — Signal Adapter ✅

- [x] Inbound event normalization + session identity
- [x] Reply/send contract on shared chassis
- [x] Signal transport/bootstrap layer
- [x] Signal markdown bodyRanges + attachment rate scheduler

### 7.B — Email + SMS Adapters ✅

- [x] Email ingress + outbound delivery contract
- [x] SMS ingress + outbound delivery contract

### 7.C — Matrix + Mattermost Adapters ✅

- [x] Threaded text adapter contract suite
- [x] Matrix shared-chassis bot seam
- [x] Matrix self/bridge sender drop helper
- [x] Mattermost shared-chassis bot seam
- [x] Matrix real client/bootstrap layer
- [x] Matrix E2EE device-id crypto-store binding
- [x] Mattermost REST/WS bootstrap layer

### 7.D — Webhook + Trigger Ingress ✅

- [x] Signed event parsing + auth gates
- [x] Prompt-to-delivery routing bridge

### 7.E — Regional + Device Adapter Backlog 🔨

- [x] BlueBubbles + HomeAssistant adapters
- [x] BlueBubbles iMessage bubble formatting parity
- [x] Feishu shared-chassis bot seam
- [x] DingTalk shared-chassis bot seam
- [x] QQ Bot shared-chassis bot seam
- [x] Feishu transport/bootstrap layer
- [x] Feishu drive-comment rule + pairing seam
- [x] Feishu drive-comment reply workflow
- [x] DingTalk transport/bootstrap layer
- [ ] DingTalk real SDK binding
- [x] DingTalk AI Cards streaming-update contract
- [x] DingTalk emoji reaction send/receive parity
- [x] DingTalk media (image/file) attachment routing
- [x] Yuanbao protocol envelope + markdown fixtures
- [x] Yuanbao media/sticker attachment normalization
- [x] Yuanbao gateway runtime + toolset registration
- [x] Microsoft Teams adapter plugin seam
- [x] QQ Bot transport/bootstrap layer
- [x] Google Chat shared-chassis platform adapter seam
- [x] Google Chat relay sender-type self-filter

## Phase 8 — Reputation & Publication 🔨

*TrebuchetDynamics has a credible public face (blog, writeups, talks) that documents Gormes's autonomous-porting methodology and one or two sharp differentiators. Reputation is built through publication cadence, not parity scope.*

### 8.A — Publication Infrastructure ⏳

- [ ] TD engineering blog scaffolded and live
- [ ] TD social presence connected to blog feed

### 8.B — Repository Messaging ✅

- [x] README rewrite to methodology-first positioning
- [x] gormes.ai landing page positioning audit

### 8.C — Engineering Writeups ⏳

- [ ] Engineering writeup #1: autonomous Hermes-porting loop

### 8.D — Sharp v1.0 🔨

- [x] Sharp v1.0 differentiator decision
- [ ] Single-binary cross-platform release pipeline
- [x] Windows install.ps1 release binary fetch selector
- [x] OCI image PR build and arm64 smoke workflow
- [x] Release build-date provenance injection
- [x] Release notes artifact size table
- [x] Release SBOM attestation binding
- [x] Release build provenance attest action contract
- [x] Release archive 30 MB size gate
- [x] Termux android/arm64 release artifact and installer selector

### 8.E — Toolkit Extraction ⏳

- [ ] Agentic-porting-kit repo scaffold

### 8.F — Cost Discipline & Loop Economics ✅

- [x] Loop $/iteration cost metric in status file

### 8.G — Community & External Contributions ⏳

- [ ] Built-with-Gormes page scaffold

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
