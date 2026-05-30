package skills

import (
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/hub"
)

type GitHubRegistryTap = hub.GitHubRegistryTap

type GitHubRegistryProvider = hub.GitHubRegistryProvider

func NewGitHubRegistryProvider(taps []GitHubRegistryTap, client *http.Client) *GitHubRegistryProvider {
	return hub.NewGitHubRegistryProvider(taps, client)
}

type SkillsShRegistryProvider = hub.SkillsShRegistryProvider

func NewSkillsShRegistryProvider(query string, limit int, client *http.Client) *SkillsShRegistryProvider {
	return hub.NewSkillsShRegistryProvider(query, limit, client)
}

type HermesIndexRegistryProvider = hub.HermesIndexRegistryProvider

func NewHermesIndexRegistryProvider(cachePath string) *HermesIndexRegistryProvider {
	return hub.NewHermesIndexRegistryProvider(cachePath)
}

type WellKnownRegistryProvider = hub.WellKnownRegistryProvider

func NewWellKnownRegistryProvider(baseURL string, client *http.Client) *WellKnownRegistryProvider {
	return hub.NewWellKnownRegistryProvider(baseURL, client)
}

type ClawHubRegistryProvider = hub.ClawHubRegistryProvider

func NewClawHubRegistryProvider(baseURL string, client *http.Client) *ClawHubRegistryProvider {
	return hub.NewClawHubRegistryProvider(baseURL, client)
}

type ClaudeMarketplaceRegistryProvider = hub.ClaudeMarketplaceRegistryProvider

func NewClaudeMarketplaceRegistryProvider(query string, limit int, client *http.Client) *ClaudeMarketplaceRegistryProvider {
	return hub.NewClaudeMarketplaceRegistryProvider(query, limit, client)
}

type LobeHubRegistryProvider = hub.LobeHubRegistryProvider

func NewLobeHubRegistryProvider(query string, limit int, client *http.Client) *LobeHubRegistryProvider {
	return hub.NewLobeHubRegistryProvider(query, limit, client)
}
