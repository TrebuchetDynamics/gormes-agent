package apiserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cron"
)

// TestAPIServerJobsCompatBuildAttribution proves the legacy
// `/api/jobs` and `/api/jobs/{id}` envelopes include build provenance
// at the top of their JSON response so fleet automation rolling out
// schedule changes across machines can attribute every legacy
// response to the binary version. Same convention as the rest of the
// JSON arc — additive to the legacy contract since clients ignore
// unknown fields.
func TestAPIServerJobsCompatBuildAttribution(t *testing.T) {
	mutator := newFakeCronJobMutator(
		cron.Job{ID: "aabbccddeeff", Name: "enabled", Schedule: "@hourly", Prompt: "visible", Provider: "telegram"},
	)
	srv := NewServer(Config{
		APIKey:         "plain-existing-token",
		ModelName:      "gormes-agent",
		BuildInfo:      BuildInfo{Version: "test-jobscompat-attr", GitCommit: "facefeed"},
		CronJobs:       mutator,
		CronJobMutator: mutator,
		MaxBodyBytes:   1_000_000,
	})
	h := srv.Handler()
	auth := map[string]string{"Authorization": "Bearer plain-existing-token"}

	// list envelope
	list := getJSON(t, h, "/api/jobs", auth)
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d; body=%s", list.Code, list.Body.String())
	}
	var listGot struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Jobs []map[string]any `json:"jobs"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &listGot); err != nil {
		t.Fatalf("decode list: %v\nbody=%s", err, list.Body.String())
	}
	if listGot.Build.Version != "test-jobscompat-attr" || listGot.Build.GitCommit != "facefeed" {
		t.Errorf("list build = %+v, want version=test-jobscompat-attr commit=facefeed", listGot.Build)
	}
	if len(listGot.Jobs) != 1 {
		t.Errorf("list jobs len = %d, want 1", len(listGot.Jobs))
	}

	// get envelope
	get := getJSON(t, h, "/api/jobs/aabbccddeeff", auth)
	if get.Code != http.StatusOK {
		t.Fatalf("get status = %d; body=%s", get.Code, get.Body.String())
	}
	var getGot struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Job map[string]any `json:"job"`
	}
	if err := json.Unmarshal(get.Body.Bytes(), &getGot); err != nil {
		t.Fatalf("decode get: %v\nbody=%s", err, get.Body.String())
	}
	if getGot.Build.Version != "test-jobscompat-attr" {
		t.Errorf("get build = %+v, want version=test-jobscompat-attr", getGot.Build)
	}
	if getGot.Job["id"] != "aabbccddeeff" {
		t.Errorf("get job id = %v, want aabbccddeeff", getGot.Job["id"])
	}
}

func TestAPIServerJobsCompatListAndIncludeDisabled(t *testing.T) {
	mutator := newFakeCronJobMutator(
		cron.Job{ID: "aabbccddeeff", Name: "enabled", Schedule: "@hourly", Prompt: "visible", Provider: "telegram"},
		cron.Job{ID: "001122334455", Name: "paused", Schedule: "@daily", Prompt: "hidden", Provider: "discord", Paused: true},
	)
	h := newCronAdminMutateTestServer(t, mutator, nil, nil)
	auth := map[string]string{"Authorization": "Bearer plain-existing-token"}

	rec := getJSON(t, h, "/api/jobs", auth)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Jobs []map[string]any `json:"jobs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode list body: %v", err)
	}
	if len(body.Jobs) != 1 || body.Jobs[0]["id"] != "aabbccddeeff" {
		t.Fatalf("default list jobs = %+v, want only enabled job", body.Jobs)
	}

	rec = getJSON(t, h, "/api/jobs?include_disabled=true", auth)
	if rec.Code != http.StatusOK {
		t.Fatalf("include_disabled status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode include_disabled body: %v", err)
	}
	if len(body.Jobs) != 2 {
		t.Fatalf("include_disabled jobs = %+v, want enabled and paused", body.Jobs)
	}
	if strings.Contains(rec.Body.String(), "hidden") {
		t.Fatalf("legacy list leaked prompt body: %s", rec.Body.String())
	}
}

func TestAPIServerJobsCompatListNormalizesPartialRecords(t *testing.T) {
	mutator := newFakeCronJobMutator(cron.Job{ID: "abc123deadbe"})
	h := newCronAdminMutateTestServer(t, mutator, nil, nil)
	auth := map[string]string{"Authorization": "Bearer plain-existing-token"}

	rec := getJSON(t, h, "/api/jobs", auth)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Jobs []map[string]any `json:"jobs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode list body: %v", err)
	}
	if len(body.Jobs) != 1 {
		t.Fatalf("jobs = %+v, want one partial job", body.Jobs)
	}
	got := body.Jobs[0]
	if got["id"] != "abc123deadbe" || got["name"] != "abc123deadbe" || got["schedule"] != "?" {
		t.Fatalf("partial job view = %+v, want id/name fallback and ? schedule", got)
	}
	if got["enabled"] != true || got["paused"] != false {
		t.Fatalf("enabled/paused = %v/%v, want true/false", got["enabled"], got["paused"])
	}
	if strings.Contains(rec.Body.String(), `"prompt"`) {
		t.Fatalf("legacy list leaked prompt field for partial job: %s", rec.Body.String())
	}
}

func TestAPIServerJobsCompatCreateUpdateDeletePauseResumeRun(t *testing.T) {
	mutator := newFakeCronJobMutator()
	mutator.nextID = "aabbccddeeff"
	trigger := &fakeCronTriggerHandler{result: TriggerResult{RunID: "run-123", Status: "queued", PromptHash: "sha256-deadbeef"}}
	auditor := &recordingAuditor{}
	h := newCronAdminMutateTestServer(t, mutator, trigger, auditor)
	auth := map[string]string{"Authorization": "Bearer plain-existing-token"}

	createBody := map[string]any{
		"name":     "morning-status",
		"schedule": "*/5 * * * *",
		"prompt":   "Send the morning status report.",
		"provider": "telegram",
	}
	rec := postJSON(t, h, "/api/jobs", createBody, auth)
	if rec.Code != http.StatusOK {
		t.Fatalf("create status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var createResp struct {
		Job map[string]any `json:"job"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResp.Job["id"] != "aabbccddeeff" {
		t.Fatalf("create job id = %v, want aabbccddeeff", createResp.Job["id"])
	}

	updateBody := map[string]any{
		"name":       "renamed-status",
		"schedule":   "0 9 * * *",
		"prompt":     "Send the daily status report.",
		"evil_field": "must be ignored",
	}
	rec = patchJSON(t, h, "/api/jobs/aabbccddeeff", updateBody, auth)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	updated, err := mutator.Get("aabbccddeeff")
	if err != nil {
		t.Fatalf("updated job missing: %v", err)
	}
	if updated.Name != "renamed-status" || updated.Schedule != "0 9 * * *" {
		t.Fatalf("updated job = %+v, want renamed 0 9 * * *", updated)
	}

	rec = postJSON(t, h, "/api/jobs/aabbccddeeff/pause", nil, auth)
	if rec.Code != http.StatusOK {
		t.Fatalf("pause status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if paused, _ := mutator.Get("aabbccddeeff"); !paused.Paused {
		t.Fatal("pause did not mark job paused")
	}
	rec = postJSON(t, h, "/api/jobs/aabbccddeeff/resume", nil, auth)
	if rec.Code != http.StatusOK {
		t.Fatalf("resume status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if resumed, _ := mutator.Get("aabbccddeeff"); resumed.Paused {
		t.Fatal("resume left job paused")
	}

	rec = postJSON(t, h, "/api/jobs/aabbccddeeff/run", nil, auth)
	if rec.Code != http.StatusOK {
		t.Fatalf("run status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var runResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &runResp); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	if runResp["job_id"] != "aabbccddeeff" || runResp["run_id"] != "run-123" {
		t.Fatalf("run response = %+v, want job_id and run_id", runResp)
	}
	if len(trigger.calls) != 1 || trigger.calls[0] != "aabbccddeeff" {
		t.Fatalf("trigger calls = %v, want [aabbccddeeff]", trigger.calls)
	}

	rec = deleteJSON(t, h, "/api/jobs/aabbccddeeff", auth)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var deleteResp struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &deleteResp); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if !deleteResp.OK {
		t.Fatalf("delete response = %+v, want ok true", deleteResp)
	}
}

func TestAPIServerJobsCompatAuthAndBodyLimits(t *testing.T) {
	mutator := newFakeCronJobMutator(cron.Job{ID: "aabbccddeeff", Name: "known", Schedule: "@hourly", Prompt: "ok"})
	h := newCronAdminMutateTestServer(t, mutator, &fakeCronTriggerHandler{}, &recordingAuditor{})
	auth := map[string]string{"Authorization": "Bearer plain-existing-token"}

	rec := getJSON(t, h, "/api/jobs", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated list status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
	rec = getJSON(t, h, "/api/jobs/aabbccddeeff", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated get status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
	rec = getJSON(t, h, "/api/jobs/not-a-valid-hex!", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated invalid-id status = %d, want auth before id validation; body=%s", rec.Code, rec.Body.String())
	}

	rec = getJSON(t, h, "/api/jobs/not-a-valid-hex!", auth)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid id status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	assertJobsCompatErrorCode(t, rec, "invalid_job_id")

	rec = postJSON(t, h, "/api/jobs", map[string]any{"schedule": "@hourly", "prompt": "ok"}, auth)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing name status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	assertJobsCompatErrorCode(t, rec, "invalid_request_body")

	rec = postJSON(t, h, "/api/jobs", map[string]any{"name": "big", "schedule": "@hourly", "prompt": strings.Repeat("x", 5001)}, auth)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("prompt length status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	assertJobsCompatErrorCode(t, rec, "invalid_prompt_length")

	rec = postJSON(t, h, "/api/jobs", map[string]any{"name": "repeat", "schedule": "@hourly", "prompt": "ok", "repeat": 0}, auth)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("repeat status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	assertJobsCompatErrorCode(t, rec, "invalid_repeat")

	srv := NewServer(Config{
		APIKey:         "plain-existing-token",
		ModelName:      "gormes-agent",
		CronJobs:       mutator,
		CronJobMutator: mutator,
		MaxBodyBytes:   64,
	})
	huge := strings.Repeat("x", 256)
	req := httptest.NewRequest(http.MethodPost, "/api/jobs", strings.NewReader(`{"name":"big","schedule":"@hourly","prompt":"`+huge+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer plain-existing-token")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized status = %d, want 413; body=%s", rec.Code, rec.Body.String())
	}
	assertJobsCompatErrorCode(t, rec, "body_too_large")
}

func assertJobsCompatErrorCode(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	var env map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v; body=%s", err, rec.Body.String())
	}
	if got := env["error"]["code"]; got != want {
		t.Fatalf("error.code = %v, want %s; body=%s", got, want, rec.Body.String())
	}
}
