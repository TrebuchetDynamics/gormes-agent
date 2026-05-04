# Architecture Planner Loop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Go `architecture-planner-loop` command that improves the building-gormes architecture plan by running a Codex/Claude planner over local Hermes, GBrain, Goncho, and Gormes documentation context.

**Architecture:** Create a small `internal/architectureplanner` package for config, context collection, prompt construction, backend execution, artifacts, and validation orchestration. Add `cmd/architecture-planner-loop` as a thin CLI layer similar to `cmd/autoloop`.

**Tech Stack:** Go standard library, existing `internal/autoloop.Runner` command abstraction, existing `internal/progress` loader, `go test`.

---

### Task 1: Planner Package Behavior

**Files:**
- Create: `internal/architectureplanner/config.go`
- Create: `internal/architectureplanner/context.go`
- Create: `internal/architectureplanner/prompt.go`
- Create: `internal/architectureplanner/run.go`
- Test: `internal/architectureplanner/config_test.go`
- Test: `internal/architectureplanner/run_test.go`

- [ ] **Step 1: Write failing config/context tests**

```go
func TestConfigFromEnvDefaultsToArchitecturePlannerPaths(t *testing.T) {
	root := filepath.Join("tmp", "repo")
	cfg, err := architectureplanner.ConfigFromEnv(root, map[string]string{})
	if err != nil {
		t.Fatalf("ConfigFromEnv() error = %v", err)
	}
	if cfg.ProgressJSON != filepath.Join(root, "docs", "content", "building-gormes", "architecture_plan", "progress.json") {
		t.Fatalf("ProgressJSON = %q", cfg.ProgressJSON)
	}
	if cfg.RunRoot != filepath.Join(root, ".codex", "architecture-planner") {
		t.Fatalf("RunRoot = %q", cfg.RunRoot)
	}
}
```

- [ ] **Step 2: Run tests to verify RED**

Run: `go test ./internal/architectureplanner -count=1`

Expected: FAIL because `internal/architectureplanner` does not exist.

- [ ] **Step 3: Implement minimal config/context/prompt/run code**

Create config defaults, scan the configured source directories, write `context.json`, `latest_prompt.txt`, `latest_planner_report.md`, and `planner_state.json`, and support `DryRun`.

- [ ] **Step 4: Run package tests to verify GREEN**

Run: `go test ./internal/architectureplanner -count=1`

Expected: PASS.

### Task 2: Command CLI

**Files:**
- Create: `cmd/architecture-planner-loop/main.go`
- Test: `cmd/architecture-planner-loop/main_test.go`

- [ ] **Step 1: Write failing command tests**

```go
func TestRunDryRunPrintsPlannerSummary(t *testing.T) {
	t.Setenv("RUN_ROOT", filepath.Join(repoRoot, "planner"))
	err := run([]string{"run", "--dry-run"})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "architecture planner dry-run") {
		t.Fatalf("stdout missing dry-run summary")
	}
}
```

- [ ] **Step 2: Run tests to verify RED**

Run: `go test ./cmd/architecture-planner-loop -count=1`

Expected: FAIL because the command does not exist.

- [ ] **Step 3: Implement CLI dispatch**

Support `run`, `status`, `show-report`, `doctor`, `--dry-run`, `--codexu`, `--claudeu`, `--mode`, and `--help`.

- [ ] **Step 4: Run command tests to verify GREEN**

Run: `go test ./cmd/architecture-planner-loop -count=1`

Expected: PASS.

### Task 3: Command Documentation

**Files:**
- Create: `cmd/architecture-planner-loop/README.md`

- [ ] **Step 1: Document command contract**

Include source roots, output artifacts, backend flags, and validation behavior.

- [ ] **Step 2: Validate docs and command**

Run:

```bash
go test ./cmd/architecture-planner-loop ./internal/architectureplanner -count=1
go run ./cmd/architecture-planner-loop run --dry-run
```

Expected: tests pass and dry-run prints the selected backend, source roots, progress file, and run root.
