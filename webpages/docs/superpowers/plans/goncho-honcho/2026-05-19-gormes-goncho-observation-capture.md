# Gormes Goncho Observation Capture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Wire Gormes runtime turns and tool calls into Goncho's append-only observation/audit lane.

**Architecture:** Keep the kernel independent from the external Goncho package by adding a small kernel observation value and method on `GonchoStore`. The Gormes gateway adapter maps that value to `goncho.ObservationParams`, and `sqlOpenGoncho` applies Goncho observation migrations before the first write.

**Tech Stack:** Go, `database/sql`, SQLite, `github.com/TrebuchetDynamics/goncho`.

---

### Task 1: Pin Migration And Adapter Contracts

**Files:**
- Modify: `cmd/gormes/sqlopen_goncho_test.go`
- Create: `cmd/gormes/goncho_observation_adapter_test.go`
- Modify: `cmd/gormes/gateway.go`
- Create: `cmd/gormes/goncho_gateway_adapter.go`

- [x] Write a failing test proving `sqlOpenGoncho` creates `goncho_observations` and `goncho_audit_events`.
- [x] Write a failing test proving `gonchoAdapter.Observe` creates a scoped observation plus matching audit row.
- [x] Run only those tests and confirm they fail because the new Goncho symbols/schema are not wired yet.
- [x] Apply `goncho.RunMigrations(db)` inside `sqlOpenGonchoRaw`, then restore Gormes' `busy_timeout >= 5000` invariant.
- [x] Implement `gonchoAdapter.Observe` as the mapper from kernel observations to Goncho observation params.
- [x] Run the focused tests and confirm they pass.

### Task 2: Pin Kernel Event Mapping

**Files:**
- Create: `internal/kernel/goncho_observation_test.go`
- Modify: `internal/kernel/goncho_integration.go`
- Modify: `internal/kernel/kernel.go`
- Modify: `internal/kernel/toolexec.go`

- [x] Write a failing kernel test proving one turn with a tool call emits `user_prompt`, `tool_call`, `tool_result`, and `assistant_response` observations with session/context scope.
- [x] Add a small kernel-level `GonchoObservation` type so the kernel does not import the external Goncho package.
- [x] Emit prompt and assistant observations around the existing Goncho turn persistence calls.
- [x] Emit tool call and tool result/error observations from the tool execution path.
- [x] Fix assistant turn persistence to pass the committed `finalContent`, not the cleared draft buffer.
- [x] Run the focused kernel test and existing Goncho turn tests.

### Task 3: Local Module Link And Verification

**Files:**
- Modify: `go.mod`

- [x] Link `github.com/TrebuchetDynamics/goncho` to the adjacent local `../goncho` module for this extraction workspace, because the required observation API is not in the pinned pseudo-version yet.
- [x] Run focused Gormes tests for the adapter, SQL opener, and kernel Goncho paths.
- [x] Run focused Goncho tests for the observation API if any Gormes failures point back into the library.
