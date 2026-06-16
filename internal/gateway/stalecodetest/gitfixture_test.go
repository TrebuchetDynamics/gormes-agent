package stalecodetest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteNormalGitHEAD(t *testing.T) {
	root := t.TempDir()
	WriteNormalGitHEAD(t, root, "development", "abc123")
	for _, path := range []string{
		filepath.Join(root, ".git", "HEAD"),
		filepath.Join(root, ".git", "refs", "heads", "development"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}

func TestWriteFileCreatesParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "file.txt")
	WriteFile(t, path, "body")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if string(body) != "body" {
		t.Fatalf("fixture body = %q, want body", body)
	}
}
