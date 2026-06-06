---
title: "Fleet Integration Plan"
description: "Detailed planning: mapping sages-openclaw fleet operational patterns to Gormes phases, subphases, and progress.json rows."
weight: 27
---

# Fleet Integration Plan

> **Purpose:** Turn the [Fleet Operational Patterns](../fleet-operational-patterns/) analysis into concrete progress.json rows. Every pattern gets a phase assignment, subphase placement, and execution priority.
> **Audience:** gormes-planner, gormes-builder, and gormes-skill-manager skills.
> **Canonical queue:** progress.json rows created from this plan become the implementation queue.

---

## Phase Assignment Map

Each fleet pattern is assigned to the Gormes phase and subphase where it naturally fits:

| Pattern | Phase | Subphase | Lane | Priority |
|---------|-------|----------|------|----------|
| Blocker policy | 5 — Final Purge | 5.N (Misc Operator Tools) | 5 | P0 |
| Session health monitoring | 5 — Final Purge | 5.N (Misc Operator Tools) | 5 | P0 |
| Evidence-before-claims quality gate | 5 — Final Purge | 5.O (CLI Parity) | 5 | P0 |
| Git delivery contract enforcement | 5 — Final Purge | 5.N (Misc Operator Tools) | 5 | P1 |
| QMD hybrid search | 5 — Final Purge | 5.N (Misc Operator Tools) | 5 | P1 |
| Session rollover automation | 5 — Final Purge | 5.N (Misc Operator Tools) | 5 | P1 |
| Sandbox policy explain | 5 — Final Purge | 5.B (Sandboxing) | 3 | P1 |
| ACP bridge | 5 — Final Purge | 5.H (ACP Integration) | 3 | P1 |
| Interactive onboarding | 5 — Final Purge | 5.O (CLI Parity) | 5 | P1 |
| Agent hooks registry | 5 — Final Purge | 5.I (Plugins) | 3 | P2 |
| Plugin marketplace + doctor | 5 — Final Purge | 5.I (Plugins) | 3 | P2 |
| Logs command | 5 — Final Purge | 5.O (CLI Parity) | 5 | P2 |
| Multi-instance fleet management | 5 — Final Purge | 5.E (new subphase) | 4 | P2 |
| Standardized workspace layout | 6 — Learning Loop | 6.A (new subphase) | 6 | P2 |
| Memory backend plugin architecture | 5 — Final Purge | 5.I (Plugins) | 3 | P2 |
| Device pairing + token management | 5 — Final Purge | 5.N (Misc Operator Tools) | 5 | P3 |
| Container-aware runtime | 5 — Final Purge | 5.B (Sandboxing) | 3 | P3 |

---

## Row Specifications

### P0 — Immediate (Next 30 Days)

#### Row 1: Blocker Policy Integration

- **Phase:** 5.N (Misc Operator Tools)
- **Priority:** P0
- **Slice size:** small
- **Execution owner:** tools
- **Contract:** Port the sages-openclaw fleet blocker protocol into Gormes: classify blockages by type (access|infra|dependency|decision|bug|unknown), record evidence in structured format, auto-pivot to unblocked work, and surface active blockers in `gormes status` output.
- **Trust class:** operator
- **Degraded mode:** Unknown blocker type, missing evidence, or inability to pivot reports `blocker_unclassified` in status rather than crashing or silently stalling.
- **Fixture:** `internal/tools/blocker_test.go`
- **Source refs:**
  - `workspace-mineru/AGENTS.md` (blocker policy section)
  - `workspace-link/AGENTS.md` (session health monitor)
  - `fleet-operational-patterns.md`
- **Write scope:**
  - `internal/tools/blocker.go`
  - `internal/tools/blocker_test.go`
  - `internal/cli/status.go` (surface blockers)
  - `webpages/docs/content/building-gormes/progress.json`
- **Test commands:** `go test ./internal/tools -run TestBlocker -count=1`
- **Acceptance:**
  - `gormes status` shows active blockers with type, evidence, and owner
  - Blocker record format matches fleet standard: [BLOCKED] title — timestamp, blocker, evidence, unblocks when, owner, pivot, next check
  - gormes-planner can mark rows as blocked in progress.json
  - gormes-builder auto-pivots when hitting a blocker
- **Done signal:** Blocker records appear in status output, planner can mark rows blocked, builder auto-pivots.

#### Row 2: Session Health Monitoring

- **Phase:** 5.N (Misc Operator Tools)
- **Priority:** P0
- **Slice size:** small
- **Execution owner:** tools
- **Contract:** Port Link's session-health-monitor patterns: track session file sizes with 500KB/2MB tier alerts, monitor heartbeat freshness with 45min/90min tiers, and expose via `gormes health` command with structured JSON output.
- **Trust class:** operator
- **Degraded mode:** Missing session files, stale heartbeat, or unreadable metrics report `health_unavailable` with specific path/error details.
- **Fixture:** `internal/tools/health_test.go`
- **Source refs:**
  - `workspace-link/skills/session-health-monitor/SKILL.md`
  - `workspace-mineru/skills/fleet-admin/SKILL.md`
  - `internal/goncho/service.go` (session store)
  - `fleet-operational-patterns.md`
- **Write scope:**
  - `internal/tools/health.go`
  - `internal/tools/health_test.go`
  - `cmd/gormes/health.go`
  - `webpages/docs/content/building-gormes/progress.json`
- **Test commands:** `go test ./internal/tools -run TestHealth -count=1`
- **Acceptance:**
  - `gormes health` outputs session sizes with tier labels (ok/warn/critical)
  - `gormes health` outputs heartbeat age with freshness labels
  - JSON mode (`--json`) produces machine-parseable structured output
  - Goncho extraction queue depth included in health report
- **Done signal:** `gormes health` command ships with session size, heartbeat, and Goncho queue monitoring.

#### Row 3: Evidence-Before-Claims Quality Gate

- **Phase:** 5.O (CLI Parity)
- **Priority:** P0
- **Slice size:** small
- **Execution owner:** tools
- **Contract:** Port Link's evidence-before-claims pattern: doctor output and build results must include exact counts (pass/fail/skip), not summary claims. Every status line with a count must derive from an actual computation, not a hardcoded narrative. `gormes doctor --offline` must report exact pass/fail/skip per check category.
- **Trust class:** operator
- **Degraded mode:** Uncomputable counts report `count_unavailable` with the reason (missing data, corrupt store, etc.) rather than fabricating a number.
- **Fixture:** `internal/doctor/evidence_test.go`
- **Source refs:**
  - `workspace-link/skills/deep-research/SKILL.md` (evidence standard)
  - `workspace-riju/skills/fecim-physics-validation/SKILL.md` (exact thresholds)
  - `internal/doctor/doctor.go`
  - `fleet-operational-patterns.md`
- **Write scope:**
  - `internal/doctor/evidence.go`
  - `internal/doctor/evidence_test.go`
  - `webpages/docs/content/building-gormes/progress.json`
- **Test commands:** `go test ./internal/doctor -run TestEvidence -count=1`
- **Acceptance:**
  - `gormes doctor --offline` output contains exact pass/fail/skip counts per category
  - No hardcoded "all checks passed" when counts disagree
  - JSON doctor output includes per-check result objects with pass/fail/skip/error status
- **Done signal:** Doctor output uses computed evidence rather than narrative summaries.

### P1 — Short-Term (Next 90 Days)

#### Row 4: Git Delivery Contract Enforcement

- **Phase:** 5.N (Misc Operator Tools)
- **Priority:** P1
- **Slice size:** small
- **Execution owner:** tools
- **Contract:** Enforce the fleet-wide git delivery contract: split commits by concern, commit after each validated slice, push to origin, report hash/branch/push confirmation. gormes-builder skill must include post-commit validation.
- **Trust class:** operator
- **Degraded mode:** Unpushed commits, dirty working tree, or missing remote report `git_delivery_incomplete` rather than silently failing.
- **Fixture:** `internal/tools/git_delivery_test.go`
- **Source refs:**
  - All 6 workspace AGENTS.md files (Git delivery contract)
  - `SHARED-PROTOCOLS.md`
  - `fleet-operational-patterns.md`
- **Write scope:**
  - `internal/tools/git_delivery.go`
  - `internal/tools/git_delivery_test.go`
  - `webpages/docs/content/building-gormes/progress.json`
- **Test commands:** `go test ./internal/tools -run TestGitDelivery -count=1`
- **Acceptance:**
  - Builder skill validates working tree is clean and commits are pushed before declaring row complete
  - `gormes status` reports git delivery state (branch, ahead/behind, unpushed)
  - Split-commit discipline documented in AGENTS.md
- **Done signal:** Git delivery contract enforced by builder skill and visible in status.

#### Row 5: QMD Hybrid Search

- **Phase:** 5.N (Misc Operator Tools)
- **Priority:** P1
- **Slice size:** medium
- **Execution owner:** tools
- **Contract:** Port the fleet's shared QMD hybrid search (BM25 + vector) as `gormes search`. Index all markdown docs in the workspace (AGENTS.md, TOOLS.md, USER.md, MEMORY.md, memory/, sessions/, skills/SKILL.md) for offline keyword + semantic retrieval.
- **Trust class:** operator
- **Degraded mode:** Missing embedding model, corrupt index, or unreadable workspace reports `search_unavailable` with root cause. Falls back to BM25-only search when vector model unavailable.
- **Fixture:** `internal/tools/qmd_test.go`
- **Source refs:**
  - All 6 workspace `skills/qmd/SKILL.md` files
  - `internal/goncho/service.go` (embedding model integration)
  - `fleet-operational-patterns.md`
- **Write scope:**
  - `internal/tools/qmd.go`
  - `internal/tools/qmd_test.go`
  - `cmd/gormes/search.go`
  - `webpages/docs/content/building-gormes/progress.json`
- **Test commands:** `go test ./internal/tools -run TestQMD -count=1`
- **Acceptance:**
  - `gormes search "how to deploy"` returns ranked results from workspace docs
  - BM25 fallback works when no embedding model configured
  - Results include source file path and excerpt context
  - Index updates automatically on doc changes (optional, planned for P2 iteration)
- **Done signal:** `gormes search` ships with BM25 + optional vector hybrid search across workspace docs.

#### Row 6: Session Rollover Automation

- **Phase:** 5.N (Misc Operator Tools)
- **Priority:** P1
- **Slice size:** small
- **Execution owner:** tools
- **Contract:** Port the fleet's session rollover rule (1500KB threshold → write handoff summary → fresh session). `gormes session rollover` exports current session, writes 5-line handoff summary, starts fresh session. Auto-rollover at configurable threshold.
- **Trust class:** operator, system
- **Degraded mode:** Session file too large to export cleanly, corrupt session state, or rollover failure reports `session_rollover_failed` with specific path/error and keeps original session intact.
- **Fixture:** `internal/persistence/session/rollover_test.go`
- **Source refs:**
  - All 6 workspace AGENTS.md (Session reliability rules)
  - `internal/persistence/session/session.go`
  - `internal/goncho/service.go`
  - `fleet-operational-patterns.md`
- **Write scope:**
  - `internal/persistence/session/rollover.go`
  - `internal/persistence/session/rollover_test.go`
  - `cmd/gormes/session_rollover.go`
  - `webpages/docs/content/building-gormes/progress.json`
- **Test commands:** `go test ./internal/persistence/session -run TestRollover -count=1`
- **Acceptance:**
  - `gormes session rollover` exports and starts fresh session
  - Handoff summary includes: session ID, message count, time range, last 3 actions, blockages
  - Auto-rollover triggers at configurable size threshold (default 1500KB)
  - Original session preserved after rollover
- **Done signal:** `gormes session rollover` ships with auto-trigger at configurable threshold.

### P2 — Medium-Term (Next 6 Months)

#### Row 7: Agent Hooks Registry

- **Phase:** 5.I (Plugins Architecture)
- **Priority:** P2
- **Slice size:** medium
- **Execution owner:** tools
- **Contract:** Port OpenClaw's hook registry: `gormes hooks` with list/enable/disable/check/info subcommands. Hooks are inspectable at runtime from gateway config (HOOK.yaml/BOOT.md). Support enable/disable without restart.
- **Trust class:** operator
- **Degraded mode:** Hook config parse failure, missing hook implementation, or runtime error reports per-hook status with degraded hook skipped.
- **Fixture:** `internal/plugins/hooks_test.go`
- **Source refs:**
  - OpenClaw `hooks` CLI surface
  - `internal/gateway/manager.go` (HOOK.yaml loading)
  - `fleet-operational-patterns.md`
- **Write scope:**
  - `internal/plugins/hooks.go`
  - `internal/plugins/hooks_test.go`
  - `cmd/gormes/hooks.go`
  - `webpages/docs/content/building-gormes/progress.json`
- **Test commands:** `go test ./internal/plugins -run TestHooks -count=1`
- **Acceptance:**
  - `gormes hooks list` shows all hooks with enabled/disabled status
  - `gormes hooks enable/disable` toggles without restart
  - `gormes hooks info <hook>` shows source, trigger, and status
  - Runtime toggling does not require process restart
- **Done signal:** `gormes hooks` command ships with list/enable/disable/info.

#### Row 8: Plugin Marketplace + Doctor

- **Phase:** 5.I (Plugins Architecture)
- **Priority:** P2
- **Slice size:** large
- **Execution owner:** tools
- **Contract:** Port OpenClaw's plugin management surface: marketplace discovery (ClawHub-compatible), plugin doctor (load issue reporting), plugin inspect (manifest details), and third-party plugin sandboxing (WASM/subprocess isolation).
- **Trust class:** operator
- **Degraded mode:** Marketplace unreachable, plugin load failure, or sandbox crash reports per-plugin status with degraded plugin skipped.
- **Fixture:** `internal/plugins/marketplace_test.go`
- **Source refs:**
  - OpenClaw `plugins` CLI surface (install, list, enable, disable, inspect, doctor, marketplace, update, uninstall)
  - `internal/plugins/` current implementation
  - `fleet-operational-patterns.md`
- **Write scope:**
  - `internal/plugins/marketplace.go`
  - `internal/plugins/marketplace_test.go`
  - `internal/plugins/doctor.go`
  - `internal/plugins/sandbox.go`
  - `webpages/docs/content/building-gormes/progress.json`
- **Test commands:** `go test ./internal/plugins -run TestMarketplace -count=1`
- **Acceptance:**
  - `gormes plugins marketplace` lists available plugins from configured sources
  - `gormes plugins install` fetches and activates from ClawHub-compatible sources
  - `gormes plugins doctor` reports load issues per plugin
  - Third-party plugins run in WASM or subprocess sandbox
- **Done signal:** Plugin marketplace + doctor shipped with third-party sandboxing.

#### Row 9: First-Run Setup/Readiness

- **Phase:** 5.O (CLI Parity)
- **Priority:** P1
- **Slice size:** medium
- **Execution owner:** tools
- **Contract:** Keep first-run setup on `gormes setup` and machine-readable readiness on `gormes doctor --offline --target terminal --json`: model/provider selection → auth setup → gateway channel configuration → browser/CDP checks → skill discovery → dashboard launch. Match OpenClaw's onboarding depth without adding a deprecated top-level command.
- **Trust class:** operator
- **Degraded mode:** Missing provider credentials, gateway config gaps, or browser unavailability reports per-step status and allows skip with explicit warning.
- **Fixture:** `internal/cli/onboard_test.go`
- **Source refs:**
  - OpenClaw `onboard` command
  - `cmd/gormes/setup.go`
  - `fleet-operational-patterns.md`
  - `implementation-roadmap.md` (Product Hardening Borrow List)
- **Write scope:**
  - `internal/cli/onboard.go`
  - `internal/cli/onboard_test.go`
  - `cmd/gormes/onboard.go`
  - `webpages/docs/content/building-gormes/progress.json`
- **Test commands:** `go test ./internal/cli -run TestOnboard -count=1`
- **Acceptance:**
  - `gormes setup` and doctor readiness cover model → provider → auth → gateway → browser → skills → dashboard
  - Each step can be skipped with explicit warning
  - Already-configured steps are detected and pre-filled
  - End-to-end testable without live credentials (mock provider, fake channel)
- **Done signal:** Public setup/readiness surfaces ship the full first-run flow without registering `gormes onboard`.

#### Row 10: Logs Command

- **Phase:** 5.O (CLI Parity)
- **Priority:** P2
- **Slice size:** small
- **Execution owner:** tools
- **Contract:** Port `openclaw logs` as `gormes logs`: tail gateway file logs, support follow mode (`-f`), filter by level, and stream via RPC if gateway is remote.
- **Trust class:** operator
- **Degraded mode:** Missing log file, permission denied, or remote gateway unreachable reports `logs_unavailable` with root cause.
- **Fixture:** `internal/cli/logs_test.go`
- **Source refs:**
  - OpenClaw `logs` command
  - `must-have-features.md` (CLI Commands gap)
  - `fleet-operational-patterns.md`
- **Write scope:**
  - `internal/cli/logs.go`
  - `internal/cli/logs_test.go`
  - `cmd/gormes/logs.go`
  - `webpages/docs/content/building-gormes/progress.json`
- **Test commands:** `go test ./internal/cli -run TestLogs -count=1`
- **Acceptance:**
  - `gormes logs` shows recent gateway log entries
  - `gormes logs -f` follows live logs
  - `gormes logs --level error` filters by severity
- **Done signal:** `gormes logs` ships with follow mode and level filtering.

---

## New Subphase Proposals

### 5.N.x — Fleet Operational Patterns (within Phase 5.N)

Rows 1, 2, 4, 5, 6 from above belong here. This is a logical grouping within the existing 5.N subphase. No new subphase key needed if rows are added directly to 5.N items.

### 5.E.x — Multi-Instance Management (new subphase)

Currently Phase 5.E is "TTS / Voice / Transcription." Multi-instance fleet management doesn't fit there. Proposal: new subphase 5.E.x (or renumber) for fleet-level concerns.

**Alternative:** Place multi-instance management under 5.N (Misc Operator Tools).

---

## Implementation Sequence

### Sprint 1 (Weeks 1-2) — P0 Foundation
```
Row 1: Blocker Policy Integration
Row 2: Session Health Monitoring
```

### Sprint 2 (Weeks 3-4) — P0 Closeout
```
Row 3: Evidence-Before-Claims Quality Gate
```

### Sprint 3 (Weeks 5-8) — P1 Foundation
```
Row 5: QMD Hybrid Search
Row 4: Git Delivery Contract Enforcement
Row 6: Session Rollover Automation
```

### Sprint 4 (Weeks 9-12) — P1 Closeout
```
Row 9: Interactive Onboarding
Row 7: Agent Hooks Registry
```

### Backlog (Month 3+) — P2
```
Row 8: Plugin Marketplace + Doctor
Row 10: Logs Command
Row 11: Multi-Instance Fleet Management
```

---

## Dependencies

| Row | Depends On | Unblocks |
|-----|-----------|----------|
| Row 1 (Blocker) | Phase 5.N completed | All builder/planner quality |
| Row 2 (Health) | Goncho session store (Phase 3) | Row 6 (Rollover) |
| Row 3 (Evidence) | Doctor command (Phase 1) | All CLI parity work |
| Row 4 (Git Delivery) | Builder skill (Phase 1.D) | CI/CD integration |
| Row 5 (QMD) | Goncho embeddings (Phase 3) | Search UX, Row 9 (Onboard) |
| Row 6 (Rollover) | Row 2 (Health) | Session lifecycle |
| Row 7 (Hooks) | Gateway HOOK.yaml loader (Phase 2) | Row 8 (Plugins) |
| Row 8 (Plugins) | Row 7 (Hooks), Phase 5.I base | Community ecosystem |
| Row 9 (Onboard) | Row 5 (QMD), Phase 5.O base | First-run experience |
| Row 10 (Logs) | Gateway file logging | Operator visibility |

---

## Validation Checklist

Before declaring any row complete:

- [ ] Row TDD fixtures pass (`go test` with row-specific test command)
- [ ] `go run ./cmd/progress validate` passes
- [ ] Row status updated to `complete` in `progress.json`
- [ ] `contract_status` set to `validated`
- [ ] `evidence` block populated with implementation path and test results
- [ ] New commands documented in `cmd/README.md`
- [ ] Fleet operational patterns doc updated with shipped status

---

*Generated: May 1, 2026*
*Source: fleet-operational-patterns.md, all 6 sages workspaces, OpenClaw 2026.3.28*
*Cross-referenced against: progress.json, completion-plan.md, lane-roadmap.md, must-have-features.md*
