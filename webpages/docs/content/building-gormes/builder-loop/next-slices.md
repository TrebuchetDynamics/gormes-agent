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
| 5 / 5.O | gormes doctor ◆ Security Advisories section content | Add `◆ Security Advisories` diagnostic content to `gormes doctor` (parity with hermes_cli/doctor.py@55c9f3206:350). Upstream uses hermes_cli.security_advisories (detect_compromised / filter_unacked / full_remediation_text / get_acked_ids) to scan for compromised dependencies and surface unacked advisories with remediation + ack state. Gormes has no Go advisory dataset/ack-state subsystem — this row is the LARGEST child and carries its own dependency: design + port a Gormes-owned advisory source + ack-state store (internal/security or similar), THEN a doctor check that emits CheckResult{Name:"Security Advisories"} as the first section. Owned divergence: Gormes-owned advisory data + `~/.gormes` ack store, never the Python advisory DB. Likely needs gormes-interface-designer for the advisory-store boundary before TDD. | - | `internal/security/advisories_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.E | Agentic-porting-kit public repo scaffold | Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout. | operator | `TrebuchetDynamics/agentic-porting-kit:examples/python-greeter-to-go/progress.json` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
