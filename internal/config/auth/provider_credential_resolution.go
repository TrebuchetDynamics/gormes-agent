package auth

import (
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// ProviderCredentialStatus is the public status vocabulary shared by runtime
// callers that need secret material and read-model callers that need redacted
// availability evidence.
type ProviderCredentialStatus string

const (
	ProviderCredentialConfigured  ProviderCredentialStatus = "configured"
	ProviderCredentialMissing     ProviderCredentialStatus = "missing_credential"
	ProviderCredentialNotRequired ProviderCredentialStatus = "not_required"
)

// ProviderCredentialSource names the winning adapter behind the provider
// credential resolution seam. Values are evidence only; secret material stays
// in ProviderCredentialResolution.Value for runtime callers.
type ProviderCredentialSource string

const (
	ProviderCredentialSourceNone      ProviderCredentialSource = "none"
	ProviderCredentialSourceEnv       ProviderCredentialSource = "env"
	ProviderCredentialSourceSecretRef ProviderCredentialSource = "secret_ref"
	ProviderCredentialSourceInline    ProviderCredentialSource = "inline"
	ProviderCredentialSourceManifest  ProviderCredentialSource = "provider_manifest"
	ProviderCredentialSourcePool      ProviderCredentialSource = "credential_pool"
	ProviderCredentialSourceCodex     ProviderCredentialSource = "codex_oauth"
)

// ProviderCredentialRequest describes everything a caller currently has to know
// to resolve provider credential availability. LookupEnv is injectable so tests
// and router read models can cross the same seam as production runtime code.
type ProviderCredentialRequest struct {
	Provider          string
	ProfileID         string
	APIKey            string
	APIKeyEnv         string
	APIKeyRef         *credentials.SecretRef
	Secrets           credentials.SecretsCfg
	Local             bool
	Optional          bool
	LookupEnv         func(string) (string, bool)
	CredentialHome    string
	UseCredentialPool bool
	UseCodexOAuth     bool
}

// ProviderCredentialResolution separates runtime secret material (Value) from
// redacted evidence. Callers that report status should ignore Value.
type ProviderCredentialResolution struct {
	Status       ProviderCredentialStatus
	Source       ProviderCredentialSource
	Value        string
	Available    bool
	Evidence     credentials.SecretRefEvidence
	PoolEvidence credentials.CredentialPoolEvidence
}

// ResolveProviderCredential centralizes provider credential source ordering:
// route/env override, SecretRef, inline key, provider manifest env, then an
// optional credential pool/Codex auth check for CLI readiness surfaces.
func ResolveProviderCredential(req ProviderCredentialRequest) ProviderCredentialResolution {
	lookupEnv := req.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	if req.Local && req.Optional && strings.TrimSpace(req.APIKey) == "" && strings.TrimSpace(req.APIKeyEnv) == "" && req.APIKeyRef == nil {
		return ProviderCredentialResolution{Status: ProviderCredentialNotRequired, Source: ProviderCredentialSourceNone, Available: true}
	}
	if env := strings.TrimSpace(req.APIKeyEnv); env != "" {
		value, ok := lookupEnv(env)
		value = strings.TrimSpace(value)
		if ok && value != "" {
			return ProviderCredentialResolution{
				Status:    ProviderCredentialConfigured,
				Source:    ProviderCredentialSourceEnv,
				Value:     value,
				Available: true,
				Evidence:  credentials.SecretRefEvidence{Code: credentials.SecretRefEvidenceResolved, Source: string(credentials.SecretRefSourceEnv), Provider: credentials.DefaultSecretProviderAlias, ID: env, Redacted: true},
			}
		}
		return ProviderCredentialResolution{Status: ProviderCredentialMissing, Source: ProviderCredentialSourceEnv, Evidence: credentials.SecretRefEvidence{Code: credentials.SecretRefEvidenceMissing, Source: string(credentials.SecretRefSourceEnv), Provider: credentials.DefaultSecretProviderAlias, ID: env, Redacted: true}}
	}
	if req.APIKeyRef != nil {
		ref := credentials.NormalizeSecretRef(*req.APIKeyRef)
		if ref.Source == credentials.SecretRefSourceEnv {
			evidence := credentials.SecretRefEvidence{Source: string(ref.Source), Provider: ref.Provider, ID: ref.ID, Redacted: true}
			if err := credentials.ValidateSecretRef(ref); err != nil {
				evidence.Code = credentials.SecretRefEvidenceInvalid
				return ProviderCredentialResolution{Status: ProviderCredentialMissing, Source: ProviderCredentialSourceSecretRef, Evidence: evidence}
			}
			value, ok := lookupEnv(ref.ID)
			value = strings.TrimSpace(value)
			if ok && value != "" {
				evidence.Code = credentials.SecretRefEvidenceResolved
				return ProviderCredentialResolution{Status: ProviderCredentialConfigured, Source: ProviderCredentialSourceSecretRef, Value: value, Available: true, Evidence: evidence}
			}
			evidence.Code = credentials.SecretRefEvidenceMissing
			return ProviderCredentialResolution{Status: ProviderCredentialMissing, Source: ProviderCredentialSourceSecretRef, Evidence: evidence}
		}
		resolver := credentials.NewSecretResolver(credentials.SecretResolverConfig{Secrets: req.Secrets})
		value, evidence, err := resolver.ResolveString(ref)
		evidence.Redacted = true
		if err == nil && strings.TrimSpace(value) != "" {
			return ProviderCredentialResolution{Status: ProviderCredentialConfigured, Source: ProviderCredentialSourceSecretRef, Value: strings.TrimSpace(value), Available: true, Evidence: evidence}
		}
		return ProviderCredentialResolution{Status: ProviderCredentialMissing, Source: ProviderCredentialSourceSecretRef, Evidence: evidence}
	}
	if value := strings.TrimSpace(req.APIKey); value != "" {
		return ProviderCredentialResolution{Status: ProviderCredentialConfigured, Source: ProviderCredentialSourceInline, Value: value, Available: true, Evidence: credentials.SecretRefEvidence{Code: credentials.SecretRefEvidenceResolved, Redacted: true}}
	}
	if value, env := providerManifestCredential(req.Provider, lookupEnv); value != "" {
		return ProviderCredentialResolution{
			Status:    ProviderCredentialConfigured,
			Source:    ProviderCredentialSourceManifest,
			Value:     value,
			Available: true,
			Evidence:  credentials.SecretRefEvidence{Code: credentials.SecretRefEvidenceResolved, Source: string(credentials.SecretRefSourceEnv), Provider: credentials.DefaultSecretProviderAlias, ID: env, Redacted: true},
		}
	}
	if req.UseCodexOAuth && strings.EqualFold(strings.TrimSpace(req.Provider), credentials.CodexOAuthProvider) {
		status, err := credentials.NewCodexOAuthStateStore(credentials.CodexOAuthStateStoreOptions{}).CheckAuth()
		if err == nil && status.Authenticated {
			return ProviderCredentialResolution{Status: ProviderCredentialConfigured, Source: ProviderCredentialSourceCodex, Available: true, PoolEvidence: credentials.CredentialPoolEvidence{Code: credentials.CredentialPoolEvidenceLoaded, Provider: credentials.CodexOAuthProvider, Redacted: true}}
		}
	}
	if req.UseCredentialPool && strings.TrimSpace(req.Provider) != "" {
		pool, evidence, err := credentials.LoadCredentialPool(credentials.CredentialPoolOptions{HermesHome: req.CredentialHome, Provider: strings.TrimSpace(req.Provider), ProfileID: req.ProfileID})
		if err == nil {
			for _, entry := range pool.Entries() {
				if providerCredentialAvailable(entry) {
					return ProviderCredentialResolution{Status: ProviderCredentialConfigured, Source: ProviderCredentialSourcePool, Value: providerCredentialMaterial(entry), Available: true, PoolEvidence: evidence}
				}
			}
		}
		if evidence.Code != "" {
			return ProviderCredentialResolution{Status: ProviderCredentialMissing, Source: ProviderCredentialSourcePool, PoolEvidence: evidence}
		}
	}
	return ProviderCredentialResolution{Status: ProviderCredentialMissing, Source: ProviderCredentialSourceNone}
}

func providerManifestCredential(provider string, lookupEnv func(string) (string, bool)) (string, string) {
	entry, ok := llm.ResolveProviderManifestEntry(provider)
	if !ok {
		return "", ""
	}
	for _, env := range entry.EnvVars {
		value, ok := lookupEnv(env)
		if ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), env
		}
	}
	return "", ""
}

func providerCredentialAvailable(entry credentials.PooledCredential) bool {
	return providerCredentialMaterial(entry) != "" || strings.TrimSpace(entry.RefreshToken) != ""
}

func providerCredentialMaterial(entry credentials.PooledCredential) string {
	return strings.TrimSpace(firstNonEmpty(entry.AccessToken, entry.AgentKey))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
