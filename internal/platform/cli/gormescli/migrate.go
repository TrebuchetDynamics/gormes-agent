package gormescli

import (
	"io"

	appmigrate "github.com/TrebuchetDynamics/gormes-agent/internal/app/migrate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/migrationruntime"
)

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

type MigrateBuildProvenance = appmigrate.BuildProvenance

type MigrateHermesDryRunReportJSON = appmigrate.HermesDryRunReportJSON

type MigrateHermesApplyReportJSON = appmigrate.HermesApplyReportJSON

type MigrateOpenClawDryRunReportJSON = appmigrate.OpenClawDryRunReportJSON

type MigrateOpenClawApplyReportJSON = appmigrate.OpenClawApplyReportJSON

type MigrateOpenClawCleanupReportJSON = appmigrate.OpenClawCleanupReportJSON

func RunMigrateHermesDryRun(out io.Writer, source string, build BuildProvenance) error {
	return appmigrate.RunHermesDryRun(out, source, appmigrate.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit})
}

func RunMigrateHermesApply(out io.Writer, source, dest string, overwrite bool, build BuildProvenance) error {
	return appmigrate.RunHermesApply(out, source, dest, overwrite, appmigrate.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit})
}

func RunMigrateOpenClawDryRun(out io.Writer, source string, build BuildProvenance) error {
	return appmigrate.RunOpenClawDryRun(out, source, appmigrate.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit})
}

func RunMigrateOpenClawApply(out io.Writer, source, dest string, overwrite, secrets bool, build BuildProvenance) error {
	return appmigrate.RunOpenClawApply(out, source, dest, overwrite, secrets, appmigrate.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit})
}

func RunMigrateOpenClawCleanup(out io.Writer, commandLabel string, dryRun bool, build BuildProvenance) error {
	return appmigrate.RunOpenClawCleanup(out, commandLabel, dryRun, appmigrate.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit})
}

func MigrateExitCode(err error) int { return appmigrate.ExitCode(err) }

func GormesMigrationConfigPath() string { return appmigrate.GormesConfigPath() }

func GormesMigrationSkillsDir() string { return appmigrate.GormesSkillsDir() }

func GormesMigrationMemoryDir() string { return appmigrate.GormesMemoryDir() }

func GormesMigrationReportRoot() string { return appmigrate.GormesMigrationReportRoot() }

func CollectGormesEnvSnapshot() map[string]string { return appmigrate.CollectGormesEnvSnapshot() }

func CollectMigrationEnvSnapshot() map[string]string { return appmigrate.CollectMigrationEnvSnapshot() }
