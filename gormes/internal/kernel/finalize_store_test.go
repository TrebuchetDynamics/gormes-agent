package kernel

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
)

// TestKernel_FinalizeAssistantTurnReachesStore proves the kernel fires
// both AppendUserTurn (pre-stream) and FinalizeAssistantTurn (post-stream)
// on every successful turn, with matching session_id and content.
func TestKernel_FinalizeAssistantTurnReachesStore(t *testing.T) {
	rec := store.NewRecording()

	mc := hermes.NewMockClient()
	reply := "hello back"
	events := make([]hermes.Event, 0, len(reply)+1)
	for _, ch := range reply {
		events = append(events, hermes.Event{Kind: hermes.EventToken, Token: string(ch), TokensOut: 1})
	}
	events = append(events, hermes.Event{Kind: hermes.EventDone, FinishReason: "stop"})
	mc.Script(events, "sess-finalize-test")

	k := New(Config{
		Model:     "hermes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, mc, rec, telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"}); err != nil {
		t.Fatal(err)
	}

	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID != ""
	}, 3*time.Second)

	cmds := rec.Commands()
	if len(cmds) < 2 {
		t.Fatalf("len(cmds) = %d, want >= 2 (AppendUserTurn + FinalizeAssistantTurn)", len(cmds))
	}

	// First command must be AppendUserTurn with the user's text.
	if cmds[0].Kind != store.AppendUserTurn {
		t.Errorf("cmds[0].Kind = %v, want AppendUserTurn", cmds[0].Kind)
	}
	var p1 struct {
		SessionID string `json:"session_id"`
		Content   string `json:"content"`
		TsUnix    int64  `json:"ts_unix"`
	}
	if err := json.Unmarshal(cmds[0].Payload, &p1); err != nil {
		t.Fatalf("cmds[0] payload: %v", err)
	}
	if p1.Content != "hi" {
		t.Errorf("AppendUserTurn content = %q, want %q", p1.Content, "hi")
	}
	if p1.TsUnix == 0 {
		t.Errorf("AppendUserTurn ts_unix is zero")
	}

	// A later command must be FinalizeAssistantTurn with the assistant's reply.
	var foundFinalize bool
	for _, c := range cmds[1:] {
		if c.Kind != store.FinalizeAssistantTurn {
			continue
		}
		var p2 struct {
			SessionID string `json:"session_id"`
			Content   string `json:"content"`
			TsUnix    int64  `json:"ts_unix"`
		}
		if err := json.Unmarshal(c.Payload, &p2); err != nil {
			t.Fatalf("FinalizeAssistantTurn payload: %v", err)
		}
		if !strings.Contains(p2.Content, "hello back") {
			t.Errorf("FinalizeAssistantTurn content = %q, want contains 'hello back'", p2.Content)
		}
		if p2.SessionID != "sess-finalize-test" {
			t.Errorf("FinalizeAssistantTurn session_id = %q", p2.SessionID)
		}
		foundFinalize = true
		break
	}
	if !foundFinalize {
		t.Errorf("no FinalizeAssistantTurn command captured; got kinds = %v", kindStrings(cmds))
	}
}

func kindStrings(cmds []store.Command) []string {
	out := make([]string, len(cmds))
	for i, c := range cmds {
		out[i] = c.Kind.String()
	}
	return out
}
