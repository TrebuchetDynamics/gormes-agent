package tools

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
)

type scriptedMOAClient struct {
	mu       sync.Mutex
	scripts  map[string]scriptedMOAResponse
	requests []hermes.ChatRequest
}

type scriptedMOAResponse struct {
	text         string
	reasoning    string
	openErr      error
	finishReason string
}

func (c *scriptedMOAClient) OpenStream(_ context.Context, req hermes.ChatRequest) (hermes.Stream, error) {
	c.mu.Lock()
	c.requests = append(c.requests, req)
	script := c.scripts[req.Model]
	c.mu.Unlock()

	if script.openErr != nil {
		return nil, script.openErr
	}

	finishReason := script.finishReason
	if finishReason == "" {
		finishReason = "stop"
	}

	events := make([]hermes.Event, 0, 3)
	if script.reasoning != "" {
		events = append(events, hermes.Event{Kind: hermes.EventReasoning, Reasoning: script.reasoning})
	}
	if script.text != "" {
		events = append(events, hermes.Event{Kind: hermes.EventToken, Token: script.text})
	}
	events = append(events, hermes.Event{Kind: hermes.EventDone, FinishReason: finishReason})
	return &scriptedMOAStream{events: events, sessionID: req.Model + "-session"}, nil
}

func (*scriptedMOAClient) OpenRunEvents(context.Context, string) (hermes.RunEventStream, error) {
	return nil, hermes.ErrRunEventsNotSupported
}

func (*scriptedMOAClient) Health(context.Context) error { return nil }

func (c *scriptedMOAClient) Requests() []hermes.ChatRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]hermes.ChatRequest, len(c.requests))
	copy(out, c.requests)
	return out
}

type scriptedMOAStream struct {
	mu        sync.Mutex
	events    []hermes.Event
	sessionID string
	pos       int
	closed    bool
}

func (s *scriptedMOAStream) Recv(ctx context.Context) (hermes.Event, error) {
	select {
	case <-ctx.Done():
		return hermes.Event{}, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.pos >= len(s.events) {
		return hermes.Event{}, io.EOF
	}
	ev := s.events[s.pos]
	s.pos++
	return ev, nil
}

func (s *scriptedMOAStream) SessionID() string { return s.sessionID }

func (s *scriptedMOAStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func requestForModel(t *testing.T, reqs []hermes.ChatRequest, model string) hermes.ChatRequest {
	t.Helper()
	for _, req := range reqs {
		if req.Model == model {
			return req
		}
	}
	t.Fatalf("request for model %q not found", model)
	return hermes.ChatRequest{}
}

func TestMixtureOfAgentsExecuteAggregatesReferenceResponsesInDeclaredOrder(t *testing.T) {
	client := &scriptedMOAClient{
		scripts: map[string]scriptedMOAResponse{
			"ref-a": {text: "first draft"},
			"ref-b": {text: "second draft"},
			"agg":   {text: "final synthesis"},
		},
	}
	tool := &MixtureOfAgentsTool{
		Client:          client,
		ReferenceModels: []string{"ref-a", "ref-b"},
		AggregatorModel: "agg",
	}

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"user_prompt":"Solve the problem"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var got struct {
		Success    bool   `json:"success"`
		Response   string `json:"response"`
		ModelsUsed struct {
			ReferenceModels []string `json:"reference_models"`
			AggregatorModel string   `json:"aggregator_model"`
		} `json:"models_used"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output JSON: %v", err)
	}
	if !got.Success {
		t.Fatalf("success = false, output = %s", out)
	}
	if got.Response != "final synthesis" {
		t.Fatalf("response = %q, want %q", got.Response, "final synthesis")
	}
	if got.ModelsUsed.AggregatorModel != "agg" {
		t.Fatalf("aggregator_model = %q, want agg", got.ModelsUsed.AggregatorModel)
	}
	if len(got.ModelsUsed.ReferenceModels) != 2 || got.ModelsUsed.ReferenceModels[0] != "ref-a" || got.ModelsUsed.ReferenceModels[1] != "ref-b" {
		t.Fatalf("reference_models = %#v, want [ref-a ref-b]", got.ModelsUsed.ReferenceModels)
	}

	reqs := client.Requests()
	if len(reqs) != 3 {
		t.Fatalf("request count = %d, want 3", len(reqs))
	}
	aggReq := requestForModel(t, reqs, "agg")
	if len(aggReq.Messages) != 2 {
		t.Fatalf("aggregator message count = %d, want 2", len(aggReq.Messages))
	}
	if aggReq.Messages[0].Role != "system" {
		t.Fatalf("aggregator system role = %q, want system", aggReq.Messages[0].Role)
	}
	if !strings.Contains(aggReq.Messages[0].Content, "1. first draft") || !strings.Contains(aggReq.Messages[0].Content, "2. second draft") {
		t.Fatalf("aggregator system prompt = %q, want ordered reference responses", aggReq.Messages[0].Content)
	}
	if aggReq.Messages[1].Role != "user" || aggReq.Messages[1].Content != "Solve the problem" {
		t.Fatalf("aggregator user message = %+v, want original prompt", aggReq.Messages[1])
	}
}

func TestMixtureOfAgentsExecuteKeepsGoingWhenAReferenceModelFails(t *testing.T) {
	client := &scriptedMOAClient{
		scripts: map[string]scriptedMOAResponse{
			"ref-a": {text: "usable draft"},
			"ref-b": {openErr: errors.New("boom")},
			"agg":   {text: "merged answer"},
		},
	}
	tool := &MixtureOfAgentsTool{
		Client:          client,
		ReferenceModels: []string{"ref-a", "ref-b"},
		AggregatorModel: "agg",
	}

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"user_prompt":"Solve the problem"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var got struct {
		Success      bool     `json:"success"`
		Response     string   `json:"response"`
		FailedModels []string `json:"failed_models"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output JSON: %v", err)
	}
	if !got.Success {
		t.Fatalf("success = false, output = %s", out)
	}
	if got.Response != "merged answer" {
		t.Fatalf("response = %q, want %q", got.Response, "merged answer")
	}
	if len(got.FailedModels) != 1 || got.FailedModels[0] != "ref-b" {
		t.Fatalf("failed_models = %#v, want [ref-b]", got.FailedModels)
	}

	reqs := client.Requests()
	aggReq := requestForModel(t, reqs, "agg")
	if strings.Contains(aggReq.Messages[0].Content, "boom") {
		t.Fatalf("aggregator prompt leaked error text: %q", aggReq.Messages[0].Content)
	}
	if !strings.Contains(aggReq.Messages[0].Content, "usable draft") {
		t.Fatalf("aggregator prompt = %q, want successful draft", aggReq.Messages[0].Content)
	}
}

func TestMixtureOfAgentsExecuteReturnsStructuredFailureWhenAllReferencesFail(t *testing.T) {
	client := &scriptedMOAClient{
		scripts: map[string]scriptedMOAResponse{
			"ref-a": {openErr: errors.New("down")},
			"ref-b": {openErr: errors.New("down")},
		},
	}
	tool := &MixtureOfAgentsTool{
		Client:          client,
		ReferenceModels: []string{"ref-a", "ref-b"},
		AggregatorModel: "agg",
	}

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"user_prompt":"Solve the problem"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var got struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output JSON: %v", err)
	}
	if got.Success {
		t.Fatalf("success = true, want false; output = %s", out)
	}
	if !strings.Contains(got.Error, "insufficient successful reference models") {
		t.Fatalf("error = %q, want insufficient references", got.Error)
	}

	if reqs := client.Requests(); len(reqs) != 2 {
		t.Fatalf("request count = %d, want 2 reference attempts only", len(reqs))
	}
}
