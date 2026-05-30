//go:build !slim

package imagegen

import (
	"context"
	"errors"
	"sort"
	"sync"
)

// ImageGenProvider is the Go-native image generation provider registration
// contract. It mirrors Hermes' name-addressable plugin provider registry while
// using the existing fakeable ImageGenerator call surface.
type ImageGenProvider interface {
	Name() string
	ImageGenerator
}

type ImageGenActiveProvider struct {
	Name     string
	Provider ImageGenProvider
	Evidence ImageGenerationStatus
}

type ImageGenProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]ImageGenProvider
}

func NewImageGenProviderRegistry() *ImageGenProviderRegistry {
	return &ImageGenProviderRegistry{providers: map[string]ImageGenProvider{}}
}

func (r *ImageGenProviderRegistry) Register(provider any) error {
	if r == nil {
		return errors.New("image_generation: nil provider registry")
	}
	p, ok := provider.(ImageGenProvider)
	if !ok {
		return errors.New("image_generation: provider must implement ImageGenProvider")
	}
	name := normalizeImageProviderName(p.Name())
	if name == "" {
		return errors.New("image_generation: provider name is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = p
	return nil
}

func (r *ImageGenProviderRegistry) RegisterNamed(name string, provider ImageGenerator) error {
	if provider == nil {
		return errors.New("image_generation: provider is required")
	}
	return r.Register(staticImageGenProvider{name: name, generator: provider})
}

func (r *ImageGenProviderRegistry) Get(name string) (ImageGenProvider, bool) {
	if r == nil {
		return nil, false
	}
	key := normalizeImageProviderName(name)
	if key == "" {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[key]
	return provider, ok
}

func (r *ImageGenProviderRegistry) ProviderNames() []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *ImageGenProviderRegistry) ResolveActive(ctx context.Context, configured string) ImageGenActiveProvider {
	configured = normalizeImageProviderName(configured)
	if configured != "" && configured != "auto" {
		provider, ok := r.Get(configured)
		if !ok {
			return ImageGenActiveProvider{Name: configured, Evidence: ImageGenerationStatusProviderNotRegistered}
		}
		if !provider.Available(ctx) {
			return ImageGenActiveProvider{Name: configured, Evidence: ImageGenerationStatus("image_gen_provider_unavailable")}
		}
		return ImageGenActiveProvider{Name: configured, Provider: provider}
	}

	names := r.ProviderNames()
	if len(names) == 0 {
		return ImageGenActiveProvider{}
	}
	if len(names) == 1 {
		return r.resolveAvailable(ctx, names[0])
	}
	if _, ok := r.Get("fal"); ok {
		resolved := r.resolveAvailable(ctx, "fal")
		if resolved.Provider != nil {
			return resolved
		}
	}
	for _, name := range names {
		if name == "fal" {
			continue
		}
		resolved := r.resolveAvailable(ctx, name)
		if resolved.Provider != nil {
			return resolved
		}
	}
	return ImageGenActiveProvider{Evidence: ImageGenerationStatus("image_gen_provider_unavailable")}
}

func (r *ImageGenProviderRegistry) resolveAvailable(ctx context.Context, name string) ImageGenActiveProvider {
	provider, ok := r.Get(name)
	if !ok {
		return ImageGenActiveProvider{}
	}
	if !provider.Available(ctx) {
		return ImageGenActiveProvider{Name: name, Evidence: ImageGenerationStatus("image_gen_provider_unavailable")}
	}
	return ImageGenActiveProvider{Name: name, Provider: provider}
}

type staticImageGenProvider struct {
	name      string
	generator ImageGenerator
}

func (p staticImageGenProvider) Name() string { return p.name }

func (p staticImageGenProvider) Available(ctx context.Context) bool {
	return p.generator != nil && p.generator.Available(ctx)
}

func (p staticImageGenProvider) Generate(ctx context.Context, req ImageProviderRequest) (ImageProviderResult, error) {
	if p.generator == nil {
		return ImageProviderResult{}, errors.New("image_generation: provider unavailable")
	}
	return p.generator.Generate(ctx, req)
}

type ImageGenPluginDiscovery interface {
	EnsureImageGenProvidersDiscovered(context.Context, bool) error
}

type ImageGenPluginDiscoveryFunc func(context.Context, bool) error

func (f ImageGenPluginDiscoveryFunc) EnsureImageGenProvidersDiscovered(ctx context.Context, force bool) error {
	return f(ctx, force)
}
