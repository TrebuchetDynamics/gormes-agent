package gormescli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

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

type MigrateCommandOptions struct {
	BuildProvenance func() BuildProvenance
	ExitCodeError   func(code int, err error) error
}

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

func NewMigrateCommand(opts MigrateCommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "migrate",
		Short:        "Migrate state from upstream agents into Gormes (dry-run only in this slice)",
		SilenceUsage: true,
	}
	cmd.SuggestionsMinimumDistance = 2
	cmd.AddCommand(newMigrateHermesCommand(opts), newMigrateOpenClawCommand(opts))
	return cmd
}

func NewClawCommand(opts MigrateCommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "claw",
		Short:        "Hermes-compatible OpenClaw migration tools",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
	}
	cmd.AddCommand(newClawMigrateCommand(opts), newClawCleanupCommand(opts))
	return cmd
}

func newClawMigrateCommand(opts MigrateCommandOptions) *cobra.Command {
	var (
		source         string
		dryRun         bool
		yes            bool
		overwrite      bool
		migrateSecrets bool
	)
	cmd := &cobra.Command{
		Use:          "migrate",
		Short:        "Migrate OpenClaw state into Gormes using the Hermes claw migrate spelling",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dryRun && yes {
				return migrateExitCodeError(opts, 2, errors.New("gormes claw migrate: --dry-run and --yes are mutually exclusive"))
			}
			if !dryRun && !yes {
				return migrateExitCodeError(opts, 2, errors.New("gormes claw migrate: use --yes to apply or --dry-run to inspect"))
			}
			if dryRun {
				return runMigrateOpenClawDryRunCommand(cmd, source, opts)
			}
			return runMigrateOpenClawApplyCommand(cmd, source, "", overwrite, migrateSecrets, opts)
		},
	}
	cmd.Flags().StringVar(&source, "source", "", "explicit OpenClaw home directory; preferred over ~/.openclaw, ~/.clawdbot, and ~/.moltbot")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the migration manifest without writing any Gormes file")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "apply the migration manifest into Gormes")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite existing destination keys flagged as conflict in the manifest")
	cmd.Flags().BoolVar(&migrateSecrets, "migrate-secrets", false, "import secret env values; without this flag, secret rows are reported as secret_skipped")
	return cmd
}

func newClawCleanupCommand(opts MigrateCommandOptions) *cobra.Command {
	return newOpenClawCleanupCommand("gormes claw cleanup", []string{"clean"}, opts)
}

func newMigrateHermesCommand(opts MigrateCommandOptions) *cobra.Command {
	var (
		source    string
		dest      string
		dryRun    bool
		yes       bool
		overwrite bool
	)
	cmd := &cobra.Command{
		Use:          "hermes",
		Short:        "Migrate Hermes config.yaml + .env into Gormes (dry-run manifest or --yes apply)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dryRun && yes {
				return migrateExitCodeError(opts, 2, errors.New("gormes migrate hermes: --dry-run and --yes are mutually exclusive"))
			}
			if !dryRun && !yes {
				return migrateExitCodeError(opts, 2, errors.New("gormes migrate hermes: use --yes to apply or --dry-run to inspect"))
			}
			if dryRun {
				return runMigrateHermesDryRunCommand(cmd, source, opts)
			}
			return runMigrateHermesApplyCommand(cmd, source, dest, overwrite, opts)
		},
	}
	cmd.Flags().StringVar(&source, "source", "", "explicit Hermes home directory; preferred over $HERMES_HOME and ~/.hermes")
	cmd.Flags().StringVar(&dest, "dest", "", "explicit Gormes destination config dir; defaults to GormesHome")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the migration manifest without writing any Gormes file")
	cmd.Flags().BoolVar(&yes, "yes", false, "apply the migration manifest into the destination Gormes config dir + dotenv")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite existing destination keys flagged as conflict in the manifest")
	return cmd
}

func runMigrateHermesDryRunCommand(cmd *cobra.Command, source string, opts MigrateCommandOptions) error {
	return migrateExitError(opts, RunMigrateHermesDryRun(cmd.OutOrStdout(), source, migrateBuildProvenance(opts)))
}

func runMigrateHermesApplyCommand(cmd *cobra.Command, source, dest string, overwrite bool, opts MigrateCommandOptions) error {
	return migrateExitError(opts, RunMigrateHermesApply(cmd.OutOrStdout(), source, dest, overwrite, migrateBuildProvenance(opts)))
}

func newMigrateOpenClawCommand(opts MigrateCommandOptions) *cobra.Command {
	var (
		source    string
		dest      string
		dryRun    bool
		yes       bool
		overwrite bool
		secrets   bool
	)
	cmd := &cobra.Command{
		Use:          "openclaw",
		Short:        "Migrate OpenClaw config, env, memory, user, and skill surfaces into Gormes (dry-run manifest or --yes apply)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dryRun && yes {
				return migrateExitCodeError(opts, 2, errors.New("gormes migrate openclaw: --dry-run and --yes are mutually exclusive"))
			}
			if !dryRun && !yes {
				return migrateExitCodeError(opts, 2, errors.New("gormes migrate openclaw: use --yes to apply or --dry-run to inspect"))
			}
			if dryRun {
				return runMigrateOpenClawDryRunCommand(cmd, source, opts)
			}
			return runMigrateOpenClawApplyCommand(cmd, source, dest, overwrite, secrets, opts)
		},
	}
	cmd.Flags().StringVar(&source, "source", "", "explicit OpenClaw home directory; preferred over ~/.openclaw, ~/.clawdbot, and ~/.moltbot")
	cmd.Flags().StringVar(&dest, "dest", "", "explicit Gormes destination config dir; defaults to GormesHome")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the migration manifest without writing any Gormes file")
	cmd.Flags().BoolVar(&yes, "yes", false, "apply the migration manifest into the destination Gormes config dir, dotenv, memory dir, and skills dir")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite existing destination keys flagged as conflict in the manifest")
	cmd.Flags().BoolVar(&secrets, "secrets", false, "import secret env values; without --secrets, secret rows are reported as secret_skipped")
	cmd.AddCommand(newMigrateOpenClawCleanupCommand(opts))
	return cmd
}

func runMigrateOpenClawDryRunCommand(cmd *cobra.Command, source string, opts MigrateCommandOptions) error {
	return migrateExitError(opts, RunMigrateOpenClawDryRun(cmd.OutOrStdout(), source, migrateBuildProvenance(opts)))
}

func runMigrateOpenClawApplyCommand(cmd *cobra.Command, source, dest string, overwrite, secrets bool, opts MigrateCommandOptions) error {
	return migrateExitError(opts, RunMigrateOpenClawApply(cmd.OutOrStdout(), source, dest, overwrite, secrets, migrateBuildProvenance(opts)))
}

func newMigrateOpenClawCleanupCommand(opts MigrateCommandOptions) *cobra.Command {
	return newOpenClawCleanupCommand("gormes migrate openclaw cleanup", nil, opts)
}

func newOpenClawCleanupCommand(commandLabel string, aliases []string, opts MigrateCommandOptions) *cobra.Command {
	var (
		dryRun bool
		yes    bool
	)
	cmd := &cobra.Command{
		Use:          "cleanup",
		Aliases:      aliases,
		Short:        "Archive leftover OpenClaw directories under HOME by renaming them to .pre-migration variants",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dryRun && yes {
				return migrateExitCodeError(opts, 2, fmt.Errorf("%s: --dry-run and --yes are mutually exclusive", commandLabel))
			}
			if !dryRun && !yes {
				return migrateExitCodeError(opts, 2, fmt.Errorf("%s: use --yes to apply or --dry-run to inspect", commandLabel))
			}
			return migrateExitError(opts, RunMigrateOpenClawCleanup(cmd.OutOrStdout(), commandLabel, dryRun, migrateBuildProvenance(opts)))
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview the renames without modifying disk")
	cmd.Flags().BoolVar(&yes, "yes", false, "rename ~/.openclaw, ~/.clawdbot, and ~/.moltbot to .pre-migration archives without deleting data")
	return cmd
}

func migrateBuildProvenance(opts MigrateCommandOptions) BuildProvenance {
	if opts.BuildProvenance == nil {
		return BuildProvenance{}
	}
	return opts.BuildProvenance()
}

func migrateExitError(opts MigrateCommandOptions, err error) error {
	if err == nil {
		return nil
	}
	if code := MigrateExitCode(err); code != 0 {
		return migrateExitCodeError(opts, code, err)
	}
	return err
}

func migrateExitCodeError(opts MigrateCommandOptions, code int, err error) error {
	if opts.ExitCodeError != nil {
		return opts.ExitCodeError(code, err)
	}
	return err
}
