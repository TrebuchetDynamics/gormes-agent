package cronadmin

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
)

const (
	DefaultRunLimit = 20
	MaxRunLimit     = 200
)

// JobSpec is the operator-supplied shape used by create/update endpoints.
// Field names mirror the existing native cron action envelope so callers can
// serialize the same body across the cronjob tool and HTTP admin.
type JobSpec struct {
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

// JobMutator mutates cron jobs without depending on storage or scheduler
// details. Production implementations adapt *cron.Store; tests inject fakes.
type JobMutator interface {
	Create(ctx context.Context, spec JobSpec) (string, error)
	Update(ctx context.Context, id string, spec JobSpec) error
	Delete(ctx context.Context, id string) error
	Pause(ctx context.Context, id string) error
	Resume(ctx context.Context, id string) error
}

// TriggerHandler launches a one-shot run for a cron job.
type TriggerHandler interface {
	Trigger(ctx context.Context, id string) (TriggerResult, error)
}

// TriggerResult is the operator-visible shape returned by /trigger.
type TriggerResult struct {
	RunID      string `json:"run_id,omitempty"`
	Status     string `json:"status,omitempty"`
	PromptHash string `json:"prompt_hash,omitempty"`
}

// AuditEvent is the redacted audit shape recorded for each cron mutation.
type AuditEvent struct {
	Action  string
	JobID   string
	Outcome string
	Code    string
}

// Auditor records optional redacted cron admin mutation audit events.
type Auditor interface {
	RecordCronAdminEvent(ctx context.Context, event AuditEvent)
}

// JobView is the redacted shape returned by list/get. Prompt and script bodies
// are intentionally omitted to keep secret payloads out of admin listings.
type JobView struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Schedule        string   `json:"schedule"`
	Enabled         bool     `json:"enabled"`
	Paused          bool     `json:"paused"`
	CreatedAt       int64    `json:"created_at"`
	LastRunUnix     int64    `json:"last_run_unix"`
	LastStatus      string   `json:"last_status"`
	NextRunUnix     int64    `json:"next_run_unix"`
	Target          string   `json:"target"`
	Provider        string   `json:"provider,omitempty"`
	Model           string   `json:"model,omitempty"`
	Repeat          int      `json:"repeat,omitempty"`
	RepeatCompleted int      `json:"repeat_completed,omitempty"`
	Skills          []string `json:"skills,omitempty"`
	EnabledToolsets []string `json:"enabled_toolsets,omitempty"`
	HasScript       bool     `json:"has_script"`
}

// RunView is the redacted shape returned by run-history. Prompt hashes, output
// previews, and error messages are dropped because they may contain user prompt
// content or secret values from upstream tool runs.
type RunView struct {
	ID                int64  `json:"id"`
	JobID             string `json:"job_id"`
	StartedAt         int64  `json:"started_at"`
	FinishedAt        int64  `json:"finished_at"`
	Status            string `json:"status"`
	Delivered         bool   `json:"delivered"`
	SuppressionReason string `json:"suppression_reason,omitempty"`
}

func ParseLimit(raw string) int {
	if raw == "" {
		return DefaultRunLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return DefaultRunLimit
	}
	if n > MaxRunLimit {
		return MaxRunLimit
	}
	return n
}

func JobViewFor(job cron.Job, now time.Time) JobView {
	job = cron.NormalizeJobRecord(job, job.ID)
	view := JobView{
		ID:              job.ID,
		Name:            job.Name,
		Schedule:        job.Schedule,
		Enabled:         !job.Paused,
		Paused:          job.Paused,
		CreatedAt:       job.CreatedAt,
		LastRunUnix:     job.LastRunUnix,
		LastStatus:      job.LastStatus,
		Provider:        job.Provider,
		Model:           job.Model,
		Repeat:          job.Repeat,
		RepeatCompleted: job.RepeatCompleted,
		Skills:          append([]string(nil), job.Skills...),
		EnabledToolsets: append([]string(nil), job.EnabledToolsets...),
		HasScript:       strings.TrimSpace(job.Script) != "",
	}
	view.Target = TargetFor(job)
	view.NextRunUnix = NextRunUnix(job, now)
	return view
}

// TargetFor mirrors what an operator would call the delivery target for a job
// without leaking prompt or script bodies.
func TargetFor(job cron.Job) string {
	if t := strings.TrimSpace(job.Provider); t != "" {
		return t
	}
	return ""
}

// NextRunUnix uses the existing pure schedule parser to compute the projected
// next-run timestamp without starting any scheduler goroutines. It returns 0
// when the schedule cannot be parsed or is exhausted.
func NextRunUnix(job cron.Job, now time.Time) int64 {
	if job.Paused {
		return 0
	}
	parsed, err := cron.ParseCronSchedule(job.Schedule, now)
	if err != nil {
		return 0
	}
	decision := cron.CronNextRunDecision(parsed, job.LastRunUnix, job.RepeatCompleted, now)
	if decision.NextRun.IsZero() {
		return 0
	}
	return decision.NextRun.Unix()
}

func RunViewFor(run cron.Run) RunView {
	return RunView{
		ID:                run.ID,
		JobID:             run.JobID,
		StartedAt:         run.StartedAt,
		FinishedAt:        run.FinishedAt,
		Status:            run.Status,
		Delivered:         run.Delivered,
		SuppressionReason: run.SuppressionReason,
	}
}
