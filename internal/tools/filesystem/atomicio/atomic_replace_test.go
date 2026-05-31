package atomicio

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAtomicReplacePreservesSymlink(t *testing.T) {
	root := t.TempDir()
	realPath := filepath.Join(root, "real.toml")
	linkPath := filepath.Join(root, "config.toml")
	if err := os.WriteFile(realPath, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write real target: %v", err)
	}
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	tmpPath := writeAtomicReplaceTemp(t, root, "new\n", 0o600)

	result, err := AtomicReplace(tmpPath, linkPath, AtomicReplaceOptions{FirstWriteMode: 0o600})
	if err != nil {
		t.Fatalf("AtomicReplace returned error: %v", err)
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link was replaced with mode %v, want symlink preserved", info.Mode())
	}
	if got := readAtomicReplaceFile(t, realPath); got != "new\n" {
		t.Fatalf("real target = %q, want updated content", got)
	}
	if got := readAtomicReplaceFile(t, linkPath); got != "new\n" {
		t.Fatalf("link target = %q, want updated content", got)
	}
	if filepath.Clean(result.Path) != filepath.Clean(realPath) {
		t.Fatalf("result path = %q, want real path %q", result.Path, realPath)
	}
	if !result.PreservedSymlink {
		t.Fatal("result did not record preserved symlink")
	}
}

func TestAtomicReplaceBrokenSymlinkCreatesTarget(t *testing.T) {
	root := t.TempDir()
	missingPath := filepath.Join(root, "missing.toml")
	linkPath := filepath.Join(root, "config.toml")
	if err := os.Symlink(missingPath, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	tmpPath := writeAtomicReplaceTemp(t, root, "created\n", 0o600)

	result, err := AtomicReplace(tmpPath, linkPath, AtomicReplaceOptions{FirstWriteMode: 0o640})
	if err != nil {
		t.Fatalf("AtomicReplace returned error: %v", err)
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link was replaced with mode %v, want symlink preserved", info.Mode())
	}
	if got := readAtomicReplaceFile(t, missingPath); got != "created\n" {
		t.Fatalf("created target = %q, want created content", got)
	}
	if filepath.Clean(result.Path) != filepath.Clean(missingPath) {
		t.Fatalf("result path = %q, want missing target %q", result.Path, missingPath)
	}
}

func TestAtomicReplacePreservesExistingPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode preservation is not portable on Windows")
	}
	root := t.TempDir()
	realPath := filepath.Join(root, "real.toml")
	linkPath := filepath.Join(root, "config.toml")
	if err := os.WriteFile(realPath, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write real target: %v", err)
	}
	if err := os.Chmod(realPath, 0o644); err != nil {
		t.Fatalf("chmod real target: %v", err)
	}
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	tmpPath := writeAtomicReplaceTemp(t, root, "new\n", 0o600)

	if _, err := AtomicReplace(tmpPath, linkPath, AtomicReplaceOptions{FirstWriteMode: 0o600}); err != nil {
		t.Fatalf("AtomicReplace returned error: %v", err)
	}

	info, err := os.Stat(realPath)
	if err != nil {
		t.Fatalf("stat real target: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("real target mode = %#o, want 0644", got)
	}
}

func writeAtomicReplaceTemp(t *testing.T, dir, content string, mode os.FileMode) string {
	t.Helper()
	f, err := os.CreateTemp(dir, ".atomic-replace-*")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	name := f.Name()
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		t.Fatalf("write temp: %v", err)
	}
	if err := f.Chmod(mode); err != nil {
		_ = f.Close()
		t.Fatalf("chmod temp: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp: %v", err)
	}
	return name
}

func readAtomicReplaceFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}

func TestAtomicWrite_Nominal(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "test.txt")
	data := []byte("hello world")

	if err := AtomicWrite(target, data); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("content = %q, want %q", got, data)
	}
	if _, err := os.Stat(target + ".tmp"); err == nil {
		t.Fatal("temp file should be gone after successful write")
	}
}

func TestAtomicWrite_PreservesExistingPerms(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(target, []byte("orig"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := AtomicWrite(target, []byte("updated")); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("perms = %o, want 0600", info.Mode().Perm())
	}
}

func TestAtomicWrite_PartialTempCleaned(t *testing.T) {
	dir := t.TempDir()
	orig := filepath.Join(dir, "orig.txt")
	if err := os.WriteFile(orig, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, "target.txt")
	if err := AtomicWrite(target, []byte("good")); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}

	got, _ := os.ReadFile(target)
	if string(got) != "good" {
		t.Fatalf("content = %q, want good", got)
	}
	if _, err := os.Stat(target + ".tmp"); err == nil {
		t.Error("temp file leaked after successful write")
	}
}

func TestAtomicWrite_NonExistentDir(t *testing.T) {
	target := filepath.Join(t.TempDir(), "missing", "test.txt")
	err := AtomicWrite(target, []byte("data"))
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}
