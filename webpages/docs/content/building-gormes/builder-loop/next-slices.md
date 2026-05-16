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
| 8 / 8.E | Agentic-porting-kit public repo scaffold | Create the public TrebuchetDynamics/agentic-porting-kit repository from the extraction spec with README, LICENSE, progress schema, validation script, six renamed porting skills, and a tiny Python-greeter-to-Go example. The copied skills must load in a fresh Codex or Claude Code session without depending on the Gormes checkout. | operator | `TrebuchetDynamics/agentic-porting-kit:examples/python-greeter-to-go/progress.json` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.F | Stop git-tracking duplicate landing progress mirrors (build-time generate) | The canonical backlog webpages/docs/content/building-gormes/architecture_plan/progress.json (5.2 MB) is duplicated VERBATIM into two git-tracked landing mirrors, so every progress edit is a ~3-file multi-MB diff (~10.4 MB pure duplication tracked). Make both mirrors build-time generated, not committed. CONFIRMED generation today: (1) webpages/landing/src/data/progress.json is COPIED by webpages/landing/scripts/sync-assets.mjs:101-103 from the canonical file; (2) webpages/landing/legacy/go-renderer/internal/site/data/progress.json is consumed via `//go:embed data/progress.json` in webpages/landing/legacy/go-renderer/internal/site/progress.go:13 (so it MUST exist as a file in that package dir at `go build`/`go test` time — cannot simply gitignore without a regenerate-before-build step or repointing the embed). Approach: (a) add both mirror paths to .gitignore and `git rm --cached` them; (b) ensure `go run ./cmd/progress write` (and/or sync-assets.mjs) regenerates BOTH from the canonical file; (c) handle the go:embed constraint so `go test ./webpages/... -count=1` and the landing build still pass from a clean checkout — either repoint the embed at a generated path produced before compilation, or have the test/build harness regenerate the embedded file first (the row's builder must pick the smallest safe option and document it). Do NOT change the canonical file format or the rendered site output. Owned Gormes infra cleanup (not Hermes parity). | system | `internal/buildscripts_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.F | Compact completed-row shipped-evidence notes to a one-line pointer | 1001 of 1020 progress rows are status=complete and carry long multi-paragraph SHIPPED-evidence `note` prose that is redundant with git history (the commit holds the detail). Add a Gormes-owned compaction that rewrites a COMPLETED row's verbose `note` to a one-line pointer `SHIPPED <YYYY-MM-DD> <shortSHA\|see git log> — <one-line behavior>`, PRESERVING name/status/contract_status/provenance/acceptance/contract/source_refs/write_scope (only the prose `note` is compacted — nothing else is lost; not_ready_when must forbid touching any other field). Prefer extending cmd/progress (e.g. a `go run ./cmd/progress compact` maintenance subcommand) over a new binary; the compaction must be idempotent, must NOT change `go run ./cmd/progress validate` semantics or the progress-row contract schema, and must be fully reversible by git. Scope decision: SHIP the compaction helper + an ONGOING rule (future completions write a one-line note) and a GUARDED opt-in one-time sweep of existing complete rows (the bulk sweep is a large mechanical diff — keep it a separate explicit invocation, not automatic, so it can land as its own commit). Materially reduces byte size without losing provenance (git still holds full evidence). Owned Gormes infra. | system | `internal/progress/compact_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
| 8 / 8.F | Backlog split C1: lossless multi-file loader/writer behind the single-file API | Child 1 of the module-split umbrella — the smallest NON-behavior-changing first step. In internal/progress, add the ability to load AND write a split layout (a directory of per-module files, or index + per-module files) BEHIND the existing single-file public API: internal/progress.Load(path) (progress.go:245) must transparently accept EITHER the monolithic progress.json OR the split layout and return the identical in-memory model; add a round-trip pair (e.g. `go run ./cmd/progress split` / `... merge`, or internal Split()/Merge()) that is BYTE-STABLE through the existing stable marshal (internal/progress/progress_marshal.go) — merge(split(x)) == x and validate output identical. Do NOT move any real rows, do NOT change any consumer (cmd/progress, plannerloop, builderloop, status, docs/landing generators), do NOT change validate semantics. This is purely a back-compat shim + a lossless round-trip proven by tests, so a later child can flip the on-disk layout with zero behavior change. Owned Gormes infra. | system | `internal/progress/split_test.go` | Contract metadata is present; ready for a focused spec or fixture slice. |
<!-- PROGRESS:END -->
