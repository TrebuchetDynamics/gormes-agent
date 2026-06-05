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

func TestGitHubRegistryProviderReadsTrustedTapMetadata(t *testing.T) {
	requests := []string{}
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.URL.String())
		switch req.URL.String() {
		case "https://api.github.com/repos/openai/skills/contents/skills":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`[{"type":"dir","name":"planner"},{"type":"file","name":"README.md"},{"type":"dir","name":".private"}]`)),
			}, nil
		case "https://api.github.com/repos/openai/skills/contents/skills/planner/SKILL.md":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"content":"LS0tCm5hbWU6IHBsYW5uZXIKZGVzY3JpcHRpb246IFBsYW4gd29yawptZXRhZGF0YToKICBoZXJtZXM6CiAgICB0YWdzOiBbcGxhbm5pbmcsIGRvY3NdCi0tLQojIFBsYW5uZXIKQm9keQo=",
					"encoding":"base64"
				}`)),
			}, nil
		default:
			t.Fatalf("unexpected request URL: %s", req.URL.String())
			return nil, nil
		}
	})}

	provider := NewGitHubRegistryProvider([]GitHubRegistryTap{{Repo: "openai/skills", Path: "skills/"}}, client)
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
	if got.Source != "github" {
		t.Errorf("Source = %q, want github", got.Source)
	}
	if got.InstallID != "openai/skills/skills/planner" {
		t.Errorf("InstallID = %q, want openai/skills/skills/planner", got.InstallID)
	}
	if got.TrustLevel != "trusted" {
		t.Errorf("TrustLevel = %q, want trusted", got.TrustLevel)
	}
	if strings.Join(got.Tags, ",") != "planning,docs" {
		t.Errorf("Tags = %v, want [planning docs]", got.Tags)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %v, want contents listing plus SKILL.md fetch", requests)
	}
}

func TestSkillsShProviderSearchesMetadataWithoutInstalling(t *testing.T) {
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.String() != "https://skills.sh/api/search?limit=10&q=plan" {
			t.Fatalf("unexpected request URL: %s", req.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"skills": [
					{"id":"openai/skills/planner","name":"planner","source":"openai/skills","skillId":"planner","installs":1200},
					{"id":"bad","name":"skip"}
				]
			}`)),
		}, nil
	})}

	provider := NewSkillsShRegistryProvider("plan", 10, client)
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
	if got.Description != "Indexed by skills.sh from openai/skills · 1,200 installs" {
		t.Errorf("Description = %q, want indexed installs description", got.Description)
	}
	if got.Source != "skills.sh" {
		t.Errorf("Source = %q, want skills.sh", got.Source)
	}
	if got.InstallID != "skills-sh:openai/skills/planner" {
		t.Errorf("InstallID = %q, want skills-sh:openai/skills/planner", got.InstallID)
	}
	if got.TrustLevel != "trusted" {
		t.Errorf("TrustLevel = %q, want trusted", got.TrustLevel)
	}
}

func TestGitHubAndSkillsShProvidersDegradedEvidence(t *testing.T) {
	t.Run("github malformed preserves partial search evidence", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case "https://api.github.com/repos/openai/skills/contents/skills":
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`[{"type":"dir","name":"bad"}]`))}, nil
			case "https://api.github.com/repos/openai/skills/contents/skills/bad/SKILL.md":
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{`))}, nil
			default:
				t.Fatalf("unexpected request URL: %s", req.URL.String())
				return nil, nil
			}
		})}

		provider := NewGitHubRegistryProvider([]GitHubRegistryTap{{Repo: "openai/skills", Path: "skills/"}}, client)
		_, err := provider.Snapshot(context.Background())
		if !errors.Is(err, ErrRegistryMalformed) {
			t.Fatalf("Snapshot error = %v, want errors.Is(..., %v)", err, ErrRegistryMalformed)
		}

		resp, err := Search(context.Background(), "planner", []HubRegistryProvider{
			NewInMemoryRegistryProvider([]HubSearchResult{{Name: "planner", Description: "local fallback", Source: "hermes-index", InstallID: "planner", Score: 1}}, nil),
			provider,
		}, HubSearchOptions{})
		if err != nil {
			t.Fatalf("Search returned error: %v", err)
		}
		if resp.Evidence != HubSearchEvidenceRegistryMalformed {
			t.Fatalf("Evidence = %q, want %q", resp.Evidence, HubSearchEvidenceRegistryMalformed)
		}
		if len(resp.Results) != 1 || resp.Results[0].Description != "local fallback" {
			t.Fatalf("Results = %#v, want partial fallback result preserved", resp.Results)
		}
	})

	t.Run("skills sh status maps to typed evidence", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusTooManyRequests, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		})}
		provider := NewSkillsShRegistryProvider("plan", 10, client)
		_, err := provider.Snapshot(context.Background())
		if !errors.Is(err, ErrRegistryRateLimited) {
			t.Fatalf("Snapshot error = %v, want errors.Is(..., %v)", err, ErrRegistryRateLimited)
		}
	})
}

func TestClaudeMarketplaceProviderReadsPluginMetadata(t *testing.T) {
	requests := []string{}
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.URL.String())
		switch req.URL.String() {
		case "https://api.github.com/repos/anthropics/skills/contents/.claude-plugin/marketplace.json":
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{
				"plugins": [
					{"name":"planner","description":"Plan work","source":"./skills/planner"},
					{"name":"writer","description":"Draft prose","source":"community/writer"}
				]
			}`))}, nil
		case "https://api.github.com/repos/aiskillstore/marketplace/contents/.claude-plugin/marketplace.json":
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"plugins": []}`))}, nil
		default:
			t.Fatalf("unexpected request URL: %s", req.URL.String())
			return nil, nil
		}
	})}

	provider := NewClaudeMarketplaceRegistryProvider("plan", 10, client)
	results, err := provider.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot returned unexpected error: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %v, want both known marketplace indexes", requests)
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
	if got.Source != "claude-marketplace" {
		t.Errorf("Source = %q, want claude-marketplace", got.Source)
	}
	if got.InstallID != "anthropics/skills/skills/planner" {
		t.Errorf("InstallID = %q, want anthropics/skills/skills/planner", got.InstallID)
	}
	if got.TrustLevel != "trusted" {
		t.Errorf("TrustLevel = %q, want trusted", got.TrustLevel)
	}
}

func TestLobeHubProviderReadsAgentIndexMetadata(t *testing.T) {
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.String() != "https://chat-agents.lobehub.com/index.json" {
			t.Fatalf("unexpected request URL: %s", req.URL.String())
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{
			"agents": [
				{"identifier":"code-reviewer","meta":{"title":"Code Reviewer","description":"Reviews code for bugs and security issues","tags":["code","review"]}},
				{"identifier":"writer","meta":{"title":"Writer","description":"Drafts prose","tags":["writing"]}}
			]
		}`))}, nil
	})}

	provider := NewLobeHubRegistryProvider("code", 10, client)
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
	if got.Name != "code-reviewer" {
		t.Errorf("Name = %q, want code-reviewer", got.Name)
	}
	if got.Description != "Reviews code for bugs and security issues" {
		t.Errorf("Description = %q, want LobeHub description", got.Description)
	}
	if got.Source != "lobehub" {
		t.Errorf("Source = %q, want lobehub", got.Source)
	}
	if got.InstallID != "lobehub/code-reviewer" {
		t.Errorf("InstallID = %q, want lobehub/code-reviewer", got.InstallID)
	}
	if got.TrustLevel != "community" {
		t.Errorf("TrustLevel = %q, want community", got.TrustLevel)
	}
	if strings.Join(got.Tags, ",") != "code,review" {
		t.Errorf("Tags = %v, want [code review]", got.Tags)
	}
}

func TestMarketplaceProvidersPreservePartialResults(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.String(), "anthropics/skills") {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{
				"plugins": [{"name":"planner","description":"Plan work","source":"./skills/planner"}]
			}`))}, nil
		}
		return &http.Response{StatusCode: http.StatusBadGateway, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})}

	results, err := NewClaudeMarketplaceRegistryProvider("plan", 10, client).Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot returned unexpected error with partial marketplace data: %v", err)
	}
	if len(results) != 1 || results[0].Name != "planner" {
		t.Fatalf("results = %#v, want preserved planner result", results)
	}
}

func TestRegistryProvidersDoNotInstall(t *testing.T) {
	// Compile-time guard for the metadata-only slice: hub registry sources only
	// expose Snapshot and cannot mutate active skill stores or quarantine paths.
	var _ HubRegistryProvider = NewGitHubRegistryProvider(nil, nil)
	var _ HubRegistryProvider = NewSkillsShRegistryProvider("", 0, nil)
	var _ HubRegistryProvider = NewClaudeMarketplaceRegistryProvider("", 0, nil)
	var _ HubRegistryProvider = NewLobeHubRegistryProvider("", 0, nil)
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
