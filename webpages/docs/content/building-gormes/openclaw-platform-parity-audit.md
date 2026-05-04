---
title: "OpenClaw Platform Parity Audit"
description: "Complete feature gap analysis between OpenClaw 2026.3.28 platform and gormes-agent. Every OpenClaw config section mapped to gormes-agent status."
weight: 29
---

# OpenClaw Platform Parity Audit

> **Source:** OpenClaw 2026.3.28 config schema, CLI surface, state directory
> **Audience:** gormes-agent planner, builder, and reviewer skills.
> **Purpose:** Exhaustive gap analysis to inform Phase 5 (Final Purge) and Phase 6 (Learning Loop) planning.

---

## 1. Agent Configuration Gaps

OpenClaw's agent defaults surface has extensive configuration that gormes-agent lacks:

| OpenClaw Feature | Description | gormes-agent Status |
|---|---|---|
| `agents.defaults.memorySearch` | Multimodal + remote + local + chunking + sync + query + cache | Partial — Goncho has semantic search but no multimodal/chunking/sync config |
| `agents.defaults.contextPruning.tools` | Tool result pruning with policy | Shipped (Phase 4 compression) |
| `agents.defaults.contextPruning.softTrim` | Soft context trimming | Shipped |
| `agents.defaults.contextPruning.hardClear` | Hard context clearing | Gap |
| `agents.defaults.compaction.qualityGuard` | Compaction quality validation | Shipped |
| `agents.defaults.compaction.memoryFlush` | Memory flush on compaction | Gap |
| `agents.defaults.embeddedPi` | Embedded coding agent (Pi) | Gap |
| `agents.defaults.blockStreamingChunk` | Block streaming chunk config | Partial |
| `agents.defaults.blockStreamingCoalesce` | Block streaming coalesce config | Partial |
| `agents.defaults.humanDelay` | Simulated human typing delay | Gap |
| `agents.defaults.heartbeat.activeHours` | Active hours for heartbeat scheduling | Gap |
| `agents.defaults.subagents` | Subagent configuration | Partial (flat goroutine pool) |
| `agents.defaults.sandbox.docker` | Docker sandbox config | In-flight (Phase 5.B) |
| `agents.defaults.sandbox.ssh` | SSH sandbox config | Gap |
| `agents.defaults.sandbox.browser` | Browser sandbox config | Gap |
| `agents.defaults.sandbox.prune` | Sandbox prune policy | Gap |

---

## 2. Tool Configuration Gaps

| OpenClaw Feature | Description | gormes-agent Status |
|---|---|---|
| `tools.web.search` | Multi-provider web search (Brave, Firecrawl, Gemini, Grok, Kimi, Perplexity) | Shipped (9 backends with auto-routing) |
| `tools.web.fetch` | Web page fetching | Shipped |
| `tools.web.x_search` | Cross-provider search aggregation | Gap |
| `tools.media.image` | Image scope + deepgram + attachments | Partial (image generation shipped, vision gap) |
| `tools.media.audio` | Audio scope + deepgram + attachments | Partial (transcription shipped, scope gap) |
| `tools.media.video` | Video scope + deepgram + attachments | Gap |
| `tools.links` | Link handling with scope | Gap |
| `tools.sessions` | Session tools | Shipped |
| `tools.loopDetection.detectors` | 5-type loop detection with configurable detectors | Gap (must-have listed, not implemented) |
| `tools.message.crossContext` | Cross-context messaging with marker | Gap |
| `tools.message.broadcast` | Message broadcast | Partial (cron multi-target shipped) |
| `tools.agentToAgent` | Agent-to-agent communication | Gap |
| `tools.elevated` | Elevated/privileged tools | In-flight (approval mode) |
| `tools.exec.applyPatch` | Apply patch tool | In-flight |
| `tools.fs` | Filesystem tools config | Partial (basic file ops shipped) |
| `tools.subagents.tools` | Subagent tool management | Partial |
| `tools.sandbox.tools` | Sandbox tool management | Gap |
| `tools.sessions_spawn.attachments` | Session spawn with attachments | Gap |

---

## 3. Session Management Gaps

| OpenClaw Feature | Description | gormes-agent Status |
|---|---|---|
| `session.reset` | Session reset | Shipped |
| `session.resetByType.direct` | Direct message session reset | Gap (all sessions use same reset) |
| `session.resetByType.dm` | DM session reset | Gap |
| `session.resetByType.group` | Group session reset | Gap |
| `session.resetByType.thread` | Thread session reset | Gap |
| `session.sendPolicy` | Per-session send policy | Gap |
| `session.agentToAgent` | Agent-to-agent session routing | Gap |
| `session.threadBindings` | Thread binding management | Gap |
| `session.maintenance` | Session maintenance/cleanup | Shipped (session expiry) |

---

## 4. Gateway & Infrastructure Gaps

| OpenClaw Feature | Description | gormes-agent Status |
|---|---|---|
| `gateway.controlUi` | Control UI configuration | Shipped (htmx dashboard) |
| `gateway.auth` | Gateway authentication | Shipped |
| `gateway.auth.rateLimit` | Auth rate limiting | Gap |
| `gateway.auth.trustedProxy` | Trusted proxy configuration | Gap |
| `gateway.tools` | Gateway-level tool configuration | Gap |
| `gateway.tailscale` | Tailscale integration | Out of scope |
| `gateway.remote` | Remote gateway connection | Gap |
| `gateway.reload` | Hot reload configuration | Gap |
| `gateway.tls` | TLS configuration | Gap |
| `gateway.http.endpoints.chatCompletions` | OpenAI-compatible chat completions | Shipped |
| `gateway.http.endpoints.responses` | Responses API | Shipped |
| `gateway.http.securityHeaders` | Security headers | Gap |
| `gateway.push.apns` | Apple Push Notification service relay | Out of scope |
| `gateway.nodes.browser` | Node browser management | Gap |

---

## 5. Memory System Gaps

| OpenClaw Feature | Description | gormes-agent Status |
|---|---|---|
| `memory.qmd` | QMD hybrid search configuration | Gap (P1 planned) |
| `memory.qmd.mcporter` | MCP porter integration | Gap |
| `memory.qmd.sessions` | Session indexing | Partial |
| `memory.qmd.update` | Index update config | Gap |
| `memory.qmd.limits` | Index size limits | Gap |
| `memory.qmd.scope` | Search scope configuration | Gap |

---

## 6. Plugin Ecosystem (80+ plugins)

OpenClaw's plugin architecture supports 80+ provider, channel, and tool plugins, each with hooks, subagent activation, and config. gormes-agent has a basic plugin system (`internal/plugins/`) with only first-party plugins (Spotify, Google Meet).

**Key plugin categories gormes-agent should match:**

| Category | OpenClaw Plugins | gormes-agent Status |
|---|---|---|
| **Model Providers** | openai, anthropic, google, microsoft, amazon-bedrock, deepseek, mistral, groq, ollama, xai, together, sglang, vllm, etc. (20+) | Shipped in hermes package |
| **Channels** | telegram, discord, slack, whatsapp, signal, matrix, irc, line, msteams, etc. (15+) | Shipped in channels package |
| **Search** | brave, firecrawl, tavily, exa, duckduckgo, perplexity, kimi | Shipped |
| **Memory** | memory-core, memory-lancedb | Goncho (SQLite only) |
| **Infrastructure** | diagnostics-otel, device-pair, diffs, thread-ownership | Gap |
| **Voice/Audio** | talk-voice, voice-call, elevenlabs, deepgram | Partial |
| **Special** | phone-control, embedded-pi, openshell, open-prose, lobste, synthetic | Gap |

---

## 7. Additional CLI Commands (Not in gormes-agent)

| OpenClaw Command | Description | Priority |
|---|---|---|
| `openclaw directory` | Contact/group ID lookup for channels | P3 |
| `openclaw dns` | DNS helpers for Tailscale + CoreDNS | Out of scope |
| `openclaw message` | Send, read, manage messages directly | P3 |
| `openclaw nodes` | Gateway-owned node pairing and commands | P3 |
| `openclaw qr` | iOS pairing QR generation | Out of scope |
| `openclaw reset` | Reset local config/state | P2 |

---

## 8. Summary: Critical Parity Gaps

### P0 — Must Match for Production Parity

| Gap | OpenClaw Feature | Phase |
|-----|-----------------|-------|
| Loop detection | `tools.loopDetection.detectors` — 5 types with configurable detectors | 5.N |
| Session reset by channel type | `session.resetByType` — different policies for DM/group/thread | 5.N |
| Gateway hot reload | `gateway.reload` — no-restart config changes | 5.O |
| Gateway TLS | `gateway.tls` — HTTPS support | 5.Q |

### P1 — Should Match for Operational Parity

| Gap | OpenClaw Feature | Phase |
|-----|-----------------|-------|
| Memory QMD config | `memory.qmd` — hybrid search with scope/limits/update | 5.N |
| Hard context clear | `agents.defaults.contextPruning.hardClear` | 4 |
| Compaction memory flush | `agents.defaults.compaction.memoryFlush` | 4 |
| Human delay simulation | `agents.defaults.humanDelay` | 5.E |
| Heartbeat active hours | `agents.defaults.heartbeat.activeHours` | 5.N |
| SSH sandbox | `agents.defaults.sandbox.ssh` | 5.B |
| Browser sandbox | `agents.defaults.sandbox.browser` | 5.B |
| Sandbox pruning | `agents.defaults.sandbox.prune` | 5.B |
| Agent-to-agent tool | `tools.agentToAgent` | 5.M |
| Cross-context messaging | `tools.message.crossContext` | 5.N |
| Video media tools | `tools.media.video` | 5.D |
| Gateway security headers | `gateway.http.securityHeaders` | 5.Q |
| Gateway node browser | `gateway.nodes.browser` | 5.C |

### P2 — Differentiators

| Gap | OpenClaw Feature | Phase |
|-----|-----------------|-------|
| Embedded Pi coding agent | `agents.defaults.embeddedPi` | 6 |
| Cross-provider search aggregation | `tools.web.x_search` | 5.A |
| Link handling with scope | `tools.links` | 5.N |
| Session spawn with attachments | `tools.sessions_spawn.attachments` | 5.M |
| Gateway remote connection | `gateway.remote` | 5.N |

---

*Generated: May 1, 2026*
*Source: OpenClaw 2026.3.28 config schema and CLI surface*
*Cross-referenced: must-have-features.md, fleet-operational-patterns.md*
