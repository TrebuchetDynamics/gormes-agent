package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry"

type AzureFoundryConfig = azurefoundry.AzureFoundryConfig
type AzureFoundryExplicit = azurefoundry.AzureFoundryExplicit
type AzureFoundryBaseURLSource = azurefoundry.AzureFoundryBaseURLSource

const (
	AzureFoundryBaseURLSourceUnset    = azurefoundry.AzureFoundryBaseURLSourceUnset
	AzureFoundryBaseURLSourceExplicit = azurefoundry.AzureFoundryBaseURLSourceExplicit
	AzureFoundryBaseURLSourceConfig   = azurefoundry.AzureFoundryBaseURLSourceConfig
	AzureFoundryBaseURLSourceEnv      = azurefoundry.AzureFoundryBaseURLSourceEnv
)

type AzureFoundryKeySource = azurefoundry.AzureFoundryKeySource

const (
	AzureFoundryKeySourceUnset    = azurefoundry.AzureFoundryKeySourceUnset
	AzureFoundryKeySourceExplicit = azurefoundry.AzureFoundryKeySourceExplicit
	AzureFoundryKeySourceEnv      = azurefoundry.AzureFoundryKeySourceEnv
)

type AzureFoundryAPIModeSource = azurefoundry.AzureFoundryAPIModeSource

const (
	AzureFoundryAPIModeSourceUnset    = azurefoundry.AzureFoundryAPIModeSourceUnset
	AzureFoundryAPIModeSourceConfig   = azurefoundry.AzureFoundryAPIModeSourceConfig
	AzureFoundryAPIModeSourceProbe    = azurefoundry.AzureFoundryAPIModeSourceProbe
	AzureFoundryAPIModeSourceInferred = azurefoundry.AzureFoundryAPIModeSourceInferred
)

type AzureFoundryRuntimeInput = azurefoundry.AzureFoundryRuntimeInput
type AzureFoundryRuntime = azurefoundry.AzureFoundryRuntime

func ResolveAzureFoundryRuntime(in AzureFoundryRuntimeInput) AzureFoundryRuntime {
	return azurefoundry.ResolveAzureFoundryRuntime(in)
}
