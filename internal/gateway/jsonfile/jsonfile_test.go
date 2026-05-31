package jsonfile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

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
