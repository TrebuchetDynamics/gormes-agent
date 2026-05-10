package tools

import (
	"context"
	"encoding/json"
	"errors"
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

func TestPatchToolFuzzyReplaceLineTrimmedAppliesUniqueMatch(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pkg", "service.py")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	original := "def outer():\n    if enabled:\n        return \"ok\"\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"pkg/service.py"}`)
	out := executePatchTool(t, NewPatchTool(cfg), `{"path":"pkg/service.py","old_string":"if enabled:\nreturn \"ok\"","new_string":"    if enabled:\n        return \"patched\""}`)

	if out["status"] != "ok" || out["replacements"] != float64(1) {
		t.Fatalf("patch result = %#v, want one fuzzy replacement", out)
	}
	assertFileContent(t, path, "def outer():\n    if enabled:\n        return \"patched\"\n")
}

func TestPatchToolFuzzyReplaceWhitespaceNormalizedAppliesUniqueMatch(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pkg", "service.py")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	original := "result = call(  alpha,\t beta  )\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"pkg/service.py"}`)
	out := executePatchTool(t, NewPatchTool(cfg), `{"path":"pkg/service.py","old_string":"result = call( alpha, beta )","new_string":"result = call(alpha, beta)"}`)

	if out["status"] != "ok" || out["replacements"] != float64(1) {
		t.Fatalf("patch result = %#v, want one fuzzy replacement", out)
	}
	assertFileContent(t, path, "result = call(alpha, beta)\n")
}

func TestPatchToolFuzzyReplaceAmbiguousRequiresReplaceAll(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	original := "  Step 1:\n  Do thing.\nStep 2: Wait.\n    Step 1:\n    Do thing.\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"notes.txt"}`)
	out := executePatchTool(t, NewPatchTool(cfg), `{"path":"notes.txt","old_string":"Step 1:\nDo thing.","new_string":"Step 1:\nDone."}`)
	if !strings.Contains(asString(out["error"]), "old_string matched 2 times") {
		t.Fatalf("patch result = %#v, want ambiguous fuzzy match error", out)
	}
	assertFileContent(t, path, original)

	out = executePatchTool(t, NewPatchTool(cfg), `{"path":"notes.txt","old_string":"Step 1:\nDo thing.","new_string":"Step 1:\nDone.","replace_all":true}`)
	if out["status"] != "ok" || out["replacements"] != float64(2) {
		t.Fatalf("replace_all result = %#v, want two replacements", out)
	}
	assertFileContent(t, path, "Step 1:\nDone.\nStep 2: Wait.\nStep 1:\nDone.\n")
}

func TestPatchToolFuzzyReplaceEscapeNormalizedAppliesUniqueMatch(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	original := "alpha\n\tbeta\ngamma\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"notes.txt"}`)
	args := `{"path":"notes.txt","old_string":` + quoteJSON(t, `alpha\n\tbeta`) + `,"new_string":` + quoteJSON(t, "alpha\n\tBETA") + `}`
	out := executePatchTool(t, NewPatchTool(cfg), args)

	if out["status"] != "ok" || out["replacements"] != float64(1) {
		t.Fatalf("patch result = %#v, want one escape-normalized replacement", out)
	}
	assertFileContent(t, path, "alpha\n\tBETA\ngamma\n")
}

func TestPatchToolFuzzyReplaceUnicodeSmartQuotes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pkg", "quote.py")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	original := "print(\u201chello\u201d)\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"pkg/quote.py"}`)
	args := `{"path":"pkg/quote.py","old_string":` + quoteJSON(t, `print("hello")`) + `,"new_string":` + quoteJSON(t, `print("world")`) + `}`
	out := executePatchTool(t, NewPatchTool(cfg), args)

	if out["status"] != "ok" || out["replacements"] != float64(1) {
		t.Fatalf("patch result = %#v, want one unicode-normalized replacement", out)
	}
	assertFileContent(t, path, "print(\"world\")\n")
}

func TestPatchToolFuzzyReplaceUnicodeDashAndEllipsis(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	original := "return value\u2014fallback\u2026\nkeep\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"notes.txt"}`)
	args := `{"path":"notes.txt","old_string":` + quoteJSON(t, "return value--fallback...") + `,"new_string":` + quoteJSON(t, "return value or fallback") + `}`
	out := executePatchTool(t, NewPatchTool(cfg), args)

	if out["status"] != "ok" || out["replacements"] != float64(1) {
		t.Fatalf("patch result = %#v, want one unicode-normalized replacement", out)
	}
	assertFileContent(t, path, "return value or fallback\nkeep\n")
}

func TestPatchToolFuzzyReplaceUnicodeAmbiguousRequiresReplaceAll(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "quotes.py")
	original := "print(\u201chello\u201d)\nprint(\u201chello\u201d)\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"quotes.py"}`)
	args := `{"path":"quotes.py","old_string":` + quoteJSON(t, `print("hello")`) + `,"new_string":` + quoteJSON(t, `print("world")`) + `}`
	out := executePatchTool(t, NewPatchTool(cfg), args)
	if !strings.Contains(asString(out["error"]), "old_string matched 2 times") {
		t.Fatalf("patch result = %#v, want ambiguous unicode-normalized match error", out)
	}
	assertFileContent(t, path, original)

	args = `{"path":"quotes.py","old_string":` + quoteJSON(t, `print("hello")`) + `,"new_string":` + quoteJSON(t, `print("world")`) + `,"replace_all":true}`
	out = executePatchTool(t, NewPatchTool(cfg), args)
	if out["status"] != "ok" || out["replacements"] != float64(2) {
		t.Fatalf("replace_all result = %#v, want two replacements", out)
	}
	assertFileContent(t, path, "print(\"world\")\nprint(\"world\")\n")
}

func TestPatchToolFuzzyReplaceBlockAnchorHighSimilarity(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pkg", "service.py")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	original := "def combine():\n    x = 1\n    y = 2\n    return x + y\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"pkg/service.py"}`)
	args := `{"path":"pkg/service.py","old_string":` + quoteJSON(t, "def combine():\n    x = 1\n    y = 9\n    return x + y") + `,"new_string":` + quoteJSON(t, "def combine():\n    return 0") + `}`
	out := executePatchTool(t, NewPatchTool(cfg), args)

	if out["status"] != "ok" || out["replacements"] != float64(1) {
		t.Fatalf("patch result = %#v, want one block-anchor replacement", out)
	}
	assertFileContent(t, path, "def combine():\n    return 0\n")
}

func TestPatchToolFuzzyReplaceBlockAnchorLowSimilarityNoMutation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pkg", "service.py")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	original := "class Worker:\n    completely = 'unrelated'\n    content = 'here'\n    nothing = 'in common'\n    pass\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"pkg/service.py"}`)
	args := `{"path":"pkg/service.py","old_string":` + quoteJSON(t, "class Worker:\n    x = 1\n    y = 2\n    z = 3\n    pass") + `,"new_string":"replaced"}`
	out := executePatchTool(t, NewPatchTool(cfg), args)

	if !strings.Contains(asString(out["error"]), "old_string not found") {
		t.Fatalf("patch result = %#v, want old_string not found", out)
	}
	assertFileContent(t, path, original)
}

func TestPatchToolFuzzyReplaceBlockAnchorAmbiguousRequiresReplaceAll(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	original := strings.Join([]string{
		"section:",
		"  alpha = 1",
		"  beta = 2",
		"done",
		"",
		"section:",
		"  alpha = 1",
		"  beta = 2",
		"done",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"notes.txt"}`)
	oldString := "section:\n  alpha = 1\n  beta = 3\ndone"
	newString := "section:\n  patched = true\ndone"
	args := `{"path":"notes.txt","old_string":` + quoteJSON(t, oldString) + `,"new_string":` + quoteJSON(t, newString) + `}`
	out := executePatchTool(t, NewPatchTool(cfg), args)
	if !strings.Contains(asString(out["error"]), "old_string matched 2 times") {
		t.Fatalf("patch result = %#v, want ambiguous block-anchor match error", out)
	}
	assertFileContent(t, path, original)

	args = `{"path":"notes.txt","old_string":` + quoteJSON(t, oldString) + `,"new_string":` + quoteJSON(t, newString) + `,"replace_all":true}`
	out = executePatchTool(t, NewPatchTool(cfg), args)
	if out["status"] != "ok" || out["replacements"] != float64(2) {
		t.Fatalf("replace_all result = %#v, want two block-anchor replacements", out)
	}
	assertFileContent(t, path, "section:\n  patched = true\ndone\n\nsection:\n  patched = true\ndone\n")
}

func TestPatchToolFuzzyReplaceContextAwareHighLineSimilarity(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pkg", "service.py")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	original := strings.Join([]string{
		"def configure_feature():",
		"    alpha = \"ready\"",
		"    beta = \"stable\"",
		"    gamma = \"done\"",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"pkg/service.py"}`)
	oldString := strings.Join([]string{
		"def config_feature():",
		"    alpha = \"ready\"",
		"    beta = \"steady\"",
		"    gamma = \"done\"",
	}, "\n")
	newString := "def configure_feature():\n    return \"patched\""
	args := `{"path":"pkg/service.py","old_string":` + quoteJSON(t, oldString) + `,"new_string":` + quoteJSON(t, newString) + `}`
	out := executePatchTool(t, NewPatchTool(cfg), args)

	if out["status"] != "ok" || out["replacements"] != float64(1) {
		t.Fatalf("patch result = %#v, want one context-aware replacement", out)
	}
	assertFileContent(t, path, "def configure_feature():\n    return \"patched\"\n")
}

func TestPatchToolFuzzyReplaceContextAwareAmbiguousRequiresReplaceAll(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	original := strings.Join([]string{
		"section alpha:",
		"  ready = true",
		"  count = 2",
		"done alpha",
		"",
		"section beta:",
		"  ready = true",
		"  count = 2",
		"done beta",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"notes.txt"}`)
	oldString := "section target:\n  ready = true\n  count = 9\ndone target"
	newString := "section patched:\n  ok = true"
	args := `{"path":"notes.txt","old_string":` + quoteJSON(t, oldString) + `,"new_string":` + quoteJSON(t, newString) + `}`
	out := executePatchTool(t, NewPatchTool(cfg), args)
	if !strings.Contains(asString(out["error"]), "old_string matched 2 times") {
		t.Fatalf("patch result = %#v, want ambiguous context-aware match error", out)
	}
	assertFileContent(t, path, original)

	args = `{"path":"notes.txt","old_string":` + quoteJSON(t, oldString) + `,"new_string":` + quoteJSON(t, newString) + `,"replace_all":true}`
	out = executePatchTool(t, NewPatchTool(cfg), args)
	if out["status"] != "ok" || out["replacements"] != float64(2) {
		t.Fatalf("replace_all result = %#v, want two context-aware replacements", out)
	}
	assertFileContent(t, path, "section patched:\n  ok = true\n\nsection patched:\n  ok = true\n")
}

func TestPatchToolFuzzyReplaceContextAwareLowSimilarityNoMutation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pkg", "service.py")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	original := "class Worker:\n    completely = 'unrelated'\n    content = 'here'\n    nothing = 'in common'\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"pkg/service.py"}`)
	oldString := "class WorkerRenamed:\n    alpha = 1\n    beta = 2\n    return alpha + beta"
	args := `{"path":"pkg/service.py","old_string":` + quoteJSON(t, oldString) + `,"new_string":"patched"}`
	out := executePatchTool(t, NewPatchTool(cfg), args)

	if !strings.Contains(asString(out["error"]), "old_string not found") {
		t.Fatalf("patch result = %#v, want old_string not found", out)
	}
	assertFileContent(t, path, original)
}

func TestPatchToolReplaceNoMatchIncludesDidYouMeanHint(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pkg", "service.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	original := "package pkg\n\nfunc BuildService() string {\n\treturn \"ok\"\n}\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"pkg/service.go"}`)
	out := executePatchTool(t, NewPatchTool(cfg), `{"path":"pkg/service.go","old_string":"func MissingWidget() error {","new_string":"func BuildServiceV2() string {"}`)

	if !strings.Contains(asString(out["error"]), "old_string not found") {
		t.Fatalf("error = %v, want old_string not found", out["error"])
	}
	hint := asString(out["hint"])
	for _, want := range []string{"Did you mean one of these sections?", "| func BuildService() string {", "read_file"} {
		if !strings.Contains(hint, want) {
			t.Fatalf("hint missing %q in:\n%s", want, hint)
		}
	}
	assertFileContent(t, path, original)
}

func TestPatchToolReplaceNoMatchGenericHintWhenNoSimilarContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	original := "alpha\nbeta\ngamma\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"notes.txt"}`)
	out := executePatchTool(t, NewPatchTool(cfg), `{"path":"notes.txt","old_string":"totally_unique_xyzzy_qux","new_string":"replacement"}`)

	if !strings.Contains(asString(out["error"]), "old_string not found") {
		t.Fatalf("error = %v, want old_string not found", out["error"])
	}
	hint := asString(out["hint"])
	for _, want := range []string{"old_string not found", "read_file", "search_files"} {
		if !strings.Contains(hint, want) {
			t.Fatalf("hint missing %q in:\n%s", want, hint)
		}
	}
	if strings.Contains(hint, "alpha") || strings.Contains(hint, "beta") || strings.Contains(hint, "gamma") {
		t.Fatalf("generic hint leaked unrelated file content:\n%s", hint)
	}
	assertFileContent(t, path, original)
}

func TestPatchToolReplaceNoMatchLeavesAmbiguousMatchesUntouched(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	original := "alpha\nalpha\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"notes.txt"}`)
	out := executePatchTool(t, NewPatchTool(cfg), `{"path":"notes.txt","old_string":"alpha","new_string":"omega"}`)

	if !strings.Contains(asString(out["error"]), "old_string matched 2 times") {
		t.Fatalf("error = %v, want ambiguous-match error", out["error"])
	}
	if hint := asString(out["hint"]); hint != "" {
		t.Fatalf("ambiguous match should not get no-match hint, got:\n%s", hint)
	}
	assertFileContent(t, path, original)
}

func TestPatchToolV4AAppliesAddUpdateDelete(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "app.txt"), []byte("alpha\nbeta\nomega\n"), 0o644); err != nil {
		t.Fatalf("write app fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "delete.txt"), []byte("remove me\n"), 0o644); err != nil {
		t.Fatalf("write delete fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/app.txt"}`)
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/delete.txt"}`)

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Update File: src/app.txt",
		"@@",
		" alpha",
		"-beta",
		"+gamma",
		" omega",
		"*** Add File: src/new.txt",
		"+created",
		"+file",
		"*** Delete File: src/delete.txt",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != "ok" || out["operations"] != float64(3) {
		t.Fatalf("patch result = %#v, want ok with 3 operations", out)
	}
	assertStringListContains(t, out["files_modified"], "src/app.txt")
	assertStringListContains(t, out["files_created"], "src/new.txt")
	assertStringListContains(t, out["files_deleted"], "src/delete.txt")
	assertFileContent(t, filepath.Join(root, "src", "app.txt"), "alpha\ngamma\nomega\n")
	assertFileContent(t, filepath.Join(root, "src", "new.txt"), "created\nfile")
	if _, err := os.Stat(filepath.Join(root, "src", "delete.txt")); !os.IsNotExist(err) {
		t.Fatalf("delete.txt stat err = %v, want not exist", err)
	}
}

func TestPatchToolV4ARollbackOnApplyFailure(t *testing.T) {
	root := t.TempDir()
	appPath := filepath.Join(root, "src", "app.txt")
	deletePath := filepath.Join(root, "src", "delete.txt")
	if err := os.MkdirAll(filepath.Dir(appPath), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(appPath, []byte("alpha\nbeta\n"), 0o644); err != nil {
		t.Fatalf("write app fixture: %v", err)
	}
	if err := os.WriteFile(deletePath, []byte("keep me\n"), 0o644); err != nil {
		t.Fatalf("write delete fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/app.txt"}`)
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/delete.txt"}`)

	originalRemove := fileTaskRemove
	fileTaskRemove = func(path string) error {
		if path == deletePath {
			return errors.New("injected delete failure")
		}
		return originalRemove(path)
	}
	defer func() { fileTaskRemove = originalRemove }()

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Update File: src/app.txt",
		"@@",
		" alpha",
		"-beta",
		"+gamma",
		"*** Delete File: src/delete.txt",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != "patch_apply_failed" || out["rolled_back"] != true {
		t.Fatalf("patch result = %#v, want apply failure with rolled_back=true", out)
	}
	if !strings.Contains(asString(out["error"]), "injected delete failure") {
		t.Fatalf("error = %v, want injected delete failure", out["error"])
	}
	assertStringListContains(t, out["files_modified"], "src/app.txt")
	assertFileContent(t, appPath, "alpha\nbeta\n")
	assertFileContent(t, deletePath, "keep me\n")
}

func TestPatchToolV4ARollbackRemovesCreatedFiles(t *testing.T) {
	root := t.TempDir()
	deletePath := filepath.Join(root, "src", "delete.txt")
	createdPath := filepath.Join(root, "src", "created.txt")
	if err := os.MkdirAll(filepath.Dir(deletePath), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(deletePath, []byte("keep me\n"), 0o644); err != nil {
		t.Fatalf("write delete fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/delete.txt"}`)

	originalRemove := fileTaskRemove
	fileTaskRemove = func(path string) error {
		if path == deletePath {
			return errors.New("injected delete failure")
		}
		return originalRemove(path)
	}
	defer func() { fileTaskRemove = originalRemove }()

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Add File: src/created.txt",
		"+temporary",
		"*** Delete File: src/delete.txt",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != "patch_apply_failed" || out["rolled_back"] != true {
		t.Fatalf("patch result = %#v, want apply failure with rolled_back=true", out)
	}
	assertStringListContains(t, out["files_created"], "src/created.txt")
	if _, err := os.Stat(createdPath); !os.IsNotExist(err) {
		t.Fatalf("created file stat err = %v, want not exist after rollback", err)
	}
	assertFileContent(t, deletePath, "keep me\n")
}

func TestPatchToolV4AFuzzyHunkLineTrimmedApplies(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "src", "app.py")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte("def run():\n    if enabled:\n        return \"ok\"\n"), 0o644); err != nil {
		t.Fatalf("write app fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/app.py"}`)

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Update File: src/app.py",
		"@@",
		" def run():",
		"-if enabled:",
		"-return \"ok\"",
		"+    if enabled:",
		"+        return \"patched\"",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != "ok" || out["operations"] != float64(1) {
		t.Fatalf("patch result = %#v, want ok with one fuzzy V4A operation", out)
	}
	assertFileContent(t, path, "def run():\n    if enabled:\n        return \"patched\"\n")
}

func TestPatchToolV4AFuzzyHunkContextHintDisambiguatesNearbyWindow(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "src", "settings.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	filler := strings.Repeat("filler line keeps sections apart\n", 30)
	original := "section alpha\ntarget = old\n" + filler + "marker beta\ntarget = old\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write settings fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/settings.txt"}`)

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Update File: src/settings.txt",
		"@@ marker beta @@",
		"-target = old",
		"+target = new",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != "ok" || out["operations"] != float64(1) {
		t.Fatalf("patch result = %#v, want ok with context-hint V4A operation", out)
	}
	assertFileContent(t, path, "section alpha\ntarget = old\n"+filler+"marker beta\ntarget = new\n")
}

func TestPatchToolV4AFuzzyHunkAmbiguousNoMutation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "src", "settings.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	original := "section alpha\ntarget = old\nsection beta\ntarget = old\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write settings fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/settings.txt"}`)

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Update File: src/settings.txt",
		"@@ missing hint @@",
		"-target = old",
		"+target = new",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != "patch_validation_failed" {
		t.Fatalf("status = %v, want patch_validation_failed: %#v", out["status"], out)
	}
	if !strings.Contains(asString(out["error"]), "old_string matched 2 times") {
		t.Fatalf("error = %v, want ambiguous fuzzy match evidence", out["error"])
	}
	assertFileContent(t, path, original)
}

func TestPatchToolV4AMoveRenamesReadFile(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src", "old.txt")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(src, []byte("move me\n"), 0o644); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/old.txt"}`)

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Move File: src/old.txt -> dst/new.txt",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != "ok" || out["operations"] != float64(1) {
		t.Fatalf("patch result = %#v, want ok with 1 operation", out)
	}
	assertStringListContains(t, out["files_modified"], "src/old.txt -> dst/new.txt")
	assertFileContent(t, filepath.Join(root, "dst", "new.txt"), "move me\n")
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source stat err = %v, want not exist", err)
	}
}

func TestPatchToolV4AMoveRejectsExistingDestination(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src", "old.txt")
	dst := filepath.Join(root, "dst", "new.txt")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("mkdir source fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir destination fixture: %v", err)
	}
	if err := os.WriteFile(src, []byte("source\n"), 0o644); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	if err := os.WriteFile(dst, []byte("destination\n"), 0o644); err != nil {
		t.Fatalf("write destination fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/old.txt"}`)

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Move File: src/old.txt -> dst/new.txt",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != "patch_validation_failed" {
		t.Fatalf("status = %v, want patch_validation_failed: %#v", out["status"], out)
	}
	if !strings.Contains(asString(out["error"]), "destination already exists") {
		t.Fatalf("error = %v, want destination exists message", out["error"])
	}
	assertFileContent(t, src, "source\n")
	assertFileContent(t, dst, "destination\n")
}

func TestPatchToolV4AMoveBlocksOutsideRoot(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src", "old.txt")
	outside := filepath.Join(filepath.Dir(root), "escape.txt")
	_ = os.Remove(outside)
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(src, []byte("source\n"), 0o644); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/old.txt"}`)

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Move File: src/old.txt -> ../escape.txt",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != "patch_validation_failed" {
		t.Fatalf("status = %v, want patch_validation_failed: %#v", out["status"], out)
	}
	if !strings.Contains(asString(out["error"]), "outside workspace root") {
		t.Fatalf("error = %v, want outside-root denial", out["error"])
	}
	assertFileContent(t, src, "source\n")
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("outside file stat err = %v, want not exist", err)
	}
}

func TestPatchToolV4AMoveRejectsStaleSource(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src", "old.txt")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(src, []byte("source\n"), 0o644); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/old.txt"}`)
	if err := os.WriteFile(src, []byte("external change\n"), 0o644); err != nil {
		t.Fatalf("external write: %v", err)
	}

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Move File: src/old.txt -> dst/new.txt",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != fileStateStatusStale {
		t.Fatalf("status = %v, want %q: %#v", out["status"], fileStateStatusStale, out)
	}
	assertFileContent(t, src, "external change\n")
	if _, err := os.Stat(filepath.Join(root, "dst", "new.txt")); !os.IsNotExist(err) {
		t.Fatalf("destination stat err = %v, want not exist", err)
	}
}

func TestPatchToolV4AMoveValidatesBeforeAnyMutation(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src", "old.txt")
	dst := filepath.Join(root, "dst", "new.txt")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatalf("mkdir source fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir destination fixture: %v", err)
	}
	if err := os.WriteFile(src, []byte("source\n"), 0o644); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	if err := os.WriteFile(dst, []byte("destination\n"), 0o644); err != nil {
		t.Fatalf("write destination fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/old.txt"}`)

	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Add File: src/created.txt",
		"+should not exist",
		"*** Move File: src/old.txt -> dst/new.txt",
		"*** End Patch",
	}, "\n")
	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != "patch_validation_failed" {
		t.Fatalf("status = %v, want patch_validation_failed: %#v", out["status"], out)
	}
	assertFileContent(t, src, "source\n")
	assertFileContent(t, dst, "destination\n")
	if _, err := os.Stat(filepath.Join(root, "src", "created.txt")); !os.IsNotExist(err) {
		t.Fatalf("created.txt stat err = %v, want not exist", err)
	}
}

func TestPatchToolV4ABlocksOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "escape.txt")
	_ = os.Remove(outside)
	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Add File: ../escape.txt",
		"+do not write",
		"*** End Patch",
	}, "\n")

	out := executePatchTool(t, NewPatchTool(FileTaskToolConfig{Root: root}), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != "patch_validation_failed" {
		t.Fatalf("status = %v, want patch_validation_failed: %#v", out["status"], out)
	}
	if !strings.Contains(asString(out["error"]), "outside workspace root") {
		t.Fatalf("error = %v, want outside-root denial", out["error"])
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("outside file stat err = %v, want not exist", err)
	}
}

func TestPatchToolV4ARejectsStaleExistingFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "src", "app.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte("alpha\nbeta\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/app.txt"}`)
	if err := os.WriteFile(path, []byte("external\nbeta\n"), 0o644); err != nil {
		t.Fatalf("external write: %v", err)
	}
	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Update File: src/app.txt",
		"@@",
		" alpha",
		"-beta",
		"+gamma",
		"*** End Patch",
	}, "\n")

	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != fileStateStatusStale {
		t.Fatalf("status = %v, want %q: %#v", out["status"], fileStateStatusStale, out)
	}
	assertFileContent(t, path, "external\nbeta\n")
}

func TestPatchToolV4ARejectsMissingUpdateHunk(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "src", "app.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte("alpha\nbeta\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg := FileTaskToolConfig{Root: root, StateRegistry: NewFileStateRegistry(), TaskID: "agent-a"}
	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"src/app.txt"}`)
	patchText := strings.Join([]string{
		"*** Begin Patch",
		"*** Update File: src/app.txt",
		"@@",
		" missing",
		"-context",
		"+replacement",
		"*** End Patch",
	}, "\n")

	out := executePatchTool(t, NewPatchTool(cfg), `{"mode":"patch","patch":`+quoteJSON(t, patchText)+`}`)

	if out["status"] != "patch_validation_failed" {
		t.Fatalf("status = %v, want patch_validation_failed: %#v", out["status"], out)
	}
	if !strings.Contains(asString(out["error"]), "could not apply hunk") {
		t.Fatalf("error = %v, want hunk validation message", out["error"])
	}
	assertFileContent(t, path, "alpha\nbeta\n")
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

func assertStringListContains(t *testing.T, raw any, want string) {
	t.Helper()
	items, _ := raw.([]any)
	for _, item := range items {
		if item == want {
			return
		}
	}
	t.Fatalf("list %#v missing %q", raw, want)
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if got := string(raw); got != want {
		t.Fatalf("%s content = %q, want %q", path, got, want)
	}
}
