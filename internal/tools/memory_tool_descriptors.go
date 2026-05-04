package tools

import "encoding/json"

const (
	StoreMemoryToolName       = "store_memory"
	RetrieveMemoryToolName    = "retrieve_memory"
	UpdateMemoryToolName      = "update_memory"
	SummarizeMemoriesToolName = "summarize_memories"
	ForgetMemoryToolName      = "forget_memory"
)

var memoryToolDescriptors = []ToolDescriptor{
	{
		Name:        StoreMemoryToolName,
		Description: "Persist information to agent memory. Use to remember facts, preferences, and lessons that should survive across sessions.",
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"content":{"type":"string","description":"The information to store"},
				"tags":{"type":"array","items":{"type":"string"},"description":"Tags for categorization"},
				"importance":{"type":"number","description":"Importance from 0.0 to 1.0"},
				"metadata":{"type":"object","additionalProperties":{"type":"string"},"description":"Optional metadata to persist with provenance"}
			},
			"required":["content"]
		}`),
	},
	{
		Name:        RetrieveMemoryToolName,
		Description: "Retrieve memories relevant to the given query. Returns ranked results ordered by importance and recency.",
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"query":{"type":"string","description":"Search query for memory retrieval"},
				"limit":{"type":"integer","description":"Max results; defaults to 5"}
			},
			"required":["query"]
		}`),
	},
	{
		Name:        UpdateMemoryToolName,
		Description: "Update an existing memory entry. Use when information has changed, needs correction, or its importance should be promoted or demoted.",
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"id":{"type":"string","description":"Memory entry ID to update"},
				"content":{"type":"string","description":"New content for the memory entry"},
				"importance":{"type":"number","description":"New importance from 0.0 to 1.0"}
			},
			"required":["id"]
		}`),
	},
	{
		Name:        SummarizeMemoriesToolName,
		Description: "Summarize related memories by tag or query. Compresses multiple entries into a consolidated summary.",
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"filter":{"type":"string","description":"Tag or query to filter memories for summarization"},
				"max_items":{"type":"integer","description":"Max entries to summarize; defaults to 10"}
			},
			"required":["filter"]
		}`),
	},
	{
		Name:        ForgetMemoryToolName,
		Description: "Remove a memory entry from active storage with a soft delete. Use when information is outdated or no longer relevant.",
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"id":{"type":"string","description":"Memory entry ID to forget"}
			},
			"required":["id"]
		}`),
	},
}

// MemoryToolDescriptors returns the MCP-compatible agent memory tool catalog in
// the same stable order an agent usually chains the operations.
func MemoryToolDescriptors() []ToolDescriptor {
	out := make([]ToolDescriptor, 0, len(memoryToolDescriptors))
	for _, descriptor := range memoryToolDescriptors {
		out = append(out, cloneMemoryToolDescriptor(descriptor))
	}
	return out
}

// MemoryToolOperationSpecs returns behavioral metadata for the agent-callable
// memory tools.
func MemoryToolOperationSpecs() []OperationSpec {
	descriptors := MemoryToolDescriptors()
	out := make([]OperationSpec, 0, len(descriptors))
	for _, descriptor := range descriptors {
		out = append(out, memoryToolOperationSpec(descriptor))
	}
	return out
}

// MemoryToolOperationSpec returns the behavioral metadata for one memory tool.
func MemoryToolOperationSpec(name string) (OperationSpec, bool) {
	for _, descriptor := range MemoryToolDescriptors() {
		if descriptor.Name == name {
			return memoryToolOperationSpec(descriptor), true
		}
	}
	return OperationSpec{}, false
}

func memoryToolOperationSpec(descriptor ToolDescriptor) OperationSpec {
	spec := OperationSpec{
		ToolDescriptor: descriptor,
		Mutating:       true,
		Idempotent:     false,
		PromptSafe:     true,
		TrustClass:     []string{"operator", "child-agent", "system"},
		AuditKind:      "memory",
	}
	switch descriptor.Name {
	case RetrieveMemoryToolName, SummarizeMemoriesToolName:
		spec.Mutating = false
		spec.Idempotent = true
	case ForgetMemoryToolName:
		spec.Idempotent = true
	}
	return spec
}

func cloneMemoryToolDescriptor(descriptor ToolDescriptor) ToolDescriptor {
	descriptor.Schema = append(json.RawMessage(nil), descriptor.Schema...)
	return descriptor
}
