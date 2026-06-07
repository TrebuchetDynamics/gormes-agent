package approval

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/approval/choice"

// Choice is the bounded decision a messaging-platform approval button may
// resolve for a pending gateway approval request.
type Choice = choice.Choice

const (
	ChoiceOnce    Choice = choice.ChoiceOnce
	ChoiceSession Choice = choice.ChoiceSession
	ChoiceAlways  Choice = choice.ChoiceAlways
	ChoiceDeny    Choice = choice.ChoiceDeny
)

// Resolution is the redacted evidence passed from channel callbacks into the
// gateway approval store/resolver.
type Resolution = choice.Resolution

// Resolver owns the gateway-side approval state for pending dangerous
// operations. Channel implementations call it after a user chooses a bounded
// approval action.
type Resolver = choice.Resolver

// ResolverFunc adapts a function to Resolver.
type ResolverFunc = choice.ResolverFunc

// ParseChoice normalizes a gateway approval decision label.
func ParseChoice(value string) (Choice, bool) {
	return choice.ParseChoice(value)
}
