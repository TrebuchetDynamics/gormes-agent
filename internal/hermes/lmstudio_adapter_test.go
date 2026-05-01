package hermes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLMStudioAdapter_StatusUnreachable(t *testing.T) {
	adapter := NewLMStudioAdapter("http://localhost:59999/v1")
	status := adapter.Status()
	if status.Provider != "lmstudio" {
		t.Fatalf("expected provider=lmstudio, got %s", status.Provider)
	}
	if status.Capabilities.PromptCache.Available {
		t.Fatal("expected prompt cache unavailable for unreachable server")
	}
	if status.Capabilities.PromptCache.Reason != "lmstudio_unreachable" {
		t.Fatalf("expected reason=lmstudio_unreachable, got %s", status.Capabilities.PromptCache.Reason)
	}
}

func TestLMStudioAdapter_StatusReachable(t *testing.T) {
	server := newLMStudioFakeServer(t)
	defer server.Close()

	adapter := NewLMStudioAdapter(server.URL + "/v1")
	status := adapter.Status()
	if status.Provider != "lmstudio" {
		t.Fatalf("expected provider=lmstudio, got %s", status.Provider)
	}
	if status.Runtime != "chat_completions" {
		t.Fatalf("expected runtime=chat_completions, got %s", status.Runtime)
	}
	if status.Capabilities.PromptCache.Available {
		t.Fatal("expected prompt cache unavailable for openai-compatible local server")
	}
}

func TestLMStudioAdapter_ListModels(t *testing.T) {
	server := newLMStudioFakeServer(t)
	defer server.Close()

	adapter := NewLMStudioAdapter(server.URL + "/v1")
	ctx := context.Background()
	models, err := adapter.ListModels(ctx)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "llama-3-8b" {
		t.Fatalf("expected model id=llama-3-8b, got %s", models[0].ID)
	}
	if models[1].ID != "qwen-7b" {
		t.Fatalf("expected model id=qwen-7b, got %s", models[1].ID)
	}
}

func TestLMStudioAdapter_ListModelsUnreachable(t *testing.T) {
	adapter := NewLMStudioAdapter("http://localhost:59999/v1")
	ctx := context.Background()
	_, err := adapter.ListModels(ctx)
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
}

func TestLMStudioAdapter_DefaultBaseURL(t *testing.T) {
	adapter := NewLMStudioAdapter("")
	if adapter.baseURL != defaultLMStudioBaseURL {
		t.Fatalf("expected default base URL %s, got %s", defaultLMStudioBaseURL, adapter.baseURL)
	}
}

func newLMStudioFakeServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(ModelsResponse{
				Object: "list",
				Data: []ModelInfo{
					{ID: "llama-3-8b", Object: "model", Created: 1700000000, OwnedBy: "lmstudio"},
					{ID: "qwen-7b", Object: "model", Created: 1700000001, OwnedBy: "lmstudio"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}
