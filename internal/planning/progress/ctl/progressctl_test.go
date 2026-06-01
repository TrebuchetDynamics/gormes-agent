package progressctl

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
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
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatalf("repo root not found from %s", wd)
		}
		wd = parent
	}
}

// Backlog-efficiency #1 (2026-05-16) removed the dead Astro src/data
// progress mirror. The retired Go renderer no longer receives a generated
// go:embed mirror either, so progress edits stay out of the landing package.
func TestProgressPathsDropLandingProgressMirrors(t *testing.T) {
	root := t.TempDir()
	paths := progressPaths(root)

	if len(paths.siteProgress) != 0 {
		t.Fatalf("landing site progress mirrors must stay empty, got %#v", paths.siteProgress)
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

// Backlog split C2 (2026-05-16): the docs/landing generators resolve the
// canonical backlog through C1's dual-layout Load — they work identically
// whether the source is the monolithic progress.json or a split directory,
// the monolith stays the default, and a malformed split surfaces a typed
// error rather than a half-generated doc set.

func c2Fixture() *progress.Progress {
	return &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"1": {Name: "P1", Deliverable: "d1", Subphases: map[string]progress.Subphase{
				"1.A": {Name: "A", Items: []progress.Item{
					{Name: "r1", Status: progress.StatusComplete},
					{Name: "r2", Status: progress.StatusPlanned, Priority: "P2"},
				}},
			}},
			"10": {Name: "P10", Deliverable: "d10", Subphases: map[string]progress.Subphase{
				"10.A": {Name: "A", Status: progress.StatusPlanned},
			}},
		},
	}
}

func seedMonolith(t *testing.T, root string, p *progress.Progress) {
	t.Helper()
	mono := progressPaths(root).progressJSON
	if err := os.MkdirAll(filepath.Dir(mono), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := progress.SaveProgress(mono, p); err != nil {
		t.Fatalf("seed monolith: %v", err)
	}
}

func seedSplitOnly(t *testing.T, root string, p *progress.Progress) {
	t.Helper()
	splitDir := progressPaths(root).progressJSON
	if err := os.MkdirAll(filepath.Dir(splitDir), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := progress.WriteSplit(splitDir, p); err != nil {
		t.Fatalf("seed split: %v", err)
	}
}

func validateJSON(t *testing.T, root string) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := Validate(&buf, root, "json"); err != nil {
		t.Fatalf("Validate(%s): %v", root, err)
	}
	return buf.Bytes()
}

func seedSplitModule(t *testing.T, root string, p *progress.Progress) {
	t.Helper()
	splitDir := progressPaths(root).progressJSON
	if err := os.MkdirAll(filepath.Dir(splitDir), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := progress.WriteSplitBy(splitDir, p, "module"); err != nil {
		t.Fatalf("seed module-keyed split: %v", err)
	}
}

func emitBytes(t *testing.T, root string) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := Emit(&buf, root); err != nil {
		t.Fatalf("Emit(%s): %v", root, err)
	}
	return buf.Bytes()
}

// `progress emit` is the pure read-only split-safe seam gormes-* skill
// discovery pipes through (module-split umbrella C5d). Its stdout must be the
// byte-identical merged canonical backlog whether the canonical is a
// monolithic file, a phase-keyed split directory, or a module-keyed split
// directory — so the skill jq pipelines survive the operator-gated C5 flip.
func TestEmitIsSplitDirectorySafeAndCanonical(t *testing.T) {
	p := c2Fixture()

	monoRoot := t.TempDir()
	seedMonolith(t, monoRoot, p)

	splitPhaseRoot := t.TempDir()
	seedSplitOnly(t, splitPhaseRoot, p) // phase-keyed split dir, no monolith

	splitModRoot := t.TempDir()
	seedSplitModule(t, splitModRoot, p) // module-keyed split dir, no monolith

	b0 := emitBytes(t, monoRoot)
	b1 := emitBytes(t, splitPhaseRoot)
	b2 := emitBytes(t, splitModRoot)

	if !bytes.Equal(b0, b1) {
		t.Fatalf("emit must be byte-identical for monolith vs phase-split:\n mono=%s\n split=%s", b0, b1)
	}
	if !bytes.Equal(b0, b2) {
		t.Fatalf("emit must be byte-identical for monolith vs module-split:\n mono=%s\n split=%s", b0, b2)
	}

	// Emit is byte-faithful to the canonical on-disk monolith bytes.
	wantMono, err := os.ReadFile(progressPaths(monoRoot).progressJSON)
	if err != nil {
		t.Fatalf("read seeded monolith: %v", err)
	}
	if !bytes.Equal(b0, wantMono) {
		t.Fatalf("emit must equal the canonical monolith bytes:\n emit=%s\n file=%s", b0, wantMono)
	}

	// Emit stdout is valid JSON that round-trips to the same backlog.
	var got progress.Progress
	if err := json.Unmarshal(b0, &got); err != nil {
		t.Fatalf("emit output is not valid JSON: %v", err)
	}
	if !reflect.DeepEqual(&got, p) {
		t.Fatalf("emit output did not round-trip to the fixture backlog:\n got=%#v\n want=%#v", &got, p)
	}
}

func TestCurrentCanonicalModuleSplitDirectoryEmitsByteIdentically(t *testing.T) {
	root := repoRootForTest(t)
	p, err := loadValidProgress(root)
	if err != nil {
		t.Fatalf("load real canonical progress: %v", err)
	}
	monoBytes := emitBytes(t, root)

	splitRoot := t.TempDir()
	splitCanonical := progressPaths(splitRoot).progressJSON
	if err := os.MkdirAll(filepath.Dir(splitCanonical), 0o755); err != nil {
		t.Fatalf("mkdir split canonical parent: %v", err)
	}
	if err := progress.WriteSplitBy(splitCanonical, p, "module"); err != nil {
		t.Fatalf("WriteSplitBy(module at canonical path): %v", err)
	}
	fi, err := os.Stat(splitCanonical)
	if err != nil {
		t.Fatalf("stat split canonical path: %v", err)
	}
	if !fi.IsDir() {
		t.Fatalf("future canonical path must be a split directory, got regular file: %s", splitCanonical)
	}

	splitBytes := emitBytes(t, splitRoot)
	if !bytes.Equal(monoBytes, splitBytes) {
		t.Fatalf("module-keyed canonical directory must emit byte-identically to the current canonical monolith (%d vs %d bytes)", len(splitBytes), len(monoBytes))
	}
}

// Backlog split C5h (2026-05-16): `progress list --module <feature>` is a
// read-only view over the one logical backlog. It must produce the same module
// inventory from the monolith and from the future module-keyed split layout.
func TestListModuleIsSplitSafeReadOnlyAndScoped(t *testing.T) {
	p := &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"1": {Name: "P1", Deliverable: "d1", Subphases: map[string]progress.Subphase{
				"1.A": {Name: "A", Items: []progress.Item{
					{Name: "provider planned", Priority: "P1", Status: progress.StatusPlanned, Module: progress.ModuleProviders},
					{Name: "tts planned", Priority: "P2", Status: progress.StatusPlanned, Module: progress.ModuleTTS},
				}},
			}},
			"2": {Name: "P2", Deliverable: "d2", Subphases: map[string]progress.Subphase{
				"2.A": {Name: "A", Items: []progress.Item{
					{Name: "provider complete", Priority: "P2", Status: progress.StatusComplete, Module: progress.ModuleProviders},
				}},
			}},
		},
	}

	monoRoot := t.TempDir()
	seedMonolith(t, monoRoot, p)
	before, err := os.ReadFile(progressPaths(monoRoot).progressJSON)
	if err != nil {
		t.Fatalf("read monolith before list: %v", err)
	}

	splitRoot := t.TempDir()
	seedSplitModule(t, splitRoot, p)

	var monoOut, splitOut bytes.Buffer
	if err := List(&monoOut, monoRoot, ListOptions{Module: progress.ModuleProviders}); err != nil {
		t.Fatalf("List monolith: %v", err)
	}
	if err := List(&splitOut, splitRoot, ListOptions{Module: progress.ModuleProviders}); err != nil {
		t.Fatalf("List module split: %v", err)
	}
	if !bytes.Equal(monoOut.Bytes(), splitOut.Bytes()) {
		t.Fatalf("module list must be identical for monolith and module split:\nmono=%s\nsplit=%s", monoOut.String(), splitOut.String())
	}

	got := monoOut.String()
	for _, want := range []string{
		"progress: module providers (2 rows)\n",
		"phase\tsubphase\tstatus\tpriority\tname\n",
		"1\t1.A\tplanned\tP1\tprovider planned\n",
		"2\t2.A\tcomplete\tP2\tprovider complete\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("module list missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "tts planned") {
		t.Fatalf("module list must not broaden outside selected module:\n%s", got)
	}
	after, err := os.ReadFile(progressPaths(monoRoot).progressJSON)
	if err != nil {
		t.Fatalf("read monolith after list: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("module list must be read-only and leave the canonical monolith untouched")
	}
}

func TestNextWorkBuildDecisionUsesBuilderLoopCandidateSelectionAndIsSplitSafe(t *testing.T) {
	p := &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"3": {Name: "P3", Deliverable: "d3", Subphases: map[string]progress.Subphase{
				"3.E": {Name: "E", Items: []progress.Item{
					{Name: "normal row", Status: progress.StatusPlanned, Contract: "normal contract", ContractStatus: progress.ContractStatusDraft, SliceSize: progress.SliceSizeSmall, NoTestRequiredReason: "fixture", Module: progress.ModuleProgress},
					{Name: "p0 row", Priority: "P0", Status: progress.StatusPlanned, Contract: "p0 contract", ContractStatus: progress.ContractStatusDraft, SliceSize: progress.SliceSizeSmall, NoTestRequiredReason: "fixture", Module: progress.ModuleProgress},
				}},
			}},
		},
	}

	monoRoot := t.TempDir()
	seedMonolith(t, monoRoot, p)
	before, err := os.ReadFile(progressPaths(monoRoot).progressJSON)
	if err != nil {
		t.Fatalf("read monolith before next-work: %v", err)
	}

	splitRoot := t.TempDir()
	seedSplitModule(t, splitRoot, p)

	var monoOut, splitOut bytes.Buffer
	if err := NextWork(&monoOut, monoRoot); err != nil {
		t.Fatalf("NextWork monolith: %v", err)
	}
	if err := NextWork(&splitOut, splitRoot); err != nil {
		t.Fatalf("NextWork split: %v", err)
	}
	if !bytes.Equal(monoOut.Bytes(), splitOut.Bytes()) {
		t.Fatalf("next-work output must be identical for monolith and module split:\nmono=%s\nsplit=%s", monoOut.String(), splitOut.String())
	}

	got := monoOut.String()
	for _, want := range []string{
		"progress: next-work builder-ready (2 candidates)\n",
		"decision=build\n",
		"phase=3\n",
		"subphase=3.E\n",
		"name=p0 row\n",
		"reason=P0 handoff\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("next-work output missing %q:\n%s", want, got)
		}
	}

	after, err := os.ReadFile(progressPaths(monoRoot).progressJSON)
	if err != nil {
		t.Fatalf("read monolith after next-work: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("next-work must be read-only and leave the canonical monolith untouched")
	}
}

func TestNextWorkAndGeneratedNextSlicesAgreeOnCompletedBlocker(t *testing.T) {
	p := &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"1": {Name: "P1", Deliverable: "d1", Subphases: map[string]progress.Subphase{
				"1.A": {Name: "A", Items: []progress.Item{
					{Name: "Foundation", Status: progress.StatusComplete, Module: progress.ModuleProgress},
					{
						Name:           "Dependent",
						Status:         progress.StatusPlanned,
						Contract:       "dependent contract",
						ContractStatus: progress.ContractStatusFixtureReady,
						SliceSize:      progress.SliceSizeSmall,
						ExecutionOwner: progress.ExecutionOwnerTools,
						TrustClass:     []string{"system"},
						Fixture:        "internal/progress/ctl/progressctl_test.go",
						BlockedBy:      []string{"Foundation"},
						ReadyWhen:      []string{"Foundation is complete"},
						Acceptance:     []string{"Dependent is selected"},
						WriteScope:     []string{"internal/progress/ctl/"},
						TestCommands:   []string{"go test ./dependent"},
						DoneSignal:     []string{"dependent selected"},
						Module:         progress.ModuleProgress,
					},
				}},
			}},
		},
	}
	root := t.TempDir()
	seedMonolith(t, root, p)
	seedWriteMarkerFiles(t, root)

	var nextOut bytes.Buffer
	if err := NextWork(&nextOut, root); err != nil {
		t.Fatalf("NextWork: %v", err)
	}
	if !strings.Contains(nextOut.String(), "name=Dependent\n") {
		t.Fatalf("next-work should select completed-blocker dependent row:\n%s", nextOut.String())
	}

	if err := Write(io.Discard, root); err != nil {
		t.Fatalf("Write: %v", err)
	}
	nextSlices := mustReadFile(t, progressPaths(root).nextSlices)
	if !strings.Contains(nextSlices, "Dependent") || !strings.Contains(nextSlices, "dependent contract") {
		t.Fatalf("generated next-slices should include the same dependent row:\n%s", nextSlices)
	}
}

func TestNextWorkRepoOnlyFiltersCrossRootWriteScope(t *testing.T) {
	p := &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"9": {Name: "P9", Deliverable: "d9", Subphases: map[string]progress.Subphase{
				"9.F": {Name: "F", Items: []progress.Item{
					{Name: "cross root row", Priority: "P0", Status: progress.StatusPlanned, Contract: "cross root contract", ContractStatus: progress.ContractStatusFixtureReady, SliceSize: progress.SliceSizeSmall, NoTestRequiredReason: "fixture", WriteScope: []string{"../navivox-app/app/lib/"}, Module: progress.ModuleNavivox},
					{Name: "local row", Priority: "P1", Status: progress.StatusPlanned, Contract: "local contract", ContractStatus: progress.ContractStatusFixtureReady, SliceSize: progress.SliceSizeSmall, NoTestRequiredReason: "fixture", WriteScope: []string{"cmd/progress/main.go", "internal/progress/ctl/"}, Module: progress.ModuleProgress},
				}},
			}},
		},
	}
	root := t.TempDir()
	seedMonolith(t, root, p)

	var defaultOut bytes.Buffer
	if err := NextWork(&defaultOut, root); err != nil {
		t.Fatalf("NextWork default: %v", err)
	}
	if !strings.Contains(defaultOut.String(), "name=cross root row\n") {
		t.Fatalf("default next-work should preserve builder ranking before scope filtering:\n%s", defaultOut.String())
	}

	var scopedOut bytes.Buffer
	if err := NextWorkWithOptions(&scopedOut, root, NextWorkOptions{RepoOnly: true}); err != nil {
		t.Fatalf("NextWork repo-only: %v", err)
	}
	got := scopedOut.String()
	for _, want := range []string{
		"progress: next-work builder-ready (1 candidate)\n",
		"decision=build\n",
		"scope=repo\n",
		"name=local row\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("repo-only next-work missing %q:\n%s", want, got)
		}
	}
}

func TestNextWorkRepoOnlyPlansWhenAllCandidatesEscapeRepo(t *testing.T) {
	p := &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"9": {Name: "P9", Deliverable: "d9", Subphases: map[string]progress.Subphase{
				"9.F": {Name: "F", Items: []progress.Item{
					{Name: "sibling row", Priority: "P1", Status: progress.StatusPlanned, Contract: "sibling contract", ContractStatus: progress.ContractStatusFixtureReady, SliceSize: progress.SliceSizeSmall, NoTestRequiredReason: "fixture", WriteScope: []string{"../navivox-app/app/lib/"}, Module: progress.ModuleNavivox},
					{Name: "separate repo row", Priority: "P1", Status: progress.StatusPlanned, Contract: "separate repo contract", ContractStatus: progress.ContractStatusFixtureReady, SliceSize: progress.SliceSizeSmall, NoTestRequiredReason: "fixture", WriteScope: []string{"(separate repo) README.md"}, Module: progress.ModuleDocs},
				}},
			}},
		},
	}
	root := t.TempDir()
	seedMonolith(t, root, p)

	var out bytes.Buffer
	if err := NextWorkWithOptions(&out, root, NextWorkOptions{RepoOnly: true}); err != nil {
		t.Fatalf("NextWork repo-only empty: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"progress: next-work no in-repo builder-ready rows\n",
		"decision=plan\n",
		"scope=repo\n",
		"reason=no unblocked builder-ready rows within repo write scope\n",
		"planner_action=split or repair one row whose write_scope stays under the repo root\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("repo-only plan output missing %q:\n%s", want, got)
		}
	}
}

func TestNextWorkPlanDecisionWhenNoBuilderReadyRows(t *testing.T) {
	p := &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"8": {Name: "P8", Deliverable: "d8", Subphases: map[string]progress.Subphase{
				"8.F": {Name: "F", Items: []progress.Item{
					{Name: "complete row", Status: progress.StatusComplete, Contract: "done", ContractStatus: progress.ContractStatusValidated, SliceSize: progress.SliceSizeSmall, NoTestRequiredReason: "fixture", Module: progress.ModuleProgress},
				}},
			}},
		},
	}
	root := t.TempDir()
	seedMonolith(t, root, p)

	var out bytes.Buffer
	if err := NextWork(&out, root); err != nil {
		t.Fatalf("NextWork empty: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"progress: next-work no builder-ready rows\n",
		"decision=plan\n",
		"reason=no unblocked builder-ready rows\n",
		"planner_action=repair one planned/draft row until it satisfies the handoff contract\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("next-work plan output missing %q:\n%s", want, got)
		}
	}
}

func TestListModuleRejectsUnknownAndMultiModuleFilters(t *testing.T) {
	root := t.TempDir()
	seedMonolith(t, root, c2Fixture())

	err := List(io.Discard, root, ListOptions{Module: "providers,tts"})
	if err == nil || !strings.Contains(err.Error(), "exactly one module") {
		t.Fatalf("comma-separated modules must fail clearly, got %v", err)
	}

	err = List(io.Discard, root, ListOptions{Module: "not-a-module"})
	if err == nil || !strings.Contains(err.Error(), "unknown module") || !strings.Contains(err.Error(), progress.ModuleProviders) {
		t.Fatalf("unknown modules must fail with allowed module guidance, got %v", err)
	}
}

func TestWriteRendersModuleRoadmapPagesFromMonolithAndModuleSplit(t *testing.T) {
	p := &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"1": {Name: "P1", Deliverable: "d1", Subphases: map[string]progress.Subphase{
				"1.A": {Name: "A", Items: []progress.Item{
					{Name: "provider row", Priority: "P1", Status: progress.StatusPlanned, Module: progress.ModuleProviders},
					{Name: "tts row", Priority: "P2", Status: progress.StatusPlanned, Module: progress.ModuleTTS},
				}},
			}},
		},
	}

	monoRoot := t.TempDir()
	seedMonolith(t, monoRoot, p)
	seedWriteMarkerFiles(t, monoRoot)

	splitRoot := t.TempDir()
	seedSplitModule(t, splitRoot, p)
	seedWriteMarkerFiles(t, splitRoot)

	if err := Write(io.Discard, monoRoot); err != nil {
		t.Fatalf("Write monolith: %v", err)
	}
	if err := Write(io.Discard, splitRoot); err != nil {
		t.Fatalf("Write split: %v", err)
	}

	monoProviders := mustReadFile(t, filepath.Join(progressPaths(monoRoot).moduleRoadmapsDir, progress.ModuleRoadmapRelPath(progress.ModuleProviders)))
	splitProviders := mustReadFile(t, filepath.Join(progressPaths(splitRoot).moduleRoadmapsDir, progress.ModuleRoadmapRelPath(progress.ModuleProviders)))
	if monoProviders != splitProviders {
		t.Fatalf("module page must be identical for monolith and module split:\nmono=%s\nsplit=%s", monoProviders, splitProviders)
	}
	for _, want := range []string{
		`title: "Providers Module Roadmap"`,
		"**Module:** `providers`",
		"| `planned` | `P1` | `providers` | provider row |",
	} {
		if !strings.Contains(monoProviders, want) {
			t.Fatalf("providers page missing %q:\n%s", want, monoProviders)
		}
	}
	if strings.Contains(monoProviders, "tts row") {
		t.Fatalf("providers page must not contain tts rows:\n%s", monoProviders)
	}

	index := mustReadFile(t, filepath.Join(progressPaths(monoRoot).moduleRoadmapsDir, "_index.md"))
	if !strings.Contains(index, "| [Providers](provider-models/providers/) | 1 | 0 | 0 | 1 | `P1`: 1 |") || !strings.Contains(index, "## Provider Models") {
		t.Fatalf("module index missing provider counts:\n%s", index)
	}
}

// (1) A root backed ONLY by a split layout generates byte-identical output
// to a root backed only by the monolith.
func TestProgressctlResolvesSplitLayoutForGenerators(t *testing.T) {
	p := c2Fixture()

	monoRoot := t.TempDir()
	seedMonolith(t, monoRoot, p)

	splitRoot := t.TempDir()
	seedSplitOnly(t, splitRoot, p) // no monolithic progress.json at all

	if got, want := validateJSON(t, splitRoot), validateJSON(t, monoRoot); !bytes.Equal(got, want) {
		t.Fatalf("split-backed validate JSON must equal monolith-backed:\n split=%s\n mono=%s", got, want)
	}

	// The loaded model itself must be byte-identical through SaveProgress,
	// which transitively guarantees every generator (pure fn of the model).
	mp, err := loadValidProgress(monoRoot)
	if err != nil {
		t.Fatalf("loadValidProgress mono: %v", err)
	}
	sp, err := loadValidProgress(splitRoot)
	if err != nil {
		t.Fatalf("loadValidProgress split: %v", err)
	}
	a := filepath.Join(t.TempDir(), "a.json")
	b := filepath.Join(t.TempDir(), "b.json")
	if err := progress.SaveProgress(a, mp); err != nil {
		t.Fatal(err)
	}
	if err := progress.SaveProgress(b, sp); err != nil {
		t.Fatal(err)
	}
	ab, _ := os.ReadFile(a)
	bb, _ := os.ReadFile(b)
	if !bytes.Equal(ab, bb) {
		t.Fatal("split-resolved model must serialise byte-identically to the monolith")
	}
}

// (2) The canonical path remains architecture_plan/progress.json. That path
// may be a monolithic file or a split directory; the old progress.split staging
// directory is ignored so it cannot become a second backlog.
func TestProgressctlCanonicalSourceRetiresProgressSplit(t *testing.T) {
	p := c2Fixture()
	root := t.TempDir()
	seedMonolith(t, root, p)

	paths := progressPaths(root)
	if got := canonicalSource(root); got != paths.progressJSON {
		t.Fatalf("canonicalSource must be progress.json path, got %q want %q", got, paths.progressJSON)
	}
	staleSplit := filepath.Join(filepath.Dir(paths.progressJSON), "progress.split")
	if err := progress.WriteSplit(staleSplit, p); err != nil {
		t.Fatalf("seed stale progress.split: %v", err)
	}
	if got := canonicalSource(root); got != paths.progressJSON {
		t.Fatalf("stale progress.split must not win canonical resolution, got %q want %q", got, paths.progressJSON)
	}
	if _, err := loadValidProgress(root); err != nil {
		t.Fatalf("loadValidProgress with ignored stale progress.split must still use progress.json: %v", err)
	}
}

// (3) A malformed split layout surfaces ErrMalformedSplit (never a
// half-generated doc set); a monolith-only root is unaffected.
func TestProgressctlMalformedSplitSurfacesTypedError(t *testing.T) {
	p := c2Fixture()

	badRoot := t.TempDir()
	seedSplitOnly(t, badRoot, p)
	if err := os.Remove(filepath.Join(progressPaths(badRoot).progressJSON, "index.json")); err != nil {
		t.Fatalf("corrupt split: %v", err)
	}
	var buf bytes.Buffer
	if err := Validate(&buf, badRoot, "json"); err == nil {
		t.Fatal("Validate against a malformed split layout must error, got nil")
	} else if !errors.Is(err, progress.ErrMalformedSplit) {
		t.Fatalf("error must be progress.ErrMalformedSplit, got %v", err)
	}

	okRoot := t.TempDir()
	seedMonolith(t, okRoot, p)
	if err := Validate(io.Discard, okRoot, "json"); err != nil {
		t.Fatalf("monolith-only root must still validate: %v", err)
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

func seedWriteMarkerFiles(t *testing.T, root string) {
	t.Helper()
	paths := progressPaths(root)
	markers := map[string]string{
		paths.docsIndex:          "docs-full-checklist",
		paths.readme:             "readme-rollup",
		paths.contractReadiness:  "contract-readiness",
		paths.builderLoopHandoff: "builder-loop-handoff",
		paths.agentQueue:         "agent-queue",
		paths.nextSlices:         "next-slices",
		paths.blockedSlices:      "blocked-slices",
		paths.umbrellaCleanup:    "umbrella-cleanup",
		paths.progressSchema:     "progress-schema",
	}
	for path, kind := range markers {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		body := "before\n<!-- PROGRESS:START kind=" + kind + " -->\nold\n<!-- PROGRESS:END -->\nafter\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write marker %s: %v", path, err)
		}
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}
