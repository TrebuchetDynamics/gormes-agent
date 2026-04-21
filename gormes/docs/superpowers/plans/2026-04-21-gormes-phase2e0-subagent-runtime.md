# Gormes Phase 2.E0 — Subagent Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the first independently-testable OS-AI execution substrate: a bounded Go-native `delegate_task` tool that runs isolated child chat loops through the existing `hermes.Client`, with deterministic cancellation, timeout, depth-limit enforcement, and append-only run logging.

**Architecture:** This plan intentionally covers only `2.E0` from the approved Phase 2 OS-AI spine spec. The first slice does not touch active skills or candidate promotion yet. A `subagent.Manager` owns child lifecycle; a `subagent.ChatRunner` drives a one-off child conversation using the existing Hermes streaming client and a filtered `tools.Registry`; the parent kernel sees the child only as a normal Go-native tool call result. This keeps the runtime seam useful without requiring invasive kernel rewrites.

**Tech Stack:** Go 1.26 stdlib (`context`, `crypto/rand`, `encoding/hex`, `encoding/json`, `errors`, `fmt`, `log/slog`, `os`, `path/filepath`, `strings`, `sync`, `time`); existing `internal/hermes`, `internal/tools`, `internal/config`; existing `cmd/gormes/telegram.go`; existing `tools.MockTool` and `hermes.MockClient` test harnesses.

**Spec:** [`../specs/2026-04-21-gormes-phase2-os-ai-spine-design.md`](../specs/2026-04-21-gormes-phase2-os-ai-spine-design.md)

**Scope note:** The approved spec spans three subsystems: `2.E0` runtime, `2.G0` static skills, and `2.E1/2.G1-lite` reviewed candidate promotion. This plan deliberately implements only `2.E0` so it can ship green and be tested independently. Static skills and candidate promotion should get follow-on plans only after this branch is green.

---

## File Structure

```text
gormes/
  internal/
    config/
      config.go                 # add [delegation] config + XDG run-log path helper
      config_test.go            # defaults + path helper tests
    subagent/
      types.go                  # Spec, Event, Result, status/event enums
      types_test.go
      policy.go                 # validation + blocked-tool policy
      policy_test.go
      runner.go                 # Runner interface + ChatRunner implementation
      runner_test.go
      manager.go                # Handle, Manager, Start/Cancel/Wait
      manager_test.go
      log.go                    # append-only JSONL run logging
      log_test.go
      delegate_tool.go          # Go-native tools.Tool wrapper
      delegate_tool_test.go
      integration_test.go       # end-to-end delegate_task -> child tool -> final summary
  cmd/gormes/
    delegation.go               # registerDelegation helper for telegram wiring
    delegation_test.go          # wiring test for enabled/disabled registration
    telegram.go                 # call registerDelegation()
```

The runtime is intentionally packaged so the child loop is reusable later by the static-skills plan:

- `internal/subagent` owns lifecycle and child execution.
- `internal/config` owns XDG path defaults.
- `cmd/gormes` owns wiring.

No changes are planned under `gormes/internal/memory/` in this runtime-only slice. That avoids the current dirty files in that area.

---

## Task 1: Add Delegation Config + XDG Run-Log Path

**Files:**
- Modify: `gormes/internal/config/config.go`
- Modify: `gormes/internal/config/config_test.go`

- [ ] **Step 1: Write the failing tests**

Append these tests to `gormes/internal/config/config_test.go`:

```go
func TestLoad_DelegationDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Delegation.Enabled {
		t.Errorf("Delegation.Enabled default = true, want false")
	}
	if cfg.Delegation.DefaultMaxIterations != 8 {
		t.Errorf("DefaultMaxIterations = %d, want 8", cfg.Delegation.DefaultMaxIterations)
	}
	if cfg.Delegation.DefaultTimeout != 45*time.Second {
		t.Errorf("DefaultTimeout = %v, want 45s", cfg.Delegation.DefaultTimeout)
	}
	if cfg.Delegation.MaxChildDepth != 1 {
		t.Errorf("MaxChildDepth = %d, want 1", cfg.Delegation.MaxChildDepth)
	}
}

func TestDelegationRunLogPath_HonorsOverride(t *testing.T) {
	cfg := Config{Delegation: DelegationCfg{RunLogPath: "/tmp/custom-runs.jsonl"}}
	if got := cfg.DelegationRunLogPath(); got != "/tmp/custom-runs.jsonl" {
		t.Errorf("DelegationRunLogPath() = %q, want /tmp/custom-runs.jsonl", got)
	}
}

func TestDelegationRunLogPath_DefaultsToXDG(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/gormes-xdg")
	cfg := Config{}
	want := "/tmp/gormes-xdg/gormes/subagents/runs.jsonl"
	if got := cfg.DelegationRunLogPath(); got != want {
		t.Errorf("DelegationRunLogPath() = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd gormes && go test ./internal/config -run "TestLoad_DelegationDefaults|TestDelegationRunLogPath_" -count=1 -v`

Expected: FAIL with `cfg.Delegation undefined`, `undefined: DelegationCfg`, and `cfg.DelegationRunLogPath undefined`.

- [ ] **Step 3: Write the minimal implementation**

In `gormes/internal/config/config.go`, add this struct near `CronCfg`:

```go
type DelegationCfg struct {
	Enabled              bool          `toml:"enabled"`
	DefaultMaxIterations int           `toml:"default_max_iterations"`
	DefaultTimeout       time.Duration `toml:"default_timeout"`
	MaxChildDepth        int           `toml:"max_child_depth"`
	RunLogPath           string        `toml:"run_log_path"`
}
```

Add it to `Config`:

```go
type Config struct {
	ConfigVersion int `toml:"_config_version"`

	Hermes     HermesCfg     `toml:"hermes"`
	TUI        TUICfg        `toml:"tui"`
	Input      InputCfg      `toml:"input"`
	Telegram   TelegramCfg   `toml:"telegram"`
	Cron       CronCfg       `toml:"cron"`
	Delegation DelegationCfg `toml:"delegation"`
	Resume     string        `toml:"-"`
}
```

Add defaults inside `defaults()`:

```go
		Delegation: DelegationCfg{
			Enabled:              false,
			DefaultMaxIterations: 8,
			DefaultTimeout:       45 * time.Second,
			MaxChildDepth:        1,
			RunLogPath:           "",
		},
```

Add the path helper near `CronMirrorPath()`:

```go
// DelegationRunLogPath returns the JSONL run-log path for subagent runs.
// Explicit override wins; otherwise XDG_DATA_HOME/gormes/subagents/runs.jsonl.
func (c *Config) DelegationRunLogPath() string {
	if c.Delegation.RunLogPath != "" {
		return c.Delegation.RunLogPath
	}
	return filepath.Join(xdgDataHome(), "gormes", "subagents", "runs.jsonl")
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd gormes && go test ./internal/config -run "TestLoad_DelegationDefaults|TestDelegationRunLogPath_" -count=1 -v`

Expected: PASS, 3 tests.

- [ ] **Step 5: Commit**

```bash
git add gormes/internal/config/config.go gormes/internal/config/config_test.go
git commit -m "feat(config): add delegation defaults and run-log path"
```

---

## Task 2: Create Subagent Types + Policy

**Files:**
- Create: `gormes/internal/subagent/types.go`
- Create: `gormes/internal/subagent/types_test.go`
- Create: `gormes/internal/subagent/policy.go`
- Create: `gormes/internal/subagent/policy_test.go`

- [ ] **Step 1: Write the failing tests**

Create `gormes/internal/subagent/types_test.go`:

```go
package subagent

import "testing"

func TestEventTypeStrings(t *testing.T) {
	cases := map[EventType]string{
		EventStarted:   "started",
		EventProgress:  "progress",
		EventToolCall:  "tool_call",
		EventCompleted: "completed",
		EventFailed:    "failed",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("EventType = %q, want %q", got, want)
		}
	}
}

func TestResultStatusStrings(t *testing.T) {
	cases := map[ResultStatus]string{
		StatusCompleted: "completed",
		StatusFailed:    "failed",
		StatusCancelled: "cancelled",
		StatusTimedOut:  "timed_out",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("ResultStatus = %q, want %q", got, want)
		}
	}
}
```

Create `gormes/internal/subagent/policy_test.go`:

```go
package subagent

import (
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
)

func TestValidateSpec_RejectsEmptyGoal(t *testing.T) {
	err := ValidateSpec(Spec{}, config.DelegationCfg{MaxChildDepth: 1})
	if err == nil {
		t.Fatal("ValidateSpec: want error for empty goal")
	}
}

func TestValidateSpec_DefaultsIterationsAndTimeout(t *testing.T) {
	spec, err := ApplyDefaults(Spec{Goal: "audit this"}, config.DelegationCfg{
		DefaultMaxIterations: 8,
		DefaultTimeout:       45 * time.Second,
		MaxChildDepth:        1,
	})
	if err != nil {
		t.Fatalf("ApplyDefaults: %v", err)
	}
	if spec.MaxIterations != 8 {
		t.Errorf("MaxIterations = %d, want 8", spec.MaxIterations)
	}
	if spec.Timeout != 45*time.Second {
		t.Errorf("Timeout = %v, want 45s", spec.Timeout)
	}
}

func TestValidateSpec_RejectsDepthAboveLimit(t *testing.T) {
	_, err := ApplyDefaults(Spec{Goal: "x", Depth: 2}, config.DelegationCfg{MaxChildDepth: 1})
	if err == nil {
		t.Fatal("ApplyDefaults: want depth-limit error")
	}
}

func TestBlockedToolSet_ContainsDelegateTask(t *testing.T) {
	if !IsBlockedTool("delegate_task") {
		t.Fatal("delegate_task must be blocked to prevent recursion")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd gormes && go test ./internal/subagent -run "TestEventTypeStrings|TestResultStatusStrings|TestValidateSpec_|TestBlockedToolSet_" -count=1 -v`

Expected: FAIL with `package .../internal/subagent: no Go files`.

- [ ] **Step 3: Write the minimal implementation**

Create `gormes/internal/subagent/types.go`:

```go
package subagent

import "time"

type Spec struct {
	Goal          string
	Context       string
	Model         string
	AllowedTools  []string
	MaxIterations int
	Timeout       time.Duration
	Depth         int
}

type EventType string

const (
	EventStarted   EventType = "started"
	EventProgress  EventType = "progress"
	EventToolCall  EventType = "tool_call"
	EventCompleted EventType = "completed"
	EventFailed    EventType = "failed"
)

type Event struct {
	Type      EventType `json:"type"`
	Message   string    `json:"message,omitempty"`
	ToolName  string    `json:"tool_name,omitempty"`
	Iteration int       `json:"iteration,omitempty"`
}

type ResultStatus string

const (
	StatusCompleted ResultStatus = "completed"
	StatusFailed    ResultStatus = "failed"
	StatusCancelled ResultStatus = "cancelled"
	StatusTimedOut  ResultStatus = "timed_out"
)

type Result struct {
	RunID        string       `json:"run_id"`
	Status       ResultStatus `json:"status"`
	Summary      string       `json:"summary,omitempty"`
	Error        string       `json:"error,omitempty"`
	FinishReason string       `json:"finish_reason,omitempty"`
	ToolCalls    []string     `json:"tool_calls,omitempty"`
}
```

Create `gormes/internal/subagent/policy.go`:

```go
package subagent

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
)

var blockedTools = map[string]struct{}{
	"delegate_task": {},
}

func IsBlockedTool(name string) bool {
	_, ok := blockedTools[name]
	return ok
}

func ApplyDefaults(spec Spec, cfg config.DelegationCfg) (Spec, error) {
	spec.Goal = strings.TrimSpace(spec.Goal)
	spec.Context = strings.TrimSpace(spec.Context)
	spec.Model = strings.TrimSpace(spec.Model)
	if spec.MaxIterations <= 0 {
		spec.MaxIterations = cfg.DefaultMaxIterations
	}
	if spec.Timeout <= 0 {
		spec.Timeout = cfg.DefaultTimeout
	}
	return spec, ValidateSpec(spec, cfg)
}

func ValidateSpec(spec Spec, cfg config.DelegationCfg) error {
	if spec.Goal == "" {
		return fmt.Errorf("subagent: empty goal")
	}
	if spec.MaxIterations <= 0 {
		return fmt.Errorf("subagent: max_iterations must be > 0")
	}
	if spec.Depth > cfg.MaxChildDepth {
		return fmt.Errorf("subagent: depth %d exceeds max %d", spec.Depth, cfg.MaxChildDepth)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd gormes && go test ./internal/subagent -run "TestEventTypeStrings|TestResultStatusStrings|TestValidateSpec_|TestBlockedToolSet_" -count=1 -v`

Expected: PASS, 6 tests.

- [ ] **Step 5: Commit**

```bash
git add gormes/internal/subagent/types.go gormes/internal/subagent/types_test.go gormes/internal/subagent/policy.go gormes/internal/subagent/policy_test.go
git commit -m "feat(subagent): add runtime types and delegation policy"
```

---

## Task 3: Implement `ChatRunner` Over `hermes.Client`

**Files:**
- Create: `gormes/internal/subagent/runner.go`
- Create: `gormes/internal/subagent/runner_test.go`

- [ ] **Step 1: Write the failing tests**

Create `gormes/internal/subagent/runner_test.go`:

```go
package subagent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

func TestChatRunner_StopFinishReturnsSummary(t *testing.T) {
	cli := hermes.NewMockClient()
	cli.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "child ", TokensOut: 1},
		{Kind: hermes.EventToken, Token: "done", TokensOut: 2},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 7, TokensOut: 2},
	}, "sess-child")

	reg := tools.NewRegistry()
	runner := NewChatRunner(cli, reg, ChatRunnerConfig{Model: "hermes-agent", MaxToolDuration: 2 * time.Second})

	var events []Event
	res, err := runner.Run(context.Background(), Spec{Goal: "summarize repo", MaxIterations: 4}, func(ev Event) {
		events = append(events, ev)
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Status != StatusCompleted {
		t.Fatalf("Status = %q, want completed", res.Status)
	}
	if res.Summary != "child done" {
		t.Fatalf("Summary = %q, want child done", res.Summary)
	}
	if len(events) == 0 || events[0].Type != EventStarted {
		t.Fatalf("first event = %+v, want started", events)
	}
}

func TestChatRunner_ToolCallExecutesAllowedTool(t *testing.T) {
	cli := hermes.NewMockClient()
	cli.Script([]hermes.Event{
		{Kind: hermes.EventDone, FinishReason: "tool_calls", ToolCalls: []hermes.ToolCall{
			{ID: "call-1", Name: "echo", Arguments: json.RawMessage(`{"text":"hello"}`)},
		}},
	}, "sess-child")
	cli.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "tool ok", TokensOut: 2},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 11, TokensOut: 2},
	}, "sess-child")

	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{
		NameStr: "echo",
		ExecuteFn: func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{"text":"hello"}`), nil
		},
	})

	runner := NewChatRunner(cli, reg, ChatRunnerConfig{Model: "hermes-agent", MaxToolDuration: 2 * time.Second})
	res, err := runner.Run(context.Background(), Spec{
		Goal:         "call echo",
		MaxIterations: 4,
		AllowedTools: []string{"echo"},
	}, func(Event) {})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.ToolCalls) != 1 || res.ToolCalls[0] != "echo" {
		t.Fatalf("ToolCalls = %v, want [echo]", res.ToolCalls)
	}
	if len(cli.Requests()) != 2 {
		t.Fatalf("OpenStream requests = %d, want 2", len(cli.Requests()))
	}
}

func TestChatRunner_BlockedToolReturnsPolicyError(t *testing.T) {
	cli := hermes.NewMockClient()
	cli.Script([]hermes.Event{
		{Kind: hermes.EventDone, FinishReason: "tool_calls", ToolCalls: []hermes.ToolCall{
			{ID: "call-1", Name: "delegate_task", Arguments: json.RawMessage(`{"goal":"nested"}`)},
		}},
	}, "sess-child")
	cli.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "nested blocked", TokensOut: 2},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 9, TokensOut: 2},
	}, "sess-child")

	reg := tools.NewRegistry()
	runner := NewChatRunner(cli, reg, ChatRunnerConfig{Model: "hermes-agent", MaxToolDuration: 2 * time.Second})

	res, err := runner.Run(context.Background(), Spec{Goal: "try nested delegation", MaxIterations: 4}, func(Event) {})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Status != StatusCompleted {
		t.Fatalf("Status = %q, want completed", res.Status)
	}
	if len(cli.Requests()) != 2 {
		t.Fatalf("OpenStream requests = %d, want 2", len(cli.Requests()))
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd gormes && go test ./internal/subagent -run "TestChatRunner_" -count=1 -v`

Expected: FAIL with `undefined: NewChatRunner` and `undefined: ChatRunnerConfig`.

- [ ] **Step 3: Write the minimal implementation**

Create `gormes/internal/subagent/runner.go`:

```go
package subagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

type Runner interface {
	Run(ctx context.Context, spec Spec, emit func(Event)) (Result, error)
}

type ChatRunnerConfig struct {
	Model           string
	MaxToolDuration time.Duration
}

type ChatRunner struct {
	client hermes.Client
	reg    *tools.Registry
	cfg    ChatRunnerConfig
}

func NewChatRunner(client hermes.Client, reg *tools.Registry, cfg ChatRunnerConfig) *ChatRunner {
	return &ChatRunner{client: client, reg: reg, cfg: cfg}
}

func (r *ChatRunner) Run(ctx context.Context, spec Spec, emit func(Event)) (Result, error) {
	emit(Event{Type: EventStarted, Message: "child started"})
	req := hermes.ChatRequest{
		Model:    chooseModel(spec.Model, r.cfg.Model),
		Messages: childMessages(spec),
		Stream:   true,
		Tools:    r.descriptors(spec.AllowedTools),
	}
	var summary strings.Builder
	var seenTools []string

	for iter := 0; iter < spec.MaxIterations; iter++ {
		stream, err := r.client.OpenStream(ctx, req)
		if err != nil {
			return Result{Status: StatusFailed, Error: err.Error()}, err
		}

		var assistant strings.Builder
		var finish string
		var calls []hermes.ToolCall

		for {
			ev, err := stream.Recv(ctx)
			if err != nil {
				stream.Close()
				return Result{Status: StatusFailed, Error: err.Error()}, err
			}
			switch ev.Kind {
			case hermes.EventToken:
				assistant.WriteString(ev.Token)
			case hermes.EventDone:
				finish = ev.FinishReason
				calls = ev.ToolCalls
				goto done
			}
		}

	done:
		_ = stream.Close()
		switch finish {
		case "stop":
			summary.WriteString(assistant.String())
			emit(Event{Type: EventCompleted, Message: "child completed"})
			return Result{
				Status:       StatusCompleted,
				Summary:      strings.TrimSpace(summary.String()),
				FinishReason: finish,
				ToolCalls:    seenTools,
			}, nil
		case "tool_calls":
			req.Messages = append(req.Messages, hermes.Message{
				Role:      "assistant",
				Content:   assistant.String(),
				ToolCalls: calls,
			})
			for _, call := range calls {
				emit(Event{Type: EventToolCall, ToolName: call.Name})
				seenTools = append(seenTools, call.Name)
				msg := hermes.Message{
					Role:       "tool",
					Name:       call.Name,
					ToolCallID: call.ID,
					Content:    r.execTool(ctx, spec.AllowedTools, call),
				}
				req.Messages = append(req.Messages, msg)
			}
		default:
			return Result{Status: StatusFailed, Error: "unexpected finish reason: " + finish}, fmt.Errorf("subagent: unexpected finish reason %q", finish)
		}
	}

	return Result{Status: StatusFailed, Error: "max iterations reached"}, fmt.Errorf("subagent: max iterations reached")
}

func chooseModel(specModel, defaultModel string) string {
	if strings.TrimSpace(specModel) != "" {
		return strings.TrimSpace(specModel)
	}
	return defaultModel
}

func childMessages(spec Spec) []hermes.Message {
	system := "You are a delegated Gormes subagent. Work only on the scoped goal. Return a concise final answer."
	if spec.Context != "" {
		system += "\n\nScoped context:\n" + spec.Context
	}
	return []hermes.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: spec.Goal},
	}
}

func (r *ChatRunner) descriptors(allowed []string) []hermes.ToolDescriptor {
	if len(allowed) == 0 {
		all := r.reg.Descriptors()
		out := make([]hermes.ToolDescriptor, 0, len(all))
		for _, d := range all {
			if IsBlockedTool(d.Name) {
				continue
			}
			out = append(out, hermes.ToolDescriptor{Name: d.Name, Description: d.Description, Schema: d.Schema})
		}
		return out
	}
	out := make([]hermes.ToolDescriptor, 0, len(allowed))
	for _, name := range allowed {
		name = strings.TrimSpace(name)
		if name == "" || IsBlockedTool(name) {
			continue
		}
		t, ok := r.reg.Get(name)
		if !ok {
			continue
		}
		out = append(out, hermes.ToolDescriptor{Name: t.Name(), Description: t.Description(), Schema: t.Schema()})
	}
	return out
}

func (r *ChatRunner) execTool(ctx context.Context, allowed []string, call hermes.ToolCall) string {
	if IsBlockedTool(call.Name) {
		return `{"error":"blocked tool for child run"}`
	}
	if len(allowed) > 0 && !contains(allowed, call.Name) {
		return `{"error":"tool not allowlisted for child run"}`
	}
	t, ok := r.reg.Get(call.Name)
	if !ok {
		return `{"error":"unknown tool"}`
	}
	timeout := t.Timeout()
	if timeout <= 0 {
		timeout = r.cfg.MaxToolDuration
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	payload, err := t.Execute(callCtx, call.Arguments)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(payload)
}

func contains(names []string, want string) bool {
	for _, name := range names {
		if strings.TrimSpace(name) == want {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd gormes && go test ./internal/subagent -run "TestChatRunner_" -count=1 -v`

Expected: PASS, 3 tests.

- [ ] **Step 5: Commit**

```bash
git add gormes/internal/subagent/runner.go gormes/internal/subagent/runner_test.go
git commit -m "feat(subagent): add hermes-backed child chat runner"
```

---

## Task 4: Add `Manager`, `Handle`, And JSONL Run Logging

**Files:**
- Create: `gormes/internal/subagent/manager.go`
- Create: `gormes/internal/subagent/manager_test.go`
- Create: `gormes/internal/subagent/log.go`
- Create: `gormes/internal/subagent/log_test.go`

- [ ] **Step 1: Write the failing tests**

Create `gormes/internal/subagent/manager_test.go`:

```go
package subagent

import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
)

type scriptedRunner struct {
	run func(ctx context.Context, spec Spec, emit func(Event)) (Result, error)
}

func (s scriptedRunner) Run(ctx context.Context, spec Spec, emit func(Event)) (Result, error) {
	return s.run(ctx, spec, emit)
}

func TestManager_StartAndWait(t *testing.T) {
	mgr := NewManager(config.DelegationCfg{
		DefaultMaxIterations: 8,
		DefaultTimeout:       45 * time.Second,
		MaxChildDepth:        1,
	}, scriptedRunner{
		run: func(ctx context.Context, spec Spec, emit func(Event)) (Result, error) {
			emit(Event{Type: EventProgress, Message: "working"})
			return Result{Status: StatusCompleted, Summary: "done"}, nil
		},
	}, "")

	h, err := mgr.Start(context.Background(), Spec{Goal: "inspect"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	res, err := h.Wait(context.Background())
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if res.Status != StatusCompleted || res.Summary != "done" {
		t.Fatalf("Result = %+v, want completed/done", res)
	}
}

func TestManager_Cancel(t *testing.T) {
	mgr := NewManager(config.DelegationCfg{
		DefaultMaxIterations: 8,
		DefaultTimeout:       time.Second,
		MaxChildDepth:        1,
	}, scriptedRunner{
		run: func(ctx context.Context, spec Spec, emit func(Event)) (Result, error) {
			<-ctx.Done()
			return Result{}, ctx.Err()
		},
	}, "")

	h, err := mgr.Start(context.Background(), Spec{Goal: "wait"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := h.Cancel(); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	res, err := h.Wait(context.Background())
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if res.Status != StatusCancelled {
		t.Fatalf("Status = %q, want cancelled", res.Status)
	}
}
```

Create `gormes/internal/subagent/log_test.go`:

```go
package subagent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendRunLog_WritesJSONLLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.jsonl")
	rec := RunRecord{RunID: "run-1", Status: "completed", Summary: "ok"}
	if err := AppendRunLog(path, rec); err != nil {
		t.Fatalf("AppendRunLog: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, `"run_id":"run-1"`) || !strings.Contains(got, `"status":"completed"`) {
		t.Fatalf("log line = %q, want run_id/status json", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd gormes && go test ./internal/subagent -run "TestManager_|TestAppendRunLog_" -count=1 -v`

Expected: FAIL with `undefined: NewManager`, `undefined: RunRecord`, and `undefined: AppendRunLog`.

- [ ] **Step 3: Write the minimal implementation**

Create `gormes/internal/subagent/log.go`:

```go
package subagent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type RunRecord struct {
	RunID      string    `json:"run_id"`
	Status     string    `json:"status"`
	Summary    string    `json:"summary,omitempty"`
	Error      string    `json:"error,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}

func AppendRunLog(path string, rec RunRecord) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}
```

Create `gormes/internal/subagent/manager.go`:

```go
package subagent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
)

type Handle struct {
	ID     string
	Events <-chan Event

	cancel context.CancelFunc
	done   chan struct{}

	mu     sync.RWMutex
	result Result
}

func (h *Handle) Wait(ctx context.Context) (Result, error) {
	select {
	case <-h.done:
		h.mu.RLock()
		defer h.mu.RUnlock()
		return h.result, nil
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
}

func (h *Handle) Cancel() error {
	if h.cancel == nil {
		return errors.New("subagent: nil cancel func")
	}
	h.cancel()
	return nil
}

type Manager struct {
	cfg     config.DelegationCfg
	runner  Runner
	logPath string

	mu      sync.Mutex
	handles map[string]*Handle
}

func NewManager(cfg config.DelegationCfg, runner Runner, logPath string) *Manager {
	return &Manager{
		cfg:     cfg,
		runner:  runner,
		logPath: logPath,
		handles: make(map[string]*Handle),
	}
}

func (m *Manager) Start(parent context.Context, spec Spec) (*Handle, error) {
	spec, err := ApplyDefaults(spec, m.cfg)
	if err != nil {
		return nil, err
	}
	runID := newRunID()
	ctx := parent
	cancel := func() {}
	if spec.Timeout > 0 {
		ctx, cancel = context.WithTimeout(parent, spec.Timeout)
	} else {
		ctx, cancel = context.WithCancel(parent)
	}

	events := make(chan Event, 16)
	h := &Handle{ID: runID, Events: events, done: make(chan struct{}), cancel: cancel}

	m.mu.Lock()
	m.handles[runID] = h
	m.mu.Unlock()

	go func() {
		started := time.Now().UTC()
		defer close(events)
		defer close(h.done)
		defer cancel()
		defer func() {
			m.mu.Lock()
			delete(m.handles, runID)
			m.mu.Unlock()
		}()

		res, err := m.runner.Run(ctx, spec, func(ev Event) {
			select {
			case events <- ev:
			case <-ctx.Done():
			}
		})
		res.RunID = runID
		if err != nil {
			switch ctx.Err() {
			case context.Canceled:
				res.Status = StatusCancelled
				res.Error = ctx.Err().Error()
			case context.DeadlineExceeded:
				res.Status = StatusTimedOut
				res.Error = ctx.Err().Error()
			default:
				res.Status = StatusFailed
				res.Error = err.Error()
			}
		}
		finished := time.Now().UTC()
		_ = AppendRunLog(m.logPath, RunRecord{
			RunID:      runID,
			Status:     string(res.Status),
			Summary:    res.Summary,
			Error:      res.Error,
			StartedAt:  started,
			FinishedAt: finished,
		})
		h.mu.Lock()
		h.result = res
		h.mu.Unlock()
	}()

	return h, nil
}

func newRunID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return "sa_" + hex.EncodeToString(b[:])
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd gormes && go test ./internal/subagent -run "TestManager_|TestAppendRunLog_" -count=1 -v`

Expected: PASS, 3 tests.

- [ ] **Step 5: Commit**

```bash
git add gormes/internal/subagent/manager.go gormes/internal/subagent/manager_test.go gormes/internal/subagent/log.go gormes/internal/subagent/log_test.go
git commit -m "feat(subagent): add manager lifecycle and run logging"
```

---

## Task 5: Add `delegate_task` Tool + Telegram Wiring

**Files:**
- Create: `gormes/internal/subagent/delegate_tool.go`
- Create: `gormes/internal/subagent/delegate_tool_test.go`
- Create: `gormes/cmd/gormes/delegation.go`
- Create: `gormes/cmd/gormes/delegation_test.go`
- Modify: `gormes/cmd/gormes/telegram.go`

- [ ] **Step 1: Write the failing tests**

Create `gormes/internal/subagent/delegate_tool_test.go`:

```go
package subagent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
)

func TestDelegateTool_ExecuteReturnsChildResult(t *testing.T) {
	mgr := NewManager(config.DelegationCfg{
		DefaultMaxIterations: 8,
		DefaultTimeout:       45 * time.Second,
		MaxChildDepth:        1,
	}, scriptedRunner{
		run: func(ctx context.Context, spec Spec, emit func(Event)) (Result, error) {
			return Result{Status: StatusCompleted, Summary: "delegated ok"}, nil
		},
	}, "")

	tool := NewDelegateTool(mgr)
	out, err := tool.Execute(context.Background(), json.RawMessage(`{"goal":"inspect repo"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got struct {
		RunID   string `json:"run_id"`
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Status != "completed" || got.Summary != "delegated ok" || got.RunID == "" {
		t.Fatalf("tool output = %+v, want completed/delegated ok/non-empty run_id", got)
	}
}
```

Create `gormes/cmd/gormes/delegation_test.go`:

```go
package main

import (
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

func TestRegisterDelegation_DisabledLeavesRegistryUnchanged(t *testing.T) {
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})

	mgr := registerDelegation(config.Config{}, reg, hermes.NewMockClient())
	if mgr != nil {
		t.Fatalf("registerDelegation() returned manager when disabled")
	}
	if _, ok := reg.Get("delegate_task"); ok {
		t.Fatalf("delegate_task unexpectedly registered")
	}
}

func TestRegisterDelegation_EnabledRegistersTool(t *testing.T) {
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})

	cfg := config.Config{
		Hermes: config.HermesCfg{Model: "hermes-agent"},
		Delegation: config.DelegationCfg{
			Enabled:              true,
			DefaultMaxIterations: 8,
			DefaultTimeout:       45 * time.Second,
			MaxChildDepth:        1,
		},
	}
	mgr := registerDelegation(cfg, reg, hermes.NewMockClient())
	if mgr == nil {
		t.Fatalf("registerDelegation() returned nil manager")
	}
	if _, ok := reg.Get("delegate_task"); !ok {
		t.Fatalf("delegate_task not registered")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd gormes && go test ./internal/subagent ./cmd/gormes -run "TestDelegateTool_|TestRegisterDelegation_" -count=1 -v`

Expected: FAIL with `undefined: NewDelegateTool` and `undefined: registerDelegation`.

- [ ] **Step 3: Write the minimal implementation**

Create `gormes/internal/subagent/delegate_tool.go`:

```go
package subagent

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

type DelegateTool struct {
	mgr *Manager
}

func NewDelegateTool(mgr *Manager) *DelegateTool { return &DelegateTool{mgr: mgr} }

func (*DelegateTool) Name() string        { return "delegate_task" }
func (*DelegateTool) Description() string { return "Delegate a bounded sub-task to an isolated child run." }
func (*DelegateTool) Timeout() time.Duration { return 2 * time.Minute }
func (*DelegateTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"goal":{"type":"string"},"context":{"type":"string"},"model":{"type":"string"},"max_iterations":{"type":"integer"},"timeout_seconds":{"type":"integer"},"allowed_tools":{"type":"array","items":{"type":"string"}}},"required":["goal"]}`)
}

func (t *DelegateTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Goal          string   `json:"goal"`
		Context       string   `json:"context"`
		Model         string   `json:"model"`
		MaxIterations int      `json:"max_iterations"`
		TimeoutSeconds int     `json:"timeout_seconds"`
		AllowedTools  []string `json:"allowed_tools"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, err
	}
	spec := Spec{
		Goal:          strings.TrimSpace(in.Goal),
		Context:       strings.TrimSpace(in.Context),
		Model:         strings.TrimSpace(in.Model),
		MaxIterations: in.MaxIterations,
		AllowedTools:  in.AllowedTools,
	}
	if in.TimeoutSeconds > 0 {
		spec.Timeout = time.Duration(in.TimeoutSeconds) * time.Second
	}
	h, err := t.mgr.Start(ctx, spec)
	if err != nil {
		return nil, err
	}
	res, err := h.Wait(ctx)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		RunID   string `json:"run_id"`
		Status  string `json:"status"`
		Summary string `json:"summary,omitempty"`
		Error   string `json:"error,omitempty"`
	}{
		RunID:   res.RunID,
		Status:  string(res.Status),
		Summary: res.Summary,
		Error:   res.Error,
	})
}
```

Create `gormes/cmd/gormes/delegation.go`:

```go
package main

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/subagent"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

func registerDelegation(cfg config.Config, reg *tools.Registry, hc hermes.Client) *subagent.Manager {
	if !cfg.Delegation.Enabled {
		return nil
	}
	runner := subagent.NewChatRunner(hc, reg, subagent.ChatRunnerConfig{
		Model:           cfg.Hermes.Model,
		MaxToolDuration: 30 * time.Second,
	})
	mgr := subagent.NewManager(cfg.Delegation, runner, cfg.DelegationRunLogPath())
	reg.MustRegister(subagent.NewDelegateTool(mgr))
	return mgr
}
```

Modify `gormes/cmd/gormes/telegram.go` where the registry is built:

```go
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})
	reg.MustRegister(&tools.NowTool{})
	reg.MustRegister(&tools.RandIntTool{})
	_ = registerDelegation(cfg, reg, hc)

	tm := telemetry.New()
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd gormes && go test ./internal/subagent ./cmd/gormes -run "TestDelegateTool_|TestRegisterDelegation_" -count=1 -v`

Expected: PASS, 3 tests.

- [ ] **Step 5: Commit**

```bash
git add gormes/internal/subagent/delegate_tool.go gormes/internal/subagent/delegate_tool_test.go gormes/cmd/gormes/delegation.go gormes/cmd/gormes/delegation_test.go gormes/cmd/gormes/telegram.go
git commit -m "feat(subagent): register delegate_task in telegram runtime"
```

---

## Task 6: Prove The Runtime End-To-End

**Files:**
- Create: `gormes/internal/subagent/integration_test.go`

- [ ] **Step 1: Write the failing integration test**

Create `gormes/internal/subagent/integration_test.go`:

```go
package subagent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

func TestDelegateTask_EndToEndChildToolLoop(t *testing.T) {
	cli := hermes.NewMockClient()
	cli.Script([]hermes.Event{
		{Kind: hermes.EventDone, FinishReason: "tool_calls", ToolCalls: []hermes.ToolCall{
			{ID: "call-1", Name: "echo", Arguments: json.RawMessage(`{"text":"hello from child"}`)},
		}},
	}, "sess-child")
	cli.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "child completed", TokensOut: 2},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 12, TokensOut: 2},
	}, "sess-child")

	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{
		NameStr: "echo",
		ExecuteFn: func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{"text":"hello from child"}`), nil
		},
	})

	runner := NewChatRunner(cli, reg, ChatRunnerConfig{Model: "hermes-agent", MaxToolDuration: 2 * time.Second})
	mgr := NewManager(config.DelegationCfg{
		DefaultMaxIterations: 8,
		DefaultTimeout:       45 * time.Second,
		MaxChildDepth:        1,
	}, runner, "")

	tool := NewDelegateTool(mgr)
	out, err := tool.Execute(context.Background(), json.RawMessage(`{"goal":"run child","allowed_tools":["echo"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var got struct {
		RunID   string `json:"run_id"`
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Status != "completed" || got.Summary != "child completed" || got.RunID == "" {
		t.Fatalf("delegate_task output = %+v, want completed/child completed/non-empty run_id", got)
	}
	if len(cli.Requests()) != 2 {
		t.Fatalf("OpenStream requests = %d, want 2", len(cli.Requests()))
	}
}
```

- [ ] **Step 2: Run the integration test to verify it fails**

Run: `cd gormes && go test ./internal/subagent -run TestDelegateTask_EndToEndChildToolLoop -count=1 -v`

Expected: FAIL if any runtime seams are missing; do not proceed until the failure is for the intended reason.

- [ ] **Step 3: Make the minimal fixes needed to turn it green**

If the previous five tasks were implemented exactly, no new production files should be necessary here. Fix only the smallest issues exposed by the integration test. Typical minimal changes:

```go
// Example: if the second child request forgot to include the tool reply.
req.Messages = append(req.Messages, hermes.Message{
	Role:       "tool",
	Name:       call.Name,
	ToolCallID: call.ID,
	Content:    r.execTool(ctx, spec.AllowedTools, call),
})

// Example: if the child summary kept trailing whitespace.
Summary: strings.TrimSpace(summary.String()),
```

Do not add new features in this step. The only goal is to prove the existing runtime contracts end-to-end.

- [ ] **Step 4: Run the focused and broader suites**

Run the focused package:

`cd gormes && go test ./internal/subagent -count=1 -race -shuffle=on -v`

Expected: PASS.

Run the touched packages:

`cd gormes && go test ./internal/subagent ./internal/config ./cmd/gormes -count=1 -race -shuffle=on -v`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add gormes/internal/subagent/types.go gormes/internal/subagent/types_test.go gormes/internal/subagent/policy.go gormes/internal/subagent/policy_test.go gormes/internal/subagent/runner.go gormes/internal/subagent/runner_test.go gormes/internal/subagent/manager.go gormes/internal/subagent/manager_test.go gormes/internal/subagent/log.go gormes/internal/subagent/log_test.go gormes/internal/subagent/delegate_tool.go gormes/internal/subagent/delegate_tool_test.go gormes/internal/subagent/integration_test.go gormes/internal/config/config.go gormes/internal/config/config_test.go gormes/cmd/gormes/delegation.go gormes/cmd/gormes/delegation_test.go gormes/cmd/gormes/telegram.go
git commit -m "feat(subagent): ship phase 2.e0 delegated child runtime"
```

---

## Self-Review Checklist

Before execution, verify the plan against the approved spec:

1. **Spec coverage for this decomposed plan:** `2.E0` runtime is fully covered by Tasks 1–6. `2.G0` and `2.G1-lite` are intentionally deferred to follow-on plans rather than being smuggled into this runtime plan.
2. **Placeholder scan:** This plan contains exact file paths, code snippets, test commands, and commit messages. No `TODO`, `TBD`, or “implement later” placeholders are permitted.
3. **Type consistency:** This plan uses one stable naming set throughout:
   - `Spec`
   - `EventType`
   - `Event`
   - `ResultStatus`
   - `Result`
   - `Runner`
   - `ChatRunner`
   - `Manager`
   - `Handle`
   - `DelegateTool`

## Follow-On Plans After This Lands Green

Do **not** start these before `2.E0` is green:

1. **`2.G0` Static Skill Runtime**
   Filesystem-backed active skill store, `SKILL.md` parser, deterministic selector, prompt injection.
2. **`2.G1-lite` Reviewed Candidate Promotion**
   Candidate artifact draft/store/promotion flow that stays inactive until explicit approval.
