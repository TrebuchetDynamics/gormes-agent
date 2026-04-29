package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	migratehermes "github.com/TrebuchetDynamics/gormes-agent/internal/migrate/hermes"
	openclawmigrate "github.com/TrebuchetDynamics/gormes-agent/internal/migrate/openclaw"
)

// newMigrateCommand wires the `gormes migrate` subtree. Current slices
// print deterministic JSON dry-run manifests and never write destination
// files. Writer slices introduce `--yes`, `--overwrite`, and backup output.
func newMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "migrate",
		Short:        "Migrate state from upstream agents into Gormes (dry-run only in this slice)",
		SilenceUsage: true,
	}
	cmd.SuggestionsMinimumDistance = 2
	cmd.AddCommand(newMigrateHermesCommand(), newMigrateOpenClawCommand())
	return cmd
}

func newMigrateHermesCommand() *cobra.Command {
	var (
		source string
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:          "hermes",
		Short:        "Build a dry-run manifest for migrating Hermes config.yaml + .env into Gormes",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !dryRun {
				return newExitCodeError(2, errors.New("gormes migrate hermes: --dry-run is required in this slice; the writer subcommand is not implemented yet"))
			}
			m, err := migratehermes.BuildManifest(migratehermes.Options{
				Source:            strings.TrimSpace(source),
				ExistingGormesEnv: collectGormesEnvSnapshot(),
			})
			if err != nil {
				return newExitCodeError(2, fmt.Errorf("gormes migrate hermes: %w", err))
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			if err := enc.Encode(m); err != nil {
				return fmt.Errorf("gormes migrate hermes: encode manifest: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&source, "source", "", "explicit Hermes home directory; preferred over $HERMES_HOME and ~/.hermes")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the migration manifest without writing any Gormes file (required in this slice)")
	return cmd
}

func newMigrateOpenClawCommand() *cobra.Command {
	var (
		source string
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:          "openclaw",
		Short:        "Build a redacted dry-run manifest for migrating OpenClaw config, env, memory, user, and skill surfaces into Gormes",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !dryRun {
				return newExitCodeError(2, errors.New("gormes migrate openclaw: --dry-run is required in this slice; the writer subcommand is not implemented yet"))
			}
			m, err := openclawmigrate.BuildManifest(openclawmigrate.Options{
				Source:            strings.TrimSpace(source),
				ExistingGormesEnv: collectMigrationEnvSnapshot(),
			})
			if err != nil {
				return newExitCodeError(2, fmt.Errorf("gormes migrate openclaw: %w", err))
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			if err := enc.Encode(m); err != nil {
				return fmt.Errorf("gormes migrate openclaw: encode manifest: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&source, "source", "", "explicit OpenClaw home directory; preferred over ~/.openclaw, ~/.clawdbot, and ~/.moltbot")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the migration manifest without writing any Gormes file (required in this slice)")
	return cmd
}

// collectGormesEnvSnapshot returns the GORMES_* env keys currently set
// on the running process, so the manifest can mark Hermes .env keys
// that would overwrite already-set Gormes values as conflict. Only
// names are looked up; raw secret bytes never reach the manifest.
func collectGormesEnvSnapshot() map[string]string {
	out := make(map[string]string)
	for _, kv := range os.Environ() {
		eq := strings.IndexByte(kv, '=')
		if eq <= 0 {
			continue
		}
		k := kv[:eq]
		if !strings.HasPrefix(k, "GORMES_") {
			continue
		}
		out[k] = kv[eq+1:]
	}
	return out
}

func collectMigrationEnvSnapshot() map[string]string {
	out := collectGormesEnvSnapshot()
	for _, key := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "OPENROUTER_API_KEY"} {
		if value, ok := os.LookupEnv(key); ok {
			out[key] = value
		}
	}
	return out
}
