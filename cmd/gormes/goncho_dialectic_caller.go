package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

// HermesDialecticCaller adapts Gormes' native provider client to the Goncho
// DialecticCaller seam. It keeps honcho_reasoning fully in-process: no Python
// hermes-agent or external Honcho runtime is launched for synthesis.
type HermesDialecticCaller struct {
	client hermes.Client
	model  string
}

// NewHermesDialecticCaller returns a DialecticCaller backed by the native
// Hermes-compatible chat client used elsewhere in Gormes.
func NewHermesDialecticCaller(client hermes.Client, model string) *HermesDialecticCaller {
	return &HermesDialecticCaller{client: client, model: strings.TrimSpace(model)}
}

// Chat sends the supplied Goncho context prompt and query through the native
// provider client, collecting streamed text tokens into one synthesized answer.
func (c *HermesDialecticCaller) Chat(ctx context.Context, peer string, systemPrompt string, query string) (string, error) {
	if c == nil || c.client == nil {
		return "", errors.New("goncho: dialectic caller client is nil")
	}
	stream, err := c.client.OpenStream(ctx, hermes.ChatRequest{
		Model:     c.model,
		Stream:    true,
		SessionID: dialecticSessionID(peer),
		Messages: []hermes.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: query},
		},
	})
	if err != nil {
		return "", fmt.Errorf("goncho: dialectic provider stream: %w", err)
	}
	defer stream.Close()

	var answer strings.Builder
	for {
		ev, err := stream.Recv(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", fmt.Errorf("goncho: dialectic provider recv: %w", err)
		}
		if ev.Kind == hermes.EventToken {
			answer.WriteString(ev.Token)
		}
	}
	out := strings.TrimSpace(answer.String())
	if out == "" {
		return "", errors.New("goncho: no dialectic answer from provider")
	}
	return out, nil
}

func dialecticSessionID(peer string) string {
	peer = strings.TrimSpace(peer)
	if peer == "" {
		peer = "unknown"
	}
	return "goncho-dialectic:" + peer
}
