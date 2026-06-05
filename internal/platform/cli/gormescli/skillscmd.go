package gormescli

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	appskillscmd "github.com/TrebuchetDynamics/gormes-agent/internal/app/skillscmd"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	skillruntime "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

type SkillsBuildProvenance = appskillscmd.BuildProvenance
type SkillsProfileSyncSeams = appskillscmd.ProfileSyncSeams
type SkillsSyncOptions = appskillscmd.SyncOptions

type SkillsCLICommandOptions struct {
	SyncSeams          SkillsProfileSyncSeams
	BuildProvenance    func() BuildProvenance
	Row                string
	UnavailableCommand func(RowBackedCommandSpec) *cobra.Command
}

func ListInstalledSkills(opts skillruntime.ListOptions, disabled map[string]struct{}) []skillruntime.SkillRow {
	return appskillscmd.ListInstalledSkills(opts, disabled)
}

func SkillsURLInstallDeps() cli.SkillsURLInstallDeps {
	return appskillscmd.URLInstallDeps()
}

func SkillsCommandOptionsForConfig(cfg config.Config) gateway.SkillsCommandOptions {
	return appskillscmd.CommandOptionsForConfig(cfg)
}

func RunSkillsProfileSync(ctx context.Context, out io.Writer, seams SkillsProfileSyncSeams, opts SkillsSyncOptions) error {
	return appskillscmd.RunProfileSync(ctx, out, seams, opts)
}

func DefaultSkillProfileRoots() ([]skillruntime.SkillProfileRoot, error) {
	return appskillscmd.DefaultProfileRoots()
}

func NewSkillsCommand(opts SkillsCLICommandOptions) *cobra.Command {
	cmd := cli.NewSkillsCommand(cli.SkillsCommandDeps{
		ListInstalledSkills: ListInstalledSkills,
		DisabledSkills:      func(string) map[string]struct{} { return nil },
		URLInstall:          SkillsURLInstallDeps(),
		BuildProvenance:     func() any { return skillsBuildProvenance(opts) },
	})
	cmd.AddCommand(newSkillsSyncCommand(opts))
	cmd.AddCommand(newSkillsRowBackedCommands(opts)...)
	return cmd
}

func newSkillsSyncCommand(opts SkillsCLICommandOptions) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync bundled skills into all configured profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			build := skillsBuildProvenance(opts)
			return RunSkillsProfileSync(cmd.Context(), cmd.OutOrStdout(), opts.SyncSeams, SkillsSyncOptions{
				JSON:  jsonOut,
				Build: SkillsBuildProvenance{Version: build.Version, GitCommit: build.GitCommit},
			})
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON: `{build, summaries: [{profile, added, unchanged, conflicts, failed}]}`")
	return cmd
}

func newSkillsRowBackedCommands(opts SkillsCLICommandOptions) []*cobra.Command {
	return []*cobra.Command{
		skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "browse", Short: "Browse the Hermes skills hub", Row: opts.Row}),
		skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "search <query>", Short: "Search the Hermes skills hub", Row: opts.Row}),
		skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "inspect <name>", Short: "Inspect a skill manifest", Row: opts.Row}),
		skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "check", Short: "Check installed skill health", Row: opts.Row}),
		skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "update <name>", Short: "Update an installed skill", Row: opts.Row}),
		skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "audit", Short: "Audit installed skills", Row: opts.Row}),
		skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "uninstall <name>", Short: "Uninstall a skill", Row: opts.Row, Destructive: true, FlagSet: skillsUnavailableYesFlag}),
		skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "reset", Short: "Reset installed skills", Row: opts.Row, Destructive: true, FlagSet: skillsUnavailableYesFlag}),
		skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "publish <path>", Short: "Publish a skill", Row: opts.Row}),
		skillsUnavailableParent("snapshot", "Manage skill snapshots",
			skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "export", Short: "Export a skill snapshot", Row: opts.Row}),
			skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "import <path>", Short: "Import a skill snapshot", Row: opts.Row}),
		),
		skillsUnavailableParent("tap", "Manage skill taps",
			skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "list", Aliases: []string{"ls"}, Short: "List configured skill taps", Row: opts.Row}),
			skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "add <url>", Short: "Add a skill tap", Row: opts.Row}),
			skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "remove <name>", Aliases: []string{"rm"}, Short: "Remove a skill tap", Row: opts.Row, Destructive: true, FlagSet: skillsUnavailableYesFlag}),
		),
		skillsUnavailableCommand(opts, RowBackedCommandSpec{Use: "config", Short: "Show skill hub configuration", Row: opts.Row}),
	}
}

func skillsUnavailableParent(use, short string, children ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{Use: use, Short: short}
	cmd.AddCommand(children...)
	return cmd
}

func skillsUnavailableCommand(opts SkillsCLICommandOptions, spec RowBackedCommandSpec) *cobra.Command {
	if opts.UnavailableCommand != nil {
		return opts.UnavailableCommand(spec)
	}
	return NewRowBackedCommand(spec, RowBackedCommandOptions{})
}

func skillsUnavailableYesFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation")
}

func skillsBuildProvenance(opts SkillsCLICommandOptions) BuildProvenance {
	if opts.BuildProvenance == nil {
		return BuildProvenance{}
	}
	return opts.BuildProvenance()
}
