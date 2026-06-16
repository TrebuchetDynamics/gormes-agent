package repoctl

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/fidelity"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/repoctl/inventory"
)

type HermesContractInventoryOptions = inventory.HermesContractInventoryOptions
type HermesContractInventoryResult = inventory.HermesContractInventoryResult

func WriteHermesContractInventory(opts HermesContractInventoryOptions) (HermesContractInventoryResult, error) {
	return inventory.WriteHermesContractInventory(opts)
}

func BuildHermesContractInventory(opts HermesContractInventoryOptions) (fidelity.Report, error) {
	return inventory.BuildHermesContractInventory(opts)
}

func RenderHermesContractInventoryMarkdown(report fidelity.Report) string {
	return inventory.RenderHermesContractInventoryMarkdown(report)
}
