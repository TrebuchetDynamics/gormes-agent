package prompttemplates

import (
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

type CatalogOptions struct {
	Paths    []string
	Disabled bool
}

func Catalog(cfg config.Config, cwd string, opts CatalogOptions) tui.PromptTemplateCatalog {
	_ = cfg // Reserved for future profile-scoped prompt-template roots.
	if opts.Disabled {
		return tui.PromptTemplateCatalog{}
	}
	if cwd == "" {
		if got, err := os.Getwd(); err == nil {
			cwd = got
		}
	}
	roots := []string{filepath.Join(config.GormesHome(), "prompts")}
	if cwd != "" {
		roots = append(roots, filepath.Join(cwd, ".gormes", "prompts"))
	}
	roots = append(roots, opts.Paths...)
	return tui.PromptTemplateCatalogFromRoots(roots)
}
