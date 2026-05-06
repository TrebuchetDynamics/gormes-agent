package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileStateRegistryReadWriteRoundTrip(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("alpha\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	registry := NewFileStateRegistry()
	cfg := FileTaskToolConfig{Root: root, StateRegistry: registry, TaskID: "agent-a"}

	read := executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"notes.txt"}`)
	state := requireFileState(t, read, "read")
	if state["path"] != "notes.txt" {
		t.Fatalf("state path = %v, want notes.txt", state["path"])
	}
	if state["task_id"] != "agent-a" {
		t.Fatalf("state task_id = %v, want agent-a", state["task_id"])
	}
	if state["read_token"] == "" {
		t.Fatalf("read_token missing from state: %#v", state)
	}
	if state["hash"] == "" || state["mtime_unix_nano"] == nil || state["size_bytes"] != float64(6) {
		t.Fatalf("state lacks hash/mtime/size evidence: %#v", state)
	}

	wrote := executeWriteFileTool(t, NewWriteFileTool(cfg), `{"path":"notes.txt","content":"beta\n"}`)
	if wrote["status"] != "ok" {
		t.Fatalf("write result = %#v, want ok", wrote)
	}
	writeState := requireFileState(t, wrote, "write")
	if writeState["read_token"] == state["read_token"] {
		t.Fatalf("write did not refresh read token: before=%#v after=%#v", state, writeState)
	}
	if writeState["size_bytes"] != float64(5) {
		t.Fatalf("write state size = %v, want 5", writeState["size_bytes"])
	}
}

func TestWriteFileRejectsStaleAfterExternalChange(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("alpha\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	registry := NewFileStateRegistry()
	cfg := FileTaskToolConfig{Root: root, StateRegistry: registry, TaskID: "agent-a"}

	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"notes.txt"}`)
	if err := os.WriteFile(path, []byte("external\n"), 0o644); err != nil {
		t.Fatalf("external write: %v", err)
	}

	out := executeWriteFileTool(t, NewWriteFileTool(cfg), `{"path":"notes.txt","content":"agent\n"}`)
	if out["status"] != "file_stale" {
		t.Fatalf("status = %v, want file_stale: %#v", out["status"], out)
	}
	if !strings.Contains(asString(out["error"]), "file_stale") {
		t.Fatalf("error = %v, want file_stale evidence", out["error"])
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after stale write: %v", err)
	}
	if string(raw) != "external\n" {
		t.Fatalf("stale write mutated file to %q", raw)
	}
}

func TestPatchRejectsStaleAfterExternalChange(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("alpha beta\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	registry := NewFileStateRegistry()
	cfg := FileTaskToolConfig{Root: root, StateRegistry: registry, TaskID: "agent-a"}

	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"notes.txt"}`)
	if err := os.WriteFile(path, []byte("external beta\n"), 0o644); err != nil {
		t.Fatalf("external write: %v", err)
	}

	out := executePatchTool(t, NewPatchTool(cfg), `{"path":"notes.txt","old_string":"beta","new_string":"gamma"}`)
	if out["status"] != "file_stale" {
		t.Fatalf("status = %v, want file_stale: %#v", out["status"], out)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after stale patch: %v", err)
	}
	if string(raw) != "external beta\n" {
		t.Fatalf("stale patch mutated file to %q", raw)
	}
}

func TestSuccessfulWriteRefreshesState(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("alpha beta\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	registry := NewFileStateRegistry()
	cfg := FileTaskToolConfig{Root: root, StateRegistry: registry, TaskID: "agent-a"}

	_ = executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"notes.txt"}`)
	wrote := executeWriteFileTool(t, NewWriteFileTool(cfg), `{"path":"notes.txt","content":"alpha beta\n"}`)
	if wrote["status"] != "ok" {
		t.Fatalf("write result = %#v, want ok", wrote)
	}
	patched := executePatchTool(t, NewPatchTool(cfg), `{"path":"notes.txt","old_string":"beta","new_string":"gamma"}`)
	if patched["status"] != "ok" {
		t.Fatalf("patch after own write = %#v, want ok", patched)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read patched file: %v", err)
	}
	if string(raw) != "alpha gamma\n" {
		t.Fatalf("patched file = %q, want alpha gamma", raw)
	}
}

func TestRelativePathUsesLiveTaskCWD(t *testing.T) {
	root := t.TempDir()
	rootCopy := filepath.Join(root, "shared.txt")
	if err := os.WriteFile(rootCopy, []byte("root copy\n"), 0o644); err != nil {
		t.Fatalf("write root copy: %v", err)
	}
	workdir := filepath.Join(root, "work")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	liveCopy := filepath.Join(workdir, "shared.txt")
	if err := os.WriteFile(liveCopy, []byte("live copy\n"), 0o644); err != nil {
		t.Fatalf("write live copy: %v", err)
	}
	liveCWD := workdir
	cfg := FileTaskToolConfig{
		Root:          root,
		StateRegistry: NewFileStateRegistry(),
		TaskID:        "live-task",
		CWDResolver: func() string {
			return liveCWD
		},
	}

	read := executeReadFileTool(t, NewReadFileTool(cfg), `{"path":"shared.txt"}`)
	if !strings.Contains(asString(read["content"]), "live copy") {
		t.Fatalf("relative read used wrong cwd: %#v", read)
	}
	if err := os.WriteFile(liveCopy, []byte("live copy modified elsewhere\n"), 0o644); err != nil {
		t.Fatalf("external live write: %v", err)
	}
	out := executeWriteFileTool(t, NewWriteFileTool(cfg), `{"path":"shared.txt","content":"agent replacement\n"}`)
	if out["status"] != "file_stale" {
		t.Fatalf("status = %v, want file_stale for live cwd target: %#v", out["status"], out)
	}

	rootRaw, err := os.ReadFile(rootCopy)
	if err != nil {
		t.Fatalf("read root copy: %v", err)
	}
	if string(rootRaw) != "root copy\n" {
		t.Fatalf("root-relative file was mutated: %q", rootRaw)
	}
	liveRaw, err := os.ReadFile(liveCopy)
	if err != nil {
		t.Fatalf("read live copy: %v", err)
	}
	if string(liveRaw) != "live copy modified elsewhere\n" {
		t.Fatalf("live file was mutated despite stale guard: %q", liveRaw)
	}
}

func requireFileState(t *testing.T, out map[string]any, op string) map[string]any {
	t.Helper()
	raw, ok := out["file_state"].(map[string]any)
	if !ok {
		t.Fatalf("%s output missing file_state: %#v", op, out)
	}
	return raw
}
