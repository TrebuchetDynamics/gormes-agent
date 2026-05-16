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

## 2. Stop git-tracking duplicate landing progress mirrors (build-time generate)

- Phase: 8 / 8.F
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: The canonical backlog webpages/docs/content/building-gormes/architecture_plan/progress.json (5.2 MB) is duplicated VERBATIM into two git-tracked landing mirrors, so every progress edit is a ~3-file multi-MB diff (~10.4 MB pure duplication tracked). Make both mirrors build-time generated, not committed. CONFIRMED generation today: (1) webpages/landing/src/data/progress.json is COPIED by webpages/landing/scripts/sync-assets.mjs:101-103 from the canonical file; (2) webpages/landing/legacy/go-renderer/internal/site/data/progress.json is consumed via `//go:embed data/progress.json` in webpages/landing/legacy/go-renderer/internal/site/progress.go:13 (so it MUST exist as a file in that package dir at `go build`/`go test` time — cannot simply gitignore without a regenerate-before-build step or repointing the embed). Approach: (a) add both mirror paths to .gitignore and `git rm --cached` them; (b) ensure `go run ./cmd/progress write` (and/or sync-assets.mjs) regenerates BOTH from the canonical file; (c) handle the go:embed constraint so `go test ./webpages/... -count=1` and the landing build still pass from a clean checkout — either repoint the embed at a generated path produced before compilation, or have the test/build harness regenerate the embedded file first (the row's builder must pick the smallest safe option and document it). Do NOT change the canonical file format or the rendered site output. Owned Gormes infra cleanup (not Hermes parity).
- Trust class: system
- Ready when: Generation paths for both mirrors confirmed (done this pass: sync-assets.mjs copy + go:embed legacy)., A from-clean-checkout build/test path that regenerates the embedded mirror before compiling is identified.
- Not ready when: A mirror is gitignored but the go:embed legacy package still needs a committed file and the build now fails from a clean checkout., The canonical progress.json schema or the rendered site output changes; the mirrors stop being byte-faithful derivations of the canonical file., Stale/empty mirror is silently embedded instead of failing loudly.
- Degraded mode: If a mirror cannot be regenerated, the build/test must fail loudly with a clear 'run go run ./cmd/progress write' message — never silently embed a stale or empty mirror.
- Fixture: `internal/buildscripts_test.go`
- Write scope: `.gitignore`, `webpages/landing/scripts/sync-assets.mjs`, `webpages/landing/legacy/go-renderer/internal/site/progress.go`, `cmd/progress/main.go`, `internal/buildscripts_test.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go run ./cmd/progress write`, `go test ./webpages/docs ./webpages/... -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Mirrors untracked + build-time generated; clean-checkout build/tests green; progress edits no longer triple-diff; site output byte-unchanged.
- Acceptance: Both webpages/landing/legacy/go-renderer/internal/site/data/progress.json and webpages/landing/src/data/progress.json are untracked (in .gitignore, git rm --cached) and regenerated by the build pipeline from the canonical file., `go run ./cmd/progress write`, `go test ./webpages/docs -count=1`, `go test ./webpages/... -count=1`, and the landing build all pass from a clean checkout (no committed mirrors)., A single canonical progress-row edit produces a 1-file diff (canonical + its generated building-gormes docs), not a 3-file multi-MB diff., Rendered site progress output is unchanged (byte-equal to pre-change generation).
- Source refs: webpages/landing/scripts/sync-assets.mjs:101-103 (src mirror copy source), webpages/landing/legacy/go-renderer/internal/site/progress.go:13 //go:embed data/progress.json + :20 ReadFile, cmd/progress/main.go (canonical regenerator); internal/progress/progress.go Load, .gitignore ; AGENTS.md progress-contract lines 89/96/100
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Compact completed-row shipped-evidence notes to a one-line pointer

- Phase: 8 / 8.F
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P2`
- Contract: 1001 of 1020 progress rows are status=complete and carry long multi-paragraph SHIPPED-evidence `note` prose that is redundant with git history (the commit holds the detail). Add a Gormes-owned compaction that rewrites a COMPLETED row's verbose `note` to a one-line pointer `SHIPPED <YYYY-MM-DD> <shortSHA\|see git log> — <one-line behavior>`, PRESERVING name/status/contract_status/provenance/acceptance/contract/source_refs/write_scope (only the prose `note` is compacted — nothing else is lost; not_ready_when must forbid touching any other field). Prefer extending cmd/progress (e.g. a `go run ./cmd/progress compact` maintenance subcommand) over a new binary; the compaction must be idempotent, must NOT change `go run ./cmd/progress validate` semantics or the progress-row contract schema, and must be fully reversible by git. Scope decision: SHIP the compaction helper + an ONGOING rule (future completions write a one-line note) and a GUARDED opt-in one-time sweep of existing complete rows (the bulk sweep is a large mechanical diff — keep it a separate explicit invocation, not automatic, so it can land as its own commit). Materially reduces byte size without losing provenance (git still holds full evidence). Owned Gormes infra.
- Trust class: system
- Ready when: The progress-row contract schema + marshal stable-ordering are understood (internal/progress/progress_marshal.go); a completed row's other fields are provably untouched by the compaction.
- Not ready when: Compaction edits any field other than `note`, drops contract/acceptance/provenance, changes validate semantics or row identity, or is not reversible by git., The one-time bulk sweep runs automatically/implicitly rather than as a separate explicit guarded invocation.
- Degraded mode: If a completed row's note cannot be safely parsed for a one-line summary, leave it unchanged (never lossily truncate mid-sentence or drop a non-note field).
- Fixture: `internal/progress/compact_test.go`
- Write scope: `cmd/progress/main.go`, `internal/progress/compact.go`, `internal/progress/compact_test.go`, `internal/progress/progress.go`, `docs/content/building-gormes/architecture_plan/progress.json`
- Test commands: `go test ./internal/progress -count=1`, `go test ./webpages/docs -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Completed-row notes compact to a one-line pointer with all other fields preserved; validate/tests green; size materially reduced; reversible by git.
- Acceptance: A `cmd/progress` compaction (subcommand or documented helper) rewrites only completed-row `note` fields to the one-line `SHIPPED <date> <sha\|see git log> — <summary>` form, idempotently., All non-note fields are byte-identical after compaction (test asserts field-level preservation); `go run ./cmd/progress validate` and `go test ./internal/progress -count=1` + `go test ./webpages/docs -count=1` stay green., An opt-in one-time sweep materially reduces progress.json size; git history still contains the full pre-compaction evidence., Future completions follow the one-line-note rule (documented in references/progress-row-contract.md / the gormes-* skills).
- Source refs: cmd/progress/main.go (validate/write entrypoint to extend with compact), internal/progress/progress.go Load/marshal ; internal/progress/progress_marshal.go (stable key order), internal/progress/validate_test.go ; references/progress-row-contract.md, docs/content/building-gormes/architecture_plan/progress.json
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. Backlog split C1: lossless multi-file loader/writer behind the single-file API

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
