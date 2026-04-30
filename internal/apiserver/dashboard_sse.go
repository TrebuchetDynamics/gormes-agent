package apiserver

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (s *Server) handleDashboardSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	ch := make(chan string, 32)
	s.registerSSEClient(ch)
	defer s.unregisterSSEClient(ch)

	writeSSEEvent(w, "connected", map[string]string{"status": "ok"})
	flusher.Flush()

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) registerSSEClient(ch chan string) {
	s.sseMu.Lock()
	defer s.sseMu.Unlock()
	s.sseClients = append(s.sseClients, ch)
}

func (s *Server) unregisterSSEClient(ch chan string) {
	s.sseMu.Lock()
	defer s.sseMu.Unlock()
	for i, c := range s.sseClients {
		if c == ch {
			s.sseClients = append(s.sseClients[:i], s.sseClients[i+1:]...)
			close(ch)
			return
		}
	}
}

func (s *Server) broadcastSSE(event, data string) {
	s.sseMu.Lock()
	defer s.sseMu.Unlock()
	for _, ch := range s.sseClients {
		select {
		case ch <- data:
		default:
		}
	}
}

func (s *Server) handleDashboardStatusFragment(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="status-box"><div class="val">⚕</div><div class="lbl">Running</div></div>
<div class="status-box"><div class="val">%s</div><div class="lbl">Model</div></div>
<div class="status-box"><div class="val">%s</div><div class="lbl">Session</div></div>`,
		truncateStr(s.modelName, 12),
		truncateStr(s.providerName, 12))
}

func (s *Server) handleDashboardMemoryFragment(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	io.WriteString(w, `<div class="card" style="margin-bottom:6px">
<button hx-get="/dashboard/memory" hx-target="#memory-details" hx-swap="innerHTML">📋 Working Memory</button>
<button hx-get="/dashboard/memory" hx-target="#memory-details" hx-swap="innerHTML" style="margin-left:4px">🧠 Long-term</button>
</div>
<div style="color:var(--dim);font-size:11px">Click a memory node to inspect</div>`)
}

func (s *Server) handleAgentExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	prompt := strings.TrimSpace(r.FormValue("prompt"))
	if prompt == "" {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `<div class="card" style="border-color:var(--err)">Error: prompt is required</div>`)
		return
	}
	s.broadcastSSE("frame", fmt.Sprintf(`<div class="line">❯ %s</div>`, prompt))
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, fmt.Sprintf(`<div class="card" style="border-color:var(--ok)">✅ Task injected: %s</div>`, prompt))
}

func truncateStr(s string, max int) string {
	if len(s) > max {
		return s[:max-1] + "…"
	}
	if s == "" {
		return "—"
	}
	return s
}
