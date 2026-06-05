package cron

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	autocron "github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
)

type fakeJobStore struct {
	jobs    []autocron.Job
	deleted string
}

func (s *fakeJobStore) List() ([]autocron.Job, error) { return s.jobs, nil }
func (s *fakeJobStore) Delete(id string) error {
	s.deleted = id
	return nil
}

type fakeRunReader struct{ runs []autocron.Run }

func (r fakeRunReader) LatestRuns(_ context.Context, _ string, _ int) ([]autocron.Run, error) {
	return r.runs, nil
}

func TestListJobsEmpty(t *testing.T) {
	var out bytes.Buffer
	if err := ListJobs(&out, &fakeJobStore{}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "No cron jobs found.\n" {
		t.Fatalf("ListJobs output = %q", got)
	}
}

func TestListJobsFormatsPromptAndScriptRows(t *testing.T) {
	store := &fakeJobStore{jobs: []autocron.Job{
		{ID: "1234567890abcdef", Name: "prompt", Schedule: "@daily", Prompt: strings.Repeat("a", 60), LastStatus: "success", LastRunUnix: time.Date(2026, 6, 1, 12, 34, 0, 0, time.Local).Unix()},
		{ID: "abcdef1234567890", Name: "script", Schedule: "@hourly", Script: "/very/long/path/" + strings.Repeat("b", 60), NoAgent: true, Paused: true},
	}}
	var out bytes.Buffer
	if err := ListJobs(&out, store); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"ID", "12345678", "prompt", "success", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa...", "abcdef12", "script", "[script]", "yes"} {
		if !strings.Contains(got, want) {
			t.Fatalf("ListJobs output missing %q:\n%s", want, got)
		}
	}
}

func TestRemoveJobDeletesResolvedPrefix(t *testing.T) {
	store := &fakeJobStore{jobs: []autocron.Job{{ID: "1234567890abcdef", Name: "nightly"}}}
	var out bytes.Buffer
	if err := RemoveJob(&out, store, "12345678"); err != nil {
		t.Fatal(err)
	}
	if store.deleted != "1234567890abcdef" {
		t.Fatalf("deleted = %q", store.deleted)
	}
	if got := out.String(); got != "Removed cron job 12345678 (nightly)\n" {
		t.Fatalf("RemoveJob output = %q", got)
	}
}

func TestStatusJobIncludesRecentRuns(t *testing.T) {
	store := &fakeJobStore{jobs: []autocron.Job{{ID: "1234567890abcdef", Name: "nightly", Schedule: "@daily", CreatedAt: time.Date(2026, 6, 1, 1, 2, 3, 0, time.Local).Unix(), LastRunUnix: time.Date(2026, 6, 1, 4, 5, 6, 0, time.Local).Unix(), LastStatus: "error"}}}
	runs := fakeRunReader{runs: []autocron.Run{{StartedAt: time.Date(2026, 6, 1, 4, 5, 6, 0, time.Local).Unix(), Status: "error", OutputPreview: strings.Repeat("x", 90), ErrorMsg: "boom"}}}
	var out bytes.Buffer
	if err := StatusJob(&out, store, runs, "12345678"); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"Job:      nightly (12345678)", "Schedule: @daily", "Last run:", "Recent runs:", "ERROR"} {
		if !strings.Contains(got, want) {
			t.Fatalf("StatusJob output missing %q:\n%s", want, got)
		}
	}
}
