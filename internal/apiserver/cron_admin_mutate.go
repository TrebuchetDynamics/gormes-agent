package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cron"
)

// CronJobSpec is the operator-supplied shape used by create/update endpoints.
// Field names mirror the existing native cron action envelope so future
// callers can serialize the same body across the cronjob tool and HTTP admin.
type CronJobSpec struct {
	Name            string   `json:"name"`
	Schedule        string   `json:"schedule"`
	Prompt          string   `json:"prompt"`
	Repeat          int      `json:"repeat,omitempty"`
	Provider        string   `json:"provider,omitempty"`
	Model           string   `json:"model,omitempty"`
	Skills          []string `json:"skills,omitempty"`
	EnabledToolsets []string `json:"enabled_toolsets,omitempty"`
	Workdir         string   `json:"workdir,omitempty"`
	Script          string   `json:"script,omitempty"`
	ContextFrom     []string `json:"context_from,omitempty"`
}

// CronJobMutator is the narrow write facade the API server uses to mutate
// cron jobs without depending on bbolt or the scheduler. Production
// implementations adapt *cron.Store; tests inject a fake.
//
// The mutator MUST NOT start the scheduler, dispatch a delivery, or contact
// providers. Read traffic continues to flow through CronJobReader.
type CronJobMutator interface {
	Create(ctx context.Context, spec CronJobSpec) (string, error)
	Update(ctx context.Context, id string, spec CronJobSpec) error
	Delete(ctx context.Context, id string) error
	Pause(ctx context.Context, id string) error
	Resume(ctx context.Context, id string) error
}

// CronTriggerHandler is the trigger seam the API server uses to launch a
// one-shot run. A nil seam represents "delivery unavailable" and the trigger
// endpoint records trigger_delivery_unavailable + 503.
type CronTriggerHandler interface {
	Trigger(ctx context.Context, id string) (TriggerResult, error)
}

// TriggerResult is the operator-visible shape returned by /trigger.
// It deliberately avoids leaking provider/gateway internals.
type TriggerResult struct {
	RunID      string `json:"run_id,omitempty"`
	Status     string `json:"status,omitempty"`
	PromptHash string `json:"prompt_hash,omitempty"`
}

// CronAdminAuditEvent is the audit shape recorded for each cron mutation.
// It is intentionally redacted: only stable identifiers and outcome are kept,
// no prompt or script bodies are forwarded to the auditor.
type CronAdminAuditEvent struct {
	Action  string
	JobID   string
	Outcome string
	Code    string
}

// CronAdminAuditor is the audit seam wired into the apiserver Config. It is
// optional; when nil, mutating endpoints still work but do not record events.
type CronAdminAuditor interface {
	RecordCronAdminEvent(ctx context.Context, event CronAdminAuditEvent)
}

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
	writeJSON(w, http.StatusCreated, map[string]any{"job": cronAdminJobViewFor(job, s.now())})
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
	writeJSON(w, http.StatusOK, map[string]any{"job": cronAdminJobViewFor(job, s.now())})
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
			writeJSON(w, http.StatusOK, map[string]any{"job": cronAdminJobViewFor(job, s.now())})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"job_id": id, "paused": pause})
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
	if strings.TrimSpace(spec.Schedule) == "" {
		return http.StatusBadRequest, "invalid_schedule", "schedule is required"
	}
	if strings.TrimSpace(spec.Prompt) == "" {
		return http.StatusBadRequest, "invalid_request_body", "prompt is required"
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

