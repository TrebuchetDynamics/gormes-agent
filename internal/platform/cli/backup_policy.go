package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/backup"

// BackupReason names a precedence-resolved decision about whether a pre-update
// backup should run, or why a candidate path was excluded from a backup
// manifest. The constants are stable evidence strings consumed by update
// status reporting.
type BackupReason = backup.BackupReason

const (
	// BackupReasonSkippedDefault: neither --backup nor --no-backup nor a
	// config opt-in was supplied, so the default-off policy applies.
	BackupReasonSkippedDefault = backup.BackupReasonSkippedDefault
	// BackupReasonForced: --backup was set on the CLI for this run.
	BackupReasonForced = backup.BackupReasonForced
	// BackupReasonDisabledByFlag: --no-backup was set and overrides --backup
	// and config-level opt-in.
	BackupReasonDisabledByFlag = backup.BackupReasonDisabledByFlag
	// BackupReasonConfigEnabled: configuration opted in (updates.pre_update_backup)
	// and no CLI flag overrode it.
	BackupReasonConfigEnabled = backup.BackupReasonConfigEnabled
	// BackupReasonManifestExcludedPaths: at least one candidate path matched
	// the default exclusion rules (checkpoints/, *.db-wal/-shm/-journal).
	BackupReasonManifestExcludedPaths = backup.BackupReasonManifestExcludedPaths
)

// BackupPolicyFlags captures the inputs used to resolve backup policy. It is
// populated from CLI flags and config; callers must not mutate it after
// passing it in.
type BackupPolicyFlags = backup.BackupPolicyFlags

// BackupDecision is the typed result of resolving backup policy. It is
// purely informational: it never performs filesystem or network work.
type BackupDecision = backup.BackupDecision

// ResolveBackupPolicy applies the precedence rules:
//  1. --no-backup wins over everything (BackupReasonDisabledByFlag).
//  2. --backup forces a single-run backup (BackupReasonForced).
//  3. ConfigEnabled opts in (BackupReasonConfigEnabled).
//  4. Otherwise the default-off policy applies (BackupReasonSkippedDefault).
//
// When Candidates is non-empty, the manifest exclusion rules also run and
// any excluded paths surface as ExcludedPaths plus a secondary reason.
func ResolveBackupPolicy(flags BackupPolicyFlags) BackupDecision {
	return backup.ResolveBackupPolicy(flags)
}

// IsExcludedFromBackup returns true when relPath matches the default
// manifest exclusion rules: any path under checkpoints/ or any SQLite
// sidecar file (*.db-wal, *.db-shm, *.db-journal).
func IsExcludedFromBackup(relPath string) bool { return backup.IsExcludedFromBackup(relPath) }

// PartitionBackupCandidates splits candidates into (included, excluded)
// while preserving input order.
func PartitionBackupCandidates(candidates []string) (included, excluded []string) {
	return backup.PartitionBackupCandidates(candidates)
}
