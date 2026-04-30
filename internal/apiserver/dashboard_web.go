package apiserver

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed templates/*
var dashboardTemplates embed.FS

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"safeHTML": func(s string) template.HTML { return template.HTML(s) },
}).ParseFS(dashboardTemplates, "templates/*.html"))

func (s *Server) registerDashboardWeb(mux *http.ServeMux) {
	mux.HandleFunc("/{$}", s.handleDashboardIndex)
	mux.HandleFunc("/sessions", s.handleWebSessions)
	mux.HandleFunc("/chat", s.handleWebChat)
	mux.HandleFunc("/config", s.handleWebConfig)
	mux.HandleFunc("/cron", s.handleWebCron)
	mux.HandleFunc("/skills", s.handleWebSkills)
	mux.HandleFunc("/logs", s.handleWebLogs)
}

func (s *Server) handleDashboardIndex(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Title":   "Gormes Dashboard",
		"Content": "index",
		"Pages":   dashboardPages(),
	})
}

func (s *Server) handleWebSessions(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Title":   "Sessions — Gormes",
		"Content": "sessions",
		"Pages":   dashboardPages(),
	})
}

func (s *Server) handleWebChat(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "layout.html", map[string]any{"Title": "Chat — Gormes", "Content": "chat", "Pages": dashboardPages()})
}

func (s *Server) handleWebConfig(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "layout.html", map[string]any{"Title": "Config — Gormes", "Content": "config", "Pages": dashboardPages()})
}

func (s *Server) handleWebCron(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "layout.html", map[string]any{"Title": "Cron — Gormes", "Content": "cron", "Pages": dashboardPages()})
}

func (s *Server) handleWebSkills(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "layout.html", map[string]any{"Title": "Skills — Gormes", "Content": "skills", "Pages": dashboardPages()})
}

func (s *Server) handleWebLogs(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "layout.html", map[string]any{"Title": "Logs — Gormes", "Content": "logs", "Pages": dashboardPages()})
}

type dashboardPage struct {
	Name string
	Path string
	Icon string
}

func dashboardPages() []dashboardPage {
	return []dashboardPage{
		{"Chat", "/chat", "💬"},
		{"Sessions", "/sessions", "📋"},
		{"Cron", "/cron", "⏰"},
		{"Skills", "/skills", "📦"},
		{"Config", "/config", "⚙️"},
		{"Logs", "/logs", "📄"},
	}
}
