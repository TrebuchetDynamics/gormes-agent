package fakegateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type Server struct {
	URL    string
	Client *http.Client

	srv *httptest.Server
	mu  sync.Mutex
	got []Event
}

type Event struct {
	Platform string `json:"platform"`
	ChatID   string `json:"chat_id"`
	Text     string `json:"text"`
}

func New(t testing.TB) *Server {
	t.Helper()
	f := &Server{}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", f.handleHealth)
	mux.HandleFunc("/webhook", f.handleWebhook)
	mux.HandleFunc("/reload", f.handleReload)
	mux.HandleFunc("/logs", f.handleLogs)
	f.srv = httptest.NewServer(mux)
	f.URL = f.srv.URL
	f.Client = f.srv.Client()
	t.Cleanup(f.srv.Close)
	return f
}

func (f *Server) Events() []Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Event, len(f.got))
	copy(out, f.got)
	return out
}

func (f *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"ok": true, "status": "ready"})
}

func (f *Server) handleReload(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"ok": true, "config_reload": "applied"})
}

func (f *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"ok": true, "entries": []string{}})
}

func (f *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	f.mu.Lock()
	f.got = append(f.got, event)
	f.mu.Unlock()
	writeJSON(w, map[string]any{"ok": true, "accepted": true})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
