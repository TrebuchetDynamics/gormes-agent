package skillscmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	skillruntime "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
)

func TestRunProfileSyncUsesSeamsAndWritesHumanReport(t *testing.T) {
	var gotReq skillruntime.BundledSkillProfileSyncRequest
	var out bytes.Buffer
	err := RunProfileSync(context.Background(), &out, ProfileSyncSeams{
		BundledRoot: func() string { return "/fixture/bundled" },
		Profiles: func() ([]skillruntime.SkillProfileRoot, error) {
			return []skillruntime.SkillProfileRoot{
				{Name: "main", Root: "/fixture/main"},
				{Name: "work", Root: "/fixture/work"},
			}, nil
		},
		Sync: func(_ context.Context, req skillruntime.BundledSkillProfileSyncRequest) (skillruntime.BundledSkillProfileSyncReport, error) {
			gotReq = req
			return skillruntime.BundledSkillProfileSyncReport{
				Summaries: []skillruntime.SkillProfileSyncSummary{
					{Profile: "main", Added: 1},
					{Profile: "work", Unchanged: 1},
				},
			}, nil
		},
	}, SyncOptions{})
	if err != nil {
		t.Fatalf("RunProfileSync() error = %v", err)
	}
	wantProfiles := []skillruntime.SkillProfileRoot{
		{Name: "main", Root: "/fixture/main"},
		{Name: "work", Root: "/fixture/work"},
	}
	if gotReq.BundledRoot != "/fixture/bundled" {
		t.Fatalf("BundledRoot = %q, want fake bundled root", gotReq.BundledRoot)
	}
	if !reflect.DeepEqual(gotReq.Profiles, wantProfiles) {
		t.Fatalf("Profiles = %#v, want %#v", gotReq.Profiles, wantProfiles)
	}
	text := out.String()
	for _, want := range []string{"main\tadded=1", "work\tadded=0 unchanged=1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("stdout missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "/fixture/") {
		t.Fatalf("stdout leaked fake roots:\n%s", text)
	}
}

func TestWriteProfileSyncReportJSONIncludesBuildProvenance(t *testing.T) {
	var out bytes.Buffer
	err := WriteProfileSyncReport(&out, skillruntime.BundledSkillProfileSyncReport{
		Summaries: []skillruntime.SkillProfileSyncSummary{
			{Profile: "main", Added: 2, Unchanged: 3, Conflicts: 1},
		},
	}, true, BuildProvenance{Version: "v-test", GitCommit: "abc123"})
	if err != nil {
		t.Fatalf("WriteProfileSyncReport() error = %v", err)
	}
	var got struct {
		Build     BuildProvenance `json:"build"`
		Summaries []struct {
			Profile   string `json:"profile"`
			Added     int    `json:"added"`
			Unchanged int    `json:"unchanged"`
			Conflicts int    `json:"conflicts"`
			Failed    int    `json:"failed"`
		} `json:"summaries"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if got.Build.Version != "v-test" || got.Build.GitCommit != "abc123" {
		t.Fatalf("build = %+v, want v-test/abc123", got.Build)
	}
	if len(got.Summaries) != 1 || got.Summaries[0].Profile != "main" || got.Summaries[0].Added != 2 || got.Summaries[0].Conflicts != 1 {
		t.Fatalf("summaries = %+v", got.Summaries)
	}
}

func TestDefaultProfileRootsUsesBaseHomeFromProfileScopedProcess(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	mainRoot := filepath.Join(base, "profiles", "main")
	workRoot := filepath.Join(base, "profiles", "work")
	for _, root := range []string{mainRoot, workRoot} {
		if err := os.MkdirAll(root, 0o700); err != nil {
			t.Fatalf("mkdir profile root %s: %v", root, err)
		}
	}
	t.Setenv("GORMES_HOME", workRoot)

	profiles, err := DefaultProfileRoots()
	if err != nil {
		t.Fatalf("DefaultProfileRoots(): %v", err)
	}
	got := map[string]string{}
	for _, profile := range profiles {
		got[profile.Name] = profile.Root
	}
	if got["main"] != mainRoot {
		t.Fatalf("default skill-sync root = %q, want materialized base-home default root %q; profiles=%+v", got["main"], mainRoot, profiles)
	}
	if got["work"] != workRoot {
		t.Fatalf("work skill-sync root = %q, want base-home profile root %q; profiles=%+v", got["work"], workRoot, profiles)
	}
	for _, root := range got {
		if strings.Contains(root, filepath.Join("profiles", "gormes", "profiles")) {
			t.Fatalf("skill sync produced nested profile root %q from profile-scoped process; profiles=%+v", root, profiles)
		}
	}
}

func TestConfigSkillStoreWritesOnlyInsideActiveRoot(t *testing.T) {
	root := t.TempDir()
	store := configSkillStore{root: root}
	active := filepath.Join(root, "active")
	targetDir := filepath.Join(active, "team", "demo")
	path, err := store.WriteSkill(context.Background(), targetDir, "SKILL.md", []byte("demo skill"))
	if err != nil {
		t.Fatalf("WriteSkill inside active root: %v", err)
	}
	if path != filepath.Join(targetDir, "SKILL.md") {
		t.Fatalf("path = %q, want target SKILL.md", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if string(body) != "demo skill" {
		t.Fatalf("body = %q", body)
	}

	outside := filepath.Join(root, "outside", "demo")
	if _, err := store.WriteSkill(context.Background(), outside, "SKILL.md", []byte("escape")); err == nil {
		t.Fatal("WriteSkill outside active root succeeded; want escape error")
	}
}

func TestPathWithinRejectsSiblingEscapes(t *testing.T) {
	root := filepath.Join(t.TempDir(), "active")
	if !pathWithin(root, filepath.Join(root, "demo", "SKILL.md")) {
		t.Fatal("pathWithin rejected child path")
	}
	if pathWithin(root, filepath.Join(filepath.Dir(root), "active-sibling", "SKILL.md")) {
		t.Fatal("pathWithin accepted sibling escape")
	}
}
