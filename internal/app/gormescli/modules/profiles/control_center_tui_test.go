package profiles

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
		"actions: edit profile, add provider, add channel, disable profile, apply",
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
