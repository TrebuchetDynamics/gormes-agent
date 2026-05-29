package profileseed

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

func TestProfileSeedDraftTemplateRedactsSeedAndRequiresWorkspaceConfirmation(t *testing.T) {
	draft, err := NewDraft("work on mineru repo", DraftOptions{})
	if err != nil {
		t.Fatalf("NewDraft: %v", err)
	}
	if draft.ProfileID != "work-mineru-repo" {
		t.Fatalf("ProfileID = %q, want work-mineru-repo", draft.ProfileID)
	}
	if draft.DisplayName != "Work Mineru Repo" {
		t.Fatalf("DisplayName = %q, want Work Mineru Repo", draft.DisplayName)
	}
	if draft.GenerationSource != "template" {
		t.Fatalf("GenerationSource = %q, want template", draft.GenerationSource)
	}
	if draft.ProviderModelState.Status != "unconfigured" {
		t.Fatalf("ProviderModelState.Status = %q, want unconfigured", draft.ProviderModelState.Status)
	}
	if len(draft.WorkspaceRootSuggestions) == 0 || !draft.WorkspaceRootSuggestions[0].RequiresConfirmation {
		t.Fatalf("WorkspaceRootSuggestions = %+v, want confirmation-required suggestion", draft.WorkspaceRootSuggestions)
	}
	if draft.ToolPolicy.Mode != "safe" || len(draft.ToolPolicy.RequiresApproval) == 0 {
		t.Fatalf("ToolPolicy = %+v, want safe policy with approvals", draft.ToolPolicy)
	}
	if draft.VoiceProfileMetadata.Status != "draft" {
		t.Fatalf("VoiceProfileMetadata.Status = %q, want draft", draft.VoiceProfileMetadata.Status)
	}
	raw, err := json.Marshal(draft)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"/home/xel", "workspace-mineru", "api_key", "token=", "sk-secret"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("draft leaked %q: %s", forbidden, raw)
		}
	}
}

func TestProfileSeedDraftRejectsRiskySecretLikeSeeds(t *testing.T) {
	_, err := NewDraft("screen token=sk-secret for support", DraftOptions{})
	if !errors.Is(err, ErrUnsafeSeed) {
		t.Fatalf("NewDraft unsafe err = %v, want ErrUnsafeSeed", err)
	}
}

func TestProfileSeedModelDraftIsValidatedAndMarkedModel(t *testing.T) {
	providerDraft := Draft{
		ProfileID:    "support-triage",
		DisplayName:  "Support Triage",
		Instructions: "Triage support calls and ask before mutating workspaces.",
		WorkspaceRootSuggestions: []WorkspaceRootSuggestion{{
			Label:                "support workspace",
			Purpose:              "operator-confirmed support work",
			RequiresConfirmation: true,
		}},
	}
	draft, err := NewDraft("triage support calls", DraftOptions{
		Provider:      "openai-codex",
		Model:         "gpt-5.3-codex",
		ProviderDraft: &providerDraft,
	})
	if err != nil {
		t.Fatalf("NewDraft provider: %v", err)
	}
	if draft.GenerationSource != "model" {
		t.Fatalf("GenerationSource = %q, want model", draft.GenerationSource)
	}
	if draft.ProviderModelState.Status != "configured" || draft.ProviderModelState.Provider != "openai-codex" || draft.ProviderModelState.Model != "gpt-5.3-codex" {
		t.Fatalf("ProviderModelState = %+v, want configured provider/model", draft.ProviderModelState)
	}
	if draft.ProfileID != "support-triage" {
		t.Fatalf("ProfileID = %q, want support-triage", draft.ProfileID)
	}
}

func TestProfileSeedDefaultCreateProfileUsesBaseHomeFromProfileScopedEnv(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	t.Setenv("GORMES_HOME", filepath.Join(base, "profiles", "active"))

	result, err := Apply("support ops", ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	wantRoot := filepath.Join(base, "profiles", "support-ops")
	if result.Root != wantRoot {
		t.Fatalf("Apply root = %q, want base-home profile root %q", result.Root, wantRoot)
	}
	if strings.Contains(result.Root, filepath.Join("profiles", "active", "profiles")) {
		t.Fatalf("Apply created nested profile root under active profile: %q", result.Root)
	}
	if _, err := os.Stat(filepath.Join(wantRoot, "profile_seed.json")); err != nil {
		t.Fatalf("profile_seed.json missing under base profile root: %v", err)
	}
}

func TestProfileSeedDefaultCreateProfileCloneAllUsesMaterializedMainSource(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	materializedDefault := filepath.Join(base, "profiles", "main")
	if err := os.MkdirAll(materializedDefault, 0o700); err != nil {
		t.Fatalf("mkdir materialized main profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(materializedDefault, "config.toml"), []byte("model = 'materialized'\n"), 0o600); err != nil {
		t.Fatalf("write materialized main config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "legacy-only.txt"), []byte("legacy"), 0o600); err != nil {
		t.Fatalf("write legacy marker: %v", err)
	}
	t.Setenv("GORMES_HOME", filepath.Join(base, "profiles", "active"))

	created, err := defaultCreateProfile("seeded", true)
	if err != nil {
		t.Fatalf("defaultCreateProfile seeded cloneAll: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(created.Root, "config.toml")); err != nil || !strings.Contains(string(got), "materialized") {
		t.Fatalf("created profile config = %q, %v; want materialized default config", string(got), err)
	}
	if _, err := os.Stat(filepath.Join(created.Root, "legacy-only.txt")); !os.IsNotExist(err) {
		t.Fatalf("clone_all copied legacy base root despite materialized default, stat err=%v", err)
	}
}

func TestProfileSeedApplyCreatesProfileManifestWithoutImplicitWorkspaceGrant(t *testing.T) {
	root := filepath.Join(t.TempDir(), "profiles", "work-mineru-repo")
	result, err := Apply("work on mineru repo", ApplyOptions{
		CreateProfile: func(name string, cloneAll bool) (cli.ProfileCreateResult, error) {
			if name != "work-mineru-repo" || cloneAll {
				t.Fatalf("CreateProfile(%q,%v), want work-mineru-repo,false", name, cloneAll)
			}
			if err := os.MkdirAll(root, 0o700); err != nil {
				return cli.ProfileCreateResult{}, err
			}
			return cli.ProfileCreateResult{Name: name, Root: root}, nil
		},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.ProfileID != "work-mineru-repo" || !result.Applied {
		t.Fatalf("result = %+v, want applied work-mineru-repo", result)
	}
	manifest := filepath.Join(root, "profile_seed.json")
	if _, err := os.Stat(manifest); err != nil {
		t.Fatalf("profile_seed.json missing: %v", err)
	}
	configPath := filepath.Join(root, "config.toml")
	body, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config.toml missing: %v", err)
	}
	if strings.Contains(string(body), "workspaces") || result.WorkspaceCount != 0 {
		t.Fatalf("unconfirmed workspace should not be granted; count=%d config=%s", result.WorkspaceCount, body)
	}
}

func TestProfileSeedApplyPersistsOnlyConfirmedCanonicalWorkspaces(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "profiles", "triage-support-calls")
	workspace := filepath.Join(home, "support")
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		t.Fatal(err)
	}
	result, err := Apply("triage support calls", ApplyOptions{
		ConfirmedWorkspaces: []string{workspace},
		CreateProfile: func(name string, cloneAll bool) (cli.ProfileCreateResult, error) {
			if err := os.MkdirAll(root, 0o700); err != nil {
				return cli.ProfileCreateResult{}, err
			}
			return cli.ProfileCreateResult{Name: name, Root: root}, nil
		},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.WorkspaceCount != 1 {
		t.Fatalf("WorkspaceCount = %d, want 1", result.WorkspaceCount)
	}
	body, err := os.ReadFile(filepath.Join(root, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	canonical, _ := filepath.Abs(workspace)
	if !strings.Contains(string(body), canonical) || !strings.Contains(string(body), "workspaces =") {
		t.Fatalf("config.toml = %s, want canonical confirmed workspace %q", body, canonical)
	}
}
