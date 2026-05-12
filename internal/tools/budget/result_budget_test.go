package budget

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestToolResultBudget_TruncatesTextAndPersistsArtifact proves text output
// exceeding the budget is truncated, the full bytes are written to a sanitized
// artifact path under OutputDir, and the pointer string contains the artifact
// relative path plus a head preview.
func TestToolResultBudget_TruncatesTextAndPersistsArtifact(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(strings.Repeat("a", 4096))
	cfg := ToolResultBudgetConfig{
		OutputDir:       dir,
		TextBudgetBytes: 256,
		PreviewBytes:    64,
	}

	pointer, evidence, err := FormatToolResult(cfg, raw, "text/plain")
	if err != nil {
		t.Fatalf("FormatToolResult: %v", err)
	}

	if evidence.Artifact == "" {
		t.Fatalf("evidence.Artifact empty; want a relative artifact path")
	}
	if filepath.IsAbs(evidence.Artifact) {
		t.Fatalf("evidence.Artifact = %q must be relative to OutputDir", evidence.Artifact)
	}
	if strings.Contains(evidence.Artifact, "..") {
		t.Fatalf("evidence.Artifact = %q must not contain ..", evidence.Artifact)
	}
	if !strings.Contains(pointer, evidence.Artifact) {
		t.Fatalf("pointer %q does not reference artifact %q", pointer, evidence.Artifact)
	}
	if len(pointer) >= len(raw) {
		t.Fatalf("pointer length %d should be much smaller than raw length %d", len(pointer), len(raw))
	}
	if evidence.Bytes != len(raw) {
		t.Fatalf("evidence.Bytes = %d, want %d", evidence.Bytes, len(raw))
	}

	full, err := os.ReadFile(filepath.Join(dir, evidence.Artifact))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if string(full) != string(raw) {
		t.Fatalf("artifact bytes do not match raw input: got %d bytes, want %d", len(full), len(raw))
	}

	// Preview should be at most PreviewBytes long and a prefix of raw.
	if len(evidence.Preview) > cfg.PreviewBytes+64 {
		t.Fatalf("preview length %d exceeds PreviewBytes+slack", len(evidence.Preview))
	}
	if !strings.HasPrefix(string(raw), evidence.Preview[:min(len(evidence.Preview), cfg.PreviewBytes)]) {
		// preview head must be a prefix of the raw bytes
		t.Fatalf("preview %q is not a prefix of raw output", evidence.Preview)
	}
}

// TestToolResultBudget_PersistsJSONNonText proves non-text/JSON output is
// persisted as a JSON file and the pointer is short (no embedded JSON body).
func TestToolResultBudget_PersistsJSONNonText(t *testing.T) {
	dir := t.TempDir()
	payload := map[string]any{
		"items": make([]int, 0, 1024),
		"large": strings.Repeat("z", 2048),
	}
	for i := 0; i < 1024; i++ {
		payload["items"] = append(payload["items"].([]int), i)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	cfg := ToolResultBudgetConfig{
		OutputDir:       dir,
		TextBudgetBytes: 128,
		PreviewBytes:    32,
	}

	pointer, evidence, err := FormatToolResult(cfg, raw, "application/json")
	if err != nil {
		t.Fatalf("FormatToolResult: %v", err)
	}

	if evidence.Artifact == "" {
		t.Fatalf("evidence.Artifact empty; want a relative artifact path")
	}
	if !strings.HasSuffix(evidence.Artifact, ".json") {
		t.Fatalf("evidence.Artifact = %q; want a .json extension for JSON media", evidence.Artifact)
	}
	if strings.Contains(pointer, "\"large\"") || strings.Contains(pointer, strings.Repeat("z", 64)) {
		t.Fatalf("pointer %q embeds JSON payload; pointer must be short", pointer)
	}

	full, err := os.ReadFile(filepath.Join(dir, evidence.Artifact))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	var roundtrip map[string]any
	if err := json.Unmarshal(full, &roundtrip); err != nil {
		t.Fatalf("artifact is not valid JSON: %v", err)
	}
	if _, ok := roundtrip["large"]; !ok {
		t.Fatalf("artifact JSON missing 'large' field; got keys %v", roundtrip)
	}
}

// TestToolResultBudget_TruncatedEvidence proves the evidence code marks
// truncation when the raw output exceeds the text budget.
func TestToolResultBudget_TruncatedEvidence(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(strings.Repeat("x", 8192))
	cfg := ToolResultBudgetConfig{
		OutputDir:       dir,
		TextBudgetBytes: 512,
		PreviewBytes:    128,
	}

	_, evidence, err := FormatToolResult(cfg, raw, "text/plain")
	if err != nil {
		t.Fatalf("FormatToolResult: %v", err)
	}

	if evidence.Code != "tool_output_truncated" {
		t.Fatalf("evidence.Code = %q, want tool_output_truncated", evidence.Code)
	}
}

// TestToolResultBudget_PersistedEvidence proves evidence reports persisted
// when the helper has written the artifact file successfully.
func TestToolResultBudget_PersistedEvidence(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(strings.Repeat("y", 8192))
	cfg := ToolResultBudgetConfig{
		OutputDir:       dir,
		TextBudgetBytes: 256,
		PreviewBytes:    32,
	}

	_, evidence, err := FormatToolResult(cfg, raw, "text/plain")
	if err != nil {
		t.Fatalf("FormatToolResult: %v", err)
	}
	if evidence.Artifact == "" {
		t.Fatalf("evidence.Artifact empty; want a relative artifact path")
	}
	if _, statErr := os.Stat(filepath.Join(dir, evidence.Artifact)); statErr != nil {
		t.Fatalf("artifact not persisted: %v", statErr)
	}
	// Persisted evidence must be observable; truncated case includes persisted state too.
	if evidence.Code != "tool_output_truncated" && evidence.Code != "tool_output_persisted" {
		t.Fatalf("evidence.Code = %q, want tool_output_truncated or tool_output_persisted", evidence.Code)
	}
}

// TestToolResultBudget_PersistenceFailedEvidence proves the helper degrades
// gracefully (returns truncated text inline + persistence_failed evidence)
// when the OutputDir cannot be written.
func TestToolResultBudget_PersistenceFailedEvidence(t *testing.T) {
	parent := t.TempDir()
	// Build a path that points "through" a regular file, so MkdirAll/WriteFile
	// cannot create a directory there.
	blocker := filepath.Join(parent, "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("seed blocker file: %v", err)
	}
	bad := filepath.Join(blocker, "artifacts")

	raw := []byte(strings.Repeat("p", 4096))
	cfg := ToolResultBudgetConfig{
		OutputDir:       bad,
		TextBudgetBytes: 256,
		PreviewBytes:    64,
	}

	pointer, evidence, err := FormatToolResult(cfg, raw, "text/plain")
	if err != nil {
		t.Fatalf("FormatToolResult must degrade, not return error; got %v", err)
	}
	if evidence.Code != "tool_output_persistence_failed" {
		t.Fatalf("evidence.Code = %q, want tool_output_persistence_failed", evidence.Code)
	}
	if evidence.Artifact != "" {
		t.Fatalf("evidence.Artifact = %q, want empty when persistence failed", evidence.Artifact)
	}
	if pointer == "" {
		t.Fatalf("pointer empty; want degraded inline truncated text")
	}
	if len(pointer) > cfg.TextBudgetBytes*4 {
		t.Fatalf("pointer length %d should stay bounded under degraded path", len(pointer))
	}

	for _, secret := range []string{"sk-", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "Bearer "} {
		if strings.Contains(pointer, secret) {
			t.Fatalf("pointer leaks secret token %q", secret)
		}
	}
}

// TestToolResultBudget_PathSanitization proves artifact filenames never
// contain ".." and never resolve outside OutputDir.
func TestToolResultBudget_PathSanitization(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(strings.Repeat("s", 8192))
	cfg := ToolResultBudgetConfig{
		OutputDir:       dir,
		TextBudgetBytes: 256,
		PreviewBytes:    32,
	}

	for i := 0; i < 8; i++ {
		_, evidence, err := FormatToolResult(cfg, raw, "text/plain")
		if err != nil {
			t.Fatalf("FormatToolResult: %v", err)
		}
		if evidence.Artifact == "" {
			t.Fatalf("evidence.Artifact empty")
		}
		if filepath.IsAbs(evidence.Artifact) {
			t.Fatalf("artifact path %q must be relative", evidence.Artifact)
		}
		if strings.Contains(evidence.Artifact, "..") {
			t.Fatalf("artifact path %q must not contain ..", evidence.Artifact)
		}

		full := filepath.Join(dir, evidence.Artifact)
		cleaned := filepath.Clean(full)
		if !strings.HasPrefix(cleaned+string(filepath.Separator), filepath.Clean(dir)+string(filepath.Separator)) &&
			cleaned != filepath.Clean(dir) {
			t.Fatalf("artifact %q escapes OutputDir %q", cleaned, dir)
		}
		if _, err := os.Stat(full); err != nil {
			t.Fatalf("artifact missing on disk: %v", err)
		}
	}
}
