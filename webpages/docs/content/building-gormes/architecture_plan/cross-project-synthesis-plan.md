---
title: "Cross-Project Synthesis Plan"
weight: 20
---

# Cross-Project Synthesis Plan

**Date:** 2026-04-30
**Scope:** Synthesis of 12 opensource projects analyzed in `workspace-mineru`
**Goal:** Identify adoptable features, reference implementations, and roadmap additions for Gormes-Agent

---

## Projects Analyzed

| Project | Language | Lines | What It Is | Gormes Relevance |
|---------|----------|-------|------------|------------------|
| **hermes-agent** | Python | 1.2M+ | Upstream agent (Gormes parity target) | Canonical feature set to port |
| **honcho** | Python | ~15K | Memory/session platform (Honcho) | Memory architecture patterns |
| **gormes** | TypeScript | ~25K | Knowledge graph runtime | Auto-wiring, search, job queue |
| **browser-harness** | Python | ~5K | Browser automation (CDP) | Browser tool parity reference |
| **go-browser-harness** | Go | ~3K | Go browser automation port | Direct code donor |
| **mercury-agent** | TypeScript | ~8K | Soul-driven CLI/Telegram agent | Safety, memory, loop detection |
| **space-agent** | JavaScript | ~12K | Browser-first agent framework | Skill metadata, web UI |
| **picoclaw** | Go | ~20K | Ultra-lightweight agent | Channel adapters, provider patterns |
| **go-agent-os refs** | Go | ~50K | 8 donor repos (adk-go, nanobot, etc.) | Reusable patterns, interfaces |

---

## Top 20 Features to Adopt (Priority Order)

### P0 — Critical Path (Blocks Dogfood Milestone)

| # | Feature | Source | Gormes Phase | Effort | Rationale |
|---|---------|--------|--------------|--------|-----------|
| 1 | **Permission-hardened shell** | Mercury | 5.A (Tool Registry) | Medium | Zero safety today; 36+ blocklist patterns, cwdOnly, filesystem scoping |
| 2 | **Native prompt builder** | Hermes | 4.C | Large | `agent/prompt_builder.py` (1,122 lines) — critical for Python-free turn |
| 3 | **Context compression reconciliation** | Hermes | 4.B | Medium | Upstream `5006b220` drift; tool-result pruning, head/tail invariants |
| 4 | **Provider fallback chain** | Mercury + Picoclaw | 4.A | Small | DeepSeek → OpenAI → Anthropic → Grok → Ollama resilient routing |
| 5 | **Loop detection (5-type)** | Mercury | 5.A | Medium | Hard loop, failing loop, text repetition, no-action, same-tool |

### P1 — High Impact (Next 90 Days)

| # | Feature | Source | Gormes Phase | Effort | Rationale |
|---|---------|--------|--------------|--------|-----------|
| 6 | **Structured memory (typed)** | Mercury + Honcho | 6 | Large | 6 types: identity, preference, goal, habit, episode, reflection |
| 7 | **Skill metadata placement** | Space-agent | 6 | Small | `metadata.when`, `metadata.loaded`, `metadata.placement` in SKILL.md |
| 8 | **Browser harness doctor** | go-browser-harness | 5.B | Small | `go-browser-harness doctor` subcommand — production readiness |
| 9 | **Retry-After parsing + backoff** | Plandex (ref) | 4.H | Small | `plandex/model_error.go` pattern — classify + extract retry timing |
| 10 | **Tool result truncation** | Nanobot (ref) | 5.A | Small | 50 KiB cap, persist oversized output, sanitize paths |
| 11 | **Token budget tracker** | Axe (ref) + Mercury | 5.Q | Medium | Daily mutex-protected counter, auto-concise at 70% |
| 12 | **Zero-LLM knowledge graph wiring** | Gormes-owned | 6 | Large | Regex-based auto-links, brain-first 5-step lookup |

### P2 — Medium Term (6 Months)

| # | Feature | Source | Gormes Phase | Effort | Rationale |
|---|---------|--------|--------------|--------|-----------|
| 13 | **Credential + OAuth (XDG)** | Hermes + goclaw (ref) | 4.G | Medium | XDG-scoped token storage, Google OAuth PKCE |
| 14 | **Deterministic write queue** | Engram (ref) | 3/6 | Small | Serialized MCP/GONCHO writes, bounded channel, panic recovery |
| 15 | **Minions job queue** | Gormes-owned | 5/6 | Large | Postgres-native BullMQ-inspired, DAGs, stall detection |
| 16 | **Image token estimation** | Nanobot (ref) | 4.D | Small | Dimension-based token counting for image routing |
| 17 | **State machine transitions** | AgentControlPlane (ref) | 4.I | Medium | Turn lifecycle: admitted → planning → building → tool_calls → final |

### P3 — Long Term (12 Months)

| # | Feature | Source | Gormes Phase | Effort | Rationale |
|---|---------|--------|--------------|--------|-----------|
| 18 | **Cathedral II code navigation** | Gormes-owned | 5.J | Large | Call-graph edges, two-pass retrieval, 5 code commands |
| 19 | **Soul/personality system** | Mercury | 6 | Medium | soul.md (heart), persona.md (face), taste.md (palate), heartbeat.md |
| 20 | **Web dashboard (React)** | Hermes + Space | 5.Q | Large | Hermes has 191K-line TUI gateway; Space has browser-first widgets |

---

## Reference Implementation Quick Lookup

When implementing a row, consult these donor files before writing from scratch:

| Pattern | Donor File | Gormes Target | License |
|---------|------------|---------------|---------|
| OAuth PKCE | `goclaw/internal/oauth/openai.go` | `internal/oauth/` | CC BY-NC 4.0 (Juan-permitted) |
| Retry-After parse | `plandex/app/server/model/model_error.go` | `internal/hermes/retry.go` | Apache 2.0 |
| Tool truncation | `nanobot/pkg/agents/truncate.go` | `internal/tools/truncate.go` | Apache 2.0 |
| Token budget | `axe/internal/budget/budget.go` | `internal/hermes/budget.go` | Apache 2.0 |
| Write queue | `engram/internal/mcp/write_queue.go` | `internal/goncho/writequeue.go` | Apache 2.0 |
| SQLite/FTS5 schema | `engram/internal/store/store.go` | GONCHO enhancement | Apache 2.0 |
| State machine | `agentcontrolplane/acp/internal/controller/task/state_machine.go` | Turn lifecycle | Apache 2.0 |
| Tool declaration | `trpc-agent-go/tool/tool.go` | `internal/tools/tool.go` | Apache 2.0 |
| Await-user-reply | `trpc-agent-go/agent/await_user_reply.go` | `internal/gateway/routing.go` | Apache 2.0 |
| Provider error vocab | `goclaw/internal/oauth/openai_quota.go` | `internal/hermes/errors.go` | CC BY-NC 4.0 (Juan-permitted) |

---

## Hermes Parity Gap Register

### Critical Gaps (Blocking Dogfood)

| Hermes File | Lines | Gormes Status | Action |
|-------------|-------|---------------|--------|
| `agent/prompt_builder.py` | 1,122 | Not started | Port to `internal/hermes/prompt_builder.go` |
| `agent/context_compressor.py` | 1,414 | Partial | Reconcile upstream `5006b220` |
| `tools/terminal_tool.py` | 2,307 | Basic | Add PTY, shell blocklist, cwdOnly |
| `tools/code_execution_tool.py` | 1,609 | Not started | Sandboxed execution policy |
| `tools/mcp_tool.py` + OAuth | 3,140 + 22K | Partial | Full MCP client + OAuth flow |
| `hermes_cli/auth.py` | 4,744 | Partial | XDG token storage, OAuth flows |

### Large Gaps (Post-Dogfood)

| Hermes File | Lines | Gormes Status | Action |
|-------------|-------|---------------|--------|
| `tools/file_operations.py` | 50,764 | ~200 lines | Major expansion needed |
| `tools/web_tools.py` | 89,105 | Partial | Web scraping parity |
| `tools/vision_tools.py` | 31,333 | Not started | Vision pipeline |
| `tools/voice_mode.py` | 38,753 | Partial | Full voice mode |
| `tools/process_registry.py` | 61,200 | Basic | Process lifecycle |
| `tools/checkpoint_manager.py` | 31,551 | Basic | Checkpoint/restore |
| `gateway/run.py` | 12,215 | Partial | Gateway runtime |
| `gateway/session.py` | 54,707 | Partial | Session management |

### Massive Gaps (Long Term)

| Hermes File | Lines | Gormes Status | Action |
|-------------|-------|---------------|--------|
| `tools/skills_hub.py` | 118,874 | Basic | Full skills hub |
| `tui_gateway/server.py` | 191,064 | Minimal | TUI gateway server |
| `gateway/platforms/discord.py` | 173,474 | Basic | Full Discord parity |
| `gateway/platforms/telegram.py` | 141,434 | Basic | Full Telegram parity |
| `cron/scheduler.py` | 58,318 | Basic | Full scheduler |
| `cli.py` | 519,989 | Partial | CLI parity |

---

## Suggested New Progress.json Rows

Based on cross-project analysis, the following rows should be added to `progress.json`:

### Phase 4 — Brain Transplant

**4.J Permission-Hardened Tool Execution**
- `contract`: Shell blocklist, filesystem scoping, permission approval UX
- `source_refs`: `mercury-agent/src/core/permissions.ts`, `hermes-agent/tools/path_security.py`
- `write_scope`: `internal/tools/permissions.go`, `internal/tools/shell_blocklist.go`
- `test_commands`: `go test ./internal/tools -run TestPermission -count=1`
- `acceptance`: 36+ shell blocklist patterns, folder-level read/write scopes, inline y/n/always approval

**4.K Provider Fallback Chain**
- `contract`: DeepSeek → OpenAI → Anthropic → Grok → Ollama resilient routing
- `source_refs`: `mercury-agent/src/core/providers.ts`, `picoclaw/pkg/providers/`
- `write_scope`: `internal/hermes/fallback_chain.go`
- `test_commands`: `go test ./internal/hermes -run TestFallback -count=1`

### Phase 5 — Final Purge

**5.R Loop Detection**
- `contract`: 5-type loop detector (hard, failing, text repetition, no-action, same-tool)
- `source_refs`: `mercury-agent/src/core/loop_detection.ts`
- `write_scope`: `internal/agent/loop_detector.go`
- `test_commands`: `go test ./internal/agent -run TestLoopDetection -count=1`

**5.S Browser Harness Doctor**
- `contract`: `go-browser-harness doctor` subcommand with endpoint health checks
- `source_refs`: `go-browser-harness/docs/parity-matrix.md`
- `write_scope`: `go-browser-harness/cmd/doctor.go`
- `test_commands`: `go test ./go-browser-harness -run TestDoctor -count=1`

### Phase 6 — Learning Loop

**6.D Structured Memory Types**
- `contract`: 6 typed memory categories with confidence/durability scoring
- `source_refs`: `mercury-agent/src/core/memory.ts`, `honcho/src/models.py`
- `write_scope`: `internal/goncho/typed_memory.go`
- `test_commands`: `go test ./internal/goncho -run TestTypedMemory -count=1`

**6.E Skill Metadata Placement**
- `contract`: `metadata.when`, `metadata.loaded`, `metadata.placement` in SKILL.md schema
- `source_refs`: `space-agent/src/skills/schema.js`
- `write_scope`: `internal/skills/metadata.go`
- `test_commands`: `go test ./internal/skills -run TestMetadata -count=1`

**6.F Zero-LLM Knowledge Graph**
- `contract`: Regex-based auto-link extraction, brain-first 5-step lookup
- `source_refs`: `gormes/src/core/link-extraction.ts`, `gormes/src/core/search/hybrid.ts`
- `write_scope`: `internal/goncho/auto_link.go`, `internal/goncho/brain_first.go`
- `test_commands`: `go test ./internal/goncho -run TestAutoLink -count=1`

---

## Not Recommended for Port

| Feature | Source | Why Not |
|---------|--------|---------|
| Browser-first runtime | Space-agent | Fundamentally different model (agent in browser vs Go backend) |
| WebLLM | Space-agent | Requires browser runtime; Gormes is Go-native |
| Ultra-lightweight (<10MB) | Picoclaw | Different target (server deployment vs $10 hardware) |
| Kubernetes ACP controllers | AgentControlPlane (ref) | Use local state machine first |
| Full TUI gateway server (191K lines) | Hermes | Too large; build incrementally |
| Telegram org model | Mercury | Gormes gateway channels are different abstraction |

---

## Success Metrics

| Horizon | Metric | Target |
|---------|--------|--------|
| 30 days | Permission hardening | 36+ blocklist patterns, filesystem scoping shipped |
| 30 days | Provider parity | >80% of Hermes providers implemented |
| 30 days | Browser harness doctor | `go-browser-harness doctor` works |
| 90 days | Loop detection | 5-type detector with tests |
| 90 days | Structured memory | 6 typed categories with confidence scoring |
| 90 days | Skill metadata | `metadata.when/loaded/placement` in SKILL.md |
| 90 days | Native prompt builder | `internal/hermes/prompt_builder.go` shipped |
| 6 months | Context compression | Reconciled with upstream `5006b220` |
| 6 months | Gormes-owned patterns | Auto-links + brain-first lookup working |
| 6 months | Token budget | Daily budget + auto-concise at 70% |
| 12 months | Cathedral II | Code navigation with call-graph edges |
| 12 months | Minions queue | Postgres-native job queue with DAGs |
| 12 months | Soul system | Personality files (soul.md, persona.md, etc.) |
| 12 months | Web dashboard | React-based session/skills/chat UI |

**Current parity: ~20-30% → Target: 80%+ within 12 months**

---

## Next Actions

1. **Run `gormes-planner`** to add the suggested rows above to `progress.json`
2. **Run `gormes-parity-auditor`** on Hermes `agent/prompt_builder.py`, `tools/terminal_tool.py`, `tools/code_execution_tool.py`
3. **Start builder pass** on `4.J Permission-Hardened Tool Execution` (highest impact, smallest scope)
4. **Update completion-plan.md** with cross-project synthesis as a reference surface
