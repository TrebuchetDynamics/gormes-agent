package llm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSkillsPromptToolsetConditionsAndNilFilters(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "coding/native", "native", "Native skill", "", "metadata:\n  hermes:\n    fallback_for_toolsets: [terminal]\n")
	writeSkill(t, root, "coding/remote", "remote", "Remote skill", "", "metadata:\n  hermes:\n    requires_toolsets: [browser]\n")

	ResetSkillsPromptCacheForTest()
	all, evidence, err := BuildSkillsSystemPrompt(SkillsPromptOptions{LocalDir: root})
	if err != nil {
		t.Fatalf("BuildSkillsSystemPrompt nil filters: %v", err)
	}
	if !strings.Contains(all, "- native: Native skill") || !strings.Contains(all, "- remote: Remote skill") {
		t.Fatalf("nil filters should preserve Hermes backward-compatible show-all behavior; prompt=%q evidence=%v", all, evidence)
	}

	ResetSkillsPromptCacheForTest()
	filtered, _, err := BuildSkillsSystemPrompt(SkillsPromptOptions{LocalDir: root, AvailableToolsets: []string{"terminal"}})
	if err != nil {
		t.Fatalf("BuildSkillsSystemPrompt filtered: %v", err)
	}
	if strings.Contains(filtered, "native") {
		t.Fatalf("fallback_for_toolsets skill should hide when primary toolset is available: %q", filtered)
	}
	if strings.Contains(filtered, "remote") {
		t.Fatalf("requires_toolsets skill should hide when browser toolset is missing: %q", filtered)
	}
}

func TestSkillsPromptCacheKeyIncludesToolsetsPlatformAndDisabled(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "coding/native", "native", "Native skill", "", "metadata:\n  hermes:\n    fallback_for_toolsets: [terminal]\n")
	writeSkill(t, root, "ops/linux-only", "linux-only", "Linux skill", "platforms: [linux]\n", "")

	ResetSkillsPromptCacheForTest()
	base, _, err := BuildSkillsSystemPrompt(SkillsPromptOptions{LocalDir: root})
	if err != nil {
		t.Fatalf("base prompt: %v", err)
	}
	if !strings.Contains(base, "native") || !strings.Contains(base, "linux-only") {
		t.Fatalf("base prompt missing expected skills: %q", base)
	}

	withoutNative, _, err := BuildSkillsSystemPrompt(SkillsPromptOptions{LocalDir: root, DisabledSkillNames: []string{"native"}})
	if err != nil {
		t.Fatalf("disabled prompt: %v", err)
	}
	if strings.Contains(withoutNative, "native") || !strings.Contains(withoutNative, "linux-only") {
		t.Fatalf("disabled-skill cache key leaked stale prompt: %q", withoutNative)
	}

	withoutTerminalFallback, _, err := BuildSkillsSystemPrompt(SkillsPromptOptions{LocalDir: root, AvailableToolsets: []string{"terminal"}})
	if err != nil {
		t.Fatalf("toolset prompt: %v", err)
	}
	if strings.Contains(withoutTerminalFallback, "native") {
		t.Fatalf("toolset cache key leaked fallback skill: %q", withoutTerminalFallback)
	}

	unsupportedPlatform, _, err := BuildSkillsSystemPrompt(SkillsPromptOptions{LocalDir: root, Platform: "windows"})
	if err != nil {
		t.Fatalf("platform prompt: %v", err)
	}
	if strings.Contains(unsupportedPlatform, "linux-only") {
		t.Fatalf("platform cache key leaked unsupported skill: %q", unsupportedPlatform)
	}
}

func TestSkillsPromptSnapshotManifestInvalidation(t *testing.T) {
	root := t.TempDir()
	snapshotPath := filepath.Join(t.TempDir(), "skills-snapshot.json")
	skillPath := writeSkill(t, root, "coding/alpha", "alpha", "Alpha v1", "", "")

	ResetSkillsPromptCacheForTest()
	first, evidence, err := BuildSkillsSystemPrompt(SkillsPromptOptions{LocalDir: root, SnapshotPath: snapshotPath})
	if err != nil {
		t.Fatalf("first prompt: %v", err)
	}
	if !strings.Contains(first, "Alpha v1") || !hasEvidence(evidence, "skills_prompt_snapshot_miss") {
		t.Fatalf("first prompt should scan and record miss; prompt=%q evidence=%v", first, evidence)
	}
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("snapshot was not written: %v", err)
	}

	ResetSkillsPromptCacheForTest()
	second, evidence, err := BuildSkillsSystemPrompt(SkillsPromptOptions{LocalDir: root, SnapshotPath: snapshotPath})
	if err != nil {
		t.Fatalf("second prompt: %v", err)
	}
	if second != first || !hasEvidence(evidence, "skills_prompt_snapshot_hit") {
		t.Fatalf("valid snapshot should be reused; first=%q second=%q evidence=%v", first, second, evidence)
	}

	time.Sleep(time.Millisecond)
	content := strings.Replace(mustRead(t, skillPath), "Alpha v1", "Alpha v2", 1)
	if err := os.WriteFile(skillPath, []byte(content), 0o644); err != nil {
		t.Fatalf("mutate skill: %v", err)
	}
	ResetSkillsPromptCacheForTest()
	third, evidence, err := BuildSkillsSystemPrompt(SkillsPromptOptions{LocalDir: root, SnapshotPath: snapshotPath})
	if err != nil {
		t.Fatalf("third prompt: %v", err)
	}
	if !strings.Contains(third, "Alpha v2") || !hasEvidence(evidence, "skills_prompt_snapshot_miss") {
		t.Fatalf("stale snapshot should be invalidated; prompt=%q evidence=%v", third, evidence)
	}

	if err := os.WriteFile(snapshotPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("corrupt snapshot: %v", err)
	}
	ResetSkillsPromptCacheForTest()
	fourth, evidence, err := BuildSkillsSystemPrompt(SkillsPromptOptions{LocalDir: root, SnapshotPath: snapshotPath})
	if err != nil {
		t.Fatalf("fourth prompt: %v", err)
	}
	if !strings.Contains(fourth, "Alpha v2") || !hasEvidence(evidence, "skills_prompt_snapshot_miss") {
		t.Fatalf("malformed snapshot should fall back to scan; prompt=%q evidence=%v", fourth, evidence)
	}
}

func TestSkillsPromptExternalDirsOrderingAndStatusFiltering(t *testing.T) {
	local := t.TempDir()
	external := t.TempDir()
	writeDescription(t, local, "coding", "Local category")
	writeDescription(t, external, "research", "External category")
	writeSkill(t, external, "coding/duplicate", "duplicate", "External duplicate", "", "")
	writeSkill(t, local, "coding/duplicate", "duplicate", "Local duplicate", "", "")
	writeSkill(t, external, "research/zeta", "zeta", "Zeta external", "", "")
	writeSkill(t, local, "coding/disabled", "disabled", "Disabled skill", "", "")
	writeSkill(t, local, "coding/linux-only", "linux-only", "Linux skill", "platforms: [linux]\n", "")
	writeInvalidSkill(t, local, "coding/bad", "bad", "Bad skill")

	ResetSkillsPromptCacheForTest()
	prompt, evidence, err := BuildSkillsSystemPrompt(SkillsPromptOptions{LocalDir: local, ExternalDirs: []string{external}, DisabledSkillNames: []string{"disabled"}, Platform: "windows"})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}
	for _, want := range []string{"  coding: Local category", "- duplicate: Local duplicate", "  research: External category", "- zeta: Zeta external"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q: %q", want, prompt)
		}
	}
	for _, notWant := range []string{"External duplicate", "Disabled skill", "Linux skill", "Bad skill"} {
		if strings.Contains(prompt, notWant) {
			t.Fatalf("prompt should exclude %q: %q", notWant, prompt)
		}
	}
	for _, code := range []string{"disabled", "unsupported", "frontmatter-invalid"} {
		if !hasEvidence(evidence, code) {
			t.Fatalf("expected status evidence %q in %v", code, evidence)
		}
	}
}

func writeSkill(t *testing.T, root, rel, name, desc, extraTopLevel, extraHermes string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel), "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if desc == "" {
		desc = name + " description"
	}
	body := "---\nname: " + name + "\ndescription: " + desc + "\n" + extraTopLevel + extraHermes + "---\n\n# " + name + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	return path
}

func writeInvalidSkill(t *testing.T, root, rel, name, desc string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel), "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir invalid skill: %v", err)
	}
	body := "---\nname: " + name + "\ndescription: " + desc + "\nmetadata:\n  hermes:\n    tags: [ok\n---\n\n# bad\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write invalid skill: %v", err)
	}
	return path
}

func writeDescription(t *testing.T, root, category, desc string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(category), "DESCRIPTION.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir desc: %v", err)
	}
	if err := os.WriteFile(path, []byte("---\ndescription: "+desc+"\n---\n"), 0o644); err != nil {
		t.Fatalf("write desc: %v", err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func hasEvidence(evidence []SkillsPromptEvidence, code string) bool {
	for _, item := range evidence {
		if item.Code == code {
			return true
		}
	}
	return false
}
