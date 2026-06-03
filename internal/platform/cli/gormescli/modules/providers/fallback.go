package providers

import (
	"github.com/spf13/cobra"

	appfallback "github.com/TrebuchetDynamics/gormes-agent/internal/app/fallback"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

type FallbackEntry = appfallback.FallbackEntry

type FallbackConfig = appfallback.FallbackConfig

func NewFallbackCommand(seams ModelCommandSeams) *cobra.Command {
	return NewFallbackCommandWithSeams(seams)
}

func NewFallbackCommandWithSeams(seams ModelCommandSeams) *cobra.Command {
	return appfallback.NewFallbackCommandWithSeams(appfallback.ModelCommandSeams(seams))
}

func LoadFallbackConfig(path string) (FallbackConfig, error) {
	return appfallback.LoadFallbackConfig(path)
}

func loadFallbackConfig(path string) (FallbackConfig, error) {
	return LoadFallbackConfig(path)
}

func AppendFallbackSelection(path string, selection cli.Selection) (bool, error) {
	return appfallback.AppendFallbackSelection(path, selection)
}

func WriteFallbackChain(path string, chain []FallbackEntry) error {
	return appfallback.WriteFallbackChain(path, chain)
}
