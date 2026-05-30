package update

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/backup"

type BackupReason = backup.BackupReason

const (
	BackupReasonSkippedDefault        = backup.BackupReasonSkippedDefault
	BackupReasonForced                = backup.BackupReasonForced
	BackupReasonDisabledByFlag        = backup.BackupReasonDisabledByFlag
	BackupReasonConfigEnabled         = backup.BackupReasonConfigEnabled
	BackupReasonManifestExcludedPaths = backup.BackupReasonManifestExcludedPaths
)

type BackupPolicyFlags = backup.BackupPolicyFlags
type BackupDecision = backup.BackupDecision

func ResolveBackupPolicy(flags BackupPolicyFlags) BackupDecision {
	return backup.ResolveBackupPolicy(flags)
}
