package provider

import (
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type ProfileProviderStatus string

const (
	ProfileProviderReady             ProfileProviderStatus = "ready"
	ProfileProviderCredentialMissing ProfileProviderStatus = "provider_credential_missing"
	ProfileProviderModelsUnavailable ProfileProviderStatus = "provider_models_unavailable"
)

type ProviderModelCatalogFunc func() ([]string, error)

type ProfileProviderReadinessOptions struct {
	Catalogs             map[string]ProviderModelCatalogFunc
	SecretEnv            map[string]string
	SkipSecretValidation bool
}

type ProfileProviderReadiness struct {
	ProfileID     string                `json:"profile_id"`
	ProviderID    string                `json:"provider_id"`
	Status        ProfileProviderStatus `json:"status"`
	CredentialID  string                `json:"credential_id,omitempty"`
	OwnerProfile  string                `json:"owner_profile,omitempty"`
	SecretRef     string                `json:"secret_ref,omitempty"`
	DefaultModel  string                `json:"default_model,omitempty"`
	AllowedModels []string              `json:"allowed_models,omitempty"`
	Models        []string              `json:"models,omitempty"`
	Endpoint      string                `json:"endpoint,omitempty"`
	Warnings      []string              `json:"warnings,omitempty"`
	Evidence      []string              `json:"evidence,omitempty"`
}

func BuildProfileProviderReadiness(cfg config.Config, opts ProfileProviderReadinessOptions) []ProfileProviderReadiness {
	profileIDs := make([]string, 0, len(cfg.Profiles))
	for id, profile := range cfg.Profiles {
		if profile.Enabled {
			profileIDs = append(profileIDs, id)
		}
	}
	sort.Strings(profileIDs)

	reports := make([]ProfileProviderReadiness, 0)
	for _, profileID := range profileIDs {
		profile := cfg.Profiles[profileID]
		providers := sortedProfileProviderIDs(profile.Providers)
		for _, providerID := range providers {
			providerCfg := profile.Providers[providerID]
			if !providerCfg.Enabled {
				continue
			}
			reports = append(reports, buildProfileProviderReadiness(profileID, providerID, providerCfg, cfg.Credentials, cfg.Secrets, opts))
		}
	}
	return reports
}

func buildProfileProviderReadiness(profileID, providerID string, providerCfg config.ProfileProviderCfg, credentials map[string]config.CredentialCfg, secrets config.SecretsCfg, opts ProfileProviderReadinessOptions) ProfileProviderReadiness {
	providerID = normalizeProviderID(providerID)
	credentialID := strings.TrimSpace(providerCfg.Credential)
	report := ProfileProviderReadiness{
		ProfileID:     profileID,
		ProviderID:    providerID,
		Status:        ProfileProviderReady,
		CredentialID:  credentialID,
		DefaultModel:  strings.TrimSpace(providerCfg.DefaultModel),
		AllowedModels: cleanUniqueSorted(providerCfg.AllowedModels),
		Endpoint:      strings.TrimSpace(providerCfg.Endpoint),
	}

	credential, ok := credentials[credentialID]
	if credentialID == "" || !ok || strings.ToLower(strings.TrimSpace(credential.Kind)) != "provider" {
		report.Status = ProfileProviderCredentialMissing
		report.Evidence = append(report.Evidence, "provider_credential_missing")
		return report
	}

	report.OwnerProfile = strings.TrimSpace(credential.OwnerProfile)
	if report.OwnerProfile != "" && report.OwnerProfile != profileID {
		report.Warnings = append(report.Warnings, "credential_shared_from:"+report.OwnerProfile)
	}
	credentialProvider := normalizeProviderID(credential.Provider)
	if credentialProvider != "" && credentialProvider != providerID {
		report.Status = ProfileProviderCredentialMissing
		report.Evidence = append(report.Evidence, "provider_credential_mismatch")
		return report
	}
	if credential.SecretRef != nil {
		report.SecretRef = redactedSecretRef(*credential.SecretRef)
		if !opts.SkipSecretValidation {
			if evidence, err := validateProfileProviderSecretRef(*credential.SecretRef, secrets, opts); err != nil {
				report.Status = ProfileProviderCredentialMissing
				if strings.TrimSpace(evidence.Code) != "" {
					report.Evidence = append(report.Evidence, evidence.Code)
				}
				return report
			}
		}
	} else {
		report.Evidence = append(report.Evidence, "provider_secret_ref_missing")
		if !opts.SkipSecretValidation {
			report.Status = ProfileProviderCredentialMissing
			return report
		}
	}

	models, err := loadProviderModels(providerID, report.AllowedModels, opts)
	if err != nil {
		report.Status = ProfileProviderModelsUnavailable
		report.Evidence = append(report.Evidence, "provider_models_unavailable")
		return report
	}
	report.Models = models
	return report
}

func validateProfileProviderSecretRef(ref config.SecretRef, secrets config.SecretsCfg, opts ProfileProviderReadinessOptions) (config.SecretRefEvidence, error) {
	resolver := config.NewSecretResolver(config.SecretResolverConfig{Secrets: secrets, Env: opts.SecretEnv})
	_, evidence, err := resolver.ResolveString(ref)
	return evidence, err
}

func loadProviderModels(providerID string, allowed []string, opts ProfileProviderReadinessOptions) ([]string, error) {
	if opts.Catalogs != nil {
		if catalog := opts.Catalogs[providerID]; catalog != nil {
			models, err := catalog()
			if err != nil {
				return nil, err
			}
			return cleanUniqueSorted(models), nil
		}
	}
	return cleanUniqueSorted(allowed), nil
}

func sortedProfileProviderIDs(providers map[string]config.ProfileProviderCfg) []string {
	ids := make([]string, 0, len(providers))
	for id := range providers {
		if normalized := normalizeProviderID(id); normalized != "" {
			ids = append(ids, normalized)
		}
	}
	sort.Strings(ids)
	return ids
}

func normalizeProviderID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func cleanUniqueSorted(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func redactedSecretRef(ref config.SecretRef) string {
	return string(ref.Source) + ":" + strings.TrimSpace(ref.ID)
}
