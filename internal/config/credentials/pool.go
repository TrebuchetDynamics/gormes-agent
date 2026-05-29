package credentials

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const (
	CredentialAuthAPIKey = "api_key"
	CredentialAuthOAuth  = "oauth"

	CredentialStatusOK        = "ok"
	CredentialStatusExhausted = "exhausted"

	CredentialPoolEvidenceLoaded        = "credential_pool_loaded"
	CredentialPoolEvidenceEmpty         = "credential_pool_empty"
	CredentialPoolEvidenceCorrupt       = "credential_pool_corrupt"
	CredentialPoolEvidenceSelected      = "credential_pool_selected"
	CredentialPoolEvidenceUnavailable   = "credential_pool_unavailable"
	CredentialPoolEvidenceExhausted     = "credential_pool_exhausted"
	CredentialPoolEvidenceLeaseAcquired = "credential_pool_lease_acquired"
	CredentialPoolEvidenceLeaseReleased = "credential_pool_lease_released"
)

const (
	defaultCredentialPoolMaxConcurrent = 2
	DefaultOwnerProfile                = "main"
)

type CredentialPoolStrategy string

const (
	CredentialPoolStrategyFillFirst  CredentialPoolStrategy = "fill_first"
	CredentialPoolStrategyRoundRobin CredentialPoolStrategy = "round_robin"
	CredentialPoolStrategyLeastUsed  CredentialPoolStrategy = "least_used"
	CredentialPoolStrategyRandom     CredentialPoolStrategy = "random"
)

type CredentialPoolOptions struct {
	HermesHome                 string
	Provider                   string
	ProfileID                  string
	Strategy                   CredentialPoolStrategy
	Now                        func() time.Time
	MaxConcurrentPerCredential int
}

type PooledCredential struct {
	ID                 string `json:"id"`
	Label              string `json:"label"`
	AuthType           string `json:"auth_type"`
	Priority           int    `json:"priority"`
	Source             string `json:"source"`
	OwnerProfile       string `json:"owner_profile,omitempty"`
	AccessToken        string `json:"access_token,omitempty"`
	RefreshToken       string `json:"refresh_token,omitempty"`
	LastStatus         string `json:"last_status,omitempty"`
	LastStatusAt       int64  `json:"last_status_at,omitempty"`
	LastErrorCode      int    `json:"last_error_code,omitempty"`
	LastErrorReason    string `json:"last_error_reason,omitempty"`
	LastErrorMessage   string `json:"last_error_message,omitempty"`
	LastErrorResetAt   int64  `json:"last_error_reset_at,omitempty"`
	BaseURL            string `json:"base_url,omitempty"`
	ExpiresAt          string `json:"expires_at,omitempty"`
	ExpiresAtMS        int64  `json:"expires_at_ms,omitempty"`
	LastRefresh        string `json:"last_refresh,omitempty"`
	InferenceBaseURL   string `json:"inference_base_url,omitempty"`
	PortalBaseURL      string `json:"portal_base_url,omitempty"`
	AgentKey           string `json:"agent_key,omitempty"`
	AgentKeyID         string `json:"agent_key_id,omitempty"`
	AgentKeyExpiresAt  string `json:"agent_key_expires_at,omitempty"`
	AgentKeyObtainedAt string `json:"agent_key_obtained_at,omitempty"`
	RequestCount       int    `json:"request_count,omitempty"`
	MaxConcurrentLease int    `json:"max_concurrent_leases,omitempty"`
}

type RedactedCredentialStatus struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	AuthType         string `json:"auth_type"`
	Priority         int    `json:"priority"`
	Source           string `json:"source"`
	OwnerProfile     string `json:"owner_profile,omitempty"`
	LastStatus       string `json:"last_status,omitempty"`
	LastStatusAt     int64  `json:"last_status_at,omitempty"`
	LastErrorCode    int    `json:"last_error_code,omitempty"`
	LastErrorReason  string `json:"last_error_reason,omitempty"`
	LastErrorResetAt int64  `json:"last_error_reset_at,omitempty"`
	RequestCount     int    `json:"request_count,omitempty"`
	ActiveLeaseCount int    `json:"active_lease_count,omitempty"`
	SecretsRedacted  bool   `json:"secrets_redacted"`
}

type CredentialPoolStatus struct {
	Provider string                     `json:"provider"`
	Strategy CredentialPoolStrategy     `json:"strategy"`
	Count    int                        `json:"count"`
	Entries  []RedactedCredentialStatus `json:"entries"`
	Redacted bool                       `json:"redacted"`
}

type CredentialPoolEvidence struct {
	Code          string                 `json:"code"`
	Provider      string                 `json:"provider,omitempty"`
	Strategy      CredentialPoolStrategy `json:"strategy,omitempty"`
	Count         int                    `json:"count,omitempty"`
	FilteredCount int                    `json:"filtered_count,omitempty"`
	Selected      string                 `json:"selected,omitempty"`
	Reason        string                 `json:"reason,omitempty"`
	Message       string                 `json:"message"`
	Redacted      bool                   `json:"redacted"`
}

type CredentialExhaustion struct {
	StatusCode int
	Reason     string
	Message    string
	ResetAt    time.Time
}

type CredentialPoolError struct {
	Code string
	Err  error
}

func (e *CredentialPoolError) Error() string {
	if e == nil {
		return "credential pool: <nil>"
	}
	if e.Err != nil {
		return fmt.Sprintf("credential pool: %s: %v", e.Code, e.Err)
	}
	return fmt.Sprintf("credential pool: %s", e.Code)
}

func (e *CredentialPoolError) Unwrap() error { return e.Err }

type CredentialPool struct {
	mu            sync.Mutex
	hermesHome    string
	provider      string
	profileID     string
	strategy      CredentialPoolStrategy
	now           func() time.Time
	entries       []PooledCredential
	currentID     string
	activeLeases  map[string]int
	maxConcurrent int
	random        *rand.Rand
}

type credentialPoolAuthStore struct {
	Version           int                           `json:"version,omitempty"`
	ActiveProvider    string                        `json:"active_provider,omitempty"`
	Providers         map[string]map[string]any     `json:"providers,omitempty"`
	CredentialPool    map[string][]PooledCredential `json:"credential_pool,omitempty"`
	SuppressedSources map[string][]string           `json:"suppressed_sources,omitempty"`
}

const (
	NousOAuthProvider         = "nous"
	NousOAuthDeviceCodeSource = "device_code"
)

type NousOAuthCredentials struct {
	Label              string
	PortalBaseURL      string
	InferenceBaseURL   string
	ClientID           string
	Scope              string
	TokenType          string
	AccessToken        string
	RefreshToken       string
	ObtainedAt         string
	ExpiresAt          string
	ExpiresIn          int
	AgentKey           string
	AgentKeyID         string
	AgentKeyExpiresAt  string
	AgentKeyExpiresIn  int
	AgentKeyObtainedAt string
}

func SaveCredentialPoolEntries(opts CredentialPoolOptions, entries []PooledCredential) error {
	home, err := credentialPoolHermesHome(opts.HermesHome)
	if err != nil {
		return err
	}
	provider := strings.TrimSpace(opts.Provider)
	if provider == "" {
		return fmt.Errorf("credential pool provider is empty")
	}
	store, err := readCredentialPoolAuthStore(home)
	if err != nil {
		return err
	}
	if store.CredentialPool == nil {
		store.CredentialPool = make(map[string][]PooledCredential)
	}
	store.CredentialPool[provider] = normalizeCredentialEntries(entries)
	return writeCredentialPoolAuthStore(home, store)
}

func SaveNousOAuthCredentials(opts CredentialPoolOptions, creds NousOAuthCredentials) (PooledCredential, error) {
	home, err := credentialPoolHermesHome(opts.HermesHome)
	if err != nil {
		return PooledCredential{}, err
	}
	creds = normalizeNousOAuthCredentials(creds)
	if creds.AccessToken == "" {
		return PooledCredential{}, fmt.Errorf("nous oauth access token is empty")
	}
	store, err := readCredentialPoolAuthStore(home)
	if err != nil {
		return PooledCredential{}, err
	}
	if store.Providers == nil {
		store.Providers = make(map[string]map[string]any)
	}
	store.Providers[NousOAuthProvider] = nousProviderState(creds)
	if store.CredentialPool == nil {
		store.CredentialPool = make(map[string][]PooledCredential)
	}
	entry := PooledCredential{
		ID:                 "nous-device-code",
		Label:              creds.Label,
		AuthType:           CredentialAuthOAuth,
		Source:             NousOAuthDeviceCodeSource,
		AccessToken:        creds.AccessToken,
		RefreshToken:       creds.RefreshToken,
		BaseURL:            creds.InferenceBaseURL,
		InferenceBaseURL:   creds.InferenceBaseURL,
		PortalBaseURL:      creds.PortalBaseURL,
		ExpiresAt:          creds.ExpiresAt,
		AgentKey:           creds.AgentKey,
		AgentKeyID:         creds.AgentKeyID,
		AgentKeyExpiresAt:  creds.AgentKeyExpiresAt,
		AgentKeyObtainedAt: creds.AgentKeyObtainedAt,
		LastStatus:         CredentialStatusOK,
		MaxConcurrentLease: 1,
	}
	entries := normalizeCredentialEntries(store.CredentialPool[NousOAuthProvider])
	next := make([]PooledCredential, 0, len(entries)+1)
	for _, existing := range entries {
		source := strings.TrimSpace(existing.Source)
		if source == NousOAuthDeviceCodeSource || source == "manual:device_code" || source == "manual:dashboard_device_code" {
			continue
		}
		next = append(next, existing)
	}
	next = append(next, entry)
	store.CredentialPool[NousOAuthProvider] = normalizeCredentialEntries(next)
	if err := writeCredentialPoolAuthStore(home, store); err != nil {
		return PooledCredential{}, err
	}
	return entry, nil
}

func ListCredentialPoolProviders(opts CredentialPoolOptions) ([]string, error) {
	home, err := credentialPoolHermesHome(opts.HermesHome)
	if err != nil {
		return nil, err
	}
	store, err := readCredentialPoolAuthStore(home)
	if err != nil {
		return nil, err
	}
	providers := make([]string, 0, len(store.CredentialPool))
	for provider, entries := range store.CredentialPool {
		if strings.TrimSpace(provider) == "" || len(entries) == 0 {
			continue
		}
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	return providers, nil
}

func LoadCredentialPool(opts CredentialPoolOptions) (*CredentialPool, CredentialPoolEvidence, error) {
	home, err := credentialPoolHermesHome(opts.HermesHome)
	if err != nil {
		return nil, CredentialPoolEvidence{}, err
	}
	provider := strings.TrimSpace(opts.Provider)
	if provider == "" {
		return nil, CredentialPoolEvidence{}, fmt.Errorf("credential pool provider is empty")
	}
	pool := &CredentialPool{
		hermesHome:    home,
		provider:      provider,
		profileID:     normalizeCredentialPoolProfileID(opts.ProfileID),
		strategy:      normalizeCredentialPoolStrategy(opts.Strategy),
		now:           opts.Now,
		activeLeases:  make(map[string]int),
		maxConcurrent: opts.MaxConcurrentPerCredential,
		random:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	if pool.now == nil {
		pool.now = time.Now
	}
	if pool.maxConcurrent <= 0 {
		pool.maxConcurrent = defaultCredentialPoolMaxConcurrent
	}
	store, err := readCredentialPoolAuthStore(home)
	if err != nil {
		return pool, CredentialPoolEvidence{
			Code:     CredentialPoolEvidenceCorrupt,
			Provider: provider,
			Strategy: pool.strategy,
			Message:  "credential pool store is corrupt; file preserved for operator recovery",
			Redacted: true,
		}, &CredentialPoolError{Code: CredentialPoolEvidenceCorrupt, Err: err}
	}
	totalCount := 0
	if store.CredentialPool != nil {
		entries := normalizeCredentialEntries(store.CredentialPool[provider])
		totalCount = len(entries)
		pool.entries = filterCredentialEntriesForProfile(entries, pool.profileID)
	}
	code := CredentialPoolEvidenceLoaded
	message := "credential pool loaded"
	if len(pool.entries) == 0 {
		code = CredentialPoolEvidenceEmpty
		message = "credential pool is empty"
	}
	filteredCount := 0
	if pool.profileID != "" {
		filteredCount = len(pool.entries)
	}
	return pool, CredentialPoolEvidence{Code: code, Provider: provider, Strategy: pool.strategy, Count: totalCount, FilteredCount: filteredCount, Message: message, Redacted: true}, nil
}

func (p *CredentialPool) Entries() []PooledCredential {
	p.mu.Lock()
	defer p.mu.Unlock()
	return cloneCredentialEntries(p.entries)
}

func (p *CredentialPool) SetDeterministicRandom(seed int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.random = rand.New(rand.NewSource(seed))
}

func (p *CredentialPool) SetMaxConcurrentPerCredential(max int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if max > 0 {
		p.maxConcurrent = max
	}
}

func (p *CredentialPool) RedactedStatus() CredentialPoolStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	entries := make([]RedactedCredentialStatus, 0, len(p.entries))
	for _, entry := range p.entries {
		entries = append(entries, RedactedCredentialStatus{
			ID:               entry.ID,
			Label:            entry.Label,
			AuthType:         entry.AuthType,
			Priority:         entry.Priority,
			Source:           entry.Source,
			OwnerProfile:     effectiveCredentialOwnerProfile(entry),
			LastStatus:       entry.LastStatus,
			LastStatusAt:     entry.LastStatusAt,
			LastErrorCode:    entry.LastErrorCode,
			LastErrorReason:  entry.LastErrorReason,
			LastErrorResetAt: entry.LastErrorResetAt,
			RequestCount:     entry.RequestCount,
			ActiveLeaseCount: p.activeLeases[entry.ID],
			SecretsRedacted:  true,
		})
	}
	return CredentialPoolStatus{Provider: p.provider, Strategy: p.strategy, Count: len(entries), Entries: entries, Redacted: true}
}

func (p *CredentialPool) Select() (*PooledCredential, CredentialPoolEvidence) {
	p.mu.Lock()
	defer p.mu.Unlock()
	selected := p.selectUnlocked()
	if selected == nil {
		return nil, CredentialPoolEvidence{Code: CredentialPoolEvidenceUnavailable, Provider: p.provider, Strategy: p.strategy, Message: "credential pool has no available credentials", Redacted: true}
	}
	return selected, CredentialPoolEvidence{Code: CredentialPoolEvidenceSelected, Provider: p.provider, Strategy: p.strategy, Count: len(p.entries), Selected: selected.ID, Message: "credential selected", Redacted: true}
}

func (p *CredentialPool) MarkExhaustedAndRotate(exhaustion CredentialExhaustion) (*PooledCredential, CredentialPoolEvidence) {
	p.mu.Lock()
	defer p.mu.Unlock()
	current := p.currentUnlocked()
	if current == nil {
		current = p.selectUnlocked()
	}
	if current == nil {
		return nil, CredentialPoolEvidence{Code: CredentialPoolEvidenceUnavailable, Provider: p.provider, Strategy: p.strategy, Message: "credential pool has no available credentials to exhaust", Redacted: true}
	}
	for i := range p.entries {
		if p.entries[i].ID == current.ID {
			p.entries[i].LastStatus = CredentialStatusExhausted
			p.entries[i].LastStatusAt = p.now().Unix()
			p.entries[i].LastErrorCode = exhaustion.StatusCode
			p.entries[i].LastErrorReason = sanitizedEvidenceText(exhaustion.Reason)
			p.entries[i].LastErrorMessage = sanitizedEvidenceText(exhaustion.Message)
			if !exhaustion.ResetAt.IsZero() {
				p.entries[i].LastErrorResetAt = exhaustion.ResetAt.Unix()
			}
			break
		}
	}
	p.currentID = ""
	_ = p.persistUnlocked()
	next := p.selectUnlocked()
	return next, CredentialPoolEvidence{Code: CredentialPoolEvidenceExhausted, Provider: p.provider, Strategy: p.strategy, Count: len(p.entries), Selected: current.ID, Reason: sanitizedEvidenceText(exhaustion.Reason), Message: "credential marked exhausted and pool rotated", Redacted: true}
}

func (p *CredentialPool) AcquireLease(credentialID string) (string, CredentialPoolEvidence) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if id := strings.TrimSpace(credentialID); id != "" {
		if p.entryByIDUnlocked(id) == nil {
			return "", CredentialPoolEvidence{Code: CredentialPoolEvidenceUnavailable, Provider: p.provider, Strategy: p.strategy, Message: "requested credential is unavailable", Redacted: true}
		}
		p.activeLeases[id]++
		p.currentID = id
		return id, CredentialPoolEvidence{Code: CredentialPoolEvidenceLeaseAcquired, Provider: p.provider, Strategy: p.strategy, Selected: id, Message: "credential lease acquired", Redacted: true}
	}
	available := p.availableEntriesUnlocked(true)
	if len(available) == 0 {
		return "", CredentialPoolEvidence{Code: CredentialPoolEvidenceUnavailable, Provider: p.provider, Strategy: p.strategy, Message: "credential pool has no available credentials to lease", Redacted: true}
	}
	candidates := make([]*PooledCredential, 0, len(available))
	for _, entry := range available {
		limit := entry.MaxConcurrentLease
		if limit <= 0 {
			limit = p.maxConcurrent
		}
		if p.activeLeases[entry.ID] < limit {
			candidates = append(candidates, entry)
		}
	}
	if len(candidates) == 0 {
		candidates = available
	}
	chosen := candidates[0]
	for _, candidate := range candidates[1:] {
		if p.activeLeases[candidate.ID] < p.activeLeases[chosen.ID] || (p.activeLeases[candidate.ID] == p.activeLeases[chosen.ID] && candidate.Priority < chosen.Priority) {
			chosen = candidate
		}
	}
	p.activeLeases[chosen.ID]++
	p.currentID = chosen.ID
	return chosen.ID, CredentialPoolEvidence{Code: CredentialPoolEvidenceLeaseAcquired, Provider: p.provider, Strategy: p.strategy, Selected: chosen.ID, Message: "credential lease acquired", Redacted: true}
}

func (p *CredentialPool) ReleaseLease(credentialID string) CredentialPoolEvidence {
	p.mu.Lock()
	defer p.mu.Unlock()
	id := strings.TrimSpace(credentialID)
	if count := p.activeLeases[id]; count <= 1 {
		delete(p.activeLeases, id)
	} else {
		p.activeLeases[id] = count - 1
	}
	return CredentialPoolEvidence{Code: CredentialPoolEvidenceLeaseReleased, Provider: p.provider, Strategy: p.strategy, Selected: id, Message: "credential lease released", Redacted: true}
}

func (p *CredentialPool) ActiveLeases() map[string]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]int, len(p.activeLeases))
	for key, value := range p.activeLeases {
		out[key] = value
	}
	return out
}

func (p *CredentialPool) selectUnlocked() *PooledCredential {
	available := p.availableEntriesUnlocked(true)
	if len(available) == 0 {
		p.currentID = ""
		return nil
	}
	switch p.strategy {
	case CredentialPoolStrategyRandom:
		selected := available[p.random.Intn(len(available))]
		p.currentID = selected.ID
		return cloneCredential(selected)
	case CredentialPoolStrategyLeastUsed:
		selected := available[0]
		for _, entry := range available[1:] {
			if entry.RequestCount < selected.RequestCount || (entry.RequestCount == selected.RequestCount && entry.Priority < selected.Priority) {
				selected = entry
			}
		}
		for i := range p.entries {
			if p.entries[i].ID == selected.ID {
				p.entries[i].RequestCount++
				selected = &p.entries[i]
				break
			}
		}
		p.currentID = selected.ID
		_ = p.persistUnlocked()
		return cloneCredential(selected)
	case CredentialPoolStrategyRoundRobin:
		selected := available[0]
		for i := range p.entries {
			if p.entries[i].ID == selected.ID {
				p.entries[i].Priority = len(p.entries) - 1
			} else if p.entries[i].Priority > selected.Priority {
				p.entries[i].Priority--
			}
		}
		p.entries = normalizeCredentialEntries(p.entries)
		p.currentID = selected.ID
		_ = p.persistUnlocked()
		return cloneCredential(selected)
	default:
		selected := available[0]
		p.currentID = selected.ID
		return cloneCredential(selected)
	}
}

func (p *CredentialPool) availableEntriesUnlocked(clearExpired bool) []*PooledCredential {
	now := p.now().Unix()
	available := make([]*PooledCredential, 0, len(p.entries))
	changed := false
	for i := range p.entries {
		entry := &p.entries[i]
		if entry.LastStatus == CredentialStatusExhausted {
			resetAt := exhaustedUntil(*entry)
			if resetAt > 0 && now < resetAt {
				continue
			}
			if clearExpired {
				entry.LastStatus = CredentialStatusOK
				entry.LastStatusAt = 0
				entry.LastErrorCode = 0
				entry.LastErrorReason = ""
				entry.LastErrorMessage = ""
				entry.LastErrorResetAt = 0
				changed = true
			}
		}
		available = append(available, entry)
	}
	if changed {
		_ = p.persistUnlocked()
	}
	return available
}

func (p *CredentialPool) currentUnlocked() *PooledCredential {
	if p.currentID == "" {
		return nil
	}
	return p.entryByIDUnlocked(p.currentID)
}

func (p *CredentialPool) entryByIDUnlocked(id string) *PooledCredential {
	for i := range p.entries {
		if p.entries[i].ID == id {
			return &p.entries[i]
		}
	}
	return nil
}

func (p *CredentialPool) persistUnlocked() error {
	store, err := readCredentialPoolAuthStore(p.hermesHome)
	if err != nil {
		return err
	}
	if store.CredentialPool == nil {
		store.CredentialPool = make(map[string][]PooledCredential)
	}
	entries := normalizeCredentialEntries(p.entries)
	if p.profileID != "" {
		entries = mergeProfileFilteredCredentialEntries(normalizeCredentialEntries(store.CredentialPool[p.provider]), entries)
	}
	store.CredentialPool[p.provider] = normalizeCredentialEntries(entries)
	return writeCredentialPoolAuthStore(p.hermesHome, store)
}

func normalizeCredentialPoolProfileID(profileID string) string {
	return strings.TrimSpace(profileID)
}

func filterCredentialEntriesForProfile(entries []PooledCredential, profileID string) []PooledCredential {
	if profileID == "" {
		return entries
	}
	out := make([]PooledCredential, 0, len(entries))
	for _, entry := range entries {
		if credentialEntryVisibleToProfile(entry, profileID) {
			out = append(out, entry)
		}
	}
	return out
}

func credentialEntryVisibleToProfile(entry PooledCredential, profileID string) bool {
	profileID = normalizeCredentialPoolProfileID(profileID)
	if profileID == "" {
		return true
	}
	owner := effectiveCredentialOwnerProfile(entry)
	return owner == DefaultOwnerProfile || owner == profileID
}

func effectiveCredentialOwnerProfile(entry PooledCredential) string {
	owner := strings.TrimSpace(entry.OwnerProfile)
	if owner == "" {
		return DefaultOwnerProfile
	}
	return owner
}

func mergeProfileFilteredCredentialEntries(existing, filtered []PooledCredential) []PooledCredential {
	byID := make(map[string]PooledCredential, len(filtered))
	for _, entry := range filtered {
		byID[entry.ID] = entry
	}
	out := make([]PooledCredential, 0, len(existing)+len(filtered))
	seen := make(map[string]struct{}, len(existing)+len(filtered))
	for _, entry := range existing {
		if replacement, ok := byID[entry.ID]; ok {
			out = append(out, replacement)
		} else {
			out = append(out, entry)
		}
		seen[entry.ID] = struct{}{}
	}
	for _, entry := range filtered {
		if _, ok := seen[entry.ID]; !ok {
			out = append(out, entry)
		}
	}
	return out
}

func normalizeCredentialEntries(entries []PooledCredential) []PooledCredential {
	out := cloneCredentialEntries(entries)
	for i := range out {
		if out[i].ID == "" {
			out[i].ID = fmt.Sprintf("credential-%d", i+1)
		}
		if out[i].Label == "" {
			out[i].Label = out[i].Source
		}
		if out[i].AuthType == "" {
			out[i].AuthType = CredentialAuthAPIKey
		}
		if out[i].Source == "" {
			out[i].Source = "manual"
		}
		out[i].OwnerProfile = strings.TrimSpace(out[i].OwnerProfile)
		out[i].AccessToken = sanitizeCredentialValue("ACCESS_TOKEN", out[i].AccessToken)
		out[i].RefreshToken = sanitizeCredentialValue("REFRESH_TOKEN", out[i].RefreshToken)
		out[i].AgentKey = sanitizeCredentialValue("AGENT_KEY", out[i].AgentKey)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority == out[j].Priority {
			return out[i].ID < out[j].ID
		}
		return out[i].Priority < out[j].Priority
	})
	return out
}

func normalizeNousOAuthCredentials(creds NousOAuthCredentials) NousOAuthCredentials {
	creds.Label = strings.TrimSpace(creds.Label)
	creds.PortalBaseURL = strings.TrimRight(strings.TrimSpace(creds.PortalBaseURL), "/")
	creds.InferenceBaseURL = strings.TrimRight(strings.TrimSpace(creds.InferenceBaseURL), "/")
	creds.ClientID = strings.TrimSpace(creds.ClientID)
	creds.Scope = strings.TrimSpace(creds.Scope)
	creds.TokenType = strings.TrimSpace(creds.TokenType)
	if creds.TokenType == "" {
		creds.TokenType = "Bearer"
	}
	creds.AccessToken = strings.TrimSpace(creds.AccessToken)
	creds.RefreshToken = strings.TrimSpace(creds.RefreshToken)
	creds.ObtainedAt = strings.TrimSpace(creds.ObtainedAt)
	creds.ExpiresAt = strings.TrimSpace(creds.ExpiresAt)
	creds.AgentKey = strings.TrimSpace(creds.AgentKey)
	creds.AgentKeyID = strings.TrimSpace(creds.AgentKeyID)
	creds.AgentKeyExpiresAt = strings.TrimSpace(creds.AgentKeyExpiresAt)
	creds.AgentKeyObtainedAt = strings.TrimSpace(creds.AgentKeyObtainedAt)
	if creds.Label == "" {
		creds.Label = credentialLabelFromJWT(creds.AccessToken, "Nous device code")
	}
	return creds
}

func nousProviderState(creds NousOAuthCredentials) map[string]any {
	state := map[string]any{
		"portal_base_url":       creds.PortalBaseURL,
		"inference_base_url":    creds.InferenceBaseURL,
		"client_id":             creds.ClientID,
		"scope":                 creds.Scope,
		"token_type":            creds.TokenType,
		"access_token":          creds.AccessToken,
		"refresh_token":         creds.RefreshToken,
		"obtained_at":           creds.ObtainedAt,
		"expires_at":            creds.ExpiresAt,
		"expires_in":            creds.ExpiresIn,
		"agent_key":             creds.AgentKey,
		"agent_key_id":          creds.AgentKeyID,
		"agent_key_expires_at":  creds.AgentKeyExpiresAt,
		"agent_key_expires_in":  creds.AgentKeyExpiresIn,
		"agent_key_obtained_at": creds.AgentKeyObtainedAt,
	}
	if creds.Label != "" {
		state["label"] = creds.Label
	}
	return state
}

func credentialLabelFromJWT(accessToken, fallback string) string {
	claims, ok := decodeJWTClaims(accessToken)
	if !ok {
		return fallback
	}
	for _, key := range []string{"email", "preferred_username", "name", "sub"} {
		if value := strings.TrimSpace(fmt.Sprint(claims[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	return fallback
}

func cloneCredentialEntries(entries []PooledCredential) []PooledCredential {
	out := make([]PooledCredential, len(entries))
	copy(out, entries)
	return out
}

func cloneCredential(entry *PooledCredential) *PooledCredential {
	if entry == nil {
		return nil
	}
	cloned := *entry
	return &cloned
}

func normalizeCredentialPoolStrategy(strategy CredentialPoolStrategy) CredentialPoolStrategy {
	switch strategy {
	case CredentialPoolStrategyRoundRobin, CredentialPoolStrategyLeastUsed, CredentialPoolStrategyRandom:
		return strategy
	default:
		return CredentialPoolStrategyFillFirst
	}
}

func exhaustedUntil(entry PooledCredential) int64 {
	if entry.LastErrorResetAt > 0 {
		return entry.LastErrorResetAt
	}
	if entry.LastStatusAt > 0 {
		return entry.LastStatusAt + int64(time.Hour/time.Second)
	}
	return 0
}

func readCredentialPoolAuthStore(hermesHome string) (credentialPoolAuthStore, error) {
	path := filepath.Join(hermesHome, "auth.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return credentialPoolAuthStore{CredentialPool: make(map[string][]PooledCredential)}, nil
	}
	if err != nil {
		return credentialPoolAuthStore{}, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return credentialPoolAuthStore{CredentialPool: make(map[string][]PooledCredential)}, nil
	}
	var store credentialPoolAuthStore
	if err := json.Unmarshal(data, &store); err != nil {
		return credentialPoolAuthStore{}, err
	}
	if store.CredentialPool == nil {
		store.CredentialPool = make(map[string][]PooledCredential)
	}
	return store, nil
}

func writeCredentialPoolAuthStore(hermesHome string, store credentialPoolAuthStore) error {
	if err := os.MkdirAll(hermesHome, 0o700); err != nil {
		return err
	}
	path := filepath.Join(hermesHome, "auth.json")
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	// Write atomically: tempfile → fsync → rename over target.
	// Prevents concurrent readers from observing a partially-written auth.json.
	tmp, err := os.CreateTemp(hermesHome, ".auth-*.json")
	if err != nil {
		return fmt.Errorf("atomic auth write: create tempfile: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { os.Remove(tmpPath) }
	defer cleanup()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("atomic auth write: write tempfile: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("atomic auth write: fsync tempfile: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("atomic auth write: close tempfile: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("atomic auth write: chmod tempfile: %w", err)
	}

	_, err = tools.AtomicReplace(tmpPath, path, tools.AtomicReplaceOptions{
		FirstWriteMode: 0o600,
		Root:           hermesHome,
	})
	if err != nil {
		return fmt.Errorf("atomic auth write: replace: %w", err)
	}
	return nil
}

func credentialPoolHermesHome(input string) (string, error) {
	home := strings.TrimSpace(input)
	if home == "" {
		home = GormesBaseHome()
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return "", err
	}
	return absHome, nil
}

func sanitizedEvidenceText(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	secretHints := []string{"access_token", "refresh_token", "api_key", "authorization", "bearer ", "plain-existing-token", "token-a", "token-b", "token-c"}
	for _, hint := range secretHints {
		if strings.Contains(lower, hint) {
			return "[redacted]"
		}
	}
	if len(trimmed) > 200 {
		return trimmed[:200]
	}
	return trimmed
}

// LoadNousOAuthCredentials reads the persisted Nous OAuth device-code
// credentials from the credential pool auth store and returns a
// NousOAuthCredentials struct suitable for runtime credential resolution.
// It returns an error when the auth store is absent, the Nous provider
// has no device-code entry, or the stored entry is missing required fields.
func LoadNousOAuthCredentials(opts CredentialPoolOptions) (NousOAuthCredentials, error) {
	home, err := credentialPoolHermesHome(opts.HermesHome)
	if err != nil {
		return NousOAuthCredentials{}, err
	}
	store, err := readCredentialPoolAuthStore(home)
	if err != nil {
		return NousOAuthCredentials{}, fmt.Errorf("load nous oauth: read auth store: %w", err)
	}
	entries := store.CredentialPool[NousOAuthProvider]
	if len(entries) == 0 {
		return NousOAuthCredentials{}, fmt.Errorf("load nous oauth: no credential pool entries for provider %q", NousOAuthProvider)
	}
	var entry PooledCredential
	found := false
	for _, e := range entries {
		if strings.TrimSpace(e.Source) == NousOAuthDeviceCodeSource {
			entry = e
			found = true
			break
		}
	}
	if !found {
		return NousOAuthCredentials{}, fmt.Errorf("load nous oauth: no device_code entry for provider %q", NousOAuthProvider)
	}
	if entry.AccessToken == "" {
		return NousOAuthCredentials{}, fmt.Errorf("load nous oauth: stored entry has empty access token")
	}
	creds := NousOAuthCredentials{
		Label:              entry.Label,
		InferenceBaseURL:   entry.InferenceBaseURL,
		PortalBaseURL:      entry.PortalBaseURL,
		AccessToken:        entry.AccessToken,
		RefreshToken:       entry.RefreshToken,
		ExpiresAt:          entry.ExpiresAt,
		AgentKey:           entry.AgentKey,
		AgentKeyID:         entry.AgentKeyID,
		AgentKeyExpiresAt:  entry.AgentKeyExpiresAt,
		AgentKeyObtainedAt: entry.AgentKeyObtainedAt,
		TokenType:          "Bearer",
	}
	if state, ok := store.Providers[NousOAuthProvider]; ok {
		if v, ok := state["portal_base_url"].(string); ok && creds.PortalBaseURL == "" {
			creds.PortalBaseURL = v
		}
		if v, ok := state["inference_base_url"].(string); ok && creds.InferenceBaseURL == "" {
			creds.InferenceBaseURL = v
		}
		if v, ok := state["client_id"].(string); ok {
			creds.ClientID = v
		}
		if v, ok := state["scope"].(string); ok {
			creds.Scope = v
		}
	}
	return creds, nil
}
