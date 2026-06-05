package gormescli

import (
	"github.com/spf13/cobra"

	agentapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/agent"
	gatewayagent "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway/agent"
)

type AgentBuildProvenance = agentapp.BuildProvenance
type AgentOptions = agentapp.Options
type AgentRegistry = agentapp.Registry

func NewAgentCommand(opts AgentOptions) *cobra.Command {
	if opts.BuildProvenance == nil {
		opts.BuildProvenance = func() AgentBuildProvenance { return AgentBuildProvenance{} }
	}
	return gatewayagent.NewAgentCommandWithSeams(gatewayagent.AgentCommandSeams{
		Reset: func(cmd *cobra.Command, reset gatewayagent.AgentResetOptions) error {
			return agentapp.RunReset(cmd.OutOrStdout(), agentResetOptions(reset), opts.BuildProvenance())
		},
		Spawn: func(cmd *cobra.Command, name string, spawn gatewayagent.AgentSpawnOptions) error {
			return agentapp.RunSpawn(cmd.Context(), cmd.OutOrStdout(), name, agentSpawnOptions(spawn), opts)
		},
		List: func(cmd *cobra.Command, asJSON bool) error {
			return agentapp.RunList(cmd.Context(), cmd.OutOrStdout(), asJSON, opts)
		},
		Bind: func(cmd *cobra.Command, agentID string, match gatewayagent.AgentBindingMatch, asJSON bool) error {
			return agentapp.RunBind(cmd.Context(), cmd.OutOrStdout(), agentID, agentBindingMatch(match), asJSON, opts)
		},
		Unbind: func(cmd *cobra.Command, match gatewayagent.AgentBindingMatch, asJSON bool) error {
			return agentapp.RunUnbind(cmd.Context(), cmd.OutOrStdout(), agentBindingMatch(match), asJSON, opts)
		},
		Inspect: func(cmd *cobra.Command, match gatewayagent.AgentBindingMatch, asJSON bool) error {
			return agentapp.RunInspect(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), agentBindingMatch(match), asJSON, opts)
		},
	}, gatewayagent.AgentCommandOptions{DefaultResetTarget: opts.DefaultResetTarget})
}

func agentResetOptions(opts gatewayagent.AgentResetOptions) agentapp.ResetOptions {
	return agentapp.ResetOptions{Target: opts.Target, Force: opts.Force, DryRun: opts.DryRun, JSON: opts.JSON}
}

func agentSpawnOptions(opts gatewayagent.AgentSpawnOptions) agentapp.SpawnOptions {
	return agentapp.SpawnOptions{Persona: opts.Persona, JSON: opts.JSON}
}

func agentBindingMatch(match gatewayagent.AgentBindingMatch) agentapp.BindingMatch {
	return agentapp.BindingMatch{Channel: match.Channel, PeerKind: match.PeerKind, PeerID: match.PeerID, ThreadID: match.ThreadID}
}
