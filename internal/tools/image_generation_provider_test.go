//go:build !slim

package tools

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/imagegen"
)

func TestImageGenManagedGatewayProviderBindingWritesEnvelope(t *testing.T) {
	const rawPNGBase64 = "iVBORw0KGgo="
	const schema = `{"type":"object","properties":{"prompt":{"type":"string"},"model":{"type":"string"},"aspect_ratio":{"type":"string"},"image_size":{"type":"string"},"num_images":{"type":"number"},"output_format":{"type":"string"}}}`

	srv := newFakeManagedGatewayServer(t, func(t *testing.T, w http.ResponseWriter, r *http.Request, req fakeManagedGatewayRequest) {
		switch req.Method {
		case "initialize":
			writeManagedJSONResult(w, req.ID, `{"protocolVersion":"2024-11-05","capabilities":{}}`)
		case "tools/list":
			writeManagedJSONResult(w, req.ID, `{"tools":[{"name":"image_generate","description":"managed image generation","inputSchema":`+schema+`}]}`)
		case "tools/call":
			writeManagedJSONResult(w, req.ID, `{"structuredContent":{"image_base64":"`+rawPNGBase64+`","media_type":"image/png"},"isError":false}`)
		default:
			t.Errorf("unexpected method %q", req.Method)
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	})
	bridge := newTestManagedGatewayBridge(t, "fal-queue", srv, "nous-token")
	provider := NewManagedGatewayImageGenProvider(ManagedGatewayImageGenProviderOptions{
		Name:   "managed-fal",
		Bridge: bridge,
	})
	registry := imagegen.NewImageGenProviderRegistry()
	if err := registry.Register(provider); err != nil {
		t.Fatalf("Register: %v", err)
	}

	runner := imagegen.NewImageGenRunnerWithRegistry(imagegen.ImageGenConfig{Provider: "managed-fal"}, registry)
	outDir := t.TempDir()
	seed := 42
	steps := 7
	guidance := 3.5
	result := runner.Generate(context.Background(), imagegen.ImageGenRequest{
		Prompt:            "paint the secret launch plan",
		Model:             "fal-ai/flux-2-pro",
		AspectRatio:       "portrait",
		OutputDir:         outDir,
		NumImages:         1,
		Seed:              &seed,
		NumInferenceSteps: &steps,
		GuidanceScale:     &guidance,
	})
	if !result.Success {
		t.Fatalf("Generate = %+v, want success", result)
	}
	if result.Provider != "managed-fal" {
		t.Fatalf("Provider = %q, want managed-fal", result.Provider)
	}
	if result.FilePath == "" || strings.Contains(result.FilePath, outDir) {
		t.Fatalf("FilePath = %q, want relative artifact path", result.FilePath)
	}
	if result.ImageURL != "" {
		t.Fatalf("ImageURL = %q, want artifact envelope path instead", result.ImageURL)
	}
	if got := srv.LastCallName(); got != "image_generate" {
		t.Fatalf("gateway tool = %q, want image_generate", got)
	}
	var args map[string]any
	if err := json.Unmarshal(srv.LastCallArguments(), &args); err != nil {
		t.Fatalf("decode gateway args: %v", err)
	}
	if args["prompt"] != "paint the secret launch plan" {
		t.Fatalf("prompt arg = %v", args["prompt"])
	}
	if args["model"] != "fal-ai/flux-2-pro" || args["aspect_ratio"] != "portrait" || args["image_size"] != "portrait_16_9" {
		t.Fatalf("forwarded args = %#v", args)
	}
	if args["num_inference_steps"] != float64(7) || args["guidance_scale"] != 3.5 || args["seed"] != float64(42) {
		t.Fatalf("optional args = %#v", args)
	}
}

func TestImageGenManagedGatewayProviderDegradedEvidenceIsRedacted(t *testing.T) {
	srv := newFakeManagedGatewayServer(t, func(t *testing.T, w http.ResponseWriter, r *http.Request, req fakeManagedGatewayRequest) {
		switch req.Method {
		case "initialize":
			writeManagedJSONResult(w, req.ID, `{"protocolVersion":"2024-11-05","capabilities":{}}`)
		case "tools/list":
			writeManagedJSONResult(w, req.ID, `{"tools":[{"name":"image_generate","description":"managed image generation"}]}`)
		case "tools/call":
			writeManagedJSONError(w, req.ID, -32000, "Bearer nous-secret failed for paint the secret launch plan")
		default:
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	})
	bridge := newTestManagedGatewayBridge(t, "fal-queue", srv, "nous-secret")
	provider := NewManagedGatewayImageGenProvider(ManagedGatewayImageGenProviderOptions{Name: "managed-fal", Bridge: bridge})
	registry := imagegen.NewImageGenProviderRegistry()
	if err := registry.Register(provider); err != nil {
		t.Fatalf("Register: %v", err)
	}
	runner := imagegen.NewImageGenRunnerWithRegistry(imagegen.ImageGenConfig{Provider: "managed-fal"}, registry)

	result := runner.Generate(context.Background(), imagegen.ImageGenRequest{
		Prompt:    "paint the secret launch plan",
		OutputDir: t.TempDir(),
	})
	if result.Success {
		t.Fatalf("Generate = %+v, want failure", result)
	}
	if result.Evidence != imagegen.ImageGenerationStatus("image_gen_api_error") {
		t.Fatalf("Evidence = %q, want image_gen_api_error", result.Evidence)
	}
	if strings.Contains(result.Error, "nous-secret") || strings.Contains(result.Error, "paint the secret launch plan") || strings.Contains(result.Error, "Bearer") {
		t.Fatalf("error leaked secret or prompt: %+v", result)
	}
}

func TestImageGenManagedGatewayProviderUnavailableWhenDiscoveryFails(t *testing.T) {
	provider := NewManagedGatewayImageGenProvider(ManagedGatewayImageGenProviderOptions{Name: "managed-fal"})
	if provider.Available(context.Background()) {
		t.Fatal("Available = true, want false for nil bridge")
	}
	_, err := provider.Generate(context.Background(), imagegen.ImageProviderRequest{Prompt: "hello", Model: imagegen.DefaultFLUXModel})
	if err == nil || !errors.Is(err, imagegen.ErrImageGenProviderUnavailable) {
		t.Fatalf("Generate error = %v, want provider unavailable", err)
	}
}

var _ = time.Second
