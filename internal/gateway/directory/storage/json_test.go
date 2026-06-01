package storage

import (
	"os"
	"path/filepath"
	"testing"
)

type facadeLoadValueFixture struct {
	Name string `json:"name"`
}

func TestFacadePreservesFileSpecAndLoadValueContract(t *testing.T) {
	root := t.TempDir()
	file := Spec{Name: "fixture.json", TmpPattern: ".fixture-*.tmp", Label: "fixture"}.Apply(File{Root: NewRoot(" " + root + " ")})
	if file.Root.String() != root {
		t.Fatalf("Root = %q, want trimmed temp root", file.Root.String())
	}
	if file.Name != "fixture.json" || file.TmpPattern != ".fixture-*.tmp" || file.Label != "fixture" {
		t.Fatalf("file metadata = %+v, want spec metadata", file)
	}
	if err := os.WriteFile(filepath.Join(root, "fixture.json"), []byte(`{"name":"raw"}`), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := LoadValue(file, func() facadeLoadValueFixture { return facadeLoadValueFixture{Name: "empty"} }, func(value facadeLoadValueFixture) facadeLoadValueFixture {
		value.Name += ":ensured"
		return value
	})
	if err != nil {
		t.Fatalf("LoadValue error: %v", err)
	}
	if got.Name != "raw:ensured" {
		t.Fatalf("LoadValue Name = %q, want ensured decoded value", got.Name)
	}
}
