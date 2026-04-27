package skills

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	urlSkillSource = "url"
	urlSkillTrust  = "community"
	urlSkillFile   = "SKILL.md"
)

// URLSkillEvidence is a stable degraded-mode code for direct URL skill
// candidate parsing. Callers should key retry guidance off this value instead
// of matching error text.
type URLSkillEvidence string

const (
	URLSkillEvidenceInvalidURL         URLSkillEvidence = "url_skill_invalid_url"
	URLSkillEvidenceMissingName        URLSkillEvidence = "url_skill_missing_name"
	URLSkillEvidenceInvalidName        URLSkillEvidence = "url_skill_invalid_name"
	URLSkillEvidenceInvalidFrontmatter URLSkillEvidence = "url_skill_invalid_frontmatter"
)

// URLSkillCandidate is the pure, in-memory representation of a direct
// SKILL.md URL before any quarantine, scan, or store write can occur.
type URLSkillCandidate struct {
	URL          string                    `json:"url"`
	Name         string                    `json:"name"`
	Source       string                    `json:"source"`
	Trust        string                    `json:"trust"`
	Identifier   string                    `json:"identifier"`
	Files        map[string][]byte         `json:"files,omitempty"`
	AwaitingName bool                      `json:"awaiting_name,omitempty"`
	Evidence     URLSkillEvidence          `json:"evidence,omitempty"`
	RetryHint    string                    `json:"retry_hint,omitempty"`
	Metadata     URLSkillCandidateMetadata `json:"metadata"`
}

// URLSkillCandidateMetadata mirrors the URL-specific metadata Hermes attaches
// to UrlSource bundles.
type URLSkillCandidateMetadata struct {
	URL          string `json:"url"`
	AwaitingName bool   `json:"awaiting_name"`
}

// URLSkillCandidateError preserves the typed evidence code on hard parser
// failures while keeping the returned candidate free of unsafe names or files.
type URLSkillCandidateError struct {
	Evidence URLSkillEvidence
	Detail   string
}

func (e URLSkillCandidateError) Error() string {
	if strings.TrimSpace(e.Detail) == "" {
		return string(e.Evidence)
	}
	return string(e.Evidence) + ": " + e.Detail
}

var urlSkillNameRE = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// ParseURLSkillCandidate validates a direct HTTPS SKILL.md URL and its already
// fetched document bytes without performing network calls or store writes.
func ParseURLSkillCandidate(rawURL string, skillMD []byte) (URLSkillCandidate, error) {
	trimmedURL := strings.TrimSpace(rawURL)
	candidate := URLSkillCandidate{
		URL:        trimmedURL,
		Source:     urlSkillSource,
		Trust:      urlSkillTrust,
		Identifier: trimmedURL,
		Metadata: URLSkillCandidateMetadata{
			URL: trimmedURL,
		},
	}

	parsedURL, err := parseURLSkillURL(trimmedURL)
	if err != nil {
		return urlSkillFailure(candidate, URLSkillEvidenceInvalidURL, err.Error())
	}

	frontmatter, err := parseURLSkillFrontmatter(skillMD)
	if err != nil {
		return urlSkillFailure(candidate, URLSkillEvidenceInvalidFrontmatter, err.Error())
	}

	name, awaitingName, err := resolveURLSkillName(frontmatter, parsedURL)
	if err != nil {
		return urlSkillFailure(candidate, URLSkillEvidenceInvalidName, err.Error())
	}

	candidate.Files = map[string][]byte{urlSkillFile: append([]byte(nil), skillMD...)}
	if awaitingName {
		candidate.AwaitingName = true
		candidate.Metadata.AwaitingName = true
		candidate.Evidence = URLSkillEvidenceMissingName
		candidate.RetryHint = fmt.Sprintf("gormes skills install %s --name <your-name>", trimmedURL)
		return candidate, nil
	}

	candidate.Name = name
	return candidate, nil
}

func parseURLSkillURL(rawURL string) (*url.URL, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("URL is required")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("URL must be an absolute HTTPS URL")
	}

	path := parsed.Path
	if strings.Contains(path, "\\") {
		return nil, fmt.Errorf("URL path contains a backslash")
	}
	lowerPath := strings.ToLower(path)
	if strings.Contains(lowerPath, "/.well-known/skills/") || strings.HasSuffix(strings.TrimRight(lowerPath, "/"), "/index.json") {
		return nil, fmt.Errorf("URL is handled by a registry adapter")
	}

	segments, err := urlSkillPathSegments(parsed)
	if err != nil {
		return nil, err
	}
	if len(segments) == 0 || !strings.EqualFold(segments[len(segments)-1], urlSkillFile) {
		return nil, fmt.Errorf("URL must end with %s", urlSkillFile)
	}
	return parsed, nil
}

func parseURLSkillFrontmatter(skillMD []byte) (map[string]any, error) {
	if len(skillMD) > DefaultMaxDocumentBytes {
		return nil, fmt.Errorf("skill document too large: %d > %d bytes", len(skillMD), DefaultMaxDocumentBytes)
	}

	doc := string(skillMD)
	doc = strings.TrimPrefix(doc, "\ufeff")
	doc = strings.ReplaceAll(doc, "\r\n", "\n")

	lines := strings.Split(doc, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, fmt.Errorf("skill frontmatter must start with ---")
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return nil, fmt.Errorf("skill frontmatter closing --- not found")
	}

	var frontmatter map[string]any
	rawFrontmatter := strings.Join(lines[1:end], "\n")
	if strings.TrimSpace(rawFrontmatter) == "" {
		frontmatter = map[string]any{}
	} else if err := yaml.Unmarshal([]byte(rawFrontmatter), &frontmatter); err != nil {
		return nil, err
	}
	if frontmatter == nil {
		frontmatter = map[string]any{}
	}

	if frontmatterString(frontmatter, "description") == "" {
		return nil, fmt.Errorf("skill description is required")
	}
	if strings.TrimSpace(strings.Join(lines[end+1:], "\n")) == "" {
		return nil, fmt.Errorf("skill body is required")
	}
	return frontmatter, nil
}

func resolveURLSkillName(frontmatter map[string]any, parsedURL *url.URL) (name string, awaitingName bool, err error) {
	if frontmatterName := frontmatterString(frontmatter, "name"); frontmatterName != "" {
		if !isSafeURLSkillName(frontmatterName) {
			return "", false, fmt.Errorf("unsafe frontmatter skill name %q", frontmatterName)
		}
		return frontmatterName, false, nil
	}

	segments, err := urlSkillPathSegments(parsedURL)
	if err != nil {
		return "", false, err
	}
	if len(segments) < 2 {
		return "", true, nil
	}

	slug := segments[len(segments)-2]
	if !isSafeURLSkillName(slug) {
		return "", false, fmt.Errorf("unsafe URL skill name candidate %q", slug)
	}
	return slug, false, nil
}

func urlSkillPathSegments(parsedURL *url.URL) ([]string, error) {
	if parsedURL == nil {
		return nil, fmt.Errorf("URL is required")
	}
	parts := strings.Split(parsedURL.Path, "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		if part == "." || part == ".." {
			return nil, fmt.Errorf("URL path contains traversal")
		}
		segments = append(segments, part)
	}
	return segments, nil
}

func isSafeURLSkillName(name string) bool {
	candidate := strings.TrimSpace(name)
	if candidate == "" {
		return false
	}
	switch strings.ToLower(candidate) {
	case "skill", "readme", "index", "unnamed-skill":
		return false
	}
	return urlSkillNameRE.MatchString(candidate)
}

func urlSkillFailure(candidate URLSkillCandidate, evidence URLSkillEvidence, detail string) (URLSkillCandidate, error) {
	candidate.Name = ""
	candidate.Files = nil
	candidate.AwaitingName = false
	candidate.Metadata.AwaitingName = false
	candidate.Evidence = evidence
	return candidate, URLSkillCandidateError{Evidence: evidence, Detail: detail}
}
