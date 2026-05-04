---
title: "Gormes Behavioral Gap Matrix"
description: "Curated behavioral gap inventory for observable Hermes-vs-Gormes differences. Cross-references parity matrix, swarm audit, and progress.json."
date: 2026-05-04
draft: false
---

# Gormes Behavioral Gap Matrix

**Last updated:** 2026-05-04 (behavioral fidelity audit: UI, commands/text, tool loops)

**Derived from:** [Hermes <-> Gormes Parity Matrix](../hermes-gormes-parity-matrix/), [Swarm Feature Parity Audit](../swarm-feature-parity-audit/), [Hermes And Honcho Feature Map](../hermes-honcho-feature-map/), `progress.json`

---

## What This Matrix Tracks

Observable behavioral differences between Hermes and Gormes. The [parity matrix](../hermes-gormes-parity-matrix/) owns the full 80-row technical inventory; this page focuses on UX-critical gaps.

**Status:** `parity` / `partial` / `missing` / `regressed` / `excluded`

**Severity:** **P0** — breakage or advertised feature that doesn't work. **P1** — significant UX gap. **P2** — polish/edge-case.

---

## Section 1 — Telegram UX & Channel Behavior

| # | Behavior | Severity | Status | Progress Row |
|---|----------|----------|--------|-------------|
| 1.1 | `/status` formatting — bold labels, ⚡ marker, local timestamps | P1 | parity | `Telegram /status Hermes-format closeout` (2.B.1) |
| 1.2 | `/title` command — output formatting (emoji+bold) | P1 | partial | `Gateway /title manual session title command` (2.F.3) |
| 1.3 | Dynamic slash menu — priority tiers, hidden-count | P1 | parity | `Telegram dynamic BotCommand menu wiring` (2.B.1) |
| 1.4 | Reply quoting modes (first/all/off) + deleted-target fallback | P1 | planned | `Telegram reply_to_mode and reply-context parity` (2.B.1) |
| 1.5 | Message threading (forum topic message_thread_id) | P1 | planned | Same as 1.4 (scope expansion) |
| 1.6 | Typing action (native sendChatAction) | P2 | planned | `Gateway typing-action wiring during stream` (2.B.1) |
| 1.7 | Tool trace skin emoji overrides | P2 | parity | `Native TUI Hermes skin token renderer` + `Gateway stream/tool trace formatting fixture matrix` |
| 1.8 | Operator duplicate collapse (restart markers) | P2 | parity | `Gateway /restart command + takeover markers` + `Gateway message deduplicator bounded helper` |

## Section 2 — Session & Status Management

| # | Behavior | Severity | Status | Progress Row |
|---|----------|----------|--------|-------------|
| 2.1 | Session title auto-generation | P1 | partial | `Gateway conversational session metadata refresh` (2.F.3) |
| 2.2 | Title persistence quality | P1 | partial | Same |
| 2.3 | Created timestamp — local-timezone format, never (unknown) | P1 | partial | Same |
| 2.4 | Last activity timestamp — updated every turn | P1 | partial | Same |
| 2.5 | Token accounting — session totals vs per-turn | P1 | partial | `Gateway session token accounting parity` (2.F.3) |
| 2.6 | Session reset/new/retry/undo handlers | P1 | partial | `49-file CLI tree port` (5.O) |
| 2.7 | Session ID format — random vs deterministic fallback | P2 | partial | `Gateway conversational session metadata refresh` |

## Section 3 — Tool Calling & Progress Display

| # | Behavior | Severity | Status | Progress Row |
|---|----------|----------|--------|-------------|
| 3.1 | Tool guidance constants in live prompt | P1 | partial | `Live-turn model/tool guidance wiring` (4.A) |
| 3.2 | Skill guidance ordering in system prompt | P1 | planned | `Toolset-aware skills prompt snapshot` (4.C) |
| 3.3 | Streaming text formatting | — | parity | Complete |
| 3.4 | Tool trace rendering (all channels) | — | parity | Complete |
| 3.5 | Iteration-limit visible finalization — one summary final, no raw loop text, no stuck progress | P1 | planned | `Channel/TUI iteration-limit finalization transcript fixture` (5.Q) |

## Section 4 — Identity, Persona & Defaults

| # | Behavior | Severity | Status | Progress Row |
|---|----------|----------|--------|-------------|
| 4.1 | SOUL.md identity reaches provider (production golden test) | P0 | partial | `Telegram production live-turn provider payload golden` (2.B.5) |
| 4.2 | MEMORY.md then USER.md ordering + frozen snapshot semantics | P1 | partial | `Durable context ordering and frozen snapshot decision fixture` (3.F) |
| 4.3 | AGENTS.md/project context asserted in production test | P1 | partial | Same as 4.1 |
| 4.4 | Developer role swap (GPT-5/Codex) in production entrypoint | P0 | partial | `OpenAI-compatible developer-role API-boundary swap` (4.A) |
| 4.5 | Profile name resolver for context root | P1 | partial | `Active Hermes/Sidon profile context root resolver` (4.A) |

## Section 5 — Memory & Goncho Behavior

| # | Behavior | Severity | Status | Progress Row |
|---|----------|----------|--------|-------------|
| 5.1 | Hermes `memory` tool — IS implemented and registered | P1 | partial | `Hermes memory tool over Goncho/local durable store` (3.F) |
| 5.2 | Memory lifecycle adapter (initialize → shutdown) | P1 | partial | `Goncho memory provider lifecycle adapter` (3.F) |
| 5.3 | Memory prompt insertion — frozen snapshot semantics | P1 | partial | `System + memory + tools + history assembly` (3.F) |
| 5.4 | Session compression boundary | P1 | partial | Phase 4.B rows |
| 5.5 | GONCHO branding | — | parity | Internal naming |

## Section 6 — Provider / Auth / Runtime

| # | Behavior | Severity | Status | Progress Row |
|---|----------|----------|--------|-------------|
| 6.1 | Auth status display names | P2 | partial | `Gormes auth status per-provider aggregator` (4.G) |
| 6.2 | Auth Spotify subcommand (PKCE OAuth) | P1 | partial | `Hermes auth Spotify service-provider subcommand` (4.G) |
| 6.3 | Top-level logout shortcut | P1 | partial | `Gormes top-level logout provider shortcut` (4.G) |
| 6.4 | Codex OAuth device flow | P1 | partial | `Hermes auth OAuth provider adapters` (4.G) |
| 6.5 | Credential pool interactive readout | P1 | planned | `Gormes auth bare interactive credential-pool readout` (4.G) |
| 6.6 | Provider stream recovery | P1 | partial | `Provider stream dispatch recovery wired into cmd` (4.A) |
| 6.7 | Rate-limit evidence | P1 | partial | `Provider account usage read model + renderer` (4.H) |
| 6.8 | Provider registry gaps (Bedrock, Gemini, etc.) | P1 | partial | Phase 4.A provider rows |
| 6.9 | Redacted diagnostics | — | parity | Existing |

## Section 7 — Browser & External Tools

| # | Behavior | Severity | Status | Progress Row |
|---|----------|----------|--------|-------------|
| 7.1 | Browser action backend (CDP) | P1 | partial | `go-browser-harness Chromedp action backend` (5.C) |
| 7.2 | Browser artifact rendering | P1 | partial | `Browser artifact and console render contract` (5.C) |
| 7.3 | Telegram browser artifact rendering | P1 | planned | `Telegram browser artifact rendering` (5.C) |
| 7.4 | Browser tool contract | — | parity | Phase 5.C |
| 7.5 | Artifact budgets | — | parity | Complete |

## Section 8 — CLI / Config

| # | Behavior | Severity | Status | Progress Row |
|---|----------|----------|--------|-------------|
| 8.1 | CLI command registry inventory | P1 | parity | `Hermes CLI command-tree parity manifest` (5.O) |
| 8.1b | CLI setup/onboard/help visible text fidelity | P1 | planned | `CLI setup/onboard/help text fidelity matrix` (5.O) |
| 8.1c | Native TUI slash dispatch visible behavior — known commands never become prompt text | P1 | planned | `Native TUI Hermes slash dispatch behavioral matrix` (5.Q) |
| 8.2 | Config edit/check/migrate | P1 | partial | `Gormes config edit/check/native schema-migrate closeout` (5.O) |
| 8.3 | Diagnostics CLI (doctor/backup/logs) | P1 | partial | `Diagnostics, backup, logs, and status CLI` (5.O) |
| 8.4 | Gateway status CLI | — | parity | Complete |
| 8.5 | Config path discovery | — | parity | Complete |

---

## M-Gaps (Upstream-Verified Behavioral Gaps)

### M1-M8 (Phase 1 triage)

| # | Behavior | Severity | Status | Progress Row |
|---|----------|----------|--------|-------------|
| M1 | Browser hybrid private-URL routing | P1 | complete | `Browser hybrid private-URL local sidecar routing` (5.C) |
| M2 | Browser CDP dialog/alert supervisor | P1 | planned | `Browser CDP dialog/alert supervisor attachment` (5.C) |
| M3 | Memory injection threat scanning | P1 | planned | `Memory content injection threat scanning for add/replace` (3.F) |
| M4 | Active session bypass command set | P1 | planned | `Active session bypass command set` (2.F.1) |
| M5 | Provider resolution priority chain | P1 | planned | `Provider resolution priority chain` (4.A) |
| M6 | Per-platform Telegram skill filtering | P2 | planned | `Per-platform Telegram skill filtering` (2.B.1) |
| M7 | Auth error UX quality | P2 | planned | `Auth error user-facing guidance mapping` (4.G) |
| M8 | Telegram DM Topics lifecycle | P2 | planned | `Telegram DM forum topic lifecycle` (2.B.1) |

### M9-M16 (Phase 2 triage)

| # | Behavior | Severity | Status | Progress Row |
|---|----------|----------|--------|-------------|
| M9 | Browser inactivity reaper + orphan cleanup | P1 | planned | `Browser session inactivity reaper + orphan cleanup` (5.C) |
| M10 | Browser macOS AF_UNIX socket workaround | P1 | planned | `Browser macOS AF_UNIX socket path workaround` (5.C) |
| M11 | Gateway agent cache LRU + idle TTL | P2 | planned | `Gateway agent cache with LRU + idle TTL` (2.F.6) |
| M12 | Busy input modes (queue/steer/interrupt) | P1 | planned | `Busy input modes (queue/steer/interrupt) with debounce` (2.F.7) |
| M13 | Auto-continue freshness gate (1-hour window) | P2 | planned | `Auto-continue freshness gate (1-hour window)` (2.F.6) |
| M14 | /queue FIFO with overflow buffer | P1 | planned | `/queue FIFO with overflow buffer` (2.F.7) |
| M15 | Stuck loop detection + auto-suspension | P1 | planned | `Stuck loop detection + auto-suspension` (2.F.8) |
| M16 | Detached /restart via subprocess | P1 | planned | `Detached /restart via subprocess (zero-downtime)` (2.F.9) |

### M17-M18, M21-M27, M47-M50 (Telegram & Browser)

| # | Behavior | Severity | Status | Progress Row |
|---|----------|----------|--------|-------------|
| M17 | Browser SSRF post-redirect private-address blocking | P1 | planned | `Browser SSRF post-redirect private-address blocking` (5.C) |
| M18 | Browser stealth feature + bot detection warnings | P2 | planned | `Browser stealth and bot detection warnings` (5.C) |
| M21 | Telegram webhook mode with TELEGRAM_WEBHOOK_SECRET | P1 | planned | `Telegram webhook mode` (2.B.1) |
| M22 | Telegram polling conflict 3-retry | P1 | planned | `Telegram polling conflict handling` (2.B.1) |
| M23 | Telegram network backoff 10-retry | P1 | planned | `Telegram network error exponential backoff` (2.B.1) |
| M24 | Telegram inline approval buttons | P1 | planned | `Telegram inline approval buttons` (2.B.1) |
| M25 | Telegram inline model picker | P1 | planned | `Telegram inline model picker` (2.B.1) |
| M26 | Telegram group mention gating | P1 | planned | `Telegram group mention gating` (2.B.1) |
| M27 | Telegram fallback IP auto-discovery | P1 | planned | `Telegram fallback IP auto-discovery` (2.B.1) |
| M47 | Telegram media batch 0.8s delay | P2 | planned | `Telegram media batch delay` (2.B.1) |
| M48 | Telegram text batch 0.6s/2.0s delay | P2 | planned | `Telegram text batch delay` (2.B.1) |
| M49 | Telegram markdown table conversion | P2 | planned | `Telegram markdown table conversion` (2.B.1) |
| M50 | Telegram send_voice native handling | P2 | planned | `Telegram send_voice native handling` (2.B.1) |

### M19-M20, M30-M46 (Remaining P1+P2 — row-backed in this pass)

| # | Behavior | Severity | Status | Progress Row |
|---|----------|----------|--------|-------------|
| M19 | Gateway allowlist warning at startup | P1 | planned | `Gateway allowlist warning at startup` (2.F.3) |
| M20 | Stuck session auto-suspension with counter persistence | P1 | planned | `Stuck session auto-suspension` (2.F.3) |
| M30 | Memory file locking with separate .lock file | P1 | planned | `Memory file locking with separate .lock file` (3.F) |
| M31 | Memory atomic write (temp+fsync+rename) | P1 | planned | `Memory atomic write (temp file + fsync + rename)` (3.F) |
| M32 | Auth cross-process lock with 15s timeout | P1 | planned | `Auth cross-process lock with 15-second timeout` (4.G) |
| M33 | Z.AI endpoint probing + cached detection | P2 | missing | Not yet row-backed |
| M34 | Provider explicit config check (3-source) | P2 | missing | Not yet row-backed |
| M36 | Slash command autocomplete with file/@ completion | P1 | planned | `Slash command autocomplete with file/@ completion` (5.O) |
| M37 | Discord skill commands by category | P2 | missing | Not yet row-backed |
| M38 | Slack native slash command registration | P1 | planned | `Slack native slash command registration` (2.B.3) |
| M39 | Voice mode persistence per chat | P2 | missing | Not yet row-backed |
| M40 | Service tier / priority processing config | P2 | missing | Not yet row-backed |
| M41 | Reasoning effort per-session override command | P1 | planned | `Reasoning effort per-session override command` (4.A) |
| M42 | Gateway prefill messages from file | P2 | planned | `Ephemeral prefill messages file injection` (4.C) |
| M43 | Nous Portal agent key minting | P1 | planned | `Nous Portal agent key minting` (4.G) |
| M44 | Hermes CLI alias resolution | P2 | planned | `Hermes CLI alias and suggestion fidelity matrix` (5.O) |
| M45 | Memory duplicate entry rejection | P2 | missing | Not yet row-backed |
| M46 | Gateway shutdown notification to active sessions | P2 | planned | `Gateway shutdown notification to active sessions` (2.F.3) |

---

## Action Register

**P0 (Next Builder Cycle):**
1. Telegram production live-turn provider payload golden — capture final ChatRequest with SOUL/AGENTS/MEMORY/developer role
2. Developer-role API-boundary swap — verify production entrypoint invokes the swap

**P1 (Next 2-3 Planner Passes):**
3. Native TUI Ink behavioral transcript golden matrix
4. Native TUI Hermes slash dispatch behavioral matrix
5. CLI setup/onboard/help text fidelity matrix
6. Channel/TUI iteration-limit finalization transcript fixture
7. Gateway conversational session metadata refresh
8. Telegram reply_to_mode and reply-context parity
9. Telegram inline approval buttons + model picker
10. Telegram webhook mode + polling conflict + network backoff
11. Memory file locking + atomic writes
12. Auth cross-process lock

**P2 (Follow-Up):**
13. 49-file CLI tree port handler closeout + slash autocomplete binding
14. Telegram typing action, markdown tables, send_voice
15. Hermes CLI alias and suggestion fidelity matrix
16. Ephemeral prefill messages file injection
17. Z.AI probing, config check, memory dedup

## 2026-05-04 Behavioral Fidelity Audit

The audit reclassified several previously partial UX items as covered by completed rows and added source-backed rows for the remaining visible fidelity risk:

| Area | Audit result | Progress row |
|---|---|---|
| Active full-screen UI source of truth | Current Hermes Ink `ui-tui/src/components/appLayout.tsx`, `appChrome.tsx`, `messageLine.tsx`, `thinking.tsx`, and `queuedMessages.tsx` now supersede legacy prompt_toolkit as the primary UI parity source. | `Native TUI Ink behavioral transcript golden matrix` |
| Native TUI slash dispatch | Active Hermes Ink `createSlashHandler` treats slash text as command input first: local commands, native/gateway commands, stale slash output, catalog aliases, skill dispatch, and unavailable commands must route or degrade visibly before any model submission. | `Native TUI Hermes slash dispatch behavioral matrix` |
| Setup/onboard/help text | Functional setup/onboard rows are complete, but visible copy is spread across tests; a consolidated text matrix is needed for first-run UX regressions. | `CLI setup/onboard/help text fidelity matrix` |
| CLI alias and suggestion behavior | Hermes central command registry, legacy CLI loop, gateway dispatcher, and active Ink slash handler all define visible alias, prefix, quick-command alias, ambiguous-prefix, and unknown-command guidance that needs fixtures beyond command inventory parity. | `Hermes CLI alias and suggestion fidelity matrix` |
| Ephemeral prefill messages | Hermes CLI/gateway load `prefill_messages_file` / `HERMES_PREFILL_MESSAGES_FILE` and inject message objects into provider calls after the system prompt but before history, without saving them to sessions or displaying them. | `Ephemeral prefill messages file injection` |
| Tool-loop exhaustion | Kernel summary behavior and gateway tool progress are covered separately; one cross-surface transcript fixture is still needed. | `Channel/TUI iteration-limit finalization transcript fixture` |

---

## Change Log

| Date | Change |
|------|--------|
| 2026-04-30 | Initial creation from parity matrix + swarm audit |
| 2026-04-30 | Upstream audit corrections (1.2, 2.1, 2.3, 4.2, 4.4, 5.1) |
| 2026-04-30 | M1-M50 gaps documented and row-backed across all passes |
| 2026-04-30 | Rebuilt after subagent deletion, reflecting current progress.json state |
| 2026-05-04 | Behavioral fidelity audit added UI transcript, setup/onboard/help text, and iteration-limit transcript rows |
| 2026-05-04 | Behavioral fidelity follow-up added native TUI slash dispatch matrix for known-command non-leak behavior |
| 2026-05-04 | Behavioral fidelity follow-up row-backed Hermes CLI alias and suggestion behavior |
| 2026-05-04 | Behavioral fidelity follow-up row-backed gateway/local ephemeral prefill-message injection |
