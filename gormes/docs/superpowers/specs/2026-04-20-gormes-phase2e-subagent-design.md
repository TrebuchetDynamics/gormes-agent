# Phase 2.E — Subagent System Spec

**Date:** 2026-04-20
**Status:** Draft — pending implementation plan
**Priority:** P0
**Upstream:** `tools/delegate_tool.py`, `run_agent.py` (AIAgent class)

---

## Goal

Implement the Gormes Subagent System — **execution isolation for parallel workstreams**. Spawn subagents that run in goroutines with their own context, resource boundaries, cancellation scopes, and failure containment. This enables reliable parallel task execution within the single binary.

---

## Why Not Separate Processes

Python's Hermes uses threads (not processes) for subagents. Gormes inherits this model via goroutines. The architectural win over Hermes is that Go gives us **deterministic context cancellation** and **typed channel communication** natively — no thread-global interrupt flags needed.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Parent Agent                                                │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  SubagentManager                                     │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │   │
│  │  │ spawn goroutine │ │ cancel     │  │ collect   │ │   │
│  │  │ → child ctx    │  │ → ctx.Done │  │ → channel │ │   │
│  │  └─────────────┘  └─────────────┘  └────────────┘ │   │
│  │                                                      │   │
│  │  children: map[string]*Subagent                      │   │
│  │  tools: ToolExecutor (interface)                     │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ goroutine-per-subagent
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Child Goroutine                                            │
│  ctx, cancel := context.WithCancel(parentCtx)                │
│                                                              │
│  child.Agent = NewAIAgent(childCtx, childConfig)            │
│  child.Agent.ToolExecutor = parent.ToolExecutor             │
│                                                              │
│  for event := range child.Events {                          │
│      parent.events <- event  // streamed back              │
│  }                                                          │
│  parent.results <- &SubagentResult{...}                    │
└─────────────────────────────────────────────────────────────┘
```

---

## Data Structures

```go
// Subagent represents a single child agent
type Subagent struct {
    ID       string
    ParentID string
    Depth    int  // 0 = parent, 1 = child, 2 = grandchild (rejected if >= 2)

    ctx    context.Context
    cancel context.CancelFunc

    Config   SubagentConfig
    Events   <-chan SubagentEvent  // streamed back to parent
    Result   *SubagentResult      // final result
    closed   bool
    mu       sync.RWMutex
}

// SubagentConfig holds per-subagent configuration
type SubagentConfig struct {
    Goal           string
    Context        string
    MaxIterations  int
    MaxMemoryMB    int
    EnabledTools   []string  // empty = all parent tools minus blocked
    Model          string    // empty = inherit parent model
    Timeout        time.Duration
}

// SubagentEvent is streamed back to parent during execution
type SubagentEvent struct {
    Type    string  // "started" | "progress" | "tool_call" | "output" | "completed" | "failed" | "interrupted"
    Message string
    ToolCall *ToolCallInfo
    Progress *ProgressInfo
}

// SubagentResult is returned when subagent finishes
type SubagentResult struct {
    ID          string
    Status      string  // "completed" | "failed" | "interrupted" | "error"
    Summary     string
    ExitReason  string
    Duration    time.Duration
    Iterations  int
    ToolCalls   []ToolCallInfo
    Error       string
}

// ToolExecutor interface — decouples subagent from execution model
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
    Type   string  // "started" | "progress" | "output" | "completed" | "failed"
    Chunk  string
    Err    error
}

// In-process executor (Phase 2.E MVP)
type InProcessToolExecutor struct {
    Registry *ToolRegistry  // from internal/tools
}
```

---

## ToolExecutor Interface

```go
// internal/tools/executor.go

// ToolExecutor executes tools on behalf of an agent.
// Subagents use this interface so execution model is swappable:
// - In-process (same binary, same goroutine pool) — MVP
// - Sidecar (separate process, JSON-RPC over stdio) — future
// - Remote (HTTP RPC) — future
type ToolExecutor interface {
    Execute(ctx context.Context, req ToolRequest) (<-chan ToolEvent, error)
}

// InProcessToolExecutor executes tools directly in the binary.
// Thread-safe: uses the same tool registry as the parent agent.
// This is the Phase 2.E MVP implementation.
type InProcessToolExecutor struct {
    registry *ToolRegistry
    mu       sync.Mutex  // serialize tool calls per subagent
}

func (e *InProcessToolExecutor) Execute(ctx context.Context, req ToolRequest) (<-chan ToolEvent, error) {
    ch := make(chan ToolEvent, 1)

    // Lookup tool in registry
    tool, ok := e.registry.Get(req.ToolName)
    if !ok {
        ch <- ToolEvent{Type: "failed", Err: fmt.Errorf("tool not found: %s", req.ToolName)}
        close(ch)
        return ch, nil
    }

    // Execute in goroutine to support streaming
    go func() {
        defer close(ch)

        ch <- ToolEvent{Type: "started"}

        // Parse input
        var input map[string]any
        if err := json.Unmarshal([]byte(req.Input), &input); err != nil {
            ch <- ToolEvent{Type: "failed", Err: fmt.Errorf("invalid input: %w", err)}
            return
        }

        // Execute tool handler
        result, err := tool.Handler(ctx, input)
        if err != nil {
            ch <- ToolEvent{Type: "failed", Err: err}
            return
        }

        // Stream output chunks
        ch <- ToolEvent{Type: "output", Chunk: result}
        ch <- ToolEvent{Type: "completed"}
    }()

    return ch, nil
}
```

---

## Core Interfaces

```go
// SubagentManager orchestrates all subagents for a parent agent
type SubagentManager interface {
    Spawn(ctx context.Context, cfg SubagentConfig) (*Subagent, error)
    SpawnBatch(ctx context.Context, cfgs []SubagentConfig, maxConcurrent int) ([]*SubagentResult, error)
    Interrupt(sa *Subagent, message string) error
    Collect(sa *Subagent) *SubagentResult
    Close() error
}

// SubagentRegistry tracks all active subagents process-wide
// (for cancellation on shutdown)
type SubagentRegistry interface {
    Register(sa *Subagent)
    Unregister(id string)
    InterruptAll(message string)
    List() []*Subagent
}
```

---

## SubagentManager Implementation

```go
type subagentManager struct {
    parentCtx  context.Context
    parentID   string
    depth      int

    children   map[string]*Subagent
    mu         sync.RWMutex

    executor   ToolExecutor
    registry   *SubagentRegistryImpl
}

const MaxDepth = 2  // parent(0) → child(1) → grandchild rejected(2)

func (m *subagentManager) Spawn(ctx context.Context, cfg SubagentConfig) (*Subagent, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Depth check
    if m.depth >= MaxDepth {
        return nil, fmt.Errorf("max subagent depth (%d) reached", MaxDepth)
    }

    // Default max iterations
    if cfg.MaxIterations <= 0 {
        cfg.MaxIterations = 50
    }

    // Create child context (cancelled when parent cancels OR when cancel called)
    childCtx, cancel := context.WithCancel(m.parentCtx)

    // Create subagent
    sa := &Subagent{
        ID:       generateID(),
        ParentID: m.parentID,
        Depth:    m.depth + 1,
        ctx:      childCtx,
        cancel:   cancel,
        Config:   cfg,
        Events:   make(<-chan SubagentEvent),
        Result:   nil,
    }

    // Spawn goroutine
    go m.runSubagent(sa)

    m.children[sa.ID] = sa
    m.registry.Register(sa)

    return sa, nil
}

func (m *subagentManager) runSubagent(sa *Subagent) {
    defer func() {
        if r := recover(); r != nil {
            sa.Result = &SubagentResult{
                ID:    sa.ID,
                Status: "error",
                Error: fmt.Sprintf("panic: %v", r),
            }
        }
        m.mu.Lock()
        delete(m.children, sa.ID)
        m.mu.Unlock()
        m.registry.Unregister(sa.ID)
    }()

    // Event channel (streamed back to parent)
    events := make(chan SubagentEvent, 10)

    // Result channel (final result)
    results := make(chan *SubagentResult, 1)

    // Run agent
    go func() {
        agent := m.newAIAgent(sa.ctx, sa.Config)
        result := agent.Run(events)
        results <- result
    }()

    // Stream events + collect result
    var result *SubagentResult
    for {
        select {
        case ev := <-events:
            // Forward to parent
            // (parent receives via sa.Events channel)
        case res := <-results:
            result = res
            sa.mu.Lock()
            sa.closed = true
            sa.Result = result
            sa.mu.Unlock()
            return
        case <-sa.ctx.Done():
            // Cancelled by parent
            result = &SubagentResult{
                ID:    sa.ID,
                Status: "interrupted",
                Error: sa.ctx.Err().Error(),
            }
            return
        }
    }
}

func (m *subagentManager) Interrupt(sa *Subagent, message string) {
    m.mu.RLock()
    child, ok := m.children[sa.ID]
    m.mu.RUnlock()
    if !ok {
        return
    }
    child.cancel()  // triggers ctx.Done() in child goroutine
}
```

---

## Batch Spawning

```go
func (m *subagentManager) SpawnBatch(ctx context.Context, cfgs []SubagentConfig, maxConcurrent int) ([]*SubagentResult, error) {
    if maxConcurrent <= 0 {
        maxConcurrent = 3
    }

    results := make([]*SubagentResult, len(cfgs))
    pending := make([]*Subagent, 0, len(cfgs))

    // Spawn up to maxConcurrent
    for i, cfg := range cfgs {
        sa, err := m.Spawn(ctx, cfg)
        if err != nil {
            results[i] = &SubagentResult{Status: "error", Error: err.Error()}
            continue
        }
        pending = append(pending, sa)
    }

    // Collect results
    collected := 0
    for len(pending) > 0 && collected < len(cfgs) {
        // Wait for first result
        // In practice: select on ctx.Done() + result channels
    }

    return results, nil
}
```

---

## Blocked Tools

The following tools are always stripped from subagents (cannot be delegated further):

```go
var BlockedTools = map[string]bool{
    "delegate_task":   true,  // prevents subagent delegation loops
    "clarify":         true,  // needs user in parent context
    "memory":          true,  // subagent memory stays isolated
    "send_message":    true,  // cross-context messaging not supported
    "execute_code":    true,  // security/isolation boundary
}
```

---

## Context Isolation

| Aspect | How |
|--------|-----|
| Conversation context | Fresh; parent passes `goal` + `context` strings only |
| Memory | Subagent uses own session; no access to parent's entity graph |
| Tools | Subset of parent tools minus BlockedTools |
| Credentials | Inherited from parent (same provider/model) |
| Config | Per-subagent `SubagentConfig` |
| Context files | `skip_context_files=true` unless parent passes them |
| Cancellation | `context.WithCancel(parentCtx)` — parent cancels child |

---

## Cancellation Propagation

```
User: /stop or Ctrl+C
         │
         ▼
parent.interrupt(message)
         │
         ├─ parent.ctx cancelled
         │
         ├─ For each child in parent.children:
         │   child.cancel()  → child.ctx.Done()
         │
         └─ For each tool goroutine in child:
             ctx.Err() checked in tool executor
```

Cancellation is **recursive**: cancelling the parent cancels all children, grandchildren, etc.

---

## Resource Limits

| Resource | Limit | Enforcement |
|----------|-------|-------------|
| Max depth | 2 (parent → child → grandchild rejected) | `if m.depth >= MaxDepth` check |
| Max concurrent | 3 (configurable) | `SpawnBatch` semaphore |
| Max iterations | 50 per subagent (configurable) | `IterationBudget` per subagent |
| Memory | Soft limit (goroutine GC hints) | `runtime.GC()` after each turn |
| Timeout | `cfg.Timeout` (default: no timeout) | `context.WithTimeout` |

---

## CLI Command

```go
// delegate_task tool schema
var DelegateTaskSchema = tools.ToolSchema{
    Name: "delegate_task",
    Description: "Delegate a task to a subagent for parallel execution. " +
        "The subagent runs with its own context and returns a structured result. " +
        "Useful for: independent parallel work, research tasks, multi-step operations.",
    Parameters: []tools.ToolParam{
        {Name: "goal", Type: "string", Required: true, Description: "Task goal for subagent"},
        {Name: "context", Type: "string", Required: false, Description: "Additional context"},
        {Name: "max_iterations", Type: "integer", Required: false, Default: 50},
        {Name: "toolsets", Type: "string", Required: false, Description: "Comma-separated toolset names"},
    },
}
```

---

## Configuration

```toml
[delegation]
enabled = true
max_depth = 2
max_concurrent_children = 3
default_max_iterations = 50
default_timeout = "1h"
permitted_depth_override = false  # disallow deeper trees via config
```

---

## Error Handling

| Error | Behavior |
|-------|----------|
| Max depth reached | Return error to agent: "Cannot nest subagents beyond depth 2" |
| Max concurrent exceeded | Return error: "Max N concurrent subagents reached" |
| Child panics | Recover, return error result, log panic |
| Tool execution fails | Return error in ToolEvent, subagent continues |
| Parent cancelled | All children cancelled via context cascade |

---

## Acceptance Criteria

1. Parent agent calls `delegate_task` with a goal and context
2. Subagent goroutine spawns with `context.WithCancel(parentCtx)`
3. Subagent runs independently, streaming events back via channel
4. Parent can interrupt via `Interrupt(sa, message)` — child receives `ctx.Done()`
5. Subagent cannot spawn grandchildren beyond depth 2
6. Subagent result includes: status, summary, duration, iterations, tool calls, error
7. Batch spawning runs up to `maxConcurrent` in parallel
8. Tool execution uses `ToolExecutor` interface — in-process now, swappable later
9. Binary size impact: < 400 KB (SubagentManager + ToolExecutor interface + channel primitives)

---

## Dependencies

- `internal/kernel` — AIAgent, context management
- `internal/tools` — ToolRegistry, ToolExecutor interface
- `internal/config` — `[delegation]` config section
- `context` (stdlib) — context.WithCancel, context.WithTimeout

---

## Phase 2.E Ledger

| Subphase | Status | Description |
|----------|--------|-------------|
| 2.E.1 — Subagent data structures | ⏳ planned | Subagent, SubagentConfig, SubagentEvent, SubagentResult |
| 2.E.2 — ToolExecutor interface | ⏳ planned | Interface definition + InProcessToolExecutor |
| 2.E.3 — SubagentManager | ⏳ planned | Spawn, Interrupt, Collect, batch support |
| 2.E.4 — SubagentRegistry | ⏳ planned | Process-wide tracking for shutdown |
| 2.E.5 — AIAgent integration | ⏳ planned | delegate_task tool, result streaming |
| 2.E.6 — Cancellation propagation | ⏳ planned | Recursive cancel on parent interrupt |
| 2.E.7 — CLI / feedback | ⏳ planned | /delegate, /subagents, /stop-subagent |

**Ship criterion:** `delegate_task(goal="research X", context="...")` spawns a goroutine that runs to completion and returns a result. Parent can interrupt mid-execution. Children cannot spawn grandchildren. All goroutines cleaned up on shutdown.
