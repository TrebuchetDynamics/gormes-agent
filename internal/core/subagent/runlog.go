package subagent

import (
	"time"

	logpolicy "github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent/runlog"
)

type runLogger struct {
	inner *logpolicy.Logger
}

func newRunLogger(path string) *runLogger {
	logger := logpolicy.New(path)
	if logger == nil {
		return nil
	}
	return &runLogger{inner: logger}
}

func (l *runLogger) append(sa *Subagent, result *SubagentResult) error {
	return l.inner.Append(logpolicy.Record{
		ID:         result.ID,
		ParentID:   sa.ParentID,
		Depth:      sa.Depth,
		Goal:       sa.cfg.Goal,
		Status:     string(result.Status),
		ExitReason: result.ExitReason,
		DurationMs: result.Duration.Milliseconds(),
		Iterations: result.Iterations,
		Error:      result.Error,
		FinishedAt: time.Now().UTC(),
	})
}
