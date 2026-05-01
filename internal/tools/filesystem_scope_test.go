package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilesystemScope_CWDOnly(t *testing.T) {
	tmp := t.TempDir()
	scope := NewFilesystemScope(tmp, nil, nil)
	if !scope.CWDOnly {
		t.Fatal("expected CWDOnly=true when no paths configured")
	}

	result := scope.CheckRead("file.txt", tmp)
	if !result.Allowed {
		t.Fatal("expected file in cwd to be allowed")
	}

	result = scope.CheckRead("/etc/passwd", tmp)
	if result.Allowed {
		t.Fatal("expected file outside cwd to be denied")
	}
	if result.Evidence != "filesystem_read_scope_violation" {
		t.Fatalf("expected filesystem_read_scope_violation, got %s", result.Evidence)
	}
}

func TestFilesystemScope_AllowedReadPaths(t *testing.T) {
	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	os.MkdirAll(dataDir, 0755)

	scope := NewFilesystemScope(tmp, []string{dataDir}, nil)
	if scope.CWDOnly {
		t.Fatal("expected CWDOnly=false when paths configured")
	}

	result := scope.CheckRead("data/file.txt", tmp)
	if !result.Allowed {
		t.Fatal("expected file in allowed path to be allowed")
	}

	result = scope.CheckRead("/etc/passwd", tmp)
	if result.Allowed {
		t.Fatal("expected file outside allowed path to be denied")
	}
}

func TestFilesystemScope_AllowedWritePaths(t *testing.T) {
	tmp := t.TempDir()
	writeDir := filepath.Join(tmp, "output")
	os.MkdirAll(writeDir, 0755)

	scope := NewFilesystemScope(tmp, nil, []string{writeDir})

	result := scope.CheckWrite("output/result.txt", tmp)
	if !result.Allowed {
		t.Fatal("expected file in allowed write path to be allowed")
	}

	result = scope.CheckWrite("data/file.txt", tmp)
	if result.Allowed {
		t.Fatal("expected file outside allowed write path to be denied")
	}
	if result.Evidence != "filesystem_write_scope_violation" {
		t.Fatalf("expected filesystem_write_scope_violation, got %s", result.Evidence)
	}
}

func TestFilesystemScope_PathNormalization(t *testing.T) {
	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	os.MkdirAll(dataDir, 0755)

	scope := NewFilesystemScope(tmp, []string{dataDir}, nil)

	result := scope.CheckRead("data/../data/file.txt", tmp)
	if !result.Allowed {
		t.Fatal("expected normalized path to be allowed")
	}
	if !filepath.IsAbs(result.Normalized) {
		t.Fatalf("expected normalized path to be absolute, got %s", result.Normalized)
	}
}

func TestFilesystemScope_EmptyAllowedMeansAllowAll(t *testing.T) {
	tmp := t.TempDir()
	scope := NewFilesystemScope(tmp, []string{}, []string{})
	if scope.CWDOnly {
		t.Fatal("expected CWDOnly=false when empty slices configured")
	}

	result := scope.CheckRead("/etc/passwd", tmp)
	if !result.Allowed {
		t.Fatal("expected empty allowed paths to allow all")
	}
}

func TestFilesystemScope_DoctorReport(t *testing.T) {
	tmp := t.TempDir()
	scope := NewFilesystemScope(tmp, []string{"/data"}, []string{"/output"})
	report := scope.GetDoctorReport(tmp)
	config, ok := report["filesystem_scope_config"].(map[string]interface{})
	if !ok {
		t.Fatal("expected filesystem_scope_config map")
	}
	if config["cwd_only"].(bool) {
		t.Fatal("expected cwd_only=false")
	}
	if config["cwd"].(string) != tmp {
		t.Fatalf("expected cwd=%s, got %s", tmp, config["cwd"])
	}
}
