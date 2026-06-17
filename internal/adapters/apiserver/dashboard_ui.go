package apiserver

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver/ui/fragments"
	"github.com/a-h/templ"
)

// dashboardUIAllowed gates the HTML fragment endpoints. A loopback-bound
// dashboard (the `gormes dashboard` default) serves same-origin UI fragments
// without the programmatic bearer key, because the browser cannot attach it.
// A network-exposed dashboard must still authenticate, so fragment data is not
// leaked to anyone who can reach the host.
func (s *Server) dashboardUIAllowed(r *http.Request) bool {
	host := hostNameOnly(s.dashboardBoundHost)
	if host == "" || isLoopbackHost(host) {
		return true
	}
	return s.dashboardAuthorized(r)
}

// renderFragment writes an HTML partial for an htmx swap.
func (s *Server) renderFragment(w http.ResponseWriter, r *http.Request, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = c.Render(r.Context(), w)
}

// guardFragment enforces GET + the UI auth gate, returning false when the
// caller has already been answered with an error fragment.
func (s *Server) guardFragment(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		s.renderFragment(w, r, fragments.Notice("err", "Method not allowed"))
		return false
	}
	if !s.dashboardUIAllowed(r) {
		w.WriteHeader(http.StatusUnauthorized)
		s.renderFragment(w, r, fragments.Notice("err", "Unauthorized — dashboard is network-exposed and requires authentication"))
		return false
	}
	return true
}

func (s *Server) handleSessionsFragment(w http.ResponseWriter, r *http.Request) {
	if !s.guardFragment(w, r) {
		return
	}
	// Prefer the persistent session directory (memory.db) so the page shows the
	// same sessions as `gormes session list` and they survive restarts.
	if s.sessionsList != nil {
		sessions := s.sessionsList()
		rows := make([][]string, 0, len(sessions))
		for _, sess := range sessions {
			rows = append(rows, []string{
				truncateStr(sess.ID, 18),
				orDash(sess.Source),
				strconv.Itoa(sess.MessageCount),
				orDash(strings.TrimSpace(sess.Preview)),
			})
		}
		s.renderFragment(w, r, fragments.Table(
			[]string{"Session", "Source", "Messages", "Preview"},
			rows,
			"No sessions yet.",
		))
		return
	}

	// Fallback: in-memory response store (programmatic API sessions).
	sessions, _, err := s.responseStore.ListSessions(dashboardMaxSessionLimit, 0, s.now())
	if err != nil {
		s.renderFragment(w, r, fragments.Notice("err", "Session store error: "+err.Error()))
		return
	}
	rows := make([][]string, 0, len(sessions))
	for _, sess := range sessions {
		active := "—"
		if sess.IsActive {
			active = "active"
		}
		rows = append(rows, []string{
			sess.ID,
			derefString(sess.Model, "—"),
			strconv.Itoa(sess.MessageCount),
			strconv.Itoa(sess.ToolCallCount),
			active,
		})
	}
	s.renderFragment(w, r, fragments.Table(
		[]string{"Session", "Model", "Messages", "Tools", "State"},
		rows,
		"No sessions yet.",
	))
}

func (s *Server) handleConfigFragment(w http.ResponseWriter, r *http.Request) {
	if !s.guardFragment(w, r) {
		return
	}
	if s.configSummary == nil {
		s.renderFragment(w, r, fragments.Notice("dim", "Config summary is not wired in this dashboard process."))
		return
	}
	entries := s.configSummary()
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, []string{e.Key, e.Value})
	}
	s.renderFragment(w, r, fragments.Table([]string{"Setting", "Value"}, rows, "No configuration to show."))
}

func (s *Server) handleSkillsFragment(w http.ResponseWriter, r *http.Request) {
	if !s.guardFragment(w, r) {
		return
	}
	if s.skillsList == nil {
		s.renderFragment(w, r, fragments.Notice("dim", "Skill listing is not wired in this dashboard process."))
		return
	}
	skills := s.skillsList()
	rows := make([][]string, 0, len(skills))
	for _, sk := range skills {
		enabled := "disabled"
		if sk.Enabled {
			enabled = "enabled"
		}
		rows = append(rows, []string{sk.Name, sk.Source, enabled})
	}
	s.renderFragment(w, r, fragments.Table([]string{"Skill", "Source", "State"}, rows, "No skills installed."))
}

func (s *Server) handleCronFragment(w http.ResponseWriter, r *http.Request) {
	if !s.guardFragment(w, r) {
		return
	}
	if s.cronJobs == nil {
		s.renderFragment(w, r, fragments.Notice("dim", "Cron store is not wired in this dashboard process."))
		return
	}
	jobs, err := s.cronJobs.List()
	if err != nil {
		s.renderFragment(w, r, fragments.Notice("err", "Cron store error: "+err.Error()))
		return
	}
	rows := make([][]string, 0, len(jobs))
	for _, j := range jobs {
		state := "active"
		if j.Paused {
			state = "paused"
		}
		last := j.LastStatus
		if last == "" {
			last = "—"
		}
		rows = append(rows, []string{j.Name, j.Schedule, state, last})
	}
	s.renderFragment(w, r, fragments.Table([]string{"Job", "Schedule", "State", "Last run"}, rows, "No cron jobs."))
}

func (s *Server) handleEnvFragment(w http.ResponseWriter, r *http.Request) {
	if !s.guardFragment(w, r) {
		return
	}
	if s.envStatus == nil {
		s.renderFragment(w, r, fragments.Notice("dim", "Environment status is not wired in this dashboard process."))
		return
	}
	keys := s.envStatus()
	rows := make([][]string, 0, len(keys))
	for _, k := range keys {
		set := "— missing"
		if k.Set {
			set = "✓ set"
		}
		rows = append(rows, []string{k.Name, set, k.Source})
	}
	s.renderFragment(w, r, fragments.Table([]string{"Key", "Status", "Source"}, rows, "No credential keys known."))
}

func (s *Server) handleModelsFragment(w http.ResponseWriter, r *http.Request) {
	if !s.guardFragment(w, r) {
		return
	}
	providers := s.dashboardModelProviders()
	rows := make([][]string, 0, len(providers))
	for _, p := range providers {
		current := "—"
		if p.IsCurrent {
			current = "✓ current"
		}
		rows = append(rows, []string{p.Name, p.Slug, strconv.Itoa(p.TotalModels), current})
	}
	s.renderFragment(w, r, fragments.Table([]string{"Provider", "Slug", "Models", "Active"}, rows, "No model providers configured."))
}

func (s *Server) handleSystemFragment(w http.ResponseWriter, r *http.Request) {
	if !s.guardFragment(w, r) {
		return
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	rows := [][]string{
		{"Version", orDash(s.buildInfo.Version)},
		{"Commit", orDash(s.buildInfo.GitCommit)},
		{"Go", orDash(s.buildInfo.GoVersion, runtime.Version())},
		{"Platform", runtime.GOOS + "/" + runtime.GOARCH},
		{"CPUs", strconv.Itoa(runtime.NumCPU())},
		{"Goroutines", strconv.Itoa(runtime.NumGoroutine())},
		{"Heap in use", fmt.Sprintf("%d MiB", mem.HeapInuse/1024/1024)},
		{"Total alloc", fmt.Sprintf("%d MiB", mem.TotalAlloc/1024/1024)},
		{"Model", orDash(s.modelName)},
		{"Provider", orDash(s.providerName)},
	}
	s.renderFragment(w, r, fragments.Table([]string{"Metric", "Value"}, rows, "No system stats."))
}

func (s *Server) handleLogsFragment(w http.ResponseWriter, r *http.Request) {
	if !s.guardFragment(w, r) {
		return
	}
	entries := s.logStore.Recent()
	lines := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Level != "" {
			lines = append(lines, e.Level+": "+e.Message)
			continue
		}
		lines = append(lines, e.Message)
	}
	s.renderFragment(w, r, fragments.LogLines(lines, "No log entries yet."))
}

// handleDashboardLogs is the JSON counterpart of the logs fragment, used by
// programmatic clients. It returns the retained in-memory log ring.
func (s *Server) handleDashboardLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
		return
	}
	if !s.dashboardAuthorized(r) {
		writeDashboardUnauthorized(w)
		return
	}
	entries := s.logStore.Recent()
	if entries == nil {
		entries = nil
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"build": s.buildInfo,
		"logs":  entries,
		"total": len(entries),
	})
}

func derefString(p *string, fallback string) string {
	if p == nil || *p == "" {
		return fallback
	}
	return *p
}

func orDash(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return "—"
}
