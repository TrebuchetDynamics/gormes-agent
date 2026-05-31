package document

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/document/validation"

// SkillValidationCode classifies a malformed SKILL.md frontmatter problem.
// The stable code vocabulary lets callers report frontmatter defects without
// depending on parser-specific error strings.
type SkillValidationCode = validation.SkillValidationCode

const (
	SkillValidationMissingOpen      SkillValidationCode = validation.SkillValidationMissingOpen
	SkillValidationMissingClose     SkillValidationCode = validation.SkillValidationMissingClose
	SkillValidationEmptyFrontmatter SkillValidationCode = validation.SkillValidationEmptyFrontmatter
	SkillValidationYAMLParse        SkillValidationCode = validation.SkillValidationYAMLParse
	SkillValidationNullBytes        SkillValidationCode = validation.SkillValidationNullBytes
	SkillValidationNestedQuotes     SkillValidationCode = validation.SkillValidationNestedQuotes
	SkillValidationSlugMismatch     SkillValidationCode = validation.SkillValidationSlugMismatch
)

// SkillValidationError carries a single classified frontmatter defect with
// optional 1-based line evidence so callers can surface a useful pointer
// without parsing the file twice.
type SkillValidationError = validation.SkillValidationError

// FrontmatterValidateOptions opts a SKILL.md document into the more
// expensive cross-checks. SLUG_MISMATCH only fires when ExpectedSlug is set.
type FrontmatterValidateOptions = validation.FrontmatterValidateOptions

// ValidateSkillFrontmatter inspects raw SKILL.md bytes for the seven
// frontmatter validation classes that a forgiving YAML parser silently
// accepts. Returns nil when the document is well-formed.
func ValidateSkillFrontmatter(raw []byte, opts FrontmatterValidateOptions) []SkillValidationError {
	return validation.ValidateSkillFrontmatter(raw, opts)
}
