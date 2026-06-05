package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"

// PromptTemplateCatalog is the TUI-facing catalog type for local Markdown
// prompt-template slash expansions.
type PromptTemplateCatalog = prompttemplates.Catalog

// PromptTemplateCatalogFromRoots discovers prompt templates for the native TUI
// while reserving the built-in slash command namespace.
func PromptTemplateCatalogFromRoots(roots []string) PromptTemplateCatalog {
	catalog, _ := prompttemplates.Discover(prompttemplates.DiscoverOptions{
		Roots:         roots,
		ReservedNames: DefaultSlashCommandNames(),
	})
	return catalog
}
