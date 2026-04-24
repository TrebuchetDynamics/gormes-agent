package skills

import (
	"strings"
	"testing"
)

func TestSkillValidateRejectsOversizedDocument(t *testing.T) {
	skill := Skill{
		Name:        "dogfood",
		Description: "Systematic exploratory QA testing",
		Body:        strings.Repeat("x", 65),
	}

	err := skill.Validate(64)
	if err == nil {
		t.Fatal("Validate returned nil error, want size failure")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("error = %q, want too-large message", err)
	}
}

func TestSkillValidateRequiresBodyMetadata(t *testing.T) {
	skill := Skill{
		Name:        "dogfood",
		Description: "Systematic exploratory QA testing",
		Body:        "body",
	}

	if err := skill.Validate(64); err != nil {
		t.Fatalf("Validate returned error for minimal valid skill: %v", err)
	}
}
