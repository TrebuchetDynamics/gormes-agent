package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/agenttemplate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/goncho"
)

type agentResetOptions struct {
	Target string
	Force  bool
	DryRun bool
	JSON   bool
}

// agentResetReportJSON is the wire shape for `agent reset --json`.
// Fleet automation seeding agent context across many machines parses
// this to confirm which template files landed (or, in dry-run mode,
// which would land). Build provenance leads — same convention as the
// rest of the `--json` arc.
type agentResetReportJSON struct {
	Build  buildProvenanceJSON       `json:"build"`
	Target string                    `json:"target"`
	DryRun bool                      `json:"dry_run"`
	Files  []agentResetFileJSON      `json:"files"`
}

type agentResetFileJSON struct {
	Path   string `json:"path"`
	Action string `json:"action"`
}

func newAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "agent",
		Short:        "Manage Gormes agent context templates",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
	}
	cmd.AddCommand(
		newAgentResetCommand(),
		newAgentSpawnCommand(),
		newAgentListCommand(),
		newAgentBindCommand(),
		newAgentUnbindCommand(),
		newAgentInspectCommand(),
	)
	return cmd
}

// agentBindingMatchFlags wires the shared (channel, peer-kind, peer-id,
// thread-id) flag set used by bind, unbind, and inspect.
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

func (f agentBindingMatchFlags) toMatch() goncho.BindingMatch {
	return goncho.BindingMatch{
		Channel:  f.Channel,
		PeerKind: f.PeerKind,
		PeerID:   f.PeerID,
		ThreadID: f.ThreadID,
	}
}

type agentBindingReportJSON struct {
	Build buildProvenanceJSON  `json:"build"`
	Match agentBindingMatchJSON `json:"match"`
	// AgentID is populated by bind (the bound agent) and by inspect when
	// Bound=true; omitted when Bound=false so the JSON shape stays small.
	AgentID string `json:"agent_id,omitempty"`
	// Bound is only meaningful for inspect; bind and unbind reports omit it.
	Bound *bool `json:"bound,omitempty"`
	// Removed is only meaningful for unbind; nil otherwise.
	Removed *bool `json:"removed,omitempty"`
}

type agentBindingMatchJSON struct {
	Channel  string `json:"channel"`
	PeerKind string `json:"peer_kind"`
	PeerID   string `json:"peer_id"`
	ThreadID string `json:"thread_id,omitempty"`
}

func bindingMatchToJSON(m goncho.BindingMatch) agentBindingMatchJSON {
	return agentBindingMatchJSON{
		Channel:  m.Channel,
		PeerKind: m.PeerKind,
		PeerID:   m.PeerID,
		ThreadID: m.ThreadID,
	}
}

func newAgentBindCommand() *cobra.Command {
	var match agentBindingMatchFlags
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "bind <agent_id>",
		Short:        "Bind a (channel, peer, thread) tuple to a runtime-spawned agent",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, cleanup, err := openDynamicAgentRegistry()
			if err != nil {
				return err
			}
			defer cleanup()

			m := match.toMatch()
			if err := reg.Bind(cmd.Context(), args[0], m); err != nil {
				return fmt.Errorf("gormes agent bind: %w", err)
			}
			if asJSON {
				body, marshalErr := json.MarshalIndent(agentBindingReportJSON{
					Build:   newBuildProvenance(),
					Match:   bindingMatchToJSON(m),
					AgentID: args[0],
				}, "", "  ")
				if marshalErr != nil {
					return marshalErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(body))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "bound %s -> %s/%s/%s/%s\n",
				args[0], m.Channel, m.PeerKind, m.PeerID, m.ThreadID)
			return nil
		},
	}
	match.attach(cmd)
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `{build, match, agent_id}`")
	return cmd
}

func newAgentUnbindCommand() *cobra.Command {
	var match agentBindingMatchFlags
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "unbind",
		Short:        "Remove a runtime binding for the given (channel, peer, thread) tuple",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			reg, cleanup, err := openDynamicAgentRegistry()
			if err != nil {
				return err
			}
			defer cleanup()

			m := match.toMatch()
			if err := reg.Unbind(cmd.Context(), m); err != nil {
				return fmt.Errorf("gormes agent unbind: %w", err)
			}
			if asJSON {
				removed := true
				body, marshalErr := json.MarshalIndent(agentBindingReportJSON{
					Build:   newBuildProvenance(),
					Match:   bindingMatchToJSON(m),
					Removed: &removed,
				}, "", "  ")
				if marshalErr != nil {
					return marshalErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(body))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "unbound %s/%s/%s/%s\n",
				m.Channel, m.PeerKind, m.PeerID, m.ThreadID)
			return nil
		},
	}
	match.attach(cmd)
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `{build, match, removed}`")
	return cmd
}

func newAgentInspectCommand() *cobra.Command {
	var match agentBindingMatchFlags
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "inspect",
		Short:        "Resolve a (channel, peer, thread) tuple to its bound agent, if any",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			reg, cleanup, err := openDynamicAgentRegistry()
			if err != nil {
				return err
			}
			defer cleanup()

			m := match.toMatch()
			agentID, found, err := reg.Resolve(cmd.Context(), m)
			if err != nil {
				return fmt.Errorf("gormes agent inspect: %w", err)
			}
			if asJSON {
				bound := found
				report := agentBindingReportJSON{
					Build: newBuildProvenance(),
					Match: bindingMatchToJSON(m),
					Bound: &bound,
				}
				if found {
					report.AgentID = agentID
				}
				body, marshalErr := json.MarshalIndent(report, "", "  ")
				if marshalErr != nil {
					return marshalErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(body))
				return nil
			}
			if !found {
				fmt.Fprintln(cmd.ErrOrStderr(), "agent_not_bound")
				return newExitCodeError(1, fmt.Errorf("agent_not_bound"))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "agent: %s\n", agentID)
			return nil
		},
	}
	match.attach(cmd)
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `{build, match, bound, agent_id}`")
	return cmd
}

// agentSpawnOptions parameterizes `gormes agent spawn`.
type agentSpawnOptions struct {
	Persona string
	JSON    bool
}

// agentRecordJSON is the wire shape for one dynamic agent in --json output.
// Used by spawn, list, and inspect. Build provenance leads at the report
// level — same convention as the rest of the --json arc.
type agentRecordJSON struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Persona   string `json:"persona,omitempty"`
	CreatedAt string `json:"created_at"`
}

type agentSpawnReportJSON struct {
	Build buildProvenanceJSON `json:"build"`
	Agent agentRecordJSON     `json:"agent"`
}

type agentListReportJSON struct {
	Build  buildProvenanceJSON `json:"build"`
	Agents []agentRecordJSON   `json:"agents"`
}

func newAgentSpawnCommand() *cobra.Command {
	opts := agentSpawnOptions{}
	cmd := &cobra.Command{
		Use:          "spawn <name>",
		Short:        "Spawn a runtime-only agent persona in the dynamic registry",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentSpawnCommand(cmd, args[0], opts)
		},
	}
	cmd.Flags().StringVar(&opts.Persona, "persona", "", "free-text persona seed injected into the agent's system prompt")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "emit machine-readable JSON: `{build, agent: {id, name, persona, created_at}}`")
	return cmd
}

func runAgentSpawnCommand(cmd *cobra.Command, name string, opts agentSpawnOptions) error {
	reg, cleanup, err := openDynamicAgentRegistry()
	if err != nil {
		return err
	}
	defer cleanup()

	record, err := reg.Create(cmd.Context(), goncho.CreateAgentOptions{
		Name:    name,
		Persona: opts.Persona,
	})
	if err != nil {
		if errors.Is(err, goncho.ErrAgentIDInvalid) {
			return newExitCodeError(2, fmt.Errorf("agent_id_invalid: name %q does not normalize to a valid agent id (^[a-z][a-z0-9_-]{0,63}$)", name))
		}
		return fmt.Errorf("gormes agent spawn: %w", err)
	}
	if opts.JSON {
		body, marshalErr := json.MarshalIndent(agentSpawnReportJSON{
			Build: newBuildProvenance(),
			Agent: agentRecordToJSON(record),
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "spawned agent %s (%s)\n", record.ID, record.Name)
	return nil
}

func newAgentListCommand() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List runtime-spawned agents in the dynamic registry",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAgentListCommand(cmd, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `{build, agents: [{id, name, persona, created_at}]}`")
	return cmd
}

func runAgentListCommand(cmd *cobra.Command, asJSON bool) error {
	reg, cleanup, err := openDynamicAgentRegistry()
	if err != nil {
		return err
	}
	defer cleanup()

	records, err := reg.List(cmd.Context())
	if err != nil {
		return fmt.Errorf("gormes agent list: %w", err)
	}
	if asJSON {
		report := agentListReportJSON{
			Build:  newBuildProvenance(),
			Agents: make([]agentRecordJSON, 0, len(records)),
		}
		for _, r := range records {
			report.Agents = append(report.Agents, agentRecordToJSON(r))
		}
		body, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return nil
	}
	if len(records) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no runtime-spawned agents")
		return nil
	}
	for _, r := range records {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", r.ID, r.Name)
	}
	return nil
}

func agentRecordToJSON(r goncho.AgentRecord) agentRecordJSON {
	return agentRecordJSON{
		ID:        r.ID,
		Name:      r.Name,
		Persona:   r.Persona,
		CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// openDynamicAgentRegistry opens the Goncho SQLite database used by the
// dynamic agent registry. The caller invokes cleanup() to close the
// underlying *sql.DB. The DB lives under $GORMES_HOME/memory.db, the same
// location the gateway uses for Goncho, and the open routes through
// sqlOpenGoncho so busy_timeout and WAL mode match the gateway's path.
func openDynamicAgentRegistry() (*goncho.DynamicAgentRegistry, func(), error) {
	path := config.MemoryDBPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, func() {}, fmt.Errorf("gormes agent: create memory dir: %w", err)
	}
	db, err := sqlOpenGoncho(path)
	if err != nil {
		return nil, func() {}, fmt.Errorf("gormes agent: open registry db: %w", err)
	}
	reg, err := goncho.NewDynamicAgentRegistry(db)
	if err != nil {
		_ = db.Close()
		return nil, func() {}, err
	}
	return reg, func() { _ = db.Close() }, nil
}

func newAgentResetCommand() *cobra.Command {
	opts := agentResetOptions{Target: config.GormesHome()}
	cmd := &cobra.Command{
		Use:          "reset",
		Short:        "Seed default Gormes agent context templates",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAgentResetCommand(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Target, "target", opts.Target, "target directory for agent context templates")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "overwrite existing template files")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "report reset actions without writing files")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "emit machine-readable JSON: `{build, target, dry_run, files: [{path, action}]}`")
	return cmd
}

func runAgentResetCommand(cmd *cobra.Command, opts agentResetOptions) error {
	result, err := agenttemplate.ApplyDefaultTemplates(agenttemplate.WriteOptions{
		TargetDir: opts.Target,
		Force:     opts.Force,
		DryRun:    opts.DryRun,
	})
	if err != nil {
		return fmt.Errorf("gormes agent reset: %w", err)
	}
	if opts.JSON {
		report := agentResetReportJSON{
			Build:  newBuildProvenance(),
			Target: result.TargetDir,
			DryRun: opts.DryRun,
			Files:  make([]agentResetFileJSON, len(result.Files)),
		}
		for i, f := range result.Files {
			report.Files[i] = agentResetFileJSON{
				Path:   f.Path,
				Action: string(f.Action),
			}
		}
		body, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "target: %s\n", result.TargetDir)
	for _, file := range result.Files {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", file.Action, file.Path)
	}
	return nil
}
