package release

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

// ReleaseEvidenceCode classifies one resource-release outcome inside a cron
// run's release ledger. The vocabulary mirrors the per-run evidence emitted
// by upstream Hermes 9b55365f's run_job/agent.close() reaping; Gormes will
// later surface these into run_completion / heartbeat without changing the
// codes.
type ReleaseEvidenceCode string

const (
	ReleaseEvidenceSubprocessKilled     ReleaseEvidenceCode = "cron_release_subprocess_killed"
	ReleaseEvidenceSessionDBClosed      ReleaseEvidenceCode = "cron_release_session_db_closed"
	ReleaseEvidenceHTTPIdleClosed       ReleaseEvidenceCode = "cron_release_http_idle_closed"
	ReleaseEvidenceHTTPIdleClosedFailed ReleaseEvidenceCode = "cron_release_http_idle_closed_failed"
	ReleaseEvidenceSkippedNoResource    ReleaseEvidenceCode = "cron_release_skipped_no_resource"
)

// ReleaseEvidence is one entry in the per-run release log. Fields carries
// optional structured context (pid, label, error) without coupling the
// helper to a specific telemetry sink.
type ReleaseEvidence struct {
	Code   ReleaseEvidenceCode
	Label  string
	Fields map[string]any
}

// SubprocessKiller is the narrow seam used to terminate spawned tool
// subprocesses without depending on os.Process inside the helper. Unit
// tests pass a fake; future Executor binding will pass a real syscall-backed
// implementation.
type SubprocessKiller interface {
	Kill(pid int) error
}

type closerEntry struct {
	label  string
	closer io.Closer
	kind   ReleaseEvidenceCode
}

// RunReleaseLedger records the per-run resources a cron Executor.Run
// acquired and releases them deterministically when Release is called.
//
// Registration order is preserved on release; partial failures continue to
// the next entry; a second Release on the same ledger is a no-op recorded
// as cron_release_skipped_no_resource. The helper does not start a kernel
// turn, hit a provider, manage MCP stdio runtimes, or change the Scheduler
// goroutine lifecycle — Executor binding is a dependent row.
type RunReleaseLedger struct {
	mu       sync.Mutex
	closers  []closerEntry
	pids     []int
	released bool
}

// NewRunReleaseLedger constructs an empty ledger. Callers register
// resources through Register* and call Release exactly once at run
// completion (success, kernel-error, ctx-cancel, or timeout paths).
func NewRunReleaseLedger() *RunReleaseLedger {
	return &RunReleaseLedger{}
}

// RegisterCloser records an io.Closer the executor opened for this run
// (typically a per-session SQLite store). label is used as evidence Label.
func (l *RunReleaseLedger) RegisterCloser(label string, closer io.Closer) {
	if closer == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closers = append(l.closers, closerEntry{label: label, closer: closer, kind: ReleaseEvidenceSessionDBClosed})
}

// RegisterIdleClosable records an outbound HTTP RoundTripper or other
// idle-conn closable that should be reaped at run end.
func (l *RunReleaseLedger) RegisterIdleClosable(label string, closer io.Closer) {
	if closer == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closers = append(l.closers, closerEntry{label: label, closer: closer, kind: ReleaseEvidenceHTTPIdleClosed})
}

// RegisterSubprocess records a tool-subprocess PID to terminate at run
// end. The helper does not spawn or own the subprocess; it only forwards
// the PID to a SubprocessKiller passed into Release.
func (l *RunReleaseLedger) RegisterSubprocess(pid int) {
	if pid <= 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pids = append(l.pids, pid)
}

// Release closes every registered resource in registration order. A nil
// killer is treated as no-op for subprocesses. Errors are aggregated and
// returned via errors.Join while the helper continues to the next entry.
// A second call returns a single skipped_no_resource evidence entry and
// does not double-release.
func (l *RunReleaseLedger) Release(killer SubprocessKiller) ([]ReleaseEvidence, error) {
	l.mu.Lock()
	if l.released {
		l.mu.Unlock()
		return []ReleaseEvidence{{Code: ReleaseEvidenceSkippedNoResource, Fields: map[string]any{"reason": "already_released"}}}, nil
	}
	l.released = true
	closers := l.closers
	pids := l.pids
	l.closers = nil
	l.pids = nil
	l.mu.Unlock()

	if len(closers) == 0 && len(pids) == 0 {
		return []ReleaseEvidence{{Code: ReleaseEvidenceSkippedNoResource, Fields: map[string]any{"reason": "no_resources_registered"}}}, nil
	}

	evidence := make([]ReleaseEvidence, 0, len(closers)+len(pids))
	var errs []error

	for _, entry := range closers {
		err := entry.closer.Close()
		if err != nil {
			failedCode := releaseFailureCode(entry.kind)
			evidence = append(evidence, ReleaseEvidence{
				Code:   failedCode,
				Label:  entry.label,
				Fields: map[string]any{"error": err.Error()},
			})
			errs = append(errs, fmt.Errorf("%s %q: %w", entry.kind, entry.label, err))
			continue
		}
		evidence = append(evidence, ReleaseEvidence{
			Code:   entry.kind,
			Label:  entry.label,
			Fields: map[string]any{},
		})
	}

	for _, pid := range pids {
		fields := map[string]any{"pid": pid}
		if killer == nil {
			fields["error"] = "no killer supplied"
			evidence = append(evidence, ReleaseEvidence{Code: ReleaseEvidenceSubprocessKilled, Fields: fields})
			continue
		}
		if err := killer.Kill(pid); err != nil {
			fields["error"] = err.Error()
			evidence = append(evidence, ReleaseEvidence{Code: ReleaseEvidenceSubprocessKilled, Fields: fields})
			errs = append(errs, fmt.Errorf("kill pid %d: %w", pid, err))
			continue
		}
		evidence = append(evidence, ReleaseEvidence{Code: ReleaseEvidenceSubprocessKilled, Fields: fields})
	}

	if len(errs) > 0 {
		return evidence, errors.Join(errs...)
	}
	return evidence, nil
}

func releaseFailureCode(kind ReleaseEvidenceCode) ReleaseEvidenceCode {
	switch kind {
	case ReleaseEvidenceHTTPIdleClosed:
		return ReleaseEvidenceHTTPIdleClosedFailed
	case ReleaseEvidenceSessionDBClosed:
		return ReleaseEvidenceCode("cron_release_session_db_closed_failed")
	default:
		return ReleaseEvidenceCode(string(kind) + "_failed")
	}
}
