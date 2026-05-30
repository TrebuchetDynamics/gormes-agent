// Package gormescli provides the public compatibility facade for importable
// gormes command contracts and command factories.
package gormescli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/contractruntime"

type ModuleContract = contractruntime.ModuleContract
type CommandSpec = contractruntime.CommandSpec
type SetupSectionSpec = contractruntime.SetupSectionSpec
type SlashCommandSpec = contractruntime.SlashCommandSpec
type ReportSpec = contractruntime.ReportSpec
type CommandManifestEntry = contractruntime.CommandManifestEntry
type Registry = contractruntime.Registry

func NewRegistry(modules []ModuleContract) (*Registry, error) {
	return contractruntime.NewRegistry(modules)
}

func DefaultContracts() []ModuleContract {
	return contractruntime.DefaultContracts()
}
