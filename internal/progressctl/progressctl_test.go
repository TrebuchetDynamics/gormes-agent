package progressctl

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

// TestValidateJSON_EmitsStatsAndCounts proves that
// `progress validate --format json` emits a parseable
// `{ok, phases, subphases: {total, ...}, items: {total, ...}}`
// document so CI dashboards and fleet monitoring can ingest the
// roadmap status without parsing the human-readable line.
func TestValidateJSON_EmitsStatsAndCounts(t *testing.T) {
	root := repoRootForTest(t)
	var buf bytes.Buffer
	if err := Validate(&buf, root, "json"); err != nil {
		t.Fatalf("Validate json: %v", err)
	}
	var got struct {
		OK        bool `json:"ok"`
		Phases    int  `json:"phases"`
		Subphases struct {
			Total      int `json:"total"`
			Complete   int `json:"complete"`
			InProgress int `json:"in_progress"`
			Planned    int `json:"planned"`
		} `json:"subphases"`
		Items struct {
			Total      int `json:"total"`
			Complete   int `json:"complete"`
			InProgress int `json:"in_progress"`
			Planned    int `json:"planned"`
		} `json:"items"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\nbody=%s", err, buf.String())
	}
	if !got.OK {
		t.Errorf("ok = false, want true")
	}
	if got.Phases < 6 {
		t.Errorf("phases = %d, want >= 6 (still addressable)", got.Phases)
	}
	if got.Subphases.Total < 50 {
		t.Errorf("subphases.total = %d, want >= 50 (floor catch-all)", got.Subphases.Total)
	}
	if got.Items.Total < 100 {
		t.Errorf("items.total = %d, want >= 100", got.Items.Total)
	}
	if got.Subphases.Complete+got.Subphases.InProgress+got.Subphases.Planned > got.Subphases.Total {
		t.Errorf("subphase derived counts (%d+%d+%d) exceed total %d",
			got.Subphases.Complete, got.Subphases.InProgress, got.Subphases.Planned, got.Subphases.Total)
	}
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	wd := filepath.Join("..", "..")
	abs, err := filepath.Abs(wd)
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

// Backlog-efficiency #1 (2026-05-16): the dead webpages/landing/src/data
// mirror is no longer generated (nothing consumes it), and the go:embed
// legacy mirror is written SLIM (status/name only) instead of a verbatim
// 5.2 MB copy, so progress edits stop producing multi-MB triple diffs.
func TestProgressPathsDropDeadSrcMirrorAndSlimLegacyEmbed(t *testing.T) {
	root := t.TempDir()
	paths := progressPaths(root)

	deadSrc := filepath.Join(root, "webpages", "landing", "src", "data", "progress.json")
	for _, path := range paths.siteProgress {
		if path == deadSrc {
			t.Fatalf("the dead src mirror %q must no longer be a generated site-progress target", deadSrc)
		}
	}

	wantSlim := filepath.Join(root, "webpages", "landing", "legacy", "go-renderer", "internal", "site", "data", "progress.json")
	if paths.siteProgressSlim != wantSlim {
		t.Fatalf("siteProgressSlim = %q, want the legacy go:embed mirror %q", paths.siteProgressSlim, wantSlim)
	}
}

// slimProgress preserves everything the landing renderer reads (phase /
// subphase names + derived statuses + Stats) while dropping per-item prose,
// so the embedded artifact is tiny and a from-clean go:embed build stays
// valid.
func TestSlimProgressPreservesRenderInputsButDropsProse(t *testing.T) {
	full := &progress.Progress{
		Phases: map[string]progress.Phase{
			"1": {Name: "Phase One", Deliverable: "d1", Subphases: map[string]progress.Subphase{
				"1.A": {Name: "Alpha", Items: []progress.Item{
					{Name: "row a", Status: progress.StatusComplete, Contract: strings.Repeat("x", 5000), Note: strings.Repeat("y", 5000)},
					{Name: "row b", Status: progress.StatusPlanned, Contract: strings.Repeat("z", 5000)},
				}},
				"1.B": {Name: "Beta", Status: progress.StatusComplete},
			}},
		},
	}

	slim := slimProgress(full)

	if !reflect.DeepEqual(full.Stats(), slim.Stats()) {
		t.Fatalf("slim must preserve Stats: full=%+v slim=%+v", full.Stats(), slim.Stats())
	}
	if slim.Phases["1"].Name != "Phase One" || slim.Phases["1"].Subphases["1.A"].Name != "Alpha" {
		t.Fatalf("slim must preserve phase/subphase names: %+v", slim.Phases["1"])
	}
	if slim.Phases["1"].Subphases["1.A"].DerivedStatus() != full.Phases["1"].Subphases["1.A"].DerivedStatus() {
		t.Fatalf("slim must preserve DerivedStatus")
	}
	for _, it := range slim.Phases["1"].Subphases["1.A"].Items {
		if it.Contract != "" || it.Note != "" || it.Name != "" {
			t.Fatalf("slim items must drop prose (name/contract/note), got %+v", it)
		}
	}
	raw, err := json.Marshal(slim)
	if err != nil {
		t.Fatalf("slim must marshal: %v", err)
	}
	if len(raw) >= len(mustMarshal(t, full)) {
		t.Fatalf("slim (%d bytes) must be smaller than full (%d bytes)", len(raw), len(mustMarshal(t, full)))
	}
}

// Backlog-efficiency #3 (2026-05-16): `progress compact` rewrites verbose
// completed-row notes to a one-line "SHIPPED … see git log — …" pointer
// through the same stable marshaller, while `validate` (and `write`) stay
// pure — they must never auto-compact. Idempotent at the subcommand level.
func TestCompactSubcommandRewritesCompletedNotesAndKeepsValidatePure(t *testing.T) {
	root := t.TempDir()
	paths := progressPaths(root)
	if err := os.MkdirAll(filepath.Dir(paths.progressJSON), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	fat := "SHIPPED 2026-05-12 (gormes-tdd-slice). " +
		strings.Repeat("Did the deep behavior with full TDD coverage. ", 80)
	seed := &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"8": {Name: "P8", Deliverable: "d", Subphases: map[string]progress.Subphase{
				"8.F": {Name: "Backlog", Items: []progress.Item{
					{Name: "done", Status: progress.StatusComplete, Note: fat},
					{Name: "planned", Status: progress.StatusPlanned, Note: fat},
				}},
			}},
		},
	}
	if err := progress.SaveProgress(paths.progressJSON, seed); err != nil {
		t.Fatalf("seed SaveProgress: %v", err)
	}

	// validate must NOT auto-compact: the canonical bytes are untouched.
	before, err := os.ReadFile(paths.progressJSON)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}
	if err := Validate(io.Discard, root, "text"); err != nil {
		t.Fatalf("validate: %v", err)
	}
	after, err := os.ReadFile(paths.progressJSON)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("validate must not rewrite progress.json (it must never auto-compact)")
	}

	// compact rewrites only the completed-row note, on disk.
	var out bytes.Buffer
	if err := Compact(&out, root); err != nil {
		t.Fatalf("compact: %v", err)
	}
	got, err := progress.Load(paths.progressJSON)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	items := got.Phases["8"].Subphases["8.F"].Items
	done := items[0].Note
	if strings.Contains(done, "\n") ||
		!strings.HasPrefix(done, "SHIPPED ") ||
		!strings.Contains(done, "see git log — ") ||
		len(done) > 200 {
		t.Fatalf("completed note must be a one-line SHIPPED pointer: %q", done)
	}
	if !strings.Contains(done, "2026-05-12") {
		t.Fatalf("date present in source note must be preserved: %q", done)
	}
	if items[1].Note != fat {
		t.Fatal("planned-row note must be left untouched on disk")
	}
	if err := Validate(io.Discard, root, "text"); err != nil {
		t.Fatalf("validate post-compact: %v", err)
	}

	// Idempotent at the subcommand level: a second pass is a reported no-op.
	out.Reset()
	if err := Compact(&out, root); err != nil {
		t.Fatalf("second compact: %v", err)
	}
	if !strings.Contains(out.String(), "already compact") {
		t.Fatalf("second compact must be a no-op, got %q", out.String())
	}
}

// Backlog split C1 (2026-05-16): `progress split <dir>` emits the canonical
// backlog as a split layout that internal/progress.Load reads back into the
// byte-identical model, without touching the canonical file. validate, write,
// and compact must never produce a split layout (purity).
func TestSplitSubcommandIsLosslessAndPure(t *testing.T) {
	root := t.TempDir()
	paths := progressPaths(root)
	if err := os.MkdirAll(filepath.Dir(paths.progressJSON), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	seed := &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"1": {Name: "P1", Deliverable: "d", Subphases: map[string]progress.Subphase{
				"1.A": {Name: "A", Items: []progress.Item{{Name: "r", Status: progress.StatusComplete}}},
			}},
			"10": {Name: "P10", Deliverable: "d", Subphases: map[string]progress.Subphase{
				"10.A": {Name: "A", Status: progress.StatusPlanned},
			}},
		},
	}
	if err := progress.SaveProgress(paths.progressJSON, seed); err != nil {
		t.Fatalf("seed SaveProgress: %v", err)
	}
	canonicalBefore, err := os.ReadFile(paths.progressJSON)
	if err != nil {
		t.Fatalf("read canonical: %v", err)
	}

	// validate / write must never emit a split layout.
	if err := Validate(io.Discard, root, "text"); err != nil {
		t.Fatalf("validate: %v", err)
	}
	_ = Write(io.Discard, root) // marker files absent in temp root; we only assert no split dir
	if _, statErr := os.Stat(filepath.Join(root, "split")); statErr == nil {
		t.Fatal("validate/write must not create a split layout (purity)")
	}

	// split <dir> emits the layout and leaves the canonical file untouched.
	splitDir := filepath.Join(t.TempDir(), "split")
	var out bytes.Buffer
	if err := Split(&out, root, splitDir); err != nil {
		t.Fatalf("Split: %v", err)
	}
	canonicalAfter, err := os.ReadFile(paths.progressJSON)
	if err != nil {
		t.Fatalf("re-read canonical: %v", err)
	}
	if !bytes.Equal(canonicalBefore, canonicalAfter) {
		t.Fatal("split must not modify the canonical progress.json")
	}

	// Load(splitDir) reconstructs the byte-identical model.
	fromSplit, err := progress.Load(splitDir)
	if err != nil {
		t.Fatalf("Load(splitDir): %v", err)
	}
	fromMono, err := progress.Load(paths.progressJSON)
	if err != nil {
		t.Fatalf("Load(mono): %v", err)
	}
	a := filepath.Join(t.TempDir(), "a.json")
	b := filepath.Join(t.TempDir(), "b.json")
	if err := progress.SaveProgress(a, fromSplit); err != nil {
		t.Fatalf("save split model: %v", err)
	}
	if err := progress.SaveProgress(b, fromMono); err != nil {
		t.Fatalf("save mono model: %v", err)
	}
	sb, _ := os.ReadFile(a)
	mb, _ := os.ReadFile(b)
	if !bytes.Equal(sb, mb) {
		t.Fatal("split-layout model must serialise byte-identically to the monolith")
	}
}

func mustMarshal(t *testing.T, p *progress.Progress) []byte {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}
