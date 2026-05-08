package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cron"
)

// fakeCronJobReader is a hermetic stand-in for the bbolt-backed cron.Store
// read surface. Tests configure the slice of jobs (and optional list error)
// directly without touching the real cron store.
type fakeCronJobReader struct {
	jobs    []cron.Job
	listErr error
	getErr  error
}

func (f *fakeCronJobReader) List() ([]cron.Job, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]cron.Job, len(f.jobs))
	copy(out, f.jobs)
	return out, nil
}

func (f *fakeCronJobReader) Get(id string) (cron.Job, error) {
	if f.getErr != nil {
		return cron.Job{}, f.getErr
	}
	for _, j := range f.jobs {
		if j.ID == id {
			return j, nil
		}
	}
	return cron.Job{}, cron.ErrJobNotFound
}

// fakeCronRunReader returns canned run history rows without touching SQLite.
type fakeCronRunReader struct {
	runs    map[string][]cron.Run
	listErr error
}

func (f *fakeCronRunReader) LatestRuns(_ context.Context, jobID string, limit int) ([]cron.Run, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	rows := f.runs[jobID]
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	out := make([]cron.Run, len(rows))
	copy(out, rows)
	return out, nil
}

func newCronAdminTestServer(t *testing.T, jobs *fakeCronJobReader, runs *fakeCronRunReader) http.Handler {
	t.Helper()
	srv := NewServer(Config{
		APIKey:       "plain-existing-token",
		ModelName:    "gormes-agent",
		CronJobs:     jobs,
		CronRuns:     runs,
		MaxBodyBytes: 1_000_000,
	})
	return srv.Handler()
}

// TestAPIServerCronAdmin_BuildAttributionAcrossReadEndpoints proves
// `/v1/admin/cron/jobs`, `/v1/admin/cron/jobs/{id}`, and
// `/v1/admin/cron/jobs/{id}/runs` all carry the configured BuildInfo
// at the top of their JSON response so fleet automation aggregating
// cron schedule state across machines can attribute every response to
// the binary version that emitted it. Same convention as the dashboard
// JSON arc (slices 110-113).
func TestAPIServerCronAdmin_BuildAttributionAcrossReadEndpoints(t *testing.T) {
	jobs := &fakeCronJobReader{
		jobs: []cron.Job{
			{ID: "job-1", Name: "n1", Schedule: "0 8 * * *", CreatedAt: 1700000000},
		},
	}
	runs := &fakeCronRunReader{}
	srv := NewServer(Config{
		APIKey:       "plain-existing-token",
		ModelName:    "gormes-agent",
		CronJobs:     jobs,
		CronRuns:     runs,
		MaxBodyBytes: 1_000_000,
		BuildInfo: BuildInfo{
			Version:   "test-cron-attr",
			GitCommit: "deadcafe",
		},
	})
	h := srv.Handler()
	auth := map[string]string{"Authorization": "Bearer plain-existing-token"}

	for _, path := range []string{
		"/v1/admin/cron/jobs",
		"/v1/admin/cron/jobs/job-1",
		"/v1/admin/cron/jobs/job-1/runs",
	} {
		rec := getJSON(t, h, path, auth)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d; body=%s", path, rec.Code, rec.Body.String())
		}
		var got struct {
			Build struct {
				Version   string `json:"version"`
				GitCommit string `json:"git_commit"`
			} `json:"build"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("%s: decode: %v\nbody=%s", path, err, rec.Body.String())
		}
		if got.Build.Version != "test-cron-attr" {
			t.Errorf("%s: build.version = %q, want test-cron-attr (body=%s)", path, got.Build.Version, rec.Body.String())
		}
		if got.Build.GitCommit != "deadcafe" {
			t.Errorf("%s: build.git_commit = %q, want deadcafe", path, got.Build.GitCommit)
		}
	}
}

func TestAPIServerCronAdmin_ListJobs(t *testing.T) {
	jobs := &fakeCronJobReader{
		jobs: []cron.Job{
			{
				ID:              "job-enabled",
				Name:            "morning-status",
				Schedule:        "0 8 * * *",
				Prompt:          "SECRET prompt body",
				Paused:          false,
				CreatedAt:       1700000000,
				LastRunUnix:     1700100000,
				LastStatus:      "success",
				Provider:        "telegram",
				Model:           "gormes-agent",
				Repeat:          0,
				RepeatCompleted: 0,
				Script:          "/etc/secret-script.sh",
			},
			{
				ID:          "job-paused",
				Name:        "weekly-report",
				Schedule:    "@weekly",
				Prompt:      "another secret",
				Paused:      true,
				CreatedAt:   1700000001,
				LastRunUnix: 0,
				LastStatus:  "",
				Provider:    "discord",
			},
		},
	}
	h := newCronAdminTestServer(t, jobs, &fakeCronRunReader{})

	rec := getJSON(t, h, "/v1/admin/cron/jobs", map[string]string{"Authorization": "Bearer plain-existing-token"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var body struct {
		Jobs []map[string]any `json:"jobs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode jobs: %v", err)
	}
	if len(body.Jobs) != 2 {
		t.Fatalf("len(jobs) = %d, want 2; body=%s", len(body.Jobs), rec.Body.String())
	}

	byID := map[string]map[string]any{}
	for _, j := range body.Jobs {
		id, _ := j["id"].(string)
		byID[id] = j
	}
	enabled := byID["job-enabled"]
	if enabled == nil {
		t.Fatalf("missing job-enabled in jobs: %s", rec.Body.String())
	}
	if got, want := enabled["name"], "morning-status"; got != want {
		t.Fatalf("name = %v, want %v", got, want)
	}
	if got, want := enabled["schedule"], "0 8 * * *"; got != want {
		t.Fatalf("schedule = %v, want %v", got, want)
	}
	if got := enabled["paused"]; got != false {
		t.Fatalf("paused = %v, want false", got)
	}
	if got := enabled["enabled"]; got != true {
		t.Fatalf("enabled = %v, want true", got)
	}
	if got, want := enabled["last_status"], "success"; got != want {
		t.Fatalf("last_status = %v, want %v", got, want)
	}
	if got := enabled["last_run_unix"]; got != float64(1700100000) {
		t.Fatalf("last_run_unix = %v, want 1700100000", got)
	}
	if _, ok := enabled["next_run_unix"]; !ok {
		t.Fatalf("missing next_run_unix evidence: %s", rec.Body.String())
	}
	if got, want := enabled["target"], "telegram"; got != want {
		t.Fatalf("target = %v, want %v", got, want)
	}

	paused := byID["job-paused"]
	if paused == nil {
		t.Fatalf("missing job-paused in jobs: %s", rec.Body.String())
	}
	if got := paused["paused"]; got != true {
		t.Fatalf("paused = %v, want true", got)
	}
	if got := paused["enabled"]; got != false {
		t.Fatalf("enabled = %v, want false", got)
	}

	// Redaction: prompt and script bodies must never appear in list output.
	bodyStr := rec.Body.String()
	for _, secret := range []string{"SECRET prompt body", "another secret", "/etc/secret-script.sh"} {
		if strings.Contains(bodyStr, secret) {
			t.Fatalf("list response leaked secret %q: %s", secret, bodyStr)
		}
	}
}

func TestAPIServerCronAdmin_GetJobMissing(t *testing.T) {
	jobs := &fakeCronJobReader{
		jobs: []cron.Job{{
			ID:       "job-known",
			Name:     "morning",
			Schedule: "0 8 * * *",
		}},
	}
	h := newCronAdminTestServer(t, jobs, &fakeCronRunReader{})

	rec := getJSON(t, h, "/v1/admin/cron/jobs/missing-id", map[string]string{"Authorization": "Bearer plain-existing-token"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
	var env map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	errObj, ok := env["error"]
	if !ok {
		t.Fatalf("missing error envelope: %s", rec.Body.String())
	}
	if errObj["code"] != "cron_job_missing" {
		t.Fatalf("error.code = %v, want cron_job_missing", errObj["code"])
	}
	if errObj["type"] != "invalid_request_error" {
		t.Fatalf("error.type = %v, want invalid_request_error", errObj["type"])
	}
}

func TestAPIServerCronAdmin_RunHistory(t *testing.T) {
	jobs := &fakeCronJobReader{
		jobs: []cron.Job{{ID: "job-1", Name: "morning", Schedule: "@hourly"}},
	}
	runs := &fakeCronRunReader{
		runs: map[string][]cron.Run{
			"job-1": {
				{
					ID:                7,
					JobID:             "job-1",
					StartedAt:         1700000005,
					FinishedAt:        1700000010,
					PromptHash:        "sha256-deadbeef",
					Status:            "success",
					Delivered:         true,
					SuppressionReason: "",
					OutputPreview:     "TOPSECRET-output-payload-do-not-leak",
					ErrorMsg:          "",
				},
				{
					ID:                6,
					JobID:             "job-1",
					StartedAt:         1700000000,
					FinishedAt:        1700000004,
					PromptHash:        "sha256-cafebabe",
					Status:            "error",
					Delivered:         false,
					SuppressionReason: "",
					OutputPreview:     "",
					ErrorMsg:          "TOPSECRET-error-trace-with-credentials",
				},
			},
		},
	}
	h := newCronAdminTestServer(t, jobs, runs)

	rec := getJSON(t, h, "/v1/admin/cron/jobs/job-1/runs", map[string]string{"Authorization": "Bearer plain-existing-token"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		JobID string           `json:"job_id"`
		Runs  []map[string]any `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode runs: %v", err)
	}
	if body.JobID != "job-1" {
		t.Fatalf("job_id = %q, want job-1", body.JobID)
	}
	if len(body.Runs) != 2 {
		t.Fatalf("len(runs) = %d, want 2; body=%s", len(body.Runs), rec.Body.String())
	}
	first := body.Runs[0]
	if first["status"] != "success" {
		t.Fatalf("runs[0].status = %v, want success", first["status"])
	}
	if first["delivered"] != true {
		t.Fatalf("runs[0].delivered = %v, want true", first["delivered"])
	}
	if first["started_at"] != float64(1700000005) {
		t.Fatalf("runs[0].started_at = %v, want 1700000005", first["started_at"])
	}
	if first["finished_at"] != float64(1700000010) {
		t.Fatalf("runs[0].finished_at = %v, want 1700000010", first["finished_at"])
	}
	// Secret payload redaction: prompt content, output preview, and error
	// trace must never leak through this read-only audit endpoint.
	bodyStr := rec.Body.String()
	for _, secret := range []string{
		"TOPSECRET-output-payload-do-not-leak",
		"TOPSECRET-error-trace-with-credentials",
		"output_preview",
		"error_msg",
	} {
		if strings.Contains(bodyStr, secret) {
			t.Fatalf("run history leaked redacted field %q: %s", secret, bodyStr)
		}
	}
}

func TestAPIServerCronAdmin_RunHistoryUnknownJob(t *testing.T) {
	jobs := &fakeCronJobReader{
		jobs: []cron.Job{{ID: "job-known", Name: "x", Schedule: "@hourly"}},
	}
	h := newCronAdminTestServer(t, jobs, &fakeCronRunReader{})
	rec := getJSON(t, h, "/v1/admin/cron/jobs/missing-id/runs", map[string]string{"Authorization": "Bearer plain-existing-token"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIServerCronAdmin_RequiresBearerAuth(t *testing.T) {
	jobs := &fakeCronJobReader{jobs: []cron.Job{{ID: "j", Name: "n", Schedule: "@hourly"}}}
	h := newCronAdminTestServer(t, jobs, &fakeCronRunReader{})

	for _, path := range []string{
		"/v1/admin/cron/jobs",
		"/v1/admin/cron/jobs/j",
		"/v1/admin/cron/jobs/j/runs",
	} {
		rec := getJSON(t, h, path, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("path %s status = %d, want 401; body=%s", path, rec.Code, rec.Body.String())
		}
		var body map[string]map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("error envelope JSON for %s: %v", path, err)
		}
		if body["error"]["code"] != "invalid_api_key" {
			t.Fatalf("path %s error.code = %v, want invalid_api_key", path, body["error"]["code"])
		}
	}
}

func TestAPIServerCronAdmin_StoreUnavailable(t *testing.T) {
	// No CronJobs/CronRuns wired -> degraded mode reports cron_store_unavailable
	srv := NewServer(Config{APIKey: "plain-existing-token", ModelName: "gormes-agent"})
	h := srv.Handler()

	rec := getJSON(t, h, "/v1/admin/cron/jobs", map[string]string{"Authorization": "Bearer plain-existing-token"})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("error envelope JSON: %v", err)
	}
	if body["error"]["code"] != "cron_store_unavailable" {
		t.Fatalf("error.code = %v, want cron_store_unavailable", body["error"]["code"])
	}
}

func TestAPIServerCronAdmin_ListJobsBackendError(t *testing.T) {
	jobs := &fakeCronJobReader{listErr: errors.New("boom")}
	h := newCronAdminTestServer(t, jobs, &fakeCronRunReader{})
	rec := getJSON(t, h, "/v1/admin/cron/jobs", map[string]string{"Authorization": "Bearer plain-existing-token"})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("error envelope: %v", err)
	}
	if body["error"]["code"] != "cron_store_unavailable" {
		t.Fatalf("error.code = %v, want cron_store_unavailable", body["error"]["code"])
	}
}

// Compile-time guards: production cron.Store and cron.RunStore must
// continue to satisfy the read facades the API server depends on.
var (
	_ CronJobReader = (*cron.Store)(nil)
	_ CronRunReader = (*cron.RunStore)(nil)
)
