package kernel

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel/testfixtures"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func TestKernel_InjectsSkillBlockAndRecordsUsage(t *testing.T) {
	provider := &testfixtures.SkillProvider{
		Block: "<skills>\n## careful-review\nReview carefully.\n</skills>",
		Names: []string{"careful-review"},
	}
	recorder := &testfixtures.SkillUsageRecorder{}
	mc := llm.NewMockClient()
	mc.Script([]llm.Event{{Kind: llm.EventDone, FinishReason: "stop"}}, "sess-skills")

	k := New(Config{
		Model:      "hermes-agent",
		Endpoint:   "http://mock",
		Admission:  Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Skills:     provider,
		SkillUsage: recorder,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()
	_ = k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "please review this patch"})

	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID != ""
	}, 2*time.Second)

	reqs := mc.Requests()
	if len(reqs) == 0 {
		t.Fatal("mock client received zero requests")
	}
	req := reqs[0]
	if len(req.Messages) != 3 {
		t.Fatalf("len(Messages) = %d, want 3 (skills guidance + skill block + user)", len(req.Messages))
	}
	if req.Messages[0].Role != "system" || !strings.Contains(req.Messages[0].Content, llm.SkillsGuidance) {
		t.Fatalf("Messages[0] = %+v, want skills guidance system message", req.Messages[0])
	}
	if req.Messages[1].Role != "system" {
		t.Fatalf("Messages[1].Role = %q, want system", req.Messages[1].Role)
	}
	if !strings.Contains(req.Messages[1].Content, "careful-review") {
		t.Fatalf("skill block system message = %q, want skill block", req.Messages[1].Content)
	}
	if req.Messages[2].Role != "user" || req.Messages[2].Content != "please review this patch" {
		t.Fatalf("Messages[2] = %+v, want user submit", req.Messages[2])
	}
	if provider.Calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.Calls)
	}
	if provider.Last != "please review this patch" {
		t.Fatalf("provider last user message = %q", provider.Last)
	}
	if recorder.Calls != 1 {
		t.Fatalf("recorder calls = %d, want 1", recorder.Calls)
	}
	if !reflect.DeepEqual(recorder.Got, [][]string{{"careful-review"}}) {
		t.Fatalf("recorder got = %#v, want [[careful-review]]", recorder.Got)
	}
}

func TestKernel_SkillProviderErrorFallsBackToUserOnly(t *testing.T) {
	provider := &testfixtures.SkillProvider{Err: errors.New("boom")}
	recorder := &testfixtures.SkillUsageRecorder{}
	mc := llm.NewMockClient()
	mc.Script([]llm.Event{{Kind: llm.EventDone, FinishReason: "stop"}}, "sess-skills-fallback")

	k := New(Config{
		Model:      "hermes-agent",
		Endpoint:   "http://mock",
		Admission:  Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Skills:     provider,
		SkillUsage: recorder,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()
	_ = k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "plain request"})

	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID != ""
	}, 2*time.Second)

	reqs := mc.Requests()
	if len(reqs) == 0 {
		t.Fatal("mock client received zero requests")
	}
	if len(reqs[0].Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1 (user only)", len(reqs[0].Messages))
	}
	if reqs[0].Messages[0].Role != "user" || reqs[0].Messages[0].Content != "plain request" {
		t.Fatalf("Messages[0] = %+v, want user/plain request", reqs[0].Messages[0])
	}
	if recorder.Calls != 0 {
		t.Fatalf("recorder calls = %d, want 0", recorder.Calls)
	}
}
