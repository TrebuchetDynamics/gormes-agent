package fileops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWithDefaultsFillsMissingOperations(t *testing.T) {
	ops := WithDefaults(Ops{})
	if ops.MkdirAll == nil || ops.ReadFile == nil || ops.WriteFile == nil || ops.CopyFile == nil {
		t.Fatalf("WithDefaults(Ops{}) = %+v, want all operations filled", ops)
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "source.txt")
	dst := filepath.Join(dir, "copy.txt")
	if err := os.WriteFile(src, []byte("body"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ops.CopyFile(src, dst); err != nil {
		t.Fatalf("default CopyFile failed: %v", err)
	}
	body, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "body" {
		t.Fatalf("copied body = %q, want body", body)
	}
}

func TestWithDefaultsPreservesProvidedOperations(t *testing.T) {
	called := false
	customRead := func(string) ([]byte, error) {
		called = true
		return []byte("custom"), nil
	}
	ops := WithDefaults(Ops{ReadFile: customRead})
	body, err := ops.ReadFile("ignored")
	if err != nil {
		t.Fatal(err)
	}
	if !called || string(body) != "custom" {
		t.Fatalf("custom ReadFile called=%v body=%q, want custom operation preserved", called, body)
	}
}

func TestBackupPathUsesGormesSuffix(t *testing.T) {
	got := BackupPath("/tmp/keybindings.json")
	if !strings.HasPrefix(got, "/tmp/keybindings.json.gormes-backup-") {
		t.Fatalf("BackupPath() = %q, want gormes backup suffix", got)
	}
}
