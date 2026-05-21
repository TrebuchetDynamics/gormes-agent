package repoctl

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestRepositorySourcePairsClassifyHermesToolTail(t *testing.T) {
	repoRoot := sourcePairsRepoRoot(t)
	validation, err := ValidateSourcePairs(SourcePairOptions{Root: repoRoot, RequireHighPriority: false})
	if err != nil {
		t.Fatalf("ValidateSourcePairs: %v", err)
	}
	pairs := map[string]SourcePair{}
	for _, pair := range validation.Manifest.Pairs {
		pairs[pair.HermesFile] = pair
	}
	for _, tc := range []struct {
		hermesFile string
		status     string
	}{
		{hermesFile: "tools/web_tools.py", status: "partial"},
		{hermesFile: "tools/x_search_tool.py", status: "partial"},
		{hermesFile: "tools/tts_tool.py", status: "partial"},
		{hermesFile: "tools/transcription_tools.py", status: "covered"},
		{hermesFile: "tools/image_generation_tool.py", status: "partial"},
		{hermesFile: "tools/url_safety.py", status: "covered"},
		{hermesFile: "tools/website_policy.py", status: "covered"},
	} {
		pair, ok := pairs[tc.hermesFile]
		if !ok {
			t.Fatalf("source pair %s missing", tc.hermesFile)
		}
		if pair.Status != tc.status {
			t.Fatalf("source pair %s status = %q, want %q", tc.hermesFile, pair.Status, tc.status)
		}
		if len(pair.GormesTargets) == 0 || len(pair.Tests) == 0 || len(pair.ProgressRows) == 0 || len(pair.UpstreamTests) == 0 {
			t.Fatalf("source pair %s lacks actionable targets/tests/rows/upstream tests: %+v", tc.hermesFile, pair)
		}
	}
}

func sourcePairsRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
