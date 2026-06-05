package tools

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestEnvironmentContract_FakeBackendRecordsPathFileTimeoutCleanup(t *testing.T) {
	hostRoot := t.TempDir()
	mapper := NewEnvironmentPathMapper(hostRoot, "/workspace/project")
	env := NewFakeEnvironment("fake", mapper)
	env.RegisterCleanup("snapshot")
	env.RegisterCleanup("workspace")

	hostPath := filepath.Join(hostRoot, "internal", "tools", "tool.go")
	environmentPath, err := env.MapPath(hostPath)
	if err != nil {
		t.Fatalf("MapPath returned error: %v", err)
	}
	if environmentPath != "/workspace/project/internal/tools/tool.go" {
		t.Fatalf("MapPath() = %q", environmentPath)
	}

	upload, err := env.Upload(context.Background(), FileSyncIntent{
		Direction:       FileSyncUpload,
		HostPath:        hostPath,
		EnvironmentPath: environmentPath,
		Checksum:        "sha256:upload",
	})
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}
	if upload.Evidence.Code != EnvironmentFileUploadRecorded {
		t.Fatalf("Upload evidence code = %q", upload.Evidence.Code)
	}

	result, err := env.Execute(context.Background(), EnvironmentCommand{
		Command:    "go test ./internal/tools",
		WorkingDir: "/workspace/project",
		Timeout:    42 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Command.Timeout != 42*time.Second {
		t.Fatalf("Execute timeout = %s", result.Command.Timeout)
	}
	if result.Evidence[0].Code != EnvironmentCommandRecorded {
		t.Fatalf("Execute evidence code = %q", result.Evidence[0].Code)
	}

	download, err := env.Download(context.Background(), FileSyncIntent{
		Direction:       FileSyncDownload,
		HostPath:        filepath.Join(hostRoot, ".hermes", "state.json"),
		EnvironmentPath: "/workspace/project/.hermes/state.json",
		Checksum:        "sha256:download",
	})
	if err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	if download.Evidence.Code != EnvironmentFileDownloadRecorded {
		t.Fatalf("Download evidence code = %q", download.Evidence.Code)
	}

	cleanup, err := env.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}
	gotCleanup := []string{cleanup.Evidence[0].Resource, cleanup.Evidence[1].Resource}
	if want := []string{"workspace", "snapshot"}; !reflect.DeepEqual(gotCleanup, want) {
		t.Fatalf("cleanup order = %v, want %v", gotCleanup, want)
	}

	gotOps := operationKinds(env.Operations())
	wantOps := []EnvironmentOperationKind{
		EnvironmentOperationMapPath,
		EnvironmentOperationUpload,
		EnvironmentOperationExecute,
		EnvironmentOperationDownload,
		EnvironmentOperationCleanup,
	}
	if !reflect.DeepEqual(gotOps, wantOps) {
		t.Fatalf("operation kinds = %v, want %v", gotOps, wantOps)
	}
}

func TestEnvironmentContract_FileSyncPlanChecksumDeleteAndNormalization(t *testing.T) {
	hostRoot := t.TempDir()
	mapper := NewEnvironmentPathMapper(filepath.Join(hostRoot, "."), "/remote/../remote/project/")
	unchangedHost := filepath.Join(hostRoot, "unchanged.txt")
	changedHost := filepath.Join(hostRoot, "nested", "..", "changed.txt")
	newHost := filepath.Join(hostRoot, "new.txt")

	plan, err := BuildFileSyncPlan(FileSyncState{
		ChecksumsByEnvironmentPath: map[string]string{
			"/remote/project/unchanged.txt": "sha256:same",
			"/remote/project/changed.txt":   "sha256:old",
			"/remote/project/deleted.txt":   "sha256:deleted",
		},
	}, []HostFileSnapshot{
		{HostPath: unchangedHost, Checksum: "sha256:same"},
		{HostPath: changedHost, Checksum: "sha256:new"},
		{HostPath: newHost, Checksum: "sha256:fresh"},
	}, mapper)
	if err != nil {
		t.Fatalf("BuildFileSyncPlan returned error: %v", err)
	}

	got := plan.Intents
	want := []FileSyncIntent{
		{Direction: FileSyncUpload, HostPath: filepath.Clean(changedHost), EnvironmentPath: "/remote/project/changed.txt", Checksum: "sha256:new"},
		{Direction: FileSyncUpload, HostPath: filepath.Clean(newHost), EnvironmentPath: "/remote/project/new.txt", Checksum: "sha256:fresh"},
		{Direction: FileSyncDelete, EnvironmentPath: "/remote/project/deleted.txt", Checksum: "sha256:deleted"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("plan intents = %#v, want %#v", got, want)
	}
}

func TestToolEnvironmentExecutionNormalizesEvidenceAndCommandDefaults(t *testing.T) {
	command := normalizeEnvironmentCommand(EnvironmentCommand{Command: "go test ./internal/tools"}, 45*time.Second)
	if command.Timeout != 45*time.Second {
		t.Fatalf("defaulted timeout = %s, want 45s", command.Timeout)
	}
	explicit := normalizeEnvironmentCommand(EnvironmentCommand{Command: "go test ./...", Timeout: 2 * time.Second}, 45*time.Second)
	if explicit.Timeout != 2*time.Second {
		t.Fatalf("explicit timeout = %s, want preserved 2s", explicit.Timeout)
	}

	recorded := recordedEnvironmentEvidence("fake", EnvironmentOperationExecute, "go test", "recorded command")
	if recorded.Code != EnvironmentCommandRecorded || recorded.Status != EnvironmentStatusRecorded || recorded.Backend != "fake" || recorded.Operation != "execute" || recorded.Resource != "go test" || recorded.Message != "recorded command" {
		t.Fatalf("recorded evidence = %#v, want canonical execute evidence", recorded)
	}

	cwd := cwdDeletedEnvironmentEvidence("fake", "/tmp/deleted")
	if cwd.Code != EnvironmentTerminalCWDDeleted || cwd.Status != EnvironmentStatusRecorded || cwd.Operation != "execute" || cwd.Resource != "/tmp/deleted" {
		t.Fatalf("cwd deleted evidence = %#v, want canonical cwd fallback evidence", cwd)
	}

	err := unavailableEnvironmentError("modal", EnvironmentOperationExecute, "not configured")
	evidence, ok := EnvironmentEvidenceFromError(err)
	if !ok {
		t.Fatalf("unavailableEnvironmentError returned %T, want EnvironmentEvidenceError", err)
	}
	if evidence.Code != EnvironmentBackendUnavailable || evidence.Status != EnvironmentStatusUnavailable || evidence.Backend != "modal" || evidence.Operation != "execute" || evidence.Message != "not configured" {
		t.Fatalf("unavailable evidence = %#v, want canonical unavailable execute evidence", evidence)
	}
}

func TestEnvironmentContract_UnsupportedBackendEvidence(t *testing.T) {
	env := UnsupportedEnvironment{Backend: "modal", Reason: "not configured"}
	_, err := env.Execute(context.Background(), EnvironmentCommand{Command: "true"})
	if err == nil {
		t.Fatal("Execute returned nil error")
	}
	var evidenceErr *EnvironmentEvidenceError
	if !errors.As(err, &evidenceErr) {
		t.Fatalf("Execute error type = %T, want *EnvironmentEvidenceError", err)
	}
	if evidenceErr.Evidence.Code != EnvironmentBackendUnavailable {
		t.Fatalf("evidence code = %q", evidenceErr.Evidence.Code)
	}
	if evidenceErr.Evidence.Status != EnvironmentStatusUnavailable {
		t.Fatalf("evidence status = %q", evidenceErr.Evidence.Status)
	}

	evidence, ok := EnvironmentEvidenceFromError(err)
	if !ok {
		t.Fatal("EnvironmentEvidenceFromError did not recognize the error")
	}
	if evidence.Backend != "modal" || evidence.Operation != "execute" {
		t.Fatalf("evidence = %#v", evidence)
	}
}

func operationKinds(ops []EnvironmentOperation) []EnvironmentOperationKind {
	out := make([]EnvironmentOperationKind, 0, len(ops))
	for _, op := range ops {
		out = append(out, op.Kind)
	}
	return out
}
