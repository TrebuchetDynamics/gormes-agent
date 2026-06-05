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
