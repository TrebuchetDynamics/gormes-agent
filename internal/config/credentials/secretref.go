package credentials

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type SecretRefSource string

const (
	SecretRefSourceEnv  SecretRefSource = "env"
	SecretRefSourceFile SecretRefSource = "file"
	SecretRefSourceExec SecretRefSource = "exec"

	DefaultSecretProviderAlias = "default"
	SecretProviderModeJSON     = "json"
	SecretProviderModeSingle   = "single_value"

	SecretRefEvidenceResolved             = "secret_ref_resolved"
	SecretRefEvidenceMissing              = "secret_ref_missing"
	SecretRefEvidenceInvalid              = "secret_ref_invalid"
	SecretRefEvidenceProviderUnconfigured = "secret_provider_unconfigured"
	SecretRefEvidenceProviderMismatch     = "secret_provider_source_mismatch"
	SecretRefEvidenceInsecurePath         = "secret_provider_insecure_path"
	SecretRefEvidenceReadFailed           = "secret_provider_read_failed"
	SecretRefEvidenceUnsupported          = "secret_ref_unsupported"
)

const defaultSecretFileMaxBytes = 1024 * 1024

var (
	secretProviderAliasPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	envSecretRefIDPattern      = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,127}$`)
	fileSecretRefSegmentBad    = regexp.MustCompile(`~(?:[^01]|$)`)
)

type SecretRef struct {
	Source   SecretRefSource `json:"source" toml:"source" yaml:"source"`
	Provider string          `json:"provider" toml:"provider" yaml:"provider"`
	ID       string          `json:"id" toml:"id" yaml:"id"`
}

type SecretsCfg struct {
	Defaults  SecretProviderDefaults       `toml:"defaults" yaml:"defaults"`
	Providers map[string]SecretProviderCfg `toml:"providers" yaml:"providers"`
}

type SecretProviderDefaults struct {
	Env  string `toml:"env" yaml:"env"`
	File string `toml:"file" yaml:"file"`
	Exec string `toml:"exec" yaml:"exec"`
}

type SecretProviderCfg struct {
	Source            SecretRefSource `toml:"source" yaml:"source"`
	Path              string          `toml:"path" yaml:"path"`
	Mode              string          `toml:"mode" yaml:"mode"`
	Allowlist         []string        `toml:"allowlist" yaml:"allowlist"`
	MaxBytes          int64           `toml:"max_bytes" yaml:"max_bytes"`
	AllowInsecurePath bool            `toml:"allow_insecure_path" yaml:"allow_insecure_path"`
}

type SecretResolverConfig struct {
	Secrets SecretsCfg
	Env     map[string]string
}

type SecretRefEvidence struct {
	Code     string `json:"code"`
	Source   string `json:"source,omitempty"`
	Provider string `json:"provider,omitempty"`
	ID       string `json:"id,omitempty"`
	Redacted bool   `json:"redacted"`
}

type SecretResolver struct {
	cfg SecretResolverConfig
}

func NewSecretResolver(cfg SecretResolverConfig) *SecretResolver {
	return &SecretResolver{cfg: cfg}
}

func (r *SecretResolver) ResolveString(ref SecretRef) (string, SecretRefEvidence, error) {
	ref = normalizeSecretRef(ref)
	evidence := SecretRefEvidence{Source: string(ref.Source), Provider: ref.Provider, ID: ref.ID, Redacted: true}
	if err := validateSecretRef(ref); err != nil {
		evidence.Code = SecretRefEvidenceInvalid
		return "", evidence, secretRefError(evidence, err.Error())
	}

	provider, errEvidence, err := r.resolveProvider(ref)
	if err != nil {
		return "", errEvidence, err
	}

	switch ref.Source {
	case SecretRefSourceEnv:
		return r.resolveEnvString(ref, provider)
	case SecretRefSourceFile:
		return r.resolveFileString(ref, provider)
	case SecretRefSourceExec:
		evidence.Code = SecretRefEvidenceUnsupported
		return "", evidence, secretRefError(evidence, "exec SecretRef providers are not implemented in Gormes yet")
	default:
		evidence.Code = SecretRefEvidenceInvalid
		return "", evidence, secretRefError(evidence, "unknown SecretRef source")
	}
}

func (r *SecretResolver) resolveProvider(ref SecretRef) (SecretProviderCfg, SecretRefEvidence, error) {
	evidence := SecretRefEvidence{Source: string(ref.Source), Provider: ref.Provider, ID: ref.ID, Redacted: true}
	if provider, ok := r.cfg.Secrets.Providers[ref.Provider]; ok {
		provider.Source = normalizeSecretRefSource(provider.Source)
		if provider.Source != ref.Source {
			evidence.Code = SecretRefEvidenceProviderMismatch
			return SecretProviderCfg{}, evidence, secretRefError(evidence, fmt.Sprintf("provider source %q does not match ref source %q", provider.Source, ref.Source))
		}
		return provider, evidence, nil
	}
	if ref.Source == SecretRefSourceEnv && ref.Provider == r.defaultProviderAlias(SecretRefSourceEnv) {
		return SecretProviderCfg{Source: SecretRefSourceEnv}, evidence, nil
	}
	evidence.Code = SecretRefEvidenceProviderUnconfigured
	return SecretProviderCfg{}, evidence, secretRefError(evidence, "secret provider is not configured")
}

func (r *SecretResolver) resolveEnvString(ref SecretRef, provider SecretProviderCfg) (string, SecretRefEvidence, error) {
	evidence := SecretRefEvidence{Code: SecretRefEvidenceResolved, Source: string(ref.Source), Provider: ref.Provider, ID: ref.ID, Redacted: true}
	if len(provider.Allowlist) > 0 && !stringInSlice(ref.ID, provider.Allowlist) {
		evidence.Code = SecretRefEvidenceInvalid
		return "", evidence, secretRefError(evidence, "environment variable is not allowlisted for provider")
	}
	value, ok := r.envLookup(ref.ID)
	if !ok || strings.TrimSpace(value) == "" {
		evidence.Code = SecretRefEvidenceMissing
		return "", evidence, secretRefError(evidence, "environment variable is missing or empty")
	}
	return value, evidence, nil
}

func (r *SecretResolver) resolveFileString(ref SecretRef, provider SecretProviderCfg) (string, SecretRefEvidence, error) {
	evidence := SecretRefEvidence{Code: SecretRefEvidenceResolved, Source: string(ref.Source), Provider: ref.Provider, ID: ref.ID, Redacted: true}
	path := strings.TrimSpace(provider.Path)
	if path == "" || !filepath.IsAbs(path) {
		evidence.Code = SecretRefEvidenceInsecurePath
		return "", evidence, secretRefError(evidence, "file provider path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil {
		evidence.Code = SecretRefEvidenceReadFailed
		return "", evidence, secretRefError(evidence, "file provider path is not readable")
	}
	if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		evidence.Code = SecretRefEvidenceInsecurePath
		return "", evidence, secretRefError(evidence, "file provider path must be a regular file")
	}
	if !provider.AllowInsecurePath && info.Mode().Perm()&0o077 != 0 {
		evidence.Code = SecretRefEvidenceInsecurePath
		return "", evidence, secretRefError(evidence, "file provider permissions are too open")
	}
	maxBytes := provider.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultSecretFileMaxBytes
	}
	if info.Size() > maxBytes {
		evidence.Code = SecretRefEvidenceReadFailed
		return "", evidence, secretRefError(evidence, "file provider payload exceeds max_bytes")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		evidence.Code = SecretRefEvidenceReadFailed
		return "", evidence, secretRefError(evidence, "file provider read failed")
	}
	text := strings.TrimPrefix(string(body), "\ufeff")
	mode := strings.TrimSpace(provider.Mode)
	if mode == "" {
		mode = SecretProviderModeJSON
	}
	switch mode {
	case SecretProviderModeJSON:
		var root any
		if err := json.Unmarshal([]byte(text), &root); err != nil {
			evidence.Code = SecretRefEvidenceReadFailed
			return "", evidence, secretRefError(evidence, "file provider payload is not valid JSON")
		}
		value, err := readSecretJSONPointer(root, ref.ID)
		if err != nil {
			evidence.Code = SecretRefEvidenceMissing
			return "", evidence, secretRefError(evidence, err.Error())
		}
		secret, ok := value.(string)
		if !ok || strings.TrimSpace(secret) == "" {
			evidence.Code = SecretRefEvidenceMissing
			return "", evidence, secretRefError(evidence, "file provider value is missing or not a string")
		}
		return secret, evidence, nil
	case SecretProviderModeSingle:
		if ref.ID != "value" {
			evidence.Code = SecretRefEvidenceInvalid
			return "", evidence, secretRefError(evidence, `single_value file provider requires id "value"`)
		}
		return strings.TrimSuffix(text, "\n"), evidence, nil
	default:
		evidence.Code = SecretRefEvidenceInvalid
		return "", evidence, secretRefError(evidence, "unsupported file provider mode")
	}
}

func (r *SecretResolver) defaultProviderAlias(source SecretRefSource) string {
	switch source {
	case SecretRefSourceEnv:
		return firstSecretRefNonEmpty(r.cfg.Secrets.Defaults.Env, DefaultSecretProviderAlias)
	case SecretRefSourceFile:
		return firstSecretRefNonEmpty(r.cfg.Secrets.Defaults.File, DefaultSecretProviderAlias)
	case SecretRefSourceExec:
		return firstSecretRefNonEmpty(r.cfg.Secrets.Defaults.Exec, DefaultSecretProviderAlias)
	default:
		return DefaultSecretProviderAlias
	}
}

func (r *SecretResolver) envLookup(key string) (string, bool) {
	if r.cfg.Env != nil {
		value, ok := r.cfg.Env[key]
		return value, ok
	}
	return os.LookupEnv(key)
}

func validateSecretRef(ref SecretRef) error {
	if ref.Source == "" {
		return fmt.Errorf("SecretRef source is required")
	}
	if ref.Provider == "" || !secretProviderAliasPattern.MatchString(ref.Provider) {
		return fmt.Errorf("SecretRef provider must match ^[a-z][a-z0-9_-]{0,63}$")
	}
	if ref.ID == "" {
		return fmt.Errorf("SecretRef id is required")
	}
	switch ref.Source {
	case SecretRefSourceEnv:
		if !envSecretRefIDPattern.MatchString(ref.ID) {
			return fmt.Errorf("env SecretRef id must match ^[A-Z][A-Z0-9_]{0,127}$")
		}
	case SecretRefSourceFile:
		if !isValidFileSecretRefID(ref.ID) {
			return fmt.Errorf("file SecretRef id must be an absolute JSON pointer or value")
		}
	case SecretRefSourceExec:
		return fmt.Errorf("exec SecretRef source is reserved for a future side-effect-controlled resolver")
	default:
		return fmt.Errorf("unknown SecretRef source %q", ref.Source)
	}
	return nil
}

func normalizeSecretRef(ref SecretRef) SecretRef {
	ref.Source = normalizeSecretRefSource(ref.Source)
	ref.Provider = strings.TrimSpace(ref.Provider)
	ref.ID = strings.TrimSpace(ref.ID)
	if ref.Provider == "" {
		ref.Provider = DefaultSecretProviderAlias
	}
	return ref
}

func normalizeSecretRefSource(source SecretRefSource) SecretRefSource {
	return SecretRefSource(strings.ToLower(strings.TrimSpace(string(source))))
}

func isValidFileSecretRefID(id string) bool {
	if id == "value" {
		return true
	}
	if !strings.HasPrefix(id, "/") {
		return false
	}
	for _, segment := range strings.Split(strings.TrimPrefix(id, "/"), "/") {
		if fileSecretRefSegmentBad.MatchString(segment) {
			return false
		}
	}
	return true
}

func readSecretJSONPointer(root any, pointer string) (any, error) {
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("file-backed secret id must be an absolute JSON pointer")
	}
	current := root
	for _, raw := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		token := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[token]
			if !ok {
				return nil, fmt.Errorf("JSON pointer segment %q does not exist", token)
			}
			current = next
		case []any:
			return nil, fmt.Errorf("JSON pointer arrays are not supported for SecretRef ids")
		default:
			return nil, fmt.Errorf("JSON pointer segment %q does not exist", token)
		}
	}
	return current, nil
}

func secretRefError(evidence SecretRefEvidence, message string) error {
	return fmt.Errorf("%s source=%s provider=%s id=%s: %s", evidence.Code, evidence.Source, evidence.Provider, evidence.ID, message)
}

func stringInSlice(value string, candidates []string) bool {
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == value {
			return true
		}
	}
	return false
}

func firstSecretRefNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// NormalizeSecretRef normalizes SecretRef metadata for compatibility shims and callers
// that need validation-compatible defaults without resolving secret material.
func NormalizeSecretRef(ref SecretRef) SecretRef {
	return normalizeSecretRef(ref)
}

// NormalizeSecretRefSource normalizes a SecretRef source token.
func NormalizeSecretRefSource(source SecretRefSource) SecretRefSource {
	return normalizeSecretRefSource(source)
}

// ValidateSecretRef validates a SecretRef without resolving secret material.
func ValidateSecretRef(ref SecretRef) error {
	return validateSecretRef(ref)
}
