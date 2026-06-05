package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/contextfiles"

const durableUserContextDefaultMaxChars = 20000

type DurableUserContextOptions = contextfiles.DurableUserContextOptions
type DurableUserContextReport = contextfiles.DurableUserContextReport

func BuildDurableUserContextPrompt(opts DurableUserContextOptions) (string, DurableUserContextReport) {
	return contextfiles.BuildDurableUserContextPrompt(opts)
}
