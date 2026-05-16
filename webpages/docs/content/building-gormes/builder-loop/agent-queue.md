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
## 1. Termux real-device smoke evidence

- Phase: 1 / 5.X
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Capture a dated real-device no-root Android Termux smoke record for the current release: install via repo-root install.sh release asset, run gormes version, gormes doctor --offline --json, gormes config check, initialize SQLite/Goncho state, and run a provider-backed gormes chat -q "hello from Termux" when a test credential is available. The evidence must record Android/Termux versions, device arch, install method, and any caveats without leaking credentials.
- Trust class: operator, system
- Ready when: Termux runtime doctor check is complete., Termux install and release smoke guide is complete., A real no-root Android arm64/aarch64 Termux environment is available to the operator.
- Not ready when: The evidence is only CI simulation or local Linux fake TERMUX_VERSION output., The smoke transcript includes raw provider keys, bot tokens, device-private paths beyond normal Termux paths, or personal chat IDs., The smoke uses source build as the primary install path unless the release asset is explicitly unavailable.
- Degraded mode: If no provider credential is available, record provider-backed oneshot as skipped with credential-unavailable evidence; local install/version/doctor/config/Goncho smoke remains required.
- Fixture: `webpages/docs/content/install/termux-smoke.md or release evidence note`
- Write scope: `webpages/docs/content/install/`, `docs/content/building-gormes/architecture_plan/progress.json`, `README.md`
- Test commands: -
- No test required: Manual real-device evidence row; CI simulation cannot replace the Android smoke transcript.
- Done signal: A dated redacted real-device Termux smoke record is checked in and linked from the install docs/progress row.
- Acceptance: Evidence records exact date, device arch, Android version, Termux version, and Gormes version/commit., Evidence shows install.sh release-binary path into $PREFIX/bin/gormes., Evidence includes gormes version, gormes doctor --offline --json, gormes config check, and SQLite/Goncho initialization outputs or redacted summaries., Provider-backed gormes chat -q succeeds or is explicitly skipped for missing test credential., The public compatibility claim remains bounded to the proven support matrix.
- Source refs: install.sh, cmd/gormes/version.go, cmd/gormes/doctor.go, cmd/gormes/config.go, cmd/gormes/goncho.go, internal/doctor/termux.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 2. Termux remote execution guidance

- Phase: 1 / 5.X
- Owner: `docs`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Document and, where useful, add setup/status guidance for using Termux Gormes as the mobile operator/controller while SSHing to stronger machines for heavy builds, Docker, local browser automation, and GPU/local model inference. The guidance must preserve PC-like local Gormes CLI behavior while making remote execution the credible path for workstation/server workloads.
- Trust class: operator, system
- Ready when: Termux runtime doctor check is complete., Termux install docs exist., Current shell/terminal/SSH tool behavior is documented from existing Gormes command surfaces.
- Not ready when: The docs claim local Termux can run Docker, heavy browser automation, GPU/local LLM, or large test suites like a workstation., The guidance introduces a new top-level gormes run command instead of existing gormes chat -q/gateway surfaces., The guidance requires root, privileged Android features, or a specific private server.
- Degraded mode: If no remote host is configured, Termux remains a local CLI/TUI/gateway runtime and doctor/docs explain which heavy workloads are out of local scope.
- Fixture: `webpages/docs/content/install/ Termux remote-execution docs`
- Write scope: `webpages/docs/content/install/`, `README.md`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./webpages/docs -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Termux docs make the mobile-control-plane plus remote-executor architecture concrete without expanding local Termux support beyond proven capability.
- Acceptance: Docs include the architecture: phone equals Gormes controller/light executor, remote host equals heavy build/browser/Docker/GPU executor., Docs give concrete SSH/tmux command examples without adding a new top-level gormes run command., Doctor or install docs point Termux users at remote execution for local browser/Docker/GPU-heavy workloads., The support matrix remains explicit about what is local, optional, and remote/degraded.
- Source refs: cmd/gormes/doctor.go, internal/doctor/termux.go, internal/tools/, webpages/docs/content/install/linux-macos.md
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. gormes doctor actionable issues summary and --fix auto-remediation

- Phase: 5 / 5.O
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P1`
- Contract: `gormes doctor` must end every run with the Hermes parity affordance it currently lacks: an aggregated, actionable issues summary plus a `gormes doctor --fix` auto-remediation path. Today internal/doctor/doctor.go is a flat CheckResult reporter and cmd/gormes/doctor.go prints checks then stops (the transcript ends at `[SKIP] gateway/discord: disabled` with no summary); cmd/gormes/doctor.go has `--offline` but no `--fix`. Port Hermes hermes_cli/doctor.py:run_doctor end-of-report behavior: after all checks, collect every WARN/FAIL into a numbered `Found N issue(s) to address:` list whose count N is computed from actual results (not narrated), each line carrying the Gormes-owned remediation hint, followed by a `Tip: run "gormes doctor --fix" to auto-fix what's possible.` line when any issue is auto-fixable. Add a `--fix` flag that auto-remediates at least the source-backed fixable classes — config schema/version migrate (Hermes doctor.py:693/722) and broken/missing published-command symlink repair (doctor.py:979/1003) — then re-runs the affected checks and reports what was fixed vs still-manual. Owned divergence: all paths and wording are Gormes (`~/.gormes`, `gormes setup`, `gormes config migrate`), never `~/.hermes`/`hermes setup`; the issues count must be evidence-derived consistent with the existing 'doctor counts must be computed' contract. `--offline` continues to skip network checks and `--fix` performs no network remediation under `--offline`. This row does not rework unrelated doctor checks or add new diagnostic sections (Security Advisories / Directory Structure / Skills Hub are separate parity rows).
- Trust class: -
- Ready when: internal/doctor reporter + CheckResult model already exist (multiple complete doctor rows)., Gormes config schema/version + `gormes config migrate` and the install.sh published-command symlink are existing seams the --fix path can call without new subsystems., Tests can drive doctor with injected WARN/FAIL results and a temp GORMES_HOME; no live providers/network required.
- Not ready when: The summary or remediation prints `~/.hermes`, `hermes setup`, or `hermes doctor --fix` instead of Gormes-owned paths/commands., The issue count is hardcoded/narrated rather than computed from actual check results., `--fix` performs network remediation under `--offline`, or reworks/renames unrelated existing doctor checks, or adds the separate Security Advisories/Directory Structure/Skills Hub sections (different rows)., `--fix` claims a fix it did not apply.
- Degraded mode: If a fixable class cannot be auto-remediated (no write permission, ambiguous state, or `--offline`), `--fix` leaves it in the manual issues list with the Gormes remediation hint and a typed reason rather than failing the whole command or claiming a fix that did not happen. With zero issues, doctor prints a clean 'no issues' line and no `--fix` tip.
- Fixture: `cmd/gormes/doctor_fix_test.go`
- Write scope: `cmd/gormes/doctor.go`, `cmd/gormes/doctor_fix_test.go`, `internal/doctor/doctor.go`, `internal/doctor/doctor_fix.go`, `internal/doctor/doctor_fix_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/doctor ./cmd/gormes -run 'Doctor' -count=1`, `go test ./cmd/gormes -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Doctor fixtures prove the computed `Found N issue(s) to address` summary, the conditional `gormes doctor --fix` tip, `--fix` remediating ≥1 fixable class with fixed-vs-manual re-report, `--offline`/`--fix --offline` network skipping, and Gormes-owned wording (no ~/.hermes/hermes), with source evidence citing Hermes 6784c8079 doctor.py run_doctor.
- Acceptance: A doctor run with N WARN/FAIL checks prints a numbered `Found N issue(s) to address:` list where N equals the actual WARN+FAIL count, each line with a Gormes-owned remediation hint., When at least one issue is auto-fixable, doctor prints a `gormes doctor --fix` tip; with zero issues it prints a clean summary and no tip., `gormes doctor --fix` remediates at least one fixable class (config schema/version migrate OR broken/missing published-command symlink repair), re-checks, and reports fixed vs still-manual., `gormes doctor --offline` still skips network checks and `gormes doctor --fix --offline` performs no network remediation., All summary/remediation text uses `~/.gormes`, `gormes setup`, `gormes config migrate` — never `~/.hermes`/`hermes`., Existing doctor checks are unchanged except for being aggregated into the new summary.
- Source refs: ../hermes-agent/hermes_cli/doctor.py@6784c8079:297:run_doctor, ../hermes-agent/hermes_cli/doctor.py@6784c8079:1824:"Found {n} issue(s) to address", ../hermes-agent/hermes_cli/doctor.py@6784c8079:1830:"Tip: run 'hermes doctor --fix' to auto-fix what's possible", ../hermes-agent/hermes_cli/doctor.py@6784c8079:693,722:config migrate fixable issue; :917:WAL checkpoint; :979,1003:symlink repair fixable issue, ../hermes-agent/hermes_cli/main.py@6784c8079:10578:doctor --fix flag, internal/doctor/doctor.go (flat CheckResult reporter — add aggregated WARN/FAIL collection + remediation classification), cmd/gormes/doctor.go (--offline exists; add --fix; currently prints no end-of-report summary), Phase 5.O sibling 'doctorCustomEndpointReadiness check function' (typed CheckResult doctor row pattern); progress.json:40064 'doctor counts must be computed, not narrated' consistency anchor, install.sh published-command/symlink seam + Gormes config schema-version/migrate seam for the two fixable classes
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. Agentic-porting-kit public repo scaffold

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
