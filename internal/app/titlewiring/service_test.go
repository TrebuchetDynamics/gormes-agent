package titlewiring

import (
	"context"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type fakeTitleClient struct {
	req llm.ChatRequest
}

func (f *fakeTitleClient) OpenStream(_ context.Context, req llm.ChatRequest) (llm.Stream, error) {
	f.req = req
	return &fakeTitleStream{events: []llm.Event{{Kind: llm.EventToken, Token: "hello"}, {Kind: llm.EventToken, Token: " world"}, {Kind: llm.EventDone}}}, nil
}

func (f *fakeTitleClient) OpenRunEvents(context.Context, string) (llm.RunEventStream, error) {
	return nil, nil
}

func (f *fakeTitleClient) Health(context.Context) error { return nil }

type fakeTitleStream struct {
	events []llm.Event
	idx    int
}

func (s *fakeTitleStream) Recv(context.Context) (llm.Event, error) {
	ev := s.events[s.idx]
	s.idx++
	return ev, nil
}

func (s *fakeTitleStream) SessionID() string { return "" }

func (s *fakeTitleStream) Close() error { return nil }

func TestBuildTitleModelFuncStreamsTokensAndMapsMessages(t *testing.T) {
	client := &fakeTitleClient{}
	model := BuildTitleModelFunc(client, "title-model")
	got, err := model(context.Background(), llm.TitleModelRequest{Messages: []llm.TitleModelMessage{{Role: "user", Content: "name this"}}})
	if err != nil {
		t.Fatalf("title model: %v", err)
	}
	if got != "hello world" {
		t.Fatalf("title = %q, want hello world", got)
	}
	if client.req.Model != "title-model" || !client.req.Stream || len(client.req.Messages) != 1 || client.req.Messages[0].Content != "name this" {
		t.Fatalf("chat request = %#v", client.req)
	}
}
