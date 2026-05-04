package tools

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMemoryToolDescriptorsExposeMCPCompatibleAgentMemoryCatalog(t *testing.T) {
	descriptors := MemoryToolDescriptors()
	wantNames := []string{
		StoreMemoryToolName,
		RetrieveMemoryToolName,
		UpdateMemoryToolName,
		SummarizeMemoriesToolName,
		ForgetMemoryToolName,
	}
	var gotNames []string
	for _, descriptor := range descriptors {
		gotNames = append(gotNames, descriptor.Name)
		assertMemoryToolDescriptorEnvelope(t, descriptor)
	}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("descriptor names = %#v, want %#v", gotNames, wantNames)
	}

	assertMemoryToolSchema(t, descriptors, StoreMemoryToolName, []string{"content"}, []string{"content", "tags", "importance", "metadata"})
	assertMemoryToolSchema(t, descriptors, RetrieveMemoryToolName, []string{"query"}, []string{"query", "limit"})
	assertMemoryToolSchema(t, descriptors, UpdateMemoryToolName, []string{"id"}, []string{"id", "content", "importance"})
	assertMemoryToolSchema(t, descriptors, SummarizeMemoriesToolName, []string{"filter"}, []string{"filter", "max_items"})
	assertMemoryToolSchema(t, descriptors, ForgetMemoryToolName, []string{"id"}, []string{"id"})

	specs := MemoryToolOperationSpecs()
	if len(specs) != len(wantNames) {
		t.Fatalf("operation specs = %d, want %d", len(specs), len(wantNames))
	}
	expected := map[string]struct {
		mutating   bool
		idempotent bool
	}{
		StoreMemoryToolName:       {mutating: true, idempotent: false},
		RetrieveMemoryToolName:    {mutating: false, idempotent: true},
		UpdateMemoryToolName:      {mutating: true, idempotent: false},
		SummarizeMemoriesToolName: {mutating: false, idempotent: true},
		ForgetMemoryToolName:      {mutating: true, idempotent: true},
	}
	for _, name := range wantNames {
		spec, ok := MemoryToolOperationSpec(name)
		if !ok {
			t.Fatalf("%s operation spec missing", name)
		}
		if spec.AuditKind != "memory" || !spec.PromptSafe {
			t.Fatalf("%s spec = %+v, want prompt-safe memory audit spec", name, spec)
		}
		if !stringSliceContains(spec.TrustClass, "operator") || !stringSliceContains(spec.TrustClass, "system") {
			t.Fatalf("%s trust class = %#v, want operator and system callers", name, spec.TrustClass)
		}
		want := expected[name]
		if spec.Mutating != want.mutating || spec.Idempotent != want.idempotent {
			t.Fatalf("%s mutating/idempotent = %v/%v, want %v/%v", name, spec.Mutating, spec.Idempotent, want.mutating, want.idempotent)
		}
	}
	if _, ok := MemoryToolOperationSpec("purge_memory"); ok {
		t.Fatal("purge_memory should not be in the normal memory tool catalog")
	}
}

func assertMemoryToolDescriptorEnvelope(t *testing.T, descriptor ToolDescriptor) {
	t.Helper()
	if descriptor.Name == "" || descriptor.Description == "" || len(descriptor.Schema) == 0 {
		t.Fatalf("descriptor = %+v, want name/description/schema", descriptor)
	}
	raw, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatalf("marshal descriptor %s: %v", descriptor.Name, err)
	}
	var envelope struct {
		Type     string `json:"type"`
		Function struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
		} `json:"function"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("decode descriptor envelope %s: %v", descriptor.Name, err)
	}
	if envelope.Type != "function" || envelope.Function.Name != descriptor.Name || envelope.Function.Description != descriptor.Description {
		t.Fatalf("descriptor envelope = %s", raw)
	}
	if len(envelope.Function.Parameters) == 0 {
		t.Fatalf("descriptor envelope %s missing parameters", descriptor.Name)
	}
}

func assertMemoryToolSchema(t *testing.T, descriptors []ToolDescriptor, name string, required, properties []string) {
	t.Helper()
	descriptor, ok := memoryToolDescriptorByName(descriptors, name)
	if !ok {
		t.Fatalf("%s descriptor missing", name)
	}
	var schema struct {
		Type       string                     `json:"type"`
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(descriptor.Schema, &schema); err != nil {
		t.Fatalf("decode %s schema: %v", name, err)
	}
	if schema.Type != "object" {
		t.Fatalf("%s schema type = %q, want object", name, schema.Type)
	}
	for _, property := range properties {
		if _, ok := schema.Properties[property]; !ok {
			t.Fatalf("%s schema missing property %q", name, property)
		}
	}
	for _, field := range required {
		if !stringSliceContains(schema.Required, field) {
			t.Fatalf("%s required = %#v, missing %q", name, schema.Required, field)
		}
	}
}

func memoryToolDescriptorByName(descriptors []ToolDescriptor, name string) (ToolDescriptor, bool) {
	for _, descriptor := range descriptors {
		if descriptor.Name == name {
			return descriptor, true
		}
	}
	return ToolDescriptor{}, false
}
