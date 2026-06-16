package apiserver

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/internal/httpstream"
)

func (s *Server) handleDashboardSSE(w http.ResponseWriter, r *http.Request) {
	httpstream.SetSSEHeaders(w)
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
		html.EscapeString(truncateStr(s.modelName, 12)),
		html.EscapeString(truncateStr(s.providerName, 12)))
}

func (s *Server) handleDashboardMemoryFragment(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	io.WriteString(w, `<div class="card" style="margin-bottom:6px">
<button hx-get="/dashboard/memory" hx-target="#memory-details" hx-swap="innerHTML">📋 Working Memory</button>
<button hx-get="/dashboard/memory" hx-target="#memory-details" hx-swap="innerHTML" style="margin-left:4px">🧠 Long-term</button>
</div>
<div style="color:var(--dim);font-size:11px">Click a memory node to inspect</div>`)
}

// dashboardChatSessionID keeps dashboard chat turns on one continuous native
// session so the kernel preserves conversation history across messages.
const dashboardChatSessionID = "dashboard-chat"

// dashboardChatTurnTimeout bounds a single dashboard chat turn. It is generous
// because agent turns can run tools, but it prevents an orphaned turn from
// living forever after the page is closed.
const dashboardChatTurnTimeout = 10 * time.Minute

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
	safe := html.EscapeString(prompt)
	s.logStore.Append("chat", "user: "+prompt)
	s.broadcastSSE("frame", fmt.Sprintf(`<div class="line">❯ %s</div>`, safe))

	if s.loop == nil {
		// Display-only degrade: no native turn loop is wired.
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, fmt.Sprintf(`<div class="card" style="border-color:var(--warn)">⚠ Chat is display-only (no turn loop): %s</div>`, safe))
		return
	}

	// Run the turn asynchronously and stream the reply over SSE. The request
	// context ends when this handler returns, so the turn uses an independent
	// bounded context instead.
	go s.runDashboardChatTurn(prompt)
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `<div class="card" style="border-color:var(--ok)">✅ Sent</div>`)
}

// runDashboardChatTurn executes one native turn for the dashboard chat surface
// and broadcasts streamed tokens plus the final reply (or error) to all SSE
// subscribers. Tokens are HTML-escaped before broadcast.
func (s *Server) runDashboardChatTurn(prompt string) {
	ctx, cancel := context.WithTimeout(context.Background(), dashboardChatTurnTimeout)
	defer cancel()

	streamed := false
	res, err := s.loop.StreamTurn(ctx, TurnRequest{
		UserMessage: prompt,
		SessionID:   dashboardChatSessionID,
	}, StreamCallbacks{
		OnToken: func(token string) error {
			if token == "" {
				return nil
			}
			streamed = true
			s.broadcastSSE("frame", fmt.Sprintf(`<span class="stream">%s</span>`, html.EscapeString(token)))
			return nil
		},
	})
	if err != nil {
		s.logStore.Append("chat", "error: "+err.Error())
		s.broadcastSSE("frame", fmt.Sprintf(`<div class="line error">⚠ %s</div>`, html.EscapeString(err.Error())))
		return
	}
	s.logStore.Append("chat", "assistant reply delivered")
	if streamed {
		// Terminate the streamed token line so the next turn starts fresh.
		s.broadcastSSE("frame", `<div class="line"></div>`)
		return
	}
	// Non-streaming loop: emit the complete assistant reply as one frame.
	if reply := strings.TrimSpace(res.Content); reply != "" {
		s.broadcastSSE("frame", fmt.Sprintf(`<div class="line stream">%s</div>`, html.EscapeString(reply)))
	}
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
