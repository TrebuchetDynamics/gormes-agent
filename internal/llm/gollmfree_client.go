package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gollmfree"
	"github.com/TrebuchetDynamics/gollmfree/providers"
)

// gollmfreeClient wraps the gollmfree library as a Gormes llm.Client.
// It routes chat completions to free anonymous providers (e.g. PollinationsAI)
// with no API key required.
type gollmfreeClient struct {
	inner *gollmfree.Client
}

// NewGollmfreeClient returns a Client backed by gollmfree's provider pool.
// It registers PollinationsAI as the initial provider (no API key required).
func NewGollmfreeClient() Client {
	pollinations := providers.NewPollinationsAI()
	registry, err := gollmfree.NewRegistry(gollmfree.ProviderInfo{
		Name:            pollinations.Name(),
		Provider:        pollinations,
		SupportedModels: pollinations.SupportedModels(),
		DefaultPriority: 1,
	})
	if err != nil {
		registry = &gollmfree.Registry{}
	}
	return &gollmfreeClient{inner: gollmfree.NewClient(
		gollmfree.WithRegistry(registry),
		gollmfree.WithTimeout(60*time.Second),
		gollmfree.WithMaxRetries(2),
	)}
}

func (c *gollmfreeClient) OpenStream(ctx context.Context, req ChatRequest) (Stream, error) {
	gReq := toGollmfreeRequest(req)
	resp, err := c.inner.ChatCompletion(ctx, gReq)
	if err != nil {
		return nil, fmt.Errorf("gollmfree: %w", err)
	}
	events := gollmfreeResponseToEvents(resp)
	return newStaticProviderStream(events), nil
}

func (c *gollmfreeClient) OpenRunEvents(_ context.Context, _ string) (RunEventStream, error) {
	return nil, fmt.Errorf("gollmfree: run events not supported")
}

func (c *gollmfreeClient) Health(ctx context.Context) error {
	snapshots := c.inner.Health()
	for _, s := range snapshots {
		if s.ConsecutiveFailures == 0 {
			return nil
		}
	}
	if len(snapshots) == 0 {
		return nil
	}
	return fmt.Errorf("gollmfree: all providers reporting failures")
}

func toGollmfreeRequest(req ChatRequest) gollmfree.ChatRequest {
	msgs := make([]gollmfree.Message, 0, len(req.Messages))
	for _, m := range req.Messages {
		content := m.Content
		if content == "" && len(m.ContentParts) > 0 {
			var parts []string
			for _, p := range m.ContentParts {
				if p.Text != "" {
					parts = append(parts, p.Text)
				}
			}
			content = strings.Join(parts, "")
		}
		msgs = append(msgs, gollmfree.Message{
			Role:    m.Role,
			Content: content,
		})
	}
	return gollmfree.ChatRequest{
		Model:    req.Model,
		Messages: msgs,
	}
}

func gollmfreeResponseToEvents(resp gollmfree.CompletionResponse) []Event {
	if len(resp.Choices) == 0 {
		return []Event{{Kind: EventDone}}
	}
	text := resp.Choices[0].Message.Content
	events := make([]Event, 0, 2)
	if text != "" {
		events = append(events, Event{
			Kind:  EventToken,
			Token: text,
		})
	}
	events = append(events, Event{Kind: EventDone})
	return events
}
