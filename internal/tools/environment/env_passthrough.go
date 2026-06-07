package environment

import (
	"sort"
	"strings"
	"sync"
)

// ProviderCredentialEnvBlocklist names Hermes/Gormes-managed provider
// credentials that skills must not be allowed to smuggle into sandboxed child
// processes. The list intentionally covers the common provider API key/token
// variables from Hermes' _HERMES_PROVIDER_ENV_BLOCKLIST; non-provider third
// party keys such as TENOR_API_KEY remain registerable.
var ProviderCredentialEnvBlocklist = map[string]struct{}{
	"ANTHROPIC_API_KEY":     {},
	"ANTHROPIC_AUTH_TOKEN":  {},
	"ANTHROPIC_TOKEN":       {},
	"AWS_ACCESS_KEY_ID":     {},
	"AWS_SECRET_ACCESS_KEY": {},
	"AZURE_OPENAI_API_KEY":  {},
	"COHERE_API_KEY":        {},
	"GEMINI_API_KEY":        {},
	"GOOGLE_API_KEY":        {},
	"GROQ_API_KEY":          {},
	"MISTRAL_API_KEY":       {},
	"NOUS_API_KEY":          {},
	"OPENAI_API_KEY":        {},
	"OPENROUTER_API_KEY":    {},
	"XAI_API_KEY":           {},
}

// EnvPassthroughRegistry is a session-scoped allowlist for environment
// variables that may pass through to sandboxed tools. It mirrors Hermes'
// ContextVar-backed registry with an explicit Go object so callers can keep
// separate sessions isolated and tests can prove no cross-session bleed.
type EnvPassthroughRegistry struct {
	mu         sync.RWMutex
	registered map[string]struct{}
	configured map[string]struct{}
	blocklist  map[string]struct{}
}

// NewEnvPassthroughRegistry creates a registry with config-sourced allowlist
// entries. Provider credentials in the configured list are ignored, matching
// Hermes' safety rule that operator config also cannot override sandbox
// credential scrubbing.
func NewEnvPassthroughRegistry(configured []string) *EnvPassthroughRegistry {
	r := &EnvPassthroughRegistry{
		registered: map[string]struct{}{},
		configured: map[string]struct{}{},
		blocklist:  cloneEnvNameSet(ProviderCredentialEnvBlocklist),
	}
	for _, raw := range configured {
		candidate := classifyEnvPassthroughCandidate(raw, r.blocklist)
		if !candidate.Valid || candidate.ProviderCredential {
			continue
		}
		r.configured[candidate.Name] = struct{}{}
	}
	return r
}

// Register adds skill-declared environment variables to the session allowlist
// and returns the sanitized names that were rejected as provider credentials.
func (r *EnvPassthroughRegistry) Register(names []string) []string {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	blocked := make([]string, 0)
	for _, raw := range names {
		candidate := classifyEnvPassthroughCandidate(raw, r.blocklist)
		if !candidate.Valid {
			continue
		}
		if candidate.ProviderCredential {
			blocked = append(blocked, candidate.Name)
			continue
		}
		r.registered[candidate.Name] = struct{}{}
	}
	return blocked
}

// IsAllowed reports whether name was registered for this session or listed in
// the config allowlist.
func (r *EnvPassthroughRegistry) IsAllowed(name string) bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	candidate := classifyEnvPassthroughCandidate(name, r.blocklist)
	if !candidate.Valid || candidate.ProviderCredential {
		return false
	}
	if _, ok := r.registered[candidate.Name]; ok {
		return true
	}
	_, ok := r.configured[candidate.Name]
	return ok
}

// All returns the sorted union of session-registered and configured allowlist
// names. Sorting keeps tool/runtime evidence deterministic.
func (r *EnvPassthroughRegistry) All() []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	return sortedEnvNameUnion(r.registered, r.configured)
}

// ClearRegistered resets only the skill/session-scoped allowlist. Configured
// allowlist entries remain, matching Hermes' clear_env_passthrough semantics.
func (r *EnvPassthroughRegistry) ClearRegistered() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	clear(r.registered)
}

func (r *EnvPassthroughRegistry) isProviderCredential(name string) bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	return isProviderCredentialName(name, r.blocklist)
}

type envPassthroughCandidate struct {
	Name               string
	Valid              bool
	ProviderCredential bool
}

func classifyEnvPassthroughCandidate(raw string, blocklist map[string]struct{}) envPassthroughCandidate {
	name := normalizeEnvPassthroughName(raw)
	if !isValidEnvPassthroughName(name) {
		return envPassthroughCandidate{}
	}
	return envPassthroughCandidate{
		Name:               name,
		Valid:              true,
		ProviderCredential: isProviderCredentialName(name, blocklist),
	}
}

func normalizeEnvPassthroughName(name string) string {
	return strings.TrimSpace(name)
}

func isValidEnvPassthroughName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		b := name[i]
		if i == 0 {
			if !isEnvPassthroughNameStart(b) {
				return false
			}
			continue
		}
		if !isEnvPassthroughNamePart(b) {
			return false
		}
	}
	return true
}

func isEnvPassthroughNameStart(b byte) bool {
	return b == '_' || ('A' <= b && b <= 'Z') || ('a' <= b && b <= 'z')
}

func isEnvPassthroughNamePart(b byte) bool {
	return isEnvPassthroughNameStart(b) || ('0' <= b && b <= '9')
}

func isProviderCredentialName(name string, blocklist map[string]struct{}) bool {
	if blocklist == nil {
		return false
	}
	_, ok := blocklist[canonicalProviderCredentialName(name)]
	return ok
}

func cloneEnvNameSet(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for name := range in {
		canonical := canonicalProviderCredentialName(name)
		if canonical == "" {
			continue
		}
		out[canonical] = struct{}{}
	}
	return out
}

func sortedEnvNameUnion(sets ...map[string]struct{}) []string {
	size := 0
	for _, set := range sets {
		size += len(set)
	}
	union := make(map[string]struct{}, size)
	for _, set := range sets {
		for name := range set {
			union[name] = struct{}{}
		}
	}
	out := make([]string, 0, len(union))
	for name := range union {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func canonicalProviderCredentialName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}
