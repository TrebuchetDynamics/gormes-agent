package plugins

// LoadTeamsPipeline loads Hermes' Teams meeting pipeline plugin as inert
// metadata. It does not import the Python plugin, build Microsoft Graph
// clients, open subscriptions, fetch transcripts, write Notion pages, or send
// Teams summaries.
func LoadTeamsPipeline(dir string, opts LoadOptions) PluginStatus {
	status := LoadDir(dir, opts)
	if status.Manifest.Name != "teams_pipeline" {
		return status
	}

	runtime := []Evidence{
		evidence(EvidenceTeamsPipelineRuntimeUnavailable, "teams_pipeline", "Teams pipeline runtime is not yet implemented in Go; metadata discovery is disabled-only"),
		evidence(EvidenceGraphCredentialsRequired, "MSGRAPH_CLIENT_ID", "Microsoft Graph client credentials are required before subscriptions, transcript fetch, or token health can run"),
		evidence(EvidenceGraphCredentialsRequired, "MSGRAPH_CLIENT_SECRET", "Microsoft Graph client credentials are required before subscriptions, transcript fetch, or token health can run"),
		evidence(EvidenceGraphCredentialsRequired, "MSGRAPH_TENANT_ID", "Microsoft Graph tenant configuration is required before subscriptions, transcript fetch, or token health can run"),
		evidence(EvidenceTeamsDeliveryTargetRequired, "teams_delivery", "Teams delivery needs an incoming_webhook_url or Graph chat/channel target before summaries can be sent"),
	}
	status.Evidence = append(status.Evidence, runtime...)
	for i := range status.Capabilities {
		if status.Capabilities[i].Kind == CapabilityCLICommand {
			status.Capabilities[i].Evidence = append(status.Capabilities[i].Evidence, runtime...)
		}
	}
	return sortPluginStatus(status)
}

// TeamsPipelinePluginYAML mirrors the upstream Hermes manifest at
// hermes-agent/plugins/teams_pipeline/plugin.yaml@968082707.
const TeamsPipelinePluginYAML = `name: teams_pipeline
version: 0.1.0
description: "Microsoft Teams meeting pipeline plugin with durable runtime state and operator CLI flows for Graph-backed transcript-first meeting summaries."
author: NousResearch
kind: standalone
platforms:
  - linux
  - macos
  - windows
`

// TeamsPipelinePluginInitPy mirrors the upstream CLI registration without any
// runtime imports or side effects.
const TeamsPipelinePluginInitPy = `"""Teams meeting pipeline plugin.

Registers only operator-facing CLI surfaces. The agent should invoke these via
the terminal tool; no model tools are added by this plugin.
"""

from __future__ import annotations

from plugins.teams_pipeline.cli import register_cli, teams_pipeline_command


def register(ctx) -> None:
    ctx.register_cli_command(
        name="teams-pipeline",
        help="Inspect and operate the Microsoft Teams meeting pipeline",
        setup_fn=register_cli,
        handler_fn=teams_pipeline_command,
        description=(
            "Operator CLI for the Microsoft Teams meeting pipeline. "
            "Lists jobs, inspects stored runs, replays jobs, validates Graph "
            "setup, and maintains Graph subscriptions."
        ),
    )
`
