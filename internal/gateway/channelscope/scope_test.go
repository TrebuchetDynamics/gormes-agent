package channelscope

import (
	"reflect"
	"testing"
)

func TestResolveSkillsAndPrompts(t *testing.T) {
	bindings := []SkillBinding{
		{ID: "C-exact", Skills: []string{" research ", "review", "research", ""}},
		{ID: "C-parent", Skill: "parent-skill"},
	}
	if got := ResolveSkills(bindings, "C-exact", "C-parent"); !reflect.DeepEqual(got, []string{"research", "review"}) {
		t.Fatalf("exact ResolveSkills = %#v, want research/review", got)
	}
	if got := ResolveSkills(bindings, "C-thread", "C-parent"); !reflect.DeepEqual(got, []string{"parent-skill"}) {
		t.Fatalf("parent ResolveSkills = %#v, want parent-skill", got)
	}

	mixed := []any{
		"skip",
		map[string]any{"id": "C-map", "skills": []any{"alpha", 7, "beta", "alpha"}},
	}
	if got := ResolveSkills(mixed, "C-map", ""); !reflect.DeepEqual(got, []string{"alpha", "beta"}) {
		t.Fatalf("map ResolveSkills = %#v, want alpha/beta", got)
	}

	prompts := map[string]any{
		"C-exact":  " Exact prompt ",
		"C-parent": "Parent prompt",
		"C-blank":  "   ",
	}
	if got := ResolvePrompt(prompts, "C-exact", "C-parent"); got != "Exact prompt" {
		t.Fatalf("exact ResolvePrompt = %q, want Exact prompt", got)
	}
	if got := ResolvePrompt(prompts, "C-thread", "C-parent"); got != "Parent prompt" {
		t.Fatalf("parent ResolvePrompt = %q, want Parent prompt", got)
	}
	if got := ResolvePrompt(prompts, "C-blank", ""); got != "" {
		t.Fatalf("blank ResolvePrompt = %q, want empty", got)
	}
}
