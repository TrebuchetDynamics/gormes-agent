package plugins

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDotEnvSaveSortsAndLookupReadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	env := DotEnv{Path: path}
	if err := env.Save("Z_KEY", "last"); err != nil {
		t.Fatalf("Save Z_KEY: %v", err)
	}
	if err := env.Save("A_KEY", "first"); err != nil {
		t.Fatalf("Save A_KEY: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(body) != "A_KEY=first\nZ_KEY=last\n" {
		t.Fatalf("dotenv body = %q", body)
	}
	if got, ok := env.Lookup("A_KEY"); !ok || got != "first" {
		t.Fatalf("Lookup(A_KEY) = %q, %t; want first, true", got, ok)
	}
}

func TestReadDotEnvIgnoresCommentsAndMalformedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("# comment\nGOOD=value\nmalformed\nEMPTY=\n SPACED = trimmed \n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := ReadDotEnv(path)
	if err != nil {
		t.Fatalf("ReadDotEnv: %v", err)
	}
	want := map[string]string{"GOOD": "value", "EMPTY": "", "SPACED": "trimmed"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadDotEnv = %#v, want %#v", got, want)
	}
}
