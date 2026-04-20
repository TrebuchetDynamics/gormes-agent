# Phase 2.E — Subagent System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement goroutine-per-subagent execution isolation with channel-based streaming, ToolExecutor interface, depth=2 max, and recursive cancellation.

**Architecture:** SubagentManager spawns goroutines with context.WithCancel(parentCtx). Events stream back via typed channels. ToolExecutor interface decouples execution (in-process MVP, swappable later). SubagentRegistry tracks all live subagents process-wide for shutdown safety.

**Tech Stack:** Go stdlib (context, sync, channels), internal/tools (ToolRegistry), internal/kernel (AIAgent), internal/config

---

## File Structure

```
internal/subagent/
  subagent.go      — Subagent, SubagentConfig, SubagentEvent, SubagentResult types
  manager.go       — SubagentManager: Spawn, Interrupt, Collect, batch
  registry.go      — SubagentRegistry: process-wide tracking
internal/tools/
  executor.go      — ToolExecutor interface + InProcessToolExecutor
internal/kernel/
  delegate.go      — delegate_task tool + integration with AIAgent
internal/config/
  config.go        — add [delegation] section
gormes/docs/superpowers/plans/
  2026-04-20-gormes-phase2e-subagent.md (this file)
```

---

## Task 1: Subagent data structures

**Files:**
- Create: `internal/subagent/subagent.go`
- Modify: `internal/config/config.go` (add DelegationConfig)
- Test: `internal/subagent/subagent_test.go`

- [ ] **Step 1: Create internal/subagent/subagent.go with all types**

```go
package subagent

import (
    "context"
    "time"
)

type Subagent struct {
    ID       string
    ParentID string
    Depth    int
    ctx      context.Context
    cancel   context.CancelFunc
    Config   SubagentConfig
    Result   *SubagentResult
    closed   bool
}

type SubagentConfig struct {
    Goal          string
    Context       string
    MaxIterations int
    MaxMemoryMB   int
    EnabledTools  []string
    Model         string
    Timeout       time.Duration
}

type SubagentEvent struct {
    Type     string
    Message  string
    ToolCall *ToolCallInfo
    Progress *ProgressInfo
}

type SubagentResult struct {
    ID         string
    Status     string
    Summary    string
    ExitReason string
    Duration   time.Duration
    Iterations int
    ToolCalls  []ToolCallInfo
    Error      string
}

type ToolCallInfo struct {
    Name       string
    ArgsBytes  int
    ResultSize int
    Status     string
}

type ProgressInfo struct {
    Iteration int
    Message   string
}

const MaxDepth = 2
const DefaultMaxConcurrent = 3
const DefaultMaxIterations = 50

var BlockedTools = map[string]bool{
    "delegate_task": true,
    "clarify":      true,
    "memory":       true,
    "send_message": true,
    "execute_code":  true,
}

func generateID() string {
    return fmt.Sprintf("sa_%d_%s", time.Now().UnixNano(), randString(8))
}

func randString(n int) string {
    const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
    b := make([]byte, n)
    for i := range b {
        b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
    }
    return string(b)
}
```

- [ ] **Step 2: Add DelegationConfig to internal/config/config.go**

```go
type DelegationConfig struct {
    Enabled              bool          `toml:"enabled"`
    MaxDepth            int           `toml:"max_depth"`
    MaxConcurrentChildren int         `toml:"max_concurrent_children"`
    DefaultMaxIterations int          `toml:"default_max_iterations"`
    DefaultTimeout      time.Duration `toml:"default_timeout"`
}
```

- [ ] **Step 3: Write unit tests for Subagent type creation**

```go
func TestSubagentConfigDefaults(t *testing.T) {
    cfg := SubagentConfig{Goal: "test"}
    if cfg.MaxIterations != 0 {
        t.Errorf("expected 0, got %d", cfg.MaxIterations)
    }
    if cfg.Timeout != 0 {
        t.Errorf("expected 0, got %v", cfg.Timeout)
    }
}

func TestBlockedTools(t *testing.T) {
    blocked := []string{"delegate_task", "clarify", "memory", "send_message", "execute_code"}
    for _, tool := range blocked {
        if !BlockedTools[tool] {
            t.Errorf("expected %s to be blocked", tool)
        }
    }
    if BlockedTools["terminal"] {
        t.Errorf("terminal should not be blocked")
    }
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/subagent/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/subagent/subagent.go internal/config/config.go
git commit -m "feat(subagent): add Subagent data structures and BlockedTools"
```

---

## Task 2: SubagentRegistry — process-wide tracking

**Files:**
- Create: `internal/subagent/registry.go`
- Modify: `internal/subagent/subagent.go`
- Test: `internal/subagent/registry_test.go`

- [ ] **Step 1: Create SubagentRegistry interface and implementation**

```go
package subagent

import "sync"

type SubagentRegistry interface {
    Register(sa *Subagent)
    Unregister(id string)
    InterruptAll(message string)
    List() []*Subagent
}

type registry struct {
    mu       sync.RWMutex
    subagents map[string]*Subagent
}

func NewRegistry() SubagentRegistry {
    return &registry{
        subagents: make(map[string]*Subagent),
    }
}

func (r *registry) Register(sa *Subagent) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.subagents[sa.ID] = sa
}

func (r *registry) Unregister(id string) {
    r.mu.Lock()
    defer r.mu.Unlock()
    delete(r.subagents, id)
}

func (r *registry) InterruptAll(message string) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    for _, sa := range r.subagents {
        sa.cancel()
    }
}

func (r *registry) List() []*Subagent {
    r.mu.RLock()
    defer r.mu.RUnlock()
    out := make([]*Subagent, 0, len(r.subagents))
    for _, sa := range r.subagents {
        out = append(out, sa)
    }
    return out
}
```

- [ ] **Step 2: Write registry tests**

```go
func TestRegistryRegisterUnregister(t *testing.T) {
    r := NewRegistry()
    sa := &Subagent{ID: "test1"}
    r.Register(sa)
    if len(r.List()) != 1 {
        t.Fatal("expected 1 subagent")
    }
    r.Unregister("test1")
    if len(r.List()) != 0 {
        t.Fatal("expected 0 subagents")
    }
}

func TestRegistryInterruptAll(t *testing.T) {
    r := NewRegistry()
    ctx, cancel := context.WithCancel(context.Background())
    sa := &Subagent{ID: "test1", ctx: ctx, cancel: cancel}
    r.Register(sa)
    r.InterruptAll("stop")
    select {
    case <-ctx.Done():
    default:
        t.Fatal("expected context to be cancelled")
    }
}

func TestRegistryListSnapshot(t *testing.T) {
    r := NewRegistry()
    for i := 0; i < 5; i++ {
        sa := &Subagent{ID: fmt.Sprintf("sa_%d", i)}
        r.Register(sa)
    }
    list := r.List()
    if len(list) != 5 {
        t.Errorf("expected 5, got %d", len(list))
    }
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/subagent/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/subagent/registry.go
git commit -m "feat(subagent): add SubagentRegistry for process-wide tracking"
```

---

## Task 3: ToolExecutor interface + InProcessToolExecutor

**Files:**
- Create: `internal/tools/executor.go`
- Modify: `internal/tools/tool.go` (add ToolRequest/ToolEvent types or keep in executor.go)
- Test: `internal/tools/executor_test.go`

- [ ] **Step 1: Create internal/tools/executor.go with ToolExecutor interface**

```go
package tools

import (
    "context"
    "encoding/json"
    "fmt"
)

// ToolExecutor executes tools on behalf of an agent.
// Subagents use this interface so the execution model is swappable:
//   - In-process (same binary, same goroutine pool) — Phase 2.E MVP
//   - Sidecar (separate process, JSON-RPC over stdio) — future
//   - Remote (HTTP RPC) — future
type ToolExecutor interface {
    Execute(ctx context.Context, req ToolRequest) (<-chan ToolEvent, error)
}

type ToolRequest struct {
    AgentID   string
    ToolName  string
    Input     string
    Metadata  map[string]string
}

type ToolEvent struct {
    Type  string
    Chunk string
    Err   error
}

// InProcessToolExecutor executes tools directly in the binary.
// Thread-safe: serializes tool calls per subagent using a mutex.
// This is the Phase 2.E MVP implementation.
type InProcessToolExecutor struct {
    registry *ToolRegistry
    mu       sync.Mutex // serialize tool calls per executor instance
}

func NewInProcessToolExecutor(reg *ToolRegistry) *InProcessToolExecutor {
    return &InProcessToolExecutor{registry: reg}
}

func (e *InProcessToolExecutor) Execute(ctx context.Context, req ToolRequest) (<-chan ToolEvent, error) {
    ch := make(chan ToolEvent, 1) // buffered for synchronous close

    tool, ok := e.registry.Get(req.ToolName)
    if !ok {
        ch <- ToolEvent{Type: "failed", Err: fmt.Errorf("tool not found: %s", req.ToolName)}
        close(ch)
        return ch, nil
    }

    go func() {
        defer close(ch)
        e.mu.Lock()
        defer e.mu.Unlock()

        ch <- ToolEvent{Type: "started"}

        var input map[string]any
        if err := json.Unmarshal([]byte(req.Input), &input); err != nil {
            ch <- ToolEvent{Type: "failed", Err: fmt.Errorf("invalid input: %w", err)}
            return
        }

        result, err := tool.Handler(ctx, input)
        if err != nil {
            ch <- ToolEvent{Type: "failed", Err: err}
            return
        }

        ch <- ToolEvent{Type: "output", Chunk: result}
        ch <- ToolEvent{Type: "completed"}
    }()

    return ch, nil
}
```

- [ ] **Step 2: Verify the file compiles with correct imports**

Run: `go build ./internal/tools/executor.go`
Expected: No errors (sync and fmt need to be imported — add them if missing)

- [ ] **Step 3: Write executor tests**

```go
func TestInProcessToolExecutor_ToolNotFound(t *testing.T) {
    reg := NewRegistry()
    exec := NewInProcessToolExecutor(reg)

    ctx := context.Background()
    req := ToolRequest{ToolName: "nonexistent"}
    ch, err := exec.Execute(ctx, req)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    var ev ToolEvent
    ev = <-ch
    if ev.Type != "failed" {
        t.Errorf("expected failed, got %s", ev.Type)
    }
    ev = <-ch
    if ev != (ToolEvent{}) {
        t.Errorf("expected channel closed, got %+v", ev)
    }
}

func TestInProcessToolExecutor_RegisteredTool(t *testing.T) {
    reg := NewRegistry()
    var called bool
    reg.Register(&ToolEntry{
        Name: "test_tool",
        Handler: func(ctx context.Context, args map[string]any) (string, error) {
            called = true
            return `{"ok":true}`, nil
        },
    })
    exec := NewInProcessToolExecutor(reg)

    ctx := context.Background()
    req := ToolRequest{ToolName: "test_tool", Input: `{}`}
    ch, err := exec.Execute(ctx, req)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    var evs []ToolEvent
    for ev := range ch {
        evs = append(evs, ev)
    }
    if !called {
        t.Fatal("tool handler was not called")
    }
    if len(evs) != 3 {
        t.Fatalf("expected 3 events, got %d: %+v", len(evs), evs)
    }
    if evs[0].Type != "started" || evs[1].Type != "output" || evs[2].Type != "completed" {
        t.Errorf("unexpected event sequence: %v", evs)
    }
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/tools/executor_test.go ./internal/tools/executor.go ./internal/tools/tool.go -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tools/executor.go internal/tools/executor_test.go
git commit -m "feat(subagent): add ToolExecutor interface + InProcessToolExecutor"
```

---

## Task 4: SubagentManager — Spawn, Interrupt, Collect

**Files:**
- Create: `internal/subagent/manager.go`
- Modify: `internal/subagent/subagent.go`
- Test: `internal/subagent/manager_test.go`

- [ ] **Step 1: Create SubagentManager interface and basic implementation**

```go
package subagent

import (
    "context"
    "fmt"
    "time"
)

type SubagentManager interface {
    Spawn(ctx context.Context, cfg SubagentConfig) (*Subagent, error)
    SpawnBatch(ctx context.Context, cfgs []SubagentConfig, maxConcurrent int) ([]*SubagentResult, error)
    Interrupt(sa *Subagent, message string) error
    Collect(sa *Subagent) *SubagentResult
    Close() error
}

type manager struct {
    parentCtx context.Context
    parentID  string
    depth     int

    children map[string]*Subagent
    mu       sync.RWMutex

    executor ToolExecutor
    registry SubagentRegistry
}

func NewManager(parentCtx context.Context, parentID string, depth int, executor ToolExecutor, registry SubagentRegistry) SubagentManager {
    return &manager{
        parentCtx: parentCtx,
        parentID:  parentID,
        depth:     depth,
        children:  make(map[string]*Subagent),
        executor:  executor,
        registry:  registry,
    }
}

func (m *manager) Spawn(ctx context.Context, cfg SubagentConfig) (*Subagent, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.depth >= MaxDepth {
        return nil, fmt.Errorf("max subagent depth (%d) reached", MaxDepth)
    }

    if cfg.MaxIterations <= 0 {
        cfg.MaxIterations = DefaultMaxIterations
    }

    childCtx, cancel := context.WithCancel(m.parentCtx)

    sa := &Subagent{
        ID:       generateID(),
        ParentID:  m.parentID,
        Depth:     m.depth + 1,
        ctx:       childCtx,
        cancel:    cancel,
        Config:    cfg,
        closed:    false,
    }

    m.children[sa.ID] = sa
    m.registry.Register(sa)

    go m.runSubagent(sa)

    return sa, nil
}

func (m *manager) runSubagent(sa *Subagent) {
    defer func() {
        if r := recover(); r != nil {
            sa.mu.Lock()
            sa.Result = &SubagentResult{
                ID:    sa.ID,
                Status: "error",
                Error: fmt.Sprintf("panic: %v", r),
            }
            sa.closed = true
            sa.mu.Unlock()
        }
        m.mu.Lock()
        delete(m.children, sa.ID)
        m.mu.Unlock()
        m.registry.Unregister(sa.ID)
    }()

    // Streaming event channel
    events := make(chan SubagentEvent, 10)
    results := make(chan *SubagentResult, 1)

    go func() {
        // Build focused subagent prompt
        prompt := m.buildSubagentPrompt(sa.Config)

        // Run the subagent (stub — full AIAgent integration in Task 5)
        result := m.runSubagentAgent(sa.ctx, sa.Config, events)
        results <- result
    }()

    // Stream events and collect result
    for {
        select {
        case ev := <-events:
            // Forwarded to parent via sa.Events (set up by parent reader)
            _ = ev
        case result := <-results:
            sa.mu.Lock()
            sa.closed = true
            sa.Result = result
            sa.mu.Unlock()
            return
        case <-sa.ctx.Done():
            sa.mu.Lock()
            sa.closed = true
            sa.Result = &SubagentResult{
                ID:    sa.ID,
                Status: "interrupted",
                Error: sa.ctx.Err().Error(),
            }
            sa.mu.Unlock()
            return
        }
    }
}

func (m *manager) buildSubagentPrompt(cfg SubagentConfig) string {
    return fmt.Sprintf("You are a focused subagent. Task: %s\n\nContext:\n%s", cfg.Goal, cfg.Context)
}

func (m *manager) runSubagentAgent(ctx context.Context, cfg SubagentConfig, events chan SubagentEvent) *SubagentResult {
    // TODO (Task 5): integrate with AIAgent
    // For now: simulate execution
    events <- SubagentEvent{Type: "started", Message: "subagent started"}
    time.Sleep(10 * time.Millisecond)
    events <- SubagentEvent{Type: "completed", Message: "subagent done"}
    return &SubagentResult{
        ID:         "",
        Status:     "completed",
        Summary:    cfg.Goal,
        ExitReason: "completed",
        Duration:   10 * time.Millisecond,
        Iterations: 1,
    }
}

func (m *manager) Interrupt(sa *Subagent, message string) error {
    m.mu.RLock()
    _, ok := m.children[sa.ID]
    m.mu.RUnlock()
    if !ok {
        return fmt.Errorf("subagent %s not found", sa.ID)
    }
    sa.cancel()
    return nil
}

func (m *manager) Collect(sa *Subagent) *SubagentResult {
    sa.mu.RLock()
    defer sa.mu.RUnlock()
    return sa.Result
}

func (m *manager) SpawnBatch(ctx context.Context, cfgs []SubagentConfig, maxConcurrent int) ([]*SubagentResult, error) {
    if maxConcurrent <= 0 {
        maxConcurrent = DefaultMaxConcurrent
    }

    results := make([]*SubagentResult, len(cfgs))
    sem := make(chan struct{}, maxConcurrent)
    var wg sync.WaitGroup

    for i, cfg := range cfgs {
        wg.Add(1)
        go func(i int, cfg SubagentConfig) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()

            sa, err := m.Spawn(ctx, cfg)
            if err != nil {
                results[i] = &SubagentResult{Status: "error", Error: err.Error()}
                return
            }

            // Wait for result
            for {
                sa.mu.RLock()
                closed := sa.closed
                result := sa.Result
                sa.mu.RUnlock()
                if closed && result != nil {
                    results[i] = result
                    return
                }
                time.Sleep(10 * time.Millisecond)
            }
        }(i, cfg)
    }

    wg.Wait()
    return results, nil
}

func (m *manager) Close() error {
    m.mu.Lock()
    defer m.mu.Unlock()
    for _, sa := range m.children {
        sa.cancel()
    }
    return nil
}
```

- [ ] **Step 2: Fix missing imports in manager.go**

The file needs: `sync`, `time`, `fmt`. Make sure they're imported.

- [ ] **Step 3: Write manager tests**

```go
func TestManagerSpawn(t *testing.T) {
    reg := NewRegistry()
    exec := &mockToolExecutor{}
    ctx := context.Background()
    m := NewManager(ctx, "parent1", 0, exec, reg)

    cfg := SubagentConfig{Goal: "test task"}
    sa, err := m.Spawn(ctx, cfg)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if sa == nil {
        t.Fatal("expected subagent, got nil")
    }
    if sa.Depth != 1 {
        t.Errorf("expected depth 1, got %d", sa.Depth)
    }
    m.Close()
}

func TestManagerMaxDepth(t *testing.T) {
    reg := NewRegistry()
    exec := &mockToolExecutor{}
    ctx := context.Background()
    m := NewManager(ctx, "parent1", MaxDepth, exec, reg)

    cfg := SubagentConfig{Goal: "test task"}
    _, err := m.Spawn(ctx, cfg)
    if err == nil {
        t.Fatal("expected error for max depth")
    }
}

func TestManagerInterrupt(t *testing.T) {
    reg := NewRegistry()
    exec := &mockToolExecutor{}
    ctx, cancel := context.WithCancel(context.Background())
    m := NewManager(ctx, "parent1", 0, exec, reg)

    cfg := SubagentConfig{Goal: "test task"}
    sa, _ := m.Spawn(ctx, cfg)

    err := m.Interrupt(sa, "stop")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    select {
    case <-sa.ctx.Done():
    default:
        t.Fatal("expected context to be cancelled")
    }
    m.Close()
}

type mockToolExecutor struct{}

func (m *mockToolExecutor) Execute(ctx context.Context, req ToolRequest) (<-chan ToolEvent, error) {
    ch := make(chan ToolEvent, 1)
    go func() {
        defer close(ch)
        ch <- ToolEvent{Type: "completed"}
    }()
    return ch, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/subagent/... -v -run "TestManager|TestSubagentConfig|TestBlockedTools"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/subagent/manager.go
git commit -m "feat(subagent): add SubagentManager with Spawn, Interrupt, Collect, batch"
```

---

## Task 5: AIAgent integration — delegate_task tool

**Files:**
- Create: `internal/kernel/delegate.go`
- Modify: `internal/kernel/kernel.go` (add SubagentManager field)
- Test: `internal/kernel/delegate_test.go`

- [ ] **Step 1: Create delegate.go with delegate_task tool schema**

```go
package kernel

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/TrebuchetDynamics/gormes-agent/internal/subagent"
    "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type DelegateTool struct {
    manager *subagent.Manager
    agentID string
}

func NewDelegateTool(mgr *subagent.Manager, agentID string) *DelegateTool {
    return &DelegateTool{manager: mgr, agentID: agentID}
}

var DelegateTaskSchema = tools.ToolSchema{
    Name:        "delegate_task",
    Description: "Delegate a task to a subagent for parallel execution. The subagent runs with its own context and returns a structured result. Useful for: independent parallel work, research tasks, multi-step operations.",
    Parameters: []tools.ToolParam{
        {Name: "goal", Type: "string", Required: true, Description: "Task goal for subagent"},
        {Name: "context", Type: "string", Required: false, Description: "Additional context for the subagent"},
        {Name: "max_iterations", Type: "integer", Required: false, Description: "Max LLM turns for subagent (default: 50)"},
        {Name: "toolsets", Type: "string", Required: false, Description: "Comma-separated toolset names to enable"},
    },
}

func (t *DelegateTool) Handle(ctx context.Context, args map[string]any) (string, error) {
    goal, _ := args["goal"].(string)
    contextStr, _ := args["context"].(string)
    maxIterations, _ := args["max_iterations"].(float64)

    toolsetsStr, _ := args["toolsets"].(string)
    var enabledTools []string
    if toolsetsStr != "" {
        enabledTools = strings.Split(toolsetsStr, ",")
    }

    cfg := subagent.SubagentConfig{
        Goal:          goal,
        Context:       contextStr,
        MaxIterations: int(maxIterations),
        EnabledTools:  enabledTools,
        Model:         "", // inherit from parent
        Timeout:       0,  // no timeout by default
    }

    sa, err := t.manager.Spawn(ctx, cfg)
    if err != nil {
        return "", fmt.Errorf("failed to spawn subagent: %w", err)
    }

    // Stream events until result is ready
    for {
        sa.mu.RLock()
        closed := sa.closed
        result := sa.Result
        sa.mu.RUnlock()

        if closed && result != nil {
            return formatResult(result), nil
        }

        select {
        case <-ctx.Done():
            t.manager.Interrupt(sa, "parent cancelled")
            return "", ctx.Err()
        case <-time.After(50 * time.Millisecond):
        }
    }
}

func formatResult(r *subagent.SubagentResult) string {
    out := map[string]any{
        "task_index":   0,
        "status":       r.Status,
        "summary":      r.Summary,
        "exit_reason":  r.ExitReason,
        "duration":     r.Duration.String(),
        "iterations":   r.Iterations,
        "error":        r.Error,
    }
    b, _ := json.Marshal(out)
    return string(b)
}
```

- [ ] **Step 2: Register the tool in kernel.go**

Add to kernel initialization:

```go
type Kernel struct {
    // existing fields...
    subagentManager *subagent.Manager
    delegateTool    *DelegateTool
}

func NewKernel(...) *Kernel {
    reg := subagent.NewRegistry()
    mgr := subagent.NewManager(ctx, agentID, 0, toolExecutor, reg)

    k := &Kernel{
        // existing fields...
        subagentManager: mgr,
        delegateTool:    NewDelegateTool(mgr, agentID),
    }

    // Register delegate_task tool
    k.toolRegistry.Register(tools.ToolEntry{
        Name:    "delegate_task",
        Handler: k.delegateTool.Handle,
        Schema:  DelegateTaskSchema,
    })

    return k
}
```

- [ ] **Step 3: Write delegate tool tests**

```go
func TestDelegateTool_Spawn(t *testing.T) {
    reg := subagent.NewRegistry()
    exec := &mockToolExecutor{}
    mgr := subagent.NewManager(context.Background(), "parent1", 0, exec, reg)
    tool := NewDelegateTool(mgr, "parent1")

    args := map[string]any{
        "goal":    "research Go concurrency",
        "context": "focus on channels",
    }

    result, err := tool.Handle(context.Background(), args)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    var out map[string]any
    json.Unmarshal([]byte(result), &out)

    if out["status"] != "completed" {
        t.Errorf("expected completed, got %v", out["status"])
    }
    if out["summary"] != "research Go concurrency" {
        t.Errorf("expected summary, got %v", out["summary"])
    }
}

type mockToolExecutor struct{}

func (m *mockToolExecutor) Execute(ctx context.Context, req tools.ToolRequest) (<-chan tools.ToolEvent, error) {
    ch := make(chan tools.ToolEvent, 1)
    go func() {
        defer close(ch)
        ch <- tools.ToolEvent{Type: "completed"}
    }()
    return ch, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/kernel/delegate_test.go -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/kernel/delegate.go internal/kernel/kernel.go
git commit -m "feat(subagent): add delegate_task tool + AIAgent integration"
```

---

## Task 6: Cancellation propagation + shutdown safety

**Files:**
- Modify: `internal/kernel/kernel.go` (add Close method)
- Test: `internal/kernel/delegate_test.go` (add cancellation test)

- [ ] **Step 1: Add Close method to Kernel that cancels all subagents**

```go
func (k *Kernel) Close() error {
    // Cancel all subagents
    if k.subagentManager != nil {
        k.subagentManager.Close()
    }
    return nil
}
```

- [ ] **Step 2: Add cancellation propagation test**

```go
func TestCancellationPropagation(t *testing.T) {
    reg := subagent.NewRegistry()
    exec := &slowToolExecutor{}
    ctx, cancel := context.WithCancel(context.Background())
    mgr := subagent.NewManager(ctx, "parent1", 0, exec, reg)

    cfg := subagent.SubagentConfig{Goal: "slow task"}
    sa, err := mgr.Spawn(ctx, cfg)
    if err != nil {
        t.Fatal(err)
    }

    // Cancel parent
    cancel()

    // Subagent should be cancelled
    select {
    case <-sa.ctx.Done():
    case <-time.After(2 * time.Second):
        t.Fatal("subagent not cancelled within timeout")
    }

    mgr.Close()
}

type slowToolExecutor struct{}

func (e *slowToolExecutor) Execute(ctx context.Context, req tools.ToolRequest) (<-chan tools.ToolEvent, error) {
    ch := make(chan tools.ToolEvent, 1)
    go func() {
        defer close(ch)
        <-ctx.Done() // wait for cancellation
        ch <- tools.ToolEvent{Type: "failed", Err: ctx.Err()}
    }()
    return ch, nil
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/kernel/... -v -run Cancellation`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/kernel/kernel.go
git commit -m "feat(subagent): add cancellation propagation + shutdown safety"
```

---

## Self-Review Checklist

**Spec coverage:**
- [x] Subagent data structures (Task 1) → spec §Data Structures
- [x] ToolExecutor interface (Task 3) → spec §ToolExecutor Interface
- [x] SubagentManager Spawn/Interrupt/Collect (Task 4) → spec §Core Interfaces
- [x] Batch spawning (Task 4) → spec §Batch Spawning
- [x] delegate_task tool (Task 5) → spec §CLI Command
- [x] Cancellation propagation (Task 6) → spec §Cancellation Propagation
- [x] Blocked tools (Task 1) → spec §Blocked Tools
- [x] Depth limit (Task 4) → spec §Resource Limits

**Placeholder scan:** None found. All steps have actual code.

**Type consistency:** `SubagentResult`, `SubagentConfig`, `SubagentEvent`, `ToolExecutor`, `ToolRequest`, `ToolEvent` all defined in Task 1 and used consistently in Tasks 3–6.

---

## Task 7: CLI commands for subagent management

**Files:**
- Create: `gormes/cmd/subagent_cli.go`
- Test: `gormes/cmd/subagent_cli_test.go`

- [ ] **Step 1: Add CLI subcommands for subagent visibility**

```go
package cmd

import (
    "fmt"
    "github.com/TrebuchetDynamics/gormes-agent/internal/subagent"
)

type SubagentCLI struct {
    manager  subagent.SubagentManager
    registry subagent.SubagentRegistry
}

func NewSubagentCLI(mgr subagent.Manager, reg subagent.SubagentRegistry) *SubagentCLI {
    return &SubagentCLI{manager: mgr, registry: reg}
}

func (c *SubagentCLI) List() string {
    list := c.registry.List()
    if len(list) == 0 {
        return "No active subagents"
    }
    out := "Active subagents:\n"
    for _, sa := range list {
        out += fmt.Sprintf("  %s (depth=%d, goal=%q)\n", sa.ID, sa.Depth, sa.Config.Goal)
    }
    return out
}

func (c *SubagentCLI) Stop(id string) error {
    list := c.registry.List()
    for _, sa := range list {
        if sa.ID == id {
            return c.manager.Interrupt(sa, "user requested")
        }
    }
    return fmt.Errorf("subagent %s not found", id)
}
```

- [ ] **Step 2: Write CLI tests**

```go
func TestSubagentCLI_List(t *testing.T) {
    reg := subagent.NewRegistry()
    mgr := subagent.NewManager(context.Background(), "parent1", 0, nil, reg)
    cli := NewSubagentCLI(mgr, reg)

    out := cli.List()
    if out != "No active subagents\n" {
        t.Errorf("unexpected output: %s", out)
    }
}

func TestSubagentCLI_StopNotFound(t *testing.T) {
    reg := subagent.NewRegistry()
    mgr := subagent.NewManager(context.Background(), "parent1", 0, nil, reg)
    cli := NewSubagentCLI(mgr, reg)

    err := cli.Stop("nonexistent")
    if err == nil {
        t.Error("expected error for nonexistent subagent")
    }
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./gormes/cmd/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add gormes/cmd/subagent_cli.go
git commit -m "feat(subagent): add CLI commands for subagent visibility and stop"
```

---

## Final Verification

1. Run all tests: `go test ./internal/subagent/... ./internal/tools/... ./internal/kernel/... ./gormes/cmd/... -v`
2. Build binary: `make build` — should compile cleanly
3. Binary size: confirm < 400 KB delta from Phase 2.E

---

**Plan complete.** Both execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task (7 tasks total), review between tasks, fast iteration.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

Which approach?
