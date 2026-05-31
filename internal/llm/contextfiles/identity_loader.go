package contextfiles

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/contextfiles/identity"

type IdentityLoaderOptions = identity.IdentityLoaderOptions

type IdentityLoaderResult = identity.IdentityLoaderResult

func LoadAgentIdentity(opts IdentityLoaderOptions) IdentityLoaderResult {
	return identity.LoadAgentIdentity(opts)
}
