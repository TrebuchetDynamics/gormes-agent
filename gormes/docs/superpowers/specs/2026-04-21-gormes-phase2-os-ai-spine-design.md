# Gormes Phase 2 — OS-AI Spine Design

**Status:** Approved 2026-04-21 · implementation plan pending
**Depends on:** Phase 2.A, 2.B.1, 2.C, and 2.D shipped on `main`

## Related Documents

- [`gormes/docs/content/building-gormes/architecture_plan/phase-2-gateway.md`](../../content/building-gormes/architecture_plan/phase-2-gateway.md) — current Phase 2 roadmap ledger and priorities.
- [`2026-04-20-gormes-phase2e-subagent-design.md`](2026-04-20-gormes-phase2e-subagent-design.md) — broad Phase 2.E draft for subagent execution.
- [`2026-04-20-gormes-phase2g-skills-design.md`](2026-04-20-gormes-phase2g-skills-design.md) — broad Phase 2.G draft for skill lifecycle.
- [`2026-04-20-gormes-phase2d-cron-design.md`](2026-04-20-gormes-phase2d-cron-design.md) — example of a bounded Go-native subsystem that already follows the kernel-first pattern.
- [`2026-04-20-gormes-phase3e-mirrors-spec.md`](2026-04-20-gormes-phase3e-mirrors-spec.md) — useful reference for operator-visible mirrors and reviewable artifacts.

> This document is the sequencing and boundary contract for the first shipped Phase 2 OS-AI slice. Where the earlier Phase 2.E and 2.G drafts are broader than this first slice, this spec controls the initial implementation.

---

## 1. Goal

Turn Gormes Phase 2 from a capable single-agent bridge into the first real **Operative System AI** layer:

- **Subagents** are the execution units.
- **Skills** are the reusable operator procedures.
- **The kernel** is the scheduler and policy boundary between them.

The first shipped slice must prove three things:

1. Gormes can delegate bounded work into deterministic child runs without corrupting parent state.
2. Gormes can load approved procedural knowledge from local skill artifacts and inject it into parent or child turns predictably.
3. Gormes can draft new candidate skills from successful delegated work without auto-activating them.

This is the Phase 2 spine. It is more important than another adapter because it defines how Gormes compounds operator capability over time instead of remaining a stateless chat shell.

## 2. Why This Slice

Implementing skills before subagent boundaries would reduce the learning loop to a prompt snippet library. Implementing subagents without skills would create a parallel work primitive but no durable procedural memory.

The correct first cut is:

- **2.E0 — deterministic subagent runtime**
- **2.G0 — static skill runtime**
- **2.E1 / 2.G1-lite — one reviewed vertical proof**

That ordering gives Gormes an OS-style model:

- process model: subagents
- procedure model: skills
- policy model: kernel delegation and promotion rules

It also fits strict TDD: the concurrency and state-isolation rules are testable before the learning loop is allowed to change future behavior.

## 3. Non-Goals

- **No autonomous skill auto-activation.** Generated candidate skills stay inactive until explicit promotion.
- **No autonomous skill auto-improvement.** Feedback-driven rewriting belongs to a later Phase 2.G/Phase 6 slice.
- **No LLM-based complexity detector in the first slice.** Candidate extraction is triggered by explicit delegation policy, not a fuzzy global classifier.
- **No cross-turn skill mutation.** Active skills are fixed for the duration of a parent turn and any children spawned from it.
- **No memory-owned delegation semantics.** Phase 3 can support this system, but it does not own subagent or skill policy.
- **No process isolation beyond goroutine + context boundaries.** Separate-process executors can come later if needed.
- **No hidden recursive delegation.** Depth stays low and explicit.
- **No database-first skill store.** The first slice is filesystem-first so artifacts are human-readable and easy to review in git-style workflows.

## 4. Boundary Contract

### 4.1 Kernel owns policy

The kernel decides:

- whether a turn stays local or becomes delegated
- which child spec to launch
- which active skills are injected
- whether a successful child run is eligible for candidate extraction
- how child results are merged back into the parent turn

The kernel does **not** own:

- subagent lifecycle bookkeeping
- skill document parsing
- skill artifact storage

### 4.2 Subagent runtime owns execution

The subagent runtime is a deterministic control plane for child work:

- spawn child run
- stream child events
- enforce timeout and cancellation
- enforce depth limit
- enforce tool allowlist
- collect typed result

It must work even if the skills system is disabled.

### 4.3 Skill runtime owns reusable procedure

The skill runtime owns:

- parsing and validating `SKILL.md`
- loading active skills from disk
- deterministic skill selection
- prompt injection rendering
- candidate artifact storage
- promotion from candidate to active

It must work even if no subagent is active.

### 4.4 Candidate pipeline is supervised

Candidate extraction is observational output from a completed run. It does not modify the active skill set for the current turn. Promotion is a separate step.

### 4.5 Persistence stays separated

The first slice uses separate persistence surfaces:

- delegated run records
- active skills
- candidate skills
- skill usage log

Do not collapse these into one table or one shared blob. The OS-AI layer needs auditability more than clever storage.

## 5. Architecture

```text
user turn
   |
   v
+------------------------+
| kernel                 |
| - local vs delegated   |
| - select active skills |
| - build child spec     |
+-----------+------------+
            |
     +------+------+
     |             |
     v             v
+------------------------+   +---------------------------+
| subagent runtime       |   | skill runtime             |
| - spawn                |   | - load active skills      |
| - stream events        |   | - render prompt block     |
| - timeout/cancel       |   | - store candidates        |
| - collect result       |   | - promote reviewed skill  |
+-----------+------------+   +-------------+-------------+
            |                              |
            v
     child run result             filesystem artifacts
            |                      active/ + candidates/
            +-----------+------------------+
                        |
                        v
                 kernel merge + optional
                 candidate extraction
```

The important design rule is directional:

- kernel depends on subagent and skills
- subagent does not depend on skills
- skills do not depend on subagent internals

That keeps the concurrency layer testable and the procedural layer portable.

## 6. Staged Delivery

### 6.1 2.E0 — Deterministic Subagent Runtime

This is the first required ship target.

Capabilities:

- spawn one child run with explicit goal and scoped context
- stream typed child lifecycle events to the parent
- cancel child on parent cancellation or timeout
- hard depth limit
- per-child tool allowlist
- typed final result with status, summary, tool usage, duration, and error reason

Definition of done:

- parent state is not mutated by child execution except through the final typed result
- child failure cannot poison the parent session
- cancellation is recursive and deterministic

### 6.2 2.G0 — Static Skill Runtime

This lands immediately on top of the runtime, but remains conservative.

Capabilities:

- load active skills from disk
- validate frontmatter and content bounds
- select a small relevant subset deterministically
- inject a stable prompt block into parent or child runs
- record usage events for later analysis

Definition of done:

- invalid skill files are rejected cleanly
- skill injection is stable and capped
- active skills do not mutate during a live run

### 6.3 2.E1 / 2.G1-lite — Reviewed Vertical Proof

This is the first learning-loop proof, but still supervised.

Capabilities:

- a delegated child run can produce a candidate `SKILL.md`
- candidate is written to a separate inactive store
- candidate metadata records source run, timestamps, and promotion status
- active skill store is unchanged until explicit promotion

Definition of done:

- one successful delegated task can produce one reviewable candidate artifact
- promotion is explicit
- the next run sees the promoted skill only after promotion succeeds

### 6.4 Deferred work

Explicitly deferred beyond this spec:

- autonomous complexity detection
- autonomous skill improvement
- semantic skill retrieval
- remote or process-isolated executors
- multi-parent subagent graphs
- memory-assisted skill ranking

## 7. Proposed File Boundaries

```text
gormes/internal/
  subagent/
    types.go         # child spec, events, result types
    manager.go       # spawn, stream, collect, timeout, cancellation
    registry.go      # process-wide live child tracking
    policy.go        # depth limit, allowlist rules, validation

  skills/
    document.go      # parse + validate SKILL.md
    store.go         # filesystem active/candidate store
    selector.go      # deterministic selection
    prompt.go        # render injected skills block
    candidate.go     # candidate extraction + metadata writes
    usage.go         # append-only usage log

  kernel/
    delegate.go      # local vs delegated path, merge policy
    delegate_tool.go # explicit delegate tool surface if needed
    prompt.go        # skill block insertion hook

  config/
    config.go        # [delegation] and [skills] config

gormes/cmd/gormes/
  telegram.go        # wires runtime + skills into current messaging entrypoint
```

The exact filenames can move slightly during planning, but the package boundaries should hold. In particular:

- `internal/subagent` must not parse `SKILL.md`
- `internal/skills` must not own child goroutines
- `internal/kernel` must not absorb persistence details from either subsystem

## 8. Execution Flow

The approved first-slice execution flow is:

1. Parent turn enters the kernel.
2. Kernel decides local vs delegated execution.
3. Kernel normalizes a child run spec:
   - goal
   - scoped context
   - max iterations
   - timeout
   - allowed tools
   - depth
   - selected active skills
4. Subagent runtime starts the child with its own context and event stream.
5. Active skills are injected at child start only.
6. Parent observes streamed child events.
7. Child result returns as a typed object.
8. Kernel merges the child result back into the parent turn.
9. If delegation policy allows it and the run succeeded, the system may draft a candidate skill artifact.
10. Candidate remains inactive until explicit promotion.

Hard constraints:

- parent cancellation recursively stops children
- child failure never mutates live parent state
- no hidden delegation chains
- no skill mutation mid-run
- no shared mutable transcript between parent and child

## 9. Storage Model

### 9.1 Filesystem-first rule

The first slice uses the filesystem as the source of truth for skills because:

- operators can inspect artifacts directly
- review/promotion is easy to reason about
- TDD stays simpler than a hybrid SQLite + filesystem store
- the later move to indexed metadata remains possible without invalidating the artifact format

### 9.2 Active skills

Path shape:

```text
~/.local/share/gormes/skills/active/<slug>/SKILL.md
~/.local/share/gormes/skills/active/<slug>/meta.json
```

`meta.json` records:

- slug
- title
- version
- created_at
- promoted_at
- source_candidate_id
- status = `active`

### 9.3 Candidate skills

Path shape:

```text
~/.local/share/gormes/skills/candidates/<candidate-id>/SKILL.md
~/.local/share/gormes/skills/candidates/<candidate-id>/meta.json
```

`candidate-id` should be deterministic enough for debugging, for example:

```text
<utc-timestamp>-<source-run-id>-<slug>
```

Candidate metadata records:

- candidate_id
- slug
- source_run_id
- parent_session_id
- child_agent_id
- created_at
- promoted_at or null
- status = `candidate` | `promoted` | `rejected`

### 9.4 Usage and run records

Append-only logs:

```text
~/.local/share/gormes/skills/usage.jsonl
~/.local/share/gormes/subagents/runs.jsonl
```

Why JSONL:

- auditable
- easy to tail
- easy to test
- no schema migration pressure in the first slice

Structured querying can come later if the logs become too large.

## 10. Skill Selection And Promotion Rules

### 10.1 Selection

The first slice must use deterministic selection, not a learned ranker.

Selection order:

1. explicitly requested skill names
2. exact tag/keyword hits against the goal/context
3. prefix/substring matches on skill name or short description

Limits:

- max 3 injected skills per run
- max total rendered skill bytes
- stable order by explicitness, then lexical score, then slug

If selection overflows the cap, lower-ranked skills are dropped. No summarization step is allowed in the first slice.

### 10.2 Candidate generation

Candidate generation may run only when all of the following are true:

- child status is `completed`
- delegation policy enabled candidate capture for that run
- child emitted at least one tool event, unless an explicit operator override allows a no-tool draft
- the kernel captured the child summary plus the event log needed to draft a candidate artifact

If candidate generation fails, the child run still counts as successful. Candidate drafting is a sidecar, not part of the critical path.

### 10.3 Promotion

Promotion is explicit and supervised:

- copy or move candidate into the active store
- write active metadata
- mark candidate as promoted
- make the promoted skill visible only to future runs

The promotion surface may be CLI, API, or a later Telegram/admin surface. For this spec, the critical requirement is the service boundary and data model, not the operator UI.

## 11. Failure Model

### 11.1 Subagent runtime failures

| Condition | Required behavior |
|---|---|
| child timeout | child ends with `timeout`; parent receives typed failure result |
| parent cancellation | all children are cancelled before the parent turn exits |
| blocked tool request | child gets a deterministic policy error |
| depth overflow | spawn is rejected before child start |
| child panic | recovered, surfaced as typed child failure |

### 11.2 Skill runtime failures

| Condition | Required behavior |
|---|---|
| invalid `SKILL.md` | reject file, exclude from selection, log reason |
| malformed metadata | exclude file, log reason, do not crash turn |
| too many selected skills | enforce cap deterministically |
| candidate draft failure | log it, keep parent/child run successful |
| promotion conflict | reject promotion cleanly; active store remains unchanged |

The system must fail closed:

- no invalid skill silently injected
- no child silently promoted to active knowledge
- no partial promotion visible to the next run

## 12. TDD Strategy

This slice must follow strict TDD:

1. write failing tests
2. verify the failure is for the intended reason
3. write minimal implementation
4. re-run targeted tests
5. re-run broader package tests
6. refactor only after green

No production code for subagents or skills should be written before its failing test exists.

### 12.1 Unit-test groups

**Subagent runtime**

- spawn success
- depth rejection
- timeout handling
- recursive cancellation
- event ordering
- tool allowlist enforcement

**Skill runtime**

- `SKILL.md` parse + validation
- store load/save
- deterministic selection
- prompt render capping
- usage logging

**Candidate pipeline**

- success path writes candidate artifact
- failed child does not write candidate
- candidate stays inactive before promotion
- promotion makes skill visible only to future runs

### 12.2 Integration-test groups

**Kernel + subagent**

- delegated run streams events and returns typed result
- parent state survives child failure
- cancellation propagates correctly

**Kernel + skills**

- selected skills appear in parent or child prompt block
- invalid skills never reach prompt assembly

**Vertical proof**

- parent delegates one real task
- child completes with bounded tools
- candidate artifact is written
- candidate is invisible before promotion
- promoted skill is visible on the next run

## 13. Acceptance Criteria

This spec is satisfied when all of the following are true:

1. Gormes can spawn a bounded child run with deterministic cancellation and timeout semantics.
2. Child runs stream observable lifecycle events back to the parent.
3. Parent and child state remain isolated except for the typed result contract.
4. Gormes can load active `SKILL.md` artifacts from disk and inject a capped subset deterministically.
5. A successful delegated run can produce a candidate `SKILL.md` artifact without auto-activating it.
6. Promotion is explicit, reviewable, and only affects future runs.
7. The first vertical slice is green under TDD with unit and integration coverage proving the OS-AI spine, not just isolated helpers.

## 14. Implementation Order Recommendation

Write the plan and execute in this order:

1. `internal/subagent` types + manager + policy
2. kernel delegation path + typed child result merge
3. `internal/skills` document parser + filesystem store
4. deterministic selector + prompt injection
5. candidate artifact writer
6. one vertical proof test spanning delegation -> candidate -> promotion visibility

This keeps concurrency risk first, prompt risk second, and learning-loop risk last.

## 15. Final Decision

Phase 2 should not ship “skills first” or “subagents first” as isolated features. The first correct OS-AI slice is:

- **runtime first**
- **skills second**
- **reviewed learning proof third**

That gives Gormes a real operating-system backbone instead of a looser collection of agent tricks.
