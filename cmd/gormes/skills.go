package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	skillruntime "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type skillsProfileSyncSeams = gormescli.SkillsProfileSyncSeams

func newSkillsCommand() *cobra.Command {
	return newSkillsCommandWithProfileSync(skillsProfileSyncSeams{})
}

func newSkillsCommandWithProfileSync(syncSeams skillsProfileSyncSeams) *cobra.Command {
	cmd := cli.NewSkillsCommand(cli.SkillsCommandDeps{
		ListInstalledSkills: gormescli.ListInstalledSkills,
		DisabledSkills:      func(string) map[string]struct{} { return nil },
		URLInstall:          gormescli.SkillsURLInstallDeps(),
		BuildProvenance:     func() any { return newBuildProvenance() },
	})
	cmd.AddCommand(newSkillsSyncCommand(syncSeams))
	cmd.AddCommand(newSkillsRowBackedCommands()...)
	return cmd
}

func newSkillsRowBackedCommands() []*cobra.Command {
	return []*cobra.Command{
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "browse",
			Short: "Browse the Hermes skills hub",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "search <query>",
			Short: "Search the Hermes skills hub",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "inspect <name>",
			Short: "Inspect a skill manifest",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "check",
			Short: "Check installed skill health",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "update <name>",
			Short: "Update an installed skill",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "audit",
			Short: "Audit installed skills",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:         "uninstall <name>",
			Short:       "Uninstall a skill",
			Row:         hermesSkillsRow,
			Destructive: true,
			FlagSet:     hermesUnavailableYesFlag,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:         "reset",
			Short:       "Reset installed skills",
			Row:         hermesSkillsRow,
			Destructive: true,
			FlagSet:     hermesUnavailableYesFlag,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "publish <path>",
			Short: "Publish a skill",
			Row:   hermesSkillsRow,
		}),
		newHermesUnavailableParent(
			"snapshot",
			"Manage skill snapshots",
			newHermesUnavailableCommand(hermesUnavailableCommandSpec{
				Use:   "export",
				Short: "Export a skill snapshot",
				Row:   hermesSkillsRow,
			}),
			newHermesUnavailableCommand(hermesUnavailableCommandSpec{
				Use:   "import <path>",
				Short: "Import a skill snapshot",
				Row:   hermesSkillsRow,
			}),
		),
		newHermesUnavailableParent(
			"tap",
			"Manage skill taps",
			newHermesUnavailableCommand(hermesUnavailableCommandSpec{
				Use:     "list",
				Aliases: []string{"ls"},
				Short:   "List configured skill taps",
				Row:     hermesSkillsRow,
			}),
			newHermesUnavailableCommand(hermesUnavailableCommandSpec{
				Use:   "add <url>",
				Short: "Add a skill tap",
				Row:   hermesSkillsRow,
			}),
			newHermesUnavailableCommand(hermesUnavailableCommandSpec{
				Use:         "remove <name>",
				Aliases:     []string{"rm"},
				Short:       "Remove a skill tap",
				Row:         hermesSkillsRow,
				Destructive: true,
				FlagSet:     hermesUnavailableYesFlag,
			}),
		),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "config",
			Short: "Show skill hub configuration",
			Row:   hermesSkillsRow,
		}),
	}
}

func newSkillsSyncCommand(seams skillsProfileSyncSeams) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync bundled skills into all configured profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return gormescli.RunSkillsProfileSync(cmd.Context(), cmd.OutOrStdout(), seams, gormescli.SkillsSyncOptions{
				JSON:  jsonOut,
				Build: skillsBuildProvenance(),
			})
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON: `{build, summaries: [{profile, added, unchanged, conflicts, failed}]}`")
	return cmd
}

func skillsCommandOptionsForConfig(cfg config.Config) gateway.SkillsCommandOptions {
	return gormescli.SkillsCommandOptionsForConfig(cfg)
}

func defaultSkillSyncProfiles() ([]skillruntime.SkillProfileRoot, error) {
	return gormescli.DefaultSkillProfileRoots()
}

func skillsBuildProvenance() gormescli.SkillsBuildProvenance {
	build := newBuildProvenance()
	return gormescli.SkillsBuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}
