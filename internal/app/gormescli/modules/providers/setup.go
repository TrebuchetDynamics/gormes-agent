package providers

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

// SetupSections declares the provider-owned setup surfaces.
func SetupSections() []gormescli.SetupSection {
	return []gormescli.SetupSection{
		{Name: "provider", Label: "Provider", Module: progress.ModuleProviders},
		{Name: "model", Label: "Model", Module: progress.ModuleProviders},
		{Name: "fallback", Label: "Fallback Providers", Module: progress.ModuleProviders},
	}
}
