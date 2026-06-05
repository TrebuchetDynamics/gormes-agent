package provider

import (
	"errors"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestProfileProviderReadinessSeparatesCredentialsAndCatalogs(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Providers: map[string]config.ProfileProviderCfg{
					"openrouter": {
						Enabled:       true,
						Credential:    "main-openrouter",
						DefaultModel:  "openai/gpt-5.2",
						AllowedModels: []string{"openai/gpt-5.2", "anthropic/claude-sonnet-4.5"},
					},
				},
			},
			"tulin": {
				Enabled: true,
				Providers: map[string]config.ProfileProviderCfg{
					"openrouter": {
						Enabled:       true,
						Credential:    "tulin-openrouter",
						DefaultModel:  "meta-llama/llama-4",
						AllowedModels: []string{"meta-llama/llama-4"},
					},
					"anthropic": {
						Enabled:      true,
						Credential:   "shared-anthropic",
						DefaultModel: "claude-sonnet-4.5",
					},
					"groq": {
						Enabled:      true,
						Credential:   "missing-groq",
						DefaultModel: "llama-3.3-70b-versatile",
					},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-openrouter": {
				Kind:         "provider",
				Provider:     "openrouter",
				OwnerProfile: "main",
				SecretRef:    &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_MAIN_OPENROUTER_API_KEY"},
			},
			"tulin-openrouter": {
				Kind:         "provider",
				Provider:     "openrouter",
				OwnerProfile: "tulin",
				SecretRef:    &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_TULIN_OPENROUTER_API_KEY"},
			},
			"shared-anthropic": {
				Kind:         "provider",
				Provider:     "anthropic",
				OwnerProfile: "main",
				SecretRef:    &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_SHARED_ANTHROPIC_API_KEY"},
			},
		},
	}

	reports := BuildProfileProviderReadiness(cfg, ProfileProviderReadinessOptions{
		SecretEnv: map[string]string{
			"GORMES_MAIN_OPENROUTER_API_KEY":  "sk-main-openrouter",
			"GORMES_TULIN_OPENROUTER_API_KEY": "sk-tulin-openrouter",
			"GORMES_SHARED_ANTHROPIC_API_KEY": "sk-shared-anthropic",
		},
		Catalogs: map[string]ProviderModelCatalogFunc{
			"openrouter": func() ([]string, error) {
				return []string{"zai/glm-4.6", "openai/gpt-5.2", "anthropic/claude-sonnet-4.5", "meta-llama/llama-4"}, nil
			},
			"anthropic": func() ([]string, error) {
				return nil, errors.New("upstream catalog unavailable: token redacted")
			},
		},
	})

	mainOpenRouter := findProfileProviderReadiness(t, reports, "main", "openrouter")
	if mainOpenRouter.Status != ProfileProviderReady || mainOpenRouter.CredentialID != "main-openrouter" {
		t.Fatalf("main openrouter = %+v, want ready with own credential", mainOpenRouter)
	}
	if !reflect.DeepEqual(mainOpenRouter.Models, []string{"anthropic/claude-sonnet-4.5", "meta-llama/llama-4", "openai/gpt-5.2", "zai/glm-4.6"}) {
		t.Fatalf("openrouter full catalog = %#v", mainOpenRouter.Models)
	}
	if mainOpenRouter.SecretRef != "env:GORMES_MAIN_OPENROUTER_API_KEY" {
		t.Fatalf("secret ref = %q, want redacted env ref", mainOpenRouter.SecretRef)
	}

	tulinOpenRouter := findProfileProviderReadiness(t, reports, "tulin", "openrouter")
	if tulinOpenRouter.CredentialID != "tulin-openrouter" || tulinOpenRouter.Status != ProfileProviderReady {
		t.Fatalf("tulin openrouter = %+v, want separate ready credential", tulinOpenRouter)
	}
	if tulinOpenRouter.DefaultModel != "meta-llama/llama-4" {
		t.Fatalf("tulin default model = %q", tulinOpenRouter.DefaultModel)
	}

	shared := findProfileProviderReadiness(t, reports, "tulin", "anthropic")
	if shared.Status != ProfileProviderModelsUnavailable {
		t.Fatalf("shared anthropic status = %q, want catalog degraded", shared.Status)
	}
	if !containsString(shared.Warnings, "credential_shared_from:main") {
		t.Fatalf("shared warnings = %#v, want owner warning", shared.Warnings)
	}
	if !containsString(shared.Evidence, "provider_models_unavailable") {
		t.Fatalf("shared evidence = %#v, want catalog failure evidence", shared.Evidence)
	}

	missing := findProfileProviderReadiness(t, reports, "tulin", "groq")
	if missing.Status != ProfileProviderCredentialMissing {
		t.Fatalf("missing groq status = %q, want missing credential", missing.Status)
	}
	if containsString(missing.Evidence, "GORMES_SHARED_ANTHROPIC_API_KEY") {
		t.Fatalf("missing evidence leaked unrelated secret ref: %#v", missing.Evidence)
	}
}

func TestProfileProviderReadinessMarksMissingSecretRefEnvAsCredentialMissing(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Providers: map[string]config.ProfileProviderCfg{
					"openrouter": {Enabled: true, Credential: "main-openrouter", DefaultModel: "openai/gpt-5.2"},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-openrouter": {
				Kind:      "provider",
				Provider:  "openrouter",
				SecretRef: &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_MAIN_OPENROUTER_API_KEY"},
			},
		},
	}

	reports := BuildProfileProviderReadiness(cfg, ProfileProviderReadinessOptions{SecretEnv: map[string]string{}})
	got := findProfileProviderReadiness(t, reports, "main", "openrouter")
	if got.Status != ProfileProviderCredentialMissing {
		t.Fatalf("openrouter status = %q, want credential missing when SecretRef env is empty: %+v", got.Status, got)
	}
	if got.SecretRef != "env:GORMES_MAIN_OPENROUTER_API_KEY" {
		t.Fatalf("secret ref = %q, want redacted env ref", got.SecretRef)
	}
	if !containsString(got.Evidence, config.SecretRefEvidenceMissing) {
		t.Fatalf("evidence = %#v, want %s", got.Evidence, config.SecretRefEvidenceMissing)
	}
}

func findProfileProviderReadiness(t *testing.T, reports []ProfileProviderReadiness, profileID, providerID string) ProfileProviderReadiness {
	t.Helper()
	for _, report := range reports {
		if report.ProfileID == profileID && report.ProviderID == providerID {
			return report
		}
	}
	t.Fatalf("readiness report for %s/%s missing in %#v", profileID, providerID, reports)
	return ProfileProviderReadiness{}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
