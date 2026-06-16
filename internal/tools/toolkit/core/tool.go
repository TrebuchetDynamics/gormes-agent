// Package core defines the Go-native tool surface that the Gormes kernel
// executes when the LLM emits tool_calls. Every Tool is a Go type compiled
// into the Gormes binary; the Registry is populated explicitly by main.go
// (init() is permitted for third-party packages but not used in core).
package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/plugins"
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

// OperationSpec declares the behavioral metadata for a tool. Every tool that
// implements Spec() returns its OperationSpec; tools without it report
// descriptor_missing in doctor and inherit safe defaults in the executor.
// This is the contract-first operation catalog from the Phase 5.A row.
type OperationSpec struct {
	ToolDescriptor
	Mutating   bool     // true when the tool modifies state (files, memory, etc.)
	Idempotent bool     // true when the tool can be safely retried
	PromptSafe bool     // true when the schema contains no secrets
	TrustClass []string // allowed caller roles: operator, child-agent, system
	AuditKind  string   // audit taxonomy: file, web, terminal, memory, skills
}

// Spec is an optional interface that tools implement to declare their
// OperationSpec. Tools that don't implement this get a safe default.
type Spec interface {
	Spec() OperationSpec
}

// DefaultSpec returns a conservative OperationSpec for tools that don't
// implement the Spec interface. Mutating=true, Idempotent=false, PromptSafe=true.
func DefaultSpec(name, desc string, schema json.RawMessage) OperationSpec {
	return OperationSpec{
		ToolDescriptor: ToolDescriptor{Name: name, Description: desc, Schema: schema},
		Mutating:       true,
		Idempotent:     false,
		PromptSafe:     true,
		TrustClass:     []string{"operator", "child-agent", "system"},
		AuditKind:      "tool",
	}
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
	mu                 sync.RWMutex
	tools              map[string]Tool
	pluginCapabilities []plugins.CapabilityStatus
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
	if r.tools == nil {
		r.tools = make(map[string]Tool)
	}
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

// FilterPolicy returns a registry view that exposes only tools allowed by the
// supplied allow/deny policy. Deny wins. An empty allow list means all tools
// remain eligible unless denied.
func (r *Registry) FilterPolicy(allow, deny []string) *Registry {
	if r == nil {
		return nil
	}
	allowSet := registryPolicySet(allow)
	denySet := registryPolicySet(deny)
	out := NewRegistry()
	r.mu.RLock()
	defer r.mu.RUnlock()
	for name, tool := range r.tools {
		key := registryPolicyName(name)
		if _, denied := denySet[key]; denied {
			continue
		}
		if len(allowSet) > 0 {
			if _, allowed := allowSet[key]; !allowed {
				continue
			}
		}
		out.tools[name] = tool
	}
	out.pluginCapabilities = clonePluginCapabilities(r.pluginCapabilities)
	return out
}

// RecordPluginInventory records plugin capability status rows without
// registering executable tool handlers. This keeps plugin discovery metadata
// visible while runtime execution remains disabled.
func (r *Registry) RecordPluginInventory(inventory plugins.Inventory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pluginCapabilities = clonePluginCapabilities(inventory.Capabilities)
	sortPluginCapabilities(r.pluginCapabilities)
}

// PluginCapabilities returns all recorded plugin capability rows.
func (r *Registry) PluginCapabilities() []plugins.CapabilityStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clonePluginCapabilities(r.pluginCapabilities)
}

// DisabledPluginCapabilities returns the disabled plugin capability rows.
func (r *Registry) DisabledPluginCapabilities() []plugins.CapabilityStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]plugins.CapabilityStatus, 0, len(r.pluginCapabilities))
	for _, row := range r.pluginCapabilities {
		if row.State == plugins.StateDisabled {
			out = append(out, clonePluginCapability(row))
		}
	}
	sortPluginCapabilities(out)
	return out
}

func clonePluginCapabilities(in []plugins.CapabilityStatus) []plugins.CapabilityStatus {
	out := make([]plugins.CapabilityStatus, len(in))
	for i, row := range in {
		out[i] = clonePluginCapability(row)
	}
	return out
}

func clonePluginCapability(in plugins.CapabilityStatus) plugins.CapabilityStatus {
	out := in
	out.Evidence = append([]plugins.Evidence(nil), in.Evidence...)
	return out
}

func sortPluginCapabilities(rows []plugins.CapabilityStatus) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Plugin != rows[j].Plugin {
			return rows[i].Plugin < rows[j].Plugin
		}
		if rows[i].Kind != rows[j].Kind {
			return rows[i].Kind < rows[j].Kind
		}
		return rows[i].Name < rows[j].Name
	})
}

func registryPolicySet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		if name := registryPolicyName(value); name != "" {
			out[name] = struct{}{}
		}
	}
	return out
}

func registryPolicyName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
