package skills

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillValidationCode classifies a malformed SKILL.md frontmatter problem.
// Codes mirror GBrain v0.22.4 src/core/markdown.ts ParseValidationCode so the
// same triage vocabulary applies across markdown corpora.
type SkillValidationCode string

const (
	SkillValidationMissingOpen      SkillValidationCode = "MISSING_OPEN"
	SkillValidationMissingClose     SkillValidationCode = "MISSING_CLOSE"
	SkillValidationEmptyFrontmatter SkillValidationCode = "EMPTY_FRONTMATTER"
	SkillValidationYAMLParse        SkillValidationCode = "YAML_PARSE"
	SkillValidationNullBytes        SkillValidationCode = "NULL_BYTES"
	SkillValidationNestedQuotes     SkillValidationCode = "NESTED_QUOTES"
	SkillValidationSlugMismatch     SkillValidationCode = "SLUG_MISMATCH"
)

// SkillValidationError carries a single classified frontmatter defect with
// optional 1-based line evidence so callers can surface a useful pointer
// without parsing the file twice.
type SkillValidationError struct {
	Code    SkillValidationCode
	Message string
	Line    int
}

// FrontmatterValidateOptions opts a SKILL.md document into the more
// expensive cross-checks. SLUG_MISMATCH only fires when ExpectedSlug is set.
type FrontmatterValidateOptions struct {
	ExpectedSlug string
}

var (
	frontmatterKeyValuePattern = regexp.MustCompile(`^\s*[A-Za-z_][\w-]*\s*:\s*(.*)$`)
	headingPattern             = regexp.MustCompile(`^#{1,6}\s`)
)

// ValidateSkillFrontmatter inspects raw SKILL.md bytes for the seven
// frontmatter validation classes that a forgiving YAML parser silently
// accepts. Returns nil when the document is well-formed.
//
// Order of checks is deliberate: cheap byte-level (NULL_BYTES) first, then
// structural (MISSING_OPEN, MISSING_CLOSE, EMPTY_FRONTMATTER), then
// value-shape (NESTED_QUOTES), then YAML parse, then slug cross-check.
func ValidateSkillFrontmatter(raw []byte, opts FrontmatterValidateOptions) []SkillValidationError {
	content := string(raw)
	content = strings.TrimPrefix(content, "\uFEFF")
	content = strings.ReplaceAll(content, "\r\n", "\n")

	var errs []SkillValidationError

	if idx := strings.IndexByte(content, 0); idx >= 0 {
		line := strings.Count(content[:idx], "\n") + 1
		errs = append(errs, SkillValidationError{
			Code:    SkillValidationNullBytes,
			Message: "content contains null bytes (likely binary corruption)",
			Line:    line,
		})
	}

	lines := strings.Split(content, "\n")
	firstNonEmpty := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			firstNonEmpty = i
			break
		}
	}
	if firstNonEmpty == -1 {
		errs = append(errs, SkillValidationError{
			Code:    SkillValidationMissingOpen,
			Message: "file is empty or whitespace-only; expected frontmatter starting with ---",
			Line:    1,
		})
		return errs
	}
	if strings.TrimSpace(lines[firstNonEmpty]) != "---" {
		errs = append(errs, SkillValidationError{
			Code:    SkillValidationMissingOpen,
			Message: "frontmatter must start with --- on the first non-empty line",
			Line:    firstNonEmpty + 1,
		})
		return errs
	}

	closeLine := -1
	headingBeforeClose := -1
	for i := firstNonEmpty + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" {
			closeLine = i
			break
		}
		if headingBeforeClose == -1 && headingPattern.MatchString(trimmed) {
			headingBeforeClose = i
		}
	}
	if closeLine == -1 {
		message := "no closing --- delimiter found"
		evidenceLine := firstNonEmpty + 1
		if headingBeforeClose >= 0 {
			message = fmt.Sprintf("no closing --- before heading at line %d", headingBeforeClose+1)
			evidenceLine = headingBeforeClose + 1
		}
		errs = append(errs, SkillValidationError{
			Code:    SkillValidationMissingClose,
			Message: message,
			Line:    evidenceLine,
		})
		return errs
	}
	if headingBeforeClose >= 0 && headingBeforeClose < closeLine {
		errs = append(errs, SkillValidationError{
			Code:    SkillValidationMissingClose,
			Message: fmt.Sprintf("heading at line %d found inside frontmatter zone (closing --- comes after)", headingBeforeClose+1),
			Line:    headingBeforeClose + 1,
		})
	}

	frontmatterLines := lines[firstNonEmpty+1 : closeLine]
	frontmatterBody := strings.Join(frontmatterLines, "\n")
	if strings.TrimSpace(frontmatterBody) == "" {
		errs = append(errs, SkillValidationError{
			Code:    SkillValidationEmptyFrontmatter,
			Message: "frontmatter block is empty",
			Line:    firstNonEmpty + 1,
		})
	}

	for offset, line := range frontmatterLines {
		match := frontmatterKeyValuePattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		if countUnescapedDoubleQuotes(match[1]) >= 3 {
			errs = append(errs, SkillValidationError{
				Code:    SkillValidationNestedQuotes,
				Message: "nested double quotes in YAML value (use single quotes for the outer)",
				Line:    firstNonEmpty + 1 + offset + 1,
			})
		}
	}

	if strings.TrimSpace(frontmatterBody) != "" {
		var probe map[string]any
		if err := yaml.Unmarshal([]byte(frontmatterBody), &probe); err != nil {
			errs = append(errs, SkillValidationError{
				Code:    SkillValidationYAMLParse,
				Message: "yaml parse failed: " + err.Error(),
				Line:    firstNonEmpty + 1,
			})
		} else if expected := strings.TrimSpace(opts.ExpectedSlug); expected != "" {
			if declared, ok := probe["slug"].(string); ok {
				if strings.TrimSpace(declared) != expected {
					errs = append(errs, SkillValidationError{
						Code:    SkillValidationSlugMismatch,
						Message: fmt.Sprintf("frontmatter slug %q does not match path-derived slug %q", declared, expected),
						Line:    firstNonEmpty + 1,
					})
				}
			}
		}
	}

	return errs
}

func countUnescapedDoubleQuotes(value string) int {
	count := 0
	for i := 0; i < len(value); i++ {
		if value[i] != '"' {
			continue
		}
		if i > 0 && value[i-1] == '\\' {
			continue
		}
		count++
	}
	return count
}
