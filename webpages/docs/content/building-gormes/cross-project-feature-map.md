# Cross-Project Feature Map & Long-Term Plan for Gormes-Agent

## Executive Summary

Analysis completed on 2026-04-30 of all 12 opensource projects in workspace-mineru:
- **hermes-agent** (Python upstream - 519K lines CLI, 702K lines agent runner)
- **honcho** (Python memory/session - Peer paradigm, 3-agent memory system)
- **gormes** (TypeScript memory/runtime - Brain-first knowledge graph, minions queue)
- **browser-harness** (Python browser automation - CDP, Camofox, daemon lifecycle)
- **go-browser-harness** (Go browser automation - Chromedp, stateless actions)
- **mercury-agent** (TypeScript CLI/Telegram - Soul-driven, permission-hardened)
- **space-agent** (JavaScript browser-first - Agent reshapes interface, WebLLM)
- **picoclaw** (Go lightweight agent - 18+ channels, 30+ providers, MCP)
- **go-agent-os references** (8 donor repos - patterns for OAuth, retry, tools, state machines)

**Gormes-agent current parity: ~20-30% of hermes-agent features**

---

## Master Feature Matrix

### Category 1: Provider Adapters

| Provider | Hermes | Gormes Status | Gap Severity |
|----------|--------|---------------|--------------|
| Anthropic | `anthropic_adapter.py` (1,888) | `anthropic_client.go` | Complete |
| OpenAI/ChatGPT | `chat_completions.py` (20,659) | `http_client.go` | Complete |
| Codex Responses | `codex_responses_adapter.py` (999) | `codex_responses_adapter.go` | Complete |
| AWS Bedrock | `bedrock_adapter.py` (1,264) | `bedrock_*.go` | Complete |
| Gemini CloudCode | `gemini_cloudcode_adapter.py` (903) | `gemini_cloudcode.go` | Complete |
| Google Code Assist | `google_code_assist.py` (452) | `google_code_assist.go` | Complete |
| OpenRouter | `openrouter_client.py` (1,091) | `openrouter.go` | Complete |
| Google OAuth | `google_oauth.py` (1,048) | `google_oauth_state.go` | Partial |
| Azure Foundry | Via `azure_detect.py` (11,951) | `azure_foundry_*.go` | Partial |
| LM Studio | `lmstudio_reasoning.py` (48) | Not implemented | Gap |
| Moonshot/Kimi | `moonshot_schema.py` | Schema only | Gap |
| xAI/Grok | `xai_http.py` (349) | Not implemented | Gap |
| DeepSeek | Via reasoning isolation | Partial | Gap |

**Learnings from other projects:**
- **Picoclaw**: 30+ providers with unified interface - could adopt pattern for provider registration
- **Mercury**: Provider fallback chain (DeepSeek → OpenAI → Anthropic → Grok → Ollama) - resilient routing
- **Gormes-owned**: Multi-provider LLM abstraction with fallback models per provider

### Category 2: Browser Automation

| Feature | Hermes (Python) | Go Harness | Gormes | Priority |
|---------|-----------------|------------|--------|----------|
| Navigate/Snapshot/Click | Complete | Complete | Complete | Done |
| Type/Scroll/Back/Press | Complete | Complete | Complete | Done |
| Console/Vision/CDP | Complete | Complete | Complete | Done |
| Dialog/Get Images | Complete | Complete | Complete | Done |
| HTTP GET (no browser) | Complete | **Gap** | **Gap** | Medium |
| Daemon lifecycle | Complete | **Gap** | **Gap** | High |
| Profile management | Complete | **Gap** | **Gap** | Medium |
| Cloud browser (Browser Use) | Complete | **Gap** | **Gap** | Low |
| Tab enumeration/switch | Complete | **Gap** | **Gap** | Medium |
| Iframe targeting | Complete | **Gap** | **Gap** | Low |
| Event drain | Complete | **Gap** | **Gap** | Medium |
| Doctor/diagnostics | Complete | **Gap** | **Gap** | High |

**Learnings:**
- **Space-agent**: Browser-first architecture with floating windows, DOM capture with typed refs (`[link 12]`), nested iframe support - completely different paradigm
- **Hermes**: 2,991-line `browser_tool.py` + 1,362-line `browser_supervisor.py` + 563-line `browser_cdp_tool.py`

### Category 3: Tools Registry

| Tool Category | Hermes | Gormes | Gap |
|---------------|--------|--------|-----|
| **Core tools** | ~50 tools | ~15 tools | **Major** |
| Browser automation | 4 files, ~4,500 lines | 2 files, ~500 lines | Large |
| File operations | 2 files, ~102,000 lines | 1 file, ~200 lines | **Massive** |
| Terminal/Shell | `terminal_tool.py` (2,307) | `terminal_tool.go` | Medium |
| Code execution | `code_execution_tool.py` (1,609) | Not implemented | Gap |
| Web scraping | `web_tools.py` (89,105) | Partial | **Massive** |
| MCP client | `mcp_tool.py` (3,140) + OAuth (22K) | Partial | Large |
| Delegation/Subagents | `delegate_tool.py` (2,531) | `subagent/` minimal | Large |
| Skills hub | `skills_hub.py` (118,874!) | `skills/` basic | **Massive** |
| Cron/Scheduler | `cronjob_tools.py` (26,525) | `cron/` basic | Large |
| TTS/Voice | `voice_mode.py` (38,753) | Partial | Large |
| Image generation | `image_generation_tool.py` (1,002) | `image_generation.go` | Complete |
| Vision | `vision_tools.py` (31,333) | Not implemented | Gap |
| RL Training | `rl_training_tool.py` (1,396) | Not implemented | Gap |
| Process registry | `process_registry.py` (61,200) | Basic | **Massive** |
| Checkpoint manager | `checkpoint_manager.py` (31,551) | Basic | Large |
| Approval workflow | `approval.py` (52,681) | Not implemented | Gap |

**Learnings from other projects:**
- **Mercury**: 31 built-in tools with permission-hardened execution (shell blocklist, filesystem scoping)
- **Space-agent**: Skill system with metadata-driven placement (`metadata.when`, `metadata.loaded`, `metadata.placement`)
- **Picoclaw**: Cron, web search (DuckDuckGo, Baidu, Tavily, Brave, Perplexity, SearXNG), filesystem, shell, spawn
- **Gormes-owned**: Cathedral II code navigation (call-graph edges, two-pass retrieval)

### Category 4: Memory Systems

| Feature | Hermes | Honcho | Gormes-owned | Mercury | Gormes |
|---------|--------|--------|--------|---------|--------|
| Storage | SQLite | PostgreSQL+pgvector | SQLite+FTS5 / PostgreSQL | SQLite+FTS5 | SQLite (Goncho) |
| Vector search | Not core | ✅ HNSW cosine | ✅ Hybrid (RRF+cosine) | FTS5 keyword | Not yet |
| Typed memories | Not core | ✅ 10 types | Pages+Chunks+Links | ✅ 10 types | Session-based |
| Auto-extraction | Not core | ✅ Deriver agent | Auto-link extraction | ✅ Post-conversation | Not yet |
| Conflict resolution | Not core | ✅ Confidence-based | N/A | ✅ Confidence+recency | Not yet |
| Auto-pruning | Not core | Not core | Maintenance cycle | ✅ 21-day stale | Not yet |
| Peer paradigm | Not core | ✅ Unified users+agents | Not core | Not core | Not yet |
| Knowledge graph | Not core | Observation links | ✅ Auto-wiring (zero-LLM) | Not core | Not yet |
| Session context | ✅ | ✅ Combined | ✅ Tiered | ✅ Top-5 injection | ✅ Partial |

**Learnings:**
- **Honcho**: Three-agent memory system (Deriver/Dialectic/Dreamer) with observation levels (explicit/deductive/inductive/contradiction)
- **Gormes-owned**: Brain-first lookup (5-step before external API), compiled truth + timeline pattern, tiered enrichment
- **Mercury**: Second Brain with 10 typed memories, hourly heartbeat consolidation, confidence/durability scoring

### Category 5: Gateway/Channels

| Channel | Hermes | Picoclaw | Gormes | Status |
|---------|--------|----------|--------|--------|
| Discord | 173K lines | ✅ | ✅ | Complete |
| Telegram | 141K lines | ✅ | ✅ | Complete |
| Slack | 102K lines | ✅ | ⚠️ Basic | Partial |
| Feishu/Lark | 192K lines | ✅ | ⚠️ Basic | Partial |
| WhatsApp | 45K lines | ✅ | ⚠️ Basic | Partial |
| WeCom | 65K lines | ✅ | ⚠️ Basic | Partial |
| Yuanbao | 185K lines | ✅ | ⚠️ Basic | Partial |
| Matrix | 105K lines | ✅ | ❌ | Gap |
| Signal | 50K lines | ✅ | ❌ | Gap |
| SMS | 14K lines | ✅ | ❌ | Gap |
| Email | 23K lines | ✅ | ❌ | Gap |
| DingTalk | 56K lines | ✅ | ⚠️ Basic | Partial |
| Mattermost | 28K lines | ✅ | ❌ | Gap |
| HomeAssistant | 16K lines | ✅ | ⚠️ Basic | Partial |
| QQ | Partial | ✅ | ❌ | Gap |
| BlueBubbles | 34K lines | ❌ | ❌ | Gap |
| LINE | Partial | ✅ | ❌ | Gap |
| VK | Partial | ✅ | ❌ | Gap |
| IRC | Partial | ✅ | ❌ | Gap |
| Webhook | 30K lines | ✅ | Partial | Partial |

**Learnings:**
- **Picoclaw**: 18+ channels as donor repo for Go channel-edge work. Gateway Donor Map explicitly documents adaptation patterns.
- **Mercury**: Multi-user Telegram org model (admin/member roles, pairing codes)

### Category 6: CLI/Operator Surface

| Feature | Hermes | Mercury | Gormes | Gap |
|---------|--------|---------|--------|-----|
| Auth system | `auth.py` (4,744) | Basic | Partial | Medium |
| Config management | `config.py` (4,548) | YAML | `config.go` | Partial |
| Model switching | `model_switch.py` (1,588) | Provider fallback | Partial | Medium |
| Setup wizard | `setup.py` (3,488) | None | `install.sh` | Partial |
| Diagnostics | `doctor.py` (1,390) | None | `doctor` | Partial |
| Profiles | `profiles.py` (1,167) | None | Partial | Gap |
| Backup/restore | `backup.py` (926) | None | Not yet | Gap |
| Logs viewing | `logs.py` (13,346) | None | Not yet | Gap |
| Web server | `web_server.py` (3,195) | None | Not yet | Gap |
| Tips system | `tips.py` (27,607) | None | `tips.go` | Partial |
| Status display | `status.py` (23,006) | None | Partial | Partial |
| Voice commands | `voice.py` (21,085) | None | Not yet | Gap |
| Cron commands | `cron.py` (10,839) | Scheduler | `cron/` | Partial |
| Skills config | `skills_config.py` (7,151) | Not core | Partial | Gap |
| Plugin commands | `plugins_cmd.py` (1,280) | Not core | Partial | Gap |
| Completion | `completion.py` (10,916) | Not core | Partial | Gap |

**Learnings:**
- **Mercury**: Token budget system (daily enforcement, auto-concise at 70%), loop detection (5 types), permission manifest
- **Picoclaw**: Desktop (WebUI), Headless (TUI), Android APK, Docker, Termux deployment modes

### Category 7: Security & Safety

| Feature | Hermes | Mercury | Gormes | Gap |
|---------|--------|---------|--------|-----|
| Shell blocklist | Partial | ✅ 36+ patterns | Not yet | **Critical** |
| Filesystem scoping | Partial | ✅ Folder-level | Not yet | **Critical** |
| Permission approval | Basic | ✅ Inline y/n/always | Not yet | High |
| SSRF guard | `url_safety.py` (9,429) | Basic | `browser_contract.go` | Partial |
| Path security | `path_security.py` (1,322) | Basic | Not yet | Medium |
| Credential redaction | `redact.py` (392) | Basic | Partial | Medium |
| Website policy | `website_policy.py` (9,786) | Not core | Not yet | Gap |
| Tirith security | `tirith_security.py` (26,121) | Not core | Not yet | Gap |
| OSV vulnerability check | `osv_check.py` (4,925) | Not core | Not yet | Gap |

**Learnings:**
- **Mercury**: Most mature permission system - shell blocklist, cwdOnly restriction, auto-approved patterns, YAML manifest
- **Plandex** (reference): Provider error classification with Retry-After parsing

### Category 8: Personality/Soul Systems

| Feature | Hermes | Mercury | Space | Gormes |
|---------|--------|---------|-------|--------|
| Soul files | Not core | ✅ soul.md, persona.md, taste.md, heartbeat.md | personality.system.include.md | Not yet (Phase 6) |
| Identity model | Not core | Human analogy | User-editable file | Not yet |
| Loading | Not core | User-owned markdown | Plain text include | Not yet |

**Learnings:**
- Mercury's soul system: soul=heart, persona=face, taste=palate, heartbeat=breathing
- Space's approach: single personality include file (simpler stepping stone)

### Category 9: Scheduling & Background Tasks

| Feature | Hermes | Mercury | Gormes-owned | Gormes |
|---------|--------|---------|--------|--------|
| Cron scheduling | `cron/scheduler.py` (58,318) | ✅ Cron + delayed | Not core | `cron/` basic |
| Job queue | Postgres-backed | YAML-persisted | ✅ Minions (BullMQ-inspired) | Not yet |
| Task persistence | DB | YAML | Postgres rows | Not yet |
| Fan-out | Not core | Not core | ✅ N children + aggregator | Not yet |
| DAG dependencies | Not core | Not core | ✅ Parent-child with policies | Not yet |
| Token accounting | Not core | Not core | ✅ Per-job tracking | Not yet |

**Learnings:**
- **Gormes durable jobs**: Postgres-native job queue, BullMQ-inspired, zero infra, stall detection, retry with backoff, supervisor auto-restart
- **Mercury**: Configurable heartbeat interval, episodic prune + second brain consolidate

### Category 10: Unique Paradigms from Other Projects

| Project | Unique Feature | Relevance to Gormes |
|---------|---------------|---------------------|
| **Space-agent** | Browser-first runtime (agent lives in browser) | Different paradigm - agent as peer to frontend vs backend orchestrator |
| **Space-agent** | Agent reshapes interface (builds pages/tools/widgets) | Could inform web dashboard capabilities |
| **Space-agent** | Layered customware (L0 firmware → L1 group → L2 user) | Multi-tenant configuration model |
| **Space-agent** | Git-backed time travel (rollback/revert) | Useful for workspace state recovery |
| **Space-agent** | WebLLM + HuggingFace (browser-side inference) | Novel - requires browser runtime |
| **Gormes-owned** | Zero-LLM knowledge graph wiring (regex-based auto-links) | HIGH - significantly reduces LLM calls for entity resolution |
| **Gormes-owned** | Cathedral II code navigation (call-graph edges) | HIGH - for code-aware agent capabilities |
| **Gormes-owned** | Thin harness, fat skills (intelligence in skills not runtime) | Architectural philosophy alignment |
| **Gormes-owned** | Fail-improve loop (logs regex failures, generates better patterns) | Self-improving deterministic classifiers |
| **Honcho** | Peer paradigm (users AND agents are "Peers") | Multi-agent interaction model |
| **Honcho** | Three-agent memory (Deriver/Dialectic/Dreamer) | Memory quality differentiation |
| **Honcho** | Observation levels (explicit/deductive/inductive/contradiction) | Richer memory than simple key-value |
| **Mercury** | Permission-hardened shell (36+ blocklist patterns) | **Critical** - foundational safety |
| **Mercury** | Second Brain (10 typed memories with conflict resolution) | **High** - structured memory UX |
| **Mercury** | Loop detection (5 types: hard/failing/text/no-action/same-tool) | **High** - runaway loop prevention |
| **Mercury** | Token budget + auto-concise | Cost control |
| **Picoclaw** | Ultra-lightweight (<10MB RAM, $10 hardware) | Deployment target diversity |
| **Picoclaw** | Self-bootstrapping (95% AI-generated) | Development methodology |

---

## Strategic Recommendations for Gormes

### Immediate Actions (Next 30 Days)

1. **Permission Hardening** (from Mercury)
   - Port shell blocklist (36+ patterns)
   - Implement filesystem scoping (folder-level read/write)
   - Add permission approval UX (inline y/n/always)
   - **Rationale**: Gormes has zero permission hardening; this is foundational for safe agent operation

2. **Provider Completion**
   - Complete DeepSeek/Kimi reasoning isolation
   - Implement LM Studio provider
   - Add xAI/Grok provider
   - **Rationale**: Provider parity is critical path for Python-free normal agent turn

3. **Browser Harness Gaps**
   - Implement `go-browser-harness doctor` subcommand
   - Add daemon lifecycle (start/stop/ensure)
   - **Rationale**: Core browser tools work but lack production lifecycle

### Short-Term (Next 90 Days)

4. **Loop Detection** (from Mercury)
   - Port 5-type loop detector (~200 lines of TypeScript)
   - Hard loop, failing loop, text repetition, no-action, same-tool
   - **Rationale**: Runaway loops are a real production problem

5. **Structured Memory Enhancement** (from Mercury + Honcho)
   - Extend Goncho with typed memory categories (start with 6: identity, preference, goal, habit, episode, reflection)
   - Add confidence/durability scoring
   - Implement conflict resolution (confidence wins, equal → newer)
   - **Rationale**: Current Goncho is session-based; structured memory is major UX improvement

6. **Skill Metadata** (from Space-agent)
   - Add `metadata.when`, `metadata.loaded`, `metadata.placement` to SKILL.md schema
   - Implement hierarchical routing skill pattern
   - **Rationale**: More granular skill activation control

7. **Native Prompt Builder** (Hermes parity)
   - Port `agent/prompt_builder.py` to Go
   - Context files, model-specific guidance, skills snapshots, memory/session-search assembly
   - **Rationale**: Critical path for Python-free normal agent turn

### Medium-Term (Next 6 Months)

8. **Context Compression Reconciliation** (Hermes parity)
   - Reconcile with upstream `5006b220` changes
   - Tool-result pruning, protected head/tail invariants
   - **Rationale**: Compression behavior drift from upstream

9. **Gormes-owned Patterns** (from Gormes-owned)
   - Zero-LLM knowledge graph wiring (regex-based auto-links)
   - Brain-first lookup (5-step before external API)
   - **Rationale**: Significantly reduces LLM calls, compounds knowledge

10. **Token Budget System** (from Mercury)
    - Daily budget tracking with mutex-protected counter
    - Auto-concise at 70% threshold
    - Budget commands (`/budget`)
    - **Rationale**: Cost control for production deployments

11. **Credential + OAuth** (Hermes parity)
    - XDG-scoped token storage
    - Google OAuth flows
    - **Rationale**: Required for full provider parity

### Long-Term (Next 12 Months)

12. **Cathedral II Code Navigation** (from Gormes-owned)
    - Call-graph edges, two-pass retrieval
    - 5 commands: code-callers, code-callees, code-def, code-refs, query --near-symbol
    - **Rationale**: Code-aware agent capabilities

13. **Minions Job Queue** (from Gormes-owned)
    - Postgres-native BullMQ-inspired queue
    - Parent-child DAGs, stall detection, retry with backoff
    - **Rationale**: Background task durability, subagent coordination

14. **Soul/Personality System** (Phase 6)
    - User-owned markdown personality files
    - soul.md (heart), persona.md (face), taste.md (palate), heartbeat.md (breathing)
    - **Rationale**: Planned for gormes Phase 6; Mercury's implementation is reference

15. **Web Dashboard** (Hermes parity)
    - TypeScript/React web UI (Hermes has full web/ directory)
    - Session management, skills, chat, config, logs
    - **Rationale**: Hermes has 191K-line TUI gateway server

16. **Multi-Memory Backends** (from Honcho)
    - Turbopuffer, LanceDB vector store options
    - Redis caching layer
    - **Rationale**: Scale beyond single-node SQLite

### Not Recommended for Port

- Space's browser-first architecture (fundamentally different runtime model)
- Space's WebLLM (requires browser runtime)
- Mercury's Telegram org model (Gormes gateway channels are different abstraction)
- Picoclaw's ultra-lightweight constraints (different target: $10 hardware vs server deployment)
- Full Kubernetes ACP controllers (from references) - use local state machine instead

---

## Reference Implementation Priority

From `references/go-agent-os/`:

| Priority | Pattern | Source | Target |
|----------|---------|--------|--------|
| 1 | Retry-After parsing + backoff | `plandex/model_error.go` | `internal/hermes/retry.go` |
| 2 | Tool result truncation | `nanobot/pkg/agents/truncate.go` | `internal/tools/truncate.go` |
| 3 | Image token estimation | `nanobot/pkg/agents/tokencount.go` | `internal/hermes/image_routing.go` |
| 4 | Token budget tracker | `axe/internal/budget/budget.go` | `internal/hermes/budget.go` |
| 5 | Deterministic write queue | `engram/internal/mcp/write_queue.go` | `internal/goncho/writequeue.go` |
| 6 | SQLite/FTS5 memory schema | `engram/internal/store/store.go` | GONCHO enhancement |
| 7 | State machine transitions | `agentcontrolplane/task/state_machine.go` | Turn lifecycle |
| 8 | OAuth PKCE | `goclaw/internal/oauth/openai.go` | `internal/oauth/` |
| 9 | Tool declaration schema | `trpc-agent-go/tool/tool.go` | `internal/tools/tool.go` |
| 10 | Await-user-reply route | `trpc-agent-go/agent/await_user_reply.go` | `internal/gateway/routing.go` |

---

## Success Metrics

- **30 days**: Permission hardening shipped, provider parity >80%, browser harness doctor working
- **90 days**: Loop detection, structured memory (6 types), skill metadata, native prompt builder
- **6 months**: Context compression reconciled, Gormes-owned patterns (auto-links, brain-first), token budget
- **12 months**: Cathedral II, Minions queue, Soul system, Web dashboard, multi-memory backends

**Current parity: ~20-30% → Target: 80%+ within 12 months**
