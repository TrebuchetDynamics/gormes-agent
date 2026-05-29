package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

func TestProfileSeedDryRunTemplateJSONDoesNotCreateProfile(t *testing.T) {
	t.Setenv("GORMES_HOME", filepath.Join(t.TempDir(), "gormes"))
	fake := &profileCommandFakeSeams{
		createProfile: func(name string, cloneAll bool) (cli.ProfileCreateResult, error) {
			t.Fatalf("CreateProfile should not run during --dry-run; got name=%q clone_all=%v", name, cloneAll)
			return cli.ProfileCreateResult{}, nil
		},
	}
	stdout, stderr, err := runProfileTestCommand(t, fake.defaults(), "seed", "work on mineru repo", "--json", "--dry-run")
	if err != nil {
		t.Fatalf("profile seed dry-run: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(fake.createProfileCalls) != 0 {
		t.Fatalf("createProfileCalls = %+v, want none", fake.createProfileCalls)
	}
	var got struct {
		Action string `json:"action"`
		Status string `json:"status"`
		Draft  struct {
			ProfileID          string `json:"profile_id"`
			DisplayName        string `json:"display_name"`
			GenerationSource   string `json:"generation_source"`
			ProviderModelState struct {
				Status string `json:"status"`
			} `json:"provider_model_state"`
			WorkspaceRootSuggestions []struct {
				RequiresConfirmation bool `json:"requires_confirmation"`
			} `json:"workspace_root_suggestions"`
		} `json:"draft"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", err, stdout)
	}
	if got.Action != "profile_seed_draft" || got.Status != "draft" || got.Draft.ProfileID != "work-mineru-repo" || got.Draft.GenerationSource != "template" {
		t.Fatalf("unexpected dry-run report: %+v", got)
	}
	if got.Draft.ProviderModelState.Status != "unconfigured" {
		t.Fatalf("provider state = %+v, want unconfigured", got.Draft.ProviderModelState)
	}
	if len(got.Draft.WorkspaceRootSuggestions) == 0 || !got.Draft.WorkspaceRootSuggestions[0].RequiresConfirmation {
		t.Fatalf("workspace suggestions = %+v, want confirmation required", got.Draft.WorkspaceRootSuggestions)
	}
	for _, forbidden := range []string{"/home/operator-secret", "sk-secret", "api_key", "token="} {
		if strings.Contains(stdout+stderr, forbidden) {
			t.Fatalf("profile seed output leaked %q:\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}
}

func TestProfileSeedApplyJSONCreatesProfileWithoutImplicitWorkspaceGrant(t *testing.T) {
	t.Setenv("GORMES_HOME", filepath.Join(t.TempDir(), "gormes"))
	profileRoot := filepath.Join(t.TempDir(), "operator-secret", "profiles", "work-mineru-repo")
	fake := &profileCommandFakeSeams{
		createProfile: func(name string, cloneAll bool) (cli.ProfileCreateResult, error) {
			if name != "work-mineru-repo" || cloneAll {
				t.Fatalf("CreateProfile(%q,%v), want work-mineru-repo,false", name, cloneAll)
			}
			if err := os.MkdirAll(profileRoot, 0o700); err != nil {
				return cli.ProfileCreateResult{}, err
			}
			return cli.ProfileCreateResult{Name: name, Root: profileRoot, CloneAll: cloneAll}, nil
		},
	}
	stdout, stderr, err := runProfileTestCommand(t, fake.defaults(), "seed", "work on mineru repo", "--json", "--apply")
	if err != nil {
		t.Fatalf("profile seed apply: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if len(fake.createProfileCalls) != 1 || fake.createProfileCalls[0] != (profileCreateCall{name: "work-mineru-repo", cloneAll: false}) {
		t.Fatalf("createProfileCalls = %+v, want one work-mineru-repo non-clone", fake.createProfileCalls)
	}
	var got struct {
		Action         string `json:"action"`
		Status         string `json:"status"`
		Applied        bool   `json:"applied"`
		ProfileID      string `json:"profile_id"`
		Root           string `json:"root"`
		WorkspaceCount int    `json:"workspace_count"`
		Draft          struct {
			GenerationSource string `json:"generation_source"`
		} `json:"draft"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", err, stdout)
	}
	if got.Action != "profile_seed_applied" || got.Status != "applied" || !got.Applied || got.ProfileID != "work-mineru-repo" || got.WorkspaceCount != 0 {
		t.Fatalf("unexpected apply report: %+v", got)
	}
	if got.Root != ".../work-mineru-repo" {
		t.Fatalf("root = %q, want redacted .../work-mineru-repo", got.Root)
	}
	for _, path := range []string{filepath.Join(profileRoot, "profile_seed.json"), filepath.Join(profileRoot, "config.toml")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
	body, err := os.ReadFile(filepath.Join(profileRoot, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "workspaces") {
		t.Fatalf("unconfirmed seed suggestions must not write workspace grants:\n%s", body)
	}
	if strings.Contains(stdout+stderr, "operator-secret") {
		t.Fatalf("profile seed apply leaked raw root:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}
