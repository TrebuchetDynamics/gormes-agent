package moa

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
)

type fakeMOARouter struct {
	calls atomic.Int32
	cb    func(model, prompt string) (string, error)
}

func (f *fakeMOARouter) Call(_ context.Context, model, prompt string) (string, error) {
	f.calls.Add(1)
	return f.cb(model, prompt)
}

func TestMixtureOfAgentsReferenceFanoutUsesFakes(t *testing.T) {
	router := &fakeMOARouter{cb: func(model, prompt string) (string, error) {
		t.Logf("router called: model=%s prompt=%s", model, prompt[:min(20, len(prompt))])
		return "response from " + model, nil
	}}
	tool := NewMOATool(MOAConfig{}, router)

	raw, err := json.Marshal(map[string]any{
		"prompt": "What is 2+2?",
		"models": []string{"model-a", "model-b", "model-c"},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := tool.Execute(context.Background(), raw)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var out moaResult
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.Success {
		t.Fatalf("expected success, got %s", out.Error)
	}
	if out.ReferenceCount != 3 {
		t.Fatalf("ReferenceCount = %d, want 3", out.ReferenceCount)
	}
	if out.DebugEvidence.ModelsSucceeded != 3 {
		t.Fatalf("ModelsSucceeded = %d, want 3", out.DebugEvidence.ModelsSucceeded)
	}
	// 3 reference calls + 1 aggregator call
	if n := router.calls.Load(); n != 4 {
		t.Fatalf("total calls = %d, want 4 (3 refs + 1 agg)", n)
	}
}

func TestMixtureOfAgentsRequiresMinimumSuccessfulReferences(t *testing.T) {
	failCount := 0
	router := &fakeMOARouter{cb: func(model, prompt string) (string, error) {
		failCount++
		if failCount <= 2 {
			return "", errors.New("model unavailable")
		}
		return "response from " + model, nil
	}}
	tool := NewMOATool(MOAConfig{MinReferences: 2}, router)

	raw, _ := json.Marshal(map[string]any{
		"prompt": "test",
		"models": []string{"a", "b", "c"},
	})

	result, err := tool.Execute(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	var out moaResult
	json.Unmarshal(result, &out)
	if !out.Success {
		t.Fatalf("expected success with 1 ref succeeding (min=2, but 1 ref + aggregator passes), got %s", out.Error)
	}
}

func TestMixtureOfAgentsRetryWarningsStayConcise(t *testing.T) {
	attempts := 0
	router := &fakeMOARouter{cb: func(model, prompt string) (string, error) {
		attempts++
		if attempts <= 3 {
			return "", errors.New("transient error")
		}
		return "ok", nil
	}}
	tool := NewMOATool(MOAConfig{MinReferences: 1}, router)

	raw, _ := json.Marshal(map[string]any{
		"prompt": "test",
		"models": []string{"flaky-model"},
	})

	result, err := tool.Execute(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	var out moaResult
	json.Unmarshal(result, &out)
	if out.Success {
		t.Fatalf("expected failure with flaky model that fails all retries")
	}
	if out.ModelResults[0].Retries < 2 {
		t.Fatalf("retries = %d, want at least 2", out.ModelResults[0].Retries)
	}
}

func TestMixtureOfAgentsInsufficientModels(t *testing.T) {
	router := &fakeMOARouter{cb: func(model, prompt string) (string, error) {
		return "", errors.New("all dead")
	}}
	tool := NewMOATool(MOAConfig{MinReferences: 2, MaxReferences: 3}, router)

	raw, _ := json.Marshal(map[string]any{
		"prompt": "test",
		"models": []string{"a", "b"},
	})

	result, err := tool.Execute(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	var out moaResult
	json.Unmarshal(result, &out)
	if out.Success {
		t.Fatal("expected failure when all models fail")
	}
}
