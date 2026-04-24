# Gormes Phase 2.E1 / 2.G1-lite Candidate Promotion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the reviewed vertical proof where a successful delegated run can draft an inactive candidate skill artifact and an explicit promotion step can make it visible to future runs.

**Architecture:** Keep the implementation filesystem-first and deterministic. `internal/skills` owns candidate artifact writing and promotion, while `delegate_task` only calls a narrow drafting seam after a successful child result. Promotion remains explicit and separate from drafting so no child run can silently mutate the active skill set.

**Tech Stack:** Go stdlib (`encoding/json`, `os`, `path/filepath`, `strings`, `time`), existing `internal/skills`, existing `internal/subagent`, existing `cmd/gormes`, progress/docs generator.

---

## File Map

- Create: `internal/skills/candidate.go`
- Create: `internal/skills/candidate_test.go`
- Modify: `internal/skills/store.go`
- Modify: `internal/subagent/delegate_tool.go`
- Modify: `internal/subagent/delegate_tool_test.go`
- Modify: `gormes/cmd/gormes/registry.go`
- Modify: `gormes/cmd/gormes/registry_test.go`
- Modify: `docs/content/building-gormes/architecture_plan/progress.json`
- Modify: `docs/content/building-gormes/architecture_plan/phase-2-gateway.md`
- Modify: `docs/content/building-gormes/architecture_plan/why-go.md`

## Task 1: Candidate store + promotion API

- [ ] **Step 1: Write failing tests**
  Add tests proving:
  - a candidate draft writes `candidates/<id>/SKILL.md` and `meta.json`
  - the active snapshot ignores drafted candidates before promotion
  - `PromoteCandidate()` copies the candidate into `active/<slug>/`, updates candidate metadata to `promoted`, and makes the skill visible on the next snapshot

- [ ] **Step 2: Run targeted tests and verify RED**
  Run:
  `go test ./internal/skills -run 'TestCandidate|TestSkillStore' -count=1 -v`

- [ ] **Step 3: Implement the minimal candidate/promotion store**
  Add a deterministic candidate ID, JSON metadata writer, and promotion function in `internal/skills`.

- [ ] **Step 4: Run targeted tests and verify GREEN**
  Run:
  `go test ./internal/skills -run 'TestCandidate|TestSkillStore' -count=1 -v`

## Task 2: delegate_task draft hook

- [ ] **Step 1: Write failing tests**
  Add tests proving:
  - `delegate_task` can request candidate drafting with explicit operator intent
  - a successful child result drafts a candidate artifact
  - a child result that is not eligible does not draft a candidate

- [ ] **Step 2: Run targeted tests and verify RED**
  Run:
  `go test ./internal/subagent ./cmd/gormes -run 'Delegate|Registry' -count=1 -v`

- [ ] **Step 3: Implement the minimal drafting seam**
  Extend `delegate_task` with an injected candidate drafter interface and wire it from `buildDefaultRegistry()`.

- [ ] **Step 4: Run targeted tests and verify GREEN**
  Run:
  `go test ./internal/subagent ./cmd/gormes -run 'Delegate|Registry' -count=1 -v`

## Task 3: Docs + progress sync

- [ ] **Step 1: Update the Phase 2 ledger**
  Mark `Candidate drafting + promotion flow` complete in `progress.json`.

- [ ] **Step 2: Regenerate progress-driven markdown**
  Run:
  `go run ./cmd/progress-gen -write`

- [ ] **Step 3: Verify docs/progress**
  Run:
  `go run ./cmd/progress-gen -validate`
  `go test ./internal/progress ./docs -count=1`

## Branch Gate

Run before claiming completion:

```bash
go test ./internal/skills ./internal/subagent ./cmd/gormes ./internal/progress -count=1
go run ./cmd/progress-gen -validate
go test ./docs -count=1
```
