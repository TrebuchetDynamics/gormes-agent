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

// sseClientBufferSize bounds each SSE connection's pending-frame queue. It is
// sized for bursty token-by-token chat streaming so deltas are not dropped by
// the non-blocking broadcast.
const sseClientBufferSize = 256

func (s *Server) handleDashboardSSE(w http.ResponseWriter, r *http.Request) {
	httpstream.SetSSEHeaders(w)
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	// Buffered generously so a burst of streamed chat token frames is not
	// dropped by the non-blocking broadcast before the writer drains them.
	ch := make(chan string, sseClientBufferSize)
	s.registerSSEClient(ch)
	defer s.unregisterSSEClient(ch)

	writeSSEEvent(w, "connected", map[string]string{"status": "ok"})
	// Push an immediate status frame so the nav indicator goes live on connect
	// instead of waiting a full tick.
	io.WriteString(w, sseFrame("status", s.statusFrameHTML()))
	flusher.Flush()

	ticker := time.NewTicker(dashboardStatusInterval)
	defer ticker.Stop()

	for {
		select {
		case msg := <-ch:
			// msg is already a fully-formatted SSE frame (event + data).
			io.WriteString(w, msg)
			flusher.Flush()
		case <-ticker.C:
			io.WriteString(w, sseFrame("status", s.statusFrameHTML()))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// dashboardStatusInterval is how often each connected dashboard receives a
// refreshed nav status frame over SSE.
const dashboardStatusInterval = 5 * time.Second

// statusFrameHTML renders the compact nav status indicator swapped into
// #nav-status (sse-swap="status"). It replaces the static "Connecting…" text
// once the SSE stream is live.
func (s *Server) statusFrameHTML() string {
	return fmt.Sprintf(`<span style="color:var(--ok)">● live</span> %s`, html.EscapeString(truncateStr(s.modelName, 14)))
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
	frame := sseFrame(event, data)
	s.sseMu.Lock()
	defer s.sseMu.Unlock()
	for _, ch := range s.sseClients {
		select {
		case ch <- frame:
		default:
		}
	}
}

// sseFrame formats a named Server-Sent Event. The event name must be set so
// htmx's sse extension (sse-swap="<name>") matches it; an unnamed event would
// arrive as the default "message" type and never trigger the swap. Multi-line
// data is split into one `data:` field per line per the SSE spec.
func sseFrame(event, data string) string {
	var b strings.Builder
	if event != "" {
		b.WriteString("event: ")
		b.WriteString(event)
		b.WriteByte('\n')
	}
	for _, line := range strings.Split(data, "\n") {
		b.WriteString("data: ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	return b.String()
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

// dashboardChatSessionID is the default dashboard chat session id. Turns share
// it so the kernel preserves conversation history across messages; "new chat"
// rotates to a fresh id so the next conversation persists as its own session.
const dashboardChatSessionID = "dashboard-chat"

// SessionResetter is implemented by turn loops that can clear conversation
// state (history + session id). *KernelTurnLoop satisfies it; non-kernel or
// fake loops may not, in which case "new chat" still clears the visible feed
// and rotates the session id.
type SessionResetter interface {
	ResetSession() error
}

func (s *Server) currentChatSessionID() string {
	s.chatMu.Lock()
	defer s.chatMu.Unlock()
	if s.chatSessionID == "" {
		s.chatSessionID = dashboardChatSessionID
	}
	return s.chatSessionID
}

// startNewChat rotates the dashboard chat session id so subsequent turns
// persist under a fresh session distinct from the previous conversation.
func (s *Server) startNewChat() {
	s.chatMu.Lock()
	defer s.chatMu.Unlock()
	s.chatSessionID = "dashboard-chat-" + randomHexFromTime(s.now())[:12]
}

// handleDashboardNewChat resets the agent conversation (clearing kernel
// context when a resettable loop is wired) and rotates the chat session id,
// returning a fresh feed body that replaces the visible conversation.
func (s *Server) handleDashboardNewChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, `<div class="line error">Method not allowed</div>`)
		return
	}
	if !s.dashboardUIAllowed(r) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `<div class="line error">Unauthorized</div>`)
		return
	}
	if s.loop != nil {
		if resetter, ok := s.loop.(SessionResetter); ok {
			if err := resetter.ResetSession(); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				io.WriteString(w, fmt.Sprintf(`<div class="line error">⚠ Could not start a new chat: %s</div>`, html.EscapeString(err.Error())))
				return
			}
		}
	}
	s.startNewChat()
	s.logStore.Append("chat", "new chat started")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `<div class="line thinking">🆕 Started a new chat.</div>`)
}

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

	streamStarted := false
	res, err := s.loop.StreamTurn(ctx, TurnRequest{
		UserMessage: prompt,
		SessionID:   s.currentChatSessionID(),
	}, StreamCallbacks{
		OnToken: func(token string) error {
			if token == "" {
				return nil
			}
			if !streamStarted {
				streamStarted = true
				// Open the assistant reply inline so subsequent token deltas
				// flow into a single growing message after this marker.
				s.broadcastSSE("frame", `<span class="stream reply-marker">🤖 </span>`)
			}
			// Each token delta is its own SSE frame appended to the feed
			// (hx-swap="beforeend"), so the reply renders token by token.
			s.broadcastSSE("frame", fmt.Sprintf(`<span class="stream">%s</span>`, html.EscapeString(token)))
			return nil
		},
	})
	if err != nil {
		s.logStore.Append("chat", "error: "+err.Error())
		if streamStarted {
			// Close the partially streamed line before the error.
			s.broadcastSSE("frame", `<div class="line"></div>`)
		}
		s.broadcastSSE("frame", fmt.Sprintf(`<div class="line error">⚠ %s</div>`, html.EscapeString(err.Error())))
		return
	}
	s.logStore.Append("chat", "assistant reply delivered")
	if streamStarted {
		// Terminate the streamed reply line so the next turn starts fresh.
		s.broadcastSSE("frame", `<div class="line"></div>`)
		return
	}
	// Non-streaming loop: emit the complete assistant reply as one frame.
	if reply := strings.TrimSpace(res.Content); reply != "" {
		s.broadcastSSE("frame", fmt.Sprintf(`<div class="line stream">🤖 %s</div>`, html.EscapeString(reply)))
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
