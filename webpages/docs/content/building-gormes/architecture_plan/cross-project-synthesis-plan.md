---
title: "Cross-Project Synthesis Plan"
weight: 20
---

# Cross-Project Synthesis Plan

**Date:** 2026-04-30  
**Scope:** Active donor projects after retiring the knowledge-runtime donor from public Gormes planning.  
**Goal:** Identify adoptable features, reference implementations, and roadmap inputs for Gormes-Agent without reintroducing retired upstream dependencies.

## Projects Analyzed

| Project | Language | What It Is | Gormes Relevance |
|---|---|---|---|
| **hermes-agent** | Python | Upstream agent runtime | Canonical parity target |
| **honcho** | Python | Memory/session platform | Goncho compatibility target |
| **browser-harness** | Python | Browser automation | Browser parity reference |
| **go-browser-harness** | Go | Browser automation port | Go-native browser tool patterns |
| **mercury-agent** | TypeScript | CLI/Telegram agent | Safety, memory, loop detection |
| **space-agent** | JavaScript | Browser-first agent framework | Skill metadata and web UX patterns |
| **picoclaw** | Go | Lightweight agent | Channel and provider patterns |
| **go-agent-os refs** | Go | Donor repositories | OAuth, retry, tools, state, and store patterns |

## Priority Features

| Priority | Feature | Source | Gormes phase | Rationale |
|---|---|---|---|---|
| P0 | Permission-hardened shell | Mercury + Hermes | 5.A | Block dangerous shell/file behavior before handler execution. |
| P0 | Native prompt builder | Hermes | 4.C | Required for a Python-free turn with stable prompt layers. |
| P0 | Context compression reconciliation | Hermes | 4.B | Preserve head/tail and tool-result invariants. |
| P1 | Provider fallback chain | Mercury + Picoclaw | 4.A | Improve routing resilience across provider failures. |
| P1 | Retry-after parsing | Plandex ref | 4.H | Make rate-limit behavior observable and retryable. |
| P1 | Browser harness doctor | go-browser-harness | 5.B | Prove browser runtime readiness locally. |
| P2 | Structured memory types | Honcho + Mercury | 6 | Strengthen Goncho memory shape and provenance. |
| P2 | Skill metadata placement | Space Agent + Hermes | 6 | Keep skill loading/routing explicit and testable. |
| P2 | Durable write queue | Engram ref | 3/6 | Serialize local writes and expose queue health. |
| P3 | Web dashboard | Hermes + Space Agent | 5.Q | Surface sessions, gateway state, skills, and logs. |

## Reference Implementation Quick Lookup

| Pattern | Donor file | Gormes target |
|---|---|---|
| OAuth PKCE | `goclaw/internal/oauth/openai.go` | `internal/oauth/` |
| Retry-after parse | `plandex/app/server/model/model_error.go` | `internal/provider` |
| Tool truncation | `nanobot/pkg/agents/truncate.go` | `internal/tools` |
| Token budget | `axe/internal/budget/budget.go` | `internal/budget` |
| Write queue | `engram/internal/mcp/write_queue.go` | `internal/goncho`, `internal/subagent` |
| SQLite/FTS5 schema | `engram/internal/persistence/store/store.go` | Goncho storage |
| State machine | `agentcontrolplane/acp/internal/controller/task/state_machine.go` | Turn lifecycle |
| Tool declaration | `trpc-agent-go/tool/tool.go` | Tool descriptor layer |
| Await-user-reply | `trpc-agent-go/agent/await_user_reply.go` | Gateway routing |

## Roadmap Handling

Do not add broad rows from this document directly. Each adoption must go
through `gormes-planner` or `gormes-parity-auditor`, cite the exact Hermes,
Honcho, or Go-donor source, and become a small builder-ready `progress.json`
row with focused tests.

## Success Metrics

| Horizon | Metric | Target |
|---|---|---|
| 30 days | Permission hardening | Shell blocklist and filesystem scopes shipped |
| 30 days | Browser diagnostics | Browser doctor evidence available |
| 90 days | Loop detection | 5-type detector with replay fixtures |
| 90 days | Structured memory | Typed categories with confidence/provenance |
| 6 months | Context compression | Reconciled with current Hermes behavior |
| 6 months | Durable jobs | Queue health, replay/inbox, and worker status evidence |
| 12 months | Dashboard | React session/skills/gateway UI |
