---
title: "Gateway Module Roadmap"
aliases:
  - /building-gormes/modules/gateway/
---

# Gateway Module Roadmap

Generated from the single logical backlog. This page is a scoped review view; `progress.json` remains canonical.

**Module group:** Channel Gateway
**Module:** `gateway`
**Rows:** 161
**Status counts:** `complete`: 161 · `in_progress`: 0 · `planned`: 0
**Priority counts:** `P0`: 14 · `P1`: 52 · `P2`: 37 · `P3`: 3 · `P4`: 1 · `unset`: 54

## Phase 1 — The Dashboard

### 1.A — Core TUI

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | SSE reconnect |

### 5.X — Termux Runtime Compatibility

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `gateway` | Termux gateway foreground tmux lifecycle |

## Phase 2 — The Gateway

### 2.A — Tool Registry

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | In-process Go tool registry |
| `complete` | `unset` | `gateway` | Streamed tool_calls accumulation |
| `complete` | `unset` | `gateway` | Kernel tool loop |
| `complete` | `P1` | `gateway` | Coding-agent delegation tooling (codex/claude-code/opencode) |
| `complete` | `P1` | `gateway` | Coding-agent delegation: Phase 1 scaffold (internal/codingagents) |

### 2.B.5 — Session Context + Delivery Routing

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | Gateway session store + SessionSource parity |
| `complete` | `P2` | `gateway` | Gateway manual reset session-boundary hooks |
| `complete` | `P2` | `gateway` | Gateway session reset notification parity |
| `complete` | `P1` | `gateway` | Gateway slash-confirm session-boundary cleanup |
| `complete` | `unset` | `gateway` | SessionContext prompt injection |
| `complete` | `P0` | `gateway` | Hermes live-turn prompt assembly parity (channel-neutral) |
| `complete` | `P0` | `gateway` | Live-turn SOUL.md and project context wiring (channel-neutral) |
| `complete` | `P1` | `gateway` | Live-turn USER.md and MEMORY.md durable user context block (channel-neutral) |
| `complete` | `P1` | `gateway` | Live-turn timestamp + model/provider/session metadata block + self-help guidance (channel-neutral) |
| `complete` | `P1` | `gateway` | Hermes MEMORY_GUIDANCE stale-artifact exclusion refresh |
| `complete` | `P1` | `gateway` | Live-turn metadata production wiring (cmd/gormes -> Manager seams) |
| `complete` | `P0` | `gateway` | Gateway /title manual session title command |
| `complete` | `P0` | `gateway` | Session metadata manual-title protection flag |
| `complete` | `P1` | `gateway` | Gateway auto-title generation wiring |
| `complete` | `P1` | `gateway` | Gateway typing-action wiring during stream |
| `complete` | `P1` | `gateway` | Placeholder edit-failure fallback hardening |
| `complete` | `P1` | `gateway` | Gateway stream/tool trace formatting fixture matrix |
| `complete` | `P1` | `gateway` | Durable context ordering and frozen snapshot decision fixture |
| `complete` | `P1` | `gateway` | Live-turn model/tool guidance wiring |
| `complete` | `P2` | `gateway` | Gateway active-turn policy manifest closeout |
| `complete` | `P2` | `gateway` | Gateway conversational session metadata refresh |
| `complete` | `P2` | `gateway` | Gateway session token accounting parity |
| `complete` | `P1` | `gateway` | Gateway startup allowlist + weak credential guard |
| `complete` | `unset` | `gateway` | DeliveryRouter + --deliver target parsing |
| `complete` | `unset` | `gateway` | Gateway stream consumer for agent-event fan-out |
| `complete` | `P3` | `gateway` | Non-editable gateway progress/commentary send fallback |
| `complete` | `P0` | `gateway` | Gateway fresh-final eligibility helper |
| `complete` | `P1` | `gateway` | Gateway fresh-final send/delete fallback |
| `complete` | `P2` | `gateway` | Gateway message deduplicator bounded helper |
| `complete` | `P2` | `gateway` | Gateway inbound dedup key helper |
| `complete` | `P2` | `gateway` | Gateway inbound dedup manager binding |
| `complete` | `P1` | `gateway` | Cross-platform image/document MEDIA delivery routing |
| `complete` | `P2` | `gateway` | Cross-platform multi-image native batching |
| `complete` | `P1` | `gateway` | Gateway platform reconnect isolation + channel health limits |
| `complete` | `P1` | `gateway` | Gateway per-platform circuit breaker + /platform pause/resume/list command |
| `complete` | `P1` | `gateway` | Gateway /model interactive provider/model picker |
| `complete` | `P1` | `gateway` | Gateway memory monitor pressure policy |

### 2.B.12 — Channel-Neutral Native Runtime Adapter

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P0` | `gateway` | Channel-neutral native runtime turn adapter |
| `complete` | `P1` | `gateway` | Hermes gateway platform registry manifest |
| `complete` | `P1` | `gateway` | Bundled platform plugin manifest drift guard |
| `complete` | `P1` | `gateway` | Multimodal photo attachment passthrough |
| `complete` | `P1` | `gateway` | Hermes-style default prompt and image-path hints for inbound photos |

### 2.C — Thin Mapping Persistence

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | bbolt session resume |
| `complete` | `unset` | `gateway` | (platform, chat_id) -> session_id |

### 2.D — Cron / Scheduled Automations

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `gateway` | Cron no-agent script-only short-circuit |

### 2.E.0 — OS-AI Spine: Deterministic Subagent Runtime

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | Deterministic subagent runtime |
| `complete` | `unset` | `gateway` | Max-depth guard + bounded batch execution |
| `complete` | `unset` | `gateway` | Timeout + cancellation scopes |
| `complete` | `unset` | `gateway` | Typed result envelope |
| `complete` | `unset` | `gateway` | Append-only run log |

### 2.E.1 — OS-AI Spine: Delegation Policy + Child Execution

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | Runner-enforced tool allowlists + blocked-tool policy |
| `complete` | `unset` | `gateway` | Tool-call audit in typed child results |
| `complete` | `unset` | `gateway` | Real child Hermes stream loop |
| `complete` | `P2` | `gateway` | Durable subagent/job ledger |

### 2.E.2 — OS-AI Spine: Concurrent-Tool Cancellation

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | Interrupt propagation to concurrent-tool workers |

### 2.F.1 — Slash Command Registry + Gateway Dispatch

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | Canonical CommandDef registry |
| `complete` | `unset` | `gateway` | Gateway slash dispatch + per-platform exposure |
| `complete` | `P2` | `gateway` | Gateway slash registry parity sweep (recognized-name expansion) |
| `complete` | `P1` | `gateway` | Gateway /commands paginated command and skill catalog |

### 2.F.2 — Hook Registry + BOOT.md

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | Gateway per-event hook registry |
| `complete` | `unset` | `gateway` | Hook manifest discovery + handler loading |
| `complete` | `unset` | `gateway` | Built-in BOOT.md startup hook |

### 2.F.3 — Restart / Pairing / Status

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | Graceful restart drain + managed shutdown |
| `complete` | `unset` | `gateway` | Adapter startup failure cleanup contract |
| `complete` | `P1` | `gateway` | Gateway channel disconnect timeout on failed startup |
| `complete` | `P1` | `gateway` | Gateway shutdown capped adapter disconnect |
| `complete` | `unset` | `gateway` | Active-turn follow-up queue + late-arrival drain policy |
| `complete` | `P2` | `gateway` | Drain-timeout resume_pending recovery |
| `complete` | `unset` | `gateway` | Pairing read-model schema + atomic persistence |
| `complete` | `unset` | `gateway` | Pairing approval + rate-limit semantics |
| `complete` | `unset` | `gateway` | Unauthorized DM pairing response contract |
| `complete` | `P2` | `gateway` | `gormes gateway status` read-only command |
| `complete` | `P2` | `gateway` | Runtime status JSON + PID/process validation |
| `complete` | `P2` | `gateway` | Token-scoped gateway locks |
| `complete` | `P2` | `gateway` | Gateway /restart command + takeover markers |
| `complete` | `P1` | `gateway` | Gateway restart notification opt-out |
| `complete` | `P2` | `gateway` | Session expiry finalized-flag migration |
| `complete` | `P2` | `gateway` | Session expiry hook cleanup retry evidence |
| `complete` | `unset` | `gateway` | Channel lifecycle writers into status model |

### 2.F.4 — Home Channel + Operator Surfaces

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P0` | `gateway` | Home channel ownership resolver fixtures |
| `complete` | `P1` | `gateway` | Notify-to delivery routing |
| `complete` | `unset` | `gateway` | Channel directory atomic persistence + lookup |
| `complete` | `unset` | `gateway` | Channel directory refresh + stale-target invalidation |
| `complete` | `P1` | `gateway` | Manager remember-source hook |
| `complete` | `P2` | `gateway` | Mirror + sticker cache surfaces |
| `complete` | `P1` | `gateway` | Gateway delivery evidence in operator run report |

### 2.F.5 — Gateway Mid-Run Steering + Active-Turn Policy

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | Steer slash command parser + preview helper |
| `complete` | `unset` | `gateway` | Steer slash command registry + queue fallback |
| `complete` | `P0` | `gateway` | Mid-run steer injection between tool calls |
| `complete` | `P0` | `gateway` | Gateway-handled slash commands bypass active-session guard |
| `complete` | `P1` | `gateway` | Gateway persistent goal loop + continuation judge |
| `complete` | `P1` | `gateway` | Gateway/TUI /queue explicit FIFO slash parity |

### 2.G — OS-AI Spine: Skills Runtime

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | SKILL.md parsing + active store |
| `complete` | `unset` | `gateway` | Deterministic selection + prompt block |
| `complete` | `unset` | `gateway` | Kernel injection + usage log |
| `complete` | `unset` | `gateway` | Inactive candidate drafting |
| `complete` | `unset` | `gateway` | Explicit promotion flow |

### 2.H — Gormes-owned: Dynamic agents and per-thread spawn UX

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `gateway` | gormes agent spawn/list/inspect/bind/unbind CLI |

## Phase 3 — The Black Box (Memory)

### 3.E.8 — Session Lineage + Cross-Source Search

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | Gateway resume follows compression continuation |

## Phase 4 — The Brain Transplant

### 4.A — Provider Adapters

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | Bedrock stream event decoding (SSE fixtures) |

### 4.E — Trajectory + Insights

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `gateway` | Trajectory compressor + compressed-evidence lineage |

### 4.H — Rate / Retry / Caching

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | Gateway /usage command binding over provider account usage |

### 4.I — Native Agent Turn Closure

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P0` | `gateway` | Native runtime provider gateway binding |

## Phase 5 — The Final Purge

### 5.G — MCP Integration

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | Managed tool gateway bridge |

### 5.J — Approval / Security Guards

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `gateway` | Gateway approval FIFO queue resolver |
| `complete` | `P2` | `gateway` | Gateway hook auto-accept strict parser |
| `complete` | `P1` | `gateway` | Gateway allowed_chats/channels/rooms whitelist parity |

### 5.N — Misc Operator Tools

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `gateway` | Gateway probe auth/capability HTTP closeout |
| `complete` | `P1` | `gateway` | Gateway Discover and Probe |
| `complete` | `P0` | `gateway` | Multi-agent gateway runtime activation |
| `complete` | `P2` | `gateway` | Gateway auto-resume on restart |

### 5.O — Hermes CLI Parity

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `gateway` | Gateway /reasoning command parser |
| `complete` | `P2` | `gateway` | Gateway /reasoning apply + dispatch |
| `complete` | `P2` | `gateway` | Gormes setup gateway platform checklist command binding |
| `complete` | `unset` | `gateway` | Gateway management CLI read-model closeout |
| `complete` | `unset` | `gateway` | Gateway mutating-subcommand unavailability stub |
| `complete` | `P1` | `gateway` | Windows gateway Scheduled Task lifecycle commands |
| `complete` | `P1` | `gateway` | Windows detached gateway Ctrl+C boundary |
| `complete` | `P1` | `gateway` | Gormes update bundled assets and skills sync |
| `complete` | `P1` | `gateway` | Gateway planned stop marker + WSL systemd PATH parity |
| `complete` | `P1` | `gateway` | Gateway stale-code self-check uses git HEAD SHA |
| `complete` | `P1` | `gateway` | cmd/gormes gateway row-backed command package extraction |
| `complete` | `P1` | `gateway` | cmd/gormes live gateway command package extraction |

### 5.Q — API Server + TUI Gateway Streaming

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `gateway` | TUI gateway tool-progress mode normalizer |
| `complete` | `P2` | `gateway` | TUI gateway completion path normalizer |
| `complete` | `P2` | `gateway` | TUI gateway tool summary formatter |
| `complete` | `P0` | `gateway` | TUI gateway config health null-section probe |
| `complete` | `P2` | `gateway` | TUI launch model override + static alias resolver |
| `complete` | `P3` | `gateway` | TUI prompt-submit auto-title eligibility helper |
| `complete` | `unset` | `gateway` | TUI TerminalNativeSelectionHelp constant + help-string fixture |
| `complete` | `P3` | `gateway` | TUI running-agent placeholder surfaces interrupt + queued slash actions |
| `complete` | `P0` | `gateway` | Channel/TUI iteration-limit finalization transcript fixture |
| `complete` | `P1` | `gateway` | TUI websocket attach transport |
| `complete` | `unset` | `gateway` | OpenAI-compatible chat-completions API server |
| `complete` | `P1` | `gateway` | API server multimodal content preservation |
| `complete` | `unset` | `gateway` | Responses API store + run event stream |
| `complete` | `unset` | `gateway` | API server disconnect snapshot persistence |
| `complete` | `unset` | `gateway` | Gateway proxy mode forwarding contract |
| `complete` | `P1` | `gateway` | Gateway proxy replay assistant metadata preservation |
| `complete` | `unset` | `gateway` | Dashboard API client contract |
| `complete` | `unset` | `gateway` | Dashboard PTY chat sidecar contract |
| `complete` | `P2` | `gateway` | API server detailed health snapshot contract |
| `complete` | `P2` | `gateway` | API server detailed health endpoint |
| `complete` | `P2` | `gateway` | API server cron admin read-only endpoints |
| `complete` | `P2` | `gateway` | API server cron admin mutating endpoints |
| `complete` | `P0` | `gateway` | API server legacy jobs routes + default toolset |
| `complete` | `P2` | `gateway` | Provider client lazy-init for TUI cold-start budget |
| `complete` | `P2` | `gateway` | Kernel cross-provider client swap for in-session model switch |
| `complete` | `P1` | `gateway` | Hermes web dashboard strict-fidelity contract map |
| `complete` | `P1` | `gateway` | Gormes JSONL RPC mode over agent runtime events |

### 5.V — Unified Event Bus

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `gateway` | Gateway outbound sends publish message-sent events |
| `complete` | `P1` | `gateway` | Event bus integration test: full message flow |

## Phase 6 — The Learning Loop (Soul)

### 6.D — Skill Retrieval + Matching

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `gateway` | Delta-bounded skill and memory maintenance passes |

## Phase 7 — Paused Channel Backlog

### 7.E — Regional + Device Adapter Backlog

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P4` | `gateway` | Yuanbao gateway runtime + toolset registration |
