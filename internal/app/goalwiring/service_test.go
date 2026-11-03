package goalwiring

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type fakeGoalClient struct {
	req    llm.ChatRequest
	events []llm.Event
	err    error
}

func (f *fakeGoalClient) OpenStream(_ context.Context, req llm.ChatRequest) (llm.Stream, error) {
	f.req = req
	if f.err != nil {
		return nil, f.err
	}
	return &fakeGoalStream{events: f.events}, nil
}

func (f *fakeGoalClient) OpenRunEvents(context.Context, string) (llm.RunEventStream, error) {
	return nil, nil
}

func (f *fakeGoalClient) Health(context.Context) error { return nil }

type fakeGoalStream struct {
	events []llm.Event
	idx    int
}

func (s *fakeGoalStream) Recv(context.Context) (llm.Event, error) {
	event := s.events[s.idx]
	s.idx++
	return event, nil
}

func (s *fakeGoalStream) SessionID() string { return "" }
func (s *fakeGoalStream) Close() error      { return nil }

func TestBuildGoalJudgeFuncStreamsAndParsesDone(t *testing.T) {
	client := &fakeGoalClient{events: []llm.Event{
		{Kind: llm.EventToken, Token: `{"done": true, `},
		{Kind: llm.EventToken, Token: `"reason": "finished"}`},
		{Kind: llm.EventDone},
	}}
	judge := BuildGoalJudgeFunc(client, "judge-model")
	raw, err := judge.JudgeGoal(context.Background(), "ship parity", "the deliverable exists")
	if err != nil {
		t.Fatalf("JudgeGoal: %v", err)
	}
	if raw != `{"done": true, "reason": "finished"}` {
		t.Fatalf("raw = %q", raw)
	}
	if client.req.Model != "judge-model" || !client.req.Stream || len(client.req.Messages) != 2 {
		t.Fatalf("request = %#v", client.req)
	}
	if client.req.Messages[0].Role != "system" || !strings.Contains(client.req.Messages[0].Content, "JSON") {
		t.Fatalf("system prompt = %#v", client.req.Messages[0])
	}
	if !strings.Contains(client.req.Messages[1].Content, "ship parity") || !strings.Contains(client.req.Messages[1].Content, "the deliverable exists") {
		t.Fatalf("user prompt = %q", client.req.Messages[1].Content)
	}
	verdict := gateway.ParseGoalJudgeResponse(raw)
	if verdict.Verdict != gateway.GoalJudgeVerdictDone {
		t.Fatalf("verdict = %+v", verdict)
	}
}

func TestBuildGoalJudgeFuncNilClientFailsOpenEvidence(t *testing.T) {
	judge := BuildGoalJudgeFunc(nil, "judge-model")
	if _, err := judge.JudgeGoal(context.Background(), "goal", "response"); err == nil {
		t.Fatal("nil client judge returned no error")
	}
}

func TestBuildJudgeUserPromptBoundsResponse(t *testing.T) {
	long := strings.Repeat("x", maxJudgeResponseChars+500)
	prompt := BuildJudgeUserPrompt("goal", long)
	if !strings.Contains(prompt, "Goal:\ngoal") {
		t.Fatalf("prompt missing goal: %q", prompt[:80])
	}
	if strings.Contains(prompt, strings.Repeat("x", maxJudgeResponseChars+1)) {
		t.Fatal("prompt did not truncate the response")
	}
	if !strings.HasSuffix(prompt, long[len(long)-100:]) {
		t.Fatal("prompt did not keep the response tail")
	}
}
