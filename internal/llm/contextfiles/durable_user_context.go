package contextfiles

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/contextfiles/durable"

type DurableUserContextOptions = durable.DurableUserContextOptions

type DurableUserContextReport = durable.DurableUserContextReport

func BuildDurableUserContextPrompt(opts DurableUserContextOptions) (string, DurableUserContextReport) {
	return durable.BuildDurableUserContextPrompt(opts)
}
