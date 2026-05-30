package sendmessage

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestSendMessageTool_Schema(t *testing.T) {
	tool := NewSendMessageTool(nil)
	if tool.Name() != "send_message" {
		t.Fatalf("Name() = %q", tool.Name())
	}
	if tool.Schema() == nil {
		t.Fatal("Schema() is nil")
	}
}

func TestSendMessageTool_Execute(t *testing.T) {
	var sentTarget, sentMessage string
	tool := NewSendMessageTool(func(target, message string) error {
		sentTarget = target
		sentMessage = message
		return nil
	})
	result, err := tool.Execute(nil, sendMessageJSON("telegram:123", "hello"))
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
	result, err := tool.Execute(nil, sendMessageJSON("test", "msg"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(string(result), "send failed") {
		t.Fatalf("expected error in result: %s", result)
	}
}

func sendMessageJSON(target, message string) []byte {
	b, _ := json.Marshal(map[string]string{"target": target, "message": message})
	return b
}
