package kernel

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

func TestKernel_SubmitPreservesMultimodalContentParts(t *testing.T) {
	client := hermes.NewMockClient()
	client.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "A cat"},
		{Kind: hermes.EventDone, FinishReason: "stop"},
	}, "sess-multimodal")
	k := New(Config{
		Model:     "gormes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, client, store.NewNoop(), telemetry.New(), slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = k.Run(ctx) }()
	initial := <-k.Render()

	parts := []hermes.MessageContentPart{
		{Type: "text", Text: "describe this"},
		{Type: "image_url", ImageURL: "https://example.com/cat.png", Detail: "high"},
	}
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "describe this", ContentParts: parts}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	drainUntilIdle(t, k.Render(), initial.Seq, time.Second)

	requests := client.Requests()
	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
	gotParts := requests[0].Messages[len(requests[0].Messages)-1].ContentParts
	if len(gotParts) != len(parts) {
		t.Fatalf("len(ContentParts) = %d, want %d: %+v", len(gotParts), len(parts), gotParts)
	}
	for i := range parts {
		if gotParts[i] != parts[i] {
			t.Fatalf("ContentParts[%d] = %+v, want %+v", i, gotParts[i], parts[i])
		}
	}
}

func TestKernel_SubmitAllowsImageOnlyContentParts(t *testing.T) {
	client := hermes.NewMockClient()
	client.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "Image received"},
		{Kind: hermes.EventDone, FinishReason: "stop"},
	}, "sess-image-only")
	k := New(Config{
		Model:     "gormes-agent",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
	}, client, store.NewNoop(), telemetry.New(), slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = k.Run(ctx) }()
	initial := <-k.Render()

	parts := []hermes.MessageContentPart{{Type: "image_url", ImageURL: "data:image/png;base64,AAAA"}}
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, ContentParts: parts}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	drainUntilIdle(t, k.Render(), initial.Seq, time.Second)

	requests := client.Requests()
	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}
	if got := requests[0].Messages[len(requests[0].Messages)-1].Content; got != "" {
		t.Fatalf("Content = %q, want empty text for image-only submit", got)
	}
	if got := requests[0].Messages[len(requests[0].Messages)-1].ContentParts; len(got) != 1 || got[0] != parts[0] {
		t.Fatalf("ContentParts = %+v, want %+v", got, parts)
	}
}
