package skills

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBundledSkillProfileSyncWritesActiveAndNamedProfiles(t *testing.T) {
	bundledRoot := t.TempDir()
	writeProfileSyncSkill(t, bundledRoot, "productivity", "reviewer", "Review docs")

	defaultRoot := filepath.Join(t.TempDir(), "default")
	activeRoot := filepath.Join(t.TempDir(), "active")
	workRoot := filepath.Join(t.TempDir(), "work")

	report, err := SyncBundledSkillsToProfiles(context.Background(), BundledSkillProfileSyncRequest{
		BundledRoot: bundledRoot,
		Profiles: []SkillProfileRoot{
			{Name: "active", Root: activeRoot},
			{Name: "default", Root: defaultRoot},
			{Name: "work", Root: workRoot},
		},
	})
	if err != nil {
		t.Fatalf("SyncBundledSkillsToProfiles() error = %v", err)
	}

	for _, root := range []string{activeRoot, defaultRoot, workRoot} {
		assertProfileSkillBody(t, root, "productivity", "reviewer", "Review docs")
	}
	gotProfiles := profileSyncSummaryNames(report.Summaries)
	wantProfiles := []string{"active", "default", "work"}
	if !reflect.DeepEqual(gotProfiles, wantProfiles) {
		t.Fatalf("profiles = %#v, want %#v", gotProfiles, wantProfiles)
	}
	for _, summary := range report.Summaries {
		if summary.Added != 1 || summary.Unchanged != 0 || summary.Conflicts != 0 || summary.Failed != 0 {
			t.Fatalf("summary for %s = %+v, want 1 added only", summary.Profile, summary)
		}
	}
}

func TestBundledSkillProfileSyncPreservesUserModifiedSkills(t *testing.T) {
	bundledRoot := t.TempDir()
	writeProfileSyncSkill(t, bundledRoot, "productivity", "reviewer", "Review docs")

	profileRoot := t.TempDir()
	target := profileSkillPath(profileRoot, "productivity", "reviewer")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll(target): %v", err)
	}
	if err := os.WriteFile(target, []byte("user edited skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(target): %v", err)
	}

	report, err := SyncBundledSkillsToProfiles(context.Background(), BundledSkillProfileSyncRequest{
		BundledRoot: bundledRoot,
		Profiles:    []SkillProfileRoot{{Name: "work", Root: profileRoot}},
	})
	if err != nil {
		t.Fatalf("SyncBundledSkillsToProfiles() error = %v", err)
	}

	assertFileText(t, target, "user edited skill")
	if len(report.Summaries) != 1 {
		t.Fatalf("summaries = %#v, want one", report.Summaries)
	}
	summary := report.Summaries[0]
	if summary.Conflicts != 1 || summary.Added != 0 {
		t.Fatalf("summary = %+v, want one conflict and no add", summary)
	}
	if len(report.Evidence) != 1 {
		t.Fatalf("evidence = %#v, want one conflict", report.Evidence)
	}
	if ev := report.Evidence[0]; ev.Code != SkillProfileSyncConflict || ev.Profile != "work" || ev.Skill != "reviewer" {
		t.Fatalf("evidence = %+v, want redacted work/reviewer conflict", ev)
	}
}

func TestBundledSkillProfileSyncSummaryDeterministic(t *testing.T) {
	bundledRoot := t.TempDir()
	writeProfileSyncSkill(t, bundledRoot, "ops", "deploy", "Deploy apps")

	alphaRoot := t.TempDir()
	betaRoot := t.TempDir()
	assertWriteProfileSkill(t, alphaRoot, "ops", "deploy", profileSyncSkillDoc("deploy", "Deploy apps"))
	assertWriteProfileSkill(t, betaRoot, "ops", "deploy", "operator changes")

	report, err := SyncBundledSkillsToProfiles(context.Background(), BundledSkillProfileSyncRequest{
		BundledRoot: bundledRoot,
		Profiles: []SkillProfileRoot{
			{Name: "beta", Root: betaRoot},
			{Name: "alpha", Root: alphaRoot},
		},
	})
	if err != nil {
		t.Fatalf("SyncBundledSkillsToProfiles() error = %v", err)
	}

	gotProfiles := profileSyncSummaryNames(report.Summaries)
	wantProfiles := []string{"alpha", "beta"}
	if !reflect.DeepEqual(gotProfiles, wantProfiles) {
		t.Fatalf("profiles = %#v, want sorted %#v", gotProfiles, wantProfiles)
	}
	if report.Summaries[0].Unchanged != 1 || report.Summaries[1].Conflicts != 1 {
		t.Fatalf("summaries = %+v, want alpha unchanged then beta conflict", report.Summaries)
	}
	for _, ev := range report.Evidence {
		if ev.Path != "" {
			t.Fatalf("evidence leaked path: %+v", ev)
		}
	}
}

func writeProfileSyncSkill(t *testing.T, root, category, name, description string) {
	t.Helper()
	path := filepath.Join(root, category, name, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(profileSyncSkillDoc(name, description)), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func profileSyncSkillDoc(name, description string) string {
	return "---\nname: " + name + "\ndescription: " + description + "\nreview_state: reviewed\n---\n\n" + description + "."
}

func assertProfileSkillBody(t *testing.T, root, category, name, wantBody string) {
	t.Helper()
	raw, err := os.ReadFile(profileSkillPath(root, category, name))
	if err != nil {
		t.Fatalf("ReadFile profile skill: %v", err)
	}
	if got := string(raw); !contains(got, wantBody) {
		t.Fatalf("profile skill body = %q, want to contain %q", got, wantBody)
	}
}

func assertWriteProfileSkill(t *testing.T, root, category, name, body string) {
	t.Helper()
	path := profileSkillPath(root, category, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func profileSkillPath(root, category, name string) string {
	return filepath.Join(root, "skills", "active", category, name, "SKILL.md")
}

func assertFileText(t *testing.T, path, want string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if string(raw) != want {
		t.Fatalf("file %q = %q, want %q", path, string(raw), want)
	}
}

func profileSyncSummaryNames(summaries []SkillProfileSyncSummary) []string {
	out := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		out = append(out, summary.Profile)
	}
	return out
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
