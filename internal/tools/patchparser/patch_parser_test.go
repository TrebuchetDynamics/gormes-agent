package patchparser

import (
	"encoding/json"
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
	result, err := tool.Execute(nil, patchParserJSON(patch))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out struct {
		Success bool            `json:"success"`
		Files   []PatchFileInfo `json:"files"`
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
	result, err := tool.Execute(nil, patchParserJSON(patch))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out struct {
		Files []PatchFileInfo `json:"files"`
	}
	json.Unmarshal(result, &out)
	if len(out.Files) != 2 {
		t.Fatalf("files = %d, want 2", len(out.Files))
	}
}

func TestPatchParserTool_EmptyInput(t *testing.T) {
	tool := NewPatchParserTool()
	_, err := tool.Execute(nil, patchParserJSON(""))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func patchParserJSON(s string) []byte {
	b, _ := json.Marshal(map[string]string{"patch": s})
	return b
}
