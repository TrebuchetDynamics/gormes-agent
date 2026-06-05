package apiserver

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver/cronadmin"
	"github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
)

// CronJobReader is the narrow read facade the API server uses to list and
// fetch cron jobs without depending on bbolt directly. The production
// implementation is *cron.Store, which already satisfies this interface.
//
// The read-only cron admin endpoints intentionally do not expose a write
// facade — mutating endpoints are a dependent row.
type CronJobReader interface {
	List() ([]cron.Job, error)
	Get(id string) (cron.Job, error)
}

// CronRunReader is the narrow read facade for the cron run audit log used by
// the run-history endpoint. *cron.RunStore satisfies it.
type CronRunReader interface {
	LatestRuns(ctx context.Context, jobID string, limit int) ([]cron.Run, error)
}

type cronAdminJobView = cronadmin.JobView

type cronAdminRunView = cronadmin.RunView

func (s *Server) handleCronAdminJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// fall through to read path below
	case http.MethodPost:
		s.handleCronAdminCreate(w, r)
		return
	default:
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
		return
	}
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
	now := s.now()
	views := make([]cronAdminJobView, 0, len(jobs))
	for _, job := range jobs {
		views = append(views, cronAdminJobViewFor(job, now))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"build": s.buildInfo,
		"jobs":  views,
	})
}

func (s *Server) handleCronAdminJobByID(w http.ResponseWriter, r *http.Request) {
	jobID, sub, ok := parseCronAdminJobPath(r.URL.Path)
	if !ok {
		writeOpenAIError(w, http.StatusNotFound, "Cron admin route not found", "invalid_request_error", "", "not_found")
		return
	}

	// Mutating subroutes are dispatched first so auth + envelope handling
	// stays consistent with the read path: /pause, /resume, /trigger.
	switch sub {
	case "pause":
		if r.Method != http.MethodPost {
			writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
			return
		}
		s.handleCronAdminPause(w, r, jobID)
		return
	case "resume":
		if r.Method != http.MethodPost {
			writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
			return
		}
		s.handleCronAdminResume(w, r, jobID)
		return
	case "trigger":
		if r.Method != http.MethodPost {
			writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
			return
		}
		s.handleCronAdminTrigger(w, r, jobID)
		return
	}

	// Top-level /v1/admin/cron/jobs/{id} accepts GET (read), PATCH/PUT
	// (update), and DELETE (remove). The "runs" subresource remains GET-only.
	if sub == "" {
		switch r.Method {
		case http.MethodPatch, http.MethodPut:
			s.handleCronAdminUpdate(w, r, jobID)
			return
		case http.MethodDelete:
			s.handleCronAdminDelete(w, r, jobID)
			return
		case http.MethodGet:
			// fall through to read path below
		default:
			writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
			return
		}
	} else if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
		return
	}

	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.cronJobs == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron job store is not configured", "server_error", "", "cron_store_unavailable")
		return
	}

	switch sub {
	case "":
		s.respondCronAdminJob(w, r, jobID)
	case "runs":
		s.respondCronAdminRunHistory(w, r, jobID)
	default:
		writeOpenAIError(w, http.StatusNotFound, "Cron admin route not found", "invalid_request_error", "", "not_found")
	}
}

func (s *Server) respondCronAdminJob(w http.ResponseWriter, _ *http.Request, jobID string) {
	job, err := s.cronJobs.Get(jobID)
	if err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			writeOpenAIError(w, http.StatusNotFound, "Cron job not found", "invalid_request_error", "id", "cron_job_missing")
			return
		}
		writeOpenAIError(w, http.StatusInternalServerError, "Cron job lookup failed: "+err.Error(), "server_error", "", "cron_store_unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"build": s.buildInfo,
		"job":   cronAdminJobViewFor(job, s.now()),
	})
}

func (s *Server) respondCronAdminRunHistory(w http.ResponseWriter, r *http.Request, jobID string) {
	// Verify job exists before reading runs so unknown IDs return the same
	// 404 envelope as /v1/admin/cron/jobs/{id}.
	if _, err := s.cronJobs.Get(jobID); err != nil {
		if errors.Is(err, cron.ErrJobNotFound) {
			writeOpenAIError(w, http.StatusNotFound, "Cron job not found", "invalid_request_error", "id", "cron_job_missing")
			return
		}
		writeOpenAIError(w, http.StatusInternalServerError, "Cron job lookup failed: "+err.Error(), "server_error", "", "cron_store_unavailable")
		return
	}
	if s.cronRuns == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Cron run audit is not configured", "server_error", "", "cron_runs_unavailable")
		return
	}
	limit := cronAdminParseLimit(r.URL.Query().Get("limit"))
	rows, err := s.cronRuns.LatestRuns(r.Context(), jobID, limit)
	if err != nil {
		writeOpenAIError(w, http.StatusInternalServerError, "Cron run audit unavailable: "+err.Error(), "server_error", "", "cron_runs_unavailable")
		return
	}
	views := make([]cronAdminRunView, 0, len(rows))
	for _, run := range rows {
		views = append(views, cronAdminRunViewFor(run))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"build":  s.buildInfo,
		"job_id": jobID,
		"runs":   views,
		"limit":  limit,
	})
}

// parseCronAdminJobPath splits paths shaped like
// /v1/admin/cron/jobs/{id} or /v1/admin/cron/jobs/{id}/runs into
// (jobID, subresource). Returns ok=false for malformed paths.
func parseCronAdminJobPath(path string) (jobID, sub string, ok bool) {
	const prefix = "/v1/admin/cron/jobs/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimSuffix(rest, "/")
	if rest == "" {
		return "", "", false
	}
	parts := strings.Split(rest, "/")
	switch len(parts) {
	case 1:
		if parts[0] == "" {
			return "", "", false
		}
		return parts[0], "", true
	case 2:
		if parts[0] == "" || parts[1] == "" {
			return "", "", false
		}
		return parts[0], parts[1], true
	default:
		return "", "", false
	}
}

func cronAdminParseLimit(raw string) int {
	return cronadmin.ParseLimit(raw)
}

func cronAdminJobViewFor(job cron.Job, now time.Time) cronAdminJobView {
	return cronadmin.JobViewFor(job, now)
}

func cronAdminTargetFor(job cron.Job) string {
	return cronadmin.TargetFor(job)
}

func cronAdminNextRunUnix(job cron.Job, now time.Time) int64 {
	return cronadmin.NextRunUnix(job, now)
}

func cronAdminRunViewFor(run cron.Run) cronAdminRunView {
	return cronadmin.RunViewFor(run)
}
