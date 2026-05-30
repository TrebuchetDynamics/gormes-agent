package gormescli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/contractruntime"

type SetupSection = contractruntime.SetupSection
type SetupRegistry = contractruntime.SetupRegistry

const (
	SetupModuleGateway   = contractruntime.SetupModuleGateway
	SetupModuleNavivox   = contractruntime.SetupModuleNavivox
	SetupModuleProviders = contractruntime.SetupModuleProviders
	SetupModuleTools     = contractruntime.SetupModuleTools
	SetupModuleTTS       = contractruntime.SetupModuleTTS
	SetupModuleTUI       = contractruntime.SetupModuleTUI
)

func NewSetupRegistry(sections []SetupSection) (*SetupRegistry, error) {
	return contractruntime.NewSetupRegistry(sections)
}

func MustSetupRegistry(sections []SetupSection) *SetupRegistry {
	return contractruntime.MustSetupRegistry(sections)
}
