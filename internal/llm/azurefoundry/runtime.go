package azurefoundry

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry/runtimecfg"

// AzureFoundryConfig is the subset of config.yaml that drives Azure Foundry runtime resolution.
type AzureFoundryConfig = runtimecfg.AzureFoundryConfig

// AzureFoundryExplicit captures explicit overrides (e.g., flags or kwargs to a setup wizard call).
type AzureFoundryExplicit = runtimecfg.AzureFoundryExplicit

type AzureFoundryBaseURLSource = runtimecfg.AzureFoundryBaseURLSource

const (
	AzureFoundryBaseURLSourceUnset    AzureFoundryBaseURLSource = runtimecfg.AzureFoundryBaseURLSourceUnset
	AzureFoundryBaseURLSourceExplicit AzureFoundryBaseURLSource = runtimecfg.AzureFoundryBaseURLSourceExplicit
	AzureFoundryBaseURLSourceConfig   AzureFoundryBaseURLSource = runtimecfg.AzureFoundryBaseURLSourceConfig
	AzureFoundryBaseURLSourceEnv      AzureFoundryBaseURLSource = runtimecfg.AzureFoundryBaseURLSourceEnv
)

type AzureFoundryKeySource = runtimecfg.AzureFoundryKeySource

const (
	AzureFoundryKeySourceUnset    AzureFoundryKeySource = runtimecfg.AzureFoundryKeySourceUnset
	AzureFoundryKeySourceExplicit AzureFoundryKeySource = runtimecfg.AzureFoundryKeySourceExplicit
	AzureFoundryKeySourceEnv      AzureFoundryKeySource = runtimecfg.AzureFoundryKeySourceEnv
)

type AzureFoundryAPIModeSource = runtimecfg.AzureFoundryAPIModeSource

const (
	AzureFoundryAPIModeSourceUnset    AzureFoundryAPIModeSource = runtimecfg.AzureFoundryAPIModeSourceUnset
	AzureFoundryAPIModeSourceConfig   AzureFoundryAPIModeSource = runtimecfg.AzureFoundryAPIModeSourceConfig
	AzureFoundryAPIModeSourceProbe    AzureFoundryAPIModeSource = runtimecfg.AzureFoundryAPIModeSourceProbe
	AzureFoundryAPIModeSourceInferred AzureFoundryAPIModeSource = runtimecfg.AzureFoundryAPIModeSourceInferred
)

// AzureFoundryRuntimeInput is the deterministic fixture that drives ResolveAzureFoundryRuntime.
type AzureFoundryRuntimeInput = runtimecfg.AzureFoundryRuntimeInput

// AzureFoundryRuntime is the redacted read-model the kernel and CLI status surfaces consume.
type AzureFoundryRuntime = runtimecfg.AzureFoundryRuntime

// ResolveAzureFoundryRuntime computes the redacted Azure Foundry runtime read model from deterministic inputs.
func ResolveAzureFoundryRuntime(in AzureFoundryRuntimeInput) AzureFoundryRuntime {
	return runtimecfg.ResolveAzureFoundryRuntime(in)
}
