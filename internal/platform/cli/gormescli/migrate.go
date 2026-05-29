package gormescli

import (
	migratehermes "github.com/TrebuchetDynamics/gormes-agent/internal/platform/migrate/hermes"
	openclawmigrate "github.com/TrebuchetDynamics/gormes-agent/internal/platform/migrate/openclaw"
)

type MigrateHermesManifest = migratehermes.Manifest
type MigrateHermesOptions = migratehermes.Options
type MigrateHermesWriteRequest = migratehermes.WriteRequest
type MigrateHermesWriteOutcome = migratehermes.WriteOutcome

func BuildMigrateHermesManifest(opts MigrateHermesOptions) (*MigrateHermesManifest, error) {
	return migratehermes.BuildManifest(opts)
}

func ApplyMigrateHermesManifest(req MigrateHermesWriteRequest) (MigrateHermesWriteOutcome, error) {
	return migratehermes.ApplyManifest(req)
}

type MigrateOpenClawManifest = openclawmigrate.Manifest
type MigrateOpenClawOptions = openclawmigrate.Options
type MigrateOpenClawApplyRequest = openclawmigrate.ApplyRequest
type MigrateOpenClawApplyOutcome = openclawmigrate.ApplyOutcome
type MigrateOpenClawCleanupRequest = openclawmigrate.CleanupRequest
type MigrateOpenClawCleanupOutcome = openclawmigrate.CleanupOutcome

func BuildMigrateOpenClawManifest(opts MigrateOpenClawOptions) (*MigrateOpenClawManifest, error) {
	return openclawmigrate.BuildManifest(opts)
}

func ApplyMigrateOpenClawManifest(req MigrateOpenClawApplyRequest) (MigrateOpenClawApplyOutcome, error) {
	return openclawmigrate.ApplyManifest(req)
}

func PerformMigrateOpenClawCleanup(req MigrateOpenClawCleanupRequest) (MigrateOpenClawCleanupOutcome, error) {
	return openclawmigrate.PerformCleanup(req)
}
