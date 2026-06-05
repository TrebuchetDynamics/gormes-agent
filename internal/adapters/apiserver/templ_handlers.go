package apiserver

import (
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver/ui"
)

func (s *Server) handleTemplDashboard(w http.ResponseWriter, r *http.Request) {
	ui.Dashboard().Render(r.Context(), w)
}

func (s *Server) handleTemplChat(w http.ResponseWriter, r *http.Request) {
	ui.ChatPage().Render(r.Context(), w)
}

func (s *Server) handleTemplSessions(w http.ResponseWriter, r *http.Request) {
	ui.SessionsPage().Render(r.Context(), w)
}

func (s *Server) handleTemplConfig(w http.ResponseWriter, r *http.Request) {
	ui.ConfigPage().Render(r.Context(), w)
}

func (s *Server) handleTemplSkills(w http.ResponseWriter, r *http.Request) {
	ui.SkillsPage().Render(r.Context(), w)
}

func (s *Server) handleTemplCron(w http.ResponseWriter, r *http.Request) {
	ui.CronPage().Render(r.Context(), w)
}
