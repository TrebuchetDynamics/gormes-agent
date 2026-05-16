---
title: "Next Slices"
weight: 30
aliases:
  - /building-gormes/next-slices/
---

# Next Slices

This page is generated from the canonical progress file and lists the highest
leverage contract-bearing roadmap rows to execute next.

The ordering is:

1. unblocked `P0` handoffs;
2. active `in_progress` rows;
3. `fixture_ready` rows;
4. unblocked rows that unblock other slices;
5. remaining `draft` contract rows.

Use this page when choosing implementation work. If a row is too broad, split
the row in `progress.json` before assigning it.

If no slices are listed, the next correct action is planner work: choose one
planned row from `progress.json` or a phase page and add enough contract detail
for it to appear here. Do not infer that an empty generated list means the
roadmap is complete.

<!-- PROGRESS:START kind=next-slices -->
| Phase | Slice | Contract | Trust class | Fixture | Why now |
|---|---|---|---|---|---|
| 5 / 5.O | Gormes update release planner and dry-run contract | `gormes update` must distinguish normal release-installed operators from managed source checkouts before it mutates anything. For release installs, the command plans against trusted GitHub Releases: detect install layout, current version/build, OS/arch artifact name, channel policy (`stable` by default, `development` explicit), latest release metadata, snapshot path, components to update, blocked reasons, and whether an update is available. `gormes update --dry-run` prints the exact non-mutating plan; `gormes update --check` performs no mutation and exits 0 when current, 10 when an update is available, and nonzero for check errors. Existing source-checkout behavior from `Self-update command lifecycle safety` remains the managed/dev path and must not be silently used for release installs. | operator, system | `cmd/gormes/update_release_plan_test.go; internal/cli/update_release_plan_test.go` | Unblocks Gormes update verified binary swap and rollback. |
| 1 / 5.X | Termux real-device smoke evidence | Capture a dated real-device no-root Android Termux smoke record for the current release: install via repo-root install.sh release asset, run gormes version, gormes doctor --offline --json, gormes config check, initialize SQLite/Goncho state, and run a provider-backed gormes chat -q "hello from Termux" when a test credential is available. The evidence must record Android/Termux versions, device arch, install method, and any caveats without leaking credentials. | operator, system | `webpages/docs/content/install/termux-smoke.md or release evidence note` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 1 / 5.X | Termux remote execution guidance | Document and, where useful, add setup/status guidance for using Termux Gormes as the mobile operator/controller while SSHing to stronger machines for heavy builds, Docker, local browser automation, and GPU/local model inference. The guidance must preserve PC-like local Gormes CLI behavior while making remote execution the credible path for workstation/server workloads. | operator, system | `webpages/docs/content/install/ Termux remote-execution docs` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.E | Agentic-porting-kit public repo scaffold | Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout. | operator | `TrebuchetDynamics/agentic-porting-kit:examples/python-greeter-to-go/progress.json` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
