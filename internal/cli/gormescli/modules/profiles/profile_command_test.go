package profiles

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/provider"
)

func TestProfileModuleCommandUsesInjectedSeamsAndBuildProvenance(t *testing.T) {
	var writes []string
	cmd := NewCommandWithSeams(Seams{
		ReadActiveProfileName: func() (string, error) { return "default", nil },
		ValidateProfileName:   cli.ValidateProfileName,
		ResolveProfileRoot: func(name string) (string, error) {
			return "/home/operator-secret/.gormes/profiles/" + name, nil
		},
		WriteActiveProfile: func(name string) error {
			writes = append(writes, name)
			return nil
		},
		ListKnownProfiles: func() ([]string, error) {
			return []string{"default", "work"}, nil
		},
	}, Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{Version: "test-version", GitCommit: "test-sha"}
		},
	})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"use", "work", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("profile use --json: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if len(writes) != 1 || writes[0] != "work" {
		t.Fatalf("WriteActiveProfile calls = %v, want [work]", writes)
	}
	if strings.Contains(stdout.String()+stderr.String(), "/home/operator-secret") {
		t.Fatalf("profile module leaked raw root:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Action string `json:"action"`
		Active string `json:"active"`
		Root   string `json:"root"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("profile module stdout must be JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Build.Version != "test-version" || got.Build.GitCommit != "test-sha" {
		t.Fatalf("build provenance = %+v, want injected test values", got.Build)
	}
	if got.Action != "use" || got.Active != "work" || got.Root != ".../work" {
		t.Fatalf("profile JSON = %+v, want action=use active=work root=.../work", got)
	}
}

func TestProfileModuleDefaultSeamsUseBaseHomeFromProfileScopedProcess(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	activeProfileRoot := filepath.Join(base, "profiles", "work")
	researchRoot := filepath.Join(base, "profiles", "research")
	if err := os.MkdirAll(researchRoot, 0o700); err != nil {
		t.Fatalf("mkdir research profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "active_profile"), []byte("research\n"), 0o600); err != nil {
		t.Fatalf("write active profile: %v", err)
	}
	t.Setenv("GORMES_HOME", activeProfileRoot)

	seams := DefaultSeams()
	active, err := seams.ReadActiveProfileName()
	if err != nil {
		t.Fatalf("ReadActiveProfileName: %v", err)
	}
	if active != "research" {
		t.Fatalf("active profile = %q, want research", active)
	}
	root, err := seams.ResolveProfileRoot("research")
	if err != nil {
		t.Fatalf("ResolveProfileRoot: %v", err)
	}
	if root != researchRoot {
		t.Fatalf("ResolveProfileRoot = %q, want base-home profile root %q", root, researchRoot)
	}
	known, err := seams.ListKnownProfiles()
	if err != nil {
		t.Fatalf("ListKnownProfiles: %v", err)
	}
	if !containsString(known, "research") {
		t.Fatalf("known profiles = %v, want research from base home", known)
	}

	created, err := seams.CreateProfile("ops", false)
	if err != nil {
		t.Fatalf("CreateProfile ops: %v", err)
	}
	wantCreated := filepath.Join(base, "profiles", "ops")
	if created.Root != wantCreated {
		t.Fatalf("created root = %q, want %q", created.Root, wantCreated)
	}
	if strings.Contains(created.Root, filepath.Join("profiles", "work", "profiles")) {
		t.Fatalf("created nested profile root under active profile: %q", created.Root)
	}
}

func TestProfileModuleDefaultListKnownProfilesIncludesConfigV2Profiles(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatalf("mkdir base: %v", err)
	}
	body := `config_version = 2

[profiles.main]
enabled = true

[profiles.tulin]
enabled = true
`
	if err := os.WriteFile(filepath.Join(base, "config.toml"), []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("GORMES_HOME", base)

	known, err := DefaultListKnownProfiles()
	if err != nil {
		t.Fatalf("DefaultListKnownProfiles: %v", err)
	}
	for _, want := range []string{"default", "main", "tulin"} {
		if !containsString(known, want) {
			t.Fatalf("known profiles = %v, want %q from config-v2 profile registry", known, want)
		}
	}
	root, err := DefaultSeams().ResolveProfileRoot("main")
	if err != nil {
		t.Fatalf("ResolveProfileRoot(main): %v", err)
	}
	if root != base {
		t.Fatalf("ResolveProfileRoot(main) = %q, want v2 main base-home root %q", root, base)
	}
}

func TestProfileModuleDefaultSeamsProviderReadinessUsesBaseHomeFromProfileScopedProcess(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	activeRoot := filepath.Join(base, "profiles", "work")
	if err := os.MkdirAll(activeRoot, 0o700); err != nil {
		t.Fatalf("mkdir active profile root: %v", err)
	}
	rootConfig := `config_version = 2

[profiles.main]
enabled = true

[profiles.main.providers.openrouter]
enabled = true
credential = "main-openrouter"
default_model = "openai/gpt-5.2"
allowed_models = ["openai/gpt-5.2"]

[credentials.main-openrouter]
kind = "provider"
provider = "openrouter"
owner_profile = "main"

[credentials.main-openrouter.secret_ref]
source = "env"
id = "GORMES_MAIN_OPENROUTER_API_KEY"
`
	if err := os.WriteFile(filepath.Join(base, "config.toml"), []byte(rootConfig), 0o600); err != nil {
		t.Fatalf("write base config: %v", err)
	}
	t.Setenv("GORMES_HOME", activeRoot)

	reports, err := DefaultSeams().ProviderReadiness()
	if err != nil {
		t.Fatalf("ProviderReadiness: %v", err)
	}
	if len(reports) != 1 || reports[0].ProfileID != "main" || reports[0].CredentialID != "main-openrouter" {
		t.Fatalf("ProviderReadiness = %+v, want base-home profiles.main openrouter readiness", reports)
	}
}

func TestProfileModuleDefaultSeamsCreateProfileWithoutCloneAllDoesNotInspectDefaultSource(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	profilesDir := filepath.Join(base, "profiles")
	if err := os.MkdirAll(profilesDir, 0o700); err != nil {
		t.Fatalf("mkdir profiles dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profilesDir, "default"), []byte("not-a-directory"), 0o600); err != nil {
		t.Fatalf("write bad materialized default marker: %v", err)
	}
	t.Setenv("GORMES_HOME", base)

	created, err := DefaultSeams().CreateProfile("work", false)
	if err != nil {
		t.Fatalf("CreateProfile(work, cloneAll=false) must not inspect default source: %v", err)
	}
	if created.Root != filepath.Join(base, "profiles", "work") {
		t.Fatalf("created root = %q, want base-home work profile", created.Root)
	}
}

func TestProfileModuleDefaultSeamsCreateCloneAllUsesMaterializedDefaultProfileSource(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	materializedDefault := filepath.Join(base, "profiles", "default")
	if err := os.MkdirAll(materializedDefault, 0o700); err != nil {
		t.Fatalf("mkdir materialized default profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(materializedDefault, "config.toml"), []byte("model = 'materialized'\n"), 0o600); err != nil {
		t.Fatalf("write materialized default config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "legacy-only.txt"), []byte("legacy"), 0o600); err != nil {
		t.Fatalf("write legacy marker: %v", err)
	}
	t.Setenv("GORMES_HOME", base)

	seams := DefaultSeams()
	created, err := seams.CreateProfile("work", true)
	if err != nil {
		t.Fatalf("CreateProfile(work, cloneAll): %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(created.Root, "config.toml")); err != nil || !strings.Contains(string(got), "materialized") {
		t.Fatalf("created profile config = %q, %v; want materialized default config", string(got), err)
	}
	if _, err := os.Stat(filepath.Join(created.Root, "legacy-only.txt")); !os.IsNotExist(err) {
		t.Fatalf("clone_all copied legacy base root despite materialized default, stat err=%v", err)
	}
}

func TestProfileModuleDefaultSeamsUseMaterializedDefaultProfileRoot(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	materializedDefault := filepath.Join(base, "profiles", "default")
	if err := os.MkdirAll(materializedDefault, 0o700); err != nil {
		t.Fatalf("mkdir materialized default profile: %v", err)
	}
	t.Setenv("GORMES_HOME", base)

	seams := DefaultSeams()
	root, err := seams.ResolveProfileRoot("default")
	if err != nil {
		t.Fatalf("ResolveProfileRoot(default): %v", err)
	}
	if root != materializedDefault {
		t.Fatalf("ResolveProfileRoot(default) = %q, want materialized default root %q", root, materializedDefault)
	}
}

func TestProfileModuleProvidersCommandRendersReadiness(t *testing.T) {
	cmd := NewCommandWithSeams(Seams{
		ReadActiveProfileName: func() (string, error) { return "main", nil },
		ProviderReadiness: func() ([]provider.ProfileProviderReadiness, error) {
			return []provider.ProfileProviderReadiness{
				{
					ProfileID:    "main",
					ProviderID:   "openrouter",
					Status:       provider.ProfileProviderReady,
					CredentialID: "main-openrouter",
					SecretRef:    "env:GORMES_MAIN_OPENROUTER_API_KEY",
					DefaultModel: "openai/gpt-5.2",
					Models:       []string{"anthropic/claude-sonnet-4.5", "openai/gpt-5.2"},
				},
				{
					ProfileID:    "tulin",
					ProviderID:   "anthropic",
					Status:       provider.ProfileProviderModelsUnavailable,
					CredentialID: "shared-anthropic",
					Warnings:     []string{"credential_shared_from:main"},
					Evidence:     []string{"provider_models_unavailable"},
				},
			}, nil
		},
	}, Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{Version: "test-version", GitCommit: "test-sha"}
		},
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"providers", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("profile providers --json: %v\nstdout=%s", err, stdout.String())
	}
	if strings.Contains(stdout.String(), "sk-") || strings.Contains(stdout.String(), "/home/") {
		t.Fatalf("provider readiness leaked secret-looking data:\n%s", stdout.String())
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Active    string                              `json:"active"`
		Providers []provider.ProfileProviderReadiness `json:"providers"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("profile providers stdout must be JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Build.Version != "test-version" || got.Active != "main" {
		t.Fatalf("metadata = %+v active=%q", got.Build, got.Active)
	}
	if len(got.Providers) != 2 {
		t.Fatalf("providers = %d, want 2: %+v", len(got.Providers), got.Providers)
	}
	if got.Providers[0].Status != provider.ProfileProviderReady || got.Providers[0].ProviderID != "openrouter" {
		t.Fatalf("first provider = %+v", got.Providers[0])
	}
	if got.Providers[1].Status != provider.ProfileProviderModelsUnavailable || !containsString(got.Providers[1].Warnings, "credential_shared_from:main") {
		t.Fatalf("second provider = %+v", got.Providers[1])
	}
}

func TestProfileModuleChannelsCommandRendersReadiness(t *testing.T) {
	cmd := NewCommandWithSeams(Seams{
		ReadActiveProfileName: func() (string, error) { return "main", nil },
		ChannelReadiness: func() (gateway.ProfileChannelReadinessReport, error) {
			return gateway.ProfileChannelReadinessReport{
				Bindings: []gateway.ProfileChannelBindingReadiness{
					{
						ProfileID:              "main",
						Channel:                "telegram",
						Ready:                  true,
						CredentialID:           "main-telegram",
						CredentialOwnerProfile: "main",
						CredentialHash:         gateway.TokenCredentialHash("bot-token-that-must-not-leak"),
						SecretRefConfigured:    true,
						SecretRefSource:        "env",
						AllowedChatCount:       1,
						AllowedUserCount:       1,
						RequireMention:         true,
						ToolProgress:           "compact",
					},
					{
						ProfileID:    "ops",
						Channel:      "whatsapp",
						Ready:        false,
						CredentialID: "ops-whatsapp",
						Evidence: []gateway.ProfileChannelReadinessEvidence{
							{Code: gateway.ProfileChannelEvidenceAccessPolicyMissing, ProfileID: "ops", Channel: "whatsapp", CredentialID: "ops-whatsapp", Field: "allowed_chats", Redacted: true},
						},
					},
				},
			}, nil
		},
	}, Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{Version: "test-version", GitCommit: "test-sha"}
		},
	})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"channels", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("profile channels --json: %v\nstdout=%s", err, stdout.String())
	}
	for _, leaked := range []string{"bot-token-that-must-not-leak", "GORMES_MAIN_TELEGRAM_BOT_TOKEN", "12025550123", "/home/"} {
		if strings.Contains(stdout.String(), leaked) {
			t.Fatalf("channel readiness leaked sensitive value %q:\n%s", leaked, stdout.String())
		}
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Active   string                                    `json:"active"`
		Bindings []gateway.ProfileChannelBindingReadiness  `json:"bindings"`
		Evidence []gateway.ProfileChannelReadinessEvidence `json:"evidence"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("profile channels stdout must be JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Build.Version != "test-version" || got.Active != "main" {
		t.Fatalf("metadata = %+v active=%q", got.Build, got.Active)
	}
	if len(got.Bindings) != 2 || !got.Bindings[0].Ready || got.Bindings[0].Channel != "telegram" {
		t.Fatalf("bindings = %+v, want ready telegram then degraded whatsapp", got.Bindings)
	}
	if got.Bindings[0].AllowedChatCount != 1 || got.Bindings[0].AllowedUserCount != 1 || !got.Bindings[0].RequireMention || got.Bindings[0].ToolProgress != "compact" {
		t.Fatalf("telegram binding policy = %+v", got.Bindings[0])
	}
	if got.Bindings[1].Ready || !containsChannelEvidence(got.Bindings[1].Evidence, gateway.ProfileChannelEvidenceAccessPolicyMissing) {
		t.Fatalf("whatsapp binding = %+v, want access-policy evidence", got.Bindings[1])
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsChannelEvidence(values []gateway.ProfileChannelReadinessEvidence, want string) bool {
	for _, value := range values {
		if value.Code == want {
			return true
		}
	}
	return false
}
