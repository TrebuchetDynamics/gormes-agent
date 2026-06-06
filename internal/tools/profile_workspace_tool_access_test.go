package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectModeExecuteCodeFailsClosedUnderProfileWorkspaceScope(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	scope, err := NewProfileWorkspaceScope(ProfileWorkspaceScopeOptions{
		ProjectRoots: []string{project},
		OperatorHome: root,
	})
	if err != nil {
		t.Fatalf("NewProfileWorkspaceScope: %v", err)
	}

	codeRaw, err := NewExecuteCodeTool(ExecuteCodeToolConfig{
		DefaultMode:    ExecuteCodeModeProject,
		WorkspaceScope: scope,
	}).Execute(context.Background(), json.RawMessage(`{"language":"sh","code":"printf should-not-run"}`))
	if err != nil {
		t.Fatalf("execute_code Execute: %v", err)
	}
	var code CodeExecutionResult
	if err := json.Unmarshal(codeRaw, &code); err != nil {
		t.Fatalf("decode execute_code result: %v", err)
	}
	if code.Status != "blocked" || code.Evidence != ProfileWorkspaceScopeViolation || !strings.Contains(code.Error, ProfileWorkspaceToolAccessExecuteBlocked) {
		t.Fatalf("execute_code result = %+v, want shared profile workspace execute denial", code)
	}
}
