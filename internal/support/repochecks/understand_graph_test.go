package repochecks_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestUnderstandGraphMetadataMatchesMeta(t *testing.T) {
	repoRoot := repoRootFromCaller(t)
	graphPath := filepath.Join(repoRoot, ".understand-anything", "knowledge-graph.json")
	metaPath := filepath.Join(repoRoot, ".understand-anything", "meta.json")
	fingerprintsPath := filepath.Join(repoRoot, ".understand-anything", "fingerprints.json")

	graphRaw, err := os.ReadFile(graphPath)
	if err != nil {
		t.Fatalf("read knowledge graph: %v", err)
	}
	metaRaw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read understand meta: %v", err)
	}
	fingerprintsRaw, err := os.ReadFile(fingerprintsPath)
	if err != nil {
		t.Fatalf("read understand fingerprints: %v", err)
	}

	var graph struct {
		Project struct {
			GitCommitHash string `json:"gitCommitHash"`
		} `json:"project"`
	}
	if err := json.Unmarshal(graphRaw, &graph); err != nil {
		t.Fatalf("decode knowledge graph: %v", err)
	}
	var meta struct {
		GitCommitHash string `json:"gitCommitHash"`
		AnalyzedFiles int    `json:"analyzedFiles"`
	}
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		t.Fatalf("decode understand meta: %v", err)
	}
	var fingerprints struct {
		GitCommitHash string         `json:"gitCommitHash"`
		Files         map[string]any `json:"files"`
	}
	if err := json.Unmarshal(fingerprintsRaw, &fingerprints); err != nil {
		t.Fatalf("decode understand fingerprints: %v", err)
	}

	if graph.Project.GitCommitHash == "" {
		t.Fatal("knowledge graph project.gitCommitHash is empty")
	}
	if meta.GitCommitHash == "" {
		t.Fatal("understand meta gitCommitHash is empty")
	}
	if fingerprints.GitCommitHash == "" {
		t.Fatal("understand fingerprints gitCommitHash is empty")
	}
	if graph.Project.GitCommitHash != meta.GitCommitHash {
		t.Fatalf("understand graph commit %q does not match meta commit %q", graph.Project.GitCommitHash, meta.GitCommitHash)
	}
	if fingerprints.GitCommitHash != meta.GitCommitHash {
		t.Fatalf("understand fingerprints commit %q does not match meta commit %q", fingerprints.GitCommitHash, meta.GitCommitHash)
	}
	if meta.AnalyzedFiles <= 0 {
		t.Fatalf("understand meta analyzedFiles = %d, want positive", meta.AnalyzedFiles)
	}
	if len(fingerprints.Files) != meta.AnalyzedFiles {
		t.Fatalf("understand fingerprints file count = %d, want analyzedFiles %d", len(fingerprints.Files), meta.AnalyzedFiles)
	}
}

func repoRootFromCaller(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
