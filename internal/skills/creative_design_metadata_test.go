package skills

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const claudeDesignFrontmatter = `---
name: claude-design
description: Design one-off HTML artifacts (landing, deck, prototype).
version: 1.0.0
author: BadTechBandit
license: MIT
metadata:
  hermes:
    tags: [design, html, prototype, ux, ui, creative, artifact, deck, motion, design-system]
    related_skills: [design-md, popular-web-designs, excalidraw, architecture-diagram]
---

# Claude Design for CLI/API Agents

Use this skill when the user asks for design work in a CLI/API environment.
`

const popularWebDesignsFrontmatter = `---
name: popular-web-designs
description: 54 real design systems (Stripe, Linear, Vercel) as HTML/CSS.
version: 1.0.0
author: Hermes Agent + Teknium (design systems sourced from awesome-design-md)
license: MIT
tags: [design, css, html, ui, web-development, design-systems, templates]
triggers:
  - build a page that looks like
  - make it look like stripe
  - design like linear
  - vercel style
  - landing page
---

# Popular Web Designs

54 real-world design systems ready for use when generating HTML/CSS.
`

const designMdFrontmatter = `---
name: design-md
description: Author/validate/export Google's DESIGN.md token spec files.
version: 1.0.0
author: Hermes Agent
license: MIT
metadata:
  hermes:
    tags: [design, design-system, tokens, ui, accessibility, wcag, tailwind, dtcg, google]
    related_skills: [popular-web-designs, claude-design, excalidraw, architecture-diagram]
---

# DESIGN.md Skill

Author/validate/export Google's DESIGN.md token spec files.
`

func TestCreativeSkillMetadata_ParsesClaudeDesignFrontmatter(t *testing.T) {
	skill, err := Parse([]byte(claudeDesignFrontmatter), 16*1024)
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	if skill.Name != "claude-design" {
		t.Fatalf("Name = %q, want claude-design", skill.Name)
	}
	if skill.Description != "Design one-off HTML artifacts (landing, deck, prototype)." {
		t.Fatalf("Description = %q", skill.Description)
	}
	if got := len(skill.Description); got > 60 {
		t.Fatalf("Description length = %d, want <= 60 chars", got)
	}
	if skill.Version != "1.0.0" {
		t.Fatalf("Version = %q, want 1.0.0", skill.Version)
	}
	if skill.Author != "BadTechBandit" {
		t.Fatalf("Author = %q, want BadTechBandit", skill.Author)
	}
	if skill.License != "MIT" {
		t.Fatalf("License = %q, want MIT", skill.License)
	}
	if !containsAllStrings(skill.HermesTags, []string{"design", "html", "prototype", "design-system"}) {
		t.Fatalf("HermesTags = %v, missing expected design tags", skill.HermesTags)
	}
	if !containsAllStrings(skill.RelatedSkills, []string{"design-md", "popular-web-designs"}) {
		t.Fatalf("RelatedSkills = %v, missing related design skills", skill.RelatedSkills)
	}
}

func TestCreativeSkillMetadata_DistinguishesDesignRoutes(t *testing.T) {
	root := t.TempDir()
	writeRawSkill(t, filepath.Join(root, "active", "claude-design", "SKILL.md"), claudeDesignFrontmatter)
	writeRawSkill(t, filepath.Join(root, "active", "popular-web-designs", "SKILL.md"), popularWebDesignsFrontmatter)
	writeRawSkill(t, filepath.Join(root, "active", "design-md", "SKILL.md"), designMdFrontmatter)

	runtime := NewRuntime(root, 16*1024, 1, "")

	cases := []struct {
		name  string
		query string
		want  string
	}{
		{"prototype request routes to claude-design", "design a one-off HTML prototype deck artifact", "claude-design"},
		{"stripe lookalike routes to popular-web-designs", "make it look like stripe with the linear typography", "popular-web-designs"},
		{"DESIGN.md spec routes to design-md", "author a Google DESIGN.md token spec with WCAG checks", "design-md"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, names, _, err := runtime.BuildSkillBlockWithOptions(context.Background(), tc.query, RuntimeOptions{})
			if err != nil {
				t.Fatalf("BuildSkillBlockWithOptions error = %v", err)
			}
			if len(names) != 1 {
				t.Fatalf("names = %v, want exactly one route for query %q", names, tc.query)
			}
			if names[0] != tc.want {
				t.Fatalf("query %q routed to %q, want %q (skills must remain distinct, not collapsed)", tc.query, names[0], tc.want)
			}
		})
	}
}

func TestCreativeSkillMetadata_InvalidDraftExcluded(t *testing.T) {
	root := t.TempDir()
	writeRawSkill(t, filepath.Join(root, "active", "claude-design", "SKILL.md"), claudeDesignFrontmatter)

	draftDir := filepath.Join(root, "active", "draft-design")
	if err := os.MkdirAll(draftDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	missingClose := strings.Join([]string{
		"---",
		"name: draft-design",
		"description: half-written draft",
		"# stray heading",
		"",
		"body",
	}, "\n")
	if err := os.WriteFile(filepath.Join(draftDir, "SKILL.md"), []byte(missingClose), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	runtime := NewRuntime(root, 16*1024, 5, "")
	block, names, statuses, err := runtime.BuildSkillBlockWithOptions(context.Background(), "design", RuntimeOptions{})
	if err != nil {
		t.Fatalf("BuildSkillBlockWithOptions error = %v", err)
	}
	if strings.Contains(block, "draft-design") {
		t.Fatalf("rendered block leaks invalid draft: %q", block)
	}
	for _, n := range names {
		if n == "draft-design" {
			t.Fatalf("names = %v, must not include invalid draft", names)
		}
	}
	if !anyStatusCode(statuses, SkillStatusFrontmatterInvalid) {
		t.Fatalf("statuses = %+v, want one SkillStatusFrontmatterInvalid for the draft", statuses)
	}
}

func writeRawSkill(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func containsAllStrings(haystack, needles []string) bool {
	have := map[string]bool{}
	for _, value := range haystack {
		have[value] = true
	}
	for _, want := range needles {
		if !have[want] {
			return false
		}
	}
	return true
}

func anyStatusCode(statuses []SkillStatus, code SkillStatusCode) bool {
	for _, s := range statuses {
		if s.Status == code {
			return true
		}
	}
	return false
}
