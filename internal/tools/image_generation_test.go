package tools

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestImageGenToolDescriptor(t *testing.T) {
	tool := NewImageGenTool(NewImageGenRunner(ImageGenConfig{}, nil))
	if tool.Name() != "image_generate" {
		t.Fatalf("tool name = %q", tool.Name())
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("schema invalid JSON: %v", err)
	}
	for _, field := range []string{"prompt", "aspect_ratio"} {
		if _, ok := schema.Properties[field]; !ok {
			t.Fatalf("schema missing %q in %s", field, tool.Schema())
		}
	}
	if len(schema.Required) != 1 || schema.Required[0] != "prompt" {
		t.Fatalf("required = %#v, want prompt only", schema.Required)
	}
}

func TestImageGenResultEnvelope(t *testing.T) {
	ctx := context.Background()
	provider := &fakeImageProvider{
		available: true,
		result: ImageProviderResult{
			Provider:  "fal",
			ImageURL:  "https://example.com/image.png",
			MediaType: "image/png",
		},
	}
	cfg := ImageGenConfig{
		DefaultModel: "fal-ai/flux-2/klein/9b",
		Now:          func() time.Time { return time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC) },
	}
	runner := NewImageGenRunner(cfg, map[string]ImageGenerator{
		"fal": provider,
	})
	result := runner.Generate(ctx, ImageGenRequest{
		Prompt:      "a serene mountain landscape",
		AspectRatio: "landscape",
		OutputDir:   t.TempDir(),
	})

	if !result.Success {
		t.Fatalf("result = %+v, want success", result)
	}
	if result.Evidence != ImageGenerationStatusOK {
		t.Errorf("Evidence = %q, want %q", result.Evidence, ImageGenerationStatusOK)
	}
	if result.Provider != "fal" {
		t.Errorf("Provider = %q, want %q", result.Provider, "fal")
	}
	if result.Model != "fal-ai/flux-2/klein/9b" {
		t.Errorf("Model = %q, want %q", result.Model, "fal-ai/flux-2/klein/9b")
	}
	if provider.calls != 1 {
		t.Errorf("provider calls = %d, want 1", provider.calls)
	}
}

func TestImageGenValidation(t *testing.T) {
	ctx := context.Background()
	runner := NewImageGenRunner(ImageGenConfig{}, map[string]ImageGenerator{
		"fal": &fakeImageProvider{available: true},
	})

	empty := runner.Generate(ctx, ImageGenRequest{Prompt: "   ", OutputDir: t.TempDir()})
	if empty.Success || empty.Evidence != ImageGenerationStatus("image_gen_invalid_arguments") {
		t.Fatalf("empty result = %+v, want invalid arguments", empty)
	}

	disabled := NewImageGenRunner(ImageGenConfig{Disabled: true}, map[string]ImageGenerator{
		"fal": &fakeImageProvider{available: true},
	}).Generate(ctx, ImageGenRequest{Prompt: "hello", OutputDir: t.TempDir()})
	if disabled.Success || disabled.Evidence != ImageGenerationStatus("image_gen_disabled") {
		t.Fatalf("disabled result = %+v, want disabled evidence", disabled)
	}
}

func TestImageGenErrorHandling(t *testing.T) {
	ctx := context.Background()
	runner := NewImageGenRunner(ImageGenConfig{}, map[string]ImageGenerator{
		"fal": &fakeImageProvider{
			available: true,
			err:       errors.New("Bearer sk-secret-token failed"),
		},
	})

	result := runner.Generate(ctx, ImageGenRequest{
		Prompt:    "a beach scene",
		OutputDir: t.TempDir(),
	})

	if result.Success {
		t.Fatalf("expected failure, got success")
	}
	if result.Evidence != ImageGenerationStatus("image_gen_api_error") {
		t.Errorf("Evidence = %q, want api_error", result.Evidence)
	}
	if !strings.Contains(result.Error, "[redacted]") && strings.Contains(result.Error, "sk-secret-token") {
		t.Errorf("error message should be redacted: %+v", result)
	}
}

func TestImageGenProviderFallback(t *testing.T) {
	ctx := context.Background()
	noProvider := NewImageGenRunner(ImageGenConfig{}, map[string]ImageGenerator{})
	result := noProvider.Generate(ctx, ImageGenRequest{Prompt: "hello", OutputDir: t.TempDir()})
	if result.Success {
		t.Fatalf("expected failure with no provider, got success")
	}
	if result.Evidence != ImageGenerationStatus("image_gen_provider_unavailable") {
		t.Errorf("Evidence = %q, want provider_unavailable", result.Evidence)
	}
}

func TestImageGenModelResolution(t *testing.T) {
	ctx := context.Background()
	provider := &fakeImageProvider{available: true}
	runner := NewImageGenRunner(ImageGenConfig{
		DefaultModel: "fal-ai/flux-2/klein/9b",
	}, map[string]ImageGenerator{
		"fal": provider,
	})

	result := runner.Generate(ctx, ImageGenRequest{
		Prompt:    "test",
		Model:     "fal-ai/flux-2-pro",
		OutputDir: t.TempDir(),
	})
	if result.Model != "fal-ai/flux-2-pro" {
		t.Errorf("Model = %q, want explicit model", result.Model)
	}
	if provider.lastReq.Model != "fal-ai/flux-2-pro" {
		t.Errorf("provider received model = %q, want fal-ai/flux-2-pro", provider.lastReq.Model)
	}
}

func TestImageGenAspectRatioNormalization(t *testing.T) {
	ctx := context.Background()
	provider := &fakeImageProvider{available: true}
	runner := NewImageGenRunner(ImageGenConfig{}, map[string]ImageGenerator{
		"fal": provider,
	})

	cases := []struct {
		input    string
		expected string
	}{
		{"landscape", "landscape"},
		{"square", "square"},
		{"portrait", "portrait"},
		{"", "landscape"},
		{"invalid", "landscape"},
		{"LANDSCAPE", "landscape"},
	}

	for _, tc := range cases {
		provider.calls = 0
		runner.Generate(ctx, ImageGenRequest{
			Prompt:      "test",
			AspectRatio: tc.input,
			OutputDir:   t.TempDir(),
		})
		if provider.lastReq.AspectRatio != tc.expected {
			t.Errorf("input=%q: AspectRatio = %q, want %q", tc.input, provider.lastReq.AspectRatio, tc.expected)
		}
	}
}

func TestImageGenConfigValidation(t *testing.T) {
	runner := NewImageGenRunner(ImageGenConfig{}, map[string]ImageGenerator{
		"fal": &fakeImageProvider{available: true},
	})
	if runner == nil {
		t.Fatal("runner should not be nil")
	}

	runnerNil := NewImageGenRunner(ImageGenConfig{}, nil)
	if runnerNil == nil {
		t.Fatal("runner should not be nil with nil providers")
	}
}

func TestImageGenToolExecute(t *testing.T) {
	tool := NewImageGenTool(NewImageGenRunner(ImageGenConfig{}, map[string]ImageGenerator{
		"fal": &fakeImageProvider{
			available: true,
			result: ImageProviderResult{
				Provider:  "fal",
				ImageURL:  "https://example.com/image.png",
				MediaType: "image/png",
			},
		},
	}))
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"prompt":"Hello world"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var result ImageGenResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("result JSON: %v", err)
	}
	if !result.Success {
		t.Fatalf("result = %+v, want success", result)
	}
}

func TestFALModelsCatalog(t *testing.T) {
	expected := []string{
		"fal-ai/flux-2/klein/9b",
		"fal-ai/flux-2-pro",
		"fal-ai/z-image/turbo",
		"fal-ai/nano-banana-pro",
		"fal-ai/gpt-image-1.5",
		"fal-ai/gpt-image-2",
		"fal-ai/ideogram/v3",
		"fal-ai/recraft/v4/pro/text-to-image",
		"fal-ai/qwen-image",
	}
	for _, model := range expected {
		meta, ok := FALModels[model]
		if !ok {
			t.Errorf("model %q not in FALModels catalog", model)
			continue
		}
		if meta.Display == "" {
			t.Errorf("model %q has empty Display", model)
		}
		if meta.SizeStyle == "" {
			t.Errorf("model %q has empty SizeStyle", model)
		}
		if len(meta.Sizes) == 0 {
			t.Errorf("model %q has no Sizes", model)
		}
	}
}

func TestFALModelSizeStyles(t *testing.T) {
	for model, meta := range FALModels {
		switch meta.SizeStyle {
		case "image_size_preset", "aspect_ratio", "gpt_literal":
		default:
			t.Errorf("model %q has invalid SizeStyle %q", model, meta.SizeStyle)
		}
		for ratio, size := range meta.Sizes {
			if size == "" {
				t.Errorf("model %q ratio %q has empty size", model, ratio)
			}
		}
	}
}

func TestBuildFALPayload(t *testing.T) {
	meta := FALModelMetadata{
		SizeStyle: "image_size_preset",
		Sizes: map[string]string{
			"landscape": "landscape_16_9",
			"square":    "square_hd",
		},
		Defaults: map[string]any{
			"num_inference_steps": 4,
			"output_format":      "png",
		},
		Supports: map[string]bool{
			"prompt":               true,
			"image_size":           true,
			"num_inference_steps":  true,
			"output_format":        true,
			"seed":                 true,
		},
	}
	seed := 42
	req := ImageProviderRequest{
		Prompt:      "a test prompt",
		AspectRatio: "landscape",
		SizeStyle:   "image_size_preset",
		Size:        "landscape_16_9",
		Model:       "fal-ai/flux-2/klein/9b",
		Seed:        &seed,
	}

	payload := buildFALPayload(req, meta)

	if payload["prompt"] != "a test prompt" {
		t.Errorf("prompt = %v, want 'a test prompt'", payload["prompt"])
	}
	if payload["image_size"] != "landscape_16_9" {
		t.Errorf("image_size = %v, want 'landscape_16_9'", payload["image_size"])
	}
	if payload["num_inference_steps"] != 4 {
		t.Errorf("num_inference_steps = %v, want 4", payload["num_inference_steps"])
	}
	if payload["seed"] != 42 {
		t.Errorf("seed = %v, want 42", payload["seed"])
	}
	if _, ok := payload["unknown_key"]; ok {
		t.Errorf("payload should not contain unknown_key")
	}
}

func TestRedactImageGenError(t *testing.T) {
	cases := []struct {
		input    string
		contains string
	}{
		{"Bearer sk-secret-123 failed", "[redacted]"},
		{"OPENAI_API_KEY=sk-test key=abc", "[redacted]"},
		{"normal error message", "normal error message"},
	}

	for _, tc := range cases {
		got := redactImageGenError(tc.input)
		if tc.contains == "[redacted]" {
			if got == tc.input {
				t.Errorf("input %q should be redacted", tc.input)
			}
		} else {
			if got != tc.contains {
				t.Errorf("input %q: got %q, want %q", tc.input, got, tc.contains)
			}
		}
	}
}

func TestImageGenRunnerNilProof(t *testing.T) {
	var nilRunner *ImageGenRunner
	result := nilRunner.Generate(context.Background(), ImageGenRequest{
		Prompt: "test",
	})
	if result.Success {
		t.Error("nil runner should return failure")
	}
	if result.Evidence != ImageGenerationStatus("image_gen_provider_unavailable") {
		t.Errorf("Evidence = %q, want provider_unavailable", result.Evidence)
	}
}

func TestImageGenToolTimeout(t *testing.T) {
	tool := NewImageGenTool(NewImageGenRunner(ImageGenConfig{}, nil))
	if tool.Timeout() != 120*time.Second {
		t.Errorf("Timeout = %v, want 120s", tool.Timeout())
	}
}

func TestImageGenInvalidJSON(t *testing.T) {
	tool := NewImageGenTool(NewImageGenRunner(ImageGenConfig{}, nil))
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{invalid json}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	var result ImageGenResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("result JSON: %v", err)
	}
	if result.Success {
		t.Fatalf("result = %+v, want failure for invalid JSON", result)
	}
}

func TestImageGenOutputDir(t *testing.T) {
	ctx := context.Background()
	provider := &fakeImageProvider{available: true}
	runner := NewImageGenRunner(ImageGenConfig{}, map[string]ImageGenerator{
		"fal": provider,
	})

	result := runner.Generate(ctx, ImageGenRequest{
		Prompt:    "test",
		OutputDir: "",
	})
	if !result.Success {
		t.Errorf("empty OutputDir should not cause failure: %+v", result)
	}
}

func TestImageGenResultJSON(t *testing.T) {
	result := ImageGenResult{
		Success:   true,
		ImageURL:  "https://example.com/image.png",
		Provider:  "fal",
		Model:     "fal-ai/flux-2/klein/9b",
		Evidence:  ImageGenerationStatusOK,
	}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var unmarshaled ImageGenResult
	if err := json.Unmarshal(raw, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if unmarshaled.Success != result.Success {
		t.Errorf("Success = %v, want %v", unmarshaled.Success, result.Success)
	}
	if unmarshaled.ImageURL != result.ImageURL {
		t.Errorf("ImageURL = %v, want %v", unmarshaled.ImageURL, result.ImageURL)
	}
}

type fakeImageProvider struct {
	available bool
	calls     int
	last      ImageGenRequest
	lastReq   ImageProviderRequest
	result    ImageProviderResult
	err       error
}

func (f *fakeImageProvider) Available(context.Context) bool {
	return f.available
}

func (f *fakeImageProvider) Generate(_ context.Context, req ImageProviderRequest) (ImageProviderResult, error) {
	f.calls++
	f.lastReq = req
	if f.err != nil {
		return ImageProviderResult{}, f.err
	}
	result := f.result
	if result.Provider == "" {
		result.Provider = req.Model
	}
	return result, nil
}

var _ = filepath.Dir
