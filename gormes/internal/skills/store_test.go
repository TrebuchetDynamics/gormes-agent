package skills

import (
	"context"
	"encoding/json"
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

func TestRuntimeBuildSkillBlockAutoDiscoversExternalSkillsWithLocalPrecedenceAndWritesPromptSnapshot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sharedRoot := filepath.Join(home, "shared-skills")
	t.Setenv("SHARED_SKILLS_ROOT", sharedRoot)

	root := t.TempDir()
	writeSkillDoc(t, filepath.Join(root, "active", "devops", "deploy-k8s", "SKILL.md"), "deploy-k8s", "Deploy Kubernetes clusters", "Use kubectl apply followed by rollout status checks.")
	writeSkillDoc(t, filepath.Join(home, "team-skills", "devops", "cluster-checklist", "SKILL.md"), "cluster-checklist", "Checklist for Kubernetes rollouts", "Verify rollout health and service endpoints before sign-off.")
	writeSkillDoc(t, filepath.Join(sharedRoot, "devops", "deploy-k8s", "SKILL.md"), "deploy-k8s", "EXTERNAL shadowed copy", "This external duplicate should never win over the active store.")
	writeSkillDoc(t, filepath.Join(sharedRoot, "devops", "kubectl-rollout", "SKILL.md"), "kubectl-rollout", "Inspect kubectl rollout state", "Use kubectl rollout status and rollout history.")

	snapshotPath := filepath.Join(root, ".skills_prompt_snapshot.json")
	runtime := NewRuntimeWithConfig(RuntimeConfig{
		Root:               root,
		ExternalDirs:       []string{"~/team-skills", "${SHARED_SKILLS_ROOT}", filepath.Join(root, "missing")},
		MaxDocumentBytes:   8 * 1024,
		SelectionCap:       3,
		PromptSnapshotPath: snapshotPath,
	})

	block, names, err := runtime.BuildSkillBlock(context.Background(), "deploy kubernetes rollout checklist")
	if err != nil {
		t.Fatalf("BuildSkillBlock() error = %v", err)
	}

	gotNames := append([]string(nil), names...)
	sortStrings(gotNames)
	wantNames := []string{"cluster-checklist", "deploy-k8s", "kubectl-rollout"}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("selected names = %#v, want %#v", gotNames, wantNames)
	}

	if !containsAll(block,
		"Deploy Kubernetes clusters",
		"Checklist for Kubernetes rollouts",
		"Inspect kubectl rollout state",
	) {
		t.Fatalf("BuildSkillBlock() = %q, want selected local+external skill content", block)
	}
	if containsAll(block, "EXTERNAL shadowed copy") {
		t.Fatalf("BuildSkillBlock() = %q, want local active skill to shadow external duplicate", block)
	}

	raw, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", snapshotPath, err)
	}

	var snap promptSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("json.Unmarshal(snapshot): %v", err)
	}
	if snap.UserMessage != "deploy kubernetes rollout checklist" {
		t.Fatalf("snapshot UserMessage = %q, want original query", snap.UserMessage)
	}
	if snap.GeneratedAt.IsZero() {
		t.Fatal("snapshot GeneratedAt is zero")
	}
	if snap.Block != block {
		t.Fatalf("snapshot Block = %q, want %q", snap.Block, block)
	}
	if len(snap.Skills) != 3 {
		t.Fatalf("len(snapshot Skills) = %d, want 3", len(snap.Skills))
	}

	skillPaths := map[string]string{}
	for _, item := range snap.Skills {
		skillPaths[item.Name] = item.Path
	}
	if got := skillPaths["deploy-k8s"]; got != filepath.Join(root, "active", "devops", "deploy-k8s", "SKILL.md") {
		t.Fatalf("snapshot deploy-k8s path = %q, want local active path", got)
	}
	if got := skillPaths["cluster-checklist"]; got != filepath.Join(home, "team-skills", "devops", "cluster-checklist", "SKILL.md") {
		t.Fatalf("snapshot cluster-checklist path = %q, want home-expanded external path", got)
	}
	if got := skillPaths["kubectl-rollout"]; got != filepath.Join(sharedRoot, "devops", "kubectl-rollout", "SKILL.md") {
		t.Fatalf("snapshot kubectl-rollout path = %q, want env-expanded external path", got)
	}
}

func containsAll(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			return false
		}
	}
	return true
}

func sortStrings(values []string) {
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})
}

func writeSkillDoc(t *testing.T, path, name, description, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	raw := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n" + body
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
