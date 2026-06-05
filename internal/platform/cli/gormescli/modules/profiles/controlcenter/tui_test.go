package controlcenter

import (
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestProfileControlCenterTUIScreenUsesStableScreenReaderText(t *testing.T) {
	model := BuildControlCenterModel(config.Config{
		ConfigVersion: config.CurrentConfigVersion,
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled:    true,
				Name:       "Main desk",
				Workspaces: []string{"/workspace/main"},
				Providers: map[string]config.ProfileProviderCfg{
					"openrouter": {Enabled: true, Credential: "main-openrouter"},
				},
			},
			"sleeping": {
				Enabled: false,
				Name:    "Sleeping",
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-openrouter": {
				Kind:         "provider",
				Provider:     "openrouter",
				OwnerProfile: "main",
				SecretRef:    &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_MAIN_OPENROUTER_API_KEY"},
			},
		},
	}, ControlCenterModelOptions{})

	screen := BuildControlCenterTUIScreen(model, ControlCenterTUIScreenOptions{SelectedProfileID: "main"})

	if screen.Title != "Profile Control Center" || screen.SelectedProfileID != "main" {
		t.Fatalf("screen metadata = %+v", screen)
	}
	if len(screen.Rows) != 2 || !screen.Rows[0].Selected || screen.Rows[0].ProfileID != "main" || screen.Rows[1].ProfileID != "sleeping" {
		t.Fatalf("screen rows = %+v", screen.Rows)
	}
	text := screen.AccessibleText
	for _, want := range []string{
		"Profile Control Center",
		"selected profile: main",
		"main — Main desk — enabled",
		"lanes: runtime=ready readiness=ready activity=unknown",
		"actions: create profile, edit profile, add provider, add channel, disable profile, apply, discard",
		"sleeping — Sleeping — disabled",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("accessible text missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"GORMES_MAIN_OPENROUTER_API_KEY", "sk-", "bot-token", "raw-secret"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("accessible text leaked secret-looking value %q:\n%s", forbidden, text)
		}
	}
}

func TestProfileControlCenterTUIScreenGroupsDetailsAndExcludesLiveOps(t *testing.T) {
	cfg := config.Config{
		ConfigVersion: config.CurrentConfigVersion,
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled:    true,
				Name:       "Main desk",
				Workspaces: []string{"/workspace/main"},
				Providers: map[string]config.ProfileProviderCfg{
					"openrouter": {Enabled: true, Credential: "main-openrouter"},
				},
				Channels: map[string]config.ProfileChannelCfg{
					"telegram": {Enabled: true, Credential: "main-telegram"},
				},
			},
			"sleeping": {Enabled: false, Name: "Sleeping"},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-openrouter": {Kind: "provider", Provider: "openrouter", OwnerProfile: "main", SecretRef: &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_MAIN_OPENROUTER_API_KEY"}},
			"main-telegram":   {Kind: "channel", Channel: "telegram", OwnerProfile: "main", SecretRef: &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_MAIN_TELEGRAM_BOT_TOKEN"}},
		},
	}

	model := BuildControlCenterModel(cfg, ControlCenterModelOptions{})
	screen := BuildControlCenterTUIScreen(model, ControlCenterTUIScreenOptions{SelectedProfileID: "main"})
	text := screen.AccessibleText
	for _, want := range []string{
		"enabled profiles:",
		"main — Main desk — enabled selected",
		"disabled profiles:",
		"sleeping — Sleeping — disabled",
		"details for profile: main",
		"workspaces: /workspace/main",
		"providers: openrouter credential=main-openrouter owner_profile=main readiness=ready",
		"channels: telegram credential=main-telegram owner_profile=main readiness=ready",
		"actions: create profile, edit profile, add provider, add channel, disable profile, apply, discard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("accessible text missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"GORMES_MAIN_OPENROUTER_API_KEY", "GORMES_MAIN_TELEGRAM_BOT_TOKEN", "start", "stop", "restart", "reset", "command"} {
		if strings.Contains(strings.ToLower(text), strings.ToLower(forbidden)) {
			t.Fatalf("accessible text contains forbidden %q:\n%s", forbidden, text)
		}
	}
}

func TestProfileControlCenterTUIScreenExposesMigrationStateAndGlobalActions(t *testing.T) {
	model := BuildControlCenterModel(config.Config{LegacyConfigVersion: 1}, ControlCenterModelOptions{LegacyMigrationAvailable: true})
	screen := BuildControlCenterTUIScreen(model, ControlCenterTUIScreenOptions{})
	text := screen.AccessibleText
	for _, want := range []string{
		"migration: legacy_config_detected, migration_available",
		"actions: create profile, migrate legacy profile config, apply, discard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("accessible text missing %q:\n%s", want, text)
		}
	}
}

func TestProfileDraftStagesCreateWorkspaceCredentialsApplyAndDiscard(t *testing.T) {
	cfg := config.Config{
		ConfigVersion: config.CurrentConfigVersion,
		Profiles: map[string]config.ProfileCfg{
			"main": {Enabled: true, Name: ""},
		},
	}

	draft := NewControlCenterDraft(cfg)
	if err := draft.SetProfileDisplayName("main", " Main desk "); err != nil {
		t.Fatalf("SetProfileDisplayName: %v", err)
	}
	if err := draft.AddProfile("tulin", " Tulin "); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	if err := draft.SetProfileWorkspaces("tulin", []string{" /workspace/tulin ", "/workspace/tulin", ""}); err != nil {
		t.Fatalf("SetProfileWorkspaces: %v", err)
	}
	if err := draft.SetCredential("tulin-openrouter", config.CredentialCfg{Kind: "provider", Provider: "openrouter", OwnerProfile: "tulin", SecretRef: &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_TULIN_OPENROUTER_API_KEY"}}); err != nil {
		t.Fatalf("SetCredential: %v", err)
	}
	if err := draft.SetCredential("tulin-telegram", config.CredentialCfg{Kind: "channel", Channel: "telegram", OwnerProfile: "tulin", SecretRef: &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_TULIN_TELEGRAM_BOT_TOKEN"}}); err != nil {
		t.Fatalf("SetCredential channel: %v", err)
	}
	if err := draft.AssignProviderCredential("tulin", "openrouter", "tulin-openrouter"); err != nil {
		t.Fatalf("AssignProviderCredential: %v", err)
	}
	if err := draft.SetProfileProviderModels("tulin", "openrouter", " meta-llama/llama-4 ", []string{"openai/gpt-5.2", "meta-llama/llama-4", "openai/gpt-5.2"}); err != nil {
		t.Fatalf("SetProfileProviderModels: %v", err)
	}
	if err := draft.AssignChannelCredential("tulin", "telegram", "tulin-telegram"); err != nil {
		t.Fatalf("AssignChannelCredential: %v", err)
	}
	if err := draft.SetProfileChannelPolicy("tulin", "telegram", []string{" 222 ", "222", "333"}, []string{"6586915095", "6586915095"}, true, " compact "); err != nil {
		t.Fatalf("SetProfileChannelPolicy: %v", err)
	}

	previewText := strings.Join(RenderControlCenterDraftPreview(draft.Preview()), "\n")
	for _, want := range []string{
		"profile main name: \"\" -> \"Main desk\"",
		"profile tulin created: enabled=true name=\"Tulin\"",
		"profile tulin workspaces: [] -> [/workspace/tulin]",
		"profile tulin provider openrouter credential: \"\" -> \"tulin-openrouter\"",
		"profile tulin provider openrouter default_model:  -> meta-llama/llama-4",
		"profile tulin provider openrouter allowed_models: [] -> [openai/gpt-5.2 meta-llama/llama-4]",
		"profile tulin channel telegram credential: \"\" -> \"tulin-telegram\"",
		"profile tulin channel telegram allowed_chats: [] -> [222 333]",
		"profile tulin channel telegram allowed_users: [] -> [6586915095]",
		"profile tulin channel telegram require_mention: false -> true",
		"profile tulin channel telegram tool_progress:  -> compact",
		"credential tulin-openrouter secret_ref: none -> redacted_ref(env)",
		"credential tulin-telegram secret_ref: none -> redacted_ref(env)",
	} {
		if !strings.Contains(previewText, want) {
			t.Fatalf("preview missing %q:\n%s", want, previewText)
		}
	}
	for _, forbidden := range []string{"GORMES_TULIN_OPENROUTER_API_KEY", "GORMES_TULIN_TELEGRAM_BOT_TOKEN"} {
		if strings.Contains(previewText, forbidden) {
			t.Fatalf("preview leaked secret ref id %q:\n%s", forbidden, previewText)
		}
	}

	applied, _, err := draft.Apply()
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if cfg.Profiles["main"].Name != "" || len(cfg.Profiles) != 1 {
		t.Fatalf("draft mutated base before apply: %+v", cfg.Profiles)
	}
	if got := applied.Profiles["tulin"].Workspaces; !reflect.DeepEqual(got, []string{"/workspace/tulin"}) {
		t.Fatalf("applied tulin workspaces = %#v", got)
	}
	providerCfg := applied.Profiles["tulin"].Providers["openrouter"]
	if got := providerCfg.Credential; got != "tulin-openrouter" {
		t.Fatalf("applied provider credential = %q", got)
	}
	if providerCfg.DefaultModel != "meta-llama/llama-4" {
		t.Fatalf("applied provider default model = %q", providerCfg.DefaultModel)
	}
	if got, want := providerCfg.AllowedModels, []string{"openai/gpt-5.2", "meta-llama/llama-4"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("applied provider allowed models = %#v, want %#v", got, want)
	}
	channelCfg := applied.Profiles["tulin"].Channels["telegram"]
	if got, want := channelCfg.AllowedChats, []string{"222", "333"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("applied channel allowed chats = %#v, want %#v", got, want)
	}
	if got, want := channelCfg.AllowedUsers, []string{"6586915095"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("applied channel allowed users = %#v, want %#v", got, want)
	}
	if !channelCfg.RequireMention || channelCfg.ToolProgress != "compact" {
		t.Fatalf("applied channel policy = %+v", channelCfg)
	}
	if discarded := draft.Discard(); !reflect.DeepEqual(discarded, cfg) {
		t.Fatalf("discard = %+v, want original %+v", discarded, cfg)
	}
}

func TestProfileDraftStagesAndAppliesNameWithoutMutatingBase(t *testing.T) {
	cfg := config.Config{
		ConfigVersion: config.CurrentConfigVersion,
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled:    true,
				Name:       "",
				Workspaces: []string{"/workspace/main"},
			},
		},
	}

	draft := NewControlCenterDraft(cfg)
	if err := draft.SetProfileDisplayName("main", " Main desk "); err != nil {
		t.Fatalf("SetProfileDisplayName: %v", err)
	}
	if cfg.Profiles["main"].Name != "" {
		t.Fatalf("draft mutated base config before apply: %+v", cfg.Profiles["main"])
	}

	preview := draft.Preview()
	wantPreview := []ControlCenterDraftChange{{ProfileID: "main", Field: "name", Before: "", After: "Main desk"}}
	if !reflect.DeepEqual(preview, wantPreview) {
		t.Fatalf("preview = %+v, want %+v", preview, wantPreview)
	}

	applied, changes, err := draft.Apply()
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !reflect.DeepEqual(changes, wantPreview) {
		t.Fatalf("changes = %+v, want %+v", changes, wantPreview)
	}
	if applied.Profiles["main"].Name != "Main desk" {
		t.Fatalf("applied profile name = %q", applied.Profiles["main"].Name)
	}
	if cfg.Profiles["main"].Name != "" {
		t.Fatalf("apply mutated base config: %+v", cfg.Profiles["main"])
	}
}
