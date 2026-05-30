package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/contextfiles"

type IdentityLoaderOptions = contextfiles.IdentityLoaderOptions
type IdentityLoaderResult = contextfiles.IdentityLoaderResult

func LoadAgentIdentity(opts IdentityLoaderOptions) IdentityLoaderResult {
	return contextfiles.LoadAgentIdentity(opts)
}
