package gatewaytest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFixtureJPEG(t *testing.T) {
	dir := t.TempDir()
	path := WriteFixtureJPEG(t, dir, "sample.jpg", 10, 20, 30)
	if path != filepath.Join(dir, "sample.jpg") {
		t.Fatalf("path = %q, want file in temp dir", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("fixture jpeg is empty")
	}
}
