package hermes

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestModelCatalogValidationAndCache(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	cachePath := filepath.Join(t.TempDir(), "model_catalog.json")
	manifest := validModelCatalogManifest()
	raw := mustMarshalModelCatalog(t, manifest)
	fetchCalls := 0
	catalog := NewModelCatalog(ModelCatalogOptions{
		Config: ModelCatalogConfig{
			Enabled:   true,
			URL:       "https://example.test/model-catalog.json",
			TTL:       24 * time.Hour,
			CachePath: cachePath,
		},
		Fetcher: func(_ context.Context, url string) ([]byte, error) {
			if url != "https://example.test/model-catalog.json" {
				t.Fatalf("fetch url = %q", url)
			}
			fetchCalls++
			return raw, nil
		},
		Now: func() time.Time { return now },
	})

	got, evidence, err := catalog.Get(ctx, true)
	if err != nil {
		t.Fatalf("Get force refresh: %v", err)
	}
	if got.Version != 1 || got.Providers["openrouter"].Models[0].ID != "anthropic/claude-opus-4.7" {
		t.Fatalf("manifest = %#v", got)
	}
	if evidence.Source != ModelCatalogSourceNetwork {
		t.Fatalf("evidence.Source = %q, want %q", evidence.Source, ModelCatalogSourceNetwork)
	}
	if fetchCalls != 1 {
		t.Fatalf("fetchCalls = %d, want 1", fetchCalls)
	}
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("cache not written: %v", err)
	}

	_, evidence, err = catalog.Get(ctx, false)
	if err != nil {
		t.Fatalf("Get memory cache: %v", err)
	}
	if evidence.Source != ModelCatalogSourceMemory {
		t.Fatalf("memory evidence.Source = %q", evidence.Source)
	}
	if fetchCalls != 1 {
		t.Fatalf("fetchCalls after memory hit = %d, want 1", fetchCalls)
	}

	_, evidence, err = catalog.Get(ctx, true)
	if err != nil {
		t.Fatalf("Get second force refresh: %v", err)
	}
	if evidence.Source != ModelCatalogSourceNetwork || fetchCalls != 2 {
		t.Fatalf("second refresh source=%q fetchCalls=%d", evidence.Source, fetchCalls)
	}

	old := now.Add(-48 * time.Hour)
	if err := os.Chtimes(cachePath, old, old); err != nil {
		t.Fatalf("chtimes cache: %v", err)
	}
	stale := NewModelCatalog(ModelCatalogOptions{
		Config: ModelCatalogConfig{
			Enabled:   true,
			URL:       "https://example.test/model-catalog.json",
			TTL:       time.Hour,
			CachePath: cachePath,
		},
		Fetcher: func(context.Context, string) ([]byte, error) {
			return nil, errors.New("network down")
		},
		Now: func() time.Time { return now },
	})
	got, evidence, err = stale.Get(ctx, false)
	if err != nil {
		t.Fatalf("Get stale fallback: %v", err)
	}
	if got.Providers["nous"].Models[0].ID != "moonshotai/kimi-k2.6" {
		t.Fatalf("stale fallback manifest = %#v", got)
	}
	if evidence.Source != ModelCatalogSourceDiskStale || evidence.DegradedReason != "model_catalog_fetch_failed" {
		t.Fatalf("stale evidence = %#v", evidence)
	}

	disabledFetchCalled := false
	disabled := NewModelCatalog(ModelCatalogOptions{
		Config: ModelCatalogConfig{Enabled: false, URL: "https://ignored.test", TTL: time.Hour, CachePath: cachePath},
		Fetcher: func(context.Context, string) ([]byte, error) {
			disabledFetchCalled = true
			return raw, nil
		},
	})
	got, evidence, err = disabled.Get(ctx, false)
	if err != nil {
		t.Fatalf("disabled Get: %v", err)
	}
	if len(got.Providers) != 0 || evidence.DegradedReason != "model_catalog_disabled" || disabledFetchCalled {
		t.Fatalf("disabled got=%#v evidence=%#v fetchCalled=%v", got, evidence, disabledFetchCalled)
	}

	if err := ValidateModelCatalogManifest(manifest); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	future := manifest
	future.Version = 999
	if err := ValidateModelCatalogManifest(future); err == nil {
		t.Fatal("future schema version accepted")
	}
	malformed := manifest
	malformed.Providers = nil
	if err := ValidateModelCatalogManifest(malformed); err == nil {
		t.Fatal("manifest without providers accepted")
	}
}

func TestModelCatalogOverrideURL(t *testing.T) {
	ctx := context.Background()
	master := validModelCatalogManifest()
	override := validModelCatalogManifest()
	override.Providers["openrouter"] = ModelCatalogProvider{
		Models: []ModelCatalogModel{{ID: "override/model", Description: "custom"}},
	}
	catalog := NewModelCatalog(ModelCatalogOptions{
		Config: ModelCatalogConfig{
			Enabled: true,
			URL:     "https://example.test/master.json",
			TTL:     time.Hour,
			ProviderOverrideURLs: map[string]string{
				"openrouter": "https://example.test/openrouter.json",
			},
			CachePath: filepath.Join(t.TempDir(), "model_catalog.json"),
		},
		Fetcher: func(_ context.Context, url string) ([]byte, error) {
			if url == "https://example.test/openrouter.json" {
				return mustMarshalModelCatalog(t, override), nil
			}
			return mustMarshalModelCatalog(t, master), nil
		},
	})

	got, evidence, err := catalog.CuratedOpenRouterModels(ctx)
	if err != nil {
		t.Fatalf("CuratedOpenRouterModels: %v", err)
	}
	want := []ModelCatalogChoice{{ID: "override/model", Description: "custom"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("choices = %#v, want %#v", got, want)
	}
	if evidence.Source != ModelCatalogSourceOverride || evidence.URL != "https://example.test/openrouter.json" {
		t.Fatalf("override evidence = %#v", evidence)
	}
}

func TestAIGatewayPricingAndFreePromotion(t *testing.T) {
	payload := []byte(`{"data":[
		{"id":"anthropic/claude-opus-4.7","pricing":{"input":"0.001","output":"0.002","input_cache_read":"0.0001","input_cache_write":"0.0002"}},
		{"id":"moonshotai/kimi-coder-free-preview","pricing":{"input":"0","output":"0"}},
		{"id":"moonshotai/kimi-paid","pricing":{"input":"0.001","output":"0.002"}},
		{"id":"not-language","pricing":null}
	]}`)

	pricing, err := ParseAIGatewayModelPricing(payload)
	if err != nil {
		t.Fatalf("ParseAIGatewayModelPricing: %v", err)
	}
	got := pricing["anthropic/claude-opus-4.7"]
	if got.Prompt != "0.001" || got.Completion != "0.002" || got.InputCacheRead != "0.0001" || got.InputCacheWrite != "0.0002" {
		t.Fatalf("pricing = %#v", got)
	}
	curated := []ModelCatalogChoice{{ID: "anthropic/claude-opus-4.7", Description: "recommended"}}
	merged, err := MergeAIGatewayFreePromotions(curated, payload)
	if err != nil {
		t.Fatalf("MergeAIGatewayFreePromotions: %v", err)
	}
	if merged[0] != (ModelCatalogChoice{ID: "moonshotai/kimi-coder-free-preview", Description: "recommended"}) {
		t.Fatalf("first promoted choice = %#v", merged[0])
	}
	if len(merged) != 2 {
		t.Fatalf("merged = %#v, want promoted free model plus curated model", merged)
	}
}

func TestPreferredProviderModelMerge(t *testing.T) {
	curated := []string{"kimi-k2.6", "kimi-k2.5", "mimo-v2-pro"}
	modelsDev := []string{"mimo-v2.5-pro", "mimo-v2-pro", "KIMI-K2.6"}
	got := MergePreferredProviderModels("opencode-go", curated, modelsDev)
	want := []string{"mimo-v2.5-pro", "mimo-v2-pro", "KIMI-K2.6", "kimi-k2.5"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MergePreferredProviderModels = %#v, want %#v", got, want)
	}

	offline := MergePreferredProviderModels("opencode-zen", curated, nil)
	if !reflect.DeepEqual(offline, curated) {
		t.Fatalf("offline merge = %#v, want curated fallback %#v", offline, curated)
	}
	if !IsModelsDevPreferredProvider("opencode-go") || !IsModelsDevPreferredProvider("opencode-zen") {
		t.Fatal("opencode providers should be models.dev-preferred")
	}
}

func TestOpenRouterNousCatalogBehaviorUnchanged(t *testing.T) {
	curated := []string{"anthropic/claude-opus-4.7"}
	modelsDev := []string{"unbounded/catalog-entry"}
	for _, provider := range []string{"openrouter", "nous"} {
		if IsModelsDevPreferredProvider(provider) {
			t.Fatalf("%s must not be models.dev-preferred", provider)
		}
		got := MergePreferredProviderModels(provider, curated, modelsDev)
		if !reflect.DeepEqual(got, curated) {
			t.Fatalf("%s merge = %#v, want curated unchanged", provider, got)
		}
	}
}

func validModelCatalogManifest() ModelCatalogManifest {
	return ModelCatalogManifest{
		Version:   1,
		UpdatedAt: "2026-04-25T22:00:00Z",
		Providers: map[string]ModelCatalogProvider{
			"openrouter": {
				Models: []ModelCatalogModel{
					{ID: "anthropic/claude-opus-4.7", Description: "recommended"},
					{ID: "openai/gpt-5.4"},
				},
			},
			"nous": {
				Models: []ModelCatalogModel{
					{ID: "moonshotai/kimi-k2.6"},
				},
			},
		},
	}
}

func mustMarshalModelCatalog(t *testing.T, manifest ModelCatalogManifest) []byte {
	t.Helper()
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	return raw
}
