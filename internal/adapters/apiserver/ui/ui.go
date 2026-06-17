package ui

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver/ui/chat"
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver/ui/dashboard"
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver/ui/pages"
	"github.com/a-h/templ"
)

// Dashboard returns the dashboard page component at the stable ui package boundary.
func Dashboard() templ.Component { return dashboard.Page() }

// ChatPage returns the chat page component at the stable ui package boundary.
func ChatPage() templ.Component { return chat.Page() }

// SessionsPage returns the sessions page component at the stable ui package boundary.
func SessionsPage() templ.Component { return pages.Sessions() }

// ConfigPage returns the config page component at the stable ui package boundary.
func ConfigPage() templ.Component { return pages.Config() }

// SkillsPage returns the skills page component at the stable ui package boundary.
func SkillsPage() templ.Component { return pages.Skills() }

// CronPage returns the cron page component at the stable ui package boundary.
func CronPage() templ.Component { return pages.Cron() }

// EnvPage returns the environment/keys page component at the stable ui boundary.
func EnvPage() templ.Component { return pages.Env() }

// ModelsPage returns the models page component at the stable ui boundary.
func ModelsPage() templ.Component { return pages.Models() }

// SystemPage returns the system stats page component at the stable ui boundary.
func SystemPage() templ.Component { return pages.System() }

// LogsPage returns the logs page component at the stable ui boundary.
func LogsPage() templ.Component { return pages.Logs() }
