package main

import (
	"bytes"
	"context"
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

func executeSkillsSyncCommand(cmd *cobra.Command, args ...string) (string, error) {
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), err
}
