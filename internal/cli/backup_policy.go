package cli

import (
	"path"
	"strings"
)

// BackupReason names a precedence-resolved decision about whether a pre-update
// backup should run, or why a candidate path was excluded from a backup
// manifest. The constants are stable evidence strings consumed by update
// status reporting.
type BackupReason string

const (
	// BackupReasonSkippedDefault: neither --backup nor --no-backup nor a
	// config opt-in was supplied, so the default-off policy applies.
	BackupReasonSkippedDefault BackupReason = "backup_skipped_default"
	// BackupReasonForced: --backup was set on the CLI for this run.
	BackupReasonForced BackupReason = "backup_forced"
	// BackupReasonDisabledByFlag: --no-backup was set and overrides --backup
	// and config-level opt-in.
	BackupReasonDisabledByFlag BackupReason = "backup_disabled_by_flag"
	// BackupReasonConfigEnabled: configuration opted in (updates.pre_update_backup)
	// and no CLI flag overrode it.
	BackupReasonConfigEnabled BackupReason = "backup_config_enabled"
	// BackupReasonManifestExcludedPaths: at least one candidate path matched
	// the default exclusion rules (checkpoints/, *.db-wal/-shm/-journal).
	BackupReasonManifestExcludedPaths BackupReason = "backup_manifest_excluded_paths"
)

// BackupPolicyFlags captures the inputs used to resolve backup policy. It is
// populated from CLI flags and config; callers must not mutate it after
// passing it in.
type BackupPolicyFlags struct {
	// Backup mirrors the --backup CLI flag.
	Backup bool
	// NoBackup mirrors the --no-backup CLI flag and wins over Backup.
	NoBackup bool
	// ConfigEnabled mirrors updates.pre_update_backup from config; default false.
	ConfigEnabled bool
	// Candidates is an optional list of relative paths considered for the
	// backup manifest. When non-empty, ResolveBackupPolicy reports excluded
	// paths in the resulting decision.
	Candidates []string
}

// BackupDecision is the typed result of resolving backup policy. It is
// purely informational: it never performs filesystem or network work.
type BackupDecision struct {
	// Requested is true when a pre-update backup should be attempted.
	Requested bool
	// Reason is the primary precedence-rule that decided Requested.
	Reason BackupReason
	// SecondaryReasons carry additional evidence (for example, that some
	// candidate paths were excluded from the manifest).
	SecondaryReasons []BackupReason
	// ExcludedPaths lists candidate paths that the default exclusion rules
	// strip from a backup manifest.
	ExcludedPaths []string
	// IncludedPaths lists candidate paths that survive the exclusion rules.
	IncludedPaths []string
}

// ResolveBackupPolicy applies the precedence rules:
//  1. --no-backup wins over everything (BackupReasonDisabledByFlag).
//  2. --backup forces a single-run backup (BackupReasonForced).
//  3. ConfigEnabled opts in (BackupReasonConfigEnabled).
//  4. Otherwise the default-off policy applies (BackupReasonSkippedDefault).
//
// When Candidates is non-empty, the manifest exclusion rules also run and
// any excluded paths surface as ExcludedPaths plus a secondary reason.
func ResolveBackupPolicy(flags BackupPolicyFlags) BackupDecision {
	decision := BackupDecision{}

	switch {
	case flags.NoBackup:
		decision.Requested = false
		decision.Reason = BackupReasonDisabledByFlag
	case flags.Backup:
		decision.Requested = true
		decision.Reason = BackupReasonForced
	case flags.ConfigEnabled:
		decision.Requested = true
		decision.Reason = BackupReasonConfigEnabled
	default:
		decision.Requested = false
		decision.Reason = BackupReasonSkippedDefault
	}

	if len(flags.Candidates) > 0 {
		included, excluded := PartitionBackupCandidates(flags.Candidates)
		decision.IncludedPaths = included
		decision.ExcludedPaths = excluded
		if len(excluded) > 0 {
			decision.SecondaryReasons = append(decision.SecondaryReasons, BackupReasonManifestExcludedPaths)
		}
	}

	return decision
}

// excludedDirComponents are path components that exclude the entire subtree
// from the backup manifest.
//
// `backups` is excluded because the writer puts new pre-update zips under
// `<gormes_home>/backups/`; without this skip, every subsequent backup
// would include all prior backups, growing geometrically.
var excludedDirComponents = map[string]struct{}{
	"checkpoints": {},
	"backups":     {},
}

// excludedSuffixes are filename suffixes for transient SQLite sidecars that
// must not ride alongside the database snapshot.
var excludedSuffixes = []string{
	".db-wal",
	".db-shm",
	".db-journal",
}

// IsExcludedFromBackup returns true when relPath matches the default
// manifest exclusion rules: any path under checkpoints/ or any SQLite
// sidecar file (*.db-wal, *.db-shm, *.db-journal).
func IsExcludedFromBackup(relPath string) bool {
	clean := path.Clean(strings.TrimPrefix(relPath, "/"))
	if clean == "." || clean == "" {
		return false
	}
	for _, part := range strings.Split(clean, "/") {
		if _, ok := excludedDirComponents[part]; ok {
			return true
		}
	}
	name := path.Base(clean)
	for _, suffix := range excludedSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// PartitionBackupCandidates splits candidates into (included, excluded)
// while preserving input order.
func PartitionBackupCandidates(candidates []string) (included, excluded []string) {
	for _, p := range candidates {
		if IsExcludedFromBackup(p) {
			excluded = append(excluded, p)
		} else {
			included = append(included, p)
		}
	}
	return included, excluded
}
