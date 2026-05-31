package storage

import (
	"os"
	"path/filepath"
	"testing"
)

type loadValueFixture struct {
	Name string `json:"name"`
}

func TestLoadValueUsesEmptyValueAndEnsureHook(t *testing.T) {
	root := t.TempDir()
	file := Spec{Name: "fixture.json", TmpPattern: ".fixture-*.tmp", Label: "fixture"}.File(root)
	if err := os.WriteFile(filepath.Join(root, "fixture.json"), []byte(`{"name":"raw"}`), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := LoadValue(file, func() loadValueFixture { return loadValueFixture{Name: "empty"} }, func(value loadValueFixture) loadValueFixture {
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

func TestLoadValueReturnsFreshEmptyValueOnReadError(t *testing.T) {
	file := Spec{Name: "missing.json", TmpPattern: ".missing-*.tmp", Label: "missing"}.File(t.TempDir())

	got, err := LoadValue(file, func() loadValueFixture { return loadValueFixture{Name: "empty"} }, nil)
	if err == nil {
		t.Fatal("LoadValue error = nil, want read error")
	}
	if got.Name != "empty" {
		t.Fatalf("LoadValue value = %+v, want fresh empty value", got)
	}
}
