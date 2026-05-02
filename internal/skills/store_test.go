package skills

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestSkillStoreSnapshotLoadsOnlyActiveSkills(t *testing.T) {
	root := t.TempDir()
	writeSkillDoc(t, filepath.Join(root, "active", "careful-review", "SKILL.md"), "careful-review", "Review carefully", "Follow the review checklist.")
	writeSkillDoc(t, filepath.Join(root, "candidates", "cand-1", "SKILL.md"), "candidate-only", "Should stay inactive", "Do not load me.")

	store := NewStore(root, 8*1024)
	snap, err := store.SnapshotActive()
	if err != nil {
		t.Fatalf("SnapshotActive() error = %v", err)
	}
	if len(snap.Skills) != 1 {
		t.Fatalf("len(Skills) = %d, want 1", len(snap.Skills))
	}
	if snap.Skills[0].Name != "careful-review" {
		t.Fatalf("Skills[0].Name = %q, want %q", snap.Skills[0].Name, "careful-review")
	}
}

func TestSkillStoreSnapshotIsImmutableAfterLoad(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "active", "careful-review", "SKILL.md")
	writeSkillDoc(t, path, "careful-review", "Review carefully", "Follow the review checklist.")

	store := NewStore(root, 8*1024)
	snap, err := store.SnapshotActive()
	if err != nil {
		t.Fatalf("SnapshotActive() error = %v", err)
	}

	writeSkillDoc(t, path, "careful-review-v2", "Review even more carefully", "Follow the v2 checklist.")

	if got := snap.Skills[0].Description; got != "Review carefully" {
		t.Fatalf("snapshot description mutated to %q, want original value", got)
	}
	if got := snap.Skills[0].Body; got != "Follow the review checklist." {
		t.Fatalf("snapshot body mutated to %q, want original value", got)
	}

	fresh, err := store.SnapshotActive()
	if err != nil {
		t.Fatalf("SnapshotActive() fresh error = %v", err)
	}
	if len(fresh.Skills) != 1 || fresh.Skills[0].Name != "careful-review-v2" {
		t.Fatalf("fresh snapshot = %#v, want updated skill", fresh.Skills)
	}
}

func TestRuntimeBuildSkillBlockSelectsAndRendersActiveSkills(t *testing.T) {
	root := t.TempDir()
	writeSkillDoc(t, filepath.Join(root, "active", "careful-review", "SKILL.md"), "careful-review", "Review carefully", "Follow the review checklist.")
	writeSkillDoc(t, filepath.Join(root, "active", "review-tests", "SKILL.md"), "review-tests", "Review tests and failure modes", "Check assertions before implementation.")
	writeSkillDoc(t, filepath.Join(root, "candidates", "cand-1", "SKILL.md"), "candidate-only", "Should stay inactive", "Do not load me.")

	runtime := NewRuntime(root, 8*1024, 2, "")
	block, names, err := runtime.BuildSkillBlock(context.Background(), "please review this carefully and check tests")
	if err != nil {
		t.Fatalf("BuildSkillBlock() error = %v", err)
	}

	wantNames := []string{"review-tests", "careful-review"}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("names = %#v, want %#v", names, wantNames)
	}
	wantBlock := "<skills>\n## review-tests\nReview tests and failure modes\n\nCheck assertions before implementation.\n\n## careful-review\nReview carefully\n\nFollow the review checklist.\n</skills>"
	if block != wantBlock {
		t.Fatalf("BuildSkillBlock() = %q, want %q", block, wantBlock)
	}
}

func TestRuntimeBuildSkillBlockToolsetConditions(t *testing.T) {
	root := t.TempDir()
	writeSkillDocWithFrontmatter(t, filepath.Join(root, "active", "fallback-terminal", "SKILL.md"), `name: fallback-terminal
description: Use when terminal is unavailable
metadata:
  hermes:
    fallback_for_tools: [terminal]
`, "Fallback body.")
	writeSkillDocWithFrontmatter(t, filepath.Join(root, "active", "needs-browser", "SKILL.md"), `name: needs-browser
description: Use when browser tools are enabled
metadata:
  hermes:
    requires_toolsets: [browser]
`, "Browser body.")
	writeSkillDocWithFrontmatter(t, filepath.Join(root, "active", "always", "SKILL.md"), `name: always
description: Always visible
`, "Always body.")

	runtime := NewRuntime(root, 8*1024, 10, "")
	_, names, statuses, err := runtime.BuildSkillBlockWithOptions(context.Background(), "always browser", RuntimeOptions{
		AvailableTools:    []string{"terminal"},
		AvailableToolsets: []string{"browser"},
	})
	if err != nil {
		t.Fatalf("BuildSkillBlockWithOptions() error = %v", err)
	}

	wantNames := []string{"always", "needs-browser"}
	if !sameStrings(names, wantNames) {
		t.Fatalf("names = %#v, want %#v", names, wantNames)
	}
	assertSkillStatus(t, statuses, "fallback-terminal", SkillStatusConditionExcluded)
	assertSkillStatus(t, statuses, "needs-browser", SkillStatusAvailable)

	_, names, _, err = runtime.BuildSkillBlockWithOptions(context.Background(), "always browser terminal unavailable", RuntimeOptions{})
	if err != nil {
		t.Fatalf("BuildSkillBlockWithOptions() with nil filters error = %v", err)
	}
	wantNames = []string{"always", "fallback-terminal", "needs-browser"}
	if !sameStrings(names, wantNames) {
		t.Fatalf("names with nil filters = %#v, want %#v", names, wantNames)
	}
}

func TestRuntimeBuildSkillBlockHonorsAgentAllowlist(t *testing.T) {
	root := t.TempDir()
	writeSkillDoc(t, filepath.Join(root, "active", "main-skill", "SKILL.md"), "main-skill", "Main skill", "Main-only instructions.")
	writeSkillDoc(t, filepath.Join(root, "active", "alerts-skill", "SKILL.md"), "alerts-skill", "Alerts skill", "Alerts-only instructions.")
	writeSkillDoc(t, filepath.Join(root, "active", "disabled-skill", "SKILL.md"), "disabled-skill", "Disabled skill", "Disabled instructions.")

	runtime := NewRuntime(root, 8*1024, 10, "")
	block, names, statuses, err := runtime.BuildSkillBlockWithOptions(context.Background(), "skill instructions", RuntimeOptions{
		AllowedSkillNames:  map[string]bool{"alerts-skill": true, "disabled-skill": true},
		DisabledSkillNames: map[string]bool{"disabled-skill": true},
	})
	if err != nil {
		t.Fatalf("BuildSkillBlockWithOptions() error = %v", err)
	}
	if !reflect.DeepEqual(names, []string{"alerts-skill"}) {
		t.Fatalf("names = %#v, want alerts skill only", names)
	}
	if !strings.Contains(block, "Alerts-only instructions.") || strings.Contains(block, "Main-only instructions.") || strings.Contains(block, "Disabled instructions.") {
		t.Fatalf("agent allowlist block not isolated:\n%s", block)
	}
	assertSkillStatus(t, statuses, "main-skill", SkillStatusPolicyExcluded)
	assertSkillStatus(t, statuses, "disabled-skill", SkillStatusDisabled)
}

func writeSkillDoc(t *testing.T, path, name, description, body string) {
	t.Helper()
	writeSkillDocWithFrontmatter(t, path, "name: "+name+"\ndescription: "+description, body)
}

func writeSkillDocWithFrontmatter(t *testing.T, path, frontmatter, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	raw := "---\n" + strings.TrimSpace(frontmatter) + "\n---\n\n" + body
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func sameStrings(got, want []string) bool {
	gotCopy := append([]string(nil), got...)
	wantCopy := append([]string(nil), want...)
	sort.Strings(gotCopy)
	sort.Strings(wantCopy)
	return reflect.DeepEqual(gotCopy, wantCopy)
}

func assertSkillStatus(t *testing.T, statuses []SkillStatus, name string, want SkillStatusCode) {
	t.Helper()
	for _, status := range statuses {
		if status.Name == name {
			if status.Status != want {
				t.Fatalf("status %s = %s (%s), want %s", name, status.Status, status.Reason, want)
			}
			return
		}
	}
	t.Fatalf("status for %s not found in %#v", name, statuses)
}
