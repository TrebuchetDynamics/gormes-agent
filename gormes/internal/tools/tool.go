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
// JSON shape matches OpenAI's tool-definition wrapper.
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
