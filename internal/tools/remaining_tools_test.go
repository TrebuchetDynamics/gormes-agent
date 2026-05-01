package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestPatchParserTool_Schema(t *testing.T) {
	tool := NewPatchParserTool()
	if tool.Name() != "patch_parser" {
		t.Fatalf("Name() = %q", tool.Name())
	}
	if tool.Schema() == nil {
		t.Fatal("Schema() is nil")
	}
}

func TestPatchParserTool_UnifiedDiff(t *testing.T) {
	tool := NewPatchParserTool()
	patch := `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main
+import "fmt"
 func main() {
-       println("hi")
+       fmt.Println("hello")
 }`
	result, err := tool.Execute(nil, toJSON(patch))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out struct {
		Success bool            `json:"success"`
		Files   []patchFileInfo `json:"files"`
		Count   int             `json:"count"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.Success {
		t.Fatal("expected success")
	}
	if out.Count == 0 {
		t.Fatal("expected at least one file")
	}
}

func TestPatchParserTool_MultiFile(t *testing.T) {
	tool := NewPatchParserTool()
	patch := `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1 +1,2 @@
+line
diff --git a/b.go b/b.go
--- a/b.go
+++ b/b.go
@@ -1 +1 @@
-old
+new`
	result, err := tool.Execute(nil, toJSON(patch))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out struct {
		Files []patchFileInfo `json:"files"`
	}
	json.Unmarshal(result, &out)
	if len(out.Files) != 2 {
		t.Fatalf("files = %d, want 2", len(out.Files))
	}
}

func TestPatchParserTool_EmptyInput(t *testing.T) {
	tool := NewPatchParserTool()
	_, err := tool.Execute(nil, toJSON(""))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func toJSON(s string) []byte {
	b, _ := json.Marshal(map[string]string{"patch": s})
	return b
}

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
	result, err := tool.Execute(nil, toJSONMsg("telegram:123", "hello"))
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
	result, err := tool.Execute(nil, toJSONMsg("test", "msg"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(string(result), "send failed") {
		t.Fatalf("expected error in result: %s", result)
	}
}

func toJSONMsg(target, message string) []byte {
	b, _ := json.Marshal(map[string]string{"target": target, "message": message})
	return b
}

func TestInterruptTool_Schema(t *testing.T) {
	tool := NewInterruptTool(nil)
	if tool.Name() != "interrupt" {
		t.Fatalf("Name() = %q", tool.Name())
	}
}

func TestInterruptTool_Execute(t *testing.T) {
	called := false
	tool := NewInterruptTool(func() { called = true })
	result, err := tool.Execute(nil, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !called {
		t.Fatal("interruptFn not called")
	}
	var out map[string]any
	json.Unmarshal(result, &out)
	if out["success"] != true {
		t.Fatal("expected success")
	}
}

func TestInterruptTool_NilCallback(t *testing.T) {
	tool := NewInterruptTool(nil)
	_, err := tool.Execute(nil, nil)
	if err != nil {
		t.Fatalf("Execute with nil callback: %v", err)
	}
}
