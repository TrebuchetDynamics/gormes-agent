package transaction

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/builtins"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

func TestTransactionalExecutor_SafeCommand(t *testing.T) {
	reg := toolkit.NewRegistry()
	reg.Register(&builtins.EchoTool{})
	inner := toolkit.NewInProcessToolExecutor(reg)
	te := NewExecutor(inner, safety.NewCommandClassifier())

	result, err := te.Execute(context.Background(), toolkit.ToolRequest{
		ToolName: "echo",
		Input:    json.RawMessage(`{"text":"hello"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("safe echo should succeed, got: %s", result.Error)
	}
}

func TestTransactionalExecutor_UnsafeBlocked(t *testing.T) {
	cc := safety.NewCommandClassifier()
	te := NewExecutor(nil, cc)

	result, err := te.Execute(context.Background(), toolkit.ToolRequest{
		ToolName: "terminal",
		Input:    json.RawMessage(`sudo rm -rf /`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("unsafe command should be blocked")
	}
	if result.Classification != "unsafe" {
		t.Fatalf("classification = %s, want unsafe", result.Classification)
	}
}

func TestTransactionalExecutor_UncertainTakesWorkspaceSnapshot(t *testing.T) {
	root := t.TempDir()
	reg := toolkit.NewRegistry()
	reg.Register(&builtins.NowTool{})
	inner := toolkit.NewInProcessToolExecutor(reg)
	te := NewExecutor(inner, safety.NewCommandClassifier())

	result, err := te.Execute(context.Background(), toolkit.ToolRequest{
		ToolName: "now",
		Input:    json.RawMessage(`{}`),
		Metadata: map[string]string{"workspace_root": root},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("now tool should succeed, got: %s", result.Error)
	}
	if !result.SnapshotTaken {
		t.Fatalf("uncertain command SnapshotTaken = false, want snapshot wrapper before execution")
	}
}

func TestTransactionalExecutor_RollsBackFailedUncertainToolFilesystemChanges(t *testing.T) {
	root := t.TempDir()
	mustWriteTransactionalTestFile(t, root, "existing.txt", "before\n")
	mustWriteTransactionalTestFile(t, root, "deleted.txt", "keep\n")

	reg := toolkit.NewRegistry()
	reg.Register(&toolkit.MockTool{
		NameStr: "mutate_workspace",
		ExecuteFn: func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
			mustWriteTransactionalTestFile(t, root, "existing.txt", "after\n")
			if err := os.Remove(filepath.Join(root, "deleted.txt")); err != nil {
				t.Fatalf("remove deleted fixture: %v", err)
			}
			mustWriteTransactionalTestFile(t, root, "new.txt", "created\n")
			return nil, errors.New("tool failed after mutating workspace")
		},
	})
	te := NewExecutor(toolkit.NewInProcessToolExecutor(reg), safety.NewCommandClassifier())

	result, err := te.Execute(context.Background(), toolkit.ToolRequest{
		ToolName: "mutate_workspace",
		Input:    json.RawMessage(`{"command":"custom mutating operation"}`),
		Metadata: map[string]string{"workspace_root": root},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("failed tool unexpectedly succeeded")
	}
	if !result.SnapshotTaken || !result.RolledBack {
		t.Fatalf("result = %+v, want snapshot taken and rollback", result)
	}
	assertTransactionalTestFile(t, root, "existing.txt", "before\n")
	assertTransactionalTestFile(t, root, "deleted.txt", "keep\n")
	if _, err := os.Stat(filepath.Join(root, "new.txt")); !os.IsNotExist(err) {
		t.Fatalf("new file still exists after rollback; stat err=%v", err)
	}
}

func TestTransactionalExecutor_CommitsSuccessfulUncertainToolFilesystemChanges(t *testing.T) {
	root := t.TempDir()
	mustWriteTransactionalTestFile(t, root, "existing.txt", "before\n")

	reg := toolkit.NewRegistry()
	reg.Register(&toolkit.MockTool{
		NameStr: "mutate_workspace",
		ExecuteFn: func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
			mustWriteTransactionalTestFile(t, root, "existing.txt", "after\n")
			mustWriteTransactionalTestFile(t, root, "new.txt", "created\n")
			return json.RawMessage(`{"status":"ok"}`), nil
		},
	})
	te := NewExecutor(toolkit.NewInProcessToolExecutor(reg), safety.NewCommandClassifier())

	result, err := te.Execute(context.Background(), toolkit.ToolRequest{
		ToolName: "mutate_workspace",
		Input:    json.RawMessage(`{"command":"custom mutating operation"}`),
		Metadata: map[string]string{"workspace_root": root},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.RolledBack {
		t.Fatalf("result = %+v, want success without rollback", result)
	}
	assertTransactionalTestFile(t, root, "existing.txt", "after\n")
	assertTransactionalTestFile(t, root, "new.txt", "created\n")
}

func mustWriteTransactionalTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func assertTransactionalTestFile(t *testing.T, root, rel, want string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	if got := string(raw); got != want {
		t.Fatalf("%s = %q, want %q", rel, got, want)
	}
}
