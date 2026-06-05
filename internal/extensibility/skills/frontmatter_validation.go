package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/document"

// SkillValidationCode classifies a malformed SKILL.md frontmatter problem.
// The stable code vocabulary lets callers report frontmatter defects without
// depending on parser-specific error strings.
type SkillValidationCode = document.SkillValidationCode

const (
	SkillValidationMissingOpen      SkillValidationCode = document.SkillValidationMissingOpen
	SkillValidationMissingClose     SkillValidationCode = document.SkillValidationMissingClose
	SkillValidationEmptyFrontmatter SkillValidationCode = document.SkillValidationEmptyFrontmatter
	SkillValidationYAMLParse        SkillValidationCode = document.SkillValidationYAMLParse
	SkillValidationNullBytes        SkillValidationCode = document.SkillValidationNullBytes
	SkillValidationNestedQuotes     SkillValidationCode = document.SkillValidationNestedQuotes
	SkillValidationSlugMismatch     SkillValidationCode = document.SkillValidationSlugMismatch
)

// SkillValidationError carries a single classified frontmatter defect with
// optional 1-based line evidence so callers can surface a useful pointer
// without parsing the file twice.
type SkillValidationError = document.SkillValidationError

// FrontmatterValidateOptions opts a SKILL.md document into the more
// expensive cross-checks. SLUG_MISMATCH only fires when ExpectedSlug is set.
type FrontmatterValidateOptions = document.FrontmatterValidateOptions

func ValidateSkillFrontmatter(raw []byte, opts FrontmatterValidateOptions) []SkillValidationError {
	return document.ValidateSkillFrontmatter(raw, opts)
}
