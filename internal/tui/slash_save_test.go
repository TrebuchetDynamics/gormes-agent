package tui

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/transcript"
)

// recordingExportFunc captures invocations of SessionExportFunc and returns
// the configured path/error or invokes a side-effect (used by the partial
// file test to seed a real file that the handler must os.Remove).
type recordingExportFunc struct {
	calls   int
	gotCtx  context.Context
	gotID   string
	beforeReturn func()
	path    string
	err     error
}

func (r *recordingExportFunc) call(ctx context.Context, sessionID string) (string, error) {
	r.calls++
	r.gotCtx = ctx
	r.gotID = sessionID
	if r.beforeReturn != nil {
		r.beforeReturn()
	}
	return r.path, r.err
}

func newSaveTestModel(t *testing.T, history []hermes.Message, frameSessionID string, fn SessionExportFunc, sub Submitter) Model {
	t.Helper()
	frames := make(chan kernel.RenderFrame, 1)
	if sub == nil {
		sub = func(string) {}
	}
	m := NewModelWithOptions(frames, sub, func() {}, Options{
		MouseTracking: true,
		SessionExport: fn,
	})
	m.frame.History = history
	m.frame.SessionID = frameSessionID
	return m
}

func TestSlashSave_NoConversationReturnsStatus(t *testing.T) {
	rec := &recordingExportFunc{path: "/should/not/be/used.md"}
	sub := &nopSubmitter{}
	m := newSaveTestModel(t, nil, "sess-1", rec.call, sub.submit)

	res := saveSlashHandler("/save", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true (slash MUST be consumed even with empty history)")
	}
	if res.StatusMessage != "save: no conversation" {
		t.Fatalf("StatusMessage = %q, want %q", res.StatusMessage, "save: no conversation")
	}
	if rec.calls != 0 {
		t.Fatalf("SessionExportFunc called %d times, want 0 (must short-circuit before export)", rec.calls)
	}
	if sub.calls != 0 {
		t.Fatalf("Submit called %d times, want 0 (slash must never reach kernel)", sub.calls)
	}
}

func TestSlashSave_NoActiveSessionReturnsStatus(t *testing.T) {
	history := []hermes.Message{{Role: "user", Content: "hi"}}
	rec := &recordingExportFunc{path: "/should/not/be/used.md"}
	sub := &nopSubmitter{}
	m := newSaveTestModel(t, history, "", rec.call, sub.submit)

	res := saveSlashHandler("/save", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if res.StatusMessage != "save: no active session" {
		t.Fatalf("StatusMessage = %q, want %q", res.StatusMessage, "save: no active session")
	}
	if rec.calls != 0 {
		t.Fatalf("SessionExportFunc called %d times, want 0", rec.calls)
	}
	if sub.calls != 0 {
		t.Fatalf("Submit called %d times, want 0", sub.calls)
	}
}

func TestSlashSave_HappyPathReturnsWrittenPath(t *testing.T) {
	const wantPath = "/tmp/sess-export-fixture.md"
	history := []hermes.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "ack"},
	}
	rec := &recordingExportFunc{path: wantPath}
	sub := &nopSubmitter{}
	m := newSaveTestModel(t, history, "sess-parent", rec.call, sub.submit)

	res := saveSlashHandler("/save", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if rec.calls != 1 {
		t.Fatalf("SessionExportFunc called %d times, want exactly 1", rec.calls)
	}
	if rec.gotID != "sess-parent" {
		t.Fatalf("SessionExportFunc got sessionID = %q, want sess-parent (m.frame.SessionID)", rec.gotID)
	}
	if !strings.Contains(res.StatusMessage, wantPath) {
		t.Fatalf("StatusMessage = %q, want it to contain %q", res.StatusMessage, wantPath)
	}
	if sub.calls != 0 {
		t.Fatalf("Submit called %d times, want 0 (export must not go through kernel.Submit)", sub.calls)
	}
}

func TestSlashSave_ExportFailureRemovesPartialFile(t *testing.T) {
	history := []hermes.Message{{Role: "user", Content: "hi"}}

	tmp := t.TempDir()
	partialPath := filepath.Join(tmp, "partial.md")
	exportErr := errors.New("disk full")

	rec := &recordingExportFunc{
		path: partialPath,
		err:  exportErr,
		beforeReturn: func() {
			if err := os.WriteFile(partialPath, []byte("partial"), 0o644); err != nil {
				t.Fatalf("seed partial file: %v", err)
			}
		},
	}
	sub := &nopSubmitter{}
	m := newSaveTestModel(t, history, "sess-1", rec.call, sub.submit)

	res := saveSlashHandler("/save", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if rec.calls != 1 {
		t.Fatalf("SessionExportFunc called %d times, want 1", rec.calls)
	}
	if _, err := os.Stat(partialPath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("partial file still present at %s (stat err=%v); handler must os.Remove on export failure", partialPath, err)
	}
	if !strings.HasPrefix(res.StatusMessage, "save: write failed:") {
		t.Fatalf("StatusMessage = %q, want prefix %q", res.StatusMessage, "save: write failed:")
	}
	if !strings.Contains(res.StatusMessage, exportErr.Error()) {
		t.Fatalf("StatusMessage = %q, want it to surface underlying error %q", res.StatusMessage, exportErr.Error())
	}
	if sub.calls != 0 {
		t.Fatalf("Submit called %d times, want 0", sub.calls)
	}
}

func TestSlashSave_ErrSessionNotFoundReturnsStoreUnavailable(t *testing.T) {
	history := []hermes.Message{{Role: "user", Content: "hi"}}
	rec := &recordingExportFunc{err: transcript.ErrSessionNotFound}
	sub := &nopSubmitter{}
	m := newSaveTestModel(t, history, "sess-missing", rec.call, sub.submit)

	res := saveSlashHandler("/save", &m)

	if !res.Handled {
		t.Fatal("Handled = false, want true")
	}
	if res.StatusMessage != "save: store unavailable" {
		t.Fatalf("StatusMessage = %q, want %q", res.StatusMessage, "save: store unavailable")
	}
	if sub.calls != 0 {
		t.Fatalf("Submit called %d times, want 0", sub.calls)
	}
}
