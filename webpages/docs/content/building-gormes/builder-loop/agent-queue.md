---
title: "Agent Queue"
weight: 20
aliases:
  - /building-gormes/agent-queue/
---

# Agent Queue

This page is generated from the canonical progress file:
`docs/content/building-gormes/architecture_plan/progress.json`.

It lists unblocked, non-umbrella contract rows that are ready for a focused
skill-driven implementation attempt. Each card carries the execution owner,
slice size, contract, trust class, degraded-mode requirement, fixture target,
write scope, test commands, done signal, acceptance checks, and source
references.

Shared skill handoff facts live in [Skill Builder Handoff](../builder-loop-handoff/):
the main skill entrypoint, plan, candidate source, generated docs, tests, and
candidate policy. Keep those control-plane facts in `meta.builder_loop`, and
keep row-specific execution facts in `progress.json`.

If the generated list is empty, do not switch to an ad hoc TODO list. Route
through `gormes-planner`, repair one planned/draft row until it satisfies the
handoff contract, validate `progress.json`, and then return to builder
selection.

<!-- PROGRESS:START kind=agent-queue -->
## 1. gormes doctor ◆ Profiles section content

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P3`
- Contract: Add `◆ Profiles` diagnostic content to `gormes doctor` (parity with hermes_cli/doctor.py@55c9f3206:1768): per-profile gateway-running + model summary. Gormes' profile model differs from Hermes' — this row first needs the active Gormes profile inventory contract identified (owned divergence) before TDD; emit one CheckResult{Name:"Profiles"} listing each Gormes profile with its gateway state + resolved model, Gormes-owned wording/paths. Out of scope until the Gormes profile contract is mapped.
- Trust class: -
- Ready when: ◆-section grouping shipped; the Gormes profile inventory contract is identified (owned divergence vs Hermes profiles.py).
- Not ready when: Implemented before the Gormes profile contract is mapped, or copies Hermes profile semantics blindly.
- Degraded mode: No profiles configured → single PASS item 'default profile only'; unresolved profile → WARN, never panic.
- Fixture: `internal/doctor/doctor_profiles_test.go`
- Write scope: `internal/doctor/doctor_profiles.go`, `internal/doctor/doctor_profiles_test.go`, `internal/doctor/doctor_section.go`, `cmd/gormes/doctor.go`, `cmd/gormes/doctor_profiles_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/doctor ./cmd/gormes -run 'Doctor' -count=1`, `go test ./cmd/gormes -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Fixtures prove the ◆ Profiles per-profile gateway+model summary with Gormes-owned profile semantics — citing hermes 55c9f3206 doctor.py:1768; preceded by a Gormes profile-contract mapping pass.
- Acceptance: `gormes doctor` renders `◆ Profiles` with each Gormes profile's gateway state + model in Gormes-owned wording; no-profile case is a clean PASS.
- Source refs: ../hermes-agent/hermes_cli/doctor.py@55c9f3206:1768:◆ Profiles (per-profile gateway + model), ../hermes-agent/hermes_cli/profiles.py@55c9f3206 (Hermes profile model — owned-divergence reference), Gormes profile/config seams (internal/config profile handling) — identify active contract before TDD
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 2. gormes doctor ◆ Security Advisories section content

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `large`
- Status: `planned`
- Priority: `P3`
- Contract: Add `◆ Security Advisories` diagnostic content to `gormes doctor` (parity with hermes_cli/doctor.py@55c9f3206:350). Upstream uses hermes_cli.security_advisories (detect_compromised / filter_unacked / full_remediation_text / get_acked_ids) to scan for compromised dependencies and surface unacked advisories with remediation + ack state. Gormes has no Go advisory dataset/ack-state subsystem — this row is the LARGEST child and carries its own dependency: design + port a Gormes-owned advisory source + ack-state store (internal/security or similar), THEN a doctor check that emits CheckResult{Name:"Security Advisories"} as the first section. Owned divergence: Gormes-owned advisory data + `~/.gormes` ack store, never the Python advisory DB. Likely needs gormes-interface-designer for the advisory-store boundary before TDD.
- Trust class: -
- Ready when: ◆-section grouping shipped; a Gormes advisory-store interface is designed (gormes-interface-designer) and the advisory data source decided.
- Not ready when: Attempted as a doctor-only slice without the advisory dataset + ack-state subsystem; or embeds/depends on the Python advisory DB.
- Degraded mode: No advisory data available / fetch disabled (--offline) → SKIP 'advisory scan skipped' rather than WARN; ack store missing → treat all as unacked, never panic.
- Fixture: `internal/security/advisories_test.go`
- Write scope: `internal/security/advisories.go`, `internal/security/advisories_test.go`, `internal/doctor/doctor_security_advisories.go`, `internal/doctor/doctor_security_advisories_test.go`, `internal/doctor/doctor_section.go`, `cmd/gormes/doctor.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/security ./internal/doctor ./cmd/gormes -run 'Advisor\|Doctor' -count=1`, `go test ./cmd/gormes -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Fixtures prove a Gormes-owned advisory+ack subsystem and the ◆ Security Advisories doctor section (unacked listing + remediation + --offline SKIP), no Python advisory-DB dependency — citing hermes 55c9f3206 doctor.py:350 + security_advisories.py.
- Acceptance: A Gormes-owned advisory + ack-state subsystem exists with tests; `gormes doctor` renders `◆ Security Advisories` first, listing unacked advisories with Gormes-owned remediation, --offline → clean SKIP., No dependency on Hermes' Python advisory DB; ack store under ~/.gormes.
- Source refs: ../hermes-agent/hermes_cli/doctor.py@55c9f3206:350:◆ Security Advisories, ../hermes-agent/hermes_cli/security_advisories.py@55c9f3206 (detect_compromised/filter_unacked/full_remediation_text/get_acked_ids — port reference), Gormes has no equivalent — new internal/security advisory + ack-state subsystem required (owned)
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Agentic-porting-kit public repo scaffold

- Phase: 8 / 8.E
- Owner: `skills`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout.
- Trust class: operator
- Ready when: Agentic-porting-kit extraction spec is complete., GitHub authentication can create or push to TrebuchetDynamics/agentic-porting-kit, or the operator has created the empty repo., The public repo name is confirmed as agentic-porting-kit or an equivalent name before the first push.
- Not ready when: No authenticated path exists to create or update the public TrebuchetDynamics repo., The builder plans to edit Gormes' repo-local skills in place instead of copied kit skills., The standalone example still requires cloning Gormes or running cmd/progress.
- Degraded mode: Without the public scaffold, the methodology remains inspectable only inside Gormes and cannot be cited or reused by other teams.
- Fixture: `TrebuchetDynamics/agentic-porting-kit:examples/python-greeter-to-go/progress.json`
- Write scope: `(separate repo) README.md`, `(separate repo) LICENSE`, `(separate repo) schemas/progress.schema.json`, `(separate repo) scripts/validate-example.sh`, `(separate repo) skills/`, `(separate repo) examples/python-greeter-to-go/`, `README.md`, `docs/content/building-gormes/strategy/success-plan.md`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `cd ${AGENTIC_PORTING_KIT_REPO:-../agentic-porting-kit} && ./scripts/validate-example.sh`, `go run ./cmd/progress validate`, `go test ./webpages/docs -count=1`
- Done signal: Public repo URL, standalone validation output, and Gormes backlink updates are recorded in the completed row note.
- Acceptance: Public repo exists with README.md, LICENSE, schemas/progress.schema.json, scripts/validate-example.sh, skills/, and examples/python-greeter-to-go/., README.md explains the kit independent of Gormes/Hermes and includes Codex plus Claude Code loading instructions., Each copied skill uses the porting-* name from the extraction spec and replaces hard-coded Gormes paths with target-repo variables., scripts/validate-example.sh validates the example progress file and runs the example tests without cloning Gormes., Gormes README.md and success-plan.md record the public repo URL after the repo is reachable.
- Source refs: docs/content/building-gormes/strategy/agentic-porting-kit.md, docs/content/building-gormes/strategy/success-plan.md, webpages/docs/development-skills/gormes-planner/SKILL.md, webpages/docs/development-skills/gormes-builder/SKILL.md, webpages/docs/development-skills/gormes-tdd-slice/SKILL.md, webpages/docs/development-skills/gormes-parity-auditor/SKILL.md, webpages/docs/development-skills/gormes-references/SKILL.md, webpages/docs/development-skills/gormes-skill-manager/SKILL.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
