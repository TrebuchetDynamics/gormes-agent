package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli/modules/providers"
)

type fallbackEntry = providers.FallbackEntry
type fallbackConfig = providers.FallbackConfig

func newFallbackCommand() *cobra.Command {
	return newFallbackCommandWithSeams(defaultModelCommandSeams())
}

func newFallbackCommandWithSeams(seams modelCommandSeams) *cobra.Command {
	return providers.NewFallbackCommandWithSeams(providerModelCommandSeams(seams))
}

func providerModelCommandSeams(seams modelCommandSeams) providers.ModelCommandSeams {
	return providers.ModelCommandSeams{
		IsTTY:            seams.IsTTY,
		LoadCurrent:      seams.LoadCurrent,
		ListProviders:    seams.ListProviders,
		ChooseProvider:   seams.ChooseProvider,
		ChooseModel:      seams.ChooseModel,
		PersistSelection: seams.PersistSelection,
	}
}

func loadFallbackConfig(path string) (fallbackConfig, error) {
	return providers.LoadFallbackConfig(path)
}
