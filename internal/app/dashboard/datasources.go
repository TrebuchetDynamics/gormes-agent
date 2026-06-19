package dashboard

import (
	"context"
	"os"
	"sort"
	"strconv"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver"
	cronapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/cron"
	appsession "github.com/TrebuchetDynamics/gormes-agent/internal/app/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config/paths"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/catalog"
	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/transcript"
)

// dashboardSessionListLimit caps how many recent sessions the dashboard lists.
const dashboardSessionListLimit = 50

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

// newSessionReader opens the persistent session store (memory.db) once and
// returns a sessions lister (for the Sessions page) and a chat-history loader
// (for restoring the chat feed on load), plus a closer. SQLite tolerates
// concurrent processes, so this works alongside a running gateway. Returns
// ok=false when the store cannot be opened (e.g. no sessions yet).
func newSessionReader() (list func() []apiserver.DashboardSession, history func(string) []apiserver.DashboardChatMessage, closer func(), ok bool) {
	db, err := appsession.OpenSessionDirectoryDB()
	if err != nil {
		return nil, nil, nil, false
	}
	list = func() []apiserver.DashboardSession {
		entries, err := sessionpkg.ListDirectorySessions(context.Background(), db, sessionpkg.DirectoryFilter{Limit: dashboardSessionListLimit})
		if err != nil {
			return nil
		}
		out := make([]apiserver.DashboardSession, 0, len(entries))
		for _, e := range entries {
			out = append(out, apiserver.DashboardSession{
				ID:             e.ID,
				Title:          e.Title,
				Preview:        e.Preview,
				Source:         e.Source,
				MessageCount:   e.MessageCount,
				LastActiveUnix: e.LastActiveAt,
			})
		}
		return out
	}
	history = func(sessionID string) []apiserver.DashboardChatMessage {
		msgs, err := transcript.LoadMessages(context.Background(), db, sessionID)
		if err != nil {
			return nil
		}
		out := make([]apiserver.DashboardChatMessage, 0, len(msgs))
		for _, m := range msgs {
			out = append(out, apiserver.DashboardChatMessage{Role: m.Role, Content: m.Content})
		}
		return out
	}
	return list, history, func() { _ = db.Close() }, true
}

// newCronReader opens the cron job store (backed by the bbolt session DB) and
// returns it as a read facade plus a closer. bbolt takes an exclusive file
// lock, so this returns ok=false when another process (e.g. the gateway) holds
// sessions.db — the cron panel then degrades to "not wired".
func newCronReader() (apiserver.CronJobReader, func(), bool) {
	store, smap, err := cronapp.OpenStore("")
	if err != nil {
		return nil, nil, false
	}
	return store, func() { _ = smap.Close() }, true
}
