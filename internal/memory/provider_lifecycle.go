package memory

import (
	"context"
	"regexp"
	"strings"
	"sync"
)

const memoryProviderUnavailableCode = "memory_provider_unavailable"

// LifecycleProvider is the native Go memory-provider lifecycle surface used to
// adapt Goncho/local memory into Hermes-compatible turn hooks without depending
// on Hermes Python runtime services.
type LifecycleProvider interface {
	Name() string
	Initialize(context.Context, MemoryProviderSession) error
	Prefetch(context.Context, MemoryPrefetchRequest) (string, error)
	SyncTurn(context.Context, MemoryTurn) error
	PreCompress(context.Context, []MemoryProviderMessage) (string, error)
	MemoryWrite(context.Context, MemoryWriteEvent) error
	Delegation(context.Context, MemoryDelegationEvent) error
	Shutdown(context.Context) error
}

// MemoryProviderSession describes the session-scoped metadata available when a
// memory provider is initialized.
type MemoryProviderSession struct {
	SessionID string
	Platform  string
	Model     string
	Provider  string
}

// MemoryPrefetchRequest asks providers for bounded context before a turn.
type MemoryPrefetchRequest struct {
	Query     string
	SessionID string
}

// MemoryTurn is the user/assistant exchange mirrored after a completed turn.
type MemoryTurn struct {
	SessionID string
	User      string
	Assistant string
	Platform  string
	Model     string
	Provider  string
}

// MemoryProviderMessage is the minimal role/content transcript shape exposed to
// pre-compression hooks.
type MemoryProviderMessage struct {
	Role    string
	Content string
}

// MemoryWriteEvent mirrors built-in memory-tool writes into external providers.
type MemoryWriteEvent struct {
	Action   string
	Target   string
	Content  string
	Metadata map[string]any
}

// MemoryDelegationEvent mirrors completed child-agent work into providers.
type MemoryDelegationEvent struct {
	Task           string
	Result         string
	ChildSessionID string
	Metadata       map[string]any
}

// MemoryProviderEvidence records non-fatal provider degradation without leaking
// raw provider errors or host paths.
type MemoryProviderEvidence struct {
	Provider  string
	Operation string
	Code      string
	Message   string
	Degraded  bool
}

// MemoryProviderLifecycle fans Hermes-compatible memory lifecycle hooks out to
// registered native providers. Provider failures are non-fatal: callers receive
// bounded evidence while other providers continue.
type MemoryProviderLifecycle struct {
	mu          sync.Mutex
	providers   []LifecycleProvider
	initialized map[string]struct{}
}

func NewMemoryProviderLifecycle() *MemoryProviderLifecycle {
	return &MemoryProviderLifecycle{initialized: make(map[string]struct{})}
}

func (l *MemoryProviderLifecycle) Register(provider LifecycleProvider) {
	if provider == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.providers = append(l.providers, provider)
}

func (l *MemoryProviderLifecycle) Initialize(ctx context.Context, session MemoryProviderSession) []MemoryProviderEvidence {
	var evidence []MemoryProviderEvidence
	for _, provider := range l.snapshot() {
		key := provider.Name() + "\x00" + session.SessionID
		if l.markInitialized(key) {
			continue
		}
		if err := provider.Initialize(ctx, session); err != nil {
			evidence = append(evidence, providerEvidence(provider.Name(), "initialize", err))
		}
	}
	return evidence
}

func (l *MemoryProviderLifecycle) Prefetch(ctx context.Context, req MemoryPrefetchRequest) (string, []MemoryProviderEvidence) {
	var parts []string
	var evidence []MemoryProviderEvidence
	for _, provider := range l.snapshot() {
		part, err := provider.Prefetch(ctx, req)
		if err != nil {
			evidence = append(evidence, providerEvidence(provider.Name(), "prefetch", err))
			continue
		}
		if strings.TrimSpace(part) != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "\n\n"), evidence
}

func (l *MemoryProviderLifecycle) SyncTurn(ctx context.Context, turn MemoryTurn) []MemoryProviderEvidence {
	var evidence []MemoryProviderEvidence
	for _, provider := range l.snapshot() {
		if err := provider.SyncTurn(ctx, turn); err != nil {
			evidence = append(evidence, providerEvidence(provider.Name(), "sync_turn", err))
		}
	}
	return evidence
}

func (l *MemoryProviderLifecycle) PreCompress(ctx context.Context, messages []MemoryProviderMessage) (string, []MemoryProviderEvidence) {
	var parts []string
	var evidence []MemoryProviderEvidence
	for _, provider := range l.snapshot() {
		part, err := provider.PreCompress(ctx, messages)
		if err != nil {
			evidence = append(evidence, providerEvidence(provider.Name(), "pre_compress", err))
			continue
		}
		if strings.TrimSpace(part) != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "\n\n"), evidence
}

func (l *MemoryProviderLifecycle) MemoryWrite(ctx context.Context, event MemoryWriteEvent) []MemoryProviderEvidence {
	var evidence []MemoryProviderEvidence
	for _, provider := range l.snapshot() {
		if err := provider.MemoryWrite(ctx, event); err != nil {
			evidence = append(evidence, providerEvidence(provider.Name(), "memory_write", err))
		}
	}
	return evidence
}

func (l *MemoryProviderLifecycle) Delegation(ctx context.Context, event MemoryDelegationEvent) []MemoryProviderEvidence {
	var evidence []MemoryProviderEvidence
	for _, provider := range l.snapshot() {
		if err := provider.Delegation(ctx, event); err != nil {
			evidence = append(evidence, providerEvidence(provider.Name(), "delegation", err))
		}
	}
	return evidence
}

func (l *MemoryProviderLifecycle) Shutdown(ctx context.Context) []MemoryProviderEvidence {
	providers := l.snapshot()
	var evidence []MemoryProviderEvidence
	for i := len(providers) - 1; i >= 0; i-- {
		provider := providers[i]
		if err := provider.Shutdown(ctx); err != nil {
			evidence = append(evidence, providerEvidence(provider.Name(), "shutdown", err))
		}
	}
	return evidence
}

func (l *MemoryProviderLifecycle) snapshot() []LifecycleProvider {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]LifecycleProvider, len(l.providers))
	copy(out, l.providers)
	return out
}

func (l *MemoryProviderLifecycle) markInitialized(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.initialized[key]; ok {
		return true
	}
	l.initialized[key] = struct{}{}
	return false
}

func providerEvidence(provider, operation string, err error) MemoryProviderEvidence {
	message := "provider hook failed"
	if err != nil {
		message = sanitizeProviderEvidenceMessage(err.Error())
	}
	return MemoryProviderEvidence{Provider: provider, Operation: operation, Code: memoryProviderUnavailableCode, Message: message, Degraded: true}
}

var sensitiveEvidencePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)plain-token-[a-z0-9._-]+`),
	regexp.MustCompile(`(?i)\b(token|api[_-]?key|authorization|cookie|password|secret)\b[^\s,;]*`),
	regexp.MustCompile(`(?i)(/home|/tmp|/var|/root)/[^\s,;]+`),
}

func sanitizeProviderEvidenceMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "provider hook failed"
	}
	for _, pattern := range sensitiveEvidencePatterns {
		message = pattern.ReplaceAllString(message, "[redacted]")
	}
	if len(message) > 160 {
		message = strings.TrimSpace(message[:160]) + "…"
	}
	return message
}
