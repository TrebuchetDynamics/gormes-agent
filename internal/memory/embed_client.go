package memory

import "github.com/TrebuchetDynamics/gormes-agent/internal/memory/embeddings"

// errEmbedModelNotFound is returned when the Ollama endpoint reports 404
// with a "model not found" body. Callers (the Embedder worker) handle
// this by logging a one-per-minute WARN and waiting; it's not a crash.
var errEmbedModelNotFound = embeddings.ErrModelNotFound

// EmbedClient is the cmd-visible alias for the embeddings client.
type EmbedClient = embeddings.Client

type embedClient = embeddings.Client

func newEmbedClient(baseURL, apiKey string) *embedClient {
	return embeddings.NewClient(baseURL, apiKey)
}

// NewEmbedClient constructs an EmbedClient for the given base URL
// (e.g. "http://localhost:11434") and optional API key.
func NewEmbedClient(baseURL, apiKey string) *EmbedClient {
	return newEmbedClient(baseURL, apiKey)
}
