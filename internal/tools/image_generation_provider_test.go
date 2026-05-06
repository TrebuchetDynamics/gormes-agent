//go:build !slim

package tools

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestImageGenProviderRegistryRegisterLookupAndSort(t *testing.T) {
	registry := NewImageGenProviderRegistry()

	if err := registry.Register("not a provider"); err == nil {
		t.Fatal("Register(non-provider) error = nil, want failure")
	}
	if err := registry.Register(namedImageProvider{name: "  ", fakeImageProvider: &fakeImageProvider{available: true}}); err == nil {
		t.Fatal("Register(empty-name provider) error = nil, want failure")
	}

	alpha := namedImageProvider{name: "alpha", fakeImageProvider: &fakeImageProvider{available: true}}
	zeta := namedImageProvider{name: "zeta", fakeImageProvider: &fakeImageProvider{available: true}}
	replacement := namedImageProvider{name: "alpha", fakeImageProvider: &fakeImageProvider{available: true, result: ImageProviderResult{Provider: "replacement"}}}
	for _, provider := range []ImageGenProvider{zeta, alpha, replacement} {
		if err := registry.Register(provider); err != nil {
			t.Fatalf("Register(%s): %v", provider.Name(), err)
		}
	}

	if got := registry.ProviderNames(); !reflect.DeepEqual(got, []string{"alpha", "zeta"}) {
		t.Fatalf("ProviderNames = %v, want sorted alpha/zeta", got)
	}
	got, ok := registry.Get("alpha")
	if !ok {
		t.Fatal("Get(alpha) ok=false, want replacement")
	}
	result, err := got.Generate(context.Background(), ImageProviderRequest{Model: DefaultFLUXModel})
	if err != nil {
		t.Fatalf("Generate replacement: %v", err)
	}
	if result.Provider != "replacement" {
		t.Fatalf("replacement provider result = %+v, want replacement", result)
	}
}

func TestImageGenActiveProviderResolution(t *testing.T) {
	ctx := context.Background()
	registry := NewImageGenProviderRegistry()

	if got := registry.ResolveActive(ctx, ""); got.Provider != nil || got.Evidence != "" {
		t.Fatalf("empty registry active = %+v, want nil without evidence", got)
	}

	solo := namedImageProvider{name: "solo", fakeImageProvider: &fakeImageProvider{available: true}}
	if err := registry.Register(solo); err != nil {
		t.Fatalf("Register solo: %v", err)
	}
	if got := registry.ResolveActive(ctx, ""); got.Name != "solo" || got.Provider == nil {
		t.Fatalf("single provider active = %+v, want solo", got)
	}

	if err := registry.Register(namedImageProvider{name: "fal", fakeImageProvider: &fakeImageProvider{available: true}}); err != nil {
		t.Fatalf("Register fal: %v", err)
	}
	if err := registry.Register(namedImageProvider{name: "openai", fakeImageProvider: &fakeImageProvider{available: true}}); err != nil {
		t.Fatalf("Register openai: %v", err)
	}
	if got := registry.ResolveActive(ctx, ""); got.Name != "fal" || got.Provider == nil {
		t.Fatalf("multi provider active = %+v, want fal fallback", got)
	}
	if got := registry.ResolveActive(ctx, "openai"); got.Name != "openai" || got.Provider == nil {
		t.Fatalf("configured provider active = %+v, want openai", got)
	}
	if got := registry.ResolveActive(ctx, "missing"); got.Provider != nil || got.Evidence != ImageGenerationStatusProviderNotRegistered || got.Name != "missing" {
		t.Fatalf("missing configured provider = %+v, want provider_not_registered evidence", got)
	}
}

func TestImageGenerationDispatchRoutesConfiguredPluginProvider(t *testing.T) {
	provider := namedImageProvider{
		name: "codex",
		fakeImageProvider: &fakeImageProvider{
			available: true,
			result: ImageProviderResult{
				Provider:  "codex",
				ImageURL:  "/tmp/codex-test.png",
				MediaType: "image/png",
			},
		},
	}
	runner := NewImageGenRunner(ImageGenConfig{
		Provider:     "codex",
		DefaultModel: "fal-ai/flux-2/klein/9b",
	}, map[string]ImageGenerator{
		"codex": provider,
	})

	raw, err := NewImageGenTool(runner).Execute(context.Background(), json.RawMessage(`{"prompt":"draw cat","aspect_ratio":"square"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var result ImageGenResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("result JSON: %v", err)
	}
	if !result.Success || result.Provider != "codex" || result.Model != DefaultFLUXModel || result.AspectRatio != "square" {
		t.Fatalf("result = %+v, want codex success with model/aspect ratio", result)
	}
	if provider.fakeImageProvider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.fakeImageProvider.calls)
	}
}

func TestImageGenerationDispatchForceRefreshesPluginsOnce(t *testing.T) {
	registry := NewImageGenProviderRegistry()
	var calls []bool
	runner := NewImageGenRunnerWithRegistry(ImageGenConfig{
		Provider: "codex",
		PluginDiscovery: ImageGenPluginDiscoveryFunc(func(_ context.Context, force bool) error {
			calls = append(calls, force)
			if force {
				return registry.Register(namedImageProvider{name: "codex", fakeImageProvider: &fakeImageProvider{
					available: true,
					result: ImageProviderResult{
						Provider:  "codex",
						ImageURL:  "/tmp/codex-test.png",
						MediaType: "image/png",
					},
				}})
			}
			return nil
		}),
	}, registry)

	result := runner.Generate(context.Background(), ImageGenRequest{
		Prompt:      "draw hammy",
		AspectRatio: "portrait",
		OutputDir:   t.TempDir(),
	})

	if !reflect.DeepEqual(calls, []bool{false, true}) {
		t.Fatalf("discovery calls = %v, want [false true]", calls)
	}
	if !result.Success || result.Provider != "codex" || result.AspectRatio != "portrait" {
		t.Fatalf("result = %+v, want codex success after force refresh", result)
	}
}

func TestImageGenerationPluginProviderErrorsAreRedacted(t *testing.T) {
	prompt := "secret-prompt-fragment"
	token := "sk-secret-token"
	runner := NewImageGenRunner(ImageGenConfig{Provider: "codex"}, map[string]ImageGenerator{
		"codex": namedImageProvider{name: "codex", fakeImageProvider: &fakeImageProvider{
			available: true,
			err:       errors.New("failed for " + prompt + " with Bearer " + token),
		}},
	})

	result := runner.Generate(context.Background(), ImageGenRequest{
		Prompt:    prompt,
		OutputDir: t.TempDir(),
	})
	if result.Success || result.Evidence != ImageGenerationStatus("image_gen_api_error") {
		t.Fatalf("result = %+v, want provider error", result)
	}
	if strings.Contains(result.Error, prompt) || strings.Contains(result.Error, token) || strings.Contains(result.Error, "Bearer ") {
		t.Fatalf("provider error leaked prompt or secret: %+v", result)
	}

	unavailable := NewImageGenRunner(ImageGenConfig{Provider: "codex"}, map[string]ImageGenerator{
		"codex": namedImageProvider{name: "codex", fakeImageProvider: &fakeImageProvider{available: false}},
	}).Generate(context.Background(), ImageGenRequest{
		Prompt:    prompt,
		OutputDir: t.TempDir(),
	})
	if unavailable.Success || unavailable.Evidence != ImageGenerationStatus("image_gen_provider_unavailable") {
		t.Fatalf("unavailable result = %+v, want provider unavailable", unavailable)
	}
	if strings.Contains(unavailable.Error, prompt) || strings.Contains(unavailable.Error, token) {
		t.Fatalf("unavailable provider leaked prompt or secret: %+v", unavailable)
	}
}

type namedImageProvider struct {
	name string
	*fakeImageProvider
}

func (p namedImageProvider) Name() string { return p.name }
