package configapp

import (
	"reflect"
	"testing"

	cfg "github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type stubEditorRunner struct{ found map[string]bool }

func (s stubEditorRunner) LookPath(name string) (string, bool) { return name, s.found[name] }
func (s stubEditorRunner) Run(_, _ string) error               { return nil }

func TestPickEditorHonorsEditorThenVisualThenFallback(t *testing.T) {
	t.Setenv("EDITOR", "ed-custom")
	t.Setenv("VISUAL", "visual-custom")
	if got := pickEditor(stubEditorRunner{found: map[string]bool{"ed-custom": true}}); got != "ed-custom" {
		t.Fatalf("EDITOR pick = %q", got)
	}

	t.Setenv("EDITOR", "")
	if got := pickEditor(stubEditorRunner{found: map[string]bool{"visual-custom": true}}); got != "visual-custom" {
		t.Fatalf("VISUAL pick = %q", got)
	}

	t.Setenv("VISUAL", "")
	if got := pickEditor(stubEditorRunner{found: map[string]bool{"vim": true}}); got != "vim" {
		t.Fatalf("fallback pick = %q", got)
	}
}

func TestConfigProfileMigrationProjectionHelpers(t *testing.T) {
	profiles := configProfileMigrationProfileIDs([]cfg.ProfileMigrationV2ProfileAddition{{ID: "dev"}, {ID: "ops"}})
	if !reflect.DeepEqual(profiles, []string{"dev", "ops"}) {
		t.Fatalf("profiles = %#v", profiles)
	}
	secrets := configProfileMigrationSecretTargets([]cfg.ProfileMigrationV2SecretMovement{{TargetEnv: "OPENAI_API_KEY"}})
	if !reflect.DeepEqual(secrets, []string{"OPENAI_API_KEY"}) {
		t.Fatalf("secrets = %#v", secrets)
	}
}
