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

func TestReadRejectsBlankPathWithoutReadingCwdFile(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.WriteFile(" ", []byte(`{"name":"leaked"}`), 0o600); err != nil {
		t.Fatalf("write cwd space file: %v", err)
	}
	var out struct {
		Name string `json:"name"`
	}

	exists, err := Read(context.Background(), " ", &out, "test record")
	if err == nil {
		t.Fatal("Read blank path err = nil, want error")
	}
	if exists || out.Name != "" {
		t.Fatalf("Read blank path exists=%v out=%+v, want no decoded cwd file", exists, out)
	}
}

func TestReadAllowsNilContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "record.json")
	if err := os.WriteFile(path, []byte(`{"name":"gort"}`), 0o600); err != nil {
		t.Fatalf("write record: %v", err)
	}
	var out struct {
		Name string `json:"name"`
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Read panicked with nil context: %v", r)
		}
	}()

	exists, err := Read(nil, path, &out, "test record")
	if err != nil || !exists || out.Name != "gort" {
		t.Fatalf("Read nil context = exists %v out %+v err %v, want decoded gort", exists, out, err)
	}
}

func TestWriteAtomicWithOptionsRejectsBlankPathBeforeCreatingFile(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	writerCalled := false

	err := WriteAtomicWithOptions(context.Background(), " ", map[string]string{"name": "gort"}, "test record", WriteOptions{
		Writer: func(string, []byte, os.FileMode) error {
			writerCalled = true
			return nil
		},
	})
	if err == nil {
		t.Fatal("WriteAtomicWithOptions blank path err = nil, want error")
	}
	if writerCalled {
		t.Fatal("writer was called for blank path")
	}
	entries, readErr := os.ReadDir(root)
	if readErr != nil {
		t.Fatalf("read cwd: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("blank path created cwd entries: %+v", entries)
	}
}

func TestWriteAtomicWithOptionsAllowsNilContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "record.json")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("WriteAtomicWithOptions panicked with nil context: %v", r)
		}
	}()

	if err := WriteAtomicWithOptions(nil, path, map[string]string{"name": "gort"}, "test record", WriteOptions{FileMode: 0o600}); err != nil {
		t.Fatalf("WriteAtomicWithOptions nil context: %v", err)
	}
	var out struct {
		Name string `json:"name"`
	}
	if _, err := Read(context.Background(), path, &out, "test record"); err != nil || out.Name != "gort" {
		t.Fatalf("written record out=%+v err=%v, want decoded gort", out, err)
	}
}

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

func TestWriteRawAtomicWithOptionsWritesPreEncodedPayload(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "record.json")

	if err := WriteRawAtomicWithOptions(ctx, path, []byte("{\"name\":\"raw\"}\n"), "raw record", WriteOptions{
		FileMode:   0o600,
		TmpPattern: ".raw-*.tmp",
		Sync:       true,
	}); err != nil {
		t.Fatalf("WriteRawAtomicWithOptions: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	if string(raw) != "{\"name\":\"raw\"}\n" {
		t.Fatalf("record = %q, want pre-encoded payload", string(raw))
	}
}

func TestWriteAtomicWithOptionsUsesInjectedWriterBeforeRename(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "record.json")
	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatalf("write old record: %v", err)
	}

	if err := WriteAtomicWithOptions(ctx, path, map[string]string{"name": "new"}, "test record", WriteOptions{
		FileMode:   0o600,
		TmpPattern: ".custom-*.tmp",
		Writer: func(tmpPath string, data []byte, perm os.FileMode) error {
			if perm != 0o600 {
				t.Fatalf("writer perm = %v, want 0600", perm)
			}
			if !strings.Contains(string(data), `"name": "new"`) {
				t.Fatalf("writer data = %q, want marshaled payload", string(data))
			}
			if err := os.WriteFile(tmpPath, []byte(`{"name":`), perm); err != nil {
				return err
			}
			return os.ErrInvalid
		},
	}); err == nil {
		t.Fatal("WriteAtomicWithOptions error = nil, want injected failure")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	if string(raw) != "old\n" {
		t.Fatalf("record = %q, want old record preserved", string(raw))
	}
}

func TestWriteAtomicWithOptionsHonorsContextCanceledDuringInjectedWriter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	root := t.TempDir()
	path := filepath.Join(root, "record.json")
	if err := os.WriteFile(path, []byte("old\n"), 0o600); err != nil {
		t.Fatalf("write old record: %v", err)
	}

	err := WriteAtomicWithOptions(ctx, path, map[string]string{"name": "new"}, "test record", WriteOptions{
		FileMode:   0o600,
		TmpPattern: ".custom-*.tmp",
		Writer: func(tmpPath string, data []byte, perm os.FileMode) error {
			if err := os.WriteFile(tmpPath, data, perm); err != nil {
				return err
			}
			cancel()
			return nil
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteAtomicWithOptions error = %v, want context.Canceled", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	if string(raw) != "old\n" {
		t.Fatalf("record = %q, want old record preserved after cancellation", string(raw))
	}
}

func TestReadRequiredReturnsNotExistForMissingFile(t *testing.T) {
	var out struct{}
	if err := ReadRequired(context.Background(), filepath.Join(t.TempDir(), "missing.json"), &out, "test record"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadRequired error = %v, want os.ErrNotExist", err)
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
