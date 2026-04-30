package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const wellKnownSkillsBasePath = "/.well-known/skills"

// WellKnownRegistryProvider reads metadata from a domain exposing
// /.well-known/skills/index.json. It mirrors Hermes' WellKnownSkillSource
// read-only search contract: metadata is community-trust registry data and no
// active skill store, quarantine, or install path is mutated.
type WellKnownRegistryProvider struct {
	baseURL string
	client  *http.Client
}

// NewWellKnownRegistryProvider returns a read-only provider for a well-known
// skills endpoint. baseURL may be a site root or the index.json URL.
func NewWellKnownRegistryProvider(baseURL string, client *http.Client) *WellKnownRegistryProvider {
	return &WellKnownRegistryProvider{baseURL: strings.TrimSpace(baseURL), client: client}
}

// Snapshot fetches and converts index metadata into hub search results.
func (p *WellKnownRegistryProvider) Snapshot(ctx context.Context) ([]HubSearchResult, error) {
	if p == nil || p.baseURL == "" {
		return nil, ErrRegistryUnavailable
	}
	client := p.client
	if client == nil {
		client = http.DefaultClient
	}

	indexURL := wellKnownIndexURL(p.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistryUnavailable, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistryUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRegistryRateLimited
	}
	if resp.StatusCode != http.StatusOK {
		return nil, ErrRegistryUnavailable
	}

	var index struct {
		Skills []struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
		} `json:"skills"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistryMalformed, err)
	}

	base := strings.TrimSuffix(indexURL, "/index.json")
	results := make([]HubSearchResult, 0, len(index.Skills))
	for _, skill := range index.Skills {
		name := strings.TrimSpace(skill.Name)
		if name == "" {
			continue
		}
		results = append(results, HubSearchResult{
			Name:        name,
			Description: skill.Description,
			Source:      "well-known",
			InstallID:   "well-known:" + base + "/" + name,
			TrustLevel:  "community",
			Tags:        append([]string(nil), skill.Tags...),
		})
	}
	return results, nil
}

func wellKnownIndexURL(raw string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	if strings.HasSuffix(trimmed, "/index.json") {
		return trimmed
	}
	if strings.Contains(trimmed, wellKnownSkillsBasePath+"/") {
		prefix := strings.SplitN(trimmed, wellKnownSkillsBasePath+"/", 2)[0]
		return prefix + wellKnownSkillsBasePath + "/index.json"
	}
	if strings.HasSuffix(trimmed, wellKnownSkillsBasePath) {
		return trimmed + "/index.json"
	}
	return trimmed + wellKnownSkillsBasePath + "/index.json"
}
