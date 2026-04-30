package skills

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHermesIndexProviderPrefersCachedIndex(t *testing.T) {
	tmp := t.TempDir()
	cachePath := tmp + "/hermes-skills-index.json"
	if err := os.WriteFile(cachePath, []byte(`{
		"skills": [
			{"name":"planner","description":"Plan work","source":"skills-sh","identifier":"skills-sh/planner","trust_level":"trusted","repo":"openai/skills","path":"skills/planner","tags":["planning","docs"],"score":0.9},
			{"name":"","description":"skip unnamed"}
		]
	}`), 0o600); err != nil {
		t.Fatalf("write cache fixture: %v", err)
	}

	provider := NewHermesIndexRegistryProvider(cachePath)
	results, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot returned unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results count = %d (%v), want 1", len(results), results)
	}
	got := results[0]
	if got.Name != "planner" {
		t.Errorf("Name = %q, want planner", got.Name)
	}
	if got.Description != "Plan work" {
		t.Errorf("Description = %q, want Plan work", got.Description)
	}
	if got.Source != "skills-sh" {
		t.Errorf("Source = %q, want skills-sh", got.Source)
	}
	if got.InstallID != "skills-sh/planner" {
		t.Errorf("InstallID = %q, want skills-sh/planner", got.InstallID)
	}
	if got.TrustLevel != "trusted" {
		t.Errorf("TrustLevel = %q, want trusted", got.TrustLevel)
	}
	if strings.Join(got.Tags, ",") != "planning,docs" {
		t.Errorf("Tags = %v, want [planning docs]", got.Tags)
	}
	if got.Score != 0.9 {
		t.Errorf("Score = %v, want 0.9", got.Score)
	}
}

func TestHermesIndexProviderMalformedOrMissingEvidence(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		provider := NewHermesIndexRegistryProvider(t.TempDir() + "/missing.json")
		_, err := provider.Snapshot(context.Background())
		if !errors.Is(err, ErrRegistryUnavailable) {
			t.Fatalf("Snapshot error = %v, want errors.Is(..., %v)", err, ErrRegistryUnavailable)
		}
	})

	t.Run("malformed", func(t *testing.T) {
		cachePath := t.TempDir() + "/index.json"
		if err := os.WriteFile(cachePath, []byte(`{`), 0o600); err != nil {
			t.Fatalf("write malformed fixture: %v", err)
		}
		provider := NewHermesIndexRegistryProvider(cachePath)
		_, err := provider.Snapshot(context.Background())
		if !errors.Is(err, ErrRegistryMalformed) {
			t.Fatalf("Snapshot error = %v, want errors.Is(..., %v)", err, ErrRegistryMalformed)
		}
	})
}

func TestSourceRouterSkipsDuplicateRemoteAPISourcesWhenIndexAvailable(t *testing.T) {
	index := NewInMemoryRegistryProvider([]HubSearchResult{{Name: "planner", Description: "from index", Source: "hermes-index", InstallID: "skills-sh/planner", Score: 1}}, nil)
	remote := NewInMemoryRegistryProvider([]HubSearchResult{{Name: "planner", Description: "from remote", Source: "skills-sh", InstallID: "skills-sh/planner", Score: 2}}, nil)

	providers, evidence := PreferHermesIndexProvider(context.Background(), index, []HubRegistryProvider{remote})
	if evidence != "" {
		t.Fatalf("evidence = %q, want empty", evidence)
	}
	if len(providers) != 1 || providers[0] != index {
		t.Fatalf("providers = %#v, want only the index provider", providers)
	}

	resp, err := Search(context.Background(), "planner", providers, HubSearchOptions{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].Description != "from index" {
		t.Fatalf("Search results = %#v, want centralized index result only", resp.Results)
	}
}

func TestWellKnownRegistryProviderReadsIndexMetadata(t *testing.T) {
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.String() != "https://skills.example/.well-known/skills/index.json" {
			t.Fatalf("unexpected request URL: %s", req.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"skills": [
					{"name":"planner","description":"Plan work","files":["SKILL.md","references/api.md"],"tags":["planning","docs"]},
					{"name":"","description":"skip unnamed"}
				]
			}`)),
		}, nil
	})}

	provider := NewWellKnownRegistryProvider("https://skills.example", client)
	results, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot returned unexpected error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
	if len(results) != 1 {
		t.Fatalf("results count = %d (%v), want 1", len(results), results)
	}
	got := results[0]
	if got.Name != "planner" {
		t.Errorf("Name = %q, want planner", got.Name)
	}
	if got.Description != "Plan work" {
		t.Errorf("Description = %q, want Plan work", got.Description)
	}
	if got.Source != "well-known" {
		t.Errorf("Source = %q, want well-known", got.Source)
	}
	if got.InstallID != "well-known:https://skills.example/.well-known/skills/planner" {
		t.Errorf("InstallID = %q, want well-known:https://skills.example/.well-known/skills/planner", got.InstallID)
	}
	if got.TrustLevel != "community" {
		t.Errorf("TrustLevel = %q, want community", got.TrustLevel)
	}
	if strings.Join(got.Tags, ",") != "planning,docs" {
		t.Errorf("Tags = %v, want [planning docs]", got.Tags)
	}
}

func TestClawHubProviderCommunityTrustAndDegradedEvidence(t *testing.T) {
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.String() != "https://clawhub.test/api/v1/skills?limit=100" {
			t.Fatalf("unexpected request URL: %s", req.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"items": [
					{"slug":"touchdesigner-agent","displayName":"TouchDesigner Agent","summary":"Build TD networks","tags":{"creative":true,"latest":"1.0.0"}},
					{"slug":"","displayName":"skip unnamed"}
				]
			}`)),
		}, nil
	})}

	provider := NewClawHubRegistryProvider("https://clawhub.test/api/v1", client)
	results, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot returned unexpected error: %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
	if len(results) != 1 {
		t.Fatalf("results count = %d (%v), want 1", len(results), results)
	}
	got := results[0]
	if got.Name != "TouchDesigner Agent" {
		t.Errorf("Name = %q, want TouchDesigner Agent", got.Name)
	}
	if got.Description != "Build TD networks" {
		t.Errorf("Description = %q, want Build TD networks", got.Description)
	}
	if got.Source != "clawhub" {
		t.Errorf("Source = %q, want clawhub", got.Source)
	}
	if got.InstallID != "clawhub:touchdesigner-agent" {
		t.Errorf("InstallID = %q, want clawhub:touchdesigner-agent", got.InstallID)
	}
	if got.TrustLevel != "community" {
		t.Errorf("TrustLevel = %q, want community", got.TrustLevel)
	}
	if strings.Join(got.Tags, ",") != "creative" {
		t.Errorf("Tags = %v, want [creative]", got.Tags)
	}
}

func TestClawHubProviderDegradedEvidence(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       error
	}{
		{name: "rate limited", statusCode: http.StatusTooManyRequests, body: `{}`, want: ErrRegistryRateLimited},
		{name: "unavailable", statusCode: http.StatusBadGateway, body: `{}`, want: ErrRegistryUnavailable},
		{name: "malformed", statusCode: http.StatusOK, body: `{`, want: ErrRegistryMalformed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			})}

			provider := NewClawHubRegistryProvider("https://clawhub.test", client)
			_, err := provider.Snapshot(context.Background())
			if !errors.Is(err, tt.want) {
				t.Fatalf("Snapshot error = %v, want errors.Is(..., %v)", err, tt.want)
			}
		})
	}
}

func TestWellKnownRegistryProviderDegradedEvidence(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       error
	}{
		{name: "rate limited", statusCode: http.StatusTooManyRequests, body: `{}`, want: ErrRegistryRateLimited},
		{name: "unavailable", statusCode: http.StatusBadGateway, body: `{}`, want: ErrRegistryUnavailable},
		{name: "malformed", statusCode: http.StatusOK, body: `{`, want: ErrRegistryMalformed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			})}

			provider := NewWellKnownRegistryProvider("https://skills.example", client)
			_, err := provider.Snapshot(context.Background())
			if !errors.Is(err, tt.want) {
				t.Fatalf("Snapshot error = %v, want errors.Is(..., %v)", err, tt.want)
			}
		})
	}
}
