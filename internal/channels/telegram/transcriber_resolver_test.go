package telegram

import (
	"context"
	"errors"
	"testing"
)

func TestResolveAudioTranscriber_PrefersLocalCLIWhenAvailable(t *testing.T) {
	local := fakeAudioTranscriber{Transcript: "from-local"}
	httpFallback := fakeAudioTranscriber{Transcript: "from-http"}

	resolved := ResolveAudioTranscriber(local, httpFallback)
	if resolved == nil {
		t.Fatal("expected resolver to return local transcriber, got nil")
	}

	got, err := resolved.Transcribe(context.Background(), AudioInput{Kind: "voice"})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if got != "from-local" {
		t.Fatalf("expected local transcript routed first, got %q (HTTP fallback should not have been called)", got)
	}
}

func TestResolveAudioTranscriber_FallsBackToHTTPProviderWhenLocalAbsent(t *testing.T) {
	httpFallback := fakeAudioTranscriber{Transcript: "from-http"}

	resolved := ResolveAudioTranscriber(nil, httpFallback)
	if resolved == nil {
		t.Fatal("expected resolver to return HTTP fallback when local is nil, got nil")
	}

	got, err := resolved.Transcribe(context.Background(), AudioInput{Kind: "voice"})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if got != "from-http" {
		t.Fatalf("expected HTTP fallback transcript, got %q", got)
	}
}

func TestResolveAudioTranscriber_ReturnsNilWhenNeitherConfigured(t *testing.T) {
	resolved := ResolveAudioTranscriber(nil, nil)
	if resolved != nil {
		t.Fatalf("expected nil when no transcribers are configured, got %T", resolved)
	}
}

func TestResolveAudioTranscriber_SkipsNilCandidatesInOrder(t *testing.T) {
	httpFallback := fakeAudioTranscriber{Transcript: "from-http"}

	// nil, nil, http -> http wins
	resolved := ResolveAudioTranscriber(nil, nil, httpFallback)
	if resolved == nil {
		t.Fatal("expected HTTP fallback after skipping nil candidates, got nil")
	}
	got, err := resolved.Transcribe(context.Background(), AudioInput{})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if got != "from-http" {
		t.Fatalf("expected HTTP transcript after skipping nils, got %q", got)
	}
}

// errorReturningTranscriber is used to confirm the resolver returns the
// candidate as-is (errors propagate rather than being swallowed).
type errorReturningTranscriber struct {
	Err error
}

func (e errorReturningTranscriber) Transcribe(ctx context.Context, audio AudioInput) (string, error) {
	_ = ctx
	_ = audio
	return "", e.Err
}

func TestResolveAudioTranscriber_PropagatesTranscribeErrors(t *testing.T) {
	want := errors.New("provider down")
	resolved := ResolveAudioTranscriber(nil, errorReturningTranscriber{Err: want})
	if resolved == nil {
		t.Fatal("resolved was nil")
	}
	if _, err := resolved.Transcribe(context.Background(), AudioInput{}); !errors.Is(err, want) {
		t.Fatalf("expected error %v to propagate, got %v", want, err)
	}
}
