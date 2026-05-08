package main

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/spf13/cobra"
)

func TestSkillsSyncCommandRunsProfileSyncWithFakeRoots(t *testing.T) {
	var gotReq skills.BundledSkillProfileSyncRequest
	cmd := newSkillsCommandWithProfileSync(skillsProfileSyncSeams{
		BundledRoot: func() string {
			return "/fixture/bundled"
		},
		Profiles: func() ([]skills.SkillProfileRoot, error) {
			return []skills.SkillProfileRoot{
				{Name: "default", Root: "/fixture/default"},
				{Name: "work", Root: "/fixture/work"},
			}, nil
		},
		Sync: func(ctx context.Context, req skills.BundledSkillProfileSyncRequest) (skills.BundledSkillProfileSyncReport, error) {
			gotReq = req
			return skills.BundledSkillProfileSyncReport{
				Summaries: []skills.SkillProfileSyncSummary{
					{Profile: "default", Added: 1},
					{Profile: "work", Unchanged: 1},
				},
			}, nil
		},
	})

	stdout, err := executeSkillsSyncCommand(cmd, "sync")
	if err != nil {
		t.Fatalf("skills sync error = %v", err)
	}

	wantProfiles := []skills.SkillProfileRoot{
		{Name: "default", Root: "/fixture/default"},
		{Name: "work", Root: "/fixture/work"},
	}
	if gotReq.BundledRoot != "/fixture/bundled" {
		t.Fatalf("BundledRoot = %q, want fake bundled root", gotReq.BundledRoot)
	}
	if !reflect.DeepEqual(gotReq.Profiles, wantProfiles) {
		t.Fatalf("Profiles = %#v, want %#v", gotReq.Profiles, wantProfiles)
	}
	for _, want := range []string{"default\tadded=1", "work\tadded=0 unchanged=1"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "/fixture/") {
		t.Fatalf("stdout leaked fake roots:\n%s", stdout)
	}
}

// TestSkillsSyncCommand_JSONEmitsStructuredReport proves
// `gormes skills sync --json` returns a parseable
// `{build, summaries: [{profile, added, unchanged, conflicts, failed}]}`
// document so fleet automation rolling out skills across many profiles
// can audit per-profile counts without scraping tab-separated output.
func TestSkillsSyncCommand_JSONEmitsStructuredReport(t *testing.T) {
	cmd := newSkillsCommandWithProfileSync(skillsProfileSyncSeams{
		BundledRoot: func() string { return "/fixture/bundled" },
		Profiles: func() ([]skills.SkillProfileRoot, error) {
			return []skills.SkillProfileRoot{
				{Name: "default", Root: "/fixture/default"},
				{Name: "work", Root: "/fixture/work"},
			}, nil
		},
		Sync: func(ctx context.Context, req skills.BundledSkillProfileSyncRequest) (skills.BundledSkillProfileSyncReport, error) {
			return skills.BundledSkillProfileSyncReport{
				Summaries: []skills.SkillProfileSyncSummary{
					{Profile: "default", Added: 2, Unchanged: 3, Conflicts: 1, Failed: 0},
					{Profile: "work", Added: 0, Unchanged: 5},
				},
			}, nil
		},
	})

	stdout, err := executeSkillsSyncCommand(cmd, "sync", "--json")
	if err != nil {
		t.Fatalf("skills sync --json: %v", err)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Summaries []struct {
			Profile   string `json:"profile"`
			Added     int    `json:"added"`
			Unchanged int    `json:"unchanged"`
			Conflicts int    `json:"conflicts"`
			Failed    int    `json:"failed"`
		} `json:"summaries"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("skills sync --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if len(got.Summaries) != 2 {
		t.Fatalf("summaries len = %d, want 2; got %+v", len(got.Summaries), got.Summaries)
	}
	if got.Summaries[0].Profile != "default" || got.Summaries[0].Added != 2 || got.Summaries[0].Conflicts != 1 {
		t.Errorf("summaries[0] = %+v", got.Summaries[0])
	}
	if got.Summaries[1].Profile != "work" || got.Summaries[1].Unchanged != 5 {
		t.Errorf("summaries[1] = %+v", got.Summaries[1])
	}
	// Fixture root paths MUST stay out of stdout.
	if strings.Contains(stdout, "/fixture/") {
		t.Fatalf("stdout leaked fake roots:\n%s", stdout)
	}
}

func executeSkillsSyncCommand(cmd *cobra.Command, args ...string) (string, error) {
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), err
}
