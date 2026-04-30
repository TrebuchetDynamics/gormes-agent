package main

import (
	"context"
	"io"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type hermesWebContentProcessor struct {
	client hermes.Client
	model  string
}

func newHermesWebContentProcessor(client hermes.Client, model string) tools.WebContentProcessor {
	if client == nil {
		return nil
	}
	return hermesWebContentProcessor{client: client, model: model}
}

func (p hermesWebContentProcessor) ProcessWebContent(ctx context.Context, req tools.WebContentProcessRequest) (string, error) {
	stream, err := p.client.OpenStream(ctx, hermes.ChatRequest{
		Model:  p.model,
		Stream: true,
		Messages: []hermes.Message{
			{
				Role:    "system",
				Content: "You are an expert content analyst. Produce a concise markdown summary that preserves key facts, figures, quotes, code snippets, and actionable details.",
			},
			{
				Role: "user",
				Content: "Title: " + req.Title + "\nSource: " + req.URL +
					"\n\nCONTENT TO PROCESS:\n" + req.Content +
					"\n\nCreate a well-organized markdown summary. Preserve important specifics and avoid introductions.",
			},
		},
		MaxTokens: 20000,
	})
	if err != nil {
		return "", err
	}
	defer stream.Close() //nolint:errcheck // best-effort close

	var out strings.Builder
	for {
		ev, err := stream.Recv(ctx)
		if err != nil {
			if err == io.EOF && out.Len() > 0 {
				return out.String(), nil
			}
			return "", err
		}
		switch ev.Kind {
		case hermes.EventToken:
			out.WriteString(ev.Token)
		case hermes.EventDone:
			return out.String(), nil
		}
	}
}
