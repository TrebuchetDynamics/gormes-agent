package providers

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

// SetupSections declares the provider-owned setup surfaces.
func SetupSections() []gormescli.SetupSection {
	return []gormescli.SetupSection{
		{Name: "provider", Label: "Provider", Module: progress.ModuleProviders},
		{Name: "model", Label: "Model", Module: progress.ModuleProviders},
		{Name: "fallback", Label: "Fallback Providers", Module: progress.ModuleProviders},
	}
}
