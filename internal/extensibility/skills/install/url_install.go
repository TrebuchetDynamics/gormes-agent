package install

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// URLFetcher fetches the bytes of a remote SKILL.md. It is the only seam
// through which network IO enters URL install policy.
type URLFetcher interface {
	Fetch(ctx context.Context, url string) ([]byte, error)
}

// QuarantineScanner runs the downloaded SKILL.md bytes through quarantine
// scan. It returns ok=false to leave the active store untouched.
type QuarantineScanner interface {
	Scan(ctx context.Context, body []byte) (ok bool, evidence string, err error)
}

// SkillStore is the minimal surface URL install policy needs to write a
// SKILL.md into the active store. It is satisfied by an in-memory test fake
// or a real on-disk implementation.
type SkillStore interface {
	ActiveDir() string
	WriteSkill(ctx context.Context, dir string, file string, body []byte) (string, error)
}

// InteractiveConsole is a placeholder for interactive prompts. nil means
// non-interactive surface — URL installs that require a name fail closed
// with retry guidance.
type InteractiveConsole interface {
	PromptName(ctx context.Context, url string) (string, bool)
	PromptCategory(ctx context.Context, existing []string) (string, bool)
}

// URLInstallPolicy bundles the seams URL install needs.
type URLInstallPolicy struct {
	Fetcher URLFetcher
	Scanner QuarantineScanner
	Store   SkillStore
	Console InteractiveConsole // nil => non-interactive
}

// URLInstallRequest is one direct-URL install request.
type URLInstallRequest struct {
	URL              string
	NameOverride     string
	CategoryOverride string
	Interactive      bool
}

// URLInstallEvidence records the outcome of a URL install attempt.
type URLInstallEvidence struct {
	Code          string
	Reason        string
	InstalledPath string
}

// Stable evidence codes for URL install outcomes.
const (
	urlSkillEvidenceInvalidCategory  = "url_skill_invalid_category"
	urlSkillEvidenceQuarantineFailed = "url_skill_quarantine_failed"
	urlSkillEvidenceInstalled        = "url_skill_installed"
)

var validCategoryRE = regexp.MustCompile(`^[a-z][a-z0-9_-]*(?:/[a-z0-9_-]+)*$`)

// PerformURLInstall is the pure URL install policy. All IO flows through
// the injected seams in p; no live network or filesystem mutation occurs
// outside the provided SkillStore.
//
// Validation order is:
//  1. NameOverride syntax (so SKILL/nested/traversal names reject before fetch)
//  2. CategoryOverride syntax (so absolute/traversal categories reject before fetch)
//  3. Fetch
//  4. Parse candidate (frontmatter validation)
//  5. Resolve final name (override > frontmatter/URL slug > error)
//  6. Quarantine/scan
//  7. Write to store
func PerformURLInstall(ctx context.Context, p URLInstallPolicy, req URLInstallRequest) URLInstallEvidence {
	if override := strings.TrimSpace(req.NameOverride); override != "" {
		if !isSafeURLSkillName(override) {
			return URLInstallEvidence{
				Code:   string(URLSkillEvidenceInvalidName),
				Reason: fmt.Sprintf("invalid --name %q: must be a lowercase identifier (letters, digits, hyphens, underscores; starts with a letter)", override),
			}
		}
	}

	if cat := strings.TrimSpace(req.CategoryOverride); cat != "" {
		if !isSafeCategory(cat) {
			return URLInstallEvidence{
				Code:   urlSkillEvidenceInvalidCategory,
				Reason: fmt.Sprintf("invalid --category %q: must be a lowercase slug (letters, digits, hyphens, underscores; optional / separators); absolute paths and traversal are rejected", cat),
			}
		}
	}

	if p.Fetcher == nil {
		return URLInstallEvidence{Code: string(URLSkillEvidenceInvalidURL), Reason: "no URL fetcher configured"}
	}
	body, err := p.Fetcher.Fetch(ctx, req.URL)
	if err != nil {
		return URLInstallEvidence{Code: string(URLSkillEvidenceInvalidURL), Reason: err.Error()}
	}

	candidate, err := ParseURLSkillCandidate(req.URL, body)
	if err != nil {
		var detail string
		if cerr, ok := err.(URLSkillCandidateError); ok {
			detail = cerr.Detail
		} else {
			detail = err.Error()
		}
		return URLInstallEvidence{Code: string(candidate.Evidence), Reason: detail}
	}

	finalName := strings.TrimSpace(req.NameOverride)
	if finalName == "" {
		if candidate.AwaitingName {
			return URLInstallEvidence{
				Code:   string(URLSkillEvidenceMissingName),
				Reason: candidate.RetryHint,
			}
		}
		finalName = candidate.Name
	}
	if finalName == "" || !isSafeURLSkillName(finalName) {
		return URLInstallEvidence{
			Code:   string(URLSkillEvidenceInvalidName),
			Reason: fmt.Sprintf("could not resolve a safe skill name for %s", req.URL),
		}
	}

	skillBody := candidate.Files[urlSkillFile]
	if len(skillBody) == 0 {
		skillBody = body
	}

	if p.Scanner != nil {
		ok, evidence, scanErr := p.Scanner.Scan(ctx, skillBody)
		if scanErr != nil {
			return URLInstallEvidence{Code: urlSkillEvidenceQuarantineFailed, Reason: scanErr.Error()}
		}
		if !ok {
			return URLInstallEvidence{Code: urlSkillEvidenceQuarantineFailed, Reason: evidence}
		}
	}

	if p.Store == nil {
		return URLInstallEvidence{Code: urlSkillEvidenceQuarantineFailed, Reason: "no skill store configured"}
	}

	dir := p.Store.ActiveDir()
	if cat := strings.TrimSpace(req.CategoryOverride); cat != "" {
		dir = filepath.Join(dir, cat)
	}
	dir = filepath.Join(dir, finalName)
	installed, err := p.Store.WriteSkill(ctx, dir, urlSkillFile, skillBody)
	if err != nil {
		return URLInstallEvidence{Code: urlSkillEvidenceQuarantineFailed, Reason: err.Error()}
	}
	return URLInstallEvidence{
		Code:          urlSkillEvidenceInstalled,
		Reason:        "installed " + finalName,
		InstalledPath: installed,
	}
}

// isSafeCategory enforces a Hermes-equivalent category rule: lowercase
// identifier-shaped segments separated by forward slashes, with no absolute
// prefix and no traversal segment.
func isSafeCategory(category string) bool {
	candidate := strings.TrimSpace(category)
	if candidate == "" {
		return false
	}
	if strings.HasPrefix(candidate, "/") {
		return false
	}
	if strings.Contains(candidate, "\\") {
		return false
	}
	for _, segment := range strings.Split(candidate, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return validCategoryRE.MatchString(candidate)
}
