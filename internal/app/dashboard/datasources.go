package dashboard

import (
	"os"
	"sort"
	"strconv"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config/paths"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/catalog"
)

// knownProviderEnvKeys is the curated list of provider credential variables the
// Env page reports presence for. Only the name, a set/unset flag, and the
// source are surfaced — never the secret value.
var knownProviderEnvKeys = []string{
	"ANTHROPIC_API_KEY",
	"OPENAI_API_KEY",
	"GEMINI_API_KEY",
	"GOOGLE_API_KEY",
	"GROQ_API_KEY",
	"DEEPSEEK_API_KEY",
	"OPENROUTER_API_KEY",
	"XAI_API_KEY",
	"MISTRAL_API_KEY",
	"TELEGRAM_BOT_TOKEN",
	"GORMES_DASHBOARD_API_KEY",
}

// buildEnvStatus reports which known provider credentials are present in the
// environment. It never reads or returns the secret values themselves.
func buildEnvStatus() []apiserver.DashboardEnvKey {
	out := make([]apiserver.DashboardEnvKey, 0, len(knownProviderEnvKeys))
	for _, name := range knownProviderEnvKeys {
		val, ok := os.LookupEnv(name)
		source := "—"
		if ok && val != "" {
			source = "env"
		}
		out = append(out, apiserver.DashboardEnvKey{Name: name, Set: ok && val != "", Source: source})
	}
	return out
}

// buildConfigSummary surfaces a small, deliberately non-secret set of effective
// configuration facts. It degrades to path-only facts if config fails to load.
func buildConfigSummary() []apiserver.DashboardKeyValue {
	out := []apiserver.DashboardKeyValue{
		{Key: "config_path", Value: paths.ConfigPath()},
		{Key: "gormes_home", Value: paths.GormesHome()},
	}
	if cfg, err := config.Load([]string{}); err == nil {
		out = append(out, apiserver.DashboardKeyValue{Key: "config_version", Value: strconv.Itoa(cfg.ConfigVersion)})
	}
	return out
}

// buildSkillsList lists installed skills (name, source, enabled state) using the
// catalog's default runtime and bundled roots.
func buildSkillsList() []apiserver.DashboardSkill {
	rows := catalog.ListInstalledSkills(catalog.ListOptions{}, nil)
	out := make([]apiserver.DashboardSkill, 0, len(rows))
	for _, r := range rows {
		out = append(out, apiserver.DashboardSkill{
			Name:    r.Name,
			Source:  r.Source,
			Enabled: r.Status == catalog.SkillStatusEnabled,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
