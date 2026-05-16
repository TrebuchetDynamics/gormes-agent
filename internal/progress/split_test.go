package progress

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// splitFixture builds a multi-phase backlog whose phase IDs ("1", "2", "10")
// exercise the natural-numeric ordering the stable marshaller guarantees, so
// a split that round-trips through SaveProgress proves order stability too.
func splitFixture() *Progress {
	return &Progress{
		Meta: Meta{Version: "2.0"},
		Phases: map[string]Phase{
			"2": {Name: "Phase Two", Deliverable: "d2", Subphases: map[string]Subphase{
				"2.B": {Name: "Beta", Items: []Item{
					{Name: "b1", Status: StatusComplete, Note: "see git log - done"},
					{Name: "b2", Status: StatusPlanned, Priority: "P2"},
				}},
				"2.A": {Name: "Alpha", Status: StatusComplete},
			}},
			"10": {Name: "Phase Ten", Deliverable: "d10", DependencyNote: "after 9", Subphases: map[string]Subphase{
				"10.A": {Name: "Tenth", Items: []Item{
					{Name: "t1", Status: StatusInProgress, Contract: "c", ContractStatus: "draft"},
				}},
			}},
			"1": {Name: "Phase One", Deliverable: "d1", Subphases: map[string]Subphase{
				"1.A": {Name: "First", Items: []Item{{Name: "a1", Status: StatusComplete}}},
			}},
		},
	}
}

func saveBytes(t *testing.T, p *Progress) []byte {
	t.Helper()
	path := filepath.Join(t.TempDir(), "progress.json")
	if err := SaveProgress(path, p); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	return b
}

// (1) Split/Merge is a lossless, byte-stable round-trip: the reconstructed
// model deep-equals the original and serialises to byte-identical canonical
// JSON through the existing stable marshaller.
func TestSplitMergeRoundTripIsLossless(t *testing.T) {
	p := splitFixture()

	sl, err := Split(p)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	back, err := Merge(sl)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	if !reflect.DeepEqual(back, p) {
		t.Fatalf("Merge(Split(p)) must deep-equal p:\n got=%+v\nwant=%+v", back, p)
	}
	if got, want := saveBytes(t, back), saveBytes(t, p); !reflect.DeepEqual(got, want) {
		t.Fatalf("Merge(Split(p)) must be byte-stable through SaveProgress\n got %d bytes\nwant %d bytes", len(got), len(want))
	}
}

// (2) Load is layout-transparent: a monolithic file and a split directory
// written from the same model both Load into models that serialise to
// byte-identical canonical JSON.
func TestLoadDualLayoutTransparency(t *testing.T) {
	p := splitFixture()

	monoPath := filepath.Join(t.TempDir(), "progress.json")
	if err := SaveProgress(monoPath, p); err != nil {
		t.Fatalf("SaveProgress mono: %v", err)
	}
	splitDir := filepath.Join(t.TempDir(), "split")
	if err := WriteSplit(splitDir, p); err != nil {
		t.Fatalf("WriteSplit: %v", err)
	}

	fromMono, err := Load(monoPath)
	if err != nil {
		t.Fatalf("Load(mono): %v", err)
	}
	fromSplit, err := Load(splitDir)
	if err != nil {
		t.Fatalf("Load(split dir): %v", err)
	}

	if !reflect.DeepEqual(fromSplit, fromMono) {
		t.Fatalf("Load(splitDir) must equal Load(monoFile):\n split=%+v\n mono=%+v", fromSplit, fromMono)
	}
	if got, want := saveBytes(t, fromSplit), saveBytes(t, fromMono); !reflect.DeepEqual(got, want) {
		t.Fatalf("both layouts must serialise byte-identically (%d vs %d bytes)", len(got), len(want))
	}
}

// (3) A malformed split layout fails with a distinguishable typed error,
// while the monolithic path keeps working unchanged.
func TestLoadMalformedSplitReturnsTypedError(t *testing.T) {
	p := splitFixture()

	splitDir := filepath.Join(t.TempDir(), "split")
	if err := WriteSplit(splitDir, p); err != nil {
		t.Fatalf("WriteSplit: %v", err)
	}
	// Corrupt the layout: remove the index manifest.
	if err := os.Remove(filepath.Join(splitDir, "index.json")); err != nil {
		t.Fatalf("rm index: %v", err)
	}

	if _, err := Load(splitDir); err == nil {
		t.Fatal("Load of a split dir without index.json must error, got nil")
	} else if !errors.Is(err, ErrMalformedSplit) {
		t.Fatalf("error must be ErrMalformedSplit, got %v", err)
	}

	// Monolith still works regardless.
	monoPath := filepath.Join(t.TempDir(), "progress.json")
	if err := SaveProgress(monoPath, p); err != nil {
		t.Fatalf("SaveProgress mono: %v", err)
	}
	if _, err := Load(monoPath); err != nil {
		t.Fatalf("monolithic Load must still work: %v", err)
	}
}

// assertStillSplit fails unless dir is a directory carrying the split index
// (i.e. the write preserved the split layout instead of collapsing it into a
// single monolithic file).
func assertStillSplit(t *testing.T, dir string) {
	t.Helper()
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		t.Fatalf("split layout collapsed: %q is not a directory (err=%v)", dir, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "index.json")); err != nil {
		t.Fatalf("split index missing after write: %v", err)
	}
}

// (C3.1) SaveProgress against a split-layout directory round-trips back into
// the split layout — it never collapses the split into a monolithic file,
// and the result is byte-stable through the canonical marshaller.
func TestSaveProgressPreservesSplitLayout(t *testing.T) {
	splitDir := filepath.Join(t.TempDir(), "split")
	if err := WriteSplit(splitDir, splitFixture()); err != nil {
		t.Fatalf("WriteSplit: %v", err)
	}

	mutated := splitFixture()
	mutated.Phases["1"].Subphases["1.A"].Items[0].Note = "C3 mutation"

	if err := SaveProgress(splitDir, mutated); err != nil {
		t.Fatalf("SaveProgress(splitDir): %v", err)
	}
	assertStillSplit(t, splitDir)

	got, err := Load(splitDir)
	if err != nil {
		t.Fatalf("Load(splitDir) after save: %v", err)
	}
	if got.Phases["1"].Subphases["1.A"].Items[0].Note != "C3 mutation" {
		t.Fatalf("mutation not persisted into the split layout, got %q",
			got.Phases["1"].Subphases["1.A"].Items[0].Note)
	}
	if a, b := saveBytes(t, got), saveBytes(t, mutated); !reflect.DeepEqual(a, b) {
		t.Fatalf("split-preserving save must be byte-stable through the canonical marshaller (%d vs %d bytes)", len(a), len(b))
	}
}

// (C3.2) The builderloop write path (ApplyHealthUpdates) preserves the split
// layout for both the normal SaveProgress branch and the single-empty-health
// raw-splice branch (which must not byte-splice a directory).
func TestApplyHealthUpdatesPreservesSplitLayout(t *testing.T) {
	t.Run("normal health mutation", func(t *testing.T) {
		splitDir := filepath.Join(t.TempDir(), "split")
		if err := WriteSplit(splitDir, splitFixture()); err != nil {
			t.Fatalf("WriteSplit: %v", err)
		}
		err := ApplyHealthUpdates(splitDir, []HealthUpdate{{
			PhaseID: "1", SubphaseID: "1.A", ItemName: "a1",
			Mutate: func(h *RowHealth) { h.AttemptCount = 3; h.ConsecutiveFailures = 2 },
		}})
		if err != nil {
			t.Fatalf("ApplyHealthUpdates(splitDir): %v", err)
		}
		assertStillSplit(t, splitDir)
		got, err := Load(splitDir)
		if err != nil {
			t.Fatalf("Load after ApplyHealthUpdates: %v", err)
		}
		h := got.Phases["1"].Subphases["1.A"].Items[0].Health
		if h == nil || h.AttemptCount != 3 || h.ConsecutiveFailures != 2 {
			t.Fatalf("health update not persisted into split layout: %+v", h)
		}
	})

	t.Run("single empty-health update (raw-splice branch)", func(t *testing.T) {
		splitDir := filepath.Join(t.TempDir(), "split")
		if err := WriteSplit(splitDir, splitFixture()); err != nil {
			t.Fatalf("WriteSplit: %v", err)
		}
		// A single update whose Mutate leaves Health zero-valued triggers
		// the insertEmptyHealthBlock raw-byte-splice path in the monolith
		// case; on a split layout it must fall back to the typed write.
		err := ApplyHealthUpdates(splitDir, []HealthUpdate{{
			PhaseID: "1", SubphaseID: "1.A", ItemName: "a1",
			Mutate: func(h *RowHealth) {},
		}})
		if err != nil {
			t.Fatalf("ApplyHealthUpdates empty-health(splitDir): %v", err)
		}
		assertStillSplit(t, splitDir)
		got, err := Load(splitDir)
		if err != nil {
			t.Fatalf("Load after empty-health update: %v", err)
		}
		if got.Phases["1"].Subphases["1.A"].Items[0].Health == nil {
			t.Fatal("empty health block must be present after split-layout update")
		}
	})
}

// (C3.3) The monolithic path is byte-for-byte unchanged: a regular file stays
// a regular file and a fresh (non-existent) path still creates a monolith.
func TestSaveProgressMonolithDefaultUnchanged(t *testing.T) {
	p := splitFixture()
	fresh := filepath.Join(t.TempDir(), "progress.json")
	if err := SaveProgress(fresh, p); err != nil {
		t.Fatalf("SaveProgress(fresh file): %v", err)
	}
	fi, err := os.Stat(fresh)
	if err != nil || fi.IsDir() {
		t.Fatalf("fresh path must be a monolithic regular file, got dir=%v err=%v", fi != nil && fi.IsDir(), err)
	}
	got, err := Load(fresh)
	if err != nil {
		t.Fatalf("Load monolith: %v", err)
	}
	if !reflect.DeepEqual(saveBytes(t, got), saveBytes(t, p)) {
		t.Fatal("monolithic SaveProgress behavior must be byte-for-byte unchanged")
	}
}
