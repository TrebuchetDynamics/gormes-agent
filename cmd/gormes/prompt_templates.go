package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

type promptTemplateCatalogOptions = gormescli.PromptTemplateCatalogOptions

func tuiPromptTemplateCatalog(cfg config.Config, cwd string, opts promptTemplateCatalogOptions) tui.PromptTemplateCatalog {
	return gormescli.PromptTemplateCatalog(cfg, cwd, opts)
}
