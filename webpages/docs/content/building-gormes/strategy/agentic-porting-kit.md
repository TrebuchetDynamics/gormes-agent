---
title: "Agentic Porting Kit Extraction Spec"
weight: 35
---

# Agentic Porting Kit Extraction Spec

**Decided on**: 2026-05-13

## Goal

`agentic-porting-kit` turns the Gormes planner/builder/TDD skill loop into a
standalone TrebuchetDynamics asset. The kit must help an operator port a
source implementation into a target implementation with progress rows,
evidence-backed planning, test-first builder passes, and validation gates.
Gormes should be the best case study, not a runtime dependency.

## Non-Goals

- It is not a replacement for Gormes' repo-local skills. Gormes keeps its
  specialized `gormes-*` skills.
- It is not a generic project-management template. Every row still needs a
  source implementation, a target implementation, tests, and acceptance.
- It does not require a custom binary in v1. The first extraction can be a
  skills repo plus schemas, examples, and validation scripts.

## Public Repo Shape

The public repository should be named `TrebuchetDynamics/agentic-porting-kit`
unless the operator chooses a final equivalent name before creation.

Minimum scaffold:

```text
README.md
LICENSE
schemas/progress.schema.json
scripts/validate-example.sh
skills/porting-skill-manager/SKILL.md
skills/porting-planner/SKILL.md
skills/porting-builder/SKILL.md
skills/porting-tdd-slice/SKILL.md
skills/porting-parity-auditor/SKILL.md
skills/porting-references/SKILL.md
examples/python-greeter-to-go/
```

The example target should stay intentionally small: a Python greeter package
ported to Go is enough. It needs a `progress.json`, one source fixture, one
target package, and a validation script that proves a planned row can move
through plan, test, and completion without cloning Gormes.

## Skill Name Mapping

| Gormes skill | Kit skill |
|---|---|
| `gormes-skill-manager` | `porting-skill-manager` |
| `gormes-planner` | `porting-planner` |
| `gormes-builder` | `porting-builder` |
| `gormes-tdd-slice` | `porting-tdd-slice` |
| `gormes-parity-auditor` | `porting-parity-auditor` |
| `gormes-references` | `porting-references` |

The extracted skills must replace product-specific terms with configurable
terms:

- `Gormes` becomes `target implementation`.
- `Hermes` becomes `source implementation` or `parity oracle`.
- `docs/content/building-gormes/architecture_plan/progress.json` becomes
  `PORTING_PROGRESS_PATH`, defaulting to `progress.json` in the target repo.
- `cmd/progress` becomes a kit validation script or schema check.
- Repo-local branch rules become host-repo rules loaded from that repo's
  `AGENTS.md` or equivalent instruction file.

## README Promise

The README should lead with a concrete operator promise:

> Validation-gated agentic porting for teams moving behavior from one runtime
> to another without losing tests, traceability, or source attribution.

The README must include:

- install/load instructions for Codex and Claude Code;
- the skill map above;
- a three-step quick start against `examples/python-greeter-to-go`;
- how to adapt the progress schema to a new port;
- how the Gormes repo proves the method at larger scale;
- a clear license section.

## Standalone Acceptance

The scaffold is acceptable when:

1. A fresh checkout of `TrebuchetDynamics/agentic-porting-kit` can run
   `scripts/validate-example.sh` successfully without cloning Gormes.
2. The example `progress.json` validates against `schemas/progress.schema.json`.
3. A Codex or Claude Code session can load the `skills/` directory and use
   `porting-planner` to refine one example row.
4. `porting-builder` and `porting-tdd-slice` name the example validation
   command and do not refer to Gormes-only paths.
5. Gormes' README and success plan record the public repository URL after the
   external repo exists.

## Extraction Order

1. Create the public repo scaffold with README, license, schema, example, and
   validation script.
2. Copy the six Gormes skill files into the mapped kit names.
3. Generalize product terms and path assumptions in the copied files only.
4. Run the standalone example validation.
5. Return to Gormes and record the public repo URL in the README, success plan,
   and Phase 8 progress row.
