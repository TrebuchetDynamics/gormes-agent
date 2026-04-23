package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProcessRegistryTracksLifecycle(t *testing.T) {
	reg := NewProcessRegistry()

	session := reg.Start("strict", "/tmp/gormes-exec")
	if session.State != RunStateRunning {
		t.Fatalf("Start state = %q, want %q", session.State, RunStateRunning)
	}

	snap, ok := reg.Session(session.ID)
	if !ok {
		t.Fatalf("Session(%q) missing after Start", session.ID)
	}
	if snap.Workspace != "/tmp/gormes-exec" {
		t.Fatalf("workspace = %q, want %q", snap.Workspace, "/tmp/gormes-exec")
	}

	reg.Finish(session.ID, ExecuteResult{
		Status: "success",
		Stdout: "ok\n",
	})

	snap, ok = reg.Session(session.ID)
	if !ok {
		t.Fatalf("Session(%q) missing after Finish", session.ID)
	}
	if snap.State != RunStateCompleted {
		t.Fatalf("Finish state = %q, want %q", snap.State, RunStateCompleted)
	}
	if snap.Result == nil || snap.Result.Stdout != "ok\n" {
		t.Fatalf("result = %#v, want stdout to be recorded", snap.Result)
	}
}

func TestExecuteCodeToolProjectModeUsesProjectRoot(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "note.txt"), []byte("project-mode-data\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(note.txt): %v", err)
	}

	tool := &ExecuteCodeTool{
		TimeoutD: 20 * time.Second,
		Getwd: func() (string, error) {
			return projectRoot, nil
		},
	}

	args := mustJSON(t, map[string]any{
		"mode": "project",
		"code": `package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	body, err := os.ReadFile("note.txt")
	if err != nil {
		panic(err)
	}
	fmt.Printf("cwd=%s note=%s", must(os.Getwd()), strings.TrimSpace(string(body)))
}

func must(v string, err error) string {
	if err != nil {
		panic(err)
	}
	return v
}
`,
	})

	out, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var got ExecuteResult
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("Unmarshal(result): %v", err)
	}
	if got.Status != "success" {
		t.Fatalf("status = %q, want success", got.Status)
	}
	if !strings.Contains(got.Stdout, "note=project-mode-data") {
		t.Fatalf("stdout = %q, want project note", got.Stdout)
	}
	if !strings.Contains(got.Stdout, "cwd="+projectRoot) {
		t.Fatalf("stdout = %q, want project cwd", got.Stdout)
	}
}

func TestExecuteCodeToolStrictModeScrubsSecretEnv(t *testing.T) {
	projectRoot := t.TempDir()
	t.Setenv("VISIBLE_VALUE", "kept")
	t.Setenv("SECRET_TOKEN", "hidden")

	tool := &ExecuteCodeTool{
		TimeoutD: 20 * time.Second,
		Getwd: func() (string, error) {
			return projectRoot, nil
		},
	}

	args := mustJSON(t, map[string]any{
		"mode": "strict",
		"code": `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Printf("visible=%s secret=%s", os.Getenv("VISIBLE_VALUE"), os.Getenv("SECRET_TOKEN"))
}
`,
	})

	out, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var got ExecuteResult
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("Unmarshal(result): %v", err)
	}
	if got.Status != "success" {
		t.Fatalf("status = %q, want success", got.Status)
	}
	if !strings.Contains(got.Stdout, "visible=kept") {
		t.Fatalf("stdout = %q, want visible env to pass through", got.Stdout)
	}
	if strings.Contains(got.Stdout, "secret=hidden") {
		t.Fatalf("stdout = %q, secret env leaked", got.Stdout)
	}
	if got.Mode != "strict" {
		t.Fatalf("mode = %q, want strict", got.Mode)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return raw
}
