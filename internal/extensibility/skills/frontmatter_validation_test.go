package skills

import (
	"strings"
	"testing"
)

func TestSkillFrontmatterValidation_MissingOpenAndClose(t *testing.T) {
	t.Run("missing open emits MISSING_OPEN at first non-empty line", func(t *testing.T) {
		raw := strings.Join([]string{
			"",
			"",
			"# Heading without frontmatter",
			"",
			"body",
		}, "\n")

		errs := ValidateSkillFrontmatter([]byte(raw), FrontmatterValidateOptions{})

		if got := codeOf(errs, SkillValidationMissingOpen); got == nil {
			t.Fatalf("errors = %+v, want a MISSING_OPEN entry", errs)
		} else if got.Line != 3 {
			t.Fatalf("MISSING_OPEN.Line = %d, want 3 (first non-empty)", got.Line)
		}

		if codeOf(errs, SkillValidationMissingClose) != nil {
			t.Fatalf("errors = %+v, want no MISSING_CLOSE when MISSING_OPEN already fired", errs)
		}
	})

	t.Run("empty file emits MISSING_OPEN", func(t *testing.T) {
		errs := ValidateSkillFrontmatter([]byte("\n\n"), FrontmatterValidateOptions{})
		if codeOf(errs, SkillValidationMissingOpen) == nil {
			t.Fatalf("errors = %+v, want MISSING_OPEN for empty file", errs)
		}
	})

	t.Run("body heading before close emits MISSING_CLOSE", func(t *testing.T) {
		raw := strings.Join([]string{
			"---",
			"name: half-frontmatter",
			"# Heading without close",
			"",
			"body",
		}, "\n")

		errs := ValidateSkillFrontmatter([]byte(raw), FrontmatterValidateOptions{})

		got := codeOf(errs, SkillValidationMissingClose)
		if got == nil {
			t.Fatalf("errors = %+v, want a MISSING_CLOSE entry", errs)
		}
		if got.Line != 3 {
			t.Fatalf("MISSING_CLOSE.Line = %d, want 3 (heading line)", got.Line)
		}
	})
}

func TestSkillFrontmatterValidation_EmptyAndYAMLParse(t *testing.T) {
	t.Run("empty frontmatter block", func(t *testing.T) {
		raw := strings.Join([]string{
			"---",
			"---",
			"",
			"body",
		}, "\n")

		errs := ValidateSkillFrontmatter([]byte(raw), FrontmatterValidateOptions{})
		if codeOf(errs, SkillValidationEmptyFrontmatter) == nil {
			t.Fatalf("errors = %+v, want EMPTY_FRONTMATTER", errs)
		}
	})

	t.Run("malformed YAML emits YAML_PARSE", func(t *testing.T) {
		raw := strings.Join([]string{
			"---",
			"name: broken",
			"description: [unterminated",
			"---",
			"",
			"body",
		}, "\n")

		errs := ValidateSkillFrontmatter([]byte(raw), FrontmatterValidateOptions{})
		if codeOf(errs, SkillValidationYAMLParse) == nil {
			t.Fatalf("errors = %+v, want YAML_PARSE for unterminated value", errs)
		}
	})
}

func TestSkillFrontmatterValidation_NullBytesAndNestedQuotes(t *testing.T) {
	t.Run("null bytes record line evidence", func(t *testing.T) {
		raw := "---\nname: tainted\n\x00garbage\n---\n\nbody"

		errs := ValidateSkillFrontmatter([]byte(raw), FrontmatterValidateOptions{})

		got := codeOf(errs, SkillValidationNullBytes)
		if got == nil {
			t.Fatalf("errors = %+v, want NULL_BYTES", errs)
		}
		if got.Line != 3 {
			t.Fatalf("NULL_BYTES.Line = %d, want 3 (line containing the null byte)", got.Line)
		}
	})

	t.Run("nested double quotes in scalar value", func(t *testing.T) {
		raw := strings.Join([]string{
			"---",
			"name: nicknamer",
			`description: "Phil "Nick" Last"`,
			"---",
			"",
			"body",
		}, "\n")

		errs := ValidateSkillFrontmatter([]byte(raw), FrontmatterValidateOptions{})
		got := codeOf(errs, SkillValidationNestedQuotes)
		if got == nil {
			t.Fatalf("errors = %+v, want NESTED_QUOTES", errs)
		}
		if got.Line != 3 {
			t.Fatalf("NESTED_QUOTES.Line = %d, want 3 (description line)", got.Line)
		}
	})
}

func TestSkillFrontmatterValidation_SlugMismatch(t *testing.T) {
	raw := strings.Join([]string{
		"---",
		"name: review-tests",
		"description: Slug differs from path",
		"slug: not-the-directory",
		"---",
		"",
		"body",
	}, "\n")

	t.Run("declared slug differs from expected", func(t *testing.T) {
		errs := ValidateSkillFrontmatter([]byte(raw), FrontmatterValidateOptions{ExpectedSlug: "review-tests"})
		if codeOf(errs, SkillValidationSlugMismatch) == nil {
			t.Fatalf("errors = %+v, want SLUG_MISMATCH", errs)
		}
	})

	t.Run("declared slug matches expected", func(t *testing.T) {
		matching := strings.ReplaceAll(raw, "not-the-directory", "review-tests")
		errs := ValidateSkillFrontmatter([]byte(matching), FrontmatterValidateOptions{ExpectedSlug: "review-tests"})
		if codeOf(errs, SkillValidationSlugMismatch) != nil {
			t.Fatalf("errors = %+v, want no SLUG_MISMATCH when slugs match", errs)
		}
	})

	t.Run("missing expected slug skips check", func(t *testing.T) {
		errs := ValidateSkillFrontmatter([]byte(raw), FrontmatterValidateOptions{})
		if codeOf(errs, SkillValidationSlugMismatch) != nil {
			t.Fatalf("errors = %+v, want no SLUG_MISMATCH when ExpectedSlug is empty", errs)
		}
	})
}

func TestSkillFrontmatterValidation_ValidDocumentReportsNoErrors(t *testing.T) {
	raw := strings.Join([]string{
		"---",
		"name: clean",
		"description: A valid skill",
		"---",
		"",
		"body",
	}, "\n")

	errs := ValidateSkillFrontmatter([]byte(raw), FrontmatterValidateOptions{ExpectedSlug: "clean"})
	if len(errs) != 0 {
		t.Fatalf("errors = %+v, want none for a clean document", errs)
	}
}

func codeOf(errs []SkillValidationError, code SkillValidationCode) *SkillValidationError {
	for i := range errs {
		if errs[i].Code == code {
			return &errs[i]
		}
	}
	return nil
}
