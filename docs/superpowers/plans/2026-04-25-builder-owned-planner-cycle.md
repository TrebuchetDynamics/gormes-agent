# Builder-Owned Planner Cycle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `builder-loop run --loop` so the builder process runs infinite builder cycles and synchronously runs one planner cycle after each completed builder cycle.

**Architecture:** Keep `builderloop.RunOnce` as the unit of builder work. Add a thin, testable CLI-level loop in `cmd/builder-loop` that calls `RunOnce`, then invokes `go run ./cmd/planner-loop run`, then sleeps before the next cycle. Update the installed builder service to execute `run --loop` so systemd restart is crash recovery, not the normal cadence driver.

**Tech Stack:** Go CLI code, existing `cmdrunner.Runner` process seam, existing `builderloop.RunOnce`, systemd unit renderer tests, Markdown docs.

---

## File Structure

- Modify `cmd/builder-loop/main.go`
  - Parse `--loop`.
  - Reject `--loop --dry-run`.
  - Add a small injectable loop runtime for tests.
  - Run planner synchronously after each successful builder cycle.
  - Parse `BUILDER_LOOP_SLEEP`.
- Modify `cmd/builder-loop/main_test.go`
  - Add CLI parse and loop-runtime tests.
  - Update service-install test expectation for `run --loop`.
- Modify `internal/builderloop/service.go`
  - Default rendered service command to `run --loop`.
- Modify `internal/builderloop/service_test.go`
  - Update systemd renderer expectations.
- Modify `cmd/builder-loop/README.md`
  - Document `run --loop` and `BUILDER_LOOP_SLEEP`.
- Modify `docs/content/building-gormes/_index.md`
  - Document steady-state builder-owned planner cadence.
- Modify `AGENTS.md`
  - Update the architecture brief with the builder-owned steady-state cycle.

Existing unrelated dirty files must not be reverted or folded into these commits.

---

### Task 1: Parse `--loop` And Reject `--loop --dry-run`

**Files:**
- Modify: `cmd/builder-loop/main.go`
- Test: `cmd/builder-loop/main_test.go`

- [ ] **Step 1: Write the failing tests**

Add these tests near the existing `TestParseRunOptions_BackendFlag` tests in `cmd/builder-loop/main_test.go`:

```go
func TestParseRunOptions_LoopFlag(t *testing.T) {
	opts, err := parseRunOptions([]string{"--loop"})
	if err != nil {
		t.Fatalf("parseRunOptions(--loop) error = %v", err)
	}
	if !opts.loop {
		t.Fatalf("loop = false, want true")
	}
}

func TestParseRunOptions_RejectsLoopDryRunCombination(t *testing.T) {
	_, err := parseRunOptions([]string{"--loop", "--dry-run"})
	if !errors.Is(err, errParse) {
		t.Fatalf("parseRunOptions(--loop --dry-run) error = %v, want errParse", err)
	}
	if err == nil || !strings.Contains(err.Error(), "--loop cannot be combined with --dry-run") {
		t.Fatalf("error = %v, want loop/dry-run message", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```sh
go test ./cmd/builder-loop -run 'TestParseRunOptions_(LoopFlag|RejectsLoopDryRunCombination)' -count=1
```

Expected: FAIL because `runOptions` has no `loop` field and `--loop` is rejected as an unknown flag.

- [ ] **Step 3: Implement the minimal parser change**

In `cmd/builder-loop/main.go`, update usage strings:

```go
const usage = "usage: builder-loop [--repo-root <path>] run [--loop] [--dry-run] [--backend codexu|claudeu|opencode] | progress validate | progress write | repo benchmark record | repo readme update | audit | digest [--output <path>] [--force] | doctor | service install | service install-audit | service disable legacy-timers"
```

Update the run subcommand usage:

```go
"run": "usage: builder-loop run [--loop] [--dry-run] [--backend codexu|claudeu|opencode]",
```

Update `runOptions`:

```go
type runOptions struct {
	dryRun  bool
	loop    bool
	backend string
}
```

Update `parseRunOptions`:

```go
func parseRunOptions(args []string) (runOptions, error) {
	opts := runOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--dry-run":
			opts.dryRun = true
		case "--loop":
			opts.loop = true
		case "--backend":
			if i+1 >= len(args) {
				return runOptions{}, fmt.Errorf("%w: --backend requires a value\n%s", errParse, subUsage["run"])
			}
			i++
			if !contains(supportedBuilderBackends, args[i]) {
				return runOptions{}, fmt.Errorf("%w: unsupported backend %q (want one of %s)\n%s",
					errParse, args[i], strings.Join(supportedBuilderBackends, ", "), subUsage["run"])
			}
			opts.backend = args[i]
		default:
			return runOptions{}, fmt.Errorf("%w\n%s", errParse, subUsage["run"])
		}
	}
	if opts.loop && opts.dryRun {
		return runOptions{}, fmt.Errorf("%w: --loop cannot be combined with --dry-run\n%s", errParse, subUsage["run"])
	}

	return opts, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```sh
go test ./cmd/builder-loop -run 'TestParseRunOptions' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```sh
git add cmd/builder-loop/main.go cmd/builder-loop/main_test.go
git commit -m "feat: parse builder loop mode flag"
```

---

### Task 2: Add A Testable Builder-Owned Loop Runtime

**Files:**
- Modify: `cmd/builder-loop/main.go`
- Test: `cmd/builder-loop/main_test.go`

- [ ] **Step 1: Write the failing tests**

Add this helper and tests in `cmd/builder-loop/main_test.go` near the run-command tests:

```go
type fakeAutoloopRuntime struct {
	builderCalls int
	plannerCalls int
	sleepCalls   int
	events       []string
	builderErr   error
	plannerErr   error
	cancelAfterSleep context.CancelFunc
}

func (f *fakeAutoloopRuntime) runtime() autoloopRuntime {
	return autoloopRuntime{
		runBuilder: func(_ context.Context, _ builderloop.Config, dryRun bool) (builderloop.RunSummary, error) {
			f.builderCalls++
			f.events = append(f.events, fmt.Sprintf("builder:%v", dryRun))
			if f.builderErr != nil {
				return builderloop.RunSummary{}, f.builderErr
			}
			return builderloop.RunSummary{
				Candidates: 1,
				Selected: []builderloop.Candidate{{PhaseID: "1", SubphaseID: "1.A", ItemName: "loop candidate", Status: "planned"}},
			}, nil
		},
		runPlanner: func(_ context.Context) error {
			f.plannerCalls++
			f.events = append(f.events, "planner")
			return f.plannerErr
		},
		sleep: func(ctx context.Context, d time.Duration) error {
			f.sleepCalls++
			f.events = append(f.events, "sleep:"+d.String())
			if f.cancelAfterSleep != nil {
				f.cancelAfterSleep()
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		},
	}
}

func TestRunAutoloopLoopRunsPlannerAfterBuilder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fake := &fakeAutoloopRuntime{cancelAfterSleep: cancel}
	var stdout bytes.Buffer
	deps := defaultDeps()
	deps.stdout = &stdout

	err := runAutoloopWithRuntime(ctx, deps, builderloop.Config{}, runOptions{loop: true}, time.Second, fake.runtime())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runAutoloopWithRuntime() error = %v, want context.Canceled", err)
	}
	wantEvents := []string{"builder:false", "planner", "sleep:1s"}
	if !reflect.DeepEqual(fake.events, wantEvents) {
		t.Fatalf("events = %#v, want %#v", fake.events, wantEvents)
	}
	if fake.builderCalls != 1 || fake.plannerCalls != 1 || fake.sleepCalls != 1 {
		t.Fatalf("calls builder=%d planner=%d sleep=%d, want 1/1/1", fake.builderCalls, fake.plannerCalls, fake.sleepCalls)
	}
	if !strings.Contains(stdout.String(), "loop candidate") {
		t.Fatalf("stdout = %q, want builder summary", stdout.String())
	}
}

func TestRunAutoloopLoopStopsOnPlannerFailure(t *testing.T) {
	wantErr := errors.New("planner failed")
	fake := &fakeAutoloopRuntime{plannerErr: wantErr}
	deps := defaultDeps()

	err := runAutoloopWithRuntime(context.Background(), deps, builderloop.Config{}, runOptions{loop: true}, time.Second, fake.runtime())
	if !errors.Is(err, wantErr) {
		t.Fatalf("runAutoloopWithRuntime() error = %v, want %v", err, wantErr)
	}
	wantEvents := []string{"builder:false", "planner"}
	if !reflect.DeepEqual(fake.events, wantEvents) {
		t.Fatalf("events = %#v, want %#v", fake.events, wantEvents)
	}
	if fake.sleepCalls != 0 {
		t.Fatalf("sleepCalls = %d, want 0 after planner failure", fake.sleepCalls)
	}
}

func TestRunAutoloopOneShotDoesNotRunPlanner(t *testing.T) {
	fake := &fakeAutoloopRuntime{}
	var stdout bytes.Buffer
	deps := defaultDeps()
	deps.stdout = &stdout

	if err := runAutoloopWithRuntime(context.Background(), deps, builderloop.Config{}, runOptions{}, time.Second, fake.runtime()); err != nil {
		t.Fatalf("runAutoloopWithRuntime() error = %v", err)
	}
	wantEvents := []string{"builder:false"}
	if !reflect.DeepEqual(fake.events, wantEvents) {
		t.Fatalf("events = %#v, want %#v", fake.events, wantEvents)
	}
	if fake.plannerCalls != 0 {
		t.Fatalf("plannerCalls = %d, want 0 for one-shot", fake.plannerCalls)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```sh
go test ./cmd/builder-loop -run 'TestRunAutoloop(LoopRunsPlannerAfterBuilder|LoopStopsOnPlannerFailure|OneShotDoesNotRunPlanner)' -count=1
```

Expected: FAIL because `autoloopRuntime` and `runAutoloopWithRuntime` do not exist.

- [ ] **Step 3: Implement the loop runtime**

In `cmd/builder-loop/main.go`, change the `run` dispatch from:

```go
return runAutoloop(ctx, deps, cfg, runOpts.dryRun)
```

to:

```go
return runAutoloop(ctx, deps, root, cfg, runOpts)
```

Replace `runAutoloop` with these helpers:

```go
type autoloopRuntime struct {
	runBuilder func(context.Context, builderloop.Config, bool) (builderloop.RunSummary, error)
	runPlanner func(context.Context) error
	sleep      func(context.Context, time.Duration) error
}

func defaultAutoloopRuntime(deps cliDeps, root string) autoloopRuntime {
	runner := deps.runner
	if runner == nil {
		runner = cmdrunner.ExecRunner{}
	}
	return autoloopRuntime{
		runBuilder: func(ctx context.Context, cfg builderloop.Config, dryRun bool) (builderloop.RunSummary, error) {
			return builderloop.RunOnce(ctx, builderloop.RunOptions{
				Config: cfg,
				Runner: runner,
				DryRun: dryRun,
			})
		},
		runPlanner: func(ctx context.Context) error {
			result := runner.Run(ctx, cmdrunner.Command{
				Name: "go",
				Args: []string{"run", "./cmd/planner-loop", "run"},
				Dir:  root,
			})
			if result.Err != nil {
				detail := strings.TrimSpace(result.Stderr)
				if detail != "" {
					return fmt.Errorf("planner command go run ./cmd/planner-loop run failed: %w: %s", result.Err, detail)
				}
				return fmt.Errorf("planner command go run ./cmd/planner-loop run failed: %w", result.Err)
			}
			return nil
		},
		sleep: sleepContext,
	}
}

func runAutoloop(ctx context.Context, deps cliDeps, root string, cfg builderloop.Config, opts runOptions) error {
	interval := time.Duration(0)
	if opts.loop {
		interval = 30 * time.Second
	}
	return runAutoloopWithRuntime(ctx, deps, cfg, opts, interval, defaultAutoloopRuntime(deps, root))
}

func runAutoloopWithRuntime(ctx context.Context, deps cliDeps, cfg builderloop.Config, opts runOptions, interval time.Duration, runtime autoloopRuntime) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		summary, err := runtime.runBuilder(ctx, cfg, opts.dryRun)
		if err != nil {
			return err
		}
		printAutoloopSummary(deps.stdout, summary)
		if !opts.loop {
			return nil
		}
		if err := runtime.runPlanner(ctx); err != nil {
			return err
		}
		if err := runtime.sleep(ctx, interval); err != nil {
			return err
		}
	}
}

func printAutoloopSummary(w io.Writer, summary builderloop.RunSummary) {
	fmt.Fprintf(w, "candidates: %d\nselected: %d\n", summary.Candidates, len(summary.Selected))
	if summary.MaxPhaseFiltered > 0 {
		fmt.Fprintf(w, "max_phase_filtered: %d\nnext_max_phase: %d\nhint: rerun with MAX_PHASE=%d to include the next queued phase\n",
			summary.MaxPhaseFiltered,
			summary.NextFilteredMaxPhase,
			summary.NextFilteredMaxPhase,
		)
	}
	for _, candidate := range summary.Selected {
		fmt.Fprintf(w, "- %s/%s %s [%s] owner=%s size=%s reason=%s\n",
			candidate.PhaseID,
			candidate.SubphaseID,
			candidate.ItemName,
			candidate.Status,
			dashIfEmpty(candidate.ExecutionOwner),
			dashIfEmpty(candidate.SliceSize),
			dashIfEmpty(candidate.SelectionReason()),
		)
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```sh
go test ./cmd/builder-loop -run 'TestRunAutoloop(LoopRunsPlannerAfterBuilder|LoopStopsOnPlannerFailure|OneShotDoesNotRunPlanner)|TestRunCommandDryRun' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```sh
git add cmd/builder-loop/main.go cmd/builder-loop/main_test.go
git commit -m "feat: run planner after builder loop cycles"
```

---

### Task 3: Parse `BUILDER_LOOP_SLEEP`

**Files:**
- Modify: `cmd/builder-loop/main.go`
- Test: `cmd/builder-loop/main_test.go`

- [ ] **Step 1: Write the failing tests**

Add these tests in `cmd/builder-loop/main_test.go`:

```go
func TestBuilderLoopSleepDefault(t *testing.T) {
	got, err := builderLoopSleep(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("builderLoopSleep(default) error = %v", err)
	}
	if got != 30*time.Second {
		t.Fatalf("builderLoopSleep(default) = %v, want 30s", got)
	}
}

func TestBuilderLoopSleepFromEnv(t *testing.T) {
	got, err := builderLoopSleep(func(key string) (string, bool) {
		if key == "BUILDER_LOOP_SLEEP" {
			return "2m", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("builderLoopSleep(2m) error = %v", err)
	}
	if got != 2*time.Minute {
		t.Fatalf("builderLoopSleep(2m) = %v, want 2m", got)
	}
}

func TestBuilderLoopSleepRejectsInvalid(t *testing.T) {
	_, err := builderLoopSleep(func(key string) (string, bool) {
		if key == "BUILDER_LOOP_SLEEP" {
			return "soon", true
		}
		return "", false
	})
	if !errors.Is(err, errParse) {
		t.Fatalf("builderLoopSleep(invalid) error = %v, want errParse", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```sh
go test ./cmd/builder-loop -run 'TestBuilderLoopSleep' -count=1
```

Expected: FAIL because `builderLoopSleep` does not exist.

- [ ] **Step 3: Implement sleep parsing**

Add this helper to `cmd/builder-loop/main.go` near `sleepContext`:

```go
func builderLoopSleep(lookup func(string) (string, bool)) (time.Duration, error) {
	value, ok := lookup("BUILDER_LOOP_SLEEP")
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return 30 * time.Second, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil || d < 0 {
		return 0, fmt.Errorf("%w: BUILDER_LOOP_SLEEP must be a non-negative Go duration (got %q)", errParse, value)
	}
	return d, nil
}
```

Confirm `runAutoloop` from Task 2 calls this helper only when `opts.loop` is true.
Replace the temporary `interval = 30 * time.Second` assignment from Task 2 with:

```go
var err error
interval, err = builderLoopSleep(os.LookupEnv)
if err != nil {
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```sh
go test ./cmd/builder-loop -run 'TestBuilderLoopSleep|TestRunAutoloopLoopRunsPlannerAfterBuilder' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```sh
git add cmd/builder-loop/main.go cmd/builder-loop/main_test.go
git commit -m "feat: configure builder loop sleep"
```

---

### Task 4: Wire The Installed Builder Service To `run --loop`

**Files:**
- Modify: `internal/builderloop/service.go`
- Modify: `cmd/builder-loop/main.go`
- Test: `internal/builderloop/service_test.go`
- Test: `cmd/builder-loop/main_test.go`

- [ ] **Step 1: Write/update failing tests**

Update `TestRenderServiceUnitInjectsPaths` in `internal/builderloop/service_test.go` so the expected default `ExecStart` is:

```go
"ExecStart=/opt/gormes/bin/autoloop run --loop",
```

Update the quoted-path expectations in the same test:

```go
`ExecStart="/tmp/gormes repo/bin/auto%%loop\"\\bin" run --loop`,
```

Update the control-character expectation:

```go
`ExecStart="/tmp/gormes\nrepo/bin/autoloop" run --loop`,
```

Update `TestInstallServiceWritesUnitAndReloadsSystemd` in `internal/builderloop/service_test.go`:

```go
"ExecStart=/opt/gormes/bin/autoloop run --loop",
```

Update `TestServiceInstallWritesUnitUnderXDGConfigHome` in `cmd/builder-loop/main_test.go`. Replace the old `wantExec` assertion with:

```go
wantExec := "ExecStart=" + filepath.Join(repoRoot, "scripts", "gormes-auto-codexu-orchestrator.sh") + " run --loop"
if !strings.Contains(string(unit), wantExec) {
	t.Fatalf("unit = %q, want stable wrapper exec %q", unit, wantExec)
}
if strings.Contains(string(unit), "go-build") {
	t.Fatalf("unit = %q, want no temporary go-build path", unit)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```sh
go test ./internal/builderloop -run 'TestRenderServiceUnitInjectsPaths|TestInstallServiceWritesUnitAndReloadsSystemd' -count=1
go test ./cmd/builder-loop -run TestServiceInstallWritesUnitUnderXDGConfigHome -count=1
```

Expected: FAIL because service rendering still uses one-shot `run` or no wrapper args.

- [ ] **Step 3: Implement service command changes**

In `internal/builderloop/service.go`, update the default args in `RenderServiceUnit`:

```go
execArgs := opts.ExecArgs
if execArgs == nil {
	execArgs = []string{"run", "--loop"}
}
```

In `cmd/builder-loop/main.go`, update `installService`:

```go
return builderloop.InstallService(ctx, builderloop.ServiceInstallOptions{
	Runner:       deps.runner,
	UnitDir:      unitDir,
	UnitName:     "gormes-orchestrator.service",
	AutoloopPath: orchestratorWrapperPath(root),
	WorkDir:      root,
	ExecArgs:     []string{"run", "--loop"},
	AutoStart:    autoStart(),
	Force:        force,
})
```

Keep `TestRenderServiceUnitAllowsCustomExecArgs` unchanged: explicit `ExecArgs: []string{}` must still mean no args.

- [ ] **Step 4: Run tests to verify they pass**

Run:

```sh
go test ./internal/builderloop -run 'TestRenderServiceUnitInjectsPaths|TestRenderServiceUnitAllowsCustomExecArgs|TestInstallServiceWritesUnitAndReloadsSystemd' -count=1
go test ./cmd/builder-loop -run TestServiceInstallWritesUnitUnderXDGConfigHome -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```sh
git add internal/builderloop/service.go internal/builderloop/service_test.go cmd/builder-loop/main.go cmd/builder-loop/main_test.go
git commit -m "feat: install builder loop service in loop mode"
```

---

### Task 5: Verify Planner Command Wiring

**Files:**
- Modify: `cmd/builder-loop/main_test.go`
- Modify: `cmd/builder-loop/main.go`

- [ ] **Step 1: Write the failing tests**

Add tests for production planner command wiring:

```go
func TestDefaultAutoloopRuntimeRunsPlannerCommandInRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	runner := &cmdrunner.FakeRunner{Results: []cmdrunner.Result{{}}}
	deps := defaultDeps()
	deps.runner = runner

	runtime := defaultAutoloopRuntime(deps, repoRoot)
	if err := runtime.runPlanner(context.Background()); err != nil {
		t.Fatalf("runPlanner() error = %v", err)
	}

	want := []cmdrunner.Command{{
		Name: "go",
		Args: []string{"run", "./cmd/planner-loop", "run"},
		Dir:  repoRoot,
	}}
	if !reflect.DeepEqual(runner.Commands, want) {
		t.Fatalf("commands = %#v, want %#v", runner.Commands, want)
	}
}

func TestDefaultAutoloopRuntimePlannerFailureIncludesCommand(t *testing.T) {
	repoRoot := t.TempDir()
	wantErr := errors.New("exit 1")
	runner := &cmdrunner.FakeRunner{Results: []cmdrunner.Result{{Err: wantErr, Stderr: "planner stderr\n"}}}
	deps := defaultDeps()
	deps.runner = runner

	runtime := defaultAutoloopRuntime(deps, repoRoot)
	err := runtime.runPlanner(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("runPlanner() error = %v, want %v", err, wantErr)
	}
	for _, want := range []string{"go run ./cmd/planner-loop run", "planner stderr"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err, want)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail or expose gaps**

Run:

```sh
go test ./cmd/builder-loop -run 'TestDefaultAutoloopRuntime' -count=1
```

Expected before Task 2 implementation: FAIL because the runtime does not exist. If Task 2 already added the runtime, the first test should pass and the second may fail if planner stderr is not included.

- [ ] **Step 3: Adjust implementation if needed**

Ensure `defaultAutoloopRuntime` uses exactly this command:

```go
cmdrunner.Command{
	Name: "go",
	Args: []string{"run", "./cmd/planner-loop", "run"},
	Dir:  root,
}
```

Ensure failure wrapping includes stderr when non-empty:

```go
detail := strings.TrimSpace(result.Stderr)
if detail != "" {
	return fmt.Errorf("planner command go run ./cmd/planner-loop run failed: %w: %s", result.Err, detail)
}
return fmt.Errorf("planner command go run ./cmd/planner-loop run failed: %w", result.Err)
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```sh
go test ./cmd/builder-loop -run 'TestDefaultAutoloopRuntime|TestRunAutoloopLoop' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```sh
git add cmd/builder-loop/main.go cmd/builder-loop/main_test.go
git commit -m "test: lock planner command wiring"
```

---

### Task 6: Document The New Cadence

**Files:**
- Modify: `AGENTS.md`
- Modify: `cmd/builder-loop/README.md`
- Modify: `docs/content/building-gormes/_index.md`

- [ ] **Step 1: Update `AGENTS.md`**

In the "Planner-Builder Loop" section after the diagram, add:

```markdown
In steady-state service mode, `cmd/builder-loop run --loop` owns the cadence:
each successful builder cycle releases the shared control-plane lock, runs one
synchronous `cmd/planner-loop run`, then starts the next builder cycle from the
planner-refreshed `progress.json`. Independent planner timer/path triggers may
still fire, but the shared `run.lock` prevents concurrent control-plane writes.
```

- [ ] **Step 2: Update `cmd/builder-loop/README.md`**

Under "Run Modes", add:

```markdown
Run continuously, scheduling one planner refresh after each completed builder
cycle:

```sh
go run ./cmd/builder-loop run --loop
```
```

Under useful environment variables, add:

```markdown
- `BUILDER_LOOP_SLEEP`: delay between a successful planner refresh and the next
  builder cycle in `run --loop` mode. Defaults to `30s`.
```

Near the "Control Plane" lock paragraph, add:

```markdown
In `run --loop` mode, the builder waits for its cycle to finish and release the
shared lock before invoking `go run ./cmd/planner-loop run`. The next builder
cycle starts only after that planner run exits successfully.
```

- [ ] **Step 3: Update `docs/content/building-gormes/_index.md`**

In "Builder-loop execution contract", after the post-promotion gate paragraph, add:

```markdown
For long-running operation, `cmd/builder-loop run --loop` is the steady-state
cadence owner: one builder cycle completes, run health and promotions are
recorded, the shared planner lock is released, one synchronous
`cmd/planner-loop run` refreshes the control plane, and then the next builder
cycle starts from the updated `progress.json`.
```

- [ ] **Step 4: Run docs-oriented checks**

Run:

```sh
go test ./cmd/builder-loop -run 'TestRunCommandHelpPrintsUsage|TestSubcommandHelpPrintsScopedUsage' -count=1
go run ./cmd/builder-loop progress validate
```

Expected: PASS. Progress validation should not require generated content changes for these prose edits.

- [ ] **Step 5: Commit**

```sh
git add AGENTS.md cmd/builder-loop/README.md docs/content/building-gormes/_index.md
git commit -m "docs: document builder-owned planner cadence"
```

---

### Task 7: Full Verification

**Files:**
- No new code unless verification exposes a defect.

- [ ] **Step 1: Run focused command tests**

```sh
go test ./cmd/builder-loop -count=1
```

Expected: PASS.

- [ ] **Step 2: Run focused service tests**

```sh
go test ./internal/builderloop -run 'TestRenderServiceUnit|TestInstallServiceWritesUnitAndReloadsSystemd' -count=1
```

Expected: PASS.

- [ ] **Step 3: Run planner/builder integration package tests**

```sh
go test ./cmd/builder-loop ./internal/builderloop ./cmd/planner-loop ./internal/plannerloop -count=1
```

Expected: PASS.

- [ ] **Step 4: Validate progress control plane**

```sh
go run ./cmd/builder-loop progress validate
```

Expected: PASS.

- [ ] **Step 5: Inspect final diff**

```sh
git diff --stat HEAD
git diff -- cmd/builder-loop/main.go cmd/builder-loop/main_test.go internal/builderloop/service.go internal/builderloop/service_test.go AGENTS.md cmd/builder-loop/README.md docs/content/building-gormes/_index.md
```

Expected: only the planned loop, service, and docs changes appear. Existing unrelated dirty files remain separate.

- [ ] **Step 6: Final commit if any verification fixes were needed**

If verification required small fixes, commit them:

```sh
git add cmd/builder-loop/main.go cmd/builder-loop/main_test.go internal/builderloop/service.go internal/builderloop/service_test.go AGENTS.md cmd/builder-loop/README.md docs/content/building-gormes/_index.md
git commit -m "fix: stabilize builder-owned planner loop"
```

Skip this commit if no verification fixes were needed after earlier task commits.
