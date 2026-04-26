package builderloop

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

const maxJobOutputTailBytes = 1200

var secretTelemetryPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization:\s*bearer\s+)[^\s]+`),
	regexp.MustCompile(`(?i)((?:api[_-]?key|token|secret|password)\s*=\s*)[^\s]+`),
}

type jobSpec struct {
	ID      string
	Kind    string
	Attempt int
	Command string
	Dir     string
	Worker  int
	Task    string
	Branch  string
}

func runLoggedJob(ctx context.Context, cfg Config, runner Runner, runID string, spec jobSpec, command Command) Result {
	start := time.Now().UTC()
	spec = normalizeJobSpec(spec, command)
	_ = appendRunLedgerEvent(cfg, LedgerEvent{
		TS:        start,
		RunID:     runID,
		Event:     "job_started",
		Worker:    spec.Worker,
		Task:      spec.Task,
		Branch:    spec.Branch,
		Status:    "started",
		JobID:     spec.ID,
		JobKind:   spec.Kind,
		Attempt:   spec.Attempt,
		Command:   spec.Command,
		Dir:       spec.Dir,
		StartedAt: start.Format(time.RFC3339Nano),
	})

	result := runner.Run(ctx, command)
	finished := time.Now().UTC()
	durationMS := finished.Sub(start).Milliseconds()
	if durationMS <= 0 {
		durationMS = 1
	}
	event := LedgerEvent{
		TS:          finished,
		RunID:       runID,
		Event:       "job_finished",
		Worker:      spec.Worker,
		Task:        spec.Task,
		Branch:      spec.Branch,
		Status:      jobStatus(result.Err),
		JobID:       spec.ID,
		JobKind:     spec.Kind,
		Attempt:     spec.Attempt,
		Command:     spec.Command,
		Dir:         spec.Dir,
		StartedAt:   start.Format(time.RFC3339Nano),
		DurationMS:  durationMS,
		StdoutBytes: len(result.Stdout),
		StderrBytes: len(result.Stderr),
	}
	if result.Err != nil {
		event.ExitError = result.Err.Error()
		event.StdoutTail = boundedRedactedTail(result.Stdout)
		event.StderrTail = boundedRedactedTail(result.Stderr)
	}
	_ = appendRunLedgerEvent(cfg, event)
	return result
}

func normalizeJobSpec(spec jobSpec, command Command) jobSpec {
	if spec.Command == "" {
		spec.Command = command.Name
		if len(command.Args) > 0 {
			spec.Command += " " + strings.Join(command.Args, " ")
		}
	}
	spec.Command = sanitizeTelemetryCommand(spec.Command)
	if spec.Dir == "" {
		spec.Dir = command.Dir
	}
	return spec
}

func sanitizeTelemetryCommand(command string) string {
	command = redactTelemetry(command)
	const maxCommandBytes = 240
	if len(command) <= maxCommandBytes {
		return command
	}
	return strings.TrimSpace(command[:maxCommandBytes]) + " ... [truncated]"
}

func jobStatus(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "failed"
	}
}

func tailString(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[len(value)-limit:]
}

func boundedRedactedTail(value string) string {
	return tailString(redactTelemetry(tailString(value, maxJobOutputTailBytes)), maxJobOutputTailBytes)
}

func redactTelemetry(value string) string {
	out := value
	for _, pattern := range secretTelemetryPatterns {
		out = pattern.ReplaceAllString(out, "${1}[REDACTED]")
	}
	return out
}
