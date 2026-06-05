package gormescli

import (
	appprompttemplates "github.com/TrebuchetDynamics/gormes-agent/internal/app/prompttemplates"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

type PromptTemplateCatalogOptions = appprompttemplates.CatalogOptions

func PromptTemplateCatalog(cfg config.Config, cwd string, opts PromptTemplateCatalogOptions) tui.PromptTemplateCatalog {
	return appprompttemplates.Catalog(cfg, cwd, opts)
}
