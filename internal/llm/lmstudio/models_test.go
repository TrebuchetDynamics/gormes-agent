package lmstudio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ModelsResponse{
			Object: "list",
			Data: []ModelInfo{
				{ID: "llama-3-8b", Object: "model", Created: 1700000000, OwnedBy: "lmstudio"},
				{ID: "qwen-7b", Object: "model", Created: 1700000001, OwnedBy: "lmstudio"},
			},
		})
	}))
	defer server.Close()

	models, err := ListModels(context.Background(), server.URL+"/v1")
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if len(models) != 2 || models[0].ID != "llama-3-8b" || models[1].ID != "qwen-7b" {
		t.Fatalf("models = %+v, want fixture models", models)
	}
}

func TestListModelsStatusError(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	if _, err := ListModels(context.Background(), server.URL+"/v1"); err == nil {
		t.Fatal("ListModels() error = nil, want status error")
	}
}
