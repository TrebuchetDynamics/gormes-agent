package repoctl

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMaintenanceSurfaceHasNoPythonScripts(t *testing.T) {
	root := repoRootForTest(t)
	matches, err := filepath.Glob(filepath.Join(root, "scripts", "*.py"))
	if err != nil {
		t.Fatalf("glob scripts/*.py: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("repo maintenance scripts must be Go or shell, found Python files: %v", matches)
	}
}

func TestRuntimeSurfaceHasNoPybridgePackage(t *testing.T) {
	root := repoRootForTest(t)
	if _, err := os.Stat(filepath.Join(root, "internal", "pybridge")); err == nil {
		t.Fatalf("internal/pybridge must not exist; keep runtime extension seams Python-neutral")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat internal/pybridge: %v", err)
	}
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
