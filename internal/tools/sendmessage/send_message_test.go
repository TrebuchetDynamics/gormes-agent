package sendmessage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSendMessageTool_Schema(t *testing.T) {
	tool := NewSendMessageTool(nil)
	if tool.Name() != "send_message" {
		t.Fatalf("Name() = %q", tool.Name())
	}
	var schema struct {
		Properties map[string]struct {
			Enum []string `json:"enum"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("Schema() invalid JSON: %v", err)
	}
	if !contains(schema.Properties["action"].Enum, "send") || !contains(schema.Properties["action"].Enum, "list") {
		t.Fatalf("action enum = %#v, want send and list", schema.Properties["action"].Enum)
	}
	if len(schema.Required) != 0 {
		t.Fatalf("required = %#v, want none so action=list is valid", schema.Required)
	}
}

func TestSendMessageToolListAndValidatedSendContract(t *testing.T) {
	t.Run("list uses injected directory", func(t *testing.T) {
		tool := NewSendMessageToolWithOptions(Options{
			Directory: fakeDirectory{targets: []Target{{Platform: "telegram", ChatID: "-100", ThreadID: "42", Name: "ops"}}},
		})
		result, err := tool.Execute(context.Background(), mustJSON(map[string]string{"action": "list"}))
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		var out struct {
			Success bool     `json:"success"`
			Targets []Target `json:"targets"`
		}
		mustUnmarshal(t, result, &out)
		if !out.Success || len(out.Targets) != 1 || out.Targets[0].String() != "telegram:-100:42" {
			t.Fatalf("list output = %s", result)
		}
	})

	t.Run("list without directory fails closed", func(t *testing.T) {
		result, err := NewSendMessageTool(nil).Execute(context.Background(), mustJSON(map[string]string{"action": "list"}))
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		assertJSONField(t, result, "evidence", "send_message_directory_unavailable")
		assertJSONField(t, result, "success", false)
	})

	t.Run("send validates required target and message", func(t *testing.T) {
		tool := NewSendMessageToolWithOptions(Options{Sender: fakeSender{}})
		result, err := tool.Execute(context.Background(), mustJSON(map[string]string{"action": "send", "target": "telegram:-100"}))
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		assertJSONField(t, result, "error", "Both 'target' and 'message' are required when action='send'")
	})

	t.Run("send parses target and calls sender once", func(t *testing.T) {
		sender := &recordingSender{}
		tool := NewSendMessageToolWithOptions(Options{Sender: sender})
		result, err := tool.Execute(context.Background(), sendMessageJSON("discord:999888777:555444333", "hello"))
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		assertJSONField(t, result, "success", true)
		if len(sender.requests) != 1 {
			t.Fatalf("sender calls = %d, want 1", len(sender.requests))
		}
		got := sender.requests[0]
		if got.Target.Platform != "discord" || got.Target.ChatID != "999888777" || got.Target.ThreadID != "555444333" || got.Message != "hello" {
			t.Fatalf("request = %#v", got)
		}
	})

	t.Run("friendly unresolved target asks model to list first", func(t *testing.T) {
		sender := &recordingSender{}
		tool := NewSendMessageToolWithOptions(Options{Directory: fakeDirectory{}, Sender: sender})
		result, err := tool.Execute(context.Background(), sendMessageJSON("slack:#engineering", "hello"))
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if len(sender.requests) != 0 {
			t.Fatalf("sender calls = %d, want 0", len(sender.requests))
		}
		if !strings.Contains(string(result), "send_message(action='list')") {
			t.Fatalf("expected list guidance, got %s", result)
		}
	})

	t.Run("send without sender fails closed after validation", func(t *testing.T) {
		result, err := NewSendMessageTool(nil).Execute(context.Background(), sendMessageJSON("telegram:-100", "hello"))
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		assertJSONField(t, result, "evidence", "send_message_backend_unavailable")
		assertJSONField(t, result, "success", false)
	})
}

func TestSendMessageTool_Execute(t *testing.T) {
	var sentTarget, sentMessage string
	tool := NewSendMessageTool(func(target, message string) error {
		sentTarget = target
		sentMessage = message
		return nil
	})
	result, err := tool.Execute(context.Background(), sendMessageJSON("telegram:123", "hello"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out map[string]any
	json.Unmarshal(result, &out)
	if out["success"] != true {
		t.Fatal("expected success")
	}
	if sentTarget != "telegram:123" || sentMessage != "hello" {
		t.Fatalf("sendFn not called: target=%q msg=%q", sentTarget, sentMessage)
	}
}

func TestSendMessageTool_Error(t *testing.T) {
	tool := NewSendMessageTool(func(_, _ string) error {
		return fmt.Errorf("send failed")
	})
	result, err := tool.Execute(context.Background(), sendMessageJSON("test:msg", "msg"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(string(result), "send failed") {
		t.Fatalf("expected error in result: %s", result)
	}
}

func sendMessageJSON(target, message string) []byte {
	return mustJSON(map[string]string{"target": target, "message": message})
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func mustUnmarshal(t *testing.T, b []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("invalid JSON %s: %v", b, err)
	}
}

func assertJSONField(t *testing.T, b []byte, key string, want any) {
	t.Helper()
	var out map[string]any
	mustUnmarshal(t, b, &out)
	if got := out[key]; got != want {
		t.Fatalf("%s = %#v, want %#v in %s", key, got, want, b)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

type fakeDirectory struct {
	targets []Target
	resolve map[string]string
	err     error
}

func (f fakeDirectory) ListTargets(context.Context) ([]Target, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.targets, nil
}

func (f fakeDirectory) ResolveTarget(_ context.Context, platform, ref string) (string, bool, error) {
	if f.err != nil {
		return "", false, f.err
	}
	if f.resolve == nil {
		return "", false, nil
	}
	v, ok := f.resolve[platform+":"+ref]
	return v, ok, nil
}

type fakeSender struct{}

func (fakeSender) SendMessage(context.Context, SendRequest) error {
	return errors.New("fake sender not implemented")
}

type recordingSender struct{ requests []SendRequest }

func (r *recordingSender) SendMessage(_ context.Context, req SendRequest) error {
	r.requests = append(r.requests, req)
	return nil
}
