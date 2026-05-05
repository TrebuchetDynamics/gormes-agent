---
title: "Fleet Operational Patterns — Sages Ecosystem Learnings"
description: "Cross-fleet analysis of the 6-agent sages-openclaw ecosystem plus OpenClaw platform. Operational patterns, infrastructure, and platform capabilities that gormes-agent should absorb."
weight: 26
---

# Fleet Operational Patterns — Sages Ecosystem Learnings

> **Source scope:** All 6 sages-openclaw workspace agents (Link, Mineru, Riju, Sidon, Tulin, Yunobo) plus OpenClaw 2026.3.28 platform.
> **Audience:** gormes-agent planner, builder, and reviewer skills.
> **Relationship to existing docs:** The [Cross-Project Feature Map](../cross-project-feature-map/) covers 12 external open-source projects (hermes-agent, honcho, gormes, mercury-agent, space-agent, picoclaw, etc.). This document covers the _operational ecosystem_ those projects don't expose: how a real fleet runs, what patterns emerge, and what gormes-agent is missing from its own operational model.

---

## Executive Summary

The Sages fleet runs 6 AI agents under OpenClaw orchestration with supervisord process management. Each agent has a standardized workspace layout (SOUL.md + IDENTITY.md + AGENTS.md + TOOLS.md + USER.md + MEMORY.md + memory/ + skills/), shared infrastructure (qmd hybrid search, agent-browser), and a formalized blocker policy. gormes-agent has none of this operational scaffolding.

**Key findings:**
- gormes-agent has **zero fleet-level awareness** — no session health monitoring, no cross-agent drift detection, no blocker policy
- The fleet's **standardized workspace model** is a template gormes-agent should adopt for its own multi-agent story
- OpenClaw's **ACP bridge, sandboxing, backup, and device management** represent capabilities gormes-agent's Phase 5/6 roadmap doesn't yet address
- The fleet's **Git delivery contract** and **session reliability rules** are proven operational patterns gormes-agent lacks

---

## 1. Fleet Infrastructure — What gormes-agent Is Missing

### 1.1 Standardized Agent Workspace Layout (ALL 6 agents)

Every Sages agent uses the same directory structure. gormes-agent has a rich internal architecture but no formalized multi-agent workspace model:

```
workspace-<agent>/
├── SOUL.md            # Identity, mission, non-negotiable behavior
├── IDENTITY.md        # Full credential: role, domain, personality
├── AGENTS.md          # Operational workflow: startup, maintenance, delivery
├── TOOLS.md           # Dev environment, commands, dependencies
├── USER.md            # Durable facts about Juan (the operator)
├── MEMORY.md          # Curated long-term decisions and context
├── memory/            # Daily notes and project objectives
│   ├── YYYY-MM-DD.md
│   └── PROJECT-OBJECTIVES.md
├── sessions/          # Conversation transcripts
└── skills/            # Agent-specific capabilities
    ├── qmd/SKILL.md           # Hybrid search (ALL agents share this)
    ├── agent-browser/SKILL.md  # Browser automation (ALL agents share this)
    └── <domain-skill>/SKILL.md
```

**What gormes-agent should adopt:**

| Element | gormes-agent current state | Fleet standard |
|---------|---------------------------|----------------|
| SOUL.md identity | Partial (docs reference, not runtime-enforced) | Every agent loads SOUL.md into every session |
| AGENTS.md workflow | Exists but project-focused | Clear startup checklist + delivery contract |
| MEMORY.md curation | Goncho covers this at DB level | Human-readable complement to Goncho's structured memory |
| USER.md operator context | USER.md content injected via prompt builder | Standardized per-agent USER.md |
| memory/YYYY-MM-DD.md | Not present | Daily handoff logs, session continuity |
| Skills directory | Rich (20+ skills) but all project-specific | Standard qmd + agent-browser shared across fleet |
| PROJECT-OBJECTIVES.md | `TODO.md` covers blockers only | Active project goals with completion criteria |

**Recommendation:** gormes-agent should split its identity/runtime model into these standardized layers. Goncho already provides the structured memory backend — it needs the human-readable frontend files (USER.md, MEMORY.md, daily notes) as first-class artifacts that the prompt builder injects.

### 1.2 Session Health Monitoring (from Link)

Link's `session-health-monitor` skill is fleet-wide agent health monitoring. gormes-agent's `doctor` command checks _local_ runtime health but has no concept of monitoring other agent instances.

**What Link monitors:**
- Session file sizes with thresholds: warn at 500KB, alert at 2MB
- Heartbeat freshness: warn at 45min, alert at 90min
- supervisord process status
- Cross-agent session comparisons to detect anomalies

**What gormes-agent should adopt:**

| Capability | Current gormes state | Fleet standard |
|------------|---------------------|----------------|
| Session size monitoring | Not present | 500KB/2MB size tiers |
| Heartbeat age alerting | Not present | 45min/90min freshness tiers |
| Cross-agent health | Not present (single-binary model) | Fleet-wide comparison |
| Degraded mode visibility | Partial (doctor output) | Explicit alert levels |

**Recommendation:** Add `gormes health` as a fleet-aware command. Even for single-binary deployments, operators running multiple gormes instances (one per channel, one per environment) need cross-instance visibility. Track session sizes, heartbeat age, Goncho extraction queue depth, and tool execution failures in a structured health output.

### 1.3 Formalized Blocker Policy (from Mineru, adopted by all)

Every fleet agent follows the same blocker handling protocol. gormes-agent has no formalized blocker policy — work just stalls silently.

**Fleet blocker protocol:**
1. Classify: `access` | `infra` | `dependency` | `decision` | `bug` | `unknown`
2. Dual-record: `memory/YYYY-MM-DD.md` (daily log) + `TODO.md` or `MEMORY.md` (durable)
3. Evidence: exact command, stderr, checked paths
4. Auto-pivot: switch to highest-impact unblocked work in same domain
5. Report: `Blocked Work` and `Pivoted Work Completed` sections with exact files

**Blocker record format:**
```
[BLOCKED] <task> — <timestamp America/Monterrey>
  blocker: <what is blocked>
  evidence: <exact error output>
  unblocks when: <condition>
  owner: <who can resolve>
  workaround/pivot: <new task in progress>
  next check: <date/time>
```

**What gormes-agent should adopt:**
- `gormes status` should report blocked slices with evidence
- `progress.json` rows should support a `blocked` state with structured blocker metadata
- Degraded mode reporting should include active blockers alongside subsystem health
- The builder skill should auto-pivot when hitting a blocker rather than stalling

### 1.4 Git Delivery Contract (Mandatory fleet-wide)

Every fleet agent follows the same Git workflow. gormes-agent's AGENTS.md mentions branch rules but not the full delivery contract:

```
1. git status --short + git rev-parse --abbrev-ref HEAD (preflight)
2. Split commits by concern (one fix, one refactor, one docs = separate commits)
3. Commit immediately after each validated slice
4. Push each validated commit to origin
5. If not pushed, report why and exact blocker
6. End report: repo path, branch, commit hashes, push confirmation
```

**What gormes-agent should adopt:**
- The split-commit discipline is stricter than gormes-agent's current "don't commit to main" rule
- Every builder pass should end with a push-or-report cycle
- The builder skill should include post-commit validation before declaring a row complete

---

## 2. Shared Fleet Infrastructure — Leverage Points for gormes-agent

### 2.1 QMD Hybrid Search (All 6 agents)

QMD is the fleet's shared local search infrastructure. Every agent has an identical `skills/qmd/SKILL.md` providing BM25 + vector hybrid search across all markdown docs in the workspace.

**What it does:**
- Searches memory/, sessions/, TOOLS.md, USER.md, AGENTS.md, MEMORY.md
- BM25 keyword + vector similarity hybrid ranking
- Works offline, no external service

**gormes-agent gap:**
- Goncho provides semantic recall for conversation context
- No equivalent for searching across documentation, runbooks, or agent instructions
- `session_search` tool is session-bound, not workspace-wide

**Recommendation:** Add a `gormes search` command (or `gormes qmd` subcommand) that indexes and searches all markdown docs in the workspace. This fills the gap between Goncho's structured memory (session-level) and the operator's need to search across all documentation, runbooks, and agent history.

### 2.2 Agent-Browser / Browser Automation (All 6 agents)

Every fleet agent has an identical Playwright-based browser automation skill. gormes-agent has its own browser tools (Chromedp-based) but the fleet standard provides useful patterns:

**Fleet browser patterns:**
- `take_screenshot` — evidence-first verification
- `extract_data` — structured extraction with CSS selectors
- `navigate_and_wait` — explicit wait conditions, not blind delays
- `fill_form` — form interaction with validation

**Recommendation:** Compare gormes-agent's browser tool schemas against the fleet standard. The fleet's `extract_data` with CSS selector patterns and explicit wait conditions are more operator-friendly than the current CDP-level interface.

### 2.3 Fleet Process Management (from Mineru)

Mineru's `fleet-admin` skill manages 7 supervisord processes with explicit port mappings:

| Agent | Port |
|-------|------|
| Mineru | 18790 |
| Link | 18791 |
| Yunobo | 18792 |
| Riju | 18793 |
| Tulin | 18794 |
| Sidon | 18795 |
| task-manager | 18800 |

**gormes-agent gap:** gormes-agent manages gateway channels (Telegram, Discord, Slack) but has no concept of _multiple gormes instances_ or port management for fleet deployments. An operator running gormes-agent for multiple channels simultaneously (e.g., one per Telegram bot, one per Discord server) has no tooling for this.

**Recommendation:** Add `gormes fleet` or `gormes instances` command that:
- Lists running gormes instances with PID, port, channels, session count
- Supports instance-level start/stop/restart
- Detects port conflicts
- This is Phase 5.E (final purge) material

---

## 3. OpenClaw Platform Capabilities — gormes-agent Parity Targets

OpenClaw 2026.3.28 provides a rich operational surface that gormes-agent should match or exceed. Below is the full capability inventory with gormes-agent parity assessment.

### 3.0 Platform Capability Matrix

| OpenClaw Command | What It Does | gormes-agent Status | Priority |
|---|---|---|---|
| `openclaw agent` | Run one agent turn via Gateway | `gormes --oneshot` | Shipped |
| `openclaw agents *` | Manage isolated agents (workspaces, auth, routing) | Not present (single-binary model) | P2 |
| `openclaw approvals *` | Manage exec approvals (allowlist, get, set) | `gormes approvals` partial | P1 |
| `openclaw backup *` | Create and verify local backup archives | Gap (`must-have` listed) | P1 |
| `openclaw channels *` | Manage chat channels (add, list, login, logout, logs, remove) | `gormes gateway` partial | P2 |
| `openclaw config *` | Config helpers (get/set/unset/file/schema/validate) | `gormes config` partial | P2 |
| `openclaw configure` | Interactive configuration wizard | `gormes onboard` partial | P2 |
| `openclaw cron *` | Cron job management (add, edit, enable, disable, list, rm, run, runs) | `gormes cron` partial (missing: edit, runs history) | P2 |
| `openclaw dashboard` | Open Control UI with token | `gormes dashboard` shipped | Shipped |
| `openclaw devices *` | Device pairing + token management | Gap | P3 |
| `openclaw directory *` | Contact/group ID lookup for channels | Gap | P3 |
| `openclaw dns *` | DNS helpers (Tailscale + CoreDNS) | Out of scope (platform concern) | — |
| `openclaw docs` | Search live docs | `gormes search` (planned via QMD) | P2 |
| `openclaw doctor` | Health checks + quick fixes | `gormes doctor --offline` | Shipped |
| `openclaw gateway *` | Gateway management (run, inspect, query) | `gormes gateway` shipped | Shipped |
| `openclaw health` | Fetch health from running gateway | `gormes gateway status` | Shipped |
| `openclaw hooks *` | Agent hook management (check, enable, disable, info, list) | Not present | P2 |
| `openclaw logs` | Tail gateway file logs via RPC | Gap (`must-have` listed) | P1 |
| `openclaw message *` | Send, read, manage messages | Implicit via gateway channels | P3 |
| `openclaw models *` | Model discovery (aliases, auth, fallbacks, status) | `gormes model` + provider registry | Partial |
| `openclaw node *` / `nodes *` | Headless node service management | Out of scope (single-binary model) | — |
| `openclaw onboard` | Interactive onboarding (gateway, workspace, skills) | `gormes setup` alias partial | P1 |
| `openclaw pairing *` | Secure DM pairing (approve inbound) | Gap | P3 |
| `openclaw plugins *` | Plugin management (install, list, enable, disable, update, uninstall, marketplace, inspect) | `gormes plugins` partial (missing: marketplace, inspect) | P2 |
| `openclaw qr` | iOS pairing QR generation | Out of scope | — |
| `openclaw reset` | Reset local config/state | Gap | P2 |
| `openclaw sandbox *` | Container sandbox management | In-flight (Docker backend) | P1 |

### 3.1 ACP Bridge (Agent Control Protocol)

OpenClaw provides an ACP bridge backed by the Gateway. This enables:
- Session key/label routing
- Reset-session capability
- Require-existing guard
- Password/token authentication
- Provenance modes (off, meta, meta+receipt)

**gormes-agent gap:** gormes-agent has an API server with OpenAI-compatible endpoints but no ACP bridge. ACP is the standard protocol for agent-to-agent and tool-to-agent communication.

**Recommendation:** Add ACP bridge capability in Phase 5.N. This would let gormes-agent interoperate with other ACP-compatible agents and tools, bridging the gap between the OpenClaw fleet ecosystem and the Go-native runtime.

### 3.2 Sandbox / Container Management

OpenClaw provides Docker-based sandbox containers for agent isolation:
- `openclaw sandbox list` — list all containers
- `openclaw sandbox explain` — explain effective sandbox/tool policy
- `openclaw sandbox recreate` — remove and recreate with updated config
- Browser-specific container support

**gormes-agent gap:** gormes-agent has a Docker sandbox backend (in-flight, Phase 5). OpenClaw's explain functionality (showing what policy applies to a given session/agent) is a UX pattern gormes-agent should match.

**Recommendation:** Add `gormes sandbox explain` that shows the effective trust class, tool allowlist, filesystem scope, and network policy for any agent context. This makes sandbox policy operator-visible (reinforcing the "visible degraded mode" contract).

### 3.3 Backup / Restore

OpenClaw provides `openclaw backup` with local archive creation and verification. gormes-agent's must-have list marks backup/restore as a gap.

**Recommendation:** Implement `gormes backup` and `gormes restore` (Phase 5). The fleet's approach of verifying archives after creation is a quality pattern to adopt.

### 3.4 Device Pairing & Token Management

OpenClaw provides `openclaw devices` for device pairing and token management. gormes-agent has provider-level auth (OAuth, API keys) but no device-level authorization model.

**gormes-agent gap:** When gormes-agent is deployed across multiple machines or accessed by multiple operators, there's no device-level access control. The current model assumes single-operator, single-machine.

**Recommendation:** Add device pairing as a Phase 5.N commodity feature. Simple implementation: TOTP or pairing codes for new device authorization, with device revocation from the operator console.

### 3.5 Agent Hooks System

OpenClaw provides a hook registry for internal agent behavior modification:
- `openclaw hooks list` — list all hooks
- `openclaw hooks check` — check eligibility status
- `openclaw hooks enable/disable` — toggle hooks
- `openclaw hooks info` — detailed hook information

**gormes-agent gap:** gormes-agent has HOOK.yaml support (gateway boot hooks) but no generalized hook registry. OpenClaw's hook system lets operators inspect and control which agent behaviors are active without editing config files.

**Recommendation:** Add `gormes hooks` command with list/enable/disable/check operations. Hooks should be inspectable at runtime, not just compile-time configured. This is Phase 5.N material.

### 3.6 Plugin Ecosystem

OpenClaw provides a full plugin management surface:
- `openclaw plugins install` — install from path, archive, npm spec, ClawHub, or marketplace
- `openclaw plugins list` — list discovered plugins
- `openclaw plugins enable/disable` — toggle in config
- `openclaw plugins inspect` — detailed inspection
- `openclaw plugins doctor` — report load issues
- `openclaw plugins marketplace` — inspect compatible marketplaces
- `openclaw plugins update/uninstall` — lifecycle management

**gormes-agent gap:** gormes-agent has a plugin inventory system (`internal/plugins/`) and first-party plugins (Spotify, Google Meet) but lacks:
- Plugin marketplace support (ClawHub-compatible or independent)
- Plugin doctor/health reporting
- Plugin inspect with manifest details
- Third-party plugin sandboxing (WASM/subprocess isolation)

**Recommendation:** Phase 5 plugin parity should target marketplace compatibility, plugin doctor, and third-party sandboxing. The market enables community plugins; the sandbox protects operators from untrusted code.

### 3.7 Interactive Onboarding

OpenClaw provides `openclaw onboard` — an interactive workflow covering gateway setup, workspace configuration, and skill installation.

**gormes-agent gap:** gormes-agent has `gormes setup` (a lightweight wizard alias) but lacks the depth of OpenClaw's onboarding: model/provider selection, auth setup, gateway channel configuration, browser/CDP checks, skill discovery, and dashboard launch in one flow.

**Recommendation:** Promote `gormes onboard` from its current alias into a full interactive flow as listed in the implementation roadmap's Product Hardening Borrow List. Match OpenClaw's depth: model/provider → auth → gateway → browser → skills → dashboard.

### 3.8 Memory Plugin Architecture

OpenClaw's memory system is plugin-based (`plugin memory-core`) with:
- `openclaw memory status` — show index and provider status
- `openclaw memory search` — search memory files
- `openclaw memory index` — reindex memory files

**gormes-agent gap:** gormes-agent's Goncho is monolithic (SQLite + FTS5 + Ollama embeddings). No plugin architecture for alternative backends (Turbopuffer, LanceDB, Redis).

**Recommendation:** The existing multi-memory backend plan (from cross-project feature map) aligns with OpenClaw's plugin architecture. gormes-agent should expose a `MemoryBackend` interface so operators can swap between SQLite, PostgreSQL+pgvector, or LanceDB without changing the Goncho API surface.

---

## 4. Cross-Fleet Architectural Patterns Worth Absorbing

### 4.1 Session Rollover at 1500KB (All 6 agents)

Every fleet agent has a session reliability rule: if the session file grows above 1500KB, write a 5-line handoff summary in `memory/YYYY-MM-DD.md` and continue in a fresh session.

**What gormes-agent should adopt:**
- Goncho already tracks session size via SQLite
- Add a `gormes session rollover` command that exports the current session, writes a handoff summary, and starts a fresh session
- Make this automatic at a configurable size threshold

### 4.2 Evidence-Before-Claims Quality Gate (from Link, adopted fleet-wide)

Link's research skill enforces: every claim must have source URL provenance. No assertion without evidence. This pattern appears across the fleet:
- Riju: `[UNVERIFIED]` marking, DOI-backed validation, exact error thresholds
- Yunobo: "Accessibility as product" — every screen TalkBack-validated
- Mineru: Evidence-first status reports, exact pass/fail counts

**What gormes-agent should adopt:**
- `gormes doctor` output should include exact evidence, not summary claims
- Build/test results should include exact pass/fail/skip counts
- The TDD slice skill should assert evidence before declaring a row complete

### 4.3 Stale-While-Revalidate Caching (from Yunobo)

Yunobo's Flutter app uses a 30-second cache with background refresh on tab switch. This is a general pattern for any system where freshness matters but responsiveness matters more.

**What gormes-agent should adopt:**
- Provider status checks: cache for 30s, background refresh
- Memory extraction status: cache Goncho queue depth, background refresh
- Gateway channel status: cache channel state, background refresh

### 4.4 Dual-Lane Workflow (from Riju, Sidon)

Riju operates in two lanes: Lane A (physics falsification) and Lane B (research dataset ingestion). Sidon operates in GPU and CPU rendering lanes with explicit switching thresholds.

**What gormes-agent should adopt:**
- Build vs. plan lanes (already exists via gormes-planner vs gormes-builder)
- Provider lanes: prime provider vs fallback provider with explicit degradation reporting
- Memory lanes: structured Goncho vs ephemeral session memory with explicit boundary

---

## 5. Concrete Features gormes-agent Should Steal

### 5.1 From the Fleet (Priority-Ordered)

| Priority | Feature | Source Agent | Target gormes Phase | Effort |
|----------|---------|-------------|---------------------|--------|
| **P0** | Formalized blocker policy with auto-pivot | Mineru (all agents) | Phase 5 — Final Purge | 1 week |
| **P0** | Session health monitoring (size + heartbeat tiers) | Link | Phase 5 — Final Purge | 1 week |
| **P0** | Evidence-before-claims quality gate in doctor/build output | Link (adopted fleet-wide) | Phase 5 — Final Purge | 1 week |
| **P1** | QMD hybrid search for docs/runbooks/memory | All 6 agents | Phase 5.N — Operator surface | 2 weeks |
| **P1** | Git delivery contract enforcement in builder skill | All agents | Existing builder skill | 1 week |
| **P1** | Session rollover automation | All agents | Phase 5 — Final Purge | 1 week |
| **P2** | Multi-instance fleet management (`gormes instances`) | Mineru fleet-admin | Phase 5.E — Commodity | 3 weeks |
| **P2** | Standardized workspace layout for multi-agent deployments | All agents | Phase 6 — Learning Loop | 2 weeks |
| **P2** | ACP bridge for agent interoperability | OpenClaw platform | Phase 5.N | 3 weeks |
| **P3** | Stale-while-revalidate caching for status checks | Yunobo | Phase 5 — Observability | 1 week |
| **P3** | Device pairing + token management | OpenClaw platform | Phase 5.N | 2 weeks |

### 5.2 From OpenClaw Platform (New Capabilities)

| Priority | Feature | Description | Target Phase |
|----------|---------|-------------|-------------|
| **P1** | Sandbox policy explain | `gormes sandbox explain` — visible effective trust class, allowlist, scope | Phase 5 — Final Purge |
| **P1** | ACP bridge | Session-based agent communication protocol | Phase 5.N |
| **P1** | Interactive onboarding | `gormes onboard` — full model/provider/auth/gateway/browser/skills/dashboard flow | Phase 5 — Final Purge |
| **P2** | Memory backend plugin architecture | Swap SQLite/Postgres/LanceDB behind Goncho interface | Phase 5 — Black Box enhancements |
| **P2** | Integrated backup with archive verification | `gormes backup` and `gormes restore` | Phase 5 — Final Purge |
| **P2** | Agent hooks registry | `gormes hooks` — list/enable/disable/check/inspect at runtime | Phase 5.N |
| **P2** | Plugin marketplace + doctor | Marketplace compatibility, plugin health reporting, third-party sandboxing | Phase 5 — Final Purge |
| **P2** | Logs command | `gormes logs` — tail gateway file logs (counterpart to `openclaw logs`) | Phase 5 — Final Purge |
| **P3** | Container-aware runtime | Track running in Docker/Podman, adapt ports/paths | Phase 5.E |
| **P3** | Device pairing + token management | Device authorization for multi-machine deployments | Phase 5.N |

---

## 6. What NOT to Adopt

Features explicitly excluded from gormes-agent scope after fleet analysis:

| Feature | Source | Reason |
|---------|--------|--------|
| supervisord process management | Mineru fleet-admin | gormes-agent is a single binary; process management belongs to the host |
| Trading bot ML pipeline | Mineru (reference for Yunobo) | Domain-specific; not an agent runtime concern |
| Flutter/React Native app dev workflows | Yunobo, Tulin, Sidon | gormes-agent is a Go agent runtime, not a mobile app builder |
| Physics validation golden files | Riju | Domain-specific |
| Fractal rendering GPU shaders | Sidon | Domain-specific |
| Ticket reconciliation | Tulin | Domain-specific |
| nanobot CLI dependency | Fleet infrastructure | gormes-agent must remain dependency-free (single binary) |

---

## 7. Implementation Sequence (Recommended)

### Immediate (Next 30 Days) — Operational Foundation

```
Week 1: Formalized blocker policy in gormes-planner + gormes-builder skills
Week 1: Evidence-before-claims in doctor output (exact counts, not summaries)
Week 2: Session health monitoring (gormes health command with size/heartbeat tiers)
Week 2: Git delivery contract enforcement in builder skill
Week 3-4: QMD hybrid search (gormes search command for doc/runbook/memory)
```

### Short-Term (Next 90 Days) — Fleet Readiness

```
Week 5-6: Session rollover automation
Week 7-8: Sandbox policy explain (visible effective policy)
Week 9-10: Multi-instance fleet management preview
Week 9-10: Interactive onboarding (model/provider/auth/gateway/browser/skills)
Week 11-12: Memory backend plugin architecture
Week 11-12: Agent hooks registry + plugin marketplace/doctor
```

### Medium-Term (Next 6 Months) — Platform Parity

```
Month 3-4: ACP bridge implementation
Month 4: Integrated backup/restore + logs command
Month 5: Device pairing + standardized multi-agent workspace layout
Month 6: Full plugin marketplace compatibility + third-party sandboxing
```

---

## 8. Success Metrics

- **30 days**: Blocker policy shipped, health monitoring active, evidence-before-claims in all status output
- **90 days**: QMD search operational, sandbox policy visible, git delivery contract enforced
- **6 months**: ACP bridge, backup/restore, multi-instance management

---

## References

- [Cross-Project Feature Map](../cross-project-feature-map/) — External open-source project analysis (12 projects)
- [Must-Have Features](../must-have-features/) — Comprehensive feature catalogue
- [Upstream Lessons](../upstream-lessons/) — Durable contracts from Hermes
- [Implementation Roadmap](../implementation-roadmap/) — Current state and execution horizons
- [Architecture Plan](../architecture_plan/) — Phase-by-phase delivery plan
- [Contract Readiness](../contract-readiness/) — Row-level handoff contract specifications

---

*Generated: May 1, 2026*
*Source: 6-agent sages-openclaw fleet ecosystem + OpenClaw 2026.3.28 platform analysis*
*Cross-referenced against: cross-project-feature-map.md, must-have-features.md, upstream-lessons.md, architecture_plan/progress.json*
