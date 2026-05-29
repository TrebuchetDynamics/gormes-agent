package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/audit"
)

const (
	TrajectoryWriteCompleted  = "trajectory_written"
	TrajectoryWriteDisabled   = "trajectory_disabled"
	TrajectoryWriteFailed     = "trajectory_write_failed"
	TrajectoryRedactionFailed = "trajectory_redaction_failed"
	defaultTrajectorySamples  = "trajectory_samples.jsonl"
	defaultFailedTrajectories = "failed_trajectories.jsonl"
	scratchpadOpenTag         = "<REASONING_SCRATCHPAD>"
	scratchpadCloseTag        = "</REASONING_SCRATCHPAD>"
	trajectoryThinkOpenTag    = "<think>"
	trajectoryThinkCloseTag   = "</think>"
)

// TrajectoryWriter appends Hermes-compatible trajectory JSONL entries.
type TrajectoryWriter struct {
	samplesPath string
	failedPath  string
	now         func() time.Time
	audit       audit.Recorder
}

type TrajectoryWriterOptions struct {
	SamplesPath string
	FailedPath  string
	Now         func() time.Time
	Audit       audit.Recorder
}

type TrajectoryWriteInput struct {
	Conversations []TrajectoryTurn
	SessionID     string
	Model         string
	Completed     bool
}

type TrajectoryWriteEvidence struct {
	Code      string
	Path      string
	Completed bool
	Redacted  bool
	Error     string
}

type trajectoryWriteEntry struct {
	Conversations []TrajectoryTurn `json:"conversations"`
	Timestamp     string           `json:"timestamp"`
	Model         string           `json:"model"`
	Completed     bool             `json:"completed"`
}

func NewTrajectoryWriter(opts TrajectoryWriterOptions) TrajectoryWriter {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return TrajectoryWriter{
		samplesPath: strings.TrimSpace(opts.SamplesPath),
		failedPath:  strings.TrimSpace(opts.FailedPath),
		now:         opts.Now,
		audit:       opts.Audit,
	}
}

func (w TrajectoryWriter) Write(in TrajectoryWriteInput) TrajectoryWriteEvidence {
	completed := in.Completed
	if trajectoryHasIncompleteScratchpad(in.Conversations) {
		completed = false
	}
	path := w.samplesPath
	if !completed {
		path = w.failedPath
	}
	if path == "" {
		ev := TrajectoryWriteEvidence{Code: TrajectoryWriteDisabled, Completed: completed, Redacted: true}
		w.recordAudit(in, ev, time.Time{})
		return ev
	}

	now := w.now().UTC()
	entry := trajectoryWriteEntry{
		Conversations: normalizeTrajectoryWriterTurns(in.Conversations),
		Timestamp:     now.Format(time.RFC3339Nano),
		Model:         audit.RedactText(strings.TrimSpace(in.Model)),
		Completed:     completed,
	}
	line, err := json.Marshal(entry)
	if err != nil {
		ev := TrajectoryWriteEvidence{Code: TrajectoryWriteFailed, Path: audit.RedactText(path), Completed: completed, Redacted: true, Error: audit.RedactText(err.Error())}
		w.recordAudit(in, ev, now)
		return ev
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		ev := TrajectoryWriteEvidence{Code: TrajectoryWriteFailed, Path: audit.RedactText(path), Completed: completed, Redacted: true, Error: audit.RedactText(err.Error())}
		w.recordAudit(in, ev, now)
		return ev
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		ev := TrajectoryWriteEvidence{Code: TrajectoryWriteFailed, Path: audit.RedactText(path), Completed: completed, Redacted: true, Error: audit.RedactText(err.Error())}
		w.recordAudit(in, ev, now)
		return ev
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		ev := TrajectoryWriteEvidence{Code: TrajectoryWriteFailed, Path: audit.RedactText(path), Completed: completed, Redacted: true, Error: audit.RedactText(err.Error())}
		w.recordAudit(in, ev, now)
		return ev
	}
	ev := TrajectoryWriteEvidence{Code: TrajectoryWriteCompleted, Path: audit.RedactText(path), Completed: completed, Redacted: true}
	w.recordAudit(in, ev, now)
	return ev
}

func (w TrajectoryWriter) recordAudit(in TrajectoryWriteInput, ev TrajectoryWriteEvidence, ts time.Time) {
	if w.audit == nil {
		return
	}
	_ = w.audit.Record(audit.TrajectoryWriteAuditRecord(audit.TrajectoryWriteAuditInput{
		Timestamp: ts,
		SessionID: in.SessionID,
		Model:     in.Model,
		Path:      ev.Path,
		Code:      ev.Code,
		Completed: ev.Completed,
		Redacted:  ev.Redacted,
		Error:     ev.Error,
	}))
}

func normalizeTrajectoryWriterTurns(turns []TrajectoryTurn) []TrajectoryTurn {
	out := make([]TrajectoryTurn, len(turns))
	for i, turn := range turns {
		out[i] = turn
		out[i].From = strings.TrimSpace(out[i].From)
		out[i].Value = audit.RedactText(ConvertScratchpadToThink(out[i].Value))
	}
	return out
}

func trajectoryHasIncompleteScratchpad(turns []TrajectoryTurn) bool {
	for _, turn := range turns {
		if HasIncompleteScratchpad(turn.Value) {
			return true
		}
	}
	return false
}

func ConvertScratchpadToThink(content string) string {
	if content == "" || !strings.Contains(content, scratchpadOpenTag) {
		return content
	}
	content = strings.ReplaceAll(content, scratchpadOpenTag, trajectoryThinkOpenTag)
	return strings.ReplaceAll(content, scratchpadCloseTag, trajectoryThinkCloseTag)
}

func HasIncompleteScratchpad(content string) bool {
	return content != "" && strings.Contains(content, scratchpadOpenTag) && !strings.Contains(content, scratchpadCloseTag)
}
