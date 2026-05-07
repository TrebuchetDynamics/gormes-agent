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
## 1. Sandbox isolation depth selection

- Phase: 5 / 5.U
- Owner: `tools`
- Size: `medium`
- Status: `planned`
- Priority: `P3`
- Contract: Operator can select sandbox isolation depth: process-level (fast, weaker isolation), container-level (Docker/gVisor, balanced), or VM-level (Firecracker, strongest isolation). Default is process-level with transactional rollback.
- Trust class: operator
- Ready when: Transactional executor exists (5.U row 2)
- Not ready when: No sandbox backend available
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/tools/isolation_depth.go`, `internal/tools/isolation_depth_test.go`
- Test commands: `go test ./internal/tools -run TestIsolationDepth -count=1`
- Done signal: Isolation depth tests prove all three levels selectable and process-level works without Docker
- Acceptance: Process-level isolation is the default and requires zero setup, Docker/gVisor isolation selectable via config, Firecracker VM isolation selectable via config, Isolation depth is per-session configurable, Deeper isolation correctly fails if backend not available
- Source refs: docs/content/papers/safety-and-deployment.md, OpenSandbox (github.com/alibaba/OpenSandbox), internal/tools/sandbox.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 2. Behavioral pattern extraction from session logs

- Phase: 6 / 6.K
- Owner: `orchestrator`
- Size: `large`
- Status: `planned`
- Priority: `P3`
- Contract: Mine session logs and tool execution audits for behavioral patterns: which tool sequences succeed vs fail, which reasoning patterns precede good outcomes, which response styles correlate with user satisfaction. Patterns feed into the self-evolution loop as candidate mutations.
- Trust class: operator
- Ready when: Session logs are structured and queryable, Tool execution audit log exists (Phase 3.E.2)
- Not ready when: No structured session data available, Tool audit log not yet implemented
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/hermes/pattern_extractor.go`, `internal/hermes/pattern_extractor_test.go`
- Test commands: `go test ./internal/hermes -run TestPatternExtractor -count=1`
- Done signal: Pattern extractor tests prove successful and failed patterns are correctly identified from log data
- Acceptance: Pattern extractor identifies tool sequences with >80% success rate, Identifies tool sequences with <30% success rate (anti-patterns), Extracts reasoning patterns preceding successful tool calls, Patterns stored in Goncho as structured behavioral knowledge, Pattern extraction is offline (does not run during agent turns)
- Source refs: docs/content/papers/agentic-os-design.md, Hermes Agent GEPA engine, Generative Agents reflection mechanism (Park et al. 2023), internal/goncho/extractor.go, internal/hermes/turn.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 3. Skill code execution runtime

- Phase: 6 / 6.L
- Owner: `skills`
- Size: `large`
- Status: `planned`
- Priority: `P2`
- Contract: Skills are not just markdown instructions — they contain executable code that can be run in a sandboxed environment. This mirrors Voyager's code-as-action pattern: skills are validated, sandboxed, and can be composed by the agent at runtime.
- Trust class: operator, system
- Ready when: Skill loader parses structured skill files, Sandbox execution exists for tool calls
- Not ready when: Skill files are plain text only (no code blocks), No sandbox isolation available
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/skills/code_executor.go`, `internal/skills/code_executor_test.go`, `internal/skills/skill_runtime.go`
- Test commands: `go test ./internal/skills -run TestCodeExecutor -count=1`, `go test ./internal/skills -run TestSkillRuntime -count=1`
- Done signal: Code executor tests prove skills with code blocks execute in sandbox with input/output contract
- Acceptance: Skill files with code blocks are executable in sandbox, Execution is sandboxed with the same isolation as tool calls, Skill code has access to skill-defined dependencies, Execution timeout prevents runaway skills, Execution output is captured and returned to agent, Skill can define input parameters accepted from agent
- Source refs: docs/content/papers/foundational-architectures.md, Voyager (arXiv:2305.16291), internal/skills/loader.go, internal/skills/executor.go, internal/tools/sandbox.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 4. Skill dependency resolution and composition

- Phase: 6 / 6.L
- Owner: `skills`
- Size: `medium`
- Status: `planned`
- Priority: `P3`
- Contract: Skills can declare dependencies on other skills. The runtime resolves the dependency graph before execution. The agent can compose skills by chaining: output of Skill A feeds into input of Skill B. Dependencies are validated at load time.
- Trust class: operator
- Ready when: Skill code execution exists (6.L row 1), Skills have structured metadata with dependency declarations
- Not ready when: Skills have no dependency model, Code execution runtime not available
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/skills/dependency_resolver.go`, `internal/skills/dependency_resolver_test.go`, `internal/skills/composer.go`
- Test commands: `go test ./internal/skills -run TestDependencyResolver -count=1`, `go test ./internal/skills -run TestComposer -count=1`
- Done signal: Dependency tests prove circular deps rejected and chained composition works with error attribution
- Acceptance: Skill dependency graph resolved at load time, Circular dependencies detected and rejected with clear error, Missing dependencies reported with skill name and missing dep, Agent can chain Skill A output → Skill B input, Composition failures surface which step in the chain failed, Load-time validation catches 100% of dependency errors before execution
- Source refs: docs/content/papers/foundational-architectures.md, Voyager skill library composition, internal/skills/loader.go, internal/skills/registry.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

## 5. Skill validation on load with execution proof

- Phase: 6 / 6.L
- Owner: `skills`
- Size: `small`
- Status: `planned`
- Priority: `P2`
- Contract: When a skill is loaded or created, run a lightweight validation: parse code blocks, execute in sandbox with a canary input, verify output contract. Skills that fail validation are marked as broken and not offered to the agent. Passing skills carry a 'validated' trust marker.
- Trust class: system
- Ready when: Skill code execution exists (6.L row 1)
- Not ready when: No sandbox execution available for validation
- Degraded mode: -
- Fixture: `-`
- Write scope: `internal/skills/validator.go`, `internal/skills/validator_test.go`
- Test commands: `go test ./internal/skills -run TestValidator -count=1`
- Done signal: Validator tests prove broken skills are caught at load time with clear error messages
- Acceptance: Skills validated on load before appearing in agent's tool list, Canary execution with minimal input verifies basic functionality, Broken skills marked with error details (not silently skipped), Validation is fast (<500ms per skill, runs in background goroutine), Operator can force-load a broken skill with explicit override flag, Validation results visible in skill registry status
- Source refs: docs/content/papers/foundational-architectures.md, Voyager iterative prompting with execution feedback, internal/skills/loader.go
- Why now: Contract metadata is present; ready for a focused spec or fixture slice.

<!-- PROGRESS:END -->
