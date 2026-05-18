package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const runtimeStatusKind = "gormes-gateway"

// GatewayState is the process-level lifecycle state persisted in
// gateway_state.json for operator readouts.
type GatewayState string

const (
	GatewayStateStarting      GatewayState = "starting"
	GatewayStateRunning       GatewayState = "running"
	GatewayStateDraining      GatewayState = "draining"
	GatewayStateStopped       GatewayState = "stopped"
	GatewayStateStartupFailed GatewayState = "startup_failed"
)

// PlatformState is the per-channel lifecycle state persisted alongside the
// process-level gateway state.
type PlatformState string

const (
	PlatformStateStarting PlatformState = "starting"
	PlatformStateRunning  PlatformState = "running"
	PlatformStateStopped  PlatformState = "stopped"
	PlatformStateFailed   PlatformState = "failed"
	// PlatformStatePaused marks a platform the per-platform circuit breaker
	// (or a manual /platform pause) has parked: it stays in the failed/retry
	// set but the reconnect watcher no longer hammers it until resumed.
	PlatformStatePaused PlatformState = "paused"
)

// RuntimeStatus is the shared gateway status read model.
type RuntimeStatus struct {
	Kind                      string                                     `json:"kind"`
	PID                       int                                        `json:"pid"`
	StartTime                 int64                                      `json:"start_time,omitempty"`
	Generation                uint64                                     `json:"generation"`
	BootGitSHA                string                                     `json:"boot_git_sha,omitempty"`
	SourceRoot                string                                     `json:"source_root,omitempty"`
	Command                   string                                     `json:"command,omitempty"`
	Argv                      []string                                   `json:"argv,omitempty"`
	ProcessValidation         RuntimeProcessValidation                   `json:"process_validation,omitempty"`
	StaleCode                 *RuntimeStaleCodeEvidence                  `json:"stale_code,omitempty"`
	GatewayState              GatewayState                               `json:"gateway_state"`
	ExitReason                string                                     `json:"exit_reason"`
	RestartRequested          bool                                       `json:"restart_requested"`
	ActiveAgents              int                                        `json:"active_agents"`
	Platforms                 map[string]PlatformRuntimeStatus           `json:"platforms"`
	Proxy                     ProxyRuntimeStatus                         `json:"proxy"`
	KanbanDispatcher          KanbanDispatcherStatus                     `json:"kanban_dispatcher"`
	TokenLocks                []TokenLockEvidence                        `json:"token_locks,omitempty"`
	MemoryPressure            RuntimeMemoryPressureEvidence              `json:"memory_pressure,omitempty"`
	DrainTimeouts             []RuntimeDrainTimeoutEvidence              `json:"drain_timeouts,omitempty"`
	ResumePending             []RuntimeResumePendingEvidence             `json:"resume_pending,omitempty"`
	NonResumable              []RuntimeNonResumableEvidence              `json:"non_resumable,omitempty"`
	ExpiryFinalized           []RuntimeExpiryFinalizedEvidence           `json:"expiry_finalized,omitempty"`
	ExpiryFinalize            []RuntimeExpiryFinalizeEvidence            `json:"expiry_finalize,omitempty"`
	TakeoverMarkers           []RuntimeRestartTakeoverEvidence           `json:"takeover_marker_seen,omitempty"`
	DuplicateRestarts         []RuntimeRestartDuplicateEvidence          `json:"duplicate_restart_suppressed,omitempty"`
	ServiceManagerUnavailable []RuntimeServiceManagerUnavailableEvidence `json:"service_manager_unavailable,omitempty"`
	ConfigReload              RuntimeConfigReloadEvidence                `json:"config_reload,omitempty"`
	UpdatedAt                 string                                     `json:"updated_at"`
}

type RuntimeConfigReloadStatus string

const (
	RuntimeConfigReloadApplied RuntimeConfigReloadStatus = "applied"
	RuntimeConfigReloadFailed  RuntimeConfigReloadStatus = "failed"
)

type RuntimeConfigReloadEvidence struct {
	Status     RuntimeConfigReloadStatus `json:"status,omitempty"`
	Generation uint64                    `json:"generation,omitempty"`
	Error      string                    `json:"error,omitempty"`
	AppliedAt  string                    `json:"applied_at,omitempty"`
	FailedAt   string                    `json:"failed_at,omitempty"`
	Redacted   bool                      `json:"redacted"`
}

// RuntimeProcessValidationStatus classifies how much trust callers can place
// in the PID identity stored next to gateway_state.json.
type RuntimeProcessValidationStatus string

const (
	RuntimeProcessValidationMissingState     RuntimeProcessValidationStatus = "missing_state"
	RuntimeProcessValidationMissingPIDFile   RuntimeProcessValidationStatus = "missing_pid_file"
	RuntimeProcessValidationStalePID         RuntimeProcessValidationStatus = "stale_pid"
	RuntimeProcessValidationPIDReused        RuntimeProcessValidationStatus = "pid_reused"
	RuntimeProcessValidationStopped          RuntimeProcessValidationStatus = "stopped_process"
	RuntimeProcessValidationPermissionDenied RuntimeProcessValidationStatus = "permission_denied"
	RuntimeProcessValidationLive             RuntimeProcessValidationStatus = "live"
)

// RuntimeProcessValidation is read-only evidence produced when a runtime
// status snapshot is checked against process identity evidence.
type RuntimeProcessValidation struct {
	Status            RuntimeProcessValidationStatus `json:"status,omitempty"`
	Live              bool                           `json:"live"`
	Message           string                         `json:"message,omitempty"`
	PID               int                            `json:"pid,omitempty"`
	ExpectedStartTime int64                          `json:"expected_start_time,omitempty"`
	ActualStartTime   int64                          `json:"actual_start_time,omitempty"`
	Command           string                         `json:"command,omitempty"`
	CheckedAt         string                         `json:"checked_at,omitempty"`
}

// PlatformRuntimeStatus is one platform/channel's status entry inside the
// shared runtime status model.
type PlatformRuntimeStatus struct {
	State        PlatformState `json:"state"`
	ErrorMessage string        `json:"error_message"`
	UpdatedAt    string        `json:"updated_at"`
}

// ProxyRuntimeStatus reports gateway proxy mode health for operator readouts.
type ProxyRuntimeStatus struct {
	State        string `json:"state"`
	URL          string `json:"url,omitempty"`
	ErrorMessage string `json:"error_message"`
	UpdatedAt    string `json:"updated_at"`
}

type KanbanDispatcherState string

const (
	KanbanDispatcherStateRunning  KanbanDispatcherState = "running"
	KanbanDispatcherStateDegraded KanbanDispatcherState = "degraded"
	KanbanDispatcherStateStopped  KanbanDispatcherState = "stopped"
)

type KanbanDispatcherStatus struct {
	State       KanbanDispatcherState `json:"state,omitempty"`
	LastTickAt  string                `json:"last_tick_at,omitempty"`
	LastError   string                `json:"last_error,omitempty"`
	Spawned     int                   `json:"spawned,omitempty"`
	SpawnFailed int                   `json:"spawn_failed,omitempty"`
	AutoBlocked int                   `json:"auto_blocked,omitempty"`
}

type RuntimeResumePendingEvidence struct {
	SessionKey string `json:"session_key,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	Source     string `json:"source,omitempty"`
	ChatID     string `json:"chat_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
	MarkedAt   string `json:"marked_at,omitempty"`
}

type RuntimeDrainTimeoutEvidence struct {
	SessionKey   string `json:"session_key,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	Source       string `json:"source,omitempty"`
	ChatID       string `json:"chat_id,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	Reason       string `json:"reason,omitempty"`
	TimeoutAt    string `json:"timeout_at,omitempty"`
	ActiveAgents int    `json:"active_agents,omitempty"`
}

type RuntimeNonResumableEvidence struct {
	SessionKey string `json:"session_key,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	Source     string `json:"source,omitempty"`
	ChatID     string `json:"chat_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	Reason     string `json:"reason,omitempty"`
	At         string `json:"at,omitempty"`
}

type RuntimeExpiryFinalizedEvidence struct {
	SessionID             string `json:"session_id,omitempty"`
	Source                string `json:"source,omitempty"`
	ChatID                string `json:"chat_id,omitempty"`
	UserID                string `json:"user_id,omitempty"`
	ExpiryFinalized       bool   `json:"expiry_finalized"`
	MigratedMemoryFlushed bool   `json:"migrated_memory_flushed,omitempty"`
}

type RuntimeExpiryFinalizeEvidence struct {
	SessionKey string `json:"session_key,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	Source     string `json:"source,omitempty"`
	ChatID     string `json:"chat_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Attempts   int    `json:"attempts"`
	Error      string `json:"error,omitempty"`
	At         string `json:"at,omitempty"`
}

type RestartTakeoverMarkerStatus string

const (
	RestartTakeoverMarkerStatusWritten RestartTakeoverMarkerStatus = "written"
	RestartTakeoverMarkerStatusSeen    RestartTakeoverMarkerStatus = "seen"
	RestartTakeoverMarkerStatusExpired RestartTakeoverMarkerStatus = "expired"
)

type RuntimeRestartTakeoverEvidence struct {
	Status     RestartTakeoverMarkerStatus `json:"status,omitempty"`
	Source     string                      `json:"source,omitempty"`
	ChatID     string                      `json:"chat_id,omitempty"`
	ThreadID   string                      `json:"thread_id,omitempty"`
	UpdateID   string                      `json:"update_id,omitempty"`
	MessageID  string                      `json:"message_id,omitempty"`
	Generation uint64                      `json:"generation,omitempty"`
	At         string                      `json:"at,omitempty"`
}

type RestartDuplicateStatus string

const (
	RestartDuplicateStatusSuppressed RestartDuplicateStatus = "suppressed"
)

type RuntimeRestartDuplicateEvidence struct {
	Status     RestartDuplicateStatus `json:"status,omitempty"`
	Source     string                 `json:"source,omitempty"`
	ChatID     string                 `json:"chat_id,omitempty"`
	ThreadID   string                 `json:"thread_id,omitempty"`
	UpdateID   string                 `json:"update_id,omitempty"`
	MessageID  string                 `json:"message_id,omitempty"`
	Generation uint64                 `json:"generation,omitempty"`
	At         string                 `json:"at,omitempty"`
}

type RuntimeServiceManagerUnavailableEvidence struct {
	Source   string `json:"source,omitempty"`
	ChatID   string `json:"chat_id,omitempty"`
	ThreadID string `json:"thread_id,omitempty"`
	Reason   string `json:"reason,omitempty"`
	At       string `json:"at,omitempty"`
}

// RuntimeStatusUpdate carries a partial update to the shared runtime status.
type RuntimeStatusUpdate struct {
	GatewayState     GatewayState
	ExitReason       string
	RestartRequested *bool
	ActiveAgents     *int

	Platform      string
	PlatformState PlatformState
	ErrorMessage  string

	ProxyState        string
	ProxyURL          string
	ProxyErrorMessage string

	KanbanDispatcher *KanbanDispatcherStatus

	DrainTimeoutEvidence              *RuntimeDrainTimeoutEvidence
	ResumePendingEvidence             *RuntimeResumePendingEvidence
	NonResumableEvidence              *RuntimeNonResumableEvidence
	ExpiryFinalizedEvidence           *RuntimeExpiryFinalizedEvidence
	ExpiryFinalizeEvidence            *RuntimeExpiryFinalizeEvidence
	TokenLockEvidence                 *TokenLockEvidence
	TakeoverMarkerEvidence            *RuntimeRestartTakeoverEvidence
	DuplicateRestartEvidence          *RuntimeRestartDuplicateEvidence
	ServiceManagerUnavailableEvidence *RuntimeServiceManagerUnavailableEvidence
	ConfigReloadEvidence              *RuntimeConfigReloadEvidence
	MemoryPressureEvidence            *RuntimeMemoryPressureEvidence
}

// RuntimeStatusSnapshot is a read-only view of the runtime status file that
// preserves whether the file was present. RuntimeStatusStore.ReadRuntimeStatus
// synthesizes startup defaults for manager writers; status commands need to
// distinguish that from "no runtime evidence has been written yet".
type RuntimeStatusSnapshot struct {
	Status     RuntimeStatus
	Missing    bool
	Validation RuntimeProcessValidation
}

// RuntimeStatusWriter is the manager-facing seam for lifecycle status writes.
type RuntimeStatusWriter interface {
	UpdateRuntimeStatus(context.Context, RuntimeStatusUpdate) error
}

const (
	plannedStopMarkerKind = "gormes-gateway-planned-stop"

	// PlannedStopMarkerTTL mirrors Hermes' short-lived planned-stop marker
	// window. Stale markers are removed and never mask later unexpected exits.
	PlannedStopMarkerTTL = time.Minute
)

// PlannedStopMarker is written before an operator-initiated gateway stop sends
// SIGTERM/SIGINT to the live gateway process.
type PlannedStopMarker struct {
	Kind            string `json:"kind"`
	TargetPID       int    `json:"target_pid"`
	TargetStartTime int64  `json:"target_start_time"`
	StopperPID      int    `json:"stopper_pid"`
	Generation      uint64 `json:"generation"`
	Reason          string `json:"reason,omitempty"`
	WrittenAt       string `json:"written_at"`
}

type PlannedStopConsumeStatus string

const (
	PlannedStopConsumeMissing    PlannedStopConsumeStatus = "missing"
	PlannedStopConsumeMatched    PlannedStopConsumeStatus = "matched"
	PlannedStopConsumeStale      PlannedStopConsumeStatus = "stale"
	PlannedStopConsumeMismatched PlannedStopConsumeStatus = "mismatched"
	PlannedStopConsumeInvalid    PlannedStopConsumeStatus = "invalid"
)

type PlannedStopConsumeResult struct {
	Status  PlannedStopConsumeStatus
	Matched bool
	Reason  string
	Marker  PlannedStopMarker
}

// PlannedStopStore persists one planned-stop marker as atomic JSON.
type PlannedStopStore struct {
	path      string
	now       func() time.Time
	pid       func() int
	startTime func(int) (int64, bool)
	ttl       time.Duration
}

func NewPlannedStopStore(path string) *PlannedStopStore {
	return &PlannedStopStore{
		path:      path,
		now:       func() time.Time { return time.Now().UTC() },
		pid:       os.Getpid,
		startTime: procProcessStartTime,
		ttl:       PlannedStopMarkerTTL,
	}
}

func DefaultPlannedStopMarkerPath(runtimeStatusPath string) string {
	if runtimeStatusPath == "" {
		return ".gateway-planned-stop.json"
	}
	return filepath.Join(filepath.Dir(runtimeStatusPath), ".gateway-planned-stop.json")
}

func (s *PlannedStopStore) Write(ctx context.Context, marker PlannedStopMarker) error {
	if s == nil || s.path == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if marker.Kind == "" {
		marker.Kind = plannedStopMarkerKind
	}
	if marker.StopperPID == 0 {
		marker.StopperPID = s.currentPID()
	}
	if marker.WrittenAt == "" {
		marker.WrittenAt = s.currentTime().Format(time.RFC3339Nano)
	}
	return writeRestartJSONAtomic(ctx, s.path, marker)
}

func (s *PlannedStopStore) ConsumeForSelf(ctx context.Context) (PlannedStopConsumeResult, error) {
	if s == nil || s.path == "" {
		return PlannedStopConsumeResult{Status: PlannedStopConsumeMissing}, nil
	}
	if err := ctx.Err(); err != nil {
		return PlannedStopConsumeResult{}, err
	}
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return PlannedStopConsumeResult{Status: PlannedStopConsumeMissing}, nil
	}
	if err != nil {
		return PlannedStopConsumeResult{}, fmt.Errorf("read planned stop marker: %w", err)
	}
	if len(raw) == 0 {
		_ = s.Clear(context.Background())
		return PlannedStopConsumeResult{Status: PlannedStopConsumeInvalid, Reason: "empty marker"}, nil
	}

	var marker PlannedStopMarker
	if err := json.Unmarshal(raw, &marker); err != nil {
		_ = s.Clear(context.Background())
		return PlannedStopConsumeResult{Status: PlannedStopConsumeInvalid, Reason: "decode marker: " + err.Error()}, nil
	}
	if marker.Kind != "" && marker.Kind != plannedStopMarkerKind {
		_ = s.Clear(context.Background())
		return PlannedStopConsumeResult{Status: PlannedStopConsumeInvalid, Reason: "marker kind mismatch", Marker: marker}, nil
	}
	if s.plannedStopMarkerStale(marker) {
		_ = s.Clear(context.Background())
		return PlannedStopConsumeResult{Status: PlannedStopConsumeStale, Reason: "marker is stale", Marker: marker}, nil
	}

	result := PlannedStopConsumeResult{Status: PlannedStopConsumeMismatched, Reason: "target pid/start_time mismatch", Marker: marker}
	if s.plannedStopMarkerMatchesSelf(marker) {
		result.Status = PlannedStopConsumeMatched
		result.Matched = true
		result.Reason = ""
	}
	_ = s.Clear(context.Background())
	return result, nil
}

func (s *PlannedStopStore) Clear(ctx context.Context) error {
	if s == nil || s.path == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove planned stop marker: %w", err)
	}
	return nil
}

func (s *PlannedStopStore) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func (s *PlannedStopStore) currentPID() int {
	if s != nil && s.pid != nil {
		return s.pid()
	}
	return os.Getpid()
}

func (s *PlannedStopStore) markerTTL() time.Duration {
	if s != nil && s.ttl > 0 {
		return s.ttl
	}
	return PlannedStopMarkerTTL
}

func (s *PlannedStopStore) plannedStopMarkerStale(marker PlannedStopMarker) bool {
	writtenAt, err := time.Parse(time.RFC3339Nano, marker.WrittenAt)
	if err != nil {
		return true
	}
	return s.currentTime().Sub(writtenAt) > s.markerTTL()
}

func (s *PlannedStopStore) plannedStopMarkerMatchesSelf(marker PlannedStopMarker) bool {
	if marker.TargetPID <= 0 || marker.TargetStartTime == 0 {
		return false
	}
	pid := s.currentPID()
	if marker.TargetPID != pid {
		return false
	}
	startTime := s.startTime
	if startTime == nil {
		startTime = procProcessStartTime
	}
	actualStartTime, ok := startTime(pid)
	return ok && actualStartTime != 0 && actualStartTime == marker.TargetStartTime
}

// RuntimeStatusStore persists the gateway runtime status as atomic JSON.
type RuntimeStatusStore struct {
	path             string
	pidPath          string
	now              func() time.Time
	pid              func() int
	startTime        func(int) (int64, bool)
	argv             func() []string
	bootGitSHA       string
	staleCodeChecker *StaleCodeChecker
	processes        runtimeProcessTable
	mu               sync.Mutex
}

// NewRuntimeStatusStore returns a JSON-backed runtime status store.
func NewRuntimeStatusStore(path string) *RuntimeStatusStore {
	return &RuntimeStatusStore{
		path:             path,
		pidPath:          filepath.Join(filepath.Dir(path), "gateway.pid"),
		now:              func() time.Time { return time.Now().UTC() },
		pid:              os.Getpid,
		startTime:        procProcessStartTime,
		argv:             func() []string { return append([]string(nil), os.Args...) },
		bootGitSHA:       RuntimeBootGitSHA(),
		staleCodeChecker: NewStaleCodeChecker(DefaultStaleCodeSourceRoot()),
		processes:        procRuntimeProcessTable{},
	}
}

// UpdateRuntimeStatus merges update into the persisted read model and writes it
// atomically.
func (s *RuntimeStatusStore) UpdateRuntimeStatus(ctx context.Context, update RuntimeStatusUpdate) error {
	if s == nil || s.path == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	status, err := s.readLocked()
	if err != nil {
		return err
	}
	s.merge(&status, update)
	return s.writeLocked(ctx, status)
}

// ReadRuntimeStatus reads the current runtime status model from disk.
func (s *RuntimeStatusStore) ReadRuntimeStatus(ctx context.Context) (RuntimeStatus, error) {
	if s == nil || s.path == "" {
		return RuntimeStatus{}, nil
	}
	if err := ctx.Err(); err != nil {
		return RuntimeStatus{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readLocked()
}

// ReadRuntimeStatusSnapshot reads the current runtime status model from disk
// without synthesizing a startup status when the file is missing or empty.
func (s *RuntimeStatusStore) ReadRuntimeStatusSnapshot(ctx context.Context) (RuntimeStatusSnapshot, error) {
	if s == nil || s.path == "" {
		return RuntimeStatusSnapshot{Missing: true}, nil
	}
	if err := ctx.Err(); err != nil {
		return RuntimeStatusSnapshot{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return RuntimeStatusSnapshot{Missing: true}, nil
	}
	if err != nil {
		return RuntimeStatusSnapshot{}, fmt.Errorf("read runtime status: %w", err)
	}
	if len(raw) == 0 {
		return RuntimeStatusSnapshot{Missing: true}, nil
	}

	var status RuntimeStatus
	if err := json.Unmarshal(raw, &status); err != nil {
		return RuntimeStatusSnapshot{}, fmt.Errorf("decode runtime status: %w", err)
	}
	if status.Platforms == nil {
		status.Platforms = map[string]PlatformRuntimeStatus{}
	}
	return RuntimeStatusSnapshot{Status: status}, nil
}

// ReadValidatedRuntimeStatusSnapshot reads runtime status and annotates it with
// PID/start-time validation evidence. When validation proves the persisted
// state is stale, the returned status is cleaned in memory so callers do not
// treat old running channels as live.
func (s *RuntimeStatusStore) ReadValidatedRuntimeStatusSnapshot(ctx context.Context) (RuntimeStatusSnapshot, error) {
	snapshot, err := s.ReadRuntimeStatusSnapshot(ctx)
	if err != nil {
		return RuntimeStatusSnapshot{}, err
	}
	if err := ctx.Err(); err != nil {
		return RuntimeStatusSnapshot{}, err
	}

	validation := s.validateRuntimeProcess(snapshot)
	snapshot.Validation = validation
	snapshot.Status = applyRuntimeProcessValidation(snapshot.Status, validation, snapshot.Missing)
	snapshot.Status = s.applyStaleCodeEvidence(snapshot.Status, validation, snapshot.Missing)
	return snapshot, nil
}

func (s *RuntimeStatusStore) applyStaleCodeEvidence(status RuntimeStatus, validation RuntimeProcessValidation, missing bool) RuntimeStatus {
	status.StaleCode = nil
	if missing || !validation.Live || status.BootGitSHA == "" {
		return status
	}
	checker := s.staleCodeChecker
	if sourceRoot := strings.TrimSpace(status.SourceRoot); sourceRoot != "" {
		if checker == nil || strings.TrimSpace(checker.SourceRoot) != sourceRoot {
			checker = NewStaleCodeChecker(sourceRoot)
		}
	}
	if checker == nil {
		checker = NewStaleCodeChecker(DefaultStaleCodeSourceRoot())
	}
	evidence := checker.Check(status.BootGitSHA)
	status.StaleCode = &evidence
	return status
}

func (s *RuntimeStatusStore) validateRuntimeProcess(snapshot RuntimeStatusSnapshot) RuntimeProcessValidation {
	checkedAt := ""
	if s != nil && s.now != nil {
		checkedAt = s.now().Format(time.RFC3339Nano)
	}
	if snapshot.Missing {
		return RuntimeProcessValidation{
			Status:    RuntimeProcessValidationMissingState,
			Live:      false,
			Message:   "runtime status is missing",
			CheckedAt: checkedAt,
		}
	}

	pidRecord, err := readRuntimeStatusRecord(s.pidPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RuntimeProcessValidation{
				Status:    RuntimeProcessValidationMissingPIDFile,
				Live:      false,
				PID:       snapshot.Status.PID,
				Message:   "runtime PID file is missing",
				CheckedAt: checkedAt,
			}
		}
		return RuntimeProcessValidation{
			Status:    RuntimeProcessValidationMissingPIDFile,
			Live:      false,
			PID:       snapshot.Status.PID,
			Message:   err.Error(),
			CheckedAt: checkedAt,
		}
	}
	if mismatch := runtimePIDRecordMismatch(snapshot.Status, pidRecord); mismatch != "" {
		return RuntimeProcessValidation{
			Status:            RuntimeProcessValidationStalePID,
			Live:              false,
			PID:               snapshot.Status.PID,
			ExpectedStartTime: snapshot.Status.StartTime,
			Command:           snapshot.Status.Command,
			Message:           mismatch,
			CheckedAt:         checkedAt,
		}
	}

	pid := pidRecord.PID
	if pid <= 0 {
		pid = snapshot.Status.PID
	}
	expectedStartTime := pidRecord.StartTime
	if expectedStartTime == 0 {
		expectedStartTime = snapshot.Status.StartTime
	}
	command := pidRecord.Command
	if command == "" {
		command = snapshot.Status.Command
	}
	validation := RuntimeProcessValidation{
		PID:               pid,
		ExpectedStartTime: expectedStartTime,
		Command:           command,
		CheckedAt:         checkedAt,
	}
	if pid <= 0 {
		validation.Status = RuntimeProcessValidationStalePID
		validation.Message = "runtime PID is missing or invalid"
		return validation
	}

	processes := s.processes
	if processes == nil {
		processes = procRuntimeProcessTable{}
	}
	process, err := processes.LookupRuntimeProcess(pid)
	if err != nil {
		validation.Live = false
		switch {
		case errors.Is(err, errRuntimeProcessPermissionDenied):
			validation.Status = RuntimeProcessValidationPermissionDenied
			validation.Message = "process lookup was denied"
		case errors.Is(err, errRuntimeProcessNotFound):
			validation.Status = RuntimeProcessValidationStalePID
			validation.Message = "process is not running"
		default:
			validation.Status = RuntimeProcessValidationStalePID
			validation.Message = err.Error()
		}
		return validation
	}

	validation.ActualStartTime = process.StartTime
	if expectedStartTime != 0 && process.StartTime != 0 && process.StartTime != expectedStartTime {
		validation.Status = RuntimeProcessValidationPIDReused
		validation.Message = "process start time does not match runtime status"
		return validation
	}
	if process.Stopped {
		validation.Status = RuntimeProcessValidationStopped
		validation.Message = "process is stopped"
		return validation
	}

	validation.Status = RuntimeProcessValidationLive
	validation.Live = true
	if validation.Command == "" {
		validation.Command = process.Command
	}
	return validation
}

func runtimePIDRecordMismatch(status RuntimeStatus, pidRecord RuntimeStatus) string {
	if status.Kind != "" && pidRecord.Kind != "" && status.Kind != pidRecord.Kind {
		return "runtime PID record kind does not match status"
	}
	if status.PID > 0 && pidRecord.PID > 0 && status.PID != pidRecord.PID {
		return "runtime PID record pid does not match status"
	}
	if status.StartTime > 0 && pidRecord.StartTime > 0 && status.StartTime != pidRecord.StartTime {
		return "runtime PID record start time does not match status"
	}
	if status.Generation > 0 && pidRecord.Generation > 0 && status.Generation != pidRecord.Generation {
		return "runtime PID record generation does not match status"
	}
	if status.Command != "" && pidRecord.Command != "" && status.Command != pidRecord.Command {
		return "runtime PID record command does not match status"
	}
	return ""
}

func (s *RuntimeStatusStore) runtimeSourceRoot() string {
	if s == nil || s.staleCodeChecker == nil {
		return ""
	}
	return strings.TrimSpace(s.staleCodeChecker.SourceRoot)
}

func (s *RuntimeStatusStore) merge(status *RuntimeStatus, update RuntimeStatusUpdate) {
	status.Kind = runtimeStatusKind
	status.PID = s.pid()
	status.BootGitSHA = s.bootGitSHA
	status.SourceRoot = s.runtimeSourceRoot()
	status.StaleCode = nil
	if startTime, ok := s.startTime(status.PID); ok {
		status.StartTime = startTime
	} else {
		status.StartTime = 0
	}
	status.Generation++
	status.Argv = append([]string(nil), s.argv()...)
	status.Command = strings.Join(status.Argv, " ")
	if status.Platforms == nil {
		status.Platforms = map[string]PlatformRuntimeStatus{}
	}
	status.UpdatedAt = s.now().Format(time.RFC3339Nano)

	if update.GatewayState != "" {
		status.GatewayState = update.GatewayState
	}
	if update.ExitReason != "" ||
		update.GatewayState == GatewayStateStarting ||
		update.GatewayState == GatewayStateRunning ||
		update.GatewayState == GatewayStateStopped {
		status.ExitReason = update.ExitReason
	}
	if update.RestartRequested != nil {
		status.RestartRequested = *update.RestartRequested
	}
	if update.ActiveAgents != nil {
		if *update.ActiveAgents < 0 {
			status.ActiveAgents = 0
		} else {
			status.ActiveAgents = *update.ActiveAgents
		}
	}
	if update.ProxyState != "" || update.ProxyURL != "" || update.ProxyErrorMessage != "" {
		if update.ProxyState != "" {
			status.Proxy.State = update.ProxyState
		}
		if update.ProxyURL != "" {
			status.Proxy.URL = update.ProxyURL
		}
		status.Proxy.ErrorMessage = update.ProxyErrorMessage
		status.Proxy.UpdatedAt = status.UpdatedAt
	}
	if update.KanbanDispatcher != nil {
		kanbanStatus := status.KanbanDispatcher
		if update.KanbanDispatcher.State != "" {
			kanbanStatus.State = update.KanbanDispatcher.State
		}
		if update.KanbanDispatcher.LastTickAt != "" {
			kanbanStatus.LastTickAt = update.KanbanDispatcher.LastTickAt
		}
		if update.KanbanDispatcher.LastError != "" || update.KanbanDispatcher.State == KanbanDispatcherStateRunning {
			kanbanStatus.LastError = update.KanbanDispatcher.LastError
		}
		kanbanStatus.Spawned += update.KanbanDispatcher.Spawned
		kanbanStatus.SpawnFailed += update.KanbanDispatcher.SpawnFailed
		kanbanStatus.AutoBlocked += update.KanbanDispatcher.AutoBlocked
		status.KanbanDispatcher = kanbanStatus
	}
	if update.DrainTimeoutEvidence != nil {
		evidence := *update.DrainTimeoutEvidence
		status.DrainTimeouts = append(status.DrainTimeouts, evidence)
	}
	if update.ResumePendingEvidence != nil {
		evidence := *update.ResumePendingEvidence
		status.ResumePending = append(status.ResumePending, evidence)
	}
	if update.NonResumableEvidence != nil {
		evidence := *update.NonResumableEvidence
		status.NonResumable = append(status.NonResumable, evidence)
	}
	if update.ExpiryFinalizedEvidence != nil {
		evidence := *update.ExpiryFinalizedEvidence
		status.ExpiryFinalized = append(status.ExpiryFinalized, evidence)
	}
	if update.ExpiryFinalizeEvidence != nil {
		evidence := *update.ExpiryFinalizeEvidence
		status.ExpiryFinalize = append(status.ExpiryFinalize, evidence)
	}
	if update.TokenLockEvidence != nil {
		evidence := *update.TokenLockEvidence
		status.TokenLocks = append(status.TokenLocks, evidence)
	}
	if update.TakeoverMarkerEvidence != nil {
		evidence := *update.TakeoverMarkerEvidence
		status.TakeoverMarkers = append(status.TakeoverMarkers, evidence)
	}
	if update.DuplicateRestartEvidence != nil {
		evidence := *update.DuplicateRestartEvidence
		status.DuplicateRestarts = append(status.DuplicateRestarts, evidence)
	}
	if update.ServiceManagerUnavailableEvidence != nil {
		evidence := *update.ServiceManagerUnavailableEvidence
		status.ServiceManagerUnavailable = append(status.ServiceManagerUnavailable, evidence)
	}
	if update.ConfigReloadEvidence != nil {
		evidence := *update.ConfigReloadEvidence
		if evidence.Generation == 0 {
			evidence.Generation = status.Generation
		}
		if evidence.Status == RuntimeConfigReloadApplied && evidence.AppliedAt == "" {
			evidence.AppliedAt = status.UpdatedAt
		}
		if evidence.Status == RuntimeConfigReloadFailed && evidence.FailedAt == "" {
			evidence.FailedAt = status.UpdatedAt
		}
		evidence.Redacted = true
		status.ConfigReload = evidence
	}
	if update.MemoryPressureEvidence != nil {
		evidence := *update.MemoryPressureEvidence
		evidence.Redacted = true
		status.MemoryPressure = evidence
	}
	if update.Platform == "" {
		return
	}

	platform := status.Platforms[update.Platform]
	if update.PlatformState != "" {
		platform.State = update.PlatformState
	}
	platform.ErrorMessage = update.ErrorMessage
	platform.UpdatedAt = status.UpdatedAt
	status.Platforms[update.Platform] = platform
}

func applyRuntimeProcessValidation(status RuntimeStatus, validation RuntimeProcessValidation, missing bool) RuntimeStatus {
	status.ProcessValidation = validation
	if missing || validation.Live {
		return status
	}
	status.GatewayState = GatewayStateStopped
	status.ActiveAgents = 0
	for name, platform := range status.Platforms {
		switch platform.State {
		case PlatformStateStarting, PlatformStateRunning:
			platform.State = PlatformStateStopped
			status.Platforms[name] = platform
		}
	}
	switch strings.TrimSpace(strings.ToLower(status.Proxy.State)) {
	case "starting", "running", "draining":
		status.Proxy.State = "stopped"
	}
	if status.KanbanDispatcher.State == KanbanDispatcherStateRunning ||
		status.KanbanDispatcher.State == KanbanDispatcherStateDegraded {
		status.KanbanDispatcher.State = KanbanDispatcherStateStopped
	}
	return status
}

func (s *RuntimeStatusStore) readLocked() (RuntimeStatus, error) {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		pid := s.pid()
		startTime, _ := s.startTime(pid)
		argv := append([]string(nil), s.argv()...)
		return RuntimeStatus{
			Kind:         runtimeStatusKind,
			PID:          pid,
			StartTime:    startTime,
			BootGitSHA:   s.bootGitSHA,
			Command:      strings.Join(argv, " "),
			Argv:         argv,
			GatewayState: GatewayStateStarting,
			Platforms:    map[string]PlatformRuntimeStatus{},
			UpdatedAt:    s.now().Format(time.RFC3339Nano),
		}, nil
	}
	if err != nil {
		return RuntimeStatus{}, fmt.Errorf("read runtime status: %w", err)
	}
	if len(raw) == 0 {
		pid := s.pid()
		startTime, _ := s.startTime(pid)
		argv := append([]string(nil), s.argv()...)
		return RuntimeStatus{
			Kind:       runtimeStatusKind,
			PID:        pid,
			StartTime:  startTime,
			BootGitSHA: s.bootGitSHA,
			Command:    strings.Join(argv, " "),
			Argv:       argv,
			Platforms:  map[string]PlatformRuntimeStatus{},
			UpdatedAt:  s.now().Format(time.RFC3339Nano),
		}, nil
	}

	var status RuntimeStatus
	if err := json.Unmarshal(raw, &status); err != nil {
		return RuntimeStatus{}, fmt.Errorf("decode runtime status: %w", err)
	}
	if status.Platforms == nil {
		status.Platforms = map[string]PlatformRuntimeStatus{}
	}
	return status, nil
}

func (s *RuntimeStatusStore) writeLocked(ctx context.Context, status RuntimeStatus) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := writeRuntimeStatusJSONAtomic(s.path, status); err != nil {
		return err
	}
	if s.pidPath == "" {
		return nil
	}
	pidRecord := RuntimeStatus{
		Kind:       status.Kind,
		PID:        status.PID,
		StartTime:  status.StartTime,
		Generation: status.Generation,
		BootGitSHA: status.BootGitSHA,
		Command:    status.Command,
		Argv:       append([]string(nil), status.Argv...),
		UpdatedAt:  status.UpdatedAt,
	}
	if err := writeRuntimeStatusJSONAtomic(s.pidPath, pidRecord); err != nil {
		return fmt.Errorf("write runtime pid record: %w", err)
	}
	return nil
}

func writeRuntimeStatusJSONAtomic(path string, status RuntimeStatus) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create runtime status dir: %w", err)
	}
	raw, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("encode runtime status: %w", err)
	}
	raw = append(raw, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), ".gateway_state-*.tmp")
	if err != nil {
		return fmt.Errorf("create runtime status temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write runtime status temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close runtime status temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace runtime status: %w", err)
	}
	return nil
}

func readRuntimeStatusRecord(path string) (RuntimeStatus, error) {
	if path == "" {
		return RuntimeStatus{}, os.ErrNotExist
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return RuntimeStatus{}, err
	}
	if len(raw) == 0 {
		return RuntimeStatus{}, os.ErrNotExist
	}
	var status RuntimeStatus
	if err := json.Unmarshal(raw, &status); err != nil {
		return RuntimeStatus{}, fmt.Errorf("decode runtime PID record: %w", err)
	}
	return status, nil
}

var (
	errRuntimeProcessNotFound         = errors.New("runtime process not found")
	errRuntimeProcessPermissionDenied = errors.New("runtime process permission denied")
)

type runtimeProcessTable interface {
	LookupRuntimeProcess(pid int) (runtimeProcessInfo, error)
}

type runtimeProcessInfo struct {
	PID       int
	StartTime int64
	Command   string
	Stopped   bool
}

type procRuntimeProcessTable struct{}

func (procRuntimeProcessTable) LookupRuntimeProcess(pid int) (runtimeProcessInfo, error) {
	if pid <= 0 {
		return runtimeProcessInfo{}, errRuntimeProcessNotFound
	}
	statPath := filepath.Join("/proc", fmt.Sprint(pid), "stat")
	raw, err := os.ReadFile(statPath)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return runtimeProcessInfo{}, errRuntimeProcessPermissionDenied
		}
		if errors.Is(err, os.ErrNotExist) {
			return runtimeProcessInfo{}, errRuntimeProcessNotFound
		}
		return runtimeProcessInfo{}, err
	}
	startTime, state, ok := parseProcStatIdentity(string(raw))
	if !ok {
		return runtimeProcessInfo{}, errRuntimeProcessNotFound
	}
	info := runtimeProcessInfo{
		PID:       pid,
		StartTime: startTime,
		Stopped:   state == "T" || state == "t",
	}
	if cmdline, ok := readProcCmdline(pid); ok {
		info.Command = cmdline
	}
	return info, nil
}

func readProcCmdline(pid int) (string, bool) {
	raw, err := os.ReadFile(filepath.Join("/proc", fmt.Sprint(pid), "cmdline"))
	if err != nil || len(raw) == 0 {
		return "", false
	}
	parts := strings.Split(string(raw), "\x00")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return "", false
	}
	return strings.Join(out, " "), true
}

func procProcessStartTime(pid int) (int64, bool) {
	if pid <= 0 {
		return 0, false
	}
	raw, err := os.ReadFile(filepath.Join("/proc", fmt.Sprint(pid), "stat"))
	if err != nil {
		return 0, false
	}
	return parseProcStatStartTime(string(raw))
}

func parseProcStatStartTime(stat string) (int64, bool) {
	startTime, _, ok := parseProcStatIdentity(stat)
	return startTime, ok
}

func parseProcStatIdentity(stat string) (int64, string, bool) {
	commEnd := strings.LastIndex(stat, ")")
	if commEnd < 0 || commEnd+2 >= len(stat) {
		return 0, "", false
	}
	fields := strings.Fields(stat[commEnd+2:])
	if len(fields) <= 19 {
		return 0, "", false
	}
	var start int64
	if _, err := fmt.Sscan(fields[19], &start); err != nil {
		return 0, "", false
	}
	return start, fields[0], true
}
