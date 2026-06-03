package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

// newMigrateCommand wires the `gormes migrate` subtree. Current slices
// print deterministic JSON dry-run manifests and never write destination
// files. Writer slices introduce `--yes`, `--overwrite`, and backup output.
func newMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "migrate",
		Short:        "Migrate state from upstream agents into Gormes (dry-run only in this slice)",
		SilenceUsage: true,
		// No Args=NoArgs here: the shared parent-command guard needs
		// the raw args so SuggestionsMinimumDistance=2 can surface
		// `did you mean "openclaw"?` for typos like `migrate ooenclaw`.
		// NoArgs would short-circuit before the guard can include that
		// suggestion, and a command like
		// `migrate ooenclaw --dry-run --source ...` would degrade to a
		// flag/arg validation error instead of OpenClaw typo guidance.
		// Tests:
		//   TestHermesCommandAliasFidelity_RootUnknownAndTypoSuggestions
		//   TestMigrateOpenClawDryRun_RejectsMissingDryRunAndTypo
	}
	cmd.SuggestionsMinimumDistance = 2
	cmd.AddCommand(newMigrateHermesCommand(), newMigrateOpenClawCommand())
	return cmd
}

func newClawCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "claw",
		Short:        "Hermes-compatible OpenClaw migration tools",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
	}
	cmd.AddCommand(newClawMigrateCommand(), newClawCleanupCommand())
	return cmd
}

func newClawMigrateCommand() *cobra.Command {
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
				return newExitCodeError(2, errors.New("gormes claw migrate: --dry-run and --yes are mutually exclusive"))
			}
			if !dryRun && !yes {
				return newExitCodeError(2, errors.New("gormes claw migrate: use --yes to apply or --dry-run to inspect"))
			}
			if dryRun {
				return runMigrateOpenClawDryRun(cmd, source)
			}
			return runMigrateOpenClawApply(cmd, source, "", overwrite, migrateSecrets)
		},
	}
	cmd.Flags().StringVar(&source, "source", "", "explicit OpenClaw home directory; preferred over ~/.openclaw, ~/.clawdbot, and ~/.moltbot")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the migration manifest without writing any Gormes file")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "apply the migration manifest into Gormes")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite existing destination keys flagged as conflict in the manifest")
	cmd.Flags().BoolVar(&migrateSecrets, "migrate-secrets", false, "import secret env values; without this flag, secret rows are reported as secret_skipped")
	return cmd
}

func newClawCleanupCommand() *cobra.Command {
	return newOpenClawCleanupCommand("gormes claw cleanup", []string{"clean"})
}

func newMigrateHermesCommand() *cobra.Command {
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
				return newExitCodeError(2, errors.New("gormes migrate hermes: --dry-run and --yes are mutually exclusive"))
			}
			if !dryRun && !yes {
				return newExitCodeError(2, errors.New("gormes migrate hermes: use --yes to apply or --dry-run to inspect"))
			}
			if dryRun {
				return runMigrateHermesDryRun(cmd, source)
			}
			return runMigrateHermesApply(cmd, source, dest, overwrite)
		},
	}
	cmd.Flags().StringVar(&source, "source", "", "explicit Hermes home directory; preferred over $HERMES_HOME and ~/.hermes")
	cmd.Flags().StringVar(&dest, "dest", "", "explicit Gormes destination config dir; defaults to GormesHome")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the migration manifest without writing any Gormes file")
	cmd.Flags().BoolVar(&yes, "yes", false, "apply the migration manifest into the destination Gormes config dir + dotenv")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite existing destination keys flagged as conflict in the manifest")
	return cmd
}

// migrateHermesDryRunReportJSON wraps the migrate hermes manifest with
// build provenance so fleet automation orchestrating Hermes-to-Gormes
// migration across machines can attribute each manifest to the binary
// version that emitted it. Existing manifest fields stay top-level via
// struct embedding — callers parsing the old shape continue to work
// because Go's JSON decoder ignores the unknown `build` field.
type migrateHermesDryRunReportJSON = gormescli.MigrateHermesDryRunReportJSON

// runMigrateHermesDryRun preserves the existing JSON manifest output for
// `gormes migrate hermes --dry-run` so dry-run callers see the same
// fixture-validated payload after the writer slice lands.
func runMigrateHermesDryRun(cmd *cobra.Command, source string) error {
	return migrateExitError(gormescli.RunMigrateHermesDryRun(cmd.OutOrStdout(), source, migrateBuildProvenance()))
}

// runMigrateHermesApply binds the manifest builder to the writer. The
// destination defaults to GormesHome; tests pass --dest to keep
// writes inside t.TempDir().
// Source discovery is delegated to BuildManifest so apply and dry-run
// use the same explicit --source > $HERMES_HOME > ~/.hermes chain.
func runMigrateHermesApply(cmd *cobra.Command, source, dest string, overwrite bool) error {
	return migrateExitError(gormescli.RunMigrateHermesApply(cmd.OutOrStdout(), source, dest, overwrite, migrateBuildProvenance()))
}

// migrateHermesApplyReportJSON wraps gormescli.MigrateHermesWriteOutcome with
// build provenance so fleet automation orchestrating
// Hermes-to-Gormes migration across machines can attribute each apply
// outcome to the binary version that emitted it.
type migrateHermesApplyReportJSON = gormescli.MigrateHermesApplyReportJSON

// gormesConfigPath returns the destination config.toml path used when
// `--dest` is not set.
func gormesConfigPath() string {
	return gormescli.GormesMigrationConfigPath()
}

func newMigrateOpenClawCommand() *cobra.Command {
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
				return newExitCodeError(2, errors.New("gormes migrate openclaw: --dry-run and --yes are mutually exclusive"))
			}
			if !dryRun && !yes {
				return newExitCodeError(2, errors.New("gormes migrate openclaw: use --yes to apply or --dry-run to inspect"))
			}
			if dryRun {
				return runMigrateOpenClawDryRun(cmd, source)
			}
			return runMigrateOpenClawApply(cmd, source, dest, overwrite, secrets)
		},
	}
	cmd.Flags().StringVar(&source, "source", "", "explicit OpenClaw home directory; preferred over ~/.openclaw, ~/.clawdbot, and ~/.moltbot")
	cmd.Flags().StringVar(&dest, "dest", "", "explicit Gormes destination config dir; defaults to GormesHome")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the migration manifest without writing any Gormes file")
	cmd.Flags().BoolVar(&yes, "yes", false, "apply the migration manifest into the destination Gormes config dir, dotenv, memory dir, and skills dir")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite existing destination keys flagged as conflict in the manifest")
	cmd.Flags().BoolVar(&secrets, "secrets", false, "import secret env values; without --secrets, secret rows are reported as secret_skipped")
	cmd.AddCommand(newMigrateOpenClawCleanupCommand())
	return cmd
}

// migrateOpenClawDryRunReportJSON wraps the migrate openclaw manifest
// with build provenance so fleet automation orchestrating
// OpenClaw-to-Gormes migration across machines can attribute each
// manifest to the binary version that emitted it. Existing manifest
// fields stay top-level via struct embedding.
type migrateOpenClawDryRunReportJSON = gormescli.MigrateOpenClawDryRunReportJSON

func runMigrateOpenClawDryRun(cmd *cobra.Command, source string) error {
	return migrateExitError(gormescli.RunMigrateOpenClawDryRun(cmd.OutOrStdout(), source, migrateBuildProvenance()))
}

func runMigrateOpenClawApply(cmd *cobra.Command, source, dest string, overwrite, secrets bool) error {
	return migrateExitError(gormescli.RunMigrateOpenClawApply(cmd.OutOrStdout(), source, dest, overwrite, secrets, migrateBuildProvenance()))
}

// migrateOpenClawApplyReportJSON wraps gormescli.MigrateOpenClawApplyOutcome
// with build provenance.
type migrateOpenClawApplyReportJSON = gormescli.MigrateOpenClawApplyReportJSON

// migrateOpenClawCleanupReportJSON wraps gormescli.MigrateOpenClawCleanupOutcome
// with build provenance.
type migrateOpenClawCleanupReportJSON = gormescli.MigrateOpenClawCleanupReportJSON

func newMigrateOpenClawCleanupCommand() *cobra.Command {
	return newOpenClawCleanupCommand("gormes migrate openclaw cleanup", nil)
}

func newOpenClawCleanupCommand(commandLabel string, aliases []string) *cobra.Command {
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
				return newExitCodeError(2, fmt.Errorf("%s: --dry-run and --yes are mutually exclusive", commandLabel))
			}
			if !dryRun && !yes {
				return newExitCodeError(2, fmt.Errorf("%s: use --yes to apply or --dry-run to inspect", commandLabel))
			}
			return migrateExitError(gormescli.RunMigrateOpenClawCleanup(cmd.OutOrStdout(), commandLabel, dryRun, migrateBuildProvenance()))
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview the renames without modifying disk")
	cmd.Flags().BoolVar(&yes, "yes", false, "rename ~/.openclaw, ~/.clawdbot, and ~/.moltbot to .pre-migration archives without deleting data")
	return cmd
}

// gormesSkillsDir returns the destination skills directory path used
// when migrating OpenClaw skills.
func gormesSkillsDir() string { return gormescli.GormesMigrationSkillsDir() }

func gormesMemoryDir() string { return gormescli.GormesMigrationMemoryDir() }

func gormesMigrationReportRoot() string { return gormescli.GormesMigrationReportRoot() }

// collectGormesEnvSnapshot returns the GORMES_* env keys currently set
// on the running process, so the manifest can mark Hermes .env keys
// that would overwrite already-set Gormes values as conflict. Only
// names are looked up; raw secret bytes never reach the manifest.
func collectGormesEnvSnapshot() map[string]string { return gormescli.CollectGormesEnvSnapshot() }

func collectMigrationEnvSnapshot() map[string]string { return gormescli.CollectMigrationEnvSnapshot() }

func migrateBuildProvenance() gormescli.BuildProvenance {
	build := newBuildProvenance()
	return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}

func migrateExitError(err error) error {
	if err == nil {
		return nil
	}
	if code := gormescli.MigrateExitCode(err); code != 0 {
		return newExitCodeError(code, err)
	}
	return err
}
