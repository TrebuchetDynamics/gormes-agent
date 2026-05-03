package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunProgressSeedRoutesToSeedCatalog(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(&stdout, &stderr, []string{"--repo-root", t.TempDir(), "progress", "seed", "fleet"})
	if err == nil || !strings.Contains(err.Error(), "progress: read") {
		t.Fatalf("run progress seed error = %v, want progress load error proving route exists", err)
	}
}
