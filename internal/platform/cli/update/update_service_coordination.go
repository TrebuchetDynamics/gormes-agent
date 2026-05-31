package update

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/update/servicecoordination"
)

const (
	UpdateEvidenceUpdateLockAcquired                UpdateEvidenceKind = servicecoordination.UpdateEvidenceUpdateLockAcquired
	UpdateEvidenceUpdateLockBlocked                 UpdateEvidenceKind = servicecoordination.UpdateEvidenceUpdateLockBlocked
	UpdateEvidenceUpdateLockReleased                UpdateEvidenceKind = servicecoordination.UpdateEvidenceUpdateLockReleased
	UpdateEvidenceUpdateLockReleaseFailed           UpdateEvidenceKind = servicecoordination.UpdateEvidenceUpdateLockReleaseFailed
	UpdateEvidenceReleaseServiceDrainCompleted      UpdateEvidenceKind = servicecoordination.UpdateEvidenceReleaseServiceDrainCompleted
	UpdateEvidenceReleaseServiceDrainFailed         UpdateEvidenceKind = servicecoordination.UpdateEvidenceReleaseServiceDrainFailed
	UpdateEvidenceReleaseServiceStopCompleted       UpdateEvidenceKind = servicecoordination.UpdateEvidenceReleaseServiceStopCompleted
	UpdateEvidenceReleaseServiceStopFailed          UpdateEvidenceKind = servicecoordination.UpdateEvidenceReleaseServiceStopFailed
	UpdateEvidenceReleaseServiceRestartCompleted    UpdateEvidenceKind = servicecoordination.UpdateEvidenceReleaseServiceRestartCompleted
	UpdateEvidenceReleaseServiceRestartFailed       UpdateEvidenceKind = servicecoordination.UpdateEvidenceReleaseServiceRestartFailed
	UpdateEvidenceReleaseServiceHealthPassed        UpdateEvidenceKind = servicecoordination.UpdateEvidenceReleaseServiceHealthPassed
	UpdateEvidenceReleaseServiceHealthFailed        UpdateEvidenceKind = servicecoordination.UpdateEvidenceReleaseServiceHealthFailed
	UpdateEvidenceReleaseServiceRestoreCompleted    UpdateEvidenceKind = servicecoordination.UpdateEvidenceReleaseServiceRestoreCompleted
	UpdateEvidenceReleaseServiceRestoreFailed       UpdateEvidenceKind = servicecoordination.UpdateEvidenceReleaseServiceRestoreFailed
	UpdateEvidenceReleaseServiceUnmanagedBlocked    UpdateEvidenceKind = servicecoordination.UpdateEvidenceReleaseServiceUnmanagedBlocked
	UpdateEvidenceReleaseServiceUnmanagedForced     UpdateEvidenceKind = servicecoordination.UpdateEvidenceReleaseServiceUnmanagedForced
	UpdateEvidenceReleaseServiceMutationUnavailable UpdateEvidenceKind = servicecoordination.UpdateEvidenceReleaseServiceMutationUnavailable
)

type UpdateLock = servicecoordination.UpdateLock
type UpdateLockHandle = servicecoordination.UpdateLockHandle
type UpdateManagedService = servicecoordination.UpdateManagedService
type UpdateUnmanagedSession = servicecoordination.UpdateUnmanagedSession
type UpdateServiceCoordinationOptions = servicecoordination.UpdateServiceCoordinationOptions
type FileUpdateLock = servicecoordination.FileUpdateLock

func RunUpdateServiceCoordination(ctx context.Context, opts UpdateServiceCoordinationOptions) (report UpdateReleaseBinaryReport) {
	return servicecoordination.RunUpdateServiceCoordination(ctx, opts)
}

func NewFileUpdateLock(path, owner string) FileUpdateLock {
	return servicecoordination.NewFileUpdateLock(path, owner)
}
