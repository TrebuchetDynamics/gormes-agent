package persisted

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpecAppliesDefaultFileMetadataWhilePreservingRoot(t *testing.T) {
	spec := Spec{Name: "channel_directory.json", TmpPattern: ".channel_directory-*.tmp", Label: "channel directory"}

	file := spec.Apply(File{Root: NewRoot(" /tmp/gormes ")})

	if file.Root.String() != "/tmp/gormes" {
		t.Fatalf("Root = %q, want trimmed root", file.Root.String())
	}
	if file.Name != "channel_directory.json" || file.TmpPattern != ".channel_directory-*.tmp" || file.Label != "channel directory" {
		t.Fatalf("file metadata = %+v, want spec metadata", file)
	}
}

func TestFileWriteRejectsControlCharactersInFileNameAndTmpPattern(t *testing.T) {
	root := t.TempDir()
	for name, file := range map[string]File{
		"file name C0":   {Root: NewRoot(root), Name: "directory\n.json", TmpPattern: ".directory-*.tmp", Label: "directory"},
		"file name C1":   {Root: NewRoot(root), Name: "directory\u009b.json", TmpPattern: ".directory-*.tmp", Label: "directory"},
		"tmp pattern C0": {Root: NewRoot(root), Name: "directory.json", TmpPattern: ".directory-\n*.tmp", Label: "directory"},
		"tmp pattern C1": {Root: NewRoot(root), Name: "directory.json", TmpPattern: ".directory-\u009b*.tmp", Label: "directory"},
	} {
		t.Run(name, func(t *testing.T) {
			called := false
			err := file.WriteAtomic(map[string]bool{"ok": true}, func(path string, raw []byte, mode os.FileMode) error {
				called = true
				return os.WriteFile(path, raw, mode)
			})
			if err == nil {
				t.Fatal("WriteAtomic with control-character path metadata succeeded, want validation error")
			}
			if !strings.Contains(err.Error(), "invalid") {
				t.Fatalf("WriteAtomic error = %v, want invalid metadata validation", err)
			}
			if called {
				t.Fatal("writer was called for control-character path metadata")
			}
		})
	}
}

func TestFileWriteRejectsEscapingTmpPattern(t *testing.T) {
	root := t.TempDir()
	called := false

	err := (File{Root: NewRoot(root), Name: "directory.json", TmpPattern: "../directory-*.tmp", Label: "directory"}).WriteAtomic(map[string]bool{"ok": true}, func(path string, raw []byte, mode os.FileMode) error {
		called = true
		if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(root)+string(os.PathSeparator)) {
			t.Fatalf("writer received temp path outside root: %s", path)
		}
		return os.WriteFile(path, raw, mode)
	})
	if err == nil {
		t.Fatal("WriteAtomic with escaping temp pattern succeeded, want validation error")
	}
	if !strings.Contains(err.Error(), "directory temp pattern is invalid") {
		t.Fatalf("WriteAtomic error = %v, want invalid temp pattern validation", err)
	}
	if called {
		t.Fatal("writer was called for invalid temp pattern")
	}
}

func TestFileWriteRejectsEscapingFileName(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "escaped-directory.json")

	err := (File{Root: NewRoot(root), Name: "../escaped-directory.json", TmpPattern: ".directory-*.tmp", Label: "directory"}).WriteAtomic(map[string]bool{"escaped": true}, nil)
	if err == nil {
		t.Fatal("WriteAtomic with escaping file name succeeded, want validation error")
	}
	if !strings.Contains(err.Error(), "directory file name is invalid") {
		t.Fatalf("WriteAtomic error = %v, want invalid file name validation", err)
	}
	if _, statErr := os.Stat(outside); statErr == nil {
		t.Fatalf("WriteAtomic created file outside root at %s", outside)
	} else if !os.IsNotExist(statErr) {
		t.Fatalf("Stat outside file: %v", statErr)
	}
}

func TestFileReadRequiresConfiguredRoot(t *testing.T) {
	tmp := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir temp: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()
	if err := os.WriteFile(filepath.Join(tmp, "directory.json"), []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("seed cwd file: %v", err)
	}

	var got map[string]bool
	err = (File{Name: "directory.json", Label: "directory"}).Read(&got)
	if err == nil {
		t.Fatalf("Read with empty root succeeded with %#v, want root validation error", got)
	}
	if !strings.Contains(err.Error(), "directory root is empty") {
		t.Fatalf("Read error = %v, want empty root validation", err)
	}
}

func TestSpecApplyKeepsConfiguredFile(t *testing.T) {
	spec := Spec{Name: "default.json", TmpPattern: ".default-*.tmp", Label: "default"}
	configured := File{Root: NewRoot("/tmp/custom"), Name: "custom.json", TmpPattern: ".custom-*.tmp", Label: "custom"}

	if got := spec.Apply(configured); got != configured {
		t.Fatalf("Apply configured = %+v, want %+v", got, configured)
	}
}
