package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/providerregistry"

type PickerProvider = providerregistry.PickerProvider

func ListPickerProviders() []PickerProvider {
	return providerregistry.ListPickerProviders()
}
