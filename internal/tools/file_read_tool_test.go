package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileTool_ReadsLineNumberedWindow(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("alpha\nbeta\ngamma\ndelta"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	tool := NewReadFileTool(ReadFileToolConfig{Root: root})
	out := executeReadFileTool(t, tool, `{"path":"notes.txt","offset":2,"limit":2}`)

	if out["path"] != "notes.txt" {
		t.Fatalf("path = %v, want notes.txt", out["path"])
	}
	if out["total_lines"] != float64(4) {
		t.Fatalf("total_lines = %v, want 4", out["total_lines"])
	}
	content, _ := out["content"].(string)
	for _, want := range []string{"     2|beta", "     3|gamma"} {
		if !strings.Contains(content, want) {
			t.Fatalf("content missing %q in:\n%s", want, content)
		}
	}
	if strings.Contains(content, "alpha") || strings.Contains(content, "delta") {
		t.Fatalf("content included lines outside requested window:\n%s", content)
	}
	if out["truncated"] != true {
		t.Fatalf("truncated = %v, want true", out["truncated"])
	}
	if !strings.Contains(asString(out["hint"]), "offset=4") {
		t.Fatalf("hint = %v, want continuation offset", out["hint"])
	}
}

func TestReadFileTool_BlocksPathsOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("do not leak"), 0o644); err != nil {
		t.Fatalf("write outside fixture: %v", err)
	}

	tool := NewReadFileTool(ReadFileToolConfig{Root: root})
	out := executeReadFileTool(t, tool, `{"path":`+quoteJSON(t, outside)+`}`)

	if !strings.Contains(asString(out["error"]), "outside workspace root") {
		t.Fatalf("error = %v, want outside-root denial", out["error"])
	}
	if strings.Contains(asString(out["content"]), "do not leak") {
		t.Fatalf("outside file content leaked: %#v", out)
	}
}

func TestReadFileTool_BlocksSymlinkEscapingRoot(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	outside := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outside, []byte("do not leak via symlink"), 0o644); err != nil {
		t.Fatalf("write outside fixture: %v", err)
	}
	link := filepath.Join(root, "linked-secret.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	tool := NewReadFileTool(ReadFileToolConfig{Root: root})
	out := executeReadFileTool(t, tool, `{"path":"linked-secret.txt"}`)

	if !strings.Contains(asString(out["error"]), "outside workspace root") {
		t.Fatalf("error = %v, want symlink outside-root denial", out["error"])
	}
	if strings.Contains(asString(out["content"]), "do not leak") {
		t.Fatalf("symlinked outside file content leaked: %#v", out)
	}
}

func TestReadFileTool_DuplicateReadReturnsStatusNotContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("alpha\nbeta"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	tool := NewReadFileTool(ReadFileToolConfig{Root: root})
	first := executeReadFileTool(t, tool, `{"path":"notes.txt"}`)
	if !strings.Contains(asString(first["content"]), "alpha") {
		t.Fatalf("first read content = %#v, want file content", first)
	}

	second := executeReadFileTool(t, tool, `{"path":"notes.txt"}`)
	if second["status"] != FileReadDedupStatusUnchanged {
		t.Fatalf("second status = %v, want %q", second["status"], FileReadDedupStatusUnchanged)
	}
	if asString(second["content"]) != "" {
		t.Fatalf("second read repeated file content instead of status stub: %#v", second)
	}
	if second["content_returned"] != false {
		t.Fatalf("content_returned = %v, want false", second["content_returned"])
	}
	if !strings.Contains(asString(second["message"]), "earlier read_file result") {
		t.Fatalf("message = %v, want dedup guidance", second["message"])
	}
}

func TestReadFileTool_DedupDoesNotBlockPagination(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("alpha\nbeta\ngamma"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	tool := NewReadFileTool(ReadFileToolConfig{Root: root})
	first := executeReadFileTool(t, tool, `{"path":"notes.txt","offset":1,"limit":1}`)
	if !strings.Contains(asString(first["content"]), "alpha") {
		t.Fatalf("first page = %#v, want alpha content", first)
	}

	second := executeReadFileTool(t, tool, `{"path":"notes.txt","offset":2,"limit":1}`)
	if second["status"] == FileReadDedupStatusUnchanged {
		t.Fatalf("second page returned duplicate status instead of next window: %#v", second)
	}
	if !strings.Contains(asString(second["content"]), "beta") {
		t.Fatalf("second page = %#v, want beta content", second)
	}
}

func TestSearchFilesTool_SearchesContentAndFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "alpha.go"), []byte("package pkg\n\nfunc Alpha() {}\n"), 0o644); err != nil {
		t.Fatalf("write alpha: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.md"), []byte("Alpha note\n"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	tool := NewSearchFilesTool(FileTaskToolConfig{Root: root})
	content := executeSearchFilesTool(t, tool, `{"pattern":"Alpha","target":"content","file_glob":"*.go"}`)
	matches, _ := content["matches"].([]any)
	if len(matches) != 1 {
		t.Fatalf("content matches = %#v, want one Go match", content["matches"])
	}
	first, _ := matches[0].(map[string]any)
	if first["path"] != "pkg/alpha.go" || first["line"] != float64(3) {
		t.Fatalf("first match = %#v, want pkg/alpha.go line 3", first)
	}

	files := executeSearchFilesTool(t, tool, `{"pattern":"*.go","target":"files"}`)
	paths, _ := files["files"].([]any)
	if len(paths) != 1 || paths[0] != "pkg/alpha.go" {
		t.Fatalf("files = %#v, want pkg/alpha.go", files["files"])
	}
}

func TestWriteFileAndPatchTools_EditInsideRoot(t *testing.T) {
	root := t.TempDir()
	write := NewWriteFileTool(FileTaskToolConfig{Root: root})
	wrote := executeWriteFileTool(t, write, `{"path":"pkg/notes.txt","content":"alpha\nbeta\n"}`)
	if wrote["status"] != "ok" || wrote["path"] != "pkg/notes.txt" {
		t.Fatalf("write result = %#v, want ok pkg/notes.txt", wrote)
	}

	patch := NewPatchTool(FileTaskToolConfig{Root: root})
	patched := executePatchTool(t, patch, `{"path":"pkg/notes.txt","old_string":"beta","new_string":"gamma"}`)
	if patched["status"] != "ok" || patched["replacements"] != float64(1) {
		t.Fatalf("patch result = %#v, want one replacement", patched)
	}

	raw, err := os.ReadFile(filepath.Join(root, "pkg", "notes.txt"))
	if err != nil {
		t.Fatalf("read patched file: %v", err)
	}
	if got, want := string(raw), "alpha\ngamma\n"; got != want {
		t.Fatalf("patched content = %q, want %q", got, want)
	}
}

func executeReadFileTool(t *testing.T, tool *ReadFileTool, args string) map[string]any {
	t.Helper()
	raw, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	return out
}

func executeSearchFilesTool(t *testing.T, tool *SearchFilesTool, args string) map[string]any {
	t.Helper()
	raw, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	return out
}

func executeWriteFileTool(t *testing.T, tool *WriteFileTool, args string) map[string]any {
	t.Helper()
	raw, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	return out
}

func executePatchTool(t *testing.T, tool *PatchTool, args string) map[string]any {
	t.Helper()
	raw, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	return out
}

func quoteJSON(t *testing.T, s string) string {
	t.Helper()
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal string: %v", err)
	}
	return string(raw)
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
