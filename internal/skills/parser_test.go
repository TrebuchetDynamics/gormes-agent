package skills

import (
	"strings"
	"testing"
)

func TestParseSkillValidDocument(t *testing.T) {
	raw := strings.Join([]string{
		"---",
		"name: dogfood",
		"description: Systematic exploratory QA testing",
		"version: 1.0.0",
		"metadata:",
		"  hermes:",
		"    tags: [qa, testing]",
		"---",
		"",
		"# Dogfood",
		"",
		"Follow the workflow carefully.",
		"",
	}, "\n")

	skill, err := Parse([]byte(raw), 8*1024)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if skill.Name != "dogfood" {
		t.Fatalf("Name = %q, want %q", skill.Name, "dogfood")
	}
	if skill.Description != "Systematic exploratory QA testing" {
		t.Fatalf("Description = %q", skill.Description)
	}
	if !strings.Contains(skill.Body, "# Dogfood") {
		t.Fatalf("Body = %q, want markdown heading", skill.Body)
	}
}

func TestParseSkillRejectsMissingRequiredHeaderFields(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "missing name",
			raw: strings.Join([]string{
				"---",
				"description: Missing a name",
				"---",
				"",
				"body",
				"",
			}, "\n"),
			want: "name",
		},
		{
			name: "missing description",
			raw: strings.Join([]string{
				"---",
				"name: nameless-doc",
				"---",
				"",
				"body",
				"",
			}, "\n"),
			want: "description",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.raw), 8*1024)
			if err == nil {
				t.Fatal("Parse returned nil error, want failure")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want mention of %q", err, tc.want)
			}
		})
	}
}
