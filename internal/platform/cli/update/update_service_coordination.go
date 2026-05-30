package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	UpdateEvidenceUpdateLockAcquired                UpdateEvidenceKind = "update_lock_acquired"
	UpdateEvidenceUpdateLockBlocked                 UpdateEvidenceKind = "update_lock_blocked"
	UpdateEvidenceUpdateLockReleased                UpdateEvidenceKind = "update_lock_released"
	UpdateEvidenceUpdateLockReleaseFailed           UpdateEvidenceKind = "update_lock_release_failed"
	UpdateEvidenceReleaseServiceDrainCompleted      UpdateEvidenceKind = "update_release_service_drain_completed"
	UpdateEvidenceReleaseServiceDrainFailed         UpdateEvidenceKind = "update_release_service_drain_failed"
	UpdateEvidenceReleaseServiceStopCompleted       UpdateEvidenceKind = "update_release_service_stop_completed"
	UpdateEvidenceReleaseServiceStopFailed          UpdateEvidenceKind = "update_release_service_stop_failed"
	UpdateEvidenceReleaseServiceRestartCompleted    UpdateEvidenceKind = "update_release_service_restart_completed"
	UpdateEvidenceReleaseServiceRestartFailed       UpdateEvidenceKind = "update_release_service_restart_failed"
	UpdateEvidenceReleaseServiceHealthPassed        UpdateEvidenceKind = "update_release_service_health_passed"
	UpdateEvidenceReleaseServiceHealthFailed        UpdateEvidenceKind = "update_release_service_health_failed"
	UpdateEvidenceReleaseServiceRestoreCompleted    UpdateEvidenceKind = "update_release_service_restore_completed"
	UpdateEvidenceReleaseServiceRestoreFailed       UpdateEvidenceKind = "update_release_service_restore_failed"
	UpdateEvidenceReleaseServiceUnmanagedBlocked    UpdateEvidenceKind = "update_release_service_unmanaged_blocked"
	UpdateEvidenceReleaseServiceUnmanagedForced     UpdateEvidenceKind = "update_release_service_unmanaged_forced"
	UpdateEvidenceReleaseServiceMutationUnavailable UpdateEvidenceKind = "update_release_service_mutation_unavailable"
)

type UpdateLock interface {
	AcquireUpdateLock(context.Context) (UpdateLockHandle, error)
}

type UpdateLockHandle interface {
	Release() error
}

type UpdateManagedService interface {
	UpdateServiceName() string
	UpdateServiceRunning(context.Context) (bool, error)
	DrainUpdateService(context.Context, time.Duration) error
	StopUpdateService(context.Context) error
	StartUpdateService(context.Context) error
	HealthCheckUpdateService(context.Context, time.Duration) error
}

type UpdateUnmanagedSession struct {
	PID     int
	Command string
	Detail  string
}

type UpdateServiceCoordinationOptions struct {
	Lock              UpdateLock
	Services          []UpdateManagedService
	UnmanagedSessions []UpdateUnmanagedSession
	Force             bool
	DrainTimeout      time.Duration
	HealthTimeout     time.Duration
	Mutation          func(context.Context) UpdateReleaseBinaryReport
}

type updateManagedServiceState struct {
	Service    UpdateManagedService
	Name       string
	WasRunning bool
}

func RunUpdateServiceCoordination(ctx context.Context, opts UpdateServiceCoordinationOptions) (report UpdateReleaseBinaryReport) {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.Lock != nil {
		handle, err := opts.Lock.AcquireUpdateLock(ctx)
		if err != nil {
			report.Failed = true
			report.add(UpdateEvidenceUpdateLockBlocked, err.Error())
			return report
		}
		report.add(UpdateEvidenceUpdateLockAcquired, "release update mutation lock acquired")
		defer func() {
			if err := handle.Release(); err != nil {
				report.add(UpdateEvidenceUpdateLockReleaseFailed, err.Error())
				if report.OperatorRecovery == "" {
					report.OperatorRecovery = "remove stale update lock after verifying no update is running: " + err.Error()
				}
				return
			}
			report.add(UpdateEvidenceUpdateLockReleased, "release update mutation lock released")
		}()
	}
	if err := ctx.Err(); err != nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseServiceMutationUnavailable, err.Error())
		return report
	}
	if len(opts.UnmanagedSessions) > 0 {
		detail := formatUpdateUnmanagedSessions(opts.UnmanagedSessions)
		if !opts.Force {
			report.Failed = true
			report.add(UpdateEvidenceReleaseServiceUnmanagedBlocked, detail)
			return report
		}
		report.add(UpdateEvidenceReleaseServiceUnmanagedForced, detail)
	}

	states, ok := stopManagedUpdateServices(ctx, opts, &report)
	if !ok {
		restoreManagedUpdateServices(ctx, opts, states, &report)
		return report
	}
	if opts.Mutation == nil {
		report.Failed = true
		report.add(UpdateEvidenceReleaseServiceMutationUnavailable, "release update mutation was not configured")
		restoreManagedUpdateServices(ctx, opts, states, &report)
		return report
	}
	mutation := opts.Mutation(ctx)
	mergeUpdateReleaseBinaryReport(&report, mutation)
	if report.Failed {
		restoreManagedUpdateServices(ctx, opts, states, &report)
		return report
	}
	restartManagedUpdateServices(ctx, opts, states, &report)
	return report
}

func stopManagedUpdateServices(ctx context.Context, opts UpdateServiceCoordinationOptions, report *UpdateReleaseBinaryReport) ([]updateManagedServiceState, bool) {
	states := make([]updateManagedServiceState, 0, len(opts.Services))
	for _, service := range opts.Services {
		if service == nil {
			continue
		}
		name := updateManagedServiceName(service)
		state := updateManagedServiceState{Service: service, Name: name}
		running, err := service.UpdateServiceRunning(ctx)
		if err != nil {
			report.Failed = true
			report.add(UpdateEvidenceReleaseServiceStopFailed, fmt.Sprintf("%s status: %v", name, err))
			return states, false
		}
		state.WasRunning = running
		states = append(states, state)
		if !running {
			continue
		}
		if err := service.DrainUpdateService(ctx, opts.DrainTimeout); err != nil {
			report.Failed = true
			report.add(UpdateEvidenceReleaseServiceDrainFailed, fmt.Sprintf("%s: %v", name, err))
			return states, false
		}
		report.add(UpdateEvidenceReleaseServiceDrainCompleted, name)
		if err := service.StopUpdateService(ctx); err != nil {
			report.Failed = true
			report.add(UpdateEvidenceReleaseServiceStopFailed, fmt.Sprintf("%s: %v", name, err))
			return states, false
		}
		report.add(UpdateEvidenceReleaseServiceStopCompleted, name)
	}
	return states, true
}

func restartManagedUpdateServices(ctx context.Context, opts UpdateServiceCoordinationOptions, states []updateManagedServiceState, report *UpdateReleaseBinaryReport) {
	for _, state := range states {
		if !state.WasRunning || state.Service == nil {
			continue
		}
		if err := state.Service.StartUpdateService(ctx); err != nil {
			report.add(UpdateEvidenceReleaseServiceRestartFailed, fmt.Sprintf("%s: %v", state.Name, err))
			appendUpdateServiceRecovery(report, state.Name)
			continue
		}
		report.add(UpdateEvidenceReleaseServiceRestartCompleted, state.Name)
		if err := state.Service.HealthCheckUpdateService(ctx, opts.HealthTimeout); err != nil {
			report.add(UpdateEvidenceReleaseServiceHealthFailed, fmt.Sprintf("%s: %v", state.Name, err))
			appendUpdateServiceRecovery(report, state.Name)
			continue
		}
		report.add(UpdateEvidenceReleaseServiceHealthPassed, state.Name)
	}
}

func restoreManagedUpdateServices(ctx context.Context, opts UpdateServiceCoordinationOptions, states []updateManagedServiceState, report *UpdateReleaseBinaryReport) {
	for _, state := range states {
		if !state.WasRunning || state.Service == nil {
			continue
		}
		if err := state.Service.StartUpdateService(ctx); err != nil {
			report.add(UpdateEvidenceReleaseServiceRestoreFailed, fmt.Sprintf("%s: %v", state.Name, err))
			continue
		}
		if err := state.Service.HealthCheckUpdateService(ctx, opts.HealthTimeout); err != nil {
			report.add(UpdateEvidenceReleaseServiceRestoreFailed, fmt.Sprintf("%s health: %v", state.Name, err))
			continue
		}
		report.add(UpdateEvidenceReleaseServiceRestoreCompleted, state.Name)
	}
}

func updateManagedServiceName(service UpdateManagedService) string {
	name := strings.TrimSpace(service.UpdateServiceName())
	if name == "" {
		return "managed-service"
	}
	return name
}

func appendUpdateServiceRecovery(report *UpdateReleaseBinaryReport, service string) {
	if report.OperatorRecovery != "" {
		return
	}
	report.OperatorRecovery = "release update completed, but service " + service + " needs manual restart or health inspection"
}

func mergeUpdateReleaseBinaryReport(dst *UpdateReleaseBinaryReport, src UpdateReleaseBinaryReport) {
	dst.Failed = dst.Failed || src.Failed
	if dst.SnapshotID == "" {
		dst.SnapshotID = src.SnapshotID
	}
	if dst.SnapshotPath == "" {
		dst.SnapshotPath = src.SnapshotPath
	}
	if dst.PreviousVersion == "" {
		dst.PreviousVersion = src.PreviousVersion
	}
	if dst.NewVersion == "" {
		dst.NewVersion = src.NewVersion
	}
	if dst.ManagedBinPath == "" {
		dst.ManagedBinPath = src.ManagedBinPath
	}
	if dst.PublishedBinPath == "" {
		dst.PublishedBinPath = src.PublishedBinPath
	}
	dst.Evidence = append(dst.Evidence, src.Evidence...)
	if dst.OperatorRecovery == "" {
		dst.OperatorRecovery = src.OperatorRecovery
	}
}

func formatUpdateUnmanagedSessions(sessions []UpdateUnmanagedSession) string {
	parts := make([]string, 0, len(sessions))
	for _, session := range sessions {
		detail := strings.TrimSpace(session.Detail)
		if detail == "" {
			detail = strings.TrimSpace(session.Command)
		}
		switch {
		case session.PID > 0 && detail != "":
			parts = append(parts, fmt.Sprintf("pid=%d %s", session.PID, detail))
		case session.PID > 0:
			parts = append(parts, fmt.Sprintf("pid=%d", session.PID))
		case detail != "":
			parts = append(parts, detail)
		}
	}
	if len(parts) == 0 {
		return "unmanaged active session detected"
	}
	return strings.Join(parts, "; ")
}

type FileUpdateLock struct {
	Path  string
	Owner string
	Now   func() time.Time
}

func NewFileUpdateLock(path, owner string) FileUpdateLock {
	return FileUpdateLock{Path: path, Owner: owner}
}

func (l FileUpdateLock) AcquireUpdateLock(ctx context.Context) (UpdateLockHandle, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path := strings.TrimSpace(l.Path)
	if path == "" {
		return nil, fmt.Errorf("missing update lock path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create update lock dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			owner := readUpdateLockOwner(path)
			if owner == "" {
				owner = "unknown owner"
			}
			return nil, fmt.Errorf("update already running; lock held by %s", owner)
		}
		return nil, fmt.Errorf("acquire update lock: %w", err)
	}
	owner := strings.TrimSpace(l.Owner)
	if owner == "" {
		owner = fmt.Sprintf("pid=%d", os.Getpid())
	}
	now := time.Now
	if l.Now != nil {
		now = l.Now
	}
	_, writeErr := fmt.Fprintf(file, "%s\n%s\n", owner, now().UTC().Format(time.RFC3339Nano))
	closeErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("write update lock: %w", writeErr)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("close update lock: %w", closeErr)
	}
	return fileUpdateLockHandle{path: path}, nil
}

type fileUpdateLockHandle struct {
	path string
}

func (h fileUpdateLockHandle) Release() error {
	if strings.TrimSpace(h.path) == "" {
		return nil
	}
	if err := os.Remove(h.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("release update lock: %w", err)
	}
	return nil
}

func readUpdateLockOwner(path string) string {
	body, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(string(body), "\n")
	return strings.TrimSpace(line)
}
