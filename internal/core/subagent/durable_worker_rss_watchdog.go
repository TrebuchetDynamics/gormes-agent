package subagent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent/durable"
)

// DurableWorkerRSSWatchdogReason is the root compatibility name for durable.RSSWatchdogReason.
type DurableWorkerRSSWatchdogReason = durable.RSSWatchdogReason

const (
	DurableWorkerRSSWatchdogDisabled    = durable.RSSWatchdogDisabled
	DurableWorkerRSSThresholdExceeded   = durable.RSSThresholdExceeded
	DurableWorkerRSSWatchdogUnavailable = durable.RSSWatchdogUnavailable
	DurableWorkerRSSDrainStarted        = durable.RSSDrainStarted
)

type DurableWorkerWatchdogRestartReason = durable.WatchdogRestartReason

const (
	DurableWorkerStableWatchdogRestart = durable.StableWatchdogRestart
	DurableWorkerWatchdogRestartCrash  = durable.WatchdogRestartCrash
)

// DurableWorkerRSSReader reads RSS in bytes. Tests inject deterministic readers;
// runtime integration can later supply process-specific measurements.
type DurableWorkerRSSReader = durable.RSSReader

// DurableWorkerClock supplies the observation timestamp for policy evidence.
type DurableWorkerClock = durable.Clock

// DurableWorkerRSSWatchdogPolicy is a value-only RSS watchdog configuration.
type DurableWorkerRSSWatchdogPolicy = durable.RSSWatchdogPolicy

// DurableWorkerRSSWatchdogDecision is the pure policy result.
type DurableWorkerRSSWatchdogDecision = durable.RSSWatchdogDecision

// DurableWorkerRSSWatchdogEvidence is suitable for later ledger/status wiring.
type DurableWorkerRSSWatchdogEvidence = durable.RSSWatchdogEvidence

// DurableWorkerWatchdogRestartPolicy classifies supervised watchdog exits.
type DurableWorkerWatchdogRestartPolicy = durable.WatchdogRestartPolicy

// DurableWorkerWatchdogRestartInput is the value-only restart observation.
type DurableWorkerWatchdogRestartInput = durable.WatchdogRestartInput

// DurableWorkerWatchdogRestartDecision is the crash-count policy result.
type DurableWorkerWatchdogRestartDecision = durable.WatchdogRestartDecision

// DurableWorkerRSSWatchdogEvent is an auditable RSS watchdog observation.
type DurableWorkerRSSWatchdogEvent struct {
	JobID     string
	WorkerID  string
	Reason    DurableWorkerRSSWatchdogReason
	Evidence  DurableWorkerRSSWatchdogEvidence
	CreatedAt time.Time
}

// DurableWorkerRSSDrain coordinates graceful RSS drains across concurrent
// DurableWorker.RunOne calls that share the same worker process.
type DurableWorkerRSSDrain struct {
	mu       sync.Mutex
	active   map[string]chan durableWorkerRSSDrainAbort
	draining bool
	abort    durableWorkerRSSDrainAbort
}

type durableWorkerRSSDrainAbort struct {
	Reason   DurableWorkerRSSWatchdogReason
	Evidence DurableWorkerRSSWatchdogEvidence
}

type durableWorkerRSSDrainRegistration struct {
	Abort      <-chan durableWorkerRSSDrainAbort
	unregister func()
}

func NewDurableWorkerRSSDrain() *DurableWorkerRSSDrain {
	return &DurableWorkerRSSDrain{}
}

func (d *DurableWorkerRSSDrain) Register(jobID, workerID string) durableWorkerRSSDrainRegistration {
	if d == nil {
		return durableWorkerRSSDrainRegistration{}
	}
	ch := make(chan durableWorkerRSSDrainAbort, 1)
	key := durableWorkerRSSDrainKey(jobID, workerID)

	d.mu.Lock()
	if d.active == nil {
		d.active = make(map[string]chan durableWorkerRSSDrainAbort)
	}
	d.active[key] = ch
	if d.draining {
		ch <- d.abort
	}
	d.mu.Unlock()

	return durableWorkerRSSDrainRegistration{
		Abort: ch,
		unregister: func() {
			d.mu.Lock()
			delete(d.active, key)
			d.mu.Unlock()
		},
	}
}

func (r durableWorkerRSSDrainRegistration) Unregister() {
	if r.unregister != nil {
		r.unregister()
	}
}

func (d *DurableWorkerRSSDrain) Start(reason DurableWorkerRSSWatchdogReason, evidence DurableWorkerRSSWatchdogEvidence) bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.draining {
		return false
	}
	d.draining = true
	d.abort = durableWorkerRSSDrainAbort{
		Reason:   reason,
		Evidence: evidence,
	}
	for _, ch := range d.active {
		select {
		case ch <- d.abort:
		default:
		}
	}
	return true
}

func durableWorkerRSSDrainKey(jobID, workerID string) string {
	return strings.TrimSpace(workerID) + "\x00" + strings.TrimSpace(jobID)
}

func (l *DurableLedger) RecordWorkerRSSWatchdogEvent(ctx context.Context, event DurableWorkerRSSWatchdogEvent) error {
	if l == nil || l.db == nil {
		return errors.New("subagent: durable ledger is nil")
	}
	workerID := strings.TrimSpace(event.WorkerID)
	if workerID == "" {
		return errors.New("subagent: durable worker id is empty")
	}
	reason := event.Reason
	if reason == "" {
		reason = event.Evidence.Reason
	}
	if reason == "" {
		return errors.New("subagent: durable worker RSS watchdog reason is empty")
	}
	createdAt := event.CreatedAt
	if createdAt.IsZero() {
		createdAt = durableTime(durableNow())
	}
	evidence := event.Evidence
	if evidence.Reason == "" {
		evidence.Reason = reason
	}
	if evidence.CheckedAt.IsZero() {
		evidence.CheckedAt = createdAt.UTC()
	}
	payload := map[string]any{
		"type":      string(reason),
		"job_id":    strings.TrimSpace(event.JobID),
		"worker_id": workerID,
		"reason":    string(reason),
		"evidence":  evidence,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = l.db.ExecContext(ctx, `
		INSERT INTO durable_worker_events
			(type, worker_id, reason, payload_json, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		string(reason), workerID, string(reason), string(raw), createdAt.UTC().UnixNano())
	return err
}
