package passthrough

import (
	"sort"
	"strings"
	"sync"
)

// defaultProviderCredentialEnvBlocklistNames names Hermes/Gormes-managed
// provider credentials that skills must not be allowed to smuggle into
// sandboxed child processes. Keep provider credential families complete; for
// example, AWS credentials are an access-key/secret/session-token triplet.
var defaultProviderCredentialEnvBlocklistNames = []string{
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_AUTH_TOKEN",
	"ANTHROPIC_TOKEN",
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_SESSION_TOKEN",
	"AZURE_OPENAI_API_KEY",
	"COHERE_API_KEY",
	"GEMINI_API_KEY",
	"GOOGLE_API_KEY",
	"GROQ_API_KEY",
	"MISTRAL_API_KEY",
	"NOUS_API_KEY",
	"OPENAI_API_KEY",
	"OPENROUTER_API_KEY",
	"XAI_API_KEY",
}

// ProviderCredentialEnvBlocklist is the mutable provider credential blocklist
// used when new session registries are constructed. Existing registries keep a
// construction-time snapshot so later mutations cannot cross session bounds.
var ProviderCredentialEnvBlocklist = envNameSetFromList(defaultProviderCredentialEnvBlocklistNames)

// Registry is a session-scoped allowlist for environment variables that may
// pass through to sandboxed tools.
type Registry struct {
	mu         sync.RWMutex
	registered map[string]struct{}
	configured map[string]struct{}
	blocklist  map[string]struct{}
}

// NewRegistry creates a registry with config-sourced allowlist entries.
func NewRegistry(configured []string) *Registry {
	return NewRegistryWithBlocklist(configured, ProviderCredentialEnvBlocklist)
}

// NewRegistryWithBlocklist creates a registry from an explicit provider
// credential blocklist snapshot source. It exists so compatibility facades can
// preserve their exported mutable blocklist variable.
func NewRegistryWithBlocklist(configured []string, blocklist map[string]struct{}) *Registry {
	r := &Registry{
		registered: map[string]struct{}{},
		configured: map[string]struct{}{},
		blocklist:  cloneEnvNameSet(blocklist),
	}
	for _, raw := range configured {
		candidate := classifyCandidate(raw, r.blocklist)
		if !candidate.Valid || candidate.ProviderCredential {
			continue
		}
		r.configured[candidate.Name] = struct{}{}
	}
	return r
}

// Register adds skill-declared environment variables to the session allowlist
// and returns the sanitized names that were rejected as provider credentials.
func (r *Registry) Register(names []string) []string {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	blocked := make([]string, 0)
	for _, raw := range names {
		candidate := classifyCandidate(raw, r.blocklist)
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
func (r *Registry) IsAllowed(name string) bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	candidate := classifyCandidate(name, r.blocklist)
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
func (r *Registry) All() []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	return sortedEnvNameUnion(r.registered, r.configured)
}

// ClearRegistered resets only the skill/session-scoped allowlist. Configured
// allowlist entries remain.
func (r *Registry) ClearRegistered() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	clear(r.registered)
}

type candidate struct {
	Name               string
	Valid              bool
	ProviderCredential bool
}

func classifyCandidate(raw string, blocklist map[string]struct{}) candidate {
	name := normalizeName(raw)
	if !isValidName(name) {
		return candidate{}
	}
	return candidate{
		Name:               name,
		Valid:              true,
		ProviderCredential: isProviderCredentialName(name, blocklist),
	}
}

func normalizeName(name string) string {
	return strings.TrimSpace(name)
}

func isValidName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		b := name[i]
		if i == 0 {
			if !isNameStart(b) {
				return false
			}
			continue
		}
		if !isNamePart(b) {
			return false
		}
	}
	return true
}

func isNameStart(b byte) bool {
	return b == '_' || ('A' <= b && b <= 'Z') || ('a' <= b && b <= 'z')
}

func isNamePart(b byte) bool {
	return isNameStart(b) || ('0' <= b && b <= '9')
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

func envNameSetFromList(names []string) map[string]struct{} {
	out := make(map[string]struct{}, len(names))
	for _, name := range names {
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
