package agent

import (
	"fmt"

	"github.com/spf13/cobra"
)

type AgentCommandOptions struct {
	DefaultResetTarget string
}

type AgentResetOptions struct {
	Target string
	Force  bool
	DryRun bool
	JSON   bool
}

type AgentSpawnOptions struct {
	Persona string
	JSON    bool
}

type AgentBindingMatch struct {
	Channel  string
	PeerKind string
	PeerID   string
	ThreadID string
}

type AgentCommandSeams struct {
	Reset   func(*cobra.Command, AgentResetOptions) error
	Spawn   func(*cobra.Command, string, AgentSpawnOptions) error
	List    func(*cobra.Command, bool) error
	Bind    func(*cobra.Command, string, AgentBindingMatch, bool) error
	Unbind  func(*cobra.Command, AgentBindingMatch, bool) error
	Inspect func(*cobra.Command, AgentBindingMatch, bool) error
}

func NewAgentCommandWithSeams(seams AgentCommandSeams, opts AgentCommandOptions) *cobra.Command {
	seams = seams.withDefaults()
	cmd := &cobra.Command{
		Use:          "agent",
		Short:        "Manage Gormes agent context templates",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
	}
	cmd.AddCommand(
		newAgentResetCommand(seams, opts),
		newAgentSpawnCommand(seams),
		newAgentListCommand(seams),
		newAgentBindCommand(seams),
		newAgentUnbindCommand(seams),
		newAgentInspectCommand(seams),
	)
	return cmd
}

func (s AgentCommandSeams) withDefaults() AgentCommandSeams {
	if s.Reset == nil {
		s.Reset = func(*cobra.Command, AgentResetOptions) error { return fmt.Errorf("agent reset seam is not configured") }
	}
	if s.Spawn == nil {
		s.Spawn = func(*cobra.Command, string, AgentSpawnOptions) error {
			return fmt.Errorf("agent spawn seam is not configured")
		}
	}
	if s.List == nil {
		s.List = func(*cobra.Command, bool) error { return fmt.Errorf("agent list seam is not configured") }
	}
	if s.Bind == nil {
		s.Bind = func(*cobra.Command, string, AgentBindingMatch, bool) error {
			return fmt.Errorf("agent bind seam is not configured")
		}
	}
	if s.Unbind == nil {
		s.Unbind = func(*cobra.Command, AgentBindingMatch, bool) error {
			return fmt.Errorf("agent unbind seam is not configured")
		}
	}
	if s.Inspect == nil {
		s.Inspect = func(*cobra.Command, AgentBindingMatch, bool) error {
			return fmt.Errorf("agent inspect seam is not configured")
		}
	}
	return s
}

func newAgentResetCommand(seams AgentCommandSeams, cmdOpts AgentCommandOptions) *cobra.Command {
	opts := AgentResetOptions{Target: cmdOpts.DefaultResetTarget}
	cmd := &cobra.Command{
		Use:          "reset",
		Short:        "Seed default Gormes agent context templates",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return seams.Reset(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Target, "target", opts.Target, "target directory for agent context templates")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "overwrite existing template files")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "report reset actions without writing files")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "emit machine-readable JSON: `{build, target, dry_run, files: [{path, action}]}`")
	return cmd
}

func newAgentSpawnCommand(seams AgentCommandSeams) *cobra.Command {
	opts := AgentSpawnOptions{}
	cmd := &cobra.Command{
		Use:          "spawn <name>",
		Short:        "Spawn a runtime-only agent persona in the dynamic registry",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return seams.Spawn(cmd, args[0], opts)
		},
	}
	cmd.Flags().StringVar(&opts.Persona, "persona", "", "free-text persona seed injected into the agent's system prompt")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "emit machine-readable JSON: `{build, agent: {id, name, persona, created_at}}`")
	return cmd
}

func newAgentListCommand(seams AgentCommandSeams) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List runtime-spawned agents in the dynamic registry",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return seams.List(cmd, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `{build, agents: [{id, name, persona, created_at}]}`")
	return cmd
}

type agentBindingMatchFlags struct {
	Channel  string
	PeerKind string
	PeerID   string
	ThreadID string
}

func (f *agentBindingMatchFlags) attach(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.Channel, "channel", "", "messaging channel (telegram, discord, slack, ...)")
	cmd.Flags().StringVar(&f.PeerKind, "peer-kind", "group", "peer kind: group or direct")
	cmd.Flags().StringVar(&f.PeerID, "peer-id", "", "channel-specific peer identifier (chat id, channel id, conversation id)")
	cmd.Flags().StringVar(&f.ThreadID, "thread-id", "", "optional thread/topic identifier inside the peer")
	_ = cmd.MarkFlagRequired("channel")
	_ = cmd.MarkFlagRequired("peer-id")
}

func (f agentBindingMatchFlags) toMatch() AgentBindingMatch {
	return AgentBindingMatch{
		Channel:  f.Channel,
		PeerKind: f.PeerKind,
		PeerID:   f.PeerID,
		ThreadID: f.ThreadID,
	}
}

func newAgentBindCommand(seams AgentCommandSeams) *cobra.Command {
	var match agentBindingMatchFlags
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "bind <agent_id>",
		Short:        "Bind a (channel, peer, thread) tuple to a runtime-spawned agent",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return seams.Bind(cmd, args[0], match.toMatch(), asJSON)
		},
	}
	match.attach(cmd)
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `{build, match, agent_id}`")
	return cmd
}

func newAgentUnbindCommand(seams AgentCommandSeams) *cobra.Command {
	var match agentBindingMatchFlags
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "unbind",
		Short:        "Remove a runtime binding for the given (channel, peer, thread) tuple",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return seams.Unbind(cmd, match.toMatch(), asJSON)
		},
	}
	match.attach(cmd)
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `{build, match, removed}`")
	return cmd
}

func newAgentInspectCommand(seams AgentCommandSeams) *cobra.Command {
	var match agentBindingMatchFlags
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "inspect",
		Short:        "Resolve a (channel, peer, thread) tuple to its bound agent, if any",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return seams.Inspect(cmd, match.toMatch(), asJSON)
		},
	}
	match.attach(cmd)
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `{build, match, bound, agent_id}`")
	return cmd
}
