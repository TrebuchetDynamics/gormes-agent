package docs_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

// canonicalProgressPath is the single logical backlog as seen from the
// webpages/docs package. It may be a monolithic file or a module-keyed split
// directory; webpages/docs tests must always read it via
// canonicalProgressBytes (internal/progress.Load) so the `go test
// ./webpages/docs` gate stays green across the module-split umbrella's
// operator-gated on-disk flip (C5). Module-split umbrella prerequisite C5c.
const canonicalProgressPath = "content/building-gormes/architecture_plan/progress.json"

// canonicalProgressBytes loads the canonical backlog through the dual-layout
// internal/progress.Load (C1/C5b: os.Stat IsDir → split) and re-encodes it
// through the real exported stable on-disk encoder (SaveProgress), so callers
// receive byte-faithful canonical JSON whether the path is a monolithic file
// or a module-keyed split directory. It never assumes a single on-disk file.
func canonicalProgressBytes(t *testing.T, path string) []byte {
	t.Helper()
	p, err := progress.Load(path)
	if err != nil {
		t.Fatalf("progress.Load(%s): %v", path, err)
	}
	tmp := filepath.Join(t.TempDir(), "progress.json")
	if err := progress.SaveProgress(tmp, p); err != nil {
		t.Fatalf("progress.SaveProgress(re-encode %s): %v", path, err)
	}
	raw, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("read re-encoded canonical bytes: %v", err)
	}
	return raw
}

// TestWebpagesDocsCanonicalReadersAreSplitDirectorySafe proves the shared
// canonical-backlog reader used by the webpages/docs tests works whether the
// canonical path is a monolithic file (today) or a module-keyed split
// directory (post module-split umbrella C5), so `go test ./webpages/docs`
// stays green across the operator-gated on-disk flip. Module-split umbrella
// prerequisite C5c.
func TestWebpagesDocsCanonicalReadersAreSplitDirectorySafe(t *testing.T) {
	// Build a module-keyed split-DIRECTORY fixture from the real canonical
	// backlog without disturbing the on-disk monolith.
	p, err := progress.Load(canonicalProgressPath)
	if err != nil {
		t.Fatalf("progress.Load(monolith canonical): %v", err)
	}
	splitDir := filepath.Join(t.TempDir(), "progress.json")
	if err := progress.WriteSplitBy(splitDir, p, "module"); err != nil {
		t.Fatalf("WriteSplitBy(module): %v", err)
	}

	for _, layout := range []struct {
		name string
		path string
	}{
		{"monolith-file", canonicalProgressPath},
		{"module-split-directory", splitDir},
	} {
		t.Run(layout.name, func(t *testing.T) {
			raw := canonicalProgressBytes(t, layout.path)

			// phase5_docs_test.go sentinels survive both layouts.
			for _, want := range []string{`"5.K": {`, `"status": "complete"`} {
				if !strings.Contains(string(raw), want) {
					t.Fatalf("%s canonical bytes missing %q", layout.name, want)
				}
			}

			// upstream_coverage_test.go row-lookup survives both layouts.
			var data any
			if err := json.Unmarshal(raw, &data); err != nil {
				t.Fatalf("%s: decode canonical bytes: %v", layout.name, err)
			}
			if _, ok := findProgressRowByName(data, "Backlog split C5c: migrate webpages/docs raw progress.json readers to internal/progress.Load"); !ok {
				t.Fatalf("%s: findProgressRowByName could not locate the C5c row", layout.name)
			}
		})
	}
}
