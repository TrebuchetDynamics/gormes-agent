package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileToolsUseProfileWorkspaceScopeAcrossRoots(t *testing.T) {
	root := t.TempDir()
	project1 := filepath.Join(root, "project1")
	project2 := filepath.Join(root, "project2")
	profile := filepath.Join(root, ".gormes", "profiles", "coder")
	for _, dir := range []string{project1, project2, profile} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeFile(t, filepath.Join(project1, "one.txt"), "project one")
	writeFile(t, filepath.Join(project2, "two.txt"), "project two")
	writeFile(t, filepath.Join(root, "outside.txt"), "outside")
	writeFile(t, filepath.Join(profile, ".env"), "secret")

	scope, err := NewProfileWorkspaceScope(ProfileWorkspaceScopeOptions{
		ProjectRoots: []string{project1, project2},
		ProfileRoot:  profile,
		OperatorHome: root,
	})
	if err != nil {
		t.Fatalf("NewProfileWorkspaceScope: %v", err)
	}
	cfg := FileTaskToolConfig{
		Root:           project1,
		WorkspaceScope: scope,
	}

	read := NewReadFileTool(cfg)
	out := executeReadFileTool(t, read, `{"path":`+quoteJSON(t, filepath.Join(project2, "two.txt"))+`}`)
	if out["error"] != nil {
		t.Fatalf("read project2 = %#v, want allowed", out)
	}
	if out["path"] != "two.txt" {
		t.Fatalf("read path = %v, want project2-relative path", out["path"])
	}

	out = executeReadFileTool(t, read, `{"path":`+quoteJSON(t, filepath.Join(root, "outside.txt"))+`}`)
	if !strings.Contains(asString(out["error"]), ProfileWorkspaceScopeViolation) {
		t.Fatalf("outside read = %#v, want %s", out, ProfileWorkspaceScopeViolation)
	}

	write := NewWriteFileTool(cfg)
	wrote := executeWriteFileTool(t, write, `{"path":`+quoteJSON(t, filepath.Join(project2, "new.txt"))+`,"content":"ok\n"}`)
	if wrote["status"] != "ok" {
		t.Fatalf("write project2 = %#v, want ok", wrote)
	}

	wrote = executeWriteFileTool(t, write, `{"path":`+quoteJSON(t, filepath.Join(profile, ".env"))+`,"content":"leak\n"}`)
	if !strings.Contains(asString(wrote["error"]), ProfileWorkspaceScopeViolation) {
		t.Fatalf("write profile secret = %#v, want %s", wrote, ProfileWorkspaceScopeViolation)
	}
}

func decodeCodeExecutionResult(t *testing.T, raw json.RawMessage) CodeExecutionResult {
	t.Helper()
	var out CodeExecutionResult
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal execute_code output %s: %v", raw, err)
	}
	return out
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
