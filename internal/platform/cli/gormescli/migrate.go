package gormescli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/migrationruntime"

type MigrateHermesManifest = migrationruntime.MigrateHermesManifest
type MigrateHermesOptions = migrationruntime.MigrateHermesOptions
type MigrateHermesWriteRequest = migrationruntime.MigrateHermesWriteRequest
type MigrateHermesWriteOutcome = migrationruntime.MigrateHermesWriteOutcome

func BuildMigrateHermesManifest(opts MigrateHermesOptions) (*MigrateHermesManifest, error) {
	return migrationruntime.BuildMigrateHermesManifest(opts)
}

func ApplyMigrateHermesManifest(req MigrateHermesWriteRequest) (MigrateHermesWriteOutcome, error) {
	return migrationruntime.ApplyMigrateHermesManifest(req)
}

type MigrateOpenClawManifest = migrationruntime.MigrateOpenClawManifest
type MigrateOpenClawOptions = migrationruntime.MigrateOpenClawOptions
type MigrateOpenClawApplyRequest = migrationruntime.MigrateOpenClawApplyRequest
type MigrateOpenClawApplyOutcome = migrationruntime.MigrateOpenClawApplyOutcome
type MigrateOpenClawCleanupRequest = migrationruntime.MigrateOpenClawCleanupRequest
type MigrateOpenClawCleanupOutcome = migrationruntime.MigrateOpenClawCleanupOutcome

func BuildMigrateOpenClawManifest(opts MigrateOpenClawOptions) (*MigrateOpenClawManifest, error) {
	return migrationruntime.BuildMigrateOpenClawManifest(opts)
}

func ApplyMigrateOpenClawManifest(req MigrateOpenClawApplyRequest) (MigrateOpenClawApplyOutcome, error) {
	return migrationruntime.ApplyMigrateOpenClawManifest(req)
}

func PerformMigrateOpenClawCleanup(req MigrateOpenClawCleanupRequest) (MigrateOpenClawCleanupOutcome, error) {
	return migrationruntime.PerformMigrateOpenClawCleanup(req)
}
