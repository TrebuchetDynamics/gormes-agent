package workspace

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

func TestWorkspaceCanonicalSourceIsProgressJSONAndIgnoresProgressSplit(t *testing.T) {
	root := t.TempDir()
	ws := New(root)
	if got := ws.CanonicalSource(); got != ws.Paths.ProgressJSON {
		t.Fatalf("CanonicalSource() = %q, want progress.json path %q", got, ws.Paths.ProgressJSON)
	}

	if err := os.MkdirAll(filepath.Join(filepath.Dir(ws.Paths.ProgressJSON), "progress.split"), 0o755); err != nil {
		t.Fatalf("mkdir stale progress.split: %v", err)
	}
	if got := ws.CanonicalSource(); got != ws.Paths.ProgressJSON {
		t.Fatalf("stale progress.split must not win canonical resolution: got %q want %q", got, ws.Paths.ProgressJSON)
	}
}

func TestWorkspaceEmitBytesMatchesMonolithAndModuleSplit(t *testing.T) {
	p := workspaceFixture()

	monoRoot := t.TempDir()
	mono := New(monoRoot)
	if err := os.MkdirAll(filepath.Dir(mono.Paths.ProgressJSON), 0o755); err != nil {
		t.Fatalf("mkdir monolith parent: %v", err)
	}
	if err := progress.SaveProgress(mono.Paths.ProgressJSON, p); err != nil {
		t.Fatalf("seed monolith: %v", err)
	}

	splitRoot := t.TempDir()
	split := New(splitRoot)
	if err := os.MkdirAll(filepath.Dir(split.Paths.ProgressJSON), 0o755); err != nil {
		t.Fatalf("mkdir split parent: %v", err)
	}
	if err := progress.WriteSplitBy(split.Paths.ProgressJSON, p, "module"); err != nil {
		t.Fatalf("seed split: %v", err)
	}

	monoBytes, err := mono.EmitBytes()
	if err != nil {
		t.Fatalf("monolith EmitBytes: %v", err)
	}
	splitBytes, err := split.EmitBytes()
	if err != nil {
		t.Fatalf("split EmitBytes: %v", err)
	}
	if !bytes.Equal(monoBytes, splitBytes) {
		t.Fatalf("EmitBytes must match for monolith and module split:\nmono=%s\nsplit=%s", monoBytes, splitBytes)
	}
}

func TestWorkspaceLoadValidReportsMalformedCanonicalSplit(t *testing.T) {
	root := t.TempDir()
	ws := New(root)
	if err := os.MkdirAll(ws.Paths.ProgressJSON, 0o755); err != nil {
		t.Fatalf("mkdir canonical split dir: %v", err)
	}

	_, err := ws.LoadValid()
	if err == nil {
		t.Fatal("LoadValid against malformed canonical split must fail")
	}
	if !errors.Is(err, progress.ErrMalformedSplit) {
		t.Fatalf("LoadValid error = %v, want ErrMalformedSplit", err)
	}
}

func TestWorkspaceLoadValidRoundTripsMonolith(t *testing.T) {
	p := workspaceFixture()
	root := t.TempDir()
	ws := New(root)
	if err := os.MkdirAll(filepath.Dir(ws.Paths.ProgressJSON), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := progress.SaveProgress(ws.Paths.ProgressJSON, p); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := ws.LoadValid()
	if err != nil {
		t.Fatalf("LoadValid: %v", err)
	}
	if !reflect.DeepEqual(got, p) {
		t.Fatalf("LoadValid = %#v, want %#v", got, p)
	}
}

func workspaceFixture() *progress.Progress {
	return &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"1": {Name: "P1", Deliverable: "d1", Subphases: map[string]progress.Subphase{
				"1.A": {Name: "A", Items: []progress.Item{
					{Name: "done", Status: progress.StatusComplete, Module: progress.ModuleProgress},
					{Name: "todo", Status: progress.StatusPlanned, Module: progress.ModuleProgress},
				}},
			}},
		},
	}
}
