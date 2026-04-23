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
	"os"
	"sort"
	"strings"
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

// ErrUnknownToolset is returned when a requested toolset has no registered tools.
var ErrUnknownToolset = errors.New("tools: unknown toolset")

// ErrUnavailableToolset is returned when every tool in a requested toolset
// fails the availability gate.
var ErrUnavailableToolset = errors.New("tools: unavailable toolset")

// ToolEntry is the registry row for one tool plus its availability metadata.
type ToolEntry struct {
	Tool        Tool
	Toolset     string
	RequiresEnv []string
	CheckFn     func() bool
}

func (e ToolEntry) name() string {
	if e.Tool == nil {
		return ""
	}
	return e.Tool.Name()
}

func (e ToolEntry) descriptor() ToolDescriptor {
	return ToolDescriptor{
		Name:        e.Tool.Name(),
		Description: e.Tool.Description(),
		Schema:      e.Tool.Schema(),
	}
}

func (e ToolEntry) availabilityError() error {
	missing := make([]string, 0, len(e.RequiresEnv))
	for _, raw := range e.RequiresEnv {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if _, ok := os.LookupEnv(name); !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("missing env: %s", strings.Join(missing, ", "))
	}
	if e.CheckFn != nil {
		available := false
		func() {
			defer func() {
				if recover() != nil {
					available = false
				}
			}()
			available = e.CheckFn()
		}()
		if !available {
			return errors.New("availability check failed")
		}
	}
	return nil
}

func (e ToolEntry) available() bool {
	return e.availabilityError() == nil
}

// Registry holds a set of named Tools. Safe for concurrent use.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]ToolEntry
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]ToolEntry)}
}

// Register adds a tool. Returns ErrDuplicate on name collision.
func (r *Registry) Register(t Tool) error {
	return r.RegisterEntry(ToolEntry{Tool: t})
}

// RegisterEntry adds a tool with toolset + availability metadata.
// Empty Toolset defaults to the tool name.
func (r *Registry) RegisterEntry(entry ToolEntry) error {
	if entry.Tool == nil {
		return errors.New("tools: nil tool")
	}
	name := entry.Tool.Name()
	if name == "" {
		return errors.New("tools: empty tool name")
	}
	if strings.TrimSpace(entry.Toolset) == "" {
		entry.Toolset = name
	} else {
		entry.Toolset = strings.TrimSpace(entry.Toolset)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicate, name)
	}
	r.tools[name] = entry
	return nil
}

// MustRegister is Register's main()-time convenience. Panics on collision.
func (r *Registry) MustRegister(t Tool) {
	if err := r.Register(t); err != nil {
		panic(err)
	}
}

// MustRegisterEntry is RegisterEntry's main()-time convenience. Panics on collision.
func (r *Registry) MustRegisterEntry(entry ToolEntry) {
	if err := r.RegisterEntry(entry); err != nil {
		panic(err)
	}
}

// Get returns the Tool for name, or (nil, false).
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.tools[name]
	if !ok {
		return nil, false
	}
	return entry.Tool, true
}

// Entry returns the metadata row for name, or (zero, false).
func (r *Registry) Entry(name string) (ToolEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.tools[name]
	return entry, ok
}

// Entries returns all registered entries, stable-sorted by tool name.
func (r *Registry) Entries() []ToolEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ToolEntry, 0, len(r.tools))
	for _, entry := range r.tools {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name() < out[j].name() })
	return out
}

// Descriptors returns the registered tools as stable-sorted (by name)
// ToolDescriptors. Deterministic ordering makes request bodies diff-friendly.
func (r *Registry) Descriptors() []ToolDescriptor {
	entries := r.Entries()
	out := make([]ToolDescriptor, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.descriptor())
	}
	return out
}

// AvailableDescriptors returns descriptors for tools that pass availability
// checks. Stable-sorted by tool name.
func (r *Registry) AvailableDescriptors() []ToolDescriptor {
	entries := r.Entries()
	out := make([]ToolDescriptor, 0, len(entries))
	for _, entry := range entries {
		if !entry.available() {
			continue
		}
		out = append(out, entry.descriptor())
	}
	return out
}

// Toolsets returns every registered toolset name, stable-sorted.
func (r *Registry) Toolsets() []string {
	entries := r.Entries()
	names := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if _, ok := seen[entry.Toolset]; ok {
			continue
		}
		seen[entry.Toolset] = struct{}{}
		names = append(names, entry.Toolset)
	}
	sort.Strings(names)
	return names
}

// AvailableToolsets returns toolsets that have at least one available tool.
func (r *Registry) AvailableToolsets() []string {
	entries := r.Entries()
	names := make([]string, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if !entry.available() {
			continue
		}
		if _, ok := seen[entry.Toolset]; ok {
			continue
		}
		seen[entry.Toolset] = struct{}{}
		names = append(names, entry.Toolset)
	}
	sort.Strings(names)
	return names
}

// CheckToolsetRequirements validates that each requested toolset exists and
// has at least one available tool.
func (r *Registry) CheckToolsetRequirements(toolsets ...string) error {
	requested := normalizeToolsetList(toolsets)
	if len(requested) == 0 {
		return nil
	}

	entries := r.Entries()
	byToolset := make(map[string][]ToolEntry, len(entries))
	for _, entry := range entries {
		byToolset[entry.Toolset] = append(byToolset[entry.Toolset], entry)
	}

	for toolset := range requested {
		group, ok := byToolset[toolset]
		if !ok {
			return fmt.Errorf("%w: %s", ErrUnknownToolset, toolset)
		}
		var reasons []string
		for _, entry := range group {
			if err := entry.availabilityError(); err == nil {
				reasons = nil
				break
			} else {
				reasons = append(reasons, err.Error())
			}
		}
		if len(reasons) == len(group) {
			sort.Strings(reasons)
			return fmt.Errorf("%w: %s (%s)", ErrUnavailableToolset, toolset, strings.Join(compactRepeats(reasons), "; "))
		}
	}
	return nil
}

// DescriptorsForToolsets returns the available tool descriptors after applying
// enable/disable toolset filtering. Unknown disabled toolsets are ignored to
// keep subtractive filters tolerant of future config drift.
func (r *Registry) DescriptorsForToolsets(enabled, disabled []string) ([]ToolDescriptor, error) {
	enabledSet := normalizeToolsetList(enabled)
	disabledSet := normalizeToolsetList(disabled)
	if len(enabledSet) > 0 {
		if err := r.CheckToolsetRequirements(enabled...); err != nil {
			return nil, err
		}
	}

	entries := r.Entries()
	out := make([]ToolDescriptor, 0, len(entries))
	for _, entry := range entries {
		if !entry.available() {
			continue
		}
		if len(enabledSet) > 0 {
			if _, ok := enabledSet[entry.Toolset]; !ok {
				continue
			}
		}
		if _, blocked := disabledSet[entry.Toolset]; blocked {
			continue
		}
		out = append(out, entry.descriptor())
	}
	return out, nil
}

func normalizeToolsetList(toolsets []string) map[string]struct{} {
	if len(toolsets) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(toolsets))
	for _, raw := range toolsets {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		out[name] = struct{}{}
	}
	return out
}

func compactRepeats(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := values[:0]
	var last string
	for i, value := range values {
		if i == 0 || value != last {
			out = append(out, value)
			last = value
		}
	}
	return out
}
