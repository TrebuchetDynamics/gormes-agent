package apiserver

import (
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver/ui"
	"github.com/a-h/templ"
)

// renderPage writes a full templ page document with the standard HTML headers.
// Page routes are GET-only; anything else gets a 405 so the dashboard never
// silently swallows an unexpected method.
func (s *Server) renderPage(w http.ResponseWriter, r *http.Request, page templ.Component) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = page.Render(r.Context(), w)
}

// handleRootPage serves the dashboard at "/" and 404s every other unmatched
// path (the "/" ServeMux pattern is a catch-all, so it must guard explicitly).
func (s *Server) handleRootPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeOpenAIError(w, http.StatusNotFound, "Not found", "invalid_request_error", "", "not_found")
		return
	}
	s.renderPage(w, r, ui.Dashboard())
}

func (s *Server) handleWebDashboard(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, ui.Dashboard())
}

func (s *Server) handlePageChat(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, ui.ChatPage())
}

func (s *Server) handlePageSessions(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, ui.SessionsPage())
}

func (s *Server) handlePageConfig(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, ui.ConfigPage())
}

func (s *Server) handlePageSkills(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, ui.SkillsPage())
}

func (s *Server) handlePageCron(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, ui.CronPage())
}

func (s *Server) handlePageEnv(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, ui.EnvPage())
}

func (s *Server) handlePageModels(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, ui.ModelsPage())
}

func (s *Server) handlePageSystem(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, ui.SystemPage())
}

func (s *Server) handlePageLogs(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, ui.LogsPage())
}
