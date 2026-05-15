package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunExportWritesStaticSite(t *testing.T) {
	out := filepath.Join(t.TempDir(), "dist")

	if err := run([]string{"export", "--out", out}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	for _, rel := range []string{"index.html", filepath.Join("static", "site.css")} {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Fatalf("export missing %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "install.sh")); !os.IsNotExist(err) {
		t.Fatalf("export should not publish install.sh: %v", err)
	}
}
