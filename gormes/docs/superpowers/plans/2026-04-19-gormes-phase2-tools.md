# Gormes Phase 2.A — Tool Registry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the Gormes kernel the ability to execute Go-native tools in response to LLM `tool_calls`, per spec `2026-04-19-gormes-phase2-tools-design.md` (commit `053f247b`). Ship the `internal/tools` package, extend `hermes` and `kernel` for tool-call plumbing, and land the "Scientific Handshake" red test: a MockTool that mimics a FeCIM Hysteresis response, proving the SSE → tool-execution → history-append → response-finalisation path works before any real FeCIM code is linked.

**Architecture:** No new processes, no new ports. `internal/tools` is an in-process `Tool` interface + `Registry`. Kernel `runTurn` gains an outer **tool loop** wrapping the existing Phase-1.5 retry loop; when the LLM emits `finish_reason: "tool_calls"`, the kernel dispatches via the Registry, appends tool results to `ChatRequest.Messages`, and issues a fresh stream. Phase-1.5 invariants (replace-latest mailbox, Route-B reconnect) are preserved and test-enforced with tool execution in play.

**Tech Stack:** Go 1.22+, stdlib `encoding/json`/`context`/`time`/`sync`, existing `hermes.Client`, `kernel.RetryBudget`, `telemetry.Snapshot`, `store.NoopStore`. No new dependencies.

---

## Prerequisites

- Phase 1 + Phase 1.5 + Route B all shipped (latest kernel commit `3ed9a6f2` or later).
- Working tree clean or at least isolated from `internal/tools/`, `internal/hermes/`, `internal/kernel/` paths.
- `go.mod` at `go 1.22` with `toolchain go1.26.1`.

## File Structure Map

```
gormes/
├── internal/
│   ├── tools/
│   │   ├── tool.go                    # NEW — Tool interface + ToolDescriptor + Registry + errors
│   │   ├── tool_test.go               # NEW — registry + descriptor tests
│   │   ├── builtin.go                 # NEW — EchoTool + NowTool + RandIntTool
│   │   ├── builtin_test.go            # NEW
│   │   ├── mock.go                    # NEW — MockTool test double
│   │   ├── mock_test.go               # NEW — compile-time interface check
│   │   └── fecim/
│   │       ├── fecim.go               # NEW — shared schema constants + types
│   │       ├── hysteresis.go          # NEW — HysteresisTool stub
│   │       ├── hysteresis_test.go
│   │       ├── crossbar.go            # NEW — CrossbarTool stub
│   │       └── crossbar_test.go
│   ├── hermes/
│   │   ├── client.go                  # MODIFY — ChatRequest.Tools, Message.ToolCalls/ToolCallID/Name, ToolDescriptor JSON shape
│   │   ├── stream.go                  # MODIFY — accumulate tool-call deltas → Event.ToolCalls
│   │   └── stream_tools_test.go       # NEW
│   └── kernel/
│       ├── kernel.go                  # MODIFY — Config.Tools/MaxToolIterations/MaxToolDuration; runTurn tool loop
│       ├── toolexec.go                # NEW — executeToolCalls helper with recover+timeout
│       ├── toolexec_test.go           # NEW
│       ├── tools_invariants_test.go   # NEW — stall + Route-B preservation tests
│       └── tools_test.go              # NEW — Scientific Handshake red test (t.Skip'd)
```

No changes to `config`, `store`, `telemetry`, `tui`, `pybridge`, `pkg/gormes`, or `cmd/gormes` in this plan. `pkg/gormes` re-exports can be added in a follow-up once Tool is stable.

---

## Task 1: Tool Interface + Registry + errors

**Files:**
- Create: `gormes/internal/tools/tool.go`
- Create: `gormes/internal/tools/tool_test.go`

- [ ] **Step 1:** Write failing tests. Create `gormes/internal/tools/tool_test.go`:

```go
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type stubTool struct {
	name, desc string
	schema     json.RawMessage
	timeout    time.Duration
}

func (s *stubTool) Name() string             { return s.name }
func (s *stubTool) Description() string      { return s.desc }
func (s *stubTool) Schema() json.RawMessage  { return s.schema }
func (s *stubTool) Timeout() time.Duration   { return s.timeout }
func (s *stubTool) Execute(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
	return json.RawMessage(`{"ok":true}`), nil
}

func TestRegistry_RegisterDuplicateReturnsError(t *testing.T) {
	r := NewRegistry()
	a := &stubTool{name: "a", schema: json.RawMessage(`{}`)}
	if err := r.Register(a); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := r.Register(a); !errors.Is(err, ErrDuplicate) {
		t.Errorf("second Register = %v, want ErrDuplicate", err)
	}
}

func TestRegistry_MustRegister_PanicsOnDuplicate(t *testing.T) {
	r := NewRegistry()
	a := &stubTool{name: "a", schema: json.RawMessage(`{}`)}
	r.MustRegister(a)

	defer func() {
		if recover() == nil {
			t.Error("MustRegister should panic on duplicate")
		}
	}()
	r.MustRegister(a)
}

func TestRegistry_GetUnknown_ReturnsFalse(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get("missing")
	if ok {
		t.Error("Get of missing tool should return false")
	}
}

func TestRegistry_DescriptorsSorted(t *testing.T) {
	r := NewRegistry()
	r.MustRegister(&stubTool{name: "zulu", desc: "z", schema: json.RawMessage(`{}`)})
	r.MustRegister(&stubTool{name: "alpha", desc: "a", schema: json.RawMessage(`{}`)})
	r.MustRegister(&stubTool{name: "mike", desc: "m", schema: json.RawMessage(`{}`)})

	ds := r.Descriptors()
	if len(ds) != 3 {
		t.Fatalf("len = %d, want 3", len(ds))
	}
	if ds[0].Name != "alpha" || ds[1].Name != "mike" || ds[2].Name != "zulu" {
		t.Errorf("Descriptors not sorted: %v", []string{ds[0].Name, ds[1].Name, ds[2].Name})
	}
}

func TestToolDescriptor_MarshalJSON_WrapsAsFunction(t *testing.T) {
	d := ToolDescriptor{
		Name:        "echo",
		Description: "return the input",
		Schema:      json.RawMessage(`{"type":"object"}`),
	}
	out, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	want := `{"type":"function","function":{"name":"echo","description":"return the input","parameters":{"type":"object"}}}`
	if got != want {
		t.Errorf("marshal = %s\nwant   = %s", got, want)
	}
}
```

- [ ] **Step 2:** Run — expect FAIL (package doesn't compile).

```bash
cd gormes
go test ./internal/tools/...
```

- [ ] **Step 3:** Implement `gormes/internal/tools/tool.go`:

```go
// Package tools defines the Go-native tool surface that the Gormes kernel
// executes when the LLM emits tool_calls. Every Tool is a Go type compiled
// into the Gormes binary; the Registry is populated explicitly by main.go
// (init() is permitted for third-party packages but not used in core).
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Tool is the contract every Go-native tool satisfies. See spec §5.1.
type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Timeout() time.Duration
	Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
}

// ToolDescriptor is the serialisable form sent to the LLM in ChatRequest.Tools.
// JSON shape matches OpenAI's tool-definition wrapper:
//
//	{"type":"function","function":{"name":"...","description":"...","parameters":{...}}}
type ToolDescriptor struct {
	Name        string
	Description string
	Schema      json.RawMessage
}

// MarshalJSON wraps the descriptor in the OpenAI {"type":"function",...} envelope.
func (d ToolDescriptor) MarshalJSON() ([]byte, error) {
	inner := struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	}{Name: d.Name, Description: d.Description, Parameters: d.Schema}
	wrap := struct {
		Type     string `json:"type"`
		Function any    `json:"function"`
	}{Type: "function", Function: inner}
	return json.Marshal(wrap)
}

// ErrDuplicate is returned by Register when a tool name is already taken.
var ErrDuplicate = errors.New("tools: duplicate tool name")

// ErrUnknownTool is returned when a caller asks for a name that's not registered.
var ErrUnknownTool = errors.New("tools: unknown tool name")

// Registry holds a set of named Tools. Safe for concurrent use.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool. Returns ErrDuplicate on name collision.
func (r *Registry) Register(t Tool) error {
	if t == nil {
		return errors.New("tools: nil tool")
	}
	name := t.Name()
	if name == "" {
		return errors.New("tools: empty tool name")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicate, name)
	}
	r.tools[name] = t
	return nil
}

// MustRegister is Register's main()-time convenience. Panics on collision.
func (r *Registry) MustRegister(t Tool) {
	if err := r.Register(t); err != nil {
		panic(err)
	}
}

// Get returns the Tool for name, or (nil, false).
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// Descriptors returns the registered tools as stable-sorted (by name)
// ToolDescriptors. Deterministic ordering makes request bodies diff-friendly.
func (r *Registry) Descriptors() []ToolDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ToolDescriptor, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, ToolDescriptor{
			Name:        t.Name(),
			Description: t.Description(),
			Schema:      t.Schema(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
```

- [ ] **Step 4:** Run — expect PASS.

```bash
go test ./internal/tools/... -v
go vet ./internal/tools/...
```

- [ ] **Step 5:** Commit.

```bash
cd ..
git add gormes/internal/tools/tool.go gormes/internal/tools/tool_test.go
git commit -m "$(cat <<'EOF'
feat(gormes/tools): Tool interface + Registry + OpenAI descriptor shape

In-process, no magic. Registry is a thread-safe map populated
explicitly by main.go (init() permitted for third-party tools but
not used in core — keeps tests able to build a minimal registry).

ToolDescriptor.MarshalJSON wraps the (name, description, schema)
triple in OpenAI's {"type":"function","function":{...}} envelope,
so ChatRequest.Tools serialises straight to the wire.

Descriptors() sorts by name so request bodies diff cleanly in tests
and logs.

Part of spec §5 (Phase 2.A Tool Registry).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Built-in Tools (Echo, Now, RandInt)

**Files:**
- Create: `gormes/internal/tools/builtin.go`
- Create: `gormes/internal/tools/builtin_test.go`

- [ ] **Step 1:** Write failing tests. Create `gormes/internal/tools/builtin_test.go`:

```go
package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEcho_RoundTrip(t *testing.T) {
	e := &EchoTool{}
	// compile-time interface check
	var _ Tool = e

	out, err := e.Execute(context.Background(), json.RawMessage(`{"text":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"hello"`) {
		t.Errorf("echo = %s, want hello", out)
	}
}

func TestEcho_EmptyArgs_Error(t *testing.T) {
	e := &EchoTool{}
	_, err := e.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Error("echo with missing text should error")
	}
}

func TestNow_ReturnsBothFields(t *testing.T) {
	n := &NowTool{}
	var _ Tool = n

	before := time.Now().Unix()
	out, err := n.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now().Unix()

	var payload struct {
		Unix int64  `json:"unix"`
		ISO  string `json:"iso"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Unix < before || payload.Unix > after {
		t.Errorf("unix = %d, want between %d and %d", payload.Unix, before, after)
	}
	if payload.ISO == "" {
		t.Error("iso is empty")
	}
}

func TestRandInt_WithinBounds(t *testing.T) {
	r := &RandIntTool{}
	var _ Tool = r

	out, err := r.Execute(context.Background(), json.RawMessage(`{"min":10,"max":20}`))
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Value int `json:"value"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Value < 10 || payload.Value > 20 {
		t.Errorf("value = %d, want in [10,20]", payload.Value)
	}
}

func TestRandInt_InvertedBounds_Error(t *testing.T) {
	r := &RandIntTool{}
	_, err := r.Execute(context.Background(), json.RawMessage(`{"min":20,"max":10}`))
	if err == nil {
		t.Error("rand_int with min>max should error")
	}
}

func TestBuiltin_DescriptorsValidJSON(t *testing.T) {
	for _, tool := range []Tool{&EchoTool{}, &NowTool{}, &RandIntTool{}} {
		var any map[string]any
		if err := json.Unmarshal(tool.Schema(), &any); err != nil {
			t.Errorf("%s schema invalid JSON: %v", tool.Name(), err)
		}
	}
}
```

- [ ] **Step 2:** Run — expect FAIL.

- [ ] **Step 3:** Implement `gormes/internal/tools/builtin.go`:

```go
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

// EchoTool round-trips its input. Proof-of-life for the Tool plumbing.
type EchoTool struct{}

func (*EchoTool) Name() string        { return "echo" }
func (*EchoTool) Description() string { return "Echo the provided text back. Useful for testing tool-call plumbing." }
func (*EchoTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string","description":"text to echo"}},"required":["text"]}`)
}
func (*EchoTool) Timeout() time.Duration { return 0 }

func (*EchoTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("echo: invalid args: %w", err)
	}
	if in.Text == "" {
		return nil, errors.New("echo: 'text' is required and must be non-empty")
	}
	out := struct {
		Text string `json:"text"`
	}{Text: in.Text}
	return json.Marshal(out)
}

// NowTool returns the current time in two formats.
type NowTool struct{}

func (*NowTool) Name() string        { return "now" }
func (*NowTool) Description() string { return "Return the current server time as unix seconds and ISO-8601 UTC." }
func (*NowTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"required":[]}`)
}
func (*NowTool) Timeout() time.Duration { return 0 }

func (*NowTool) Execute(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
	now := time.Now().UTC()
	out := struct {
		Unix int64  `json:"unix"`
		ISO  string `json:"iso"`
	}{Unix: now.Unix(), ISO: now.Format(time.RFC3339)}
	return json.Marshal(out)
}

// RandIntTool returns a uniformly-random integer in [min, max].
type RandIntTool struct{}

func (*RandIntTool) Name() string        { return "rand_int" }
func (*RandIntTool) Description() string { return "Return a uniformly random integer in [min, max]." }
func (*RandIntTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"min":{"type":"integer"},"max":{"type":"integer"}},"required":["min","max"]}`)
}
func (*RandIntTool) Timeout() time.Duration { return 0 }

func (*RandIntTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Min int `json:"min"`
		Max int `json:"max"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("rand_int: invalid args: %w", err)
	}
	if in.Min > in.Max {
		return nil, fmt.Errorf("rand_int: min (%d) must be <= max (%d)", in.Min, in.Max)
	}
	value := in.Min + rand.Intn(in.Max-in.Min+1)
	out := struct {
		Value int `json:"value"`
	}{Value: value}
	return json.Marshal(out)
}
```

- [ ] **Step 4:** Run — expect PASS.

```bash
cd gormes
go test ./internal/tools/... -v
go vet ./internal/tools/...
```

- [ ] **Step 5:** Commit.

```bash
cd ..
git add gormes/internal/tools/builtin.go gormes/internal/tools/builtin_test.go
git commit -m "$(cat <<'EOF'
feat(gormes/tools): built-in Echo / Now / RandInt tools

Proof-of-life: three Tool implementers validating the interface
end-to-end. Echo round-trips a string (arg validation), Now reports
server time (zero-arg tools work), RandInt exercises bounded
argument validation (min <= max).

Every tool carries a complete JSON-Schema in Schema() so the LLM
gets a usable tool description; schemas are valid JSON (test-enforced).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: MockTool test double

**Files:**
- Create: `gormes/internal/tools/mock.go`
- Create: `gormes/internal/tools/mock_test.go`

- [ ] **Step 1:** Create `gormes/internal/tools/mock.go`:

```go
package tools

import (
	"context"
	"encoding/json"
	"time"
)

// MockTool is a test double. Every field is independently configurable so
// tests can script happy paths, slow executions, panics, or ctx-cancel
// scenarios. Not used in production code.
type MockTool struct {
	NameStr    string
	Desc       string
	SchemaJSON json.RawMessage
	TimeoutD   time.Duration
	// ExecuteFn drives the behaviour. If nil, Execute returns `{"ok":true}`.
	ExecuteFn func(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
}

// Compile-time interface check.
var _ Tool = (*MockTool)(nil)

func (m *MockTool) Name() string {
	if m.NameStr == "" {
		return "mock"
	}
	return m.NameStr
}

func (m *MockTool) Description() string {
	if m.Desc == "" {
		return "mock tool for testing"
	}
	return m.Desc
}

func (m *MockTool) Schema() json.RawMessage {
	if len(m.SchemaJSON) == 0 {
		return json.RawMessage(`{"type":"object","properties":{},"required":[]}`)
	}
	return m.SchemaJSON
}

func (m *MockTool) Timeout() time.Duration { return m.TimeoutD }

func (m *MockTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	if m.ExecuteFn != nil {
		return m.ExecuteFn(ctx, args)
	}
	return json.RawMessage(`{"ok":true}`), nil
}
```

- [ ] **Step 2:** Create `gormes/internal/tools/mock_test.go`:

```go
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestMockTool_DefaultExecute(t *testing.T) {
	m := &MockTool{}
	out, err := m.Execute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"ok":true}` {
		t.Errorf("default Execute = %s", out)
	}
}

func TestMockTool_CustomExecute(t *testing.T) {
	m := &MockTool{
		ExecuteFn: func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
			return nil, errors.New("boom")
		},
	}
	_, err := m.Execute(context.Background(), nil)
	if err == nil || err.Error() != "boom" {
		t.Errorf("err = %v, want boom", err)
	}
}

func TestMockTool_DefaultsForMissingFields(t *testing.T) {
	m := &MockTool{}
	if m.Name() != "mock" {
		t.Errorf("default name = %q", m.Name())
	}
	if m.Description() == "" {
		t.Error("default description should be non-empty")
	}
	var schema map[string]any
	if err := json.Unmarshal(m.Schema(), &schema); err != nil {
		t.Errorf("default schema invalid: %v", err)
	}
}
```

- [ ] **Step 3:** Run — PASS.

```bash
cd gormes
go test ./internal/tools/... -v
```

- [ ] **Step 4:** Commit.

```bash
cd ..
git add gormes/internal/tools/mock.go gormes/internal/tools/mock_test.go
git commit -m "$(cat <<'EOF'
feat(gormes/tools): MockTool test double

Configurable Tool implementation for kernel tests. ExecuteFn drives
behaviour — tests script happy paths, slow runs, panics, or
ctx-cancel scenarios without allocating goroutines.

Compile-time interface check (var _ Tool = (*MockTool)(nil))
guarantees the double never drifts from the real contract.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: ~~FeCIM tools~~ — SUPERSEDED (SKIP)

**Superseded 2026-04-19** per user clarification: FeCIM and Gormes are separate entities. Gormes ships no domain-specific tools; external Go modules register themselves as `tools.Tool` implementers in consumer forks. See spec §9 (revised).

**Do not execute Task 4.** Proceed directly from Task 3 to Task 5. The entire original Task 4 content below is retained solely for historical reference.

<details>
<summary>Original Task 4 content (historical — do not implement)</summary>

## Task 4 (HISTORICAL — do not execute): FeCIM tools — HysteresisTool + CrossbarTool stubs

**Files:**
- Create: `gormes/internal/tools/fecim/fecim.go`
- Create: `gormes/internal/tools/fecim/hysteresis.go`
- Create: `gormes/internal/tools/fecim/hysteresis_test.go`
- Create: `gormes/internal/tools/fecim/crossbar.go`
- Create: `gormes/internal/tools/fecim/crossbar_test.go`

- [ ] **Step 1:** Create `gormes/internal/tools/fecim/fecim.go` (shared):

```go
// Package fecim exposes FeCIM (Ferroelectric Computer-In-Memory) operations
// as Gormes Tools. Phase 2.A ships stubs returning canned results; real
// FeCIM Go packages will be imported when the integration landing question
// (spec §15 Appendix B) is answered.
package fecim

// schemaHysteresis is the JSON-Schema for HysteresisTool arguments.
const schemaHysteresis = `{
  "type": "object",
  "properties": {
    "material":        {"type": "string", "description": "ferroelectric stack id (e.g. 'PZT-4')"},
    "field_amplitude": {"type": "number", "description": "peak electric field in MV/m"},
    "temperature_K":   {"type": "number", "default": 300},
    "cycles":          {"type": "integer", "default": 10, "minimum": 1, "maximum": 1000}
  },
  "required": ["material", "field_amplitude"]
}`

// schemaCrossbar is the JSON-Schema for CrossbarTool arguments.
const schemaCrossbar = `{
  "type": "object",
  "properties": {
    "array_size":     {"type": "array", "items": {"type": "integer"}, "minItems": 2, "maxItems": 2, "description": "[rows, cols]"},
    "cell_stack":     {"type": "string"},
    "read_voltage_V": {"type": "number", "default": 0.2},
    "precision":      {"type": "integer", "enum": [1, 2, 4, 8], "default": 4}
  },
  "required": ["array_size", "cell_stack"]
}`
```

- [ ] **Step 2:** Create `gormes/internal/tools/fecim/hysteresis.go`:

```go
package fecim

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

// HysteresisTool models ferroelectric polarization under cyclic electric field.
// Phase-2.A stub — returns canned characteristics. Real implementation calls
// into the FeCIM Go package once wired.
type HysteresisTool struct{}

var _ tools.Tool = (*HysteresisTool)(nil)

func (*HysteresisTool) Name() string { return "fecim_hysteresis" }
func (*HysteresisTool) Description() string {
	return "Compute ferroelectric hysteresis metrics (coercive field, remnant polarization, fatigue) for a given material, cyclic field amplitude, temperature, and cycle count."
}
func (*HysteresisTool) Schema() json.RawMessage  { return json.RawMessage(schemaHysteresis) }
func (*HysteresisTool) Timeout() time.Duration   { return 30 * time.Second }

type hysteresisArgs struct {
	Material       string  `json:"material"`
	FieldAmplitude float64 `json:"field_amplitude"`
	TemperatureK   float64 `json:"temperature_K,omitempty"`
	Cycles         int     `json:"cycles,omitempty"`
}

type hysteresisResult struct {
	CoerciveFieldMVPerM float64 `json:"coercive_field_MV_per_m"`
	RemnantPuCPerCm2    float64 `json:"remnant_P_uC_per_cm2"`
	FatigueFactor       float64 `json:"fatigue_factor"`
}

func (*HysteresisTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in hysteresisArgs
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, err
	}
	if in.Material == "" {
		return nil, errors.New("fecim_hysteresis: 'material' is required")
	}
	if in.FieldAmplitude <= 0 {
		return nil, errors.New("fecim_hysteresis: 'field_amplitude' must be > 0")
	}
	// Stubbed canned result. Real physics lives in the imported FeCIM package.
	out := hysteresisResult{
		CoerciveFieldMVPerM: 1.2,
		RemnantPuCPerCm2:    25.4,
		FatigueFactor:       0.97,
	}
	return json.Marshal(out)
}
```

- [ ] **Step 3:** Create `gormes/internal/tools/fecim/hysteresis_test.go`:

```go
package fecim

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

func TestHysteresis_CompilesAsTool(t *testing.T) {
	var _ tools.Tool = (*HysteresisTool)(nil)
}

func TestHysteresis_HappyPath(t *testing.T) {
	h := &HysteresisTool{}
	out, err := h.Execute(context.Background(), json.RawMessage(`{"material":"PZT-4","field_amplitude":5.0}`))
	if err != nil {
		t.Fatal(err)
	}
	var got hysteresisResult
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got.CoerciveFieldMVPerM <= 0 || got.RemnantPuCPerCm2 <= 0 {
		t.Errorf("invalid result: %+v", got)
	}
}

func TestHysteresis_MissingMaterial_Error(t *testing.T) {
	h := &HysteresisTool{}
	_, err := h.Execute(context.Background(), json.RawMessage(`{"field_amplitude":5.0}`))
	if err == nil {
		t.Error("missing material should error")
	}
}

func TestHysteresis_InvalidField_Error(t *testing.T) {
	h := &HysteresisTool{}
	_, err := h.Execute(context.Background(), json.RawMessage(`{"material":"PZT-4","field_amplitude":0}`))
	if err == nil {
		t.Error("field_amplitude=0 should error")
	}
}

func TestHysteresis_SchemaValidJSON(t *testing.T) {
	h := &HysteresisTool{}
	var any map[string]any
	if err := json.Unmarshal(h.Schema(), &any); err != nil {
		t.Errorf("schema invalid JSON: %v", err)
	}
}
```

- [ ] **Step 4:** Create `gormes/internal/tools/fecim/crossbar.go`:

```go
package fecim

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

// CrossbarTool queries a ferroelectric crossbar array's read/write
// characteristics. Phase-2.A stub — canned benchmark values.
type CrossbarTool struct{}

var _ tools.Tool = (*CrossbarTool)(nil)

func (*CrossbarTool) Name() string { return "fecim_crossbar" }
func (*CrossbarTool) Description() string {
	return "Simulate a ferroelectric crossbar array and report throughput, signal-to-noise ratio, and sneak-path error for an MVM (matrix-vector multiplication) workload."
}
func (*CrossbarTool) Schema() json.RawMessage  { return json.RawMessage(schemaCrossbar) }
func (*CrossbarTool) Timeout() time.Duration   { return 30 * time.Second }

type crossbarArgs struct {
	ArraySize    []int   `json:"array_size"`
	CellStack    string  `json:"cell_stack"`
	ReadVoltageV float64 `json:"read_voltage_V,omitempty"`
	Precision    int     `json:"precision,omitempty"`
}

type crossbarResult struct {
	ThroughputTOPS    float64 `json:"throughput_TOPS"`
	CellSNRdB         float64 `json:"cell_snr_dB"`
	SneakPathErrorPct float64 `json:"sneak_path_error_pct"`
}

func (*CrossbarTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in crossbarArgs
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, err
	}
	if len(in.ArraySize) != 2 {
		return nil, errors.New("fecim_crossbar: 'array_size' must be [rows, cols]")
	}
	if in.ArraySize[0] <= 0 || in.ArraySize[1] <= 0 {
		return nil, errors.New("fecim_crossbar: array dimensions must be > 0")
	}
	if in.CellStack == "" {
		return nil, errors.New("fecim_crossbar: 'cell_stack' is required")
	}
	out := crossbarResult{
		ThroughputTOPS:    42.0,
		CellSNRdB:         28.5,
		SneakPathErrorPct: 0.3,
	}
	return json.Marshal(out)
}
```

- [ ] **Step 5:** Create `gormes/internal/tools/fecim/crossbar_test.go`:

```go
package fecim

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

func TestCrossbar_CompilesAsTool(t *testing.T) {
	var _ tools.Tool = (*CrossbarTool)(nil)
}

func TestCrossbar_HappyPath(t *testing.T) {
	c := &CrossbarTool{}
	out, err := c.Execute(context.Background(), json.RawMessage(`{"array_size":[64,64],"cell_stack":"HZO-8nm"}`))
	if err != nil {
		t.Fatal(err)
	}
	var got crossbarResult
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got.ThroughputTOPS <= 0 {
		t.Errorf("invalid result: %+v", got)
	}
}

func TestCrossbar_InvalidArraySize_Error(t *testing.T) {
	c := &CrossbarTool{}
	_, err := c.Execute(context.Background(), json.RawMessage(`{"array_size":[64],"cell_stack":"HZO-8nm"}`))
	if err == nil {
		t.Error("1-element array_size should error")
	}
}
```

- [ ] **Step 6:** Run + commit.

```bash
cd gormes
go test ./internal/tools/... -v
go vet ./internal/tools/...
cd ..
git add gormes/internal/tools/fecim/
git commit -m "$(cat <<'EOF'
feat(gormes/tools/fecim): HysteresisTool + CrossbarTool stubs

Two narrow, single-purpose FeCIM Tools, not one mega-tool — matches
OpenAI best practice and lets the LLM reason about "which physical
property do I measure?" as a real semantic choice.

Schemas are indicative: material/field/temperature/cycles for
Hysteresis; array_size/cell_stack/read_voltage/precision for Crossbar.

Execute() returns canned JSON today; the interface is ready to accept
the real FeCIM Go package once F-1..F-5 questions from the Phase 2+
brainstorm memo are answered. No other file in the repo changes at
that point.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

</details>

---

## Task 5: Hermes — ChatRequest/Event/Message tool-call plumbing

**Files:**
- Modify: `gormes/internal/hermes/client.go` (extend types only; no behaviour change in this task)

- [ ] **Step 1:** Read `gormes/internal/hermes/client.go`. Locate `ChatRequest`, `Message`, `Event`.

- [ ] **Step 2:** Extend `Message`. Find:

```go
type Message struct {
	Role    string
	Content string
}
```

Replace with:

```go
type Message struct {
	Role       string     // "system" | "user" | "assistant" | "tool"
	Content    string
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`  // set only on assistant messages that requested tools
	ToolCallID string     `json:"tool_call_id,omitempty"` // set only on "tool" role messages replying to a call
	Name       string     `json:"name,omitempty"`         // set only on "tool" role messages; echoes the tool name
}

// ToolCall is one function-call request made by the LLM.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}
```

- [ ] **Step 3:** Extend `ChatRequest`. Find:

```go
type ChatRequest struct {
	Model     string
	Messages  []Message
	SessionID string
	Stream    bool
}
```

Replace with:

```go
type ChatRequest struct {
	Model     string
	Messages  []Message
	SessionID string
	Stream    bool
	Tools     []ToolDescriptor // omitempty at wire time
}

// ToolDescriptor mirrors tools.ToolDescriptor so hermes stays
// dependency-free of the tools package. Serialised shape is
// OpenAI's {"type":"function","function":{...}} wrapper — the
// kernel populates Tools by calling tools.Registry.Descriptors()
// and converting them.
type ToolDescriptor struct {
	Name        string
	Description string
	Schema      json.RawMessage
}

// MarshalJSON for ToolDescriptor wraps in OpenAI's function envelope.
func (d ToolDescriptor) MarshalJSON() ([]byte, error) {
	inner := struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	}{Name: d.Name, Description: d.Description, Parameters: d.Schema}
	wrap := struct {
		Type     string `json:"type"`
		Function any    `json:"function"`
	}{Type: "function", Function: inner}
	return json.Marshal(wrap)
}
```

Add `"encoding/json"` to the imports if not already present.

- [ ] **Step 4:** Extend `Event`. Find:

```go
type Event struct {
	Kind         EventKind
	Token        string
	Reasoning    string
	FinishReason string
	TokensIn     int
	TokensOut    int
	Raw          json.RawMessage
}
```

Add `ToolCalls []ToolCall` field:

```go
type Event struct {
	Kind         EventKind
	Token        string
	Reasoning    string
	FinishReason string
	TokensIn     int
	TokensOut    int
	ToolCalls    []ToolCall      // non-empty only on EventDone with FinishReason=="tool_calls"
	Raw          json.RawMessage
}
```

- [ ] **Step 5:** Verify build.

```bash
cd gormes
go build ./...
go vet ./...
go test ./... -timeout 90s
```

All existing tests still PASS — the extensions are additive.

- [ ] **Step 6:** Commit.

```bash
cd ..
git add gormes/internal/hermes/client.go
git commit -m "$(cat <<'EOF'
feat(gormes/hermes): ChatRequest/Event/Message tool-call fields

Additive: Message gains ToolCalls/ToolCallID/Name, ChatRequest gains
Tools []ToolDescriptor, Event gains ToolCalls []ToolCall. Every new
field carries omitempty/JSON tags matching OpenAI's tool-calling
wire format.

ToolDescriptor is declared here (duplicated from tools.ToolDescriptor
so hermes stays free of the tools-package dependency). The kernel
bridges by copying field values when building ChatRequest.Tools.

No behaviour change in this commit — SSE parser accumulator and
kernel tool-loop land in Task 6 and Task 7.

All Phase 1 + 1.5 tests still pass under -race.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Hermes SSE — accumulate tool-call deltas

**Files:**
- Modify: `gormes/internal/hermes/stream.go`
- Create: `gormes/internal/hermes/stream_tools_test.go`

- [ ] **Step 1:** Write the failing test first. Create `gormes/internal/hermes/stream_tools_test.go`:

```go
package hermes

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sseToolCallsFixture = `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"echo","arguments":""}}]}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"tex"}}]}}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"t\":\"hi\"}"}}]}}]}

data: {"choices":[{"finish_reason":"tool_calls"}]}

data: [DONE]

`

func TestStream_ToolCallDeltasAccumulate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, sseToolCallsFixture)
		bw.Flush()
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "")
	s, err := c.OpenStream(context.Background(), ChatRequest{
		Model:    "x",
		Messages: []Message{{Role: "user", Content: "echo hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var final Event
	for {
		e, err := s.Recv(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if e.Kind == EventDone {
			final = e
			break
		}
	}

	if final.FinishReason != "tool_calls" {
		t.Fatalf("FinishReason = %q, want tool_calls", final.FinishReason)
	}
	if len(final.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d, want 1", len(final.ToolCalls))
	}
	tc := final.ToolCalls[0]
	if tc.ID != "call_abc" {
		t.Errorf("ID = %q", tc.ID)
	}
	if tc.Name != "echo" {
		t.Errorf("Name = %q", tc.Name)
	}
	if !strings.Contains(string(tc.Arguments), `"hi"`) {
		t.Errorf("Arguments = %s, want to contain \"hi\"", tc.Arguments)
	}
}
```

- [ ] **Step 2:** Run — expect FAIL (stream parser doesn't handle tool-calls yet).

- [ ] **Step 3:** Read `gormes/internal/hermes/stream.go`. Locate `orChunkChoice` / `orChunk` / `orChunkDelta` and the `Recv` method.

- [ ] **Step 4:** Extend the internal chunk types. Find:

```go
type orChunkDelta struct {
	Content   string `json:"content"`
	Reasoning string `json:"reasoning"`
}

type orChunkChoice struct {
	Delta        orChunkDelta `json:"delta"`
	FinishReason string       `json:"finish_reason"`
}
```

Replace with:

```go
type orChunkDelta struct {
	Content   string                `json:"content"`
	Reasoning string                `json:"reasoning"`
	ToolCalls []orChunkToolCall     `json:"tool_calls,omitempty"`
}

type orChunkToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	} `json:"function"`
}

type orChunkChoice struct {
	Delta        orChunkDelta `json:"delta"`
	FinishReason string       `json:"finish_reason"`
}
```

- [ ] **Step 5:** Add a per-stream accumulator field to `chatStream`. Find:

```go
type chatStream struct {
	body      io.ReadCloser
	sse       *sseReader
	sessionID string
	closed    bool
	mu        sync.Mutex
}
```

Replace with:

```go
type chatStream struct {
	body      io.ReadCloser
	sse       *sseReader
	sessionID string
	closed    bool
	mu        sync.Mutex

	// Pending tool-call accumulator, keyed by the upstream index field.
	// Populated across tool-call delta chunks; flushed on finish_reason=="tool_calls".
	pendingCalls map[int]*pendingToolCall
}

type pendingToolCall struct {
	id        string
	name      string
	arguments strings.Builder
}
```

Add `"strings"` to the stream.go imports if missing.

In `newChatStream`, initialise the map:

```go
func newChatStream(body io.ReadCloser, sessionID string) *chatStream {
	return &chatStream{
		body:         body,
		sse:          newSSEReader(body),
		sessionID:    sessionID,
		pendingCalls: make(map[int]*pendingToolCall),
	}
}
```

- [ ] **Step 6:** Extend `Recv` to accumulate tool-call deltas and flush on finish_reason. Find the `switch c.FinishReason != ""` block and the tool-call-absent logic. Replace the event-classification block inside `Recv` (the code after `c := chunk.Choices[0]`) with:

```go
		c := chunk.Choices[0]

		// Accumulate tool-call deltas across chunks.
		for _, tc := range c.Delta.ToolCalls {
			p, ok := s.pendingCalls[tc.Index]
			if !ok {
				p = &pendingToolCall{}
				s.pendingCalls[tc.Index] = p
			}
			if tc.ID != "" {
				p.id = tc.ID
			}
			if tc.Function.Name != "" {
				p.name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				p.arguments.WriteString(tc.Function.Arguments)
			}
		}

		if c.Delta.Reasoning != "" {
			return Event{Kind: EventReasoning, Reasoning: c.Delta.Reasoning, Raw: json.RawMessage(f.data)}, nil
		}
		if c.Delta.Content != "" {
			return Event{Kind: EventToken, Token: c.Delta.Content, Raw: json.RawMessage(f.data)}, nil
		}
		if c.FinishReason != "" {
			ev := Event{Kind: EventDone, FinishReason: c.FinishReason, Raw: json.RawMessage(f.data)}
			if chunk.Usage != nil {
				ev.TokensIn = chunk.Usage.PromptTokens
				ev.TokensOut = chunk.Usage.CompletionTokens
			}
			if c.FinishReason == "tool_calls" && len(s.pendingCalls) > 0 {
				ev.ToolCalls = flushPending(s.pendingCalls)
				s.pendingCalls = make(map[int]*pendingToolCall) // reset for possible reuse
			}
			return ev, nil
		}
```

Add helper at the bottom of `stream.go`:

```go
// flushPending converts the accumulator map into a sorted, finalised ToolCall slice.
func flushPending(m map[int]*pendingToolCall) []ToolCall {
	indexes := make([]int, 0, len(m))
	for idx := range m {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	out := make([]ToolCall, 0, len(indexes))
	for _, idx := range indexes {
		p := m[idx]
		out = append(out, ToolCall{
			ID:        p.id,
			Name:      p.name,
			Arguments: json.RawMessage(p.arguments.String()),
		})
	}
	return out
}
```

Add `"sort"` to the imports.

- [ ] **Step 7:** Run tests.

```bash
cd gormes
go test -race ./internal/hermes/... -timeout 60s -v
go vet ./internal/hermes/...
```

`TestStream_ToolCallDeltasAccumulate` passes. All existing hermes tests still pass.

- [ ] **Step 8:** Commit.

```bash
cd ..
git add gormes/internal/hermes/stream.go gormes/internal/hermes/stream_tools_test.go
git commit -m "$(cat <<'EOF'
feat(gormes/hermes): accumulate tool-call SSE deltas

OpenAI streams tool_calls as partial deltas keyed by index. The
stream parser accumulates per-index (id comes first, arguments
arrive piecewise) into a per-stream pendingCalls map. When
finish_reason=="tool_calls" arrives, the accumulator is flushed
into Event.ToolCalls in index order and the map is reset.

Malformed tool-call deltas are silently skipped along with the
chunk — matches the Phase-1 "skip bad frames, keep stream alive"
policy.

All Phase 1 + 1.5 hermes tests still pass under -race; new
TestStream_ToolCallDeltasAccumulate covers the happy path with
a 3-chunk split-argument fixture.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Kernel — executeToolCalls helper

**Files:**
- Create: `gormes/internal/kernel/toolexec.go`
- Create: `gormes/internal/kernel/toolexec_test.go`

- [ ] **Step 1:** Create `gormes/internal/kernel/toolexec.go`:

```go
package kernel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

// toolResult is the internal per-call output feeding back into the next
// ChatRequest as a role=tool Message.
type toolResult struct {
	ID      string
	Name    string
	Content string // JSON string — errors are JSON-encoded {"error":"..."}
}

// executeToolCalls runs each tool call sequentially with per-call timeout
// and panic recovery. Honours runCtx cancellation between calls. Returns
// results in the same order as calls.
func (k *Kernel) executeToolCalls(runCtx context.Context, calls []hermes.ToolCall) []toolResult {
	results := make([]toolResult, len(calls))
	for i, call := range calls {
		select {
		case <-runCtx.Done():
			results[i] = toolResult{
				ID: call.ID, Name: call.Name,
				Content: `{"error":"cancelled before execution"}`,
			}
			continue
		default:
		}

		if k.cfg.Tools == nil {
			results[i] = toolResult{
				ID: call.ID, Name: call.Name,
				Content: `{"error":"no tool registry configured"}`,
			}
			continue
		}

		tool, ok := k.cfg.Tools.Get(call.Name)
		if !ok {
			results[i] = toolResult{
				ID: call.ID, Name: call.Name,
				Content: fmt.Sprintf(`{"error":"unknown tool: %q"}`, call.Name),
			}
			k.addSoul("tool unknown: " + call.Name)
			continue
		}

		timeout := tool.Timeout()
		if timeout <= 0 {
			timeout = k.cfg.MaxToolDuration
		}
		if timeout <= 0 {
			timeout = 30 * 1e9 // 30 seconds in nanoseconds, in case MaxToolDuration is zero-valued
		}

		callCtx, cancel := context.WithTimeout(runCtx, timeout)

		k.addSoul("tool: " + call.Name)
		k.emitFrame("executing tool: " + call.Name)

		payload, err := safeExecute(callCtx, tool, call.Arguments)
		cancel()

		if err != nil {
			results[i] = toolResult{
				ID: call.ID, Name: call.Name,
				Content: fmt.Sprintf(`{"error":%q}`, err.Error()),
			}
			k.addSoul("tool error: " + call.Name + ": " + err.Error())
			continue
		}
		results[i] = toolResult{ID: call.ID, Name: call.Name, Content: string(payload)}
		k.addSoul("tool done: " + call.Name)
	}
	return results
}

// safeExecute wraps Tool.Execute with panic recovery so a misbehaving tool
// cannot crash the kernel goroutine.
func safeExecute(ctx context.Context, t tools.Tool, args json.RawMessage) (result json.RawMessage, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tool panicked: %v", r)
			result = nil
		}
	}()
	return t.Execute(ctx, args)
}
```

- [ ] **Step 2:** Extend `Config` in `gormes/internal/kernel/kernel.go`. Find:

```go
type Config struct {
	Model     string
	Endpoint  string
	Admission Admission
}
```

Replace with:

```go
type Config struct {
	Model             string
	Endpoint          string
	Admission         Admission
	Tools             *tools.Registry // nil means tool_calls are treated as fatal
	MaxToolIterations int             // default 10
	MaxToolDuration   time.Duration   // default 30s
}
```

Add `"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"` to kernel.go's imports if absent. `time` is already imported.

- [ ] **Step 3:** Create `gormes/internal/kernel/toolexec_test.go`:

```go
package kernel

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

func newKernelWithRegistry(t *testing.T, reg *tools.Registry) *Kernel {
	t.Helper()
	return New(Config{
		Model:             "hermes-agent",
		Endpoint:          "http://mock",
		Admission:         Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             reg,
		MaxToolIterations: 10,
		MaxToolDuration:   30 * time.Second,
	}, hermes.NewMockClient(), store.NewNoop(), telemetry.New(), nil)
}

func TestExecuteToolCalls_UnknownToolReturnsErrorResult(t *testing.T) {
	reg := tools.NewRegistry()
	k := newKernelWithRegistry(t, reg)
	res := k.executeToolCalls(context.Background(), []hermes.ToolCall{
		{ID: "c1", Name: "not_registered", Arguments: json.RawMessage(`{}`)},
	})
	if len(res) != 1 {
		t.Fatalf("len = %d, want 1", len(res))
	}
	if !strings.Contains(res[0].Content, "unknown tool") {
		t.Errorf("content = %q, want to contain 'unknown tool'", res[0].Content)
	}
}

func TestExecuteToolCalls_PanicRecovered(t *testing.T) {
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{
		NameStr: "boom",
		ExecuteFn: func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
			panic("synthetic panic")
		},
	})
	k := newKernelWithRegistry(t, reg)
	res := k.executeToolCalls(context.Background(), []hermes.ToolCall{
		{ID: "c1", Name: "boom", Arguments: json.RawMessage(`{}`)},
	})
	if !strings.Contains(res[0].Content, "panicked") {
		t.Errorf("content = %q, want to contain 'panicked'", res[0].Content)
	}
}

func TestExecuteToolCalls_TimeoutHonoured(t *testing.T) {
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{
		NameStr:  "slow",
		TimeoutD: 20 * time.Millisecond,
		ExecuteFn: func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(1 * time.Second):
				return json.RawMessage(`{"ok":true}`), nil
			}
		},
	})
	k := newKernelWithRegistry(t, reg)
	start := time.Now()
	res := k.executeToolCalls(context.Background(), []hermes.ToolCall{
		{ID: "c1", Name: "slow", Arguments: json.RawMessage(`{}`)},
	})
	elapsed := time.Since(start)
	if elapsed > 200*time.Millisecond {
		t.Errorf("elapsed = %v, want ~20ms (the tool's timeout)", elapsed)
	}
	if !strings.Contains(res[0].Content, "deadline exceeded") && !strings.Contains(res[0].Content, "context") {
		t.Errorf("content = %q, want a context-deadline error", res[0].Content)
	}
}

func TestExecuteToolCalls_CancelBetweenCalls(t *testing.T) {
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{NameStr: "a"})
	reg.MustRegister(&tools.MockTool{NameStr: "b"})
	k := newKernelWithRegistry(t, reg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := k.executeToolCalls(ctx, []hermes.ToolCall{
		{ID: "1", Name: "a", Arguments: json.RawMessage(`{}`)},
		{ID: "2", Name: "b", Arguments: json.RawMessage(`{}`)},
	})
	for i := range res {
		if !strings.Contains(res[i].Content, "cancelled") {
			t.Errorf("res[%d] content = %q, want to mention cancelled", i, res[i].Content)
		}
	}
}
```

- [ ] **Step 4:** Run.

```bash
cd gormes
go test -race ./internal/kernel/... -run "TestExecuteToolCalls" -v
go vet ./internal/kernel/...
```

All four PASS.

- [ ] **Step 5:** Commit.

```bash
cd ..
git add gormes/internal/kernel/kernel.go gormes/internal/kernel/toolexec.go gormes/internal/kernel/toolexec_test.go
git commit -m "$(cat <<'EOF'
feat(gormes/kernel): executeToolCalls helper with recover + timeout

Adds Config.Tools / MaxToolIterations / MaxToolDuration fields and
a sequential tool-dispatch helper. Each call gets its own
ctx.WithTimeout cascaded from runCtx; panics are recovered and
formatted as {"error":"tool panicked: ..."} results rather than
crashing the kernel. Unknown tool names produce error-shaped results
(the LLM can re-plan).

Honours runCtx cancellation both BEFORE and DURING calls.
Pre-cancel fast-path returns immediately; mid-call cancellation is
enforced by the callCtx deadline.

No kernel-orchestration wiring yet — runTurn integration is Task 8.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Kernel — tool loop wrapping retry loop in runTurn

**Files:**
- Modify: `gormes/internal/kernel/kernel.go`

This task restructures `runTurn` to wrap the Route-B retry loop in an outer tool loop. The flow: admission → store → loop{ retryLoop → if tool_calls: execute + append → continue; else: break } → finalise.

- [ ] **Step 1:** Read the current `runTurn` in `gormes/internal/kernel/kernel.go`. The current structure is:
  - admission, store persist, history append, phase=Connecting, prov.LogPOSTSent
  - `retryLoop:` (the full Route-B block from commit `364c8b2a`)
  - finalisation

- [ ] **Step 2:** Wrap the retry loop in a tool loop. Replace the retry-loop-and-finalisation section with the following. The admission/store/history/emitFrame("connecting") section BEFORE the retry loop stays unchanged.

Find the block starting with `retryBudget := NewRetryBudget()` and ending with `k.emitFrame("idle")` (the very last line of runTurn before `addSoul`). Replace with:

```go
	// 4. Tool loop — wraps the Route-B retry loop. On finish_reason=="tool_calls"
	// we execute the tools in-process and issue a follow-up stream with the
	// tool results appended to the message history. Capped at MaxToolIterations
	// to prevent runaway agent loops.
	request := hermes.ChatRequest{
		Model:     k.cfg.Model,
		SessionID: k.sessionID,
		Stream:    true,
		Messages:  []hermes.Message{{Role: "user", Content: text}},
	}
	if k.cfg.Tools != nil {
		// Translate tools.ToolDescriptor → hermes.ToolDescriptor.
		descs := k.cfg.Tools.Descriptors()
		wireDescs := make([]hermes.ToolDescriptor, len(descs))
		for i, d := range descs {
			wireDescs[i] = hermes.ToolDescriptor{Name: d.Name, Description: d.Description, Schema: d.Schema}
		}
		request.Tools = wireDescs
	}
	maxIter := k.cfg.MaxToolIterations
	if maxIter <= 0 {
		maxIter = 10
	}

	var (
		cancelled          bool
		fatalErr           error
		finalDelta         hermes.Event
		gotFinal           bool
		latestSessionID    string
		toolIteration      = 0
	)

	start := time.Now()
	k.tm.StartTurn()

toolLoop:
	for {
		// Reset retry budget for each tool round. Retries are for network drops,
		// not for multi-round agent reasoning.
		retryBudget := NewRetryBudget()
		var replaceOnNextToken bool

	retryLoop:
		for {
			runCtx, cancelRun := context.WithCancel(ctx)

			stream, err := k.client.OpenStream(runCtx, request)
			if err != nil {
				cancelRun()
				if hermes.Classify(err) == hermes.ClassRetryable && !retryBudget.Exhausted() {
					k.phase = PhaseReconnecting
					k.lastError = "reconnecting: " + err.Error()
					k.emitFrame("reconnecting")
					delay := retryBudget.NextDelay()
					if werr := Wait(ctx, delay); werr != nil {
						cancelled = true
						break toolLoop
					}
					replaceOnNextToken = true
					continue retryLoop
				}
				prov.ErrorClass = hermes.Classify(err).String()
				prov.ErrorText = err.Error()
				prov.LogError(k.log)
				k.phase = PhaseFailed
				k.lastError = err.Error()
				k.emitFrame("open stream failed")
				return
			}

			k.phase = PhaseStreaming
			k.emitFrame("streaming")

			outcome := k.streamInner(ctx, runCtx, cancelRun, stream, &finalDelta, &gotFinal, &fatalErr, &cancelled, &replaceOnNextToken)
			_ = stream.Close()
			if sid := stream.SessionID(); sid != "" {
				latestSessionID = sid
			}
			cancelRun()

			switch outcome {
			case streamOutcomeDone:
				break retryLoop
			case streamOutcomeCancelled:
				break toolLoop
			case streamOutcomeFatal:
				break toolLoop
			case streamOutcomeRetryable:
				if retryBudget.Exhausted() {
					k.phase = PhaseFailed
					k.lastError = "reconnect budget exhausted"
					k.emitFrame("reconnect budget exhausted")
					return
				}
				k.phase = PhaseReconnecting
				k.emitFrame("reconnecting")
				delay := retryBudget.NextDelay()
				if werr := Wait(ctx, delay); werr != nil {
					cancelled = true
					break toolLoop
				}
				replaceOnNextToken = true
				continue retryLoop
			}
		}

		// retryLoop exited cleanly (EventDone received). Inspect finish_reason.
		if !gotFinal {
			fatalErr = fmt.Errorf("stream closed without finish_reason")
			break toolLoop
		}

		if finalDelta.FinishReason != "tool_calls" {
			// Normal end of turn — real LLM answer. Exit the tool loop.
			break toolLoop
		}

		// tool_calls round. Execute the tools and append results to request.
		toolIteration++
		if toolIteration > maxIter {
			k.phase = PhaseFailed
			k.lastError = fmt.Sprintf("tool iteration limit exceeded (%d)", maxIter)
			k.emitFrame(k.lastError)
			return
		}

		runCtx, cancelRun := context.WithCancel(ctx)
		results := k.executeToolCalls(runCtx, finalDelta.ToolCalls)
		cancelRun()

		// Append the assistant's tool-requesting message plus one tool-result
		// message per call. The draft so far is captured in the assistant message.
		assistantMsg := hermes.Message{
			Role:      "assistant",
			Content:   k.draft,
			ToolCalls: finalDelta.ToolCalls,
		}
		request.Messages = append(request.Messages, assistantMsg)
		for _, r := range results {
			request.Messages = append(request.Messages, hermes.Message{
				Role:       "tool",
				ToolCallID: r.ID,
				Name:       r.Name,
				Content:    r.Content,
			})
		}

		// Clear draft between tool iterations — the next LLM response is a
		// fresh continuation; the assistant message we appended captures what
		// we had so far.
		k.draft = ""
		gotFinal = false
		finalDelta = hermes.Event{}
		k.emitFrame("executing tools")
		// Loop: next iteration issues a new stream with the updated request.
	}

	// 5. Finalisation (unchanged shape from Route-B).
	latency := time.Since(start)
	k.tm.FinishTurn(latency)
	prov.LatencyMs = int(latency / time.Millisecond)

	if fatalErr != nil {
		prov.ErrorClass = hermes.Classify(fatalErr).String()
		prov.ErrorText = fatalErr.Error()
		prov.LogError(k.log)
		k.phase = PhaseFailed
		k.lastError = fatalErr.Error()
		k.emitFrame("stream error")
		return
	}

	if gotFinal {
		prov.FinishReason = finalDelta.FinishReason
		prov.TokensIn = finalDelta.TokensIn
		prov.TokensOut = finalDelta.TokensOut
		if finalDelta.TokensIn > 0 {
			k.tm.SetTokensIn(finalDelta.TokensIn)
		}
	}

	if latestSessionID != "" {
		k.sessionID = latestSessionID
		prov.ServerSessionID = latestSessionID
		prov.LogSSEStart(k.log)
	}

	if cancelled {
		k.phase = PhaseCancelling
		k.emitFrame("cancelled")
	} else if k.draft != "" {
		k.history = append(k.history, hermes.Message{Role: "assistant", Content: k.draft})
	}

	prov.LogDone(k.log)
	k.phase = PhaseIdle
	k.emitFrame("idle")
}
```

- [ ] **Step 3:** Run the ENTIRE kernel test suite. All existing Phase-1 + Phase-1.5 tests must still pass.

```bash
cd gormes
go test -race ./internal/kernel/... -timeout 120s -v
```

If ANY existing test fails, STOP. Inspect the failure carefully — the outer tool loop is additive (when `Tools == nil` and no tool_calls arrive, the tool loop executes exactly one iteration and falls through). The most common regression is the retry-budget being shared across tool iterations; the code above resets it per iteration.

- [ ] **Step 4:** Commit.

```bash
cd ..
git add gormes/internal/kernel/kernel.go
git commit -m "$(cat <<'EOF'
feat(gormes/kernel): tool loop wraps retry loop in runTurn

Outer tool loop on each LLM round; inner retry loop per stream
attempt. On finish_reason=="tool_calls":
  1. execute tools sequentially via executeToolCalls
  2. append assistant (with tool_calls) + one tool-role message per
     result to request.Messages
  3. clear k.draft (fresh continuation)
  4. issue a new stream
MaxToolIterations=10 prevents runaway agent loops.

Retry budget resets each tool iteration — retries are for network
drops, not for multi-round agent reasoning. A drop mid-second-round
still has a full 1/2/4/8/16s reconnect budget.

Draft semantics across loops:
  - Route-B reconnect: PRESERVED (visual continuity)
  - Tool iteration:    CLEARED (new response stream)

All Phase-1 + Phase-1.5 tests still pass; invariant-preservation
tests land in Task 9.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Kernel — invariant-preservation + Tool-Call Handshake tests

**Revised 2026-04-19:** FeCIM is external to Gormes (see spec §9 revision). The handshake test now uses the built-in `EchoTool` — same end-to-end proof that the Kernel↔Tool contract works, no domain entanglement.

**Files:**
- Create: `gormes/internal/kernel/tools_invariants_test.go`
- Create: `gormes/internal/kernel/tools_test.go`

Two distinct tests in this task:

1. **Invariant-preservation** — prove a tool call doesn't break Phase-1.5 replace-latest or Route-B. Ships PASSING.
2. **Tool-Call Handshake** — end-to-end test using the built-in `EchoTool` from Task 2. Proves Gormes can call its own tools and resume the conversation perfectly.

- [ ] **Step 1:** Create `gormes/internal/kernel/tools_invariants_test.go`:

```go
package kernel

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

// TestToolLoop_DoesNotBreakReplaceLatestMailbox: with a 500ms-slow
// MockTool in the middle of a turn, a stalled render consumer must
// not cause the kernel to block. The capacity-1 replace-latest mailbox
// must keep dropping stale frames even while the tool is running.
func TestToolLoop_DoesNotBreakReplaceLatestMailbox(t *testing.T) {
	mc := hermes.NewMockClient()
	// Round 1: tool_call to "slow".
	mc.Script([]hermes.Event{
		{
			Kind: hermes.EventDone, FinishReason: "tool_calls",
			ToolCalls: []hermes.ToolCall{
				{ID: "c1", Name: "slow", Arguments: json.RawMessage(`{}`)},
			},
		},
	}, "sess-stall-tool")
	// Round 2: streaming answer.
	events := []hermes.Event{}
	for i := 0; i < 100; i++ {
		events = append(events, hermes.Event{Kind: hermes.EventToken, Token: "z", TokensOut: i + 1})
	}
	events = append(events, hermes.Event{Kind: hermes.EventDone, FinishReason: "stop"})
	mc.Script(events, "sess-stall-tool")

	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{
		NameStr: "slow",
		ExecuteFn: func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
			select {
			case <-time.After(500 * time.Millisecond):
				return json.RawMessage(`{"done":true}`), nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	})

	k := New(Config{
		Model:             "hermes-agent",
		Endpoint:          "http://mock",
		Admission:         Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             reg,
		MaxToolIterations: 10,
		MaxToolDuration:   5 * time.Second,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go k.Run(ctx)

	initial := <-k.Render()
	if initial.Phase != PhaseIdle {
		t.Fatalf("initial = %v, want PhaseIdle", initial.Phase)
	}
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "call slow"}); err != nil {
		t.Fatal(err)
	}

	// STALL for 2s across the tool execution. Kernel must not deadlock.
	time.Sleep(2 * time.Second)

	// Peek the current frame — must be the latest state, not stale.
	var peeked RenderFrame
	select {
	case peeked = <-k.Render():
	default:
		t.Fatal("no frame after 2s stall — kernel may have deadlocked during tool execution")
	}
	// Acceptable states: Idle with the full 100-z assistant, or Streaming
	// mid-round-2 with draft accumulating.
	wantAssistant := strings.Repeat("z", 100)
	ok := false
	if peeked.Phase == PhaseIdle {
		if a := lastAssistantMessage(peeked.History); a != nil && a.Content == wantAssistant {
			ok = true
		}
	}
	if peeked.Phase == PhaseStreaming && strings.HasPrefix(wantAssistant, peeked.DraftText) && peeked.DraftText != "" {
		ok = true
	}
	if !ok {
		t.Errorf("stale peek: phase=%v draftLen=%d historyLen=%d",
			peeked.Phase, len(peeked.DraftText), len(peeked.History))
	}

	// Drain remainder.
	go func() {
		for range k.Render() {
		}
	}()
}
```

- [ ] **Step 2:** Create `gormes/internal/kernel/tools_test.go` — the Tool-Call Handshake. This test uses the built-in `EchoTool` to prove Gormes can call its own tools and resume the conversation perfectly.

```go
package kernel

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

// TestKernel_ToolCallHandshake_Echo is the "Tool-Call Handshake" test:
// proves the full SSE → tool-execution → history-append → response-
// finalisation path works end-to-end using Gormes's own built-in EchoTool.
//
// This test pins the Kernel↔Tool contract: as long as a Tool returns JSON
// matching its schema, the Kernel relays it into the next LLM round and
// finalises the assistant message in history cleanly. External domain
// tools (scientific simulators, business-logic wrappers) inherit this
// contract by satisfying the same Tool interface.
func TestKernel_ToolCallHandshake_Echo(t *testing.T) {
	mc := hermes.NewMockClient()

	// Round 1: LLM requests the built-in `echo` tool with a deterministic arg.
	mc.Script([]hermes.Event{
		{
			Kind: hermes.EventDone, FinishReason: "tool_calls",
			ToolCalls: []hermes.ToolCall{
				{
					ID:        "call_echo_1",
					Name:      "echo",
					Arguments: json.RawMessage(`{"text":"GoCo factory online"}`),
				},
			},
		},
	}, "sess-echo")

	// Round 2: LLM's final answer referencing the echoed text.
	finalAnswer := "Tool said: GoCo factory online."
	events := []hermes.Event{}
	for _, ch := range finalAnswer {
		events = append(events, hermes.Event{Kind: hermes.EventToken, Token: string(ch), TokensOut: 1})
	}
	events = append(events, hermes.Event{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 50, TokensOut: len(finalAnswer)})
	mc.Script(events, "sess-echo")

	// Register Gormes's own built-in EchoTool — no external/domain tools involved.
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})

	k := New(Config{
		Model:             "hermes-agent",
		Endpoint:          "http://mock",
		Admission:         Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             reg,
		MaxToolIterations: 10,
		MaxToolDuration:   5 * time.Second,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go k.Run(ctx)

	<-k.Render() // initial idle
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "echo 'GoCo factory online'"}); err != nil {
		t.Fatal(err)
	}

	// Wait for final idle frame with the assistant answer in history.
	final := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		if f.Phase != PhaseIdle {
			return false
		}
		a := lastAssistantMessage(f.History)
		return a != nil && a.Content == finalAnswer
	}, 5*time.Second)

	// Assert the LAST assistant message is the round-2 response (not a hybrid).
	a := lastAssistantMessage(final.History)
	if a == nil || a.Content != finalAnswer {
		t.Fatalf("final assistant content = %q, want %q", a.Content, finalAnswer)
	}

	// Sanity: the assistant content references the echoed text, which could
	// only be true if the tool-result message made it into round 2's prompt.
	if !strings.Contains(a.Content, "GoCo factory online") {
		t.Errorf("final answer doesn't reference the echoed payload: %q", a.Content)
	}
}
```

Note: `waitForFrameMatching` and `lastAssistantMessage` are already defined in `kernel_test.go` and `stall_test.go` respectively.

- [ ] **Step 3:** Run both tests.

```bash
cd gormes
go test -race ./internal/kernel/... -run "TestToolLoop_DoesNotBreakReplaceLatestMailbox|TestKernel_FeCIMHysteresisHandshake" -v -timeout 60s
```

Both PASS. If `TestKernel_ToolCallHandshake_Echo` fails, investigate — most likely cause is that the kernel's tool loop isn't advancing to round 2, which is an implementation defect in Task 8.

- [ ] **Step 4:** If a Route-B-with-tool test is additionally desired, add one more test here using the `stableProxy` + `fiveTokenHandler` helpers from `reconnect_helpers_test.go`. **Optional**: the mid-stream drop path is already heavily tested in `reconnect_test.go`; the tool-loop orthogonally tested in step 1–2 is sufficient for Phase-2.A. Skip step 4 unless implementing the additional test is quick.

- [ ] **Step 5:** Full-repo sweep.

```bash
go test -race ./... -timeout 120s
go vet ./...
```

All PASS. No DATA RACE. `vet` clean.

- [ ] **Step 6:** Commit.

```bash
cd ..
git add gormes/internal/kernel/tools_invariants_test.go gormes/internal/kernel/tools_test.go
git commit -m "$(cat <<'EOF'
test(gormes/kernel): Phase-1.5 invariants + Tool-Call Handshake (EchoTool)

Two tests lock the Phase-2.A guarantees:

1. TestToolLoop_DoesNotBreakReplaceLatestMailbox — a 500ms-slow
   MockTool sits in the middle of a two-round turn while a stalled
   render consumer holds the mailbox. Kernel must not deadlock;
   the peeked frame after the stall must be latest state, never
   stale mid-stream.

2. TestKernel_ToolCallHandshake_Echo — the Tool-Call Handshake.
   Round 1: LLM requests the built-in "echo" tool with deterministic
   args ({"text":"GoCo factory online"}). Round 2: LLM streams a
   natural-language answer quoting the echoed text. Asserts:
     - the final assistant history message is the round-2 response
     - the answer references the echoed payload (proving the
       tool-result made it into round 2's prompt)
   Pins the Kernel↔Tool contract: any Tool returning JSON matching
   its schema is relayed into the next LLM round and the turn
   finalises cleanly. External domain tools inherit this contract
   by satisfying the same interface; Gormes ships no such tools.

Phase-2.A is now fully test-covered. Domain-specific tools (scientific,
business) live in separate repos and register themselves in consumer
forks (spec §9).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Appendix A: Self-Review

**Spec coverage:**
- §5 Tool interface + Registry + descriptor → Task 1
- §6 Built-in tools (Echo/Now/RandInt) → Task 2
- §5.2 MockTool test double → Task 3
- §9 FeCIM Hysteresis + Crossbar → Task 4
- §7.1 ChatRequest.Tools, §7.2 Message tool-call fields, §7.3 Event.ToolCalls → Task 5
- §7.4 SSE delta accumulator → Task 6
- §8.2 executeToolCalls + §8.3 Config extensions → Task 7
- §8.1 runTurn tool loop → Task 8
- §11.4 invariant-preservation tests → Task 9 Step 1
- §11.2 red/green end-to-end test, reframed to Scientific Handshake → Task 9 Step 2

**Placeholder scan:** no TBD / TODO / "similar to Task N" instances. Task 9 Step 4 notes an optional extra test that the implementer may skip; that's a scoping decision, not a placeholder.

**Type consistency:** `tools.ToolDescriptor` vs. `hermes.ToolDescriptor` is an intentional duplication (Task 5 justifies it). Kernel bridges the two in Task 8 step 2. `hermes.ToolCall` is used consistently across stream parsing (Task 6), kernel tool loop (Task 8), and tests (Task 9). `Tool` + `Registry` + `MockTool` method signatures match across Tasks 1, 3, and 7/8/9.

**Scope check:** one cohesive Phase-2.A plan; 9 tasks produce one working integration.

---

## Execution Handoff

Plan complete and saved to `gormes/docs/superpowers/plans/2026-04-19-gormes-phase2-tools.md`. Per user directive, proceeding immediately with Task 1 + Task 2 (in-process tool registry + built-in Echo/Now/RandInt) via fresh subagent dispatches. Halt for user review after Task 2 lands.
