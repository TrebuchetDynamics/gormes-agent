package plugins

import (
	"context"
	"errors"
	"testing"
)

func TestTransformLLMOutput_FirstNonEmptyWins(t *testing.T) {
	r := NewTransformLLMOutputRegistry(nil)
	r.Register(func(_ context.Context, _ TransformLLMOutputInput) (string, error) {
		return "first", nil
	})
	r.Register(func(_ context.Context, _ TransformLLMOutputInput) (string, error) {
		return "second", nil
	})

	got := r.Run(context.Background(), TransformLLMOutputInput{ResponseText: "original"})
	if got != "first" {
		t.Fatalf("expected first hook result, got %q", got)
	}
}

func TestTransformLLMOutput_EmptyReturnLeavesUnchanged(t *testing.T) {
	r := NewTransformLLMOutputRegistry(nil)
	r.Register(func(_ context.Context, _ TransformLLMOutputInput) (string, error) {
		return "", nil // empty → skip
	})
	r.Register(func(_ context.Context, _ TransformLLMOutputInput) (string, error) {
		return "transformed", nil
	})

	got := r.Run(context.Background(), TransformLLMOutputInput{ResponseText: "original"})
	if got != "transformed" {
		t.Fatalf("expected second hook result after empty skip, got %q", got)
	}
}

func TestTransformLLMOutput_ErrorPreservesOriginal(t *testing.T) {
	r := NewTransformLLMOutputRegistry(nil)
	r.Register(func(_ context.Context, _ TransformLLMOutputInput) (string, error) {
		return "", errors.New("hook exploded")
	})
	r.Register(func(_ context.Context, _ TransformLLMOutputInput) (string, error) {
		return "after error", nil
	})

	got := r.Run(context.Background(), TransformLLMOutputInput{ResponseText: "original"})
	if got != "after error" {
		t.Fatalf("expected hook after error to run; got %q", got)
	}
}

func TestTransformLLMOutput_NoHooksReturnsOriginal(t *testing.T) {
	r := NewTransformLLMOutputRegistry(nil)
	got := r.Run(context.Background(), TransformLLMOutputInput{ResponseText: "original"})
	if got != "original" {
		t.Fatalf("expected original response with no hooks; got %q", got)
	}
}

func TestTransformLLMOutput_AllHooksFailReturnsOriginal(t *testing.T) {
	r := NewTransformLLMOutputRegistry(nil)
	r.Register(func(_ context.Context, _ TransformLLMOutputInput) (string, error) {
		return "", errors.New("fail1")
	})
	r.Register(func(_ context.Context, _ TransformLLMOutputInput) (string, error) {
		return "", errors.New("fail2")
	})

	got := r.Run(context.Background(), TransformLLMOutputInput{ResponseText: "original"})
	if got != "original" {
		t.Fatalf("expected original response when all hooks fail; got %q", got)
	}
}

func TestTransformLLMOutput_RegistrationOrder(t *testing.T) {
	r := NewTransformLLMOutputRegistry(nil)
	order := []string{}
	r.Register(func(_ context.Context, _ TransformLLMOutputInput) (string, error) {
		order = append(order, "a")
		return "", nil
	})
	r.Register(func(_ context.Context, _ TransformLLMOutputInput) (string, error) {
		order = append(order, "b")
		return "b-result", nil
	})
	r.Register(func(_ context.Context, _ TransformLLMOutputInput) (string, error) {
		order = append(order, "c")
		return "", nil
	})

	got := r.Run(context.Background(), TransformLLMOutputInput{ResponseText: "original"})
	if got != "b-result" {
		t.Fatalf("expected b-result; got %q", got)
	}
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("hooks should run in registration order a→b; c skipped after first non-empty; got %v", order)
	}
}

func TestTransformLLMOutput_ReceivesInputFields(t *testing.T) {
	r := NewTransformLLMOutputRegistry(nil)
	var captured TransformLLMOutputInput
	r.Register(func(_ context.Context, input TransformLLMOutputInput) (string, error) {
		captured = input
		return "ok", nil
	})

	r.Run(context.Background(), TransformLLMOutputInput{
		ResponseText: "hello world",
		SessionID:    "s1",
		Model:        "anthropic/claude-sonnet-4.6",
		Platform:     "cli",
	})

	if captured.ResponseText != "hello world" {
		t.Fatalf("ResponseText: got %q", captured.ResponseText)
	}
	if captured.SessionID != "s1" {
		t.Fatalf("SessionID: got %q", captured.SessionID)
	}
	if captured.Model != "anthropic/claude-sonnet-4.6" {
		t.Fatalf("Model: got %q", captured.Model)
	}
	if captured.Platform != "cli" {
		t.Fatalf("Platform: got %q", captured.Platform)
	}
}
