package main

import (
	"path/filepath"
	"testing"
)

func TestTargetPaths_UsesCurrentRepoReadme(t *testing.T) {
	progressPath, readmePath, docsIndexPath := targetPaths("/tmp/work/gormes")

	if progressPath != filepath.Join("/tmp/work/gormes", "docs", "content", "building-gormes", "architecture_plan", "progress.json") {
		t.Fatalf("progressPath = %q", progressPath)
	}
	if readmePath != filepath.Join("/tmp/work/gormes", "README.md") {
		t.Fatalf("readmePath = %q, want README inside repo root", readmePath)
	}
	if docsIndexPath != filepath.Join("/tmp/work/gormes", "docs", "content", "building-gormes", "architecture_plan", "_index.md") {
		t.Fatalf("docsIndexPath = %q", docsIndexPath)
	}
}
