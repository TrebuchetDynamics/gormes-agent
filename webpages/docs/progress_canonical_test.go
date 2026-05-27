package docs_test

import (
	"encoding/json"
	"fmt"
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
func TestCanonicalProgressNavivoxPathsUseCurrentSiblingAppRoot(t *testing.T) {
	stale := "../navivox-app/app"
	progressRaw := string(canonicalProgressBytes(t, canonicalProgressPath))
	if strings.Contains(progressRaw, stale) {
		t.Fatalf("canonical progress still points at stale nested Navivox app root %q", stale)
	}

	featureMap := readDoc(t, "content/building-gormes/architecture_plan/hermes-honcho-feature-map.md")
	oldAbsolute := "/home/xel/git/sages-openclaw/workspace-mineru/navivox-app/app"
	for _, reject := range []string{stale, oldAbsolute} {
		if strings.Contains(featureMap, reject) {
			t.Fatalf("feature map still points at stale Navivox app root %q", reject)
		}
	}
	if !strings.Contains(featureMap, "/home/xel/git/gormes/navivox-app") {
		t.Fatalf("feature map must name the current Navivox app root")
	}
}

func TestCompletionPlanCurrentFinishLedgerMatchesProgress(t *testing.T) {
	p, err := progress.Load(canonicalProgressPath)
	if err != nil {
		t.Fatalf("progress.Load(%s): %v", canonicalProgressPath, err)
	}
	doc := readDoc(t, "content/building-gormes/architecture_plan/completion-plan.md")
	stats := p.Stats()
	expectedSummary := fmt.Sprintf("contains %s row objects: %s complete and %d planned", humanInt(stats.Items.Total), humanInt(stats.Items.Complete), stats.Items.Planned)
	if !strings.Contains(doc, expectedSummary) {
		t.Fatalf("completion-plan current ledger missing summary %q", expectedSummary)
	}
	for _, phaseID := range progressPhaseIDs(p) {
		phase := p.Phases[phaseID]
		nonComplete := 0
		for _, subphase := range phase.Subphases {
			for _, item := range subphase.Items {
				if item.Status != progress.StatusComplete {
					nonComplete++
				}
			}
		}
		expectedRow := fmt.Sprintf("| Phase %s", phaseID)
		expectedCount := fmt.Sprintf("| %d |", nonComplete)
		rowStart := strings.Index(doc, expectedRow)
		if rowStart < 0 {
			t.Fatalf("completion-plan current ledger missing row prefix %q", expectedRow)
		}
		rowEnd := strings.IndexByte(doc[rowStart:], '\n')
		if rowEnd < 0 {
			rowEnd = len(doc) - rowStart
		}
		row := doc[rowStart : rowStart+rowEnd]
		if !strings.Contains(row, expectedCount) {
			t.Fatalf("completion-plan phase %s row = %q, want count marker %q", phaseID, row, expectedCount)
		}
	}
}

func humanInt(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	out := make([]byte, 0, len(s)+len(s)/3)
	prefix := len(s) % 3
	if prefix == 0 {
		prefix = 3
	}
	out = append(out, s[:prefix]...)
	for i := prefix; i < len(s); i += 3 {
		out = append(out, ',')
		out = append(out, s[i:i+3]...)
	}
	return string(out)
}

func progressPhaseIDs(p *progress.Progress) []string {
	ids := make([]string, 0, len(p.Phases))
	for id := range p.Phases {
		ids = append(ids, id)
	}
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && ids[j] < ids[j-1]; j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
	return ids
}

func TestCanonicalProgressRowsWithActivePlannerBlockersUseStructuredBlockers(t *testing.T) {
	var data any
	if err := json.Unmarshal(canonicalProgressBytes(t, canonicalProgressPath), &data); err != nil {
		t.Fatalf("decode canonical progress: %v", err)
	}
	for _, rowName := range []string{
		"Engineering writeup #1: autonomous Hermes-porting loop",
		"TD social presence connected to blog feed",
	} {
		rowName := rowName
		t.Run(rowName, func(t *testing.T) {
			row, ok := findProgressRowByName(data, rowName)
			if !ok {
				t.Fatalf("%s row not found", rowName)
			}
			blocker, ok := row["blocker"].(map[string]any)
			if !ok {
				t.Fatalf("%s row must record a structured blocker: %#v", rowName, row["blocker"])
			}
			for _, key := range []string{"type", "status", "blocker", "evidence", "unblocks_when", "owner", "pivot", "next_check"} {
				if value, ok := blocker[key].(string); !ok || strings.TrimSpace(value) == "" {
					t.Fatalf("%s blocker missing non-empty %q: %#v", rowName, key, blocker)
				}
			}
			if blocker["status"] != "blocked" {
				t.Fatalf("%s blocker status = %q, want blocked", rowName, blocker["status"])
			}
		})
	}
}

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
