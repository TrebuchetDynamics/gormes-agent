package skills

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
)

const wellKnownSkillsBasePath = "/.well-known/skills"

var trustedSkillRepos = map[string]bool{
	"openai/skills":     true,
	"anthropics/skills": true,
}

type GitHubRegistryTap struct {
	Repo string
	Path string
}

// GitHubRegistryProvider reads SKILL.md metadata from configured GitHub taps
// through the Contents API. It mirrors Hermes' GitHubSource search-side
// contract for Gormes' metadata-only hub registry slice: list candidate skill
// directories, fetch SKILL.md frontmatter for preview metadata, and never
// mutate the active skill store or quarantine/install directories.
type GitHubRegistryProvider struct {
	taps   []GitHubRegistryTap
	client *http.Client
}

func NewGitHubRegistryProvider(taps []GitHubRegistryTap, client *http.Client) *GitHubRegistryProvider {
	clean := make([]GitHubRegistryTap, 0, len(taps))
	for _, tap := range taps {
		repo := strings.Trim(strings.TrimSpace(tap.Repo), "/")
		if repo == "" {
			continue
		}
		clean = append(clean, GitHubRegistryTap{Repo: repo, Path: strings.Trim(strings.TrimSpace(tap.Path), "/")})
	}
	return &GitHubRegistryProvider{taps: clean, client: client}
}

func (p *GitHubRegistryProvider) Snapshot(ctx context.Context) ([]HubSearchResult, error) {
	if p == nil || len(p.taps) == 0 {
		return nil, ErrRegistryUnavailable
	}
	client := p.client
	if client == nil {
		client = http.DefaultClient
	}
	var (
		results []HubSearchResult
		lastErr error
	)
	for _, tap := range p.taps {
		entries, err := githubContents(ctx, client, tap.Repo, tap.Path)
		if err != nil {
			lastErr = err
			continue
		}
		for _, entry := range entries {
			if entry.Type != "dir" || strings.HasPrefix(entry.Name, ".") || strings.HasPrefix(entry.Name, "_") {
				continue
			}
			skillPath := strings.Trim(strings.TrimRight(tap.Path, "/")+"/"+entry.Name, "/")
			meta, err := githubSkillMetadata(ctx, client, tap.Repo, skillPath)
			if err != nil {
				lastErr = err
				continue
			}
			results = append(results, meta)
		}
	}
	if len(results) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return results, nil
}

type githubContentEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func githubContents(ctx context.Context, client *http.Client, repo, path string) ([]githubContentEntry, error) {
	reqURL := "https://api.github.com/repos/" + repo + "/contents"
	if strings.TrimSpace(path) != "" {
		reqURL += "/" + strings.Trim(path, "/")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistryUnavailable, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistryUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden {
		return nil, ErrRegistryRateLimited
	}
	if resp.StatusCode != http.StatusOK {
		return nil, ErrRegistryUnavailable
	}
	var entries []githubContentEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistryMalformed, err)
	}
	return entries, nil
}

func githubSkillMetadata(ctx context.Context, client *http.Client, repo, skillPath string) (HubSearchResult, error) {
	skillPath = strings.Trim(skillPath, "/")
	reqURL := "https://api.github.com/repos/" + repo + "/contents/" + skillPath + "/SKILL.md"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return HubSearchResult{}, fmt.Errorf("%w: %v", ErrRegistryUnavailable, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return HubSearchResult{}, fmt.Errorf("%w: %v", ErrRegistryUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden {
		return HubSearchResult{}, ErrRegistryRateLimited
	}
	if resp.StatusCode != http.StatusOK {
		return HubSearchResult{}, ErrRegistryUnavailable
	}
	var file struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return HubSearchResult{}, fmt.Errorf("%w: %v", ErrRegistryMalformed, err)
	}
	if file.Encoding != "base64" {
		return HubSearchResult{}, ErrRegistryMalformed
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(file.Content, "\n", ""))
	if err != nil {
		return HubSearchResult{}, fmt.Errorf("%w: %v", ErrRegistryMalformed, err)
	}
	skill, err := Parse(raw, DefaultMaxDocumentBytes)
	if err != nil {
		return HubSearchResult{}, fmt.Errorf("%w: %v", ErrRegistryMalformed, err)
	}
	identifier := repo + "/" + skillPath
	return HubSearchResult{
		Name:        skill.Name,
		Description: skill.Description,
		Source:      "github",
		InstallID:   identifier,
		TrustLevel:  githubTrustLevel(identifier),
		Tags:        append([]string(nil), skill.HermesTags...),
	}, nil
}

// SkillsShRegistryProvider reads skills.sh search metadata as registry preview
// data. Fetch/install resolution remains out of scope for this slice.
type SkillsShRegistryProvider struct {
	query  string
	limit  int
	client *http.Client
}

func NewSkillsShRegistryProvider(query string, limit int, client *http.Client) *SkillsShRegistryProvider {
	return &SkillsShRegistryProvider{query: strings.TrimSpace(query), limit: limit, client: client}
}

func (p *SkillsShRegistryProvider) Snapshot(ctx context.Context) ([]HubSearchResult, error) {
	if p == nil || p.query == "" {
		return nil, ErrRegistryUnavailable
	}
	client := p.client
	if client == nil {
		client = http.DefaultClient
	}
	limit := p.limit
	if limit <= 0 {
		limit = 10
	}
	u, _ := url.Parse("https://skills.sh/api/search")
	q := u.Query()
	q.Set("q", p.query)
	q.Set("limit", strconv.Itoa(limit))
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
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
	var data struct {
		Skills []map[string]any `json:"skills"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistryMalformed, err)
	}
	results := make([]HubSearchResult, 0, len(data.Skills))
	for _, item := range data.Skills {
		result, ok := skillsShResult(item)
		if ok {
			results = append(results, result)
		}
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func skillsShResult(item map[string]any) (HubSearchResult, bool) {
	canonical := asString(item["id"])
	repo := asString(item["source"])
	skillPath := asString(item["skillId"])
	if strings.Count(canonical, "/") < 2 {
		if repo == "" || skillPath == "" {
			return HubSearchResult{}, false
		}
		canonical = repo + "/" + skillPath
	}
	parts := strings.SplitN(canonical, "/", 3)
	if len(parts) < 3 {
		return HubSearchResult{}, false
	}
	repo = parts[0] + "/" + parts[1]
	skillPath = parts[2]
	name := firstNonEmpty(asString(item["name"]), pathBase(skillPath))
	desc := "Indexed by skills.sh from " + repo
	if installs, ok := numberAsInt(item["installs"]); ok {
		desc += " · " + strconv.FormatInt(int64(installs), 10) + " installs"
		desc = strings.ReplaceAll(desc, strconv.FormatInt(int64(installs), 10), commaInt(installs))
	}
	return HubSearchResult{
		Name:        name,
		Description: desc,
		Source:      "skills.sh",
		InstallID:   "skills-sh:" + canonical,
		TrustLevel:  githubTrustLevel(canonical),
	}, true
}

func githubTrustLevel(identifier string) string {
	parts := strings.SplitN(identifier, "/", 3)
	if len(parts) >= 2 && trustedSkillRepos[parts[0]+"/"+parts[1]] {
		return "trusted"
	}
	return "community"
}

func pathBase(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func numberAsInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func commaInt(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(ch))
	}
	return string(out)
}

// HermesIndexRegistryProvider reads the centralized Hermes skills index from a
// caller-supplied cache path. It mirrors Hermes' HermesIndexSource search-side
// contract for Gormes' read-only hub registry layer: no remote API calls, no
// active skill-store writes, and typed degraded evidence for missing or
// malformed cache files.
type HermesIndexRegistryProvider struct {
	cachePath string
}

// NewHermesIndexRegistryProvider returns a read-only centralized-index
// provider. cachePath must be injected by callers/tests; the provider does not
// hard-code operator-specific Hermes/Gormes homes.
func NewHermesIndexRegistryProvider(cachePath string) *HermesIndexRegistryProvider {
	return &HermesIndexRegistryProvider{cachePath: strings.TrimSpace(cachePath)}
}

// Snapshot converts cached centralized index metadata into normalized hub
// search results. Empty or unnamed entries are skipped; a syntactically valid
// empty index returns an empty result set without error.
func (p *HermesIndexRegistryProvider) Snapshot(ctx context.Context) ([]HubSearchResult, error) {
	if p == nil || p.cachePath == "" {
		return nil, ErrRegistryUnavailable
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	raw, err := os.ReadFile(p.cachePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistryUnavailable, err)
	}
	var index struct {
		Skills []struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Source      string   `json:"source"`
			Identifier  string   `json:"identifier"`
			TrustLevel  string   `json:"trust_level"`
			Tags        []string `json:"tags"`
			Score       float64  `json:"score"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistryMalformed, err)
	}
	results := make([]HubSearchResult, 0, len(index.Skills))
	for _, skill := range index.Skills {
		name := strings.TrimSpace(skill.Name)
		if name == "" {
			continue
		}
		results = append(results, HubSearchResult{
			Name:        name,
			Description: skill.Description,
			Source:      firstNonEmpty(skill.Source, "hermes-index"),
			InstallID:   strings.TrimSpace(skill.Identifier),
			Score:       skill.Score,
			TrustLevel:  firstNonEmpty(skill.TrustLevel, "community"),
			Tags:        append([]string(nil), skill.Tags...),
		})
	}
	return results, nil
}

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

// ClawHubRegistryProvider reads ClawHub catalog metadata through the same
// read-only HubRegistryProvider seam as other registry sources. Hermes treats
// all ClawHub entries as community-trust metadata because the source is not a
// trusted install path; Gormes keeps this provider metadata-only and never
// mutates active skill stores or quarantine directories.
type ClawHubRegistryProvider struct {
	baseURL string
	client  *http.Client
}

// NewClawHubRegistryProvider returns a read-only provider for a ClawHub API
// root. baseURL may be a site root such as https://clawhub.ai or an API root
// such as https://clawhub.ai/api/v1.
func NewClawHubRegistryProvider(baseURL string, client *http.Client) *ClawHubRegistryProvider {
	return &ClawHubRegistryProvider{baseURL: strings.TrimSpace(baseURL), client: client}
}

// Snapshot fetches ClawHub listing metadata and converts it into normalized hub
// search results. The provider deliberately does not fetch skill bundles or
// inspect installable files; install flows belong to later slices.
func (p *ClawHubRegistryProvider) Snapshot(ctx context.Context) ([]HubSearchResult, error) {
	if p == nil || p.baseURL == "" {
		return nil, ErrRegistryUnavailable
	}
	client := p.client
	if client == nil {
		client = http.DefaultClient
	}

	listURL, err := clawHubSkillsURL(p.baseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistryUnavailable, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
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

	var data any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistryMalformed, err)
	}
	items, ok := clawHubItems(data)
	if !ok {
		return nil, ErrRegistryMalformed
	}

	results := make([]HubSearchResult, 0, len(items))
	for _, item := range items {
		slug := strings.TrimSpace(asString(item["slug"]))
		if slug == "" {
			continue
		}
		name := firstNonEmpty(asString(item["displayName"]), asString(item["name"]), slug)
		description := firstNonEmpty(asString(item["summary"]), asString(item["description"]))
		results = append(results, HubSearchResult{
			Name:        name,
			Description: description,
			Source:      "clawhub",
			InstallID:   "clawhub:" + slug,
			TrustLevel:  "community",
			Tags:        normalizeClawHubTags(item["tags"]),
		})
	}
	return results, nil
}

func clawHubSkillsURL(raw string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return "", ErrRegistryUnavailable
	}
	if !strings.Contains(trimmed, "/api/v1") {
		trimmed += "/api/v1"
	}
	if !strings.HasSuffix(trimmed, "/skills") {
		trimmed += "/skills"
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}
	q := u.Query()
	if q.Get("limit") == "" {
		q.Set("limit", "100")
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func clawHubItems(data any) ([]map[string]any, bool) {
	var rawItems []any
	switch v := data.(type) {
	case []any:
		rawItems = v
	case map[string]any:
		switch items := v["items"].(type) {
		case []any:
			rawItems = items
		default:
			return nil, false
		}
	default:
		return nil, false
	}
	items := make([]map[string]any, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		items = append(items, item)
	}
	return items, true
}

func normalizeClawHubTags(tags any) []string {
	var out []string
	switch v := tags.(type) {
	case []any:
		for _, tag := range v {
			if s := strings.TrimSpace(asString(tag)); s != "" {
				out = append(out, s)
			}
		}
	case []string:
		for _, tag := range v {
			if s := strings.TrimSpace(tag); s != "" {
				out = append(out, s)
			}
		}
	case map[string]any:
		for key := range v {
			if key != "latest" && strings.TrimSpace(key) != "" {
				out = append(out, key)
			}
		}
	}
	sort.Strings(out)
	return out
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
