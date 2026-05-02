#!/usr/bin/env python3
"""Add fleet operational pattern rows to progress.json safely via JSON manipulation."""

import json
import sys

PROGRESS_PATH = "docs/content/building-gormes/architecture_plan/progress.json"

# All new rows organized by phase and subphase
NEW_ROWS = {
    "5": {
        "5.B": [
            {
                "name": "Sandbox Policy Explain",
                "status": "planned",
                "priority": "P1",
                "contract": "Port OpenClaw's sandbox explain: gormes sandbox explain shows the effective trust class, tool allowlist, filesystem scope, and network policy for any agent context. Make sandbox policy operator-visible, reinforcing the 'visible degraded mode' contract.",
                "contract_status": "draft",
                "slice_size": "small",
                "execution_owner": "tools",
                "trust_class": ["operator"],
                "degraded_mode": "Unparseable sandbox config, missing Docker daemon, or policy resolution failure reports sandbox_explain_unavailable with root cause and available fallback modes.",
                "fixture": "internal/tools/sandbox_explain_test.go",
                "source_refs": [
                    "openclaw sandbox explain command",
                    "docs/content/building-gormes/fleet-operational-patterns.md",
                    "docs/content/building-gormes/fleet-integration-plan.md"
                ],
                "ready_when": [
                    "Docker sandbox backend (5.B) is operational.",
                    "Sandbox config schema supports trust class and scope fields."
                ],
                "not_ready_when": [
                    "The row requires live Docker containers for testing.",
                    "The row modifies existing sandbox execution behavior."
                ],
                "unblocks": ["Sandbox operator visibility"],
                "acceptance": [
                    "gormes sandbox explain --agent default shows effective trust class, allowlist, and scope.",
                    "Output includes degraded mode indicators when backends are unavailable.",
                    "Policy inheritance (default -> agent -> session) is clearly shown."
                ],
                "write_scope": [
                    "internal/tools/sandbox_explain.go",
                    "internal/tools/sandbox_explain_test.go",
                    "cmd/gormes/sandbox.go",
                    "docs/content/building-gormes/architecture_plan/progress.json"
                ],
                "test_commands": [
                    "go test ./internal/tools -run TestSandboxExplain -count=1",
                    "go run ./cmd/progress validate"
                ],
                "done_signal": ["gormes sandbox explain ships with policy inheritance display."]
            }
        ],
        "5.I": [
            {
                "name": "Agent Hooks Registry",
                "status": "planned",
                "priority": "P2",
                "contract": "Port OpenClaw's hook registry as gormes hooks with list/enable/disable/check/info subcommands. Hooks are inspectable at runtime from gateway config (HOOK.yaml/BOOT.md). Support enable/disable without restart.",
                "contract_status": "draft",
                "slice_size": "medium",
                "execution_owner": "tools",
                "trust_class": ["operator"],
                "degraded_mode": "Hook config parse failure, missing hook implementation, or runtime error reports per-hook status with degraded hook skipped.",
                "fixture": "internal/plugins/hooks_test.go",
                "source_refs": [
                    "openclaw hooks CLI surface",
                    "internal/gateway/manager.go",
                    "docs/content/building-gormes/fleet-operational-patterns.md",
                    "docs/content/building-gormes/fleet-integration-plan.md"
                ],
                "ready_when": [
                    "Gateway HOOK.yaml loader (Phase 2) is operational.",
                    "Plugin registry supports runtime enable/disable."
                ],
                "not_ready_when": [
                    "The row requires process restart for hook changes.",
                    "The row introduces a new hook definition format conflicting with HOOK.yaml."
                ],
                "unblocks": ["Plugin Marketplace + Doctor (5.I)"],
                "acceptance": [
                    "gormes hooks list shows all hooks with enabled/disabled status.",
                    "gormes hooks enable/disable toggles without restart.",
                    "gormes hooks info <hook> shows source, trigger, and status."
                ],
                "write_scope": [
                    "internal/plugins/hooks.go",
                    "internal/plugins/hooks_test.go",
                    "cmd/gormes/hooks.go",
                    "docs/content/building-gormes/architecture_plan/progress.json"
                ],
                "test_commands": [
                    "go test ./internal/plugins -run TestHooks -count=1",
                    "go run ./cmd/progress validate"
                ],
                "done_signal": ["gormes hooks command ships with list/enable/disable/info."]
            },
            {
                "name": "Plugin Marketplace + Doctor",
                "status": "planned",
                "priority": "P2",
                "contract": "Port OpenClaw's plugin management surface: marketplace discovery (ClawHub-compatible), plugin doctor (load issue reporting), plugin inspect (manifest details), and third-party plugin sandboxing (WASM/subprocess isolation).",
                "contract_status": "draft",
                "slice_size": "large",
                "execution_owner": "tools",
                "trust_class": ["operator"],
                "degraded_mode": "Marketplace unreachable, plugin load failure, or sandbox crash reports per-plugin status with degraded plugin skipped.",
                "fixture": "internal/plugins/marketplace_test.go",
                "source_refs": [
                    "openclaw plugins CLI surface",
                    "internal/plugins/ current implementation",
                    "docs/content/building-gormes/fleet-operational-patterns.md",
                    "docs/content/building-gormes/fleet-integration-plan.md"
                ],
                "ready_when": [
                    "Agent Hooks Registry (5.I) is operational.",
                    "Plugin install/enable/disable flows are defined."
                ],
                "not_ready_when": [
                    "The row ships without third-party sandboxing.",
                    "The row requires external marketplace services for basic functionality."
                ],
                "unblocks": ["Community plugin ecosystem"],
                "acceptance": [
                    "gormes plugins marketplace lists available plugins from configured sources.",
                    "gormes plugins install fetches and activates from ClawHub-compatible sources.",
                    "gormes plugins doctor reports load issues per plugin.",
                    "Third-party plugins run in WASM or subprocess sandbox."
                ],
                "write_scope": [
                    "internal/plugins/marketplace.go",
                    "internal/plugins/marketplace_test.go",
                    "internal/plugins/doctor.go",
                    "internal/plugins/sandbox.go",
                    "docs/content/building-gormes/architecture_plan/progress.json"
                ],
                "test_commands": [
                    "go test ./internal/plugins -run 'TestMarketplace|TestDoctor' -count=1",
                    "go run ./cmd/progress validate"
                ],
                "done_signal": ["Plugin marketplace + doctor shipped with third-party sandboxing."]
            }
        ],
        "5.N": [
            {
                "name": "Blocker Policy Integration",
                "status": "planned",
                "priority": "P0",
                "contract": "Port the sages-openclaw fleet blocker protocol into Gormes: classify blockages by type (access|infra|dependency|decision|bug|unknown), record evidence in structured format, auto-pivot to unblocked work, and surface active blockers in gormes status output.",
                "contract_status": "draft",
                "slice_size": "small",
                "execution_owner": "tools",
                "trust_class": ["operator"],
                "degraded_mode": "Unknown blocker type, missing evidence, or inability to pivot reports blocker_unclassified in status rather than crashing or silently stalling.",
                "fixture": "internal/tools/blocker_test.go",
                "source_refs": [
                    "workspace-mineru/AGENTS.md",
                    "workspace-link/AGENTS.md",
                    "docs/content/building-gormes/fleet-operational-patterns.md",
                    "docs/content/building-gormes/fleet-integration-plan.md"
                ],
                "ready_when": [
                    "Gormes status command can surface structured blocker data.",
                    "Planner skill can mark rows as blocked in progress.json."
                ],
                "not_ready_when": [
                    "The row blocks without identifying a pivot task.",
                    "The row adds a new blocking dependency queue outside progress.json."
                ],
                "unblocks": ["Planner/builder quality metrics", "Operator visibility into stalled work"],
                "acceptance": [
                    "gormes status shows active blockers with type, evidence, and owner.",
                    "Blocker record format matches fleet standard.",
                    "gormes-planner can mark rows as blocked in progress.json with blocker metadata.",
                    "gormes-builder auto-pivots when hitting a blocker registered in progress.json."
                ],
                "write_scope": [
                    "internal/tools/blocker.go",
                    "internal/tools/blocker_test.go",
                    "internal/cli/status.go",
                    "docs/content/building-gormes/architecture_plan/progress.json"
                ],
                "test_commands": [
                    "go test ./internal/tools -run TestBlocker -count=1",
                    "go run ./cmd/progress validate"
                ],
                "done_signal": ["Blocker records appear in status output, planner can mark rows blocked, builder auto-pivots."]
            },
            {
                "name": "Session Health Monitoring",
                "status": "planned",
                "priority": "P0",
                "contract": "Port Link's session-health-monitor patterns: track session file sizes with 500KB/2MB tier alerts, monitor heartbeat freshness with 45min/90min tiers, and expose via gormes health command with structured JSON output.",
                "contract_status": "draft",
                "slice_size": "small",
                "execution_owner": "tools",
                "trust_class": ["operator"],
                "degraded_mode": "Missing session files, stale heartbeat, or unreadable metrics report health_unavailable with specific path/error details.",
                "fixture": "internal/tools/health_test.go",
                "source_refs": [
                    "workspace-link/skills/session-health-monitor/SKILL.md",
                    "workspace-mineru/skills/fleet-admin/SKILL.md",
                    "internal/goncho/service.go",
                    "docs/content/building-gormes/fleet-operational-patterns.md",
                    "docs/content/building-gormes/fleet-integration-plan.md"
                ],
                "ready_when": [
                    "Goncho session store exposes session size and heartbeat data.",
                    "CLI health command route is defined in cmd/gormes."
                ],
                "not_ready_when": [
                    "The row depends on live provider credentials for health checks.",
                    "The row introduces a new daemon process for monitoring."
                ],
                "unblocks": ["Session Rollover Automation (5.N)"],
                "acceptance": [
                    "gormes health outputs session sizes with tier labels (ok/warn/critical).",
                    "gormes health outputs heartbeat age with freshness labels (ok/warn/stale).",
                    "JSON mode (--json) produces machine-parseable structured output.",
                    "Goncho extraction queue depth included in health report."
                ],
                "write_scope": [
                    "internal/tools/health.go",
                    "internal/tools/health_test.go",
                    "cmd/gormes/health.go",
                    "docs/content/building-gormes/architecture_plan/progress.json"
                ],
                "test_commands": [
                    "go test ./internal/tools -run TestHealth -count=1",
                    "go run ./cmd/progress validate"
                ],
                "done_signal": ["gormes health command ships with session size, heartbeat, and Goncho queue monitoring."]
            },
            {
                "name": "Evidence-Before-Claims Quality Gate",
                "status": "planned",
                "priority": "P0",
                "contract": "Port Link's evidence-before-claims pattern: doctor output and build results must include exact counts (pass/fail/skip), not summary claims. Every status line with a count must derive from an actual computation, not a hardcoded narrative.",
                "contract_status": "draft",
                "slice_size": "small",
                "execution_owner": "tools",
                "trust_class": ["operator"],
                "degraded_mode": "Uncomputable counts report count_unavailable with the reason (missing data, corrupt store) rather than fabricating a number.",
                "fixture": "internal/doctor/evidence_test.go",
                "source_refs": [
                    "workspace-link/skills/deep-research/SKILL.md",
                    "workspace-riju/skills/fecim-physics-validation/SKILL.md",
                    "internal/doctor/doctor.go",
                    "docs/content/building-gormes/fleet-operational-patterns.md",
                    "docs/content/building-gormes/fleet-integration-plan.md"
                ],
                "ready_when": [
                    "Doctor check registry supports per-check pass/fail/skip/error status.",
                    "Existing doctor output format can be extended with evidence fields."
                ],
                "not_ready_when": [
                    "The row rewrites existing doctor checks instead of wrapping them.",
                    "The row claims counts without actual computation."
                ],
                "unblocks": ["All CLI parity quality work"],
                "acceptance": [
                    "gormes doctor --offline output contains exact pass/fail/skip counts per category.",
                    "No hardcoded 'all checks passed' when counts disagree.",
                    "JSON doctor output includes per-check result objects with pass/fail/skip/error status."
                ],
                "write_scope": [
                    "internal/doctor/evidence.go",
                    "internal/doctor/evidence_test.go",
                    "docs/content/building-gormes/architecture_plan/progress.json"
                ],
                "test_commands": [
                    "go test ./internal/doctor -run TestEvidence -count=1",
                    "go run ./cmd/progress validate"
                ],
                "done_signal": ["Doctor output uses computed evidence rather than narrative summaries."]
            },
            {
                "name": "Git Delivery Contract Enforcement",
                "status": "planned",
                "priority": "P1",
                "contract": "Enforce the fleet-wide git delivery contract: split commits by concern, commit after each validated slice, push to origin, report hash/branch/push confirmation. gormes-builder skill must include post-commit validation.",
                "contract_status": "draft",
                "slice_size": "small",
                "execution_owner": "tools",
                "trust_class": ["operator"],
                "degraded_mode": "Unpushed commits, dirty working tree, or missing remote report git_delivery_incomplete rather than silently failing.",
                "fixture": "internal/tools/git_delivery_test.go",
                "source_refs": [
                    "workspace-link/AGENTS.md",
                    "workspace-mineru/AGENTS.md",
                    "SHARED-PROTOCOLS.md",
                    "docs/content/building-gormes/fleet-operational-patterns.md",
                    "docs/content/building-gormes/fleet-integration-plan.md"
                ],
                "ready_when": [
                    "Builder skill post-commit hook is defined.",
                    "Status command can query git delivery state."
                ],
                "not_ready_when": [
                    "The row implements a new version control system.",
                    "The row bypasses existing git tools."
                ],
                "unblocks": ["CI/CD integration", "Builder quality metrics"],
                "acceptance": [
                    "Builder skill validates working tree is clean and commits are pushed before declaring row complete.",
                    "gormes status reports git delivery state (branch, ahead/behind, unpushed).",
                    "Split-commit discipline documented in AGENTS.md."
                ],
                "write_scope": [
                    "internal/tools/git_delivery.go",
                    "internal/tools/git_delivery_test.go",
                    "docs/content/building-gormes/architecture_plan/progress.json"
                ],
                "test_commands": [
                    "go test ./internal/tools -run TestGitDelivery -count=1",
                    "go run ./cmd/progress validate"
                ],
                "done_signal": ["Git delivery contract enforced by builder skill and visible in status."]
            },
            {
                "name": "QMD Hybrid Search",
                "status": "planned",
                "priority": "P1",
                "contract": "Port the fleet's shared QMD hybrid search (BM25 + vector) as gormes search. Index all markdown docs in the workspace for offline keyword + semantic retrieval, with BM25-only fallback when no embedding model is configured.",
                "contract_status": "draft",
                "slice_size": "medium",
                "execution_owner": "tools",
                "trust_class": ["operator"],
                "degraded_mode": "Missing embedding model, corrupt index, or unreadable workspace reports search_unavailable with root cause. Falls back to BM25-only search when vector model unavailable.",
                "fixture": "internal/tools/qmd_test.go",
                "source_refs": [
                    "workspace-link/skills/qmd/SKILL.md",
                    "workspace-mineru/skills/qmd/SKILL.md",
                    "internal/goncho/service.go",
                    "docs/content/building-gormes/fleet-operational-patterns.md",
                    "docs/content/building-gormes/fleet-integration-plan.md"
                ],
                "ready_when": [
                    "Goncho embedding model integration is available.",
                    "Workspace doc discovery can enumerate markdown files."
                ],
                "not_ready_when": [
                    "The row requires an external search service.",
                    "The row modifies existing Goncho storage schema."
                ],
                "unblocks": ["Interactive Onboarding (5.O)", "Fleet documentation search UX"],
                "acceptance": [
                    "gormes search 'how to deploy' returns ranked results from workspace docs.",
                    "BM25 fallback works when no embedding model configured.",
                    "Results include source file path and excerpt context."
                ],
                "write_scope": [
                    "internal/tools/qmd.go",
                    "internal/tools/qmd_test.go",
                    "cmd/gormes/search.go",
                    "docs/content/building-gormes/architecture_plan/progress.json"
                ],
                "test_commands": [
                    "go test ./internal/tools -run TestQMD -count=1",
                    "go run ./cmd/progress validate"
                ],
                "done_signal": ["gormes search ships with BM25 + optional vector hybrid search across workspace docs."]
            },
            {
                "name": "Session Rollover Automation",
                "status": "planned",
                "priority": "P1",
                "contract": "Port the fleet's session rollover rule (1500KB threshold -> write handoff summary -> fresh session). gormes session rollover exports current session, writes 5-line handoff summary, starts fresh session. Auto-rollover at configurable threshold.",
                "contract_status": "draft",
                "slice_size": "small",
                "execution_owner": "tools",
                "trust_class": ["operator", "system"],
                "degraded_mode": "Session file too large to export cleanly, corrupt session state, or rollover failure reports session_rollover_failed with specific path/error and keeps original session intact.",
                "fixture": "internal/session/rollover_test.go",
                "source_refs": [
                    "workspace-link/AGENTS.md",
                    "workspace-mineru/AGENTS.md",
                    "internal/session/session.go",
                    "internal/goncho/service.go",
                    "docs/content/building-gormes/fleet-operational-patterns.md",
                    "docs/content/building-gormes/fleet-integration-plan.md"
                ],
                "ready_when": [
                    "Session health monitoring (5.N) provides size data.",
                    "Session store supports export and fresh creation."
                ],
                "not_ready_when": [
                    "The row deletes old sessions without confirmation.",
                    "The row depends on external session management tools."
                ],
                "unblocks": ["Session lifecycle management"],
                "acceptance": [
                    "gormes session rollover exports and starts fresh session.",
                    "Handoff summary includes: session ID, message count, time range, last 3 actions, blockages.",
                    "Auto-rollover triggers at configurable size threshold (default 1500KB).",
                    "Original session preserved after rollover."
                ],
                "write_scope": [
                    "internal/session/rollover.go",
                    "internal/session/rollover_test.go",
                    "cmd/gormes/session_rollover.go",
                    "docs/content/building-gormes/architecture_plan/progress.json"
                ],
                "test_commands": [
                    "go test ./internal/session -run TestRollover -count=1",
                    "go run ./cmd/progress validate"
                ],
                "done_signal": ["gormes session rollover ships with auto-trigger at configurable threshold."]
            }
        ],
        "5.O": [
            {
                "name": "Interactive Onboarding",
                "status": "planned",
                "priority": "P1",
                "contract": "Promote gormes onboard from setup alias to full interactive flow: model/provider selection -> auth setup -> gateway channel configuration -> browser/CDP checks -> skill discovery -> dashboard launch. Match OpenClaw's onboarding depth.",
                "contract_status": "draft",
                "slice_size": "medium",
                "execution_owner": "tools",
                "trust_class": ["operator"],
                "degraded_mode": "Missing provider credentials, gateway config gaps, or browser unavailability reports per-step status and allows skip with explicit warning.",
                "fixture": "internal/cli/onboard_test.go",
                "source_refs": [
                    "openclaw onboard command",
                    "cmd/gormes/setup.go",
                    "docs/content/building-gormes/fleet-operational-patterns.md",
                    "docs/content/building-gormes/fleet-integration-plan.md"
                ],
                "ready_when": [
                    "QMD Hybrid Search (5.N) is operational for skill discovery step.",
                    "Setup wizard alias exists as scaffold."
                ],
                "not_ready_when": [
                    "The row requires live credentials for testing.",
                    "The row replaces existing setup without migration path."
                ],
                "unblocks": ["First-run user experience"],
                "acceptance": [
                    "gormes onboard walks through model -> provider -> auth -> gateway -> browser -> skills -> dashboard.",
                    "Each step can be skipped with explicit warning.",
                    "Already-configured steps are detected and pre-filled.",
                    "End-to-end testable without live credentials (mock provider, fake channel)."
                ],
                "write_scope": [
                    "internal/cli/onboard.go",
                    "internal/cli/onboard_test.go",
                    "cmd/gormes/onboard.go",
                    "docs/content/building-gormes/architecture_plan/progress.json"
                ],
                "test_commands": [
                    "go test ./internal/cli -run TestOnboard -count=1",
                    "go run ./cmd/progress validate"
                ],
                "done_signal": ["gormes onboard ships as full interactive first-run flow."]
            },
            {
                "name": "Logs Command",
                "status": "planned",
                "priority": "P2",
                "contract": "Port openclaw logs as gormes logs: tail gateway file logs, support follow mode (-f), filter by level, and stream via RPC if gateway is remote.",
                "contract_status": "draft",
                "slice_size": "small",
                "execution_owner": "tools",
                "trust_class": ["operator"],
                "degraded_mode": "Missing log file, permission denied, or remote gateway unreachable reports logs_unavailable with root cause.",
                "fixture": "internal/cli/logs_test.go",
                "source_refs": [
                    "openclaw logs command",
                    "docs/content/building-gormes/must-have-features.md",
                    "docs/content/building-gormes/fleet-operational-patterns.md",
                    "docs/content/building-gormes/fleet-integration-plan.md"
                ],
                "ready_when": [
                    "Gateway file logging is operational.",
                    "CLI command route is defined in cmd/gormes."
                ],
                "not_ready_when": [
                    "The row introduces a new logging framework.",
                    "The row requires live gateway for testing."
                ],
                "unblocks": ["Operator observability"],
                "acceptance": [
                    "gormes logs shows recent gateway log entries.",
                    "gormes logs -f follows live logs.",
                    "gormes logs --level error filters by severity."
                ],
                "write_scope": [
                    "internal/cli/logs.go",
                    "internal/cli/logs_test.go",
                    "cmd/gormes/logs.go",
                    "docs/content/building-gormes/architecture_plan/progress.json"
                ],
                "test_commands": [
                    "go test ./internal/cli -run TestLogs -count=1",
                    "go run ./cmd/progress validate"
                ],
                "done_signal": ["gormes logs ships with follow mode and level filtering."]
            }
        ]
    }
}


def main():
    with open(PROGRESS_PATH, "r") as f:
        data = json.load(f)

    phases = data["phases"]
    added = 0

    for phase_key, phase_data in NEW_ROWS.items():
        if phase_key not in phases:
            print(f"ERROR: Phase {phase_key} not found in progress.json")
            sys.exit(1)

        subphases = phases[phase_key]["subphases"]
        for sp_key, new_items in phase_data.items():
            if sp_key not in subphases:
                print(f"ERROR: Subphase {sp_key} not found in Phase {phase_key}")
                sys.exit(1)

            existing = subphases[sp_key].get("items", [])
            existing_names = {item["name"] for item in existing}

            for item in new_items:
                if item["name"] in existing_names:
                    print(f"SKIP: '{item['name']}' already exists in {sp_key}")
                    continue
                existing.append(item)
                added += 1
                print(f"ADDED: '{item['name']}' -> Phase {phase_key}.{sp_key}")

    # Write back with same indentation style (2-space)
    with open(PROGRESS_PATH, "w") as f:
        json.dump(data, f, indent=2)
        f.write("\n")

    print(f"\nDone. Added {added} rows. Total items across all phases: {sum(len(sp.get('items',[])) for p in phases.values() for sp in p.get('subphases',{}).values())}")


if __name__ == "__main__":
    main()
