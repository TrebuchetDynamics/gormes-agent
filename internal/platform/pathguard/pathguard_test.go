package pathguard

import (
	"path/filepath"
	"testing"
)

func TestCleanRelativeRejectsUnsafePaths(t *testing.T) {
	for _, path := range []string{"", "   ", ".", "/abs", "../escape", "nested/../../escape"} {
		if got, err := CleanRelative(path); err == nil {
			t.Fatalf("CleanRelative(%q) = %q, nil err; want reject", path, got)
		}
	}
}

func TestCleanRelativeNormalizesSafePaths(t *testing.T) {
	got, err := CleanRelative(" assets/../skills/SKILL.md ")
	if err != nil {
		t.Fatalf("CleanRelative returned error: %v", err)
	}
	want := filepath.Join("skills", "SKILL.md")
	if got != want {
		t.Fatalf("CleanRelative = %q, want %q", got, want)
	}
}

func TestWithin(t *testing.T) {
	root := t.TempDir()
	if !Within(root, filepath.Join(root, "child", "file.txt")) {
		t.Fatal("child path should be within root")
	}
	if Within(root, filepath.Dir(root)) {
		t.Fatal("parent path should not be within root")
	}
}

func TestJoinRelative(t *testing.T) {
	root := t.TempDir()
	got, err := JoinRelative(root, "a/../b.txt")
	if err != nil {
		t.Fatalf("JoinRelative returned error: %v", err)
	}
	want := filepath.Join(root, "b.txt")
	if got != want {
		t.Fatalf("JoinRelative = %q, want %q", got, want)
	}
}
