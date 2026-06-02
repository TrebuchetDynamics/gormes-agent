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
## 1. Hermes toolset distribution manifest and deterministic sampler

- Phase: 5 / 5.N
- Owner: `tools`
- Size: `small`
- Status: `planned`
- Priority: `P1`
- Contract: Port Hermes' toolset_distributions.py contract as a hermetic Go manifest and deterministic sampler: expose the named distribution definitions, descriptions, and percentage weights; validate referenced toolsets through the existing Gormes toolset catalog; sample each toolset independently from an injectable RNG; and guarantee the highest-probability valid toolset is selected when all rolls miss. This row does not run batch/datagen jobs or change operator toolset config persistence.
- Trust class: operator, system
- Ready when: The upstream contract is pure data plus sampling helpers; it can be tested without model calls, provider credentials, batch jobs, or live tools., Gormes already has toolset catalog/config seams under cmd/gormes and internal/platform/cli/toolsets that can validate whether a Hermes distribution names a supported toolset., The parity atom remains missing after stale debug-helper reconciliation, so this row gives the next builder a concrete Hermes tools slice instead of a broad umbrella.
- Not ready when: The builder changes platform_toolsets persistence, TUI /tools behavior, gateway command registry, batch runner execution, or datagen scheduling in this slice., Tests depend on random nondeterminism, live registry discovery, provider credentials, or external package/network access., The implementation silently returns an empty selection for a valid distribution whose rolls all miss, or accepts unknown distributions as success.
- Degraded mode: Unknown distributions return typed unavailable/error evidence; invalid distribution entries are skipped with validation evidence; if every random roll misses, the sampler chooses the highest-probability valid toolset so callers never receive an empty enabled-toolset set when a distribution has valid entries.
- Fixture: `internal/platform/cli/toolsets/distribution_test.go with deterministic RNG fixtures for default, image_gen, safe, terminal_only, unknown distribution, invalid toolset skip, and highest-probability fallback cases`
- Write scope: `internal/platform/cli/toolsets/`, `internal/tools/parity/`, `webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, `docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md`, `webpages/docs/content/building-gormes/architecture_plan/progress.json/modules/tools.json`, `docs/content/building-gormes/architecture_plan/progress.json/modules/tools.json`
- Test commands: `go test ./internal/platform/cli/toolsets -run 'TestHermesToolsetDistribution\|TestToolsetDistribution' -count=1`, `go test ./internal/tools/parity -run 'TestHermesToolsetDistribution\|TestToolsetDistribution' -count=1`, `go run ./cmd/progress validate`, `git diff --check`
- Done signal: Hermetic tests prove Gormes exposes Hermes toolset distribution names/weights and deterministic sampler behavior with validation/fallback evidence and no live batch/datagen execution.
- Acceptance: List/Get helpers expose the Hermes distribution names and descriptions, including default, image_gen, research, science, development, safe, balanced, minimal, terminal_only, terminal_web, creative, reasoning, browser_use, browser_only, browser_tasks, terminal_tasks, and mixed_tasks., Deterministic RNG tests prove independent percentage sampling, validate-toolset filtering, and highest-probability fallback behavior match Hermes semantics., Unknown distributions and invalid toolset references produce stable typed evidence without panics or live side effects., The parity atom can move from missing to covered with exact Go files/tests; broad batch/datagen runner integration remains out of scope.
- Source refs: hermes-agent/toolset_distributions.py:DISTRIBUTIONS, hermes-agent/toolset_distributions.py:get_distribution, hermes-agent/toolset_distributions.py:list_distributions, hermes-agent/toolset_distributions.py:sample_toolsets_from_distribution, hermes-agent/toolset_distributions.py:validate_distribution, hermes-agent/toolsets.py:validate_toolset, cmd/gormes/main.go:toolsetsForToolName, internal/platform/cli/toolsets/, webpages/docs/parity-evidence/HERMES-BEHAVIOR-ATOMS.md: Toolset distributions, docs/content/building-gormes/architecture_plan/hermes-honcho-feature-map.md: Tools, Sandboxes, And Security / Tool registry and toolsets
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
