package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/providerregistry"

type ProviderTimeoutConfigLoader = providerregistry.ProviderTimeoutConfigLoader
type ProviderTimeoutEvidence = providerregistry.ProviderTimeoutEvidence

const (
	ProviderTimeoutConfigured        ProviderTimeoutEvidence = providerregistry.ProviderTimeoutConfigured
	ProviderTimeoutConfigUnavailable ProviderTimeoutEvidence = providerregistry.ProviderTimeoutConfigUnavailable
	ProviderTimeoutConfigInvalid     ProviderTimeoutEvidence = providerregistry.ProviderTimeoutConfigInvalid
	ProviderTimeoutUnset             ProviderTimeoutEvidence = providerregistry.ProviderTimeoutUnset
)

type ProviderTimeoutResolution = providerregistry.ProviderTimeoutResolution

func ResolveProviderRequestTimeout(load ProviderTimeoutConfigLoader, providerID, model string) ProviderTimeoutResolution {
	return providerregistry.ResolveProviderRequestTimeout(load, providerID, model)
}

func ResolveProviderStaleTimeout(load ProviderTimeoutConfigLoader, providerID, model string) ProviderTimeoutResolution {
	return providerregistry.ResolveProviderStaleTimeout(load, providerID, model)
}
