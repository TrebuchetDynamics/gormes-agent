package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/install"

// URLSkillEvidence is a stable degraded-mode code for direct URL skill
// candidate parsing. Callers should key retry guidance off this value instead
// of matching error text.
type URLSkillEvidence = install.URLSkillEvidence

const (
	URLSkillEvidenceInvalidURL         URLSkillEvidence = install.URLSkillEvidenceInvalidURL
	URLSkillEvidenceMissingName        URLSkillEvidence = install.URLSkillEvidenceMissingName
	URLSkillEvidenceInvalidName        URLSkillEvidence = install.URLSkillEvidenceInvalidName
	URLSkillEvidenceInvalidFrontmatter URLSkillEvidence = install.URLSkillEvidenceInvalidFrontmatter
)

// URLSkillCandidate is the pure, in-memory representation of a direct
// SKILL.md URL before any quarantine, scan, or store write can occur.
type URLSkillCandidate = install.URLSkillCandidate

// URLSkillCandidateMetadata mirrors the URL-specific metadata Hermes attaches
// to UrlSource bundles.
type URLSkillCandidateMetadata = install.URLSkillCandidateMetadata

// URLSkillCandidateError preserves the typed evidence code on hard parser
// failures while keeping the returned candidate free of unsafe names or files.
type URLSkillCandidateError = install.URLSkillCandidateError

// ParseURLSkillCandidate validates a direct HTTPS SKILL.md URL and its already
// fetched document bytes without performing network calls or store writes.
func ParseURLSkillCandidate(rawURL string, skillMD []byte) (URLSkillCandidate, error) {
	return install.ParseURLSkillCandidate(rawURL, skillMD)
}
