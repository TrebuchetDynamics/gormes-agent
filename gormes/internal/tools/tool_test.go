package tools

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

type stubTool struct {
	name, desc string
	schema     json.RawMessage
	timeout    time.Duration
}

func (s *stubTool) Name() string            { return s.name }
func (s *stubTool) Description() string     { return s.desc }
func (s *stubTool) Schema() json.RawMessage { return s.schema }
func (s *stubTool) Timeout() time.Duration  { return s.timeout }
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

func TestRegistry_DescriptorsForToolsets_FiltersUnavailableToolsets(t *testing.T) {
	r := NewRegistry()
	r.MustRegisterEntry(ToolEntry{
		Tool:    &stubTool{name: "alpha", desc: "a", schema: json.RawMessage(`{}`)},
		Toolset: "core",
	})
	r.MustRegisterEntry(ToolEntry{
		Tool:        &stubTool{name: "remote", desc: "r", schema: json.RawMessage(`{}`)},
		Toolset:     "remote",
		RequiresEnv: []string{"REMOTE_TOKEN"},
	})

	ds, err := r.DescriptorsForToolsets([]string{"core"}, nil)
	if err != nil {
		t.Fatalf("DescriptorsForToolsets(core): %v", err)
	}
	if len(ds) != 1 || ds[0].Name != "alpha" {
		t.Fatalf("DescriptorsForToolsets(core) = %+v, want only alpha", ds)
	}

	if _, err := r.DescriptorsForToolsets([]string{"remote"}, nil); !errors.Is(err, ErrUnavailableToolset) {
		t.Fatalf("DescriptorsForToolsets(remote) err = %v, want ErrUnavailableToolset", err)
	}

	if got := r.AvailableToolsets(); !reflect.DeepEqual(got, []string{"core"}) {
		t.Fatalf("AvailableToolsets() = %v, want [core]", got)
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
