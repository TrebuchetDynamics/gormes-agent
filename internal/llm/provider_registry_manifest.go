// Code generated from Hermes provider inventory by Mineru; edit deliberately.
package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/providerregistry"

type ProviderImplementationStatus = providerregistry.ProviderImplementationStatus

const (
	ProviderImplemented ProviderImplementationStatus = providerregistry.ProviderImplemented
	ProviderOwned       ProviderImplementationStatus = providerregistry.ProviderOwned
	ProviderRowBacked   ProviderImplementationStatus = providerregistry.ProviderRowBacked
	ProviderExcluded    ProviderImplementationStatus = providerregistry.ProviderExcluded
)

type ProviderManifestEntry = providerregistry.ProviderManifestEntry

func HermesProviderRegistryManifest() []ProviderManifestEntry {
	return providerregistry.HermesProviderRegistryManifest()
}

func ResolveProviderManifestEntry(provider string) (ProviderManifestEntry, bool) {
	return providerregistry.ResolveProviderManifestEntry(provider)
}
