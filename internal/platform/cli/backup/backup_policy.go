package backup

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/backup/policy"

type BackupReason = policy.BackupReason

const (
	BackupReasonSkippedDefault        = policy.BackupReasonSkippedDefault
	BackupReasonForced                = policy.BackupReasonForced
	BackupReasonDisabledByFlag        = policy.BackupReasonDisabledByFlag
	BackupReasonConfigEnabled         = policy.BackupReasonConfigEnabled
	BackupReasonManifestExcludedPaths = policy.BackupReasonManifestExcludedPaths
)

type BackupPolicyFlags = policy.BackupPolicyFlags
type BackupDecision = policy.BackupDecision

func ResolveBackupPolicy(flags BackupPolicyFlags) BackupDecision {
	return policy.ResolveBackupPolicy(flags)
}
func IsExcludedFromBackup(relPath string) bool { return policy.IsExcludedFromBackup(relPath) }
func PartitionBackupCandidates(candidates []string) (included, excluded []string) {
	return policy.PartitionBackupCandidates(candidates)
}
