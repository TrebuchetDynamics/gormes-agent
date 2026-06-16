package stalecodetest

import (
	"os"
	"path/filepath"
	"testing"
)

func WriteNormalGitHEAD(t *testing.T, root, branch, sha string) {
	t.Helper()
	WriteFile(t, filepath.Join(root, ".git", "HEAD"), "ref: refs/heads/"+branch+"\n")
	WriteFile(t, filepath.Join(root, ".git", "refs", "heads", branch), sha+"\n")
}

func WriteFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
