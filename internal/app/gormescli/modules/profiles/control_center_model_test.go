package profiles

import (
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestProfileControlCenterModelListsAllProfilesAndReadiness(t *testing.T) {
	cfg := config.Config{
		ConfigVersion: config.CurrentConfigVersion,
		Profiles: map[string]config.ProfileCfg{
			"tulin": {
				Enabled:    true,
				Name:       "Tulin",
				Workspaces: []string{"/workspace/tulin"},
				Providers: map[string]config.ProfileProviderCfg{
					"openrouter": {Enabled: true, Credential: "shared-openrouter", DefaultModel: "openai/gpt-5.1"},
				},
				Channels: map[string]config.ProfileChannelCfg{
					"telegram": {Enabled: true, Credential: "tulin-telegram", AllowedChats: []string{"222"}},
				},
			},
			"main": {
				Enabled:    true,
				Name:       "",
				Workspaces: []string{"/workspace/main", "/missing/main"},
				Providers: map[string]config.ProfileProviderCfg{
					"openrouter": {Enabled: true, Credential: "shared-openrouter", DefaultModel: "anthropic/claude-sonnet-4.5"},
					"codex":      {Enabled: true},
				},
				Channels: map[string]config.ProfileChannelCfg{
					"telegram": {Enabled: true, Credential: "missing-telegram", AllowedChats: []string{"111"}},
				},
			},
			"sleeping": {
				Enabled: false,
				Name:    "Sleeping",
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"shared-openrouter": {
				Kind:         "provider",
				Provider:     "openrouter",
				OwnerProfile: "main",
				SecretRef:    &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_MAIN_OPENROUTER_API_KEY"},
			},
			"tulin-telegram": {
				Kind:         "channel",
				Channel:      "telegram",
				OwnerProfile: "tulin",
				SecretRef:    &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_TULIN_TELEGRAM_BOT_TOKEN"},
			},
		},
	}

	model := BuildControlCenterModel(cfg, ControlCenterModelOptions{
		WorkspaceExists: func(path string) bool { return !strings.Contains(path, "missing") },
	})

	gotIDs := make([]string, 0, len(model.Profiles))
	for _, profile := range model.Profiles {
		gotIDs = append(gotIDs, profile.ID)
	}
	if want := []string{"main", "tulin", "sleeping"}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("profile order = %#v, want %#v", gotIDs, want)
	}

	main := model.Profiles[0]
	if main.Group != ControlCenterProfileGroupEnabled || main.DisplayName != "" {
		t.Fatalf("main summary = %+v", main)
	}
	assertIssue(t, main.Readiness.Issues, ControlCenterIssueNameNeeded, "")
	assertIssue(t, main.Readiness.Issues, ControlCenterIssueWorkspaceMissing, "/missing/main")
	assertIssue(t, main.Readiness.Issues, ControlCenterIssueProviderCredentialMissing, "codex")
	assertIssue(t, main.Readiness.Issues, ControlCenterIssueChannelCredentialMissing, "telegram")
	assertIssue(t, main.Readiness.Issues, ControlCenterIssueCredentialShared, "shared-openrouter")
	assertAction(t, main.Actions, ControlCenterActionEditProfile)
	assertAction(t, main.Actions, ControlCenterActionAddProvider)
	assertAction(t, main.Actions, ControlCenterActionAddChannel)

	if len(main.Providers) != 2 || main.Providers[0].ID != "codex" || main.Providers[0].Readiness != ControlCenterReadinessMissingCredential {
		t.Fatalf("main providers = %+v", main.Providers)
	}
	if len(main.Channels) != 1 || main.Channels[0].ID != "telegram" || main.Channels[0].Readiness != ControlCenterReadinessMissingCredential {
		t.Fatalf("main channels = %+v", main.Channels)
	}

	tulin := model.Profiles[1]
	if tulin.Readiness.Status != ControlCenterLaneReady {
		t.Fatalf("tulin readiness = %+v, want ready", tulin.Readiness)
	}
	assertIssue(t, tulin.Readiness.Issues, ControlCenterIssueCredentialShared, "shared-openrouter")
	if got := tulin.Providers[0].OwnerProfile; got != "main" {
		t.Fatalf("shared provider owner = %q, want main", got)
	}

	sleeping := model.Profiles[2]
	if sleeping.Group != ControlCenterProfileGroupDisabled || sleeping.Runtime.Status != ControlCenterLaneDisabled {
		t.Fatalf("disabled profile lanes = %+v", sleeping)
	}

	text := model.String()
	for _, forbidden := range []string{"sk-", "bot-token", "raw-secret"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("control center model leaked secret-looking text %q:\n%s", forbidden, text)
		}
	}
}

func TestProfileControlCenterModelReportsSharedChannelCredentialWithoutSecrets(t *testing.T) {
	cfg := config.Config{
		ConfigVersion: config.CurrentConfigVersion,
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Name:    "Main",
				Channels: map[string]config.ProfileChannelCfg{
					"telegram": {Enabled: true, Credential: "shared-telegram"},
				},
			},
			"tulin": {
				Enabled: true,
				Name:    "Tulin",
				Channels: map[string]config.ProfileChannelCfg{
					"telegram": {Enabled: true, Credential: "shared-telegram"},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"shared-telegram": {
				Kind:         "channel",
				Channel:      "telegram",
				OwnerProfile: "main",
				SecretRef:    &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_TELEGRAM_BOT_TOKEN"},
			},
		},
	}

	model := BuildControlCenterModel(cfg, ControlCenterModelOptions{})
	for _, profile := range model.Profiles {
		if len(profile.Channels) != 1 || !profile.Channels[0].Shared || profile.Channels[0].OwnerProfile != "main" {
			t.Fatalf("profile %s channels = %+v, want shared channel credential owned by main", profile.ID, profile.Channels)
		}
		assertIssue(t, profile.Readiness.Issues, ControlCenterIssueCredentialShared, "shared-telegram")
	}
	if strings.Contains(model.String(), "GORMES_TELEGRAM_BOT_TOKEN") {
		t.Fatalf("model string leaked secret ref id:\n%s", model.String())
	}
}

func TestProfileControlCenterModelActionCatalogIsFiniteAndTyped(t *testing.T) {
	catalog := ControlCenterActionCatalog()
	gotCodes := make([]ControlCenterActionCode, 0, len(catalog))
	for _, action := range catalog {
		gotCodes = append(gotCodes, action.Code)
		if !action.Available || strings.TrimSpace(action.Label) == "" {
			t.Fatalf("catalog action = %+v, want available labeled action", action)
		}
	}
	wantCodes := []ControlCenterActionCode{
		ControlCenterActionCreateProfile,
		ControlCenterActionEditProfile,
		ControlCenterActionAddProvider,
		ControlCenterActionAddChannel,
		ControlCenterActionEnableProfile,
		ControlCenterActionDisableProfile,
		ControlCenterActionMigrateLegacyConfig,
		ControlCenterActionApplyDraft,
		ControlCenterActionDiscardDraft,
	}
	if !reflect.DeepEqual(gotCodes, wantCodes) {
		t.Fatalf("catalog codes = %#v, want %#v", gotCodes, wantCodes)
	}
	unknown := controlCenterAction(ControlCenterActionCode("shell rm -rf"))
	if unknown.Available || unknown.Label != "unsupported action" {
		t.Fatalf("unknown action = %+v, want unavailable unsupported action", unknown)
	}
}

func TestProfileControlCenterModelSurfacesLegacyMigrationWhenV2Missing(t *testing.T) {
	model := BuildControlCenterModel(config.Config{LegacyConfigVersion: 1}, ControlCenterModelOptions{LegacyMigrationAvailable: true})

	if len(model.Profiles) != 0 {
		t.Fatalf("legacy-only model profiles = %+v, want none", model.Profiles)
	}
	assertIssue(t, model.Issues, ControlCenterIssueLegacyConfigDetected, "")
	assertIssue(t, model.Issues, ControlCenterIssueMigrationAvailable, "")
	assertAction(t, model.Actions, ControlCenterActionMigrateLegacyConfig)
}

func assertIssue(t *testing.T, issues []ControlCenterIssue, code ControlCenterIssueCode, contains string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code != code {
			continue
		}
		if contains == "" || strings.Contains(issue.Subject, contains) || strings.Contains(issue.Message, contains) || strings.Contains(issue.CredentialID, contains) {
			return
		}
	}
	t.Fatalf("issue %s containing %q not found in %+v", code, contains, issues)
}

func assertAction(t *testing.T, actions []ControlCenterAction, code ControlCenterActionCode) {
	t.Helper()
	for _, action := range actions {
		if action.Code == code {
			return
		}
	}
	t.Fatalf("action %s not found in %+v", code, actions)
}
