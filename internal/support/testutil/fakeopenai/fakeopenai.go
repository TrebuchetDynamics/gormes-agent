package fakeopenai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type Server struct {
	URL    string
	Client *http.Client

	srv *httptest.Server

	mu       sync.Mutex
	requests []ChatRequest
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func New(t testing.TB) *Server {
	t.Helper()
	f := &Server{}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", f.handleModels)
	mux.HandleFunc("/v1/chat/completions", f.handleChatCompletions)
	f.srv = httptest.NewServer(mux)
	f.URL = f.srv.URL + "/v1"
	f.Client = f.srv.Client()
	t.Cleanup(f.srv.Close)
	return f
}

func (f *Server) Requests() []ChatRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]ChatRequest, len(f.requests))
	copy(out, f.requests)
	return out
}

func (f *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]any{
		"object": "list",
		"data":   []map[string]string{{"id": "fake-gormes-ci", "object": "model"}},
	})
}

func (f *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()

	if req.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"fake \"}}]}\n\n")
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"provider answer\"}}],\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":3,\"total_tokens\":10}}\n\n")
		fmt.Fprintf(w, "data: [DONE]\n\n")
		return
	}

	writeJSON(w, map[string]any{
		"id":     "chatcmpl-fake-gormes-ci",
		"object": "chat.completion",
		"choices": []map[string]any{{
			"index":         0,
			"finish_reason": "stop",
			"message": map[string]string{
				"role":    "assistant",
				"content": "fake provider answer",
			},
		}},
		"usage": map[string]int{"prompt_tokens": 7, "completion_tokens": 3, "total_tokens": 10},
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
