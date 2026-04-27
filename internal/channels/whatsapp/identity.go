package whatsapp

import (
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const botIdentityUnresolvedReason = "bot_identity_unresolved"

// WhatsAppIdentifierEvidence identifies degraded identity safety outcomes.
type WhatsAppIdentifierEvidence string

const (
	// WhatsAppIdentifierUnsafeEvidence is returned when a raw WhatsApp peer ID
	// is not safe to use for canonical identity or alias graph construction.
	WhatsAppIdentifierUnsafeEvidence WhatsAppIdentifierEvidence = "whatsapp_identifier_unsafe"
)

// IdentityContext carries runtime-specific identity inputs that are known
// before send/reconnect code is wired.
type IdentityContext struct {
	Runtime              RuntimeKind
	AccountMode          AccountMode
	BotIDs               []string
	NativeBotID          string
	AliasMappings        []IdentityAlias
	ReplyPrefix          string
	RecentSentMessageIDs []string
}

// IdentityAlias maps bridge-observed LID and phone forms for the same person.
type IdentityAlias struct {
	From string
	To   string
}

// InboundDecision identifies whether an inbound WhatsApp message can enter the
// gateway manager or must be stopped at the adapter boundary.
type InboundDecision string

const (
	InboundDecisionRoute              InboundDecision = "route"
	InboundDecisionDrop               InboundDecision = "drop"
	InboundDecisionSuppressSelfChat   InboundDecision = "suppress_self_chat"
	InboundDecisionUnresolvedIdentity InboundDecision = "unresolved_identity"
)

// SelfChatSuppressionReason is the stable reason surfaced for self-message
// drops that would otherwise create gateway loops.
type SelfChatSuppressionReason string

const (
	SelfChatSuppressionBotOwnMessage         SelfChatSuppressionReason = "bot_own_message"
	SelfChatSuppressionAgentEcho             SelfChatSuppressionReason = "agent_echo"
	SelfChatSuppressionBotIdentityUnresolved SelfChatSuppressionReason = botIdentityUnresolvedReason
)

// IdentityStatus is the status payload a future gateway status command can
// report when bot/self identity cannot be resolved safely.
type IdentityStatus struct {
	Source   IdentitySource
	Resolved bool
	BotID    string
	RawBotID string
	Reason   string
}

// SessionIdentity captures stable gateway peer IDs while retaining raw
// WhatsApp JIDs needed for outbound delivery.
type SessionIdentity struct {
	ChatKind          ChatKind
	ChatID            string
	UserID            string
	RawChatID         string
	RawUserID         string
	BotID             string
	RawBotID          string
	BotIdentitySource IdentitySource
}

// ReplyTarget is the raw platform peer required by future send code.
type ReplyTarget struct {
	ChatID   string
	ChatKind ChatKind
}

// SelfChatSuppression describes a self-message that was deliberately kept out
// of the kernel route.
type SelfChatSuppression struct {
	Reason    SelfChatSuppressionReason
	ChatID    string
	UserID    string
	MessageID string
}

// InboundResult contains either a routable gateway event or the reason it was
// stopped before reaching the gateway manager.
type InboundResult struct {
	Event       gateway.InboundEvent
	Identity    SessionIdentity
	Reply       ReplyTarget
	Status      IdentityStatus
	Suppression SelfChatSuppression
	Decision    InboundDecision
}

// Routed reports whether the event should be sent to gateway.Manager.
func (r InboundResult) Routed() bool {
	return r.Decision == InboundDecisionRoute
}

// NormalizeWhatsAppIdentifier validates and strips WhatsApp JID/LID/device
// syntax down to a stable peer identifier suitable for gateway equality checks.
func NormalizeWhatsAppIdentifier(value string) string {
	normalized, safe, _ := NormalizeSafeWhatsAppIdentifier(value)
	if !safe {
		return ""
	}
	return normalized
}

// NormalizeSafeWhatsAppIdentifier accepts only safe ASCII WhatsApp peer
// identifiers and returns their stable canonical peer ID. Unsafe values return
// whatsapp_identifier_unsafe evidence instead of being stripped into a
// plausible ID.
func NormalizeSafeWhatsAppIdentifier(value string) (string, bool, WhatsAppIdentifierEvidence) {
	raw := strings.TrimSpace(value)
	if raw == "" || unsafeWhatsAppIdentifier(raw) {
		return "", false, WhatsAppIdentifierUnsafeEvidence
	}

	normalized := normalizeWhatsAppIdentifierSyntax(raw)
	if normalized == "" || unsafeCanonicalWhatsAppIdentifier(normalized) {
		return "", false, WhatsAppIdentifierUnsafeEvidence
	}
	return normalized, true, ""
}

func normalizeWhatsAppIdentifierSyntax(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "+")
	if value == "" {
		return ""
	}
	if before, _, ok := strings.Cut(value, ":"); ok {
		value = before
	}
	if before, _, ok := strings.Cut(value, "@"); ok {
		value = before
	}
	return strings.TrimSpace(value)
}

func unsafeWhatsAppIdentifier(value string) bool {
	lower := strings.ToLower(value)
	if strings.Contains(value, "/") ||
		strings.Contains(value, "\\") ||
		strings.Contains(value, "..") ||
		strings.Contains(lower, "%2f") ||
		strings.Contains(lower, "%5c") {
		return true
	}
	if len(value) > 1 && strings.Contains(value[1:], "+") {
		return true
	}
	for i := 0; i < len(value); i++ {
		if !safeWhatsAppIdentifierByte(value[i]) {
			return true
		}
	}
	return false
}

func safeWhatsAppIdentifierByte(c byte) bool {
	return (c >= '0' && c <= '9') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		c == '@' ||
		c == '.' ||
		c == '+' ||
		c == '-' ||
		c == ':'
}

func unsafeCanonicalWhatsAppIdentifier(value string) bool {
	for i := 0; i < len(value); i++ {
		if !safeCanonicalWhatsAppIdentifierByte(value[i]) {
			return true
		}
	}
	return false
}

func safeCanonicalWhatsAppIdentifierByte(c byte) bool {
	return (c >= '0' && c <= '9') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		c == '-'
}

func resolveBotIdentity(ctx IdentityContext, msg InboundMessage) IdentityStatus {
	source := identitySourceForRuntime(ctx.Runtime)
	var candidates []string
	switch source {
	case IdentitySourceNativeSession:
		candidates = append(candidates, ctx.NativeBotID)
	default:
		candidates = append(candidates, msg.BotIDs...)
		candidates = append(candidates, ctx.BotIDs...)
	}

	for _, raw := range candidates {
		botID := canonicalWhatsAppUserID(raw, ctx.AliasMappings)
		if botID == "" {
			continue
		}
		return IdentityStatus{
			Source:   source,
			Resolved: true,
			BotID:    botID,
			RawBotID: strings.TrimSpace(raw),
		}
	}

	return IdentityStatus{
		Source: source,
		Reason: botIdentityUnresolvedReason,
	}
}

func canonicalWhatsAppUserID(raw string, aliases []IdentityAlias) string {
	start := NormalizeWhatsAppIdentifier(raw)
	if start == "" {
		return ""
	}

	graph := map[string][]string{}
	for _, alias := range aliases {
		from := NormalizeWhatsAppIdentifier(alias.From)
		to := NormalizeWhatsAppIdentifier(alias.To)
		if from == "" || to == "" {
			continue
		}
		graph[from] = append(graph[from], to)
		graph[to] = append(graph[to], from)
	}

	seen := map[string]bool{}
	queue := []string{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == "" || seen[current] {
			continue
		}
		seen[current] = true
		queue = append(queue, graph[current]...)
	}
	if len(seen) == 0 {
		return start
	}

	candidates := make([]string, 0, len(seen))
	for candidate := range seen {
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if len(candidates[i]) != len(candidates[j]) {
			return len(candidates[i]) < len(candidates[j])
		}
		return candidates[i] < candidates[j]
	})
	return candidates[0]
}

func canonicalWhatsAppChatID(raw string, kind ChatKind, aliases []IdentityAlias) string {
	if normalizedChatKind(kind, raw) == ChatKindDirect {
		return canonicalWhatsAppUserID(raw, aliases)
	}
	return NormalizeWhatsAppIdentifier(raw)
}

func normalizedChatKind(kind ChatKind, rawChatID string) ChatKind {
	switch ChatKind(strings.ToLower(strings.TrimSpace(string(kind)))) {
	case ChatKindDirect:
		return ChatKindDirect
	case ChatKindGroup:
		return ChatKindGroup
	}
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(rawChatID)), "@g.us") {
		return ChatKindGroup
	}
	return ChatKindDirect
}

func normalizedAccountMode(mode AccountMode) AccountMode {
	normalized := strings.TrimSpace(strings.ToLower(string(mode)))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	if normalized == string(AccountModeBot) {
		return AccountModeBot
	}
	return AccountModeSelfChat
}

func recentMessageIDSet(ids []string) map[string]bool {
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			out[id] = true
		}
	}
	return out
}
