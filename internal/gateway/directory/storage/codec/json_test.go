package codec

import (
	"os"
	"path/filepath"
	"testing"
)

type jsonFixture struct {
	Name string `json:"name"`
}

func TestReadJSONDecodesRequiredFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "fixture.json")
	if err := os.WriteFile(path, []byte(`{"name":"raw"}`), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var got jsonFixture
	if err := ReadJSON(path, &got); err != nil {
		t.Fatalf("ReadJSON error: %v", err)
	}
	if got.Name != "raw" {
		t.Fatalf("Name = %q, want raw", got.Name)
	}
}
