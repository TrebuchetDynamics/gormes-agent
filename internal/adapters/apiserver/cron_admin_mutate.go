package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver/cronadmin"
	"github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
)

const (
	legacyAPIJobIDLength  = 12
	legacyAPIMaxNameLen   = 200
	legacyAPIMaxPromptLen = 5000
)

var legacyAPIUpdateAllowedFields = map[string]struct{}{
	"name":             {},
	"schedule":         {},
	"prompt":           {},
	"deliver":          {},
	"provider":         {},
	"model":            {},
	"skills":           {},
	"skill":            {},
	"repeat":           {},
	"enabled":          {},
	"enabled_toolsets": {},
	"workdir":          {},
	"script":           {},
	"context_from":     {},
}

// CronJobSpec is the operator-supplied shape used by create/update endpoints.
type CronJobSpec = cronadmin.JobSpec

// CronJobMutator is the narrow write facade the API server uses to mutate
// cron jobs without depending on bbolt or the scheduler. Production
// implementations adapt *cron.Store; tests inject a fake.
//
// The mutator MUST NOT start the scheduler, dispatch a delivery, or contact
// providers. Read traffic continues to flow through CronJobReader.
type CronJobMutator = cronadmin.JobMutator

// CronTriggerHandler is the trigger seam the API server uses to launch a
// one-shot run. A nil seam represents "delivery unavailable" and the trigger
// endpoint records trigger_delivery_unavailable + 503.
type CronTriggerHandler = cronadmin.TriggerHandler

// TriggerResult is the operator-visible shape returned by /trigger.
type TriggerResult = cronadmin.TriggerResult

// CronAdminAuditEvent is the audit shape recorded for each cron mutation.
type CronAdminAuditEvent = cronadmin.AuditEvent

// CronAdminAuditor is the audit seam wired into the apiserver Config. It is
// optional; when nil, mutating endpoints still work but do not record events.
type CronAdminAuditor = cronadmin.Auditor

// ErrCronJobAlreadyPaused / ErrCronJobNotPaused are conflict signals returned
// by the mutator when an operator pauses or resumes a job whose state already
// matches the requested action. They surface as HTTP 409 with the shared
// error envelope.
var (
	ErrCronJobAlreadyPaused = errors.New("cron: job already paused")
	ErrCronJobNotPaused     = errors.New("cron: job not paused")
)

func (s *Server) handleCronAdminCreate(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.cronMutator == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron job store is not configured", "server_error", "", "cron_store_unavailable")
		return
	}

	body, err := readLimitedBody(w, r, s.maxBodyBytes)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) || errors.Is(err, errBodyTooLarge) {
			writeOpenAIError(w, http.StatusRequestEntityTooLarge, "Request body too large.", "invalid_request_error", "", "body_too_large")
			return
		}
		writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "", "invalid_request_body")
		return
	}
	var spec CronJobSpec
	if err := json.Unmarshal(body, &spec); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "Invalid JSON in request body", "invalid_request_error", "", "invalid_json")
		return
	}
	if status, code, msg := s.validateCronAdminSpec(spec); status != 0 {
		writeOpenAIError(w, status, msg, "invalid_request_error", "", code)
		return
	}

	id, err := s.cronMutator.Create(r.Context(), spec)
	if err != nil {
		s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "create", Outcome: "error", Code: "cron_create_failed"})
		writeOpenAIError(w, http.StatusInternalServerError, "Cron job create failed: "+err.Error(), "server_error", "", "cron_create_failed")
		return
	}
	s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "create", JobID: id, Outcome: "ok"})

	job := jobFromSpec(spec)
	job.ID = id
	writeJSON(w, http.StatusCreated, map[string]any{
		"build": s.buildInfo,
		"job":   cronAdminJobViewFor(job, s.now()),
	})
}

func (s *Server) handleLegacyAPIJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleLegacyAPIJobsList(w, r)
	case http.MethodPost:
		s.handleLegacyAPIJobsCreate(w, r)
	default:
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
	}
}

func (s *Server) handleLegacyAPIJobByID(w http.ResponseWriter, r *http.Request) {
	jobID, sub, ok := parseLegacyAPIJobPath(r.URL.Path)
	if !ok {
		writeOpenAIError(w, http.StatusNotFound, "API jobs route not found", "invalid_request_error", "", "not_found")
		return
	}
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if !legacyAPIValidJobID(jobID) {
		writeOpenAIError(w, http.StatusBadRequest, "Invalid job ID format", "invalid_request_error", "id", "invalid_job_id")
		return
	}

	switch sub {
	case "":
		switch r.Method {
		case http.MethodGet:
			s.handleLegacyAPIJobsGet(w, r, jobID)
		case http.MethodPatch, http.MethodPut:
			s.handleLegacyAPIJobsUpdate(w, r, jobID)
		case http.MethodDelete:
			s.handleLegacyAPIJobsDelete(w, r, jobID)
		default:
			writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
		}
	case "pause":
		if r.Method != http.MethodPost {
			writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
			return
		}
		s.handleLegacyAPIJobsPauseResume(w, r, jobID, true)
	case "resume":
		if r.Method != http.MethodPost {
			writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
			return
		}
		s.handleLegacyAPIJobsPauseResume(w, r, jobID, false)
	case "run":
		if r.Method != http.MethodPost {
			writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
			return
		}
		s.handleLegacyAPIJobsRun(w, r, jobID)
	default:
		writeOpenAIError(w, http.StatusNotFound, "API jobs route not found", "invalid_request_error", "", "not_found")
	}
}

func (s *Server) handleLegacyAPIJobsList(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.cronJobs == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron job store is not configured", "server_error", "", "cron_store_unavailable")
		return
	}
	jobs, err := s.cronJobs.List()
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "Cron job store unavailable: "+err.Error(), "server_error", "", "cron_store_unavailable")
		return
	}
	includeDisabled := legacyAPIIncludeDisabled(r)
	now := s.now()
	views := make([]cronAdminJobView, 0, len(jobs))
	for _, job := range jobs {
		if job.Paused && !includeDisabled {
			continue
		}
		views = append(views, cronAdminJobViewFor(job, now))
	}
	sort.SliceStable(views, func(i, j int) bool { return views[i].ID < views[j].ID })
	writeJSON(w, http.StatusOK, map[string]any{"build": s.buildInfo, "jobs": views})
}

func (s *Server) handleLegacyAPIJobsGet(w http.ResponseWriter, _ *http.Request, id string) {
	if s.cronJobs == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron job store is not configured", "server_error", "", "cron_store_unavailable")
		return
	}
	job, err := s.cronJobs.Get(id)
	if err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			writeOpenAIError(w, http.StatusNotFound, "Cron job not found", "invalid_request_error", "id", "cron_job_missing")
			return
		}
		writeOpenAIError(w, http.StatusInternalServerError, "Cron job lookup failed: "+err.Error(), "server_error", "", "cron_store_unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"build": s.buildInfo, "job": cronAdminJobViewFor(job, s.now())})
}

func (s *Server) handleLegacyAPIJobsCreate(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.cronMutator == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron job store is not configured", "server_error", "", "cron_store_unavailable")
		return
	}

	spec, err := s.readLegacyAPICronJobSpec(w, r, nil)
	if err != nil {
		s.writeLegacyAPISpecError(w, err)
		return
	}
	id, err := s.cronMutator.Create(r.Context(), spec)
	if err != nil {
		s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "create", Outcome: "error", Code: "cron_create_failed"})
		writeOpenAIError(w, http.StatusInternalServerError, "Cron job create failed: "+err.Error(), "server_error", "", "cron_create_failed")
		return
	}
	s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "create", JobID: id, Outcome: "ok"})
	s.writeLegacyAPIJobFromStoreOrSpec(w, id, spec, http.StatusOK)
}

func (s *Server) handleLegacyAPIJobsUpdate(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.cronMutator == nil || s.cronJobs == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron job store is not configured", "server_error", "", "cron_store_unavailable")
		return
	}
	existing, err := s.cronJobs.Get(id)
	if err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			writeOpenAIError(w, http.StatusNotFound, "Cron job not found", "invalid_request_error", "id", "cron_job_missing")
			return
		}
		writeOpenAIError(w, http.StatusInternalServerError, "Cron job lookup failed: "+err.Error(), "server_error", "", "cron_store_unavailable")
		return
	}

	spec, err := s.readLegacyAPICronJobSpec(w, r, &existing)
	if err != nil {
		s.writeLegacyAPISpecError(w, err)
		return
	}
	if err := s.cronMutator.Update(r.Context(), id, spec); err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "update", JobID: id, Outcome: "error", Code: "cron_job_missing"})
			writeOpenAIError(w, http.StatusNotFound, "Cron job not found", "invalid_request_error", "id", "cron_job_missing")
			return
		}
		s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "update", JobID: id, Outcome: "error", Code: "cron_update_failed"})
		writeOpenAIError(w, http.StatusInternalServerError, "Cron job update failed: "+err.Error(), "server_error", "", "cron_update_failed")
		return
	}
	s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "update", JobID: id, Outcome: "ok"})
	s.writeLegacyAPIJobFromStoreOrSpec(w, id, spec, http.StatusOK)
}

func (s *Server) handleLegacyAPIJobsDelete(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.cronMutator == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron job store is not configured", "server_error", "", "cron_store_unavailable")
		return
	}
	if err := s.cronMutator.Delete(r.Context(), id); err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "delete", JobID: id, Outcome: "error", Code: "cron_job_missing"})
			writeOpenAIError(w, http.StatusNotFound, "Cron job not found", "invalid_request_error", "id", "cron_job_missing")
			return
		}
		s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "delete", JobID: id, Outcome: "error", Code: "cron_delete_failed"})
		writeOpenAIError(w, http.StatusInternalServerError, "Cron job delete failed: "+err.Error(), "server_error", "", "cron_delete_failed")
		return
	}
	s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "delete", JobID: id, Outcome: "ok"})
	writeJSON(w, http.StatusOK, map[string]any{
		"build": s.buildInfo,
		"ok":    true,
	})
}

func (s *Server) handleLegacyAPIJobsPauseResume(w http.ResponseWriter, r *http.Request, id string, pause bool) {
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.cronMutator == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron job store is not configured", "server_error", "", "cron_store_unavailable")
		return
	}
	action := "resume"
	op := s.cronMutator.Resume
	if pause {
		action = "pause"
		op = s.cronMutator.Pause
	}
	if err := op(r.Context(), id); err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: action, JobID: id, Outcome: "error", Code: "cron_job_missing"})
			writeOpenAIError(w, http.StatusNotFound, "Cron job not found", "invalid_request_error", "id", "cron_job_missing")
			return
		}
		if errors.Is(err, ErrCronJobAlreadyPaused) || errors.Is(err, ErrCronJobNotPaused) {
			s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: action, JobID: id, Outcome: "error", Code: "cron_state_conflict"})
			writeOpenAIError(w, http.StatusConflict, "Cron job state conflict", "invalid_request_error", "", "cron_state_conflict")
			return
		}
		s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: action, JobID: id, Outcome: "error", Code: "cron_state_change_failed"})
		writeOpenAIError(w, http.StatusInternalServerError, "Cron state change failed: "+err.Error(), "server_error", "", "cron_state_change_failed")
		return
	}
	s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: action, JobID: id, Outcome: "ok"})
	if s.cronJobs != nil {
		if job, err := s.cronJobs.Get(id); err == nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"build": s.buildInfo,
				"job":   cronAdminJobViewFor(job, s.now()),
			})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"build":  s.buildInfo,
		"job_id": id,
		"paused": pause,
	})
}

func (s *Server) handleLegacyAPIJobsRun(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.cronTrigger == nil {
		s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "trigger_unavailable", JobID: id, Outcome: "error", Code: "trigger_delivery_unavailable"})
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron trigger delivery is not configured", "server_error", "", "trigger_delivery_unavailable")
		return
	}
	res, err := s.cronTrigger.Trigger(r.Context(), id)
	if err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "trigger", JobID: id, Outcome: "error", Code: "cron_job_missing"})
			writeOpenAIError(w, http.StatusNotFound, "Cron job not found", "invalid_request_error", "id", "cron_job_missing")
			return
		}
		s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "trigger", JobID: id, Outcome: "error", Code: "cron_trigger_failed"})
		writeOpenAIError(w, http.StatusInternalServerError, "Cron trigger failed: "+err.Error(), "server_error", "", "cron_trigger_failed")
		return
	}
	s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "trigger", JobID: id, Outcome: "ok"})
	writeJSON(w, http.StatusOK, map[string]any{
		"build":       s.buildInfo,
		"job_id":      id,
		"run_id":      res.RunID,
		"status":      res.Status,
		"prompt_hash": res.PromptHash,
	})
}

func (s *Server) handleCronAdminUpdate(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.cronMutator == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron job store is not configured", "server_error", "", "cron_store_unavailable")
		return
	}

	body, err := readLimitedBody(w, r, s.maxBodyBytes)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) || errors.Is(err, errBodyTooLarge) {
			writeOpenAIError(w, http.StatusRequestEntityTooLarge, "Request body too large.", "invalid_request_error", "", "body_too_large")
			return
		}
		writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "", "invalid_request_body")
		return
	}
	var spec CronJobSpec
	if err := json.Unmarshal(body, &spec); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "Invalid JSON in request body", "invalid_request_error", "", "invalid_json")
		return
	}
	if status, code, msg := s.validateCronAdminSpec(spec); status != 0 {
		writeOpenAIError(w, status, msg, "invalid_request_error", "", code)
		return
	}

	if err := s.cronMutator.Update(r.Context(), id, spec); err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "update", JobID: id, Outcome: "error", Code: "cron_job_missing"})
			writeOpenAIError(w, http.StatusNotFound, "Cron job not found", "invalid_request_error", "id", "cron_job_missing")
			return
		}
		s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "update", JobID: id, Outcome: "error", Code: "cron_update_failed"})
		writeOpenAIError(w, http.StatusInternalServerError, "Cron job update failed: "+err.Error(), "server_error", "", "cron_update_failed")
		return
	}
	s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "update", JobID: id, Outcome: "ok"})

	job := jobFromSpec(spec)
	job.ID = id
	writeJSON(w, http.StatusOK, map[string]any{
		"build": s.buildInfo,
		"job":   cronAdminJobViewFor(job, s.now()),
	})
}

func (s *Server) handleCronAdminDelete(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.cronMutator == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron job store is not configured", "server_error", "", "cron_store_unavailable")
		return
	}
	if err := s.cronMutator.Delete(r.Context(), id); err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "delete", JobID: id, Outcome: "error", Code: "cron_job_missing"})
			writeOpenAIError(w, http.StatusNotFound, "Cron job not found", "invalid_request_error", "id", "cron_job_missing")
			return
		}
		s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "delete", JobID: id, Outcome: "error", Code: "cron_delete_failed"})
		writeOpenAIError(w, http.StatusInternalServerError, "Cron job delete failed: "+err.Error(), "server_error", "", "cron_delete_failed")
		return
	}
	s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "delete", JobID: id, Outcome: "ok"})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCronAdminPause(w http.ResponseWriter, r *http.Request, id string) {
	s.handleCronAdminPauseResume(w, r, id, true)
}

func (s *Server) handleCronAdminResume(w http.ResponseWriter, r *http.Request, id string) {
	s.handleCronAdminPauseResume(w, r, id, false)
}

func (s *Server) handleCronAdminPauseResume(w http.ResponseWriter, r *http.Request, id string, pause bool) {
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.cronMutator == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron job store is not configured", "server_error", "", "cron_store_unavailable")
		return
	}
	action := "resume"
	conflictCode := "cron_job_not_paused"
	conflictMsg := "Cron job is not paused"
	conflictErr := ErrCronJobNotPaused
	op := s.cronMutator.Resume
	if pause {
		action = "pause"
		conflictCode = "cron_job_already_paused"
		conflictMsg = "Cron job is already paused"
		conflictErr = ErrCronJobAlreadyPaused
		op = s.cronMutator.Pause
	}

	if err := op(r.Context(), id); err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: action, JobID: id, Outcome: "error", Code: "cron_job_missing"})
			writeOpenAIError(w, http.StatusNotFound, "Cron job not found", "invalid_request_error", "id", "cron_job_missing")
			return
		}
		if errors.Is(err, conflictErr) {
			s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: action, JobID: id, Outcome: "error", Code: conflictCode})
			writeOpenAIError(w, http.StatusConflict, conflictMsg, "invalid_request_error", "", conflictCode)
			return
		}
		s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: action, JobID: id, Outcome: "error", Code: "cron_state_change_failed"})
		writeOpenAIError(w, http.StatusInternalServerError, "Cron state change failed: "+err.Error(), "server_error", "", "cron_state_change_failed")
		return
	}

	s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: action, JobID: id, Outcome: "ok"})
	if s.cronJobs != nil {
		if job, err := s.cronJobs.Get(id); err == nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"build": s.buildInfo,
				"job":   cronAdminJobViewFor(job, s.now()),
			})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"build":  s.buildInfo,
		"job_id": id,
		"paused": pause,
	})
}

func (s *Server) handleCronAdminTrigger(w http.ResponseWriter, r *http.Request, id string) {
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.cronTrigger == nil {
		s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "trigger_unavailable", JobID: id, Outcome: "error", Code: "trigger_delivery_unavailable"})
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron trigger delivery is not configured", "server_error", "", "trigger_delivery_unavailable")
		return
	}
	res, err := s.cronTrigger.Trigger(r.Context(), id)
	if err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "trigger", JobID: id, Outcome: "error", Code: "cron_job_missing"})
			writeOpenAIError(w, http.StatusNotFound, "Cron job not found", "invalid_request_error", "id", "cron_job_missing")
			return
		}
		s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "trigger", JobID: id, Outcome: "error", Code: "cron_trigger_failed"})
		writeOpenAIError(w, http.StatusInternalServerError, "Cron trigger failed: "+err.Error(), "server_error", "", "cron_trigger_failed")
		return
	}
	s.recordCronAdminEvent(r.Context(), CronAdminAuditEvent{Action: "trigger", JobID: id, Outcome: "ok"})
	writeJSON(w, http.StatusAccepted, map[string]any{
		"build":       s.buildInfo,
		"job_id":      id,
		"run_id":      res.RunID,
		"status":      res.Status,
		"prompt_hash": res.PromptHash,
	})
}

func (s *Server) recordCronAdminEvent(ctx context.Context, event CronAdminAuditEvent) {
	if s.cronAuditor == nil {
		return
	}
	s.cronAuditor.RecordCronAdminEvent(ctx, event)
}

// validateCronAdminSpec runs the same safety/parser seams the cronjob tool
// uses, so HTTP admin and the tool reject identical inputs. Returns
// (0, "", "") when the spec is valid.
func (s *Server) validateCronAdminSpec(spec CronJobSpec) (status int, code, message string) {
	if strings.TrimSpace(spec.Name) == "" {
		return http.StatusBadRequest, "invalid_request_body", "name is required"
	}
	if len(spec.Name) > legacyAPIMaxNameLen {
		return http.StatusBadRequest, "invalid_name_length", "name must be 200 characters or fewer"
	}
	if strings.TrimSpace(spec.Schedule) == "" {
		return http.StatusBadRequest, "invalid_schedule", "schedule is required"
	}
	if strings.TrimSpace(spec.Prompt) == "" {
		return http.StatusBadRequest, "invalid_request_body", "prompt is required"
	}
	if len(spec.Prompt) > legacyAPIMaxPromptLen {
		return http.StatusBadRequest, "invalid_prompt_length", "prompt must be 5000 characters or fewer"
	}
	if finding, blocked := cron.ScanPromptForCronThreat(spec.Prompt); blocked {
		return http.StatusBadRequest, "unsafe_prompt_rejected", "Prompt rejected by cron safety policy: " + finding.Message
	}
	if _, err := cron.ParseCronSchedule(spec.Schedule, s.scheduleNow()); err != nil {
		var parseErr *cron.ScheduleParseError
		if errors.As(err, &parseErr) {
			return http.StatusBadRequest, "invalid_schedule", "Invalid schedule: " + parseErr.Evidence.Message
		}
		return http.StatusBadRequest, "invalid_schedule", "Invalid schedule: " + err.Error()
	}
	return 0, "", ""
}

type legacyAPISpecError struct {
	status  int
	code    string
	message string
}

func (e *legacyAPISpecError) Error() string {
	return e.message
}

func (s *Server) readLegacyAPICronJobSpec(w http.ResponseWriter, r *http.Request, existing *cron.Job) (CronJobSpec, error) {
	body, err := readLimitedBody(w, r, s.maxBodyBytes)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) || errors.Is(err, errBodyTooLarge) {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusRequestEntityTooLarge, code: "body_too_large", message: "Request body too large."}
		}
		return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_request_body", message: err.Error()}
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_json", message: "Invalid JSON in request body"}
	}

	spec := CronJobSpec{}
	if existing != nil {
		spec = cronJobSpecFromJob(*existing)
		valid := false
		filtered := make(map[string]json.RawMessage, len(raw))
		for key, value := range raw {
			if _, ok := legacyAPIUpdateAllowedFields[key]; ok {
				filtered[key] = value
				valid = true
			}
		}
		if !valid {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_request_body", message: "No valid fields to update"}
		}
		raw = filtered
	}

	if value, ok := raw["name"]; ok {
		if err := json.Unmarshal(value, &spec.Name); err != nil {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_request_body", message: "name must be a string"}
		}
		spec.Name = strings.TrimSpace(spec.Name)
	}
	if value, ok := raw["schedule"]; ok {
		if err := json.Unmarshal(value, &spec.Schedule); err != nil {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_schedule", message: "schedule must be a string"}
		}
		spec.Schedule = strings.TrimSpace(spec.Schedule)
	}
	if value, ok := raw["prompt"]; ok {
		if err := json.Unmarshal(value, &spec.Prompt); err != nil {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_request_body", message: "prompt must be a string"}
		}
	}
	if value, ok := raw["deliver"]; ok {
		var deliver string
		if err := json.Unmarshal(value, &deliver); err != nil {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_request_body", message: "deliver must be a string"}
		}
		spec.Provider = strings.TrimSpace(deliver)
	}
	if value, ok := raw["provider"]; ok {
		if err := json.Unmarshal(value, &spec.Provider); err != nil {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_request_body", message: "provider must be a string"}
		}
		spec.Provider = strings.TrimSpace(spec.Provider)
	}
	if value, ok := raw["model"]; ok {
		if err := json.Unmarshal(value, &spec.Model); err != nil {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_request_body", message: "model must be a string"}
		}
		spec.Model = strings.TrimSpace(spec.Model)
	}
	if value, ok := raw["skills"]; ok {
		if err := json.Unmarshal(value, &spec.Skills); err != nil {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_request_body", message: "skills must be a list of strings"}
		}
	}
	if value, ok := raw["skill"]; ok {
		var skill string
		if err := json.Unmarshal(value, &skill); err != nil {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_request_body", message: "skill must be a string"}
		}
		if strings.TrimSpace(skill) != "" {
			spec.Skills = []string{strings.TrimSpace(skill)}
		}
	}
	if value, ok := raw["repeat"]; ok {
		if err := json.Unmarshal(value, &spec.Repeat); err != nil || spec.Repeat < 1 {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_repeat", message: "repeat must be a positive integer"}
		}
	}
	if value, ok := raw["enabled_toolsets"]; ok {
		if err := json.Unmarshal(value, &spec.EnabledToolsets); err != nil {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_request_body", message: "enabled_toolsets must be a list of strings"}
		}
	}
	if value, ok := raw["workdir"]; ok {
		if err := json.Unmarshal(value, &spec.Workdir); err != nil {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_request_body", message: "workdir must be a string"}
		}
	}
	if value, ok := raw["script"]; ok {
		if err := json.Unmarshal(value, &spec.Script); err != nil {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_request_body", message: "script must be a string"}
		}
	}
	if value, ok := raw["context_from"]; ok {
		if err := json.Unmarshal(value, &spec.ContextFrom); err != nil {
			return CronJobSpec{}, &legacyAPISpecError{status: http.StatusBadRequest, code: "invalid_request_body", message: "context_from must be a list of strings"}
		}
	}

	if status, code, msg := s.validateCronAdminSpec(spec); status != 0 {
		return CronJobSpec{}, &legacyAPISpecError{status: status, code: code, message: msg}
	}
	return spec, nil
}

func (s *Server) writeLegacyAPISpecError(w http.ResponseWriter, err error) {
	var specErr *legacyAPISpecError
	if errors.As(err, &specErr) {
		errType := "invalid_request_error"
		if specErr.status >= 500 {
			errType = "server_error"
		}
		writeOpenAIError(w, specErr.status, specErr.message, errType, "", specErr.code)
		return
	}
	writeOpenAIError(w, http.StatusBadRequest, err.Error(), "invalid_request_error", "", "invalid_request_body")
}

func (s *Server) writeLegacyAPIJobFromStoreOrSpec(w http.ResponseWriter, id string, spec CronJobSpec, status int) {
	if s.cronJobs != nil {
		if job, err := s.cronJobs.Get(id); err == nil {
			writeJSON(w, status, map[string]any{"build": s.buildInfo, "job": cronAdminJobViewFor(job, s.now())})
			return
		}
	}
	job := jobFromSpec(spec)
	job.ID = id
	writeJSON(w, status, map[string]any{"build": s.buildInfo, "job": cronAdminJobViewFor(job, s.now())})
}

func cronJobSpecFromJob(job cron.Job) CronJobSpec {
	return CronJobSpec{
		Name:            job.Name,
		Schedule:        job.Schedule,
		Prompt:          job.Prompt,
		Repeat:          job.Repeat,
		Provider:        job.Provider,
		Model:           job.Model,
		Skills:          append([]string(nil), job.Skills...),
		EnabledToolsets: append([]string(nil), job.EnabledToolsets...),
		Workdir:         job.Workdir,
		Script:          job.Script,
		ContextFrom:     append([]string(nil), job.ContextFrom...),
	}
}

func legacyAPIIncludeDisabled(r *http.Request) bool {
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("include_disabled"))) {
	case "true", "1":
		return true
	default:
		return false
	}
}

func parseLegacyAPIJobPath(path string) (jobID, sub string, ok bool) {
	const prefix = "/api/jobs/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if rest == "" {
		return "", "", false
	}
	parts := strings.Split(rest, "/")
	switch len(parts) {
	case 1:
		return parts[0], "", parts[0] != ""
	case 2:
		return parts[0], parts[1], parts[0] != "" && parts[1] != ""
	default:
		return "", "", false
	}
}

func legacyAPIValidJobID(id string) bool {
	if len(id) != legacyAPIJobIDLength {
		return false
	}
	for _, r := range id {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}

// scheduleNow returns the server's clock seam if configured, falling back to
// time.Now. The native cron schedule parser needs a wall-clock to anchor
// duration / ISO-shaped schedules.
func (s *Server) scheduleNow() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// jobFromSpec builds a partially-populated cron.Job for read-side projection
// after a successful create/update. The store-owned fields (CreatedAt,
// LastRunUnix, etc.) remain whatever the mutator set.
func jobFromSpec(spec CronJobSpec) cron.Job {
	return cron.Job{
		Name:            spec.Name,
		Schedule:        spec.Schedule,
		Prompt:          spec.Prompt,
		Repeat:          spec.Repeat,
		Provider:        spec.Provider,
		Model:           spec.Model,
		Skills:          append([]string(nil), spec.Skills...),
		EnabledToolsets: append([]string(nil), spec.EnabledToolsets...),
		Workdir:         spec.Workdir,
		Script:          spec.Script,
		ContextFrom:     append([]string(nil), spec.ContextFrom...),
	}
}
