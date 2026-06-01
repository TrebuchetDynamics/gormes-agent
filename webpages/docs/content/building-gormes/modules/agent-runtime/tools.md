---
title: "Tools Module Roadmap"
aliases:
  - /building-gormes/modules/tools/
---

# Tools Module Roadmap

Generated from the single logical backlog. This page is a scoped review view; `progress.json` remains canonical.

**Module group:** Agent Runtime
**Module:** `tools`
**Rows:** 148
**Status counts:** `complete`: 147 · `in_progress`: 0 · `planned`: 1
**Priority counts:** `P0`: 23 · `P1`: 54 · `P2`: 35 · `P3`: 8 · `P4`: 2 · `unset`: 26

## Phase 3 — The Black Box (Memory)

### 3.E.2 — Tool Execution Audit Log

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `tools` | Append-only JSONL writer + schema |
| `complete` | `unset` | `tools` | Kernel + delegate_task audit hooks |
| `complete` | `unset` | `tools` | Outcome, duration, and error capture |

### 3.E.3 — Transcript Export Command

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `tools` | Render turns, tool calls, and timestamps from SQLite |

## Phase 4 — The Brain Transplant

### 4.B — Context Engine + Compression

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `tools` | ContextEngine interface + status tool contract |
| `complete` | `P1` | `tools` | Aux compression headroom for system and tool schemas |
| `complete` | `unset` | `tools` | Tool-result pruning + protected head/tail summary |

### 4.J — Permission-Hardened Tool Execution

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P0` | `tools` | Shell blocklist + filesystem scoping + permission approval |

### 4.L — Safety-Anchored Turn Loop (MOSAIC)

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `tools` | Tool gate pre-execution validation |

## Phase 5 — The Final Purge

### 5.A — Tool Surface Port

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `tools` | 61-tool registry port |
| `complete` | `unset` | `tools` | Tool registry inventory + schema parity harness |
| `complete` | `P1` | `tools` | Tool-call JSON-string array/object coercion parity |
| `complete` | `P1` | `tools` | Tool parity manifest refresh for Hermes b35d692f |
| `complete` | `P1` | `tools` | Tool parity manifest refresh for Hermes ea86714 computer_use |
| `complete` | `P1` | `tools` | Tool parity manifest refresh for Hermes 524cbabd patch schema |
| `complete` | `P2` | `tools` | Microsoft Graph auth/client helper parity |
| `complete` | `P2` | `tools` | Home Assistant HASS_TOKEN platform-toolset carveout |
| `complete` | `P0` | `tools` | Home Assistant tool handlers + service safety validation |
| `complete` | `unset` | `tools` | Pure core tools first |
| `complete` | `P1` | `tools` | Terminal process watch notification throttle contract |
| `complete` | `P1` | `tools` | Tool output budget persisted artifact pointer |
| `complete` | `P0` | `tools` | Tool descriptor layer (OperationSpec) |
| `complete` | `P1` | `tools` | Hermes tool tail strict-fidelity source-pair expansion |

### 5.B — Sandboxing Backends

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `tools` | Environment interface + file sync contract |
| `complete` | `P4` | `tools` | Terminal snapshot source stdout suppression guard |
| `complete` | `P3` | `tools` | Terminal deleted-cwd recovery guard |
| `complete` | `unset` | `tools` | Raw tool-call parser fixture matrix |
| `complete` | `P2` | `tools` | Modal |
| `complete` | `P2` | `tools` | Daytona |
| `complete` | `P2` | `tools` | Singularity command/preflight contract |
| `complete` | `P1` | `tools` | Sandbox Policy Explain |

### 5.D — Vision + Image Generation

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `tools` | Multimodal in/out |
| `complete` | `P2` | `tools` | Image input mode router + native content parts |
| `complete` | `P1` | `tools` | vision_analyze native multimodal tool-result path |
| `complete` | `P2` | `tools` | Image-too-large shrink retry helper |
| `complete` | `P3` | `tools` | Image generation result contract |
| `complete` | `P1` | `tools` | FAL image generation queue REST binding |
| `complete` | `P2` | `tools` | Native video_analyze tool contract |

### 5.G — MCP Integration

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `tools` | MCP client |
| `complete` | `unset` | `tools` | MCP stdio transport + tool/list discovery |
| `complete` | `unset` | `tools` | MCP HTTP transport + tool/list discovery |
| `complete` | `unset` | `tools` | MCP schema normalization + structured-content adapter |
| `complete` | `P1` | `tools` | MCP circuit breaker cooldown + reconnect reset |
| `complete` | `P2` | `tools` | MCP stdio orphan cleanup after cron ticks |
| `complete` | `P1` | `tools` | Gormes-native MCP host runtime boundary |

### 5.H — ACP Integration

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `tools` | ACP server side |
| `complete` | `P1` | `tools` | ACP Client Bridge Mode |
| `complete` | `P0` | `tools` | ACP JSON-RPC stdio session/prompt closeout |
| `complete` | `P1` | `tools` | ACP stdio benign ping/probe suppression |
| `complete` | `P0` | `tools` | ACP session CWD propagation into prompt runners |
| `complete` | `P2` | `tools` | ACP setup-browser bootstrap parity |

### 5.J — Approval / Security Guards

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `tools` | Dangerous action gating |
| `complete` | `unset` | `tools` | Hardline command pattern table + DetectHardline function |
| `complete` | `unset` | `tools` | Recoverable dangerous patterns + blocked-result schema |
| `complete` | `P1` | `tools` | delegate_task batch JSON-string task recovery |
| `complete` | `P2` | `tools` | Subagent dangerous-command non-interactive approval policy |
| `complete` | `P2` | `tools` | Concurrent tool approval callback propagation |
| `complete` | `P3` | `tools` | Background review toolset restriction |
| `complete` | `P2` | `tools` | Cron dangerous-command approval mode |
| `complete` | `P3` | `tools` | Tirith external security finding ingestion |
| `complete` | `P3` | `tools` | Unified security guard decision composer |
| `complete` | `P0` | `tools` | Shell blocklist (36+ dangerous patterns) |
| `complete` | `P0` | `tools` | Filesystem scoping (folder-level read/write restrictions) |
| `complete` | `P0` | `tools` | Permission approval UX (inline y/n/always) |
| `complete` | `P0` | `tools` | Trust-class enforcement in shared tool executor |
| `complete` | `P0` | `tools` | Security Audit Command |
| `complete` | `P0` | `tools` | Auth state TOCTOU close + redaction default-on parity |

### 5.K — Code Execution

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `tools` | Sandboxed exec |

### 5.L — File Ops + Patches

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `tools` | Atomic file write helper with temp+rename pattern |
| `complete` | `P2` | `tools` | File tool atomic checkpoint integration |
| `complete` | `P2` | `tools` | Checkpoints CLI (status/list/prune/clear/clear-legacy) |
| `complete` | `P2` | `tools` | Checkpoint shadow-repo GC policy |
| `complete` | `P2` | `tools` | File read dedup cache invalidation and wrapper guard |
| `complete` | `P2` | `tools` | File read repeated-stub BLOCKED escalation |
| `complete` | `P1` | `tools` | Native file task tool surface |
| `complete` | `P2` | `tools` | V4A patch mode for native patch tool |
| `complete` | `P2` | `tools` | V4A move operation for native patch tool |
| `complete` | `P0` | `tools` | Symlink-preserving atomic writer helper |
| `complete` | `P0` | `tools` | File write/patch staleness registry + cwd tracking |
| `complete` | `P0` | `tools` | Terminal deleted-cwd recovery |
| `complete` | `P1` | `tools` | search_files hidden-root and context-line parsing drift |
| `complete` | `P1` | `tools` | Structured lint delta for native write/patch tools |
| `complete` | `P1` | `tools` | Python syntax lint delta for native write/patch tools |
| `complete` | `P1` | `tools` | Shell lint delta for native write/patch tools |
| `complete` | `P1` | `tools` | Patch replace no-match did-you-mean hint |
| `complete` | `P1` | `tools` | Core fuzzy replace strategies for native patch tool |
| `complete` | `P1` | `tools` | Unicode-normalized fuzzy replace for native patch tool |
| `complete` | `P1` | `tools` | Block-anchor fuzzy replace for native patch tool |
| `complete` | `P1` | `tools` | V4A fuzzy hunk matching for native patch tool |
| `complete` | `P1` | `tools` | Context-aware fuzzy replace for native patch tool |
| `complete` | `P1` | `tools` | V4A patch apply rollback for native patch tool |
| `complete` | `P1` | `tools` | Patch replace post-write verification |
| `complete` | `P1` | `tools` | Hermes LSP write-time semantic diagnostics |
| `complete` | `P1` | `tools` | Per-file mutation queue for native write edit and patch tools |

### 5.N — Misc Operator Tools

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `unset` | `tools` | Todo |
| `complete` | `unset` | `tools` | Clarify |
| `complete` | `P3` | `tools` | Session search tool schema and argument validation |
| `complete` | `P3` | `tools` | Session search tool execution wrapper |
| `complete` | `P2` | `tools` | Session shutdown memory transcript handoff |
| `complete` | `unset` | `tools` | Debug helpers |
| `complete` | `P4` | `tools` | Debug share paste sweep scheduler contract |
| `complete` | `P1` | `tools` | Backend usage-limit stdin health bypass |
| `complete` | `unset` | `tools` | Cronjob tool API + schedule parser parity |
| `complete` | `P2` | `tools` | Cron schedule parser + repeat state fixtures |
| `complete` | `P2` | `tools` | Cron recurring next-run failure preservation |
| `complete` | `P2` | `tools` | Cron prompt/script safety + pre-run script contract |
| `complete` | `P1` | `tools` | Cron GitHub auth-header scanner parity |
| `complete` | `P2` | `tools` | Cronjob tool action envelope over native store |
| `complete` | `P2` | `tools` | Cron run resource release contract |
| `complete` | `P2` | `tools` | Cron run resource release executor binding |
| `complete` | `unset` | `tools` | Cron context_from output chaining |
| `complete` | `unset` | `tools` | Cron prompt/script safety + pre-run script contract (deprecated umbrella) |
| `complete` | `unset` | `tools` | Cron multi-target delivery + media/live-adapter fallback |
| `complete` | `P1` | `tools` | Cron deliver=all routing intent expansion |
| `complete` | `P0` | `tools` | Blocker Policy Integration |
| `complete` | `P0` | `tools` | OpenClaw SecretRef core resolver |
| `complete` | `P0` | `tools` | SecretRef runtime snapshot activation |
| `complete` | `P0` | `tools` | OpenClaw security audit --deep --fix |
| `complete` | `P0` | `tools` | Safety-critical panic and swallowed-error closeout |
| `complete` | `P0` | `tools` | Session Health Monitoring |
| `complete` | `P0` | `tools` | Evidence-Before-Claims Quality Gate |
| `complete` | `P1` | `tools` | Git Delivery Contract Enforcement |
| `complete` | `P1` | `tools` | QMD Hybrid Search |
| `complete` | `P1` | `tools` | Session Rollover Automation |
| `complete` | `P1` | `tools` | Channels Capabilities Introspection |
| `complete` | `P2` | `tools` | Prompt Fragment Include System |
| `complete` | `P0` | `tools` | Multi-agent auth and tool-policy runtime isolation |
| `complete` | `P1` | `tools` | Cron env-ref expansion + parallel run state serialization |
| `complete` | `P1` | `tools` | Cron origin delivery isolation from session identity |
| `complete` | `P0` | `tools` | Cron script/workdir/inactivity execution binding |
| `complete` | `P1` | `tools` | Cron dashboard partial-record page |
| `complete` | `P1` | `tools` | Hermes x_search tool and auth surface |
| `planned` | `P1` | `tools` | Hermes send_message tool list and target contract |

### 5.O — Hermes CLI Parity

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `tools` | CLI terminal control-response sanitizer |
| `complete` | `P2` | `tools` | Gormes mcp login interface seam + noninteractive default |
| `complete` | `P2` | `tools` | Oneshot noninteractive safety and clarify policy |
| `complete` | `P1` | `tools` | Platform toolset mixed composite runtime expansion |

### 5.R — Code Execution Mode Policy

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P2` | `tools` | Strict-mode CWD + interpreter parity |
| `complete` | `P2` | `tools` | Project-mode CWD + active venv detection |

### 5.U — Fault-Tolerant Sandbox Execution

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `tools` | Pre-execution command classification |
| `complete` | `P1` | `tools` | Transactional tool execution with snapshot/rollback |
| `complete` | `P3` | `tools` | Sandbox isolation depth selection |

### 5.V — Unified Event Bus

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `tools` | Agent turn and tool execution events on bus |

## Phase 6 — The Learning Loop (Soul)

### 6.J — Agentic Memory Lifecycle (AgeMem)

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `tools` | Memory operations as agent-callable tools |

## Phase 8 — Reputation & Publication

### 8.F — Cost Discipline & Loop Economics

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `tools` | Internal tool compact helper package rehome |
| `complete` | `P1` | `tools` | Internal tool trace helper package rehome |
| `complete` | `P1` | `tools` | Internal session search tool package rehome |

## Phase 9 — Design & Security Hardening

### 9.G — External Issue Radar Regression Guards

| Status | Priority | Module | Row |
|---|---|---|---|
| `complete` | `P1` | `tools` | MCP Streamable HTTP session lifecycle compatibility |
