package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

type promptTemplateCatalogOptions struct {
	Paths    []string
	Disabled bool
}

func tuiPromptTemplateCatalog(cfg config.Config, cwd string, opts promptTemplateCatalogOptions) tui.PromptTemplateCatalog {
	return gormescli.PromptTemplateCatalog(cfg, cwd, gormescli.PromptTemplateCatalogOptions{
		Paths:    opts.Paths,
		Disabled: opts.Disabled,
	})
}
