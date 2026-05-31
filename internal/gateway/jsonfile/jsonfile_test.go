package jsonfile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteAtomicWithOptionsUsesFilesystemPolicy(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "state")
	path := filepath.Join(root, "record.json")

	if err := WriteAtomicWithOptions(ctx, path, map[string]string{"name": "gort"}, "test record", WriteOptions{
		DirMode:    0o700,
		FileMode:   0o600,
		TmpPattern: ".custom-*.tmp",
	}); err != nil {
		t.Fatalf("WriteAtomicWithOptions: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	if !strings.HasSuffix(string(raw), "\n") {
		t.Fatalf("record %q missing trailing newline", string(raw))
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(root)
		if err != nil {
			t.Fatalf("stat root: %v", err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Fatalf("root mode = %v, want 0700", info.Mode().Perm())
		}
		info, err = os.Stat(path)
		if err != nil {
			t.Fatalf("stat file: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("file mode = %v, want 0600", info.Mode().Perm())
		}
	}
}

func TestReadDistinguishesMissingEmptyAndDecodedJSON(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "record.json")
	var out struct {
		Name string `json:"name"`
	}

	exists, err := Read(ctx, path, &out, "test record")
	if err != nil || exists {
		t.Fatalf("missing Read = exists %v err %v, want false nil", exists, err)
	}

	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("write empty: %v", err)
	}
	exists, err = Read(ctx, path, &out, "test record")
	if !exists || !errors.Is(err, ErrEmpty) {
		t.Fatalf("empty Read = exists %v err %v, want true ErrEmpty", exists, err)
	}

	if err := os.WriteFile(path, []byte(`{"name":"gort"}`), 0o600); err != nil {
		t.Fatalf("write json: %v", err)
	}
	exists, err = Read(ctx, path, &out, "test record")
	if err != nil || !exists || out.Name != "gort" {
		t.Fatalf("json Read = exists %v out %+v err %v, want decoded gort", exists, out, err)
	}
}
