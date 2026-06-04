package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers"
)

func newFallbackCommand() *cobra.Command {
	return newFallbackCommandWithSeams(defaultModelCommandSeams())
}

func newFallbackCommandWithSeams(seams modelCommandSeams) *cobra.Command {
	return providers.NewFallbackCommandWithSeams(providerModelCommandSeams(seams))
}
