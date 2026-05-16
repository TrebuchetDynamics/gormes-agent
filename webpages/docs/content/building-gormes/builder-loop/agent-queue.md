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
## 1. Agentic-porting-kit public repo scaffold

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

## 2. Backlog split C1: lossless multi-file loader/writer behind the single-file API

- Phase: 8 / 8.F
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: Child 1 of the module-split umbrella — the smallest NON-behavior-changing first step. In internal/progress, add the ability to load AND write a split layout (a directory of per-module files, or index + per-module files) BEHIND the existing single-file public API: internal/progress.Load(path) (progress.go:245) must transparently accept EITHER the monolithic progress.json OR the split layout and return the identical in-memory model; add a round-trip pair (e.g. `go run ./cmd/progress split` / `... merge`, or internal Split()/Merge()) that is BYTE-STABLE through the existing stable marshal (internal/progress/progress_marshal.go) — merge(split(x)) == x and validate output identical. Do NOT move any real rows, do NOT change any consumer (cmd/progress, plannerloop, builderloop, status, docs/landing generators), do NOT change validate semantics. This is purely a back-compat shim + a lossless round-trip proven by tests, so a later child can flip the on-disk layout with zero behavior change. Owned Gormes infra.
- Trust class: system
- Ready when: Stable marshal + Load contract understood; existing preservation_test.go round-trip pattern is the template.
- Not ready when: Moves real rows or changes any consumer; Load's public signature/behavior for the monolithic file changes; merge(split(x)) is not byte-stable; validate output differs between layouts.
- Degraded mode: If a split layout is malformed/partial, Load must return a typed error (never a silently partial backlog) and the monolithic path must keep working unchanged.
- Fixture: `internal/progress/split_test.go`
- Write scope: `internal/progress/split.go`, `internal/progress/split_test.go`, `internal/progress/progress.go`, `cmd/progress/main.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/progress -count=1`, `go test ./webpages/docs -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Lossless split↔merge round-trip + dual-layout Load proven by tests; zero behavior/consumer change; monolith still canonical until a later child flips it.
- Acceptance: internal/progress.Load transparently reads both the monolithic file and the split layout into an identical model (test with a fixture in both layouts)., Split()+Merge() (or `cmd/progress split`/`merge`) round-trips byte-stably: merge(split(canonical)) is byte-identical to the stable-marshalled canonical, and `go run ./cmd/progress validate` output is identical for both layouts., No consumer and no real row is changed; monolithic path behavior is unchanged; `go test ./internal/progress -count=1` + `go test ./webpages/docs -count=1` green., Malformed split layout → typed error, monolith still works.
- Source refs: internal/progress/progress.go:245 Load (public API to keep stable), internal/progress/progress_marshal.go (stable key/order marshal — round-trip basis), internal/progress/validate_test.go ; internal/progress/preservation_test.go (existing round-trip/preservation patterns to mirror), cmd/progress/main.go (where a split/merge subcommand would attach)
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
