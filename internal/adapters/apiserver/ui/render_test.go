package ui

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestPublicPagesRenderExpectedShellAndContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		component templ.Component
		want      []string
	}{
		{name: "dashboard", component: Dashboard(), want: []string{"<title>Dashboard — Gormes</title>", "🧠 Brain Feed", "🎯 Task Injection"}},
		{name: "chat", component: ChatPage(), want: []string{"<title>Chat — Gormes</title>", "💬 Interactive Chat", "Type your message…"}},
		{name: "sessions", component: SessionsPage(), want: []string{"<title>Sessions — Gormes</title>", "📋 Sessions", "/ui/sessions"}},
		{name: "config", component: ConfigPage(), want: []string{"<title>Config — Gormes</title>", "⚙️ Configuration", "/ui/config"}},
		{name: "skills", component: SkillsPage(), want: []string{"<title>Skills — Gormes</title>", "📦 Skills", "/ui/skills"}},
		{name: "cron", component: CronPage(), want: []string{"<title>Cron — Gormes</title>", "⏰ Cron Jobs", "/ui/cron"}},
		{name: "env", component: EnvPage(), want: []string{"<title>Env — Gormes</title>", "🔑 Environment & Keys", "/ui/env"}},
		{name: "models", component: ModelsPage(), want: []string{"<title>Models — Gormes</title>", "🧠 Models", "/ui/models"}},
		{name: "system", component: SystemPage(), want: []string{"<title>System — Gormes</title>", "🖥️ System", "/ui/system"}},
		{name: "logs", component: LogsPage(), want: []string{"<title>Logs — Gormes</title>", "📜 Logs", "/ui/logs"}},
	}

	// Every page must carry the shared nav so navigation works end to end.
	navLinks := []string{`href="/"`, `href="/chat"`, `href="/sessions"`, `href="/skills"`, `href="/cron"`, `href="/models"`, `href="/config"`, `href="/env"`, `href="/system"`, `href="/logs"`}
	for i := range tests {
		tests[i].want = append(tests[i].want, navLinks...)
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			if err := tt.component.Render(context.Background(), &buf); err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			html := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(html, want) {
					t.Fatalf("rendered HTML missing %q\nHTML:\n%s", want, html)
				}
			}
		})
	}
}
