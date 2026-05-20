package kernel

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestKernelEmitsGonchoObservationsForTurnAndToolEvidence(t *testing.T) {
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{{
		Kind:         hermes.EventDone,
		FinishReason: "tool_calls",
		ToolCalls: []hermes.ToolCall{{
			ID:        "call_echo_observe",
			Name:      "echo",
			Arguments: json.RawMessage(`{"text":"observation payload"}`),
		}},
	}}, "sess-observe")

	finalAnswer := "captured observation evidence"
	mc.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: finalAnswer, TokensOut: len(finalAnswer)},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 25, TokensOut: len(finalAnswer)},
	}, "sess-observe")

	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})
	goncho := &recordingGonchoStore{}
	k := New(Config{
		Model:             "hermes-agent",
		Endpoint:          "mock://goncho-observe",
		Admission:         Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             reg,
		MaxToolIterations: 2,
		MaxToolDuration:   time.Second,
		InitialSessionID:  "sess-observe",
		ChatKey:           "telegram:42",
		Goncho:            goncho,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() { _ = k.Run(ctx) }()

	<-k.Render()
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "run the echo tool"}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && lastAssistantMessage(f.History) != nil
	}, 5*time.Second)

	observations := goncho.Observations()
	userPrompt := requireGonchoObservation(t, observations, GonchoObservationUserPrompt)
	if userPrompt.PeerID != "telegram:42" || userPrompt.SessionKey != "sess-observe" || userPrompt.ContextID == "" || userPrompt.Input != "run the echo tool" {
		t.Fatalf("user prompt observation = %#v, want scoped prompt input", userPrompt)
	}

	toolCall := requireGonchoObservation(t, observations, GonchoObservationToolCall)
	if toolCall.SessionKey != "sess-observe" || !strings.Contains(toolCall.Input, "observation payload") || toolCall.Metadata["tool_call_id"] != "call_echo_observe" {
		t.Fatalf("tool call observation = %#v, want scoped tool args and call id", toolCall)
	}

	toolResult := requireGonchoObservation(t, observations, GonchoObservationToolResult)
	if toolResult.Success == nil || !*toolResult.Success || !strings.Contains(toolResult.Output, "observation payload") {
		t.Fatalf("tool result observation = %#v, want successful tool output", toolResult)
	}

	assistant := requireGonchoObservation(t, observations, GonchoObservationAssistantResponse)
	if assistant.SessionKey != "sess-observe" || assistant.ContextID == "" || assistant.Output != finalAnswer {
		t.Fatalf("assistant observation = %#v, want scoped assistant output", assistant)
	}
}

type recordingGonchoStore struct {
	mu           sync.Mutex
	observations []GonchoObservation
}

func (r *recordingGonchoStore) AppendTurn(context.Context, string, string, string, string) error {
	return nil
}

func (r *recordingGonchoStore) GetContext(context.Context, string, int) (string, error) {
	return "", nil
}

func (r *recordingGonchoStore) OnSessionEnd(context.Context, string, []hermes.Message) error {
	return nil
}

func (r *recordingGonchoStore) Observe(_ context.Context, obs GonchoObservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	clone := obs
	if obs.Metadata != nil {
		clone.Metadata = make(map[string]string, len(obs.Metadata))
		for k, v := range obs.Metadata {
			clone.Metadata[k] = v
		}
	}
	r.observations = append(r.observations, clone)
	return nil
}

func (r *recordingGonchoStore) Observations() []GonchoObservation {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]GonchoObservation, len(r.observations))
	copy(out, r.observations)
	return out
}

func requireGonchoObservation(t *testing.T, observations []GonchoObservation, kind GonchoObservationKind) GonchoObservation {
	t.Helper()
	for _, obs := range observations {
		if obs.Kind == kind {
			return obs
		}
	}
	t.Fatalf("missing Goncho observation kind %q in %#v", kind, observations)
	return GonchoObservation{}
}
